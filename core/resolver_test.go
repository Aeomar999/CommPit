package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

func setupResolverTest(t *testing.T) (core.ProjectResolver, core.Store) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return core.NewProjectResolver(store), store
}

func TestResolver_LinkCredential(t *testing.T) {
	t.Run("links a new credential to an existing project", func(t *testing.T) {
		resolver, store := setupResolverTest(t)
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "linked", Settings: map[string]interface{}{}, CreatedAt: time.Now()}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		if err := resolver.LinkCredential(ctx, "twilio", "AC999", prjID); err != nil {
			t.Fatalf("LinkCredential: %v", err)
		}
		got, err := resolver.Resolve(ctx, "twilio", "AC999")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got != prjID {
			t.Errorf("expected %s, got %s", prjID, got)
		}
	})

	t.Run("moves an existing credential", func(t *testing.T) {
		resolver, store := setupResolverTest(t)
		ctx := context.Background()

		first, err := resolver.Resolve(ctx, "twilio", "AC888")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "target", Settings: map[string]interface{}{}, CreatedAt: time.Now()}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		if err := resolver.LinkCredential(ctx, "twilio", "AC888", prjID); err != nil {
			t.Fatalf("LinkCredential: %v", err)
		}
		got, err := resolver.Resolve(ctx, "twilio", "AC888")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got != prjID {
			t.Errorf("expected moved credential to resolve to %s, got %s", prjID, got)
		}
		if got == first {
			t.Errorf("credential still resolves to its original project %s", first)
		}
	})

	t.Run("missing project fails", func(t *testing.T) {
		resolver, _ := setupResolverTest(t)
		err := resolver.LinkCredential(context.Background(), "twilio", "AC777", "prj_doesnotexist00000000000000")
		if !core.IsError(err, core.ErrCodeNotFound) {
			t.Errorf("expected not_found, got %v", err)
		}
	})
}
