package sqlite

import (
	"context"
	"os"
	"testing"

	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/store/storetest"
)

func TestSQLiteStoreConformance(t *testing.T) {
	// Use in-memory database for testing
	store, cleanup := newTestStore(t)
	defer cleanup()

	storetest.RunStoreTests(t, func() (core.Store, func()) {
		return store, func() {}
	})
}

func TestSQLiteStorePersists(t *testing.T) {
	// Use file-based database to test persistence
	tmpDir := t.TempDir()

	store1, err := NewStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	ctx := context.Background()
	prj := &core.Project{
		ID:        core.NewProjectID(),
		Name:      "persist-test",
		Settings:  map[string]interface{}{"key": "value"},
		CreatedAt: core.RealClock{}.Now(),
	}
	if err := store1.CreateProject(ctx, prj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	store1.Close()

	// Reopen and verify
	store2, err := NewStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("NewStore (reopen): %v", err)
	}
	defer store2.Close()

	got, err := store2.GetProject(ctx, prj.ID)
	if err != nil {
		t.Fatalf("GetProject after reopen: %v", err)
	}
	if got.Name != "persist-test" {
		t.Errorf("expected persist-test, got %s", got.Name)
	}
}

func TestSQLiteStoreTransactionRollback(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Transaction that fails
	err := store.Transaction(ctx, func(s core.Store) error {
		msg := &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			Channel:   core.ChannelSMS,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "+15551234567",
			To:        "+15557654321",
			BodyText:  "Should rollback",
			Status:    core.StatusQueued,
			Segments:  1,
			Encoding:  "gsm7",
			CreatedAt: core.RealClock{}.Now(),
			UpdatedAt: core.RealClock{}.Now(),
		}
		if err := s.CreateMessage(ctx, msg); err != nil {
			return err
		}
		return core.NewInternal("intentional failure")
	})
	if err == nil {
		t.Error("expected transaction to fail")
	}

	// Verify rollback
	messages, _, _ := store.ListMessages(ctx, prjID, core.MessageFilter{Limit: 10})
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after rollback, got %d", len(messages))
	}
}

func TestSQLiteStoreBatchInsert(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// SendBatch should insert batch and all messages in one transaction
	batch := &core.Batch{
		ID:        core.NewBatchID(),
		ProjectID: prjID,
		Provider:  "native",
		Channel:   core.ChannelSMS,
		Total:     3,
		Counts:    map[string]int{string(core.StatusQueued): 3},
		CreatedAt: core.RealClock{}.Now(),
	}
	if err := store.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	messages := []*core.Message{}
	for i := 0; i < 3; i++ {
		messages = append(messages, &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			BatchID:   &batch.ID,
			Channel:   core.ChannelSMS,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "+15551234567",
			To:        "+15557654321",
			BodyText:  "Batch message",
			Status:    core.StatusQueued,
			Segments:  1,
			Encoding:  "gsm7",
			CreatedAt: core.RealClock{}.Now(),
			UpdatedAt: core.RealClock{}.Now(),
		})
	}

	// Use transaction for batch insert
	err := store.Transaction(ctx, func(s core.Store) error {
		for _, msg := range messages {
			if err := s.CreateMessage(ctx, msg); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transaction batch insert: %v", err)
	}

	// Verify all messages exist
	listed, _, _ := store.ListMessages(ctx, prjID, core.MessageFilter{BatchID: &batch.ID, Limit: 10})
	if len(listed) != 3 {
		t.Errorf("expected 3 messages in batch, got %d", len(listed))
	}

	// Verify batch counts
	gotBatch, _ := store.GetBatch(ctx, prjID, batch.ID)
	if gotBatch.Total != 3 {
		t.Errorf("expected batch total 3, got %d", gotBatch.Total)
	}
}

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	store, err := NewStore(":memory:", 2)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store, func() { store.Close() }
}

func TestSQLiteStoreWALMode(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	// Verify WAL mode is enabled
	row := store.getReadDB().QueryRowContext(context.Background(), `PRAGMA journal_mode`)
	var mode string
	if err := row.Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected WAL mode, got %s", mode)
	}
}

func TestSQLiteStoreConcurrentReads(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Insert some messages
	for i := 0; i < 10; i++ {
		msg := &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			Channel:   core.ChannelSMS,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "+15551234567",
			To:        "+15557654321",
			BodyText:  "Test",
			Status:    core.StatusQueued,
			Segments:  1,
			Encoding:  "gsm7",
			CreatedAt: core.RealClock{}.Now(),
			UpdatedAt: core.RealClock{}.Now(),
		}
		if err := store.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}
	}

	// Concurrent reads
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _, err := store.ListMessages(ctx, prjID, core.MessageFilter{Limit: 5})
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent read error: %v", err)
		}
	}
}

func TestMain(m *testing.M) {
	// Ensure migrations directory exists
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		// Migrations are embedded in the test binary via goose
	}
	os.Exit(m.Run())
}
