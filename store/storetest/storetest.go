package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/core"
	"github.com/google/go-cmp/cmp"
)

func RunStoreTests(t *testing.T, newStore func() (core.Store, func())) {
	t.Helper()

	t.Run("Project", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		// CreateProject
		prj := &core.Project{
			ID:        core.NewProjectID(),
			Name:      "test-project",
			Settings:  map[string]interface{}{"key": "value"},
			CreatedAt: time.Now(),
		}
		if err := store.CreateProject(ctx, prj); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// GetProject
		got, err := store.GetProject(ctx, prj.ID)
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		if diff := cmp.Diff(prj, got); diff != "" {
			t.Errorf("GetProject mismatch (-want +got):\n%s", diff)
		}

		// GetProject not found
		_, err = store.GetProject(ctx, core.NewProjectID())
		if err == nil {
			t.Error("expected error for non-existent project")
		}

		// UpdateProject
		prj.Name = "updated-name"
		if err := store.UpdateProject(ctx, prj); err != nil {
			t.Fatalf("UpdateProject: %v", err)
		}
		got, err = store.GetProject(ctx, prj.ID)
		if err != nil {
			t.Fatalf("GetProject after update: %v", err)
		}
		if got.Name != "updated-name" {
			t.Errorf("expected updated name, got %s", got.Name)
		}

		// ListProjects
		projects, cursor, err := store.ListProjects(ctx, 10, "")
		if err != nil {
			t.Fatalf("ListProjects: %v", err)
		}
		if len(projects) != 1 {
			t.Errorf("expected 1 project, got %d", len(projects))
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}
	})

	t.Run("Credential", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// CreateCredential
		cred := &core.Credential{
			ID:        core.NewRequestLogID(),
			Provider:  "twilio",
			Key:       "AC123",
			ProjectID: prjID,
			CreatedAt: time.Now(),
		}
		if err := store.CreateCredential(ctx, cred); err != nil {
			t.Fatalf("CreateCredential: %v", err)
		}

		// GetCredential
		got, err := store.GetCredential(ctx, "twilio", "AC123")
		if err != nil {
			t.Fatalf("GetCredential: %v", err)
		}
		if got.Provider != "twilio" || got.Key != "AC123" || got.ProjectID != prjID {
			t.Errorf("credential mismatch: %+v", got)
		}

		// ListCredentials
		creds, err := store.ListCredentials(ctx, prjID)
		if err != nil {
			t.Fatalf("ListCredentials: %v", err)
		}
		if len(creds) != 1 {
			t.Errorf("expected 1 credential, got %d", len(creds))
		}

		// DeleteCredential
		if err := store.DeleteCredential(ctx, cred.ID); err != nil {
			t.Fatalf("DeleteCredential: %v", err)
		}
		_, err = store.GetCredential(ctx, "twilio", "AC123")
		if err == nil {
			t.Error("expected error after delete")
		}
	})

	t.Run("Message", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// CreateMessage
		msg := &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			Channel:   core.ChannelSMS,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "+15551234567",
			To:        "+15557654321",
			BodyText:  "Hello world",
			Status:    core.StatusQueued,
			Segments:  1,
			Encoding:  "gsm7",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}

		// GetMessage
		got, err := store.GetMessage(ctx, prjID, msg.ID)
		if err != nil {
			t.Fatalf("GetMessage: %v", err)
		}
		if got.ID != msg.ID || got.To != msg.To {
			t.Errorf("message mismatch: %+v", got)
		}

		// GetMessage not found
		_, err = store.GetMessage(ctx, prjID, core.NewMessageID())
		if err == nil {
			t.Error("expected error for non-existent message")
		}

		// UpdateMessage
		msg.Status = core.StatusSent
		msg.UpdatedAt = time.Now()
		if err := store.UpdateMessage(ctx, msg); err != nil {
			t.Fatalf("UpdateMessage: %v", err)
		}
		got, err = store.GetMessage(ctx, prjID, msg.ID)
		if err != nil {
			t.Fatalf("GetMessage after update: %v", err)
		}
		if got.Status != core.StatusSent {
			t.Errorf("expected status sent, got %s", got.Status)
		}

		// ListMessages with filters
		messages, cursor, err := store.ListMessages(ctx, prjID, core.MessageFilter{
			Channel: ptrChannel(core.ChannelSMS),
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		if len(messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(messages))
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}

		// ListMessages with status filter
		messages, _, err = store.ListMessages(ctx, prjID, core.MessageFilter{
			Status: ptrStatus(core.StatusQueued),
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("ListMessages with status: %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("expected 0 queued messages, got %d", len(messages))
		}

		// DeleteMessages
		if err := store.DeleteMessages(ctx, prjID); err != nil {
			t.Fatalf("DeleteMessages: %v", err)
		}
		messages, _, err = store.ListMessages(ctx, prjID, core.MessageFilter{Limit: 10})
		if err != nil {
			t.Fatalf("ListMessages after delete: %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("expected 0 messages after delete, got %d", len(messages))
		}
	})

	t.Run("StatusEvent", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		msg := &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			Channel:   core.ChannelSMS,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "+15551234567",
			To:        "+15557654321",
			BodyText:  "Hello",
			Status:    core.StatusQueued,
			Segments:  1,
			Encoding:  "gsm7",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}

		// CreateStatusEvent
		event := &core.StatusEvent{
			ID:        core.NewWebhookDeliveryID(),
			MessageID: msg.ID,
			Status:    core.StatusSent,
			At:        time.Now(),
		}
		if err := store.CreateStatusEvent(ctx, event); err != nil {
			t.Fatalf("CreateStatusEvent: %v", err)
		}

		// GetStatusEvents
		events, err := store.GetStatusEvents(ctx, msg.ID)
		if err != nil {
			t.Fatalf("GetStatusEvents: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("expected 1 event, got %d", len(events))
		}
		if events[0].Status != core.StatusSent {
			t.Errorf("expected status sent, got %s", events[0].Status)
		}
	})

	t.Run("Batch", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// CreateBatch
		batch := &core.Batch{
			ID:        core.NewBatchID(),
			ProjectID: prjID,
			Provider:  "native",
			Channel:   core.ChannelSMS,
			Total:     5,
			Counts:    map[string]int{string(core.StatusQueued): 5},
			CreatedAt: time.Now(),
		}
		if err := store.CreateBatch(ctx, batch); err != nil {
			t.Fatalf("CreateBatch: %v", err)
		}

		// GetBatch
		got, err := store.GetBatch(ctx, prjID, batch.ID)
		if err != nil {
			t.Fatalf("GetBatch: %v", err)
		}
		if got.Total != 5 {
			t.Errorf("expected total 5, got %d", got.Total)
		}

		// UpdateBatch
		batch.Counts[string(core.StatusSent)] = 3
		batch.Counts[string(core.StatusQueued)] = 2
		if err := store.UpdateBatch(ctx, batch); err != nil {
			t.Fatalf("UpdateBatch: %v", err)
		}
		got, err = store.GetBatch(ctx, prjID, batch.ID)
		if err != nil {
			t.Fatalf("GetBatch after update: %v", err)
		}
		if got.Counts[string(core.StatusSent)] != 3 {
			t.Errorf("expected sent count 3, got %d", got.Counts[string(core.StatusSent)])
		}

		// ListBatches
		batches, cursor, err := store.ListBatches(ctx, prjID, 10, "")
		if err != nil {
			t.Fatalf("ListBatches: %v", err)
		}
		if len(batches) != 1 {
			t.Errorf("expected 1 batch, got %d", len(batches))
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}
	})

	t.Run("Verification", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// CreateVerification
		v := &core.Verification{
			ID:          core.NewVerificationID(),
			ProjectID:   prjID,
			Provider:    "native",
			ProviderRef: "vrf_123",
			To:          "+15551234567",
			Channel:     core.ChannelSMS,
			Code:        "123456",
			Status:      core.VerificationPending,
			Attempts:    0,
			MaxAttempts: 5,
			ExpiresAt:   time.Now().Add(10 * time.Minute),
			MessageID:   core.NewMessageID(),
			CreatedAt:   time.Now(),
		}
		if err := store.CreateVerification(ctx, v); err != nil {
			t.Fatalf("CreateVerification: %v", err)
		}

		// GetVerification
		got, err := store.GetVerification(ctx, prjID, v.ID)
		if err != nil {
			t.Fatalf("GetVerification: %v", err)
		}
		if got.Code != "123456" {
			t.Errorf("expected code 123456, got %s", got.Code)
		}

		// GetVerificationByProviderRef
		got, err = store.GetVerificationByProviderRef(ctx, prjID, v.ProviderRef)
		if err != nil {
			t.Fatalf("GetVerificationByProviderRef: %v", err)
		}
		if got.ID != v.ID {
			t.Errorf("expected same verification, got %s", got.ID)
		}

		// UpdateVerification
		v.Status = core.VerificationApproved
		v.Attempts = 1
		if err := store.UpdateVerification(ctx, v); err != nil {
			t.Fatalf("UpdateVerification: %v", err)
		}
		got, err = store.GetVerification(ctx, prjID, v.ID)
		if err != nil {
			t.Fatalf("GetVerification after update: %v", err)
		}
		if got.Status != core.VerificationApproved {
			t.Errorf("expected approved, got %s", got.Status)
		}

		// ListVerifications
		verifications, cursor, err := store.ListVerifications(ctx, prjID, 10, "")
		if err != nil {
			t.Fatalf("ListVerifications: %v", err)
		}
		if len(verifications) != 1 {
			t.Errorf("expected 1 verification, got %d", len(verifications))
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// CreateUnsubscribe
		unsub := &core.Unsubscribe{
			ProjectID: prjID,
			Number:    "+15551234567",
			At:        time.Now(),
		}
		if err := store.CreateUnsubscribe(ctx, unsub); err != nil {
			t.Fatalf("CreateUnsubscribe: %v", err)
		}

		// IsUnsubscribed
		unsubscribed, err := store.IsUnsubscribed(ctx, prjID, "+15551234567")
		if err != nil {
			t.Fatalf("IsUnsubscribed: %v", err)
		}
		if !unsubscribed {
			t.Error("expected unsubscribed")
		}

		// Not unsubscribed
		unsubscribed, err = store.IsUnsubscribed(ctx, prjID, "+15559999999")
		if err != nil {
			t.Fatalf("IsUnsubscribed for non-existent: %v", err)
		}
		if unsubscribed {
			t.Error("expected not unsubscribed")
		}

		// DeleteUnsubscribe
		if err := store.DeleteUnsubscribe(ctx, prjID, "+15551234567"); err != nil {
			t.Fatalf("DeleteUnsubscribe: %v", err)
		}
		unsubscribed, err = store.IsUnsubscribed(ctx, prjID, "+15551234567")
		if err != nil {
			t.Fatalf("IsUnsubscribed after delete: %v", err)
		}
		if unsubscribed {
			t.Error("expected not unsubscribed after delete")
		}
	})

	t.Run("Attachment", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		msg := &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			Channel:   core.ChannelEmail,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "test@example.com",
			To:        "to@example.com",
			Status:    core.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}

		// CreateAttachment
		att := &core.Attachment{
			ID:          core.NewAttachmentID(),
			MessageID:   msg.ID,
			Filename:    "test.txt",
			ContentType: "text/plain",
			Size:        123,
			BlobID:      core.NewBlobID(),
			InlineCID:   "",
		}
		if err := store.CreateAttachment(ctx, att); err != nil {
			t.Fatalf("CreateAttachment: %v", err)
		}

		// GetAttachment
		got, err := store.GetAttachment(ctx, att.ID)
		if err != nil {
			t.Fatalf("GetAttachment: %v", err)
		}
		if got.Filename != "test.txt" {
			t.Errorf("expected test.txt, got %s", got.Filename)
		}

		// ListAttachments
		atts, err := store.ListAttachments(ctx, msg.ID)
		if err != nil {
			t.Fatalf("ListAttachments: %v", err)
		}
		if len(atts) != 1 {
			t.Errorf("expected 1 attachment, got %d", len(atts))
		}
	})

	t.Run("WebhookDelivery", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		msg := &core.Message{
			ID:        core.NewMessageID(),
			ProjectID: prjID,
			Channel:   core.ChannelSMS,
			Direction: core.DirectionOutbound,
			Provider:  "native",
			From:      "+15551234567",
			To:        "+15557654321",
			BodyText:  "Hello",
			Status:    core.StatusQueued,
			Segments:  1,
			Encoding:  "gsm7",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}

		// CreateWebhookDelivery
		whd := &core.WebhookDelivery{
			ID:          core.NewWebhookDeliveryID(),
			ProjectID:   prjID,
			MessageID:   &msg.ID,
			Kind:        "status",
			URL:         "https://example.com/webhook",
			Payload:     map[string]interface{}{"event": "test"},
			Headers:     map[string]string{"Content-Type": "application/json"},
			Attempt:     1,
			Status:      core.WebhookPending,
			NextRetryAt: ptrTime(time.Now().Add(1 * time.Minute)),
			CreatedAt:   time.Now(),
		}
		if err := store.CreateWebhookDelivery(ctx, whd); err != nil {
			t.Fatalf("CreateWebhookDelivery: %v", err)
		}

		// GetWebhookDelivery
		got, err := store.GetWebhookDelivery(ctx, whd.ID)
		if err != nil {
			t.Fatalf("GetWebhookDelivery: %v", err)
		}
		if got.URL != "https://example.com/webhook" {
			t.Errorf("expected webhook URL, got %s", got.URL)
		}

		// UpdateWebhookDelivery
		whd.Status = core.WebhookSucceeded
		whd.ResponseStatus = ptrInt(200)
		if err := store.UpdateWebhookDelivery(ctx, whd); err != nil {
			t.Fatalf("UpdateWebhookDelivery: %v", err)
		}
		got, err = store.GetWebhookDelivery(ctx, whd.ID)
		if err != nil {
			t.Fatalf("GetWebhookDelivery after update: %v", err)
		}
		if got.Status != core.WebhookSucceeded {
			t.Errorf("expected succeeded, got %s", got.Status)
		}

		// ListPendingWebhooks
		pending, err := store.ListPendingWebhooks(ctx, 10)
		if err != nil {
			t.Fatalf("ListPendingWebhooks: %v", err)
		}
		if len(pending) != 0 {
			t.Errorf("expected 0 pending after success, got %d", len(pending))
		}
	})

	t.Run("RequestLog", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// CreateRequestLog
		log := &core.RequestLog{
			ID:             core.NewRequestLogID(),
			ProjectID:      prjID,
			Adapter:        "twilio",
			Method:         "POST",
			Path:           "/2010-04-01/Accounts/AC123/Messages.json",
			RequestHeaders: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			RequestBody:    []byte("To=%2B15551234567&From=%2B15557654321&Body=Hello"),
			ResponseStatus: 201,
			ResponseBody:   []byte(`{"sid":"SM123"}`),
			DurationMS:     42,
			CreatedAt:      time.Now(),
		}
		if err := store.CreateRequestLog(ctx, log); err != nil {
			t.Fatalf("CreateRequestLog: %v", err)
		}

		// GetRequestLog
		got, err := store.GetRequestLog(ctx, log.ID)
		if err != nil {
			t.Fatalf("GetRequestLog: %v", err)
		}
		if got.Adapter != "twilio" {
			t.Errorf("expected twilio, got %s", got.Adapter)
		}

		// ListRequestLogs
		logs, cursor, err := store.ListRequestLogs(ctx, prjID, 10, "")
		if err != nil {
			t.Fatalf("ListRequestLogs: %v", err)
		}
		if len(logs) != 1 {
			t.Errorf("expected 1 log, got %d", len(logs))
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}
	})

	t.Run("Transaction", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()
		ctx := context.Background()

		prjID := core.NewProjectID()
		if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}

		// Successful transaction
		err := store.Transaction(ctx, func(s core.Store) error {
			msg := &core.Message{
				ID:        core.NewMessageID(),
				ProjectID: prjID,
				Channel:   core.ChannelSMS,
				Direction: core.DirectionOutbound,
				Provider:  "native",
				From:      "+15551234567",
				To:        "+15557654321",
				BodyText:  "In transaction",
				Status:    core.StatusQueued,
				Segments:  1,
				Encoding:  "gsm7",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return s.CreateMessage(ctx, msg)
		})
		if err != nil {
			t.Fatalf("Transaction success: %v", err)
		}

		messages, _, _ := store.ListMessages(ctx, prjID, core.MessageFilter{Limit: 10})
		if len(messages) != 1 {
			t.Errorf("expected 1 message after successful transaction, got %d", len(messages))
		}

		// Failed transaction (rollback)
		err = store.Transaction(ctx, func(s core.Store) error {
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
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.CreateMessage(ctx, msg); err != nil {
				return err
			}
			return core.NewInternal("intentional failure")
		})
		if err == nil {
			t.Error("expected transaction to fail")
		}

		messages, _, _ = store.ListMessages(ctx, prjID, core.MessageFilter{Limit: 10})
		if len(messages) != 1 {
			t.Errorf("expected 1 message after rolled back transaction, got %d", len(messages))
		}
	})
}

func ptrChannel(c core.Channel) *core.Channel {
	return &c
}

func ptrStatus(s core.MessageStatus) *core.MessageStatus {
	return &s
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func ptrInt(i int) *int {
	return &i
}
