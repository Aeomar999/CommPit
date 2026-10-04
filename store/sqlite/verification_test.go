package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
)

func TestSQLiteStore_VerificationFlow(t *testing.T) {
	store, err := NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "test"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	bus := bus.NewEventBus()
	simulator := core.NewSimulator()
	fakeClock := core.NewFakeClock()

	service := core.NewService(core.ServiceConfig{
		Store:        store,
		BlobStore:    store,
		Bus:          bus,
		Simulator:    simulator,
		Clock:        fakeClock,
		Resolver:     core.NewProjectResolver(store),
		StepDelay:    0,
		OTPFixedCode: stringPtr("123456"),
	})

	t.Run("StartVerification_SMS", func(t *testing.T) {
		resp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:          "+14155552671",
			Channel:     core.ChannelSMS,
			CodeLength:  6,
			TTLSeconds:  300,
			MaxAttempts: 3,
			Provider:    "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		if resp.Verification == nil {
			t.Fatal("expected verification")
		}
		if resp.Verification.Code != "123456" {
			t.Errorf("expected code 123456, got %s", resp.Verification.Code)
		}
		if resp.Verification.Status != core.VerificationPending {
			t.Errorf("expected pending, got %s", resp.Verification.Status)
		}
		if resp.Verification.MaxAttempts != 3 {
			t.Errorf("expected max attempts 3, got %d", resp.Verification.MaxAttempts)
		}
		if resp.Message == nil {
			t.Fatal("expected message")
		}
		if resp.Message.BodyText != "Your verification code is 123456" {
			t.Errorf("expected verification text, got %s", resp.Message.BodyText)
		}
	})

	t.Run("StartVerification_Email", func(t *testing.T) {
		resp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:          "test@example.com",
			Channel:     core.ChannelEmail,
			CodeLength:  6,
			TTLSeconds:  300,
			MaxAttempts: 5,
			Provider:    "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		if resp.Verification == nil {
			t.Fatal("expected verification")
		}
		if resp.Verification.Code != "123456" {
			t.Errorf("expected code 123456, got %s", resp.Verification.Code)
		}
		if resp.Verification.Channel != core.ChannelEmail {
			t.Errorf("expected email channel, got %s", resp.Verification.Channel)
		}
		if resp.Message.BodyText != "Your verification code is 123456" {
			t.Errorf("expected email verification text, got %s", resp.Message.BodyText)
		}
	})

	t.Run("StartVerification_WithServiceRef", func(t *testing.T) {
		serviceRef := "MyService"
		resp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:         "+14155552671",
			Channel:    core.ChannelSMS,
			CodeLength: 6,
			ServiceRef: &serviceRef,
			Provider:   "twilio",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		expectedText := "Your MyService verification code is: 123456"
		if resp.Message.BodyText != expectedText {
			t.Errorf("expected %q, got %q", expectedText, resp.Message.BodyText)
		}
	})

	t.Run("CheckVerification_Success", func(t *testing.T) {
		startResp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:       "+14155552671",
			Channel:  core.ChannelSMS,
			Provider: "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		checkResp, err := service.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "123456",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}

		if !checkResp.Valid {
			t.Error("expected valid=true")
		}
		if checkResp.Status != core.VerificationApproved {
			t.Errorf("expected approved, got %s", checkResp.Status)
		}

		v, err := store.GetVerification(ctx, prjID, startResp.Verification.ID)
		if err != nil {
			t.Fatalf("GetVerification: %v", err)
		}
		if v.Status != core.VerificationApproved {
			t.Errorf("expected approved in store, got %s", v.Status)
		}
	})

	t.Run("CheckVerification_WrongCode", func(t *testing.T) {
		startResp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:       "+14155552671",
			Channel:  core.ChannelSMS,
			Provider: "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		// First wrong attempt
		checkResp, err := service.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "000000",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}
		if checkResp.Valid {
			t.Error("expected valid=false for wrong code")
		}
		if checkResp.Status != core.VerificationPending {
			t.Errorf("expected pending after first wrong attempt, got %s", checkResp.Status)
		}

		// Second wrong attempt (max attempts = 5 by default)
		checkResp, err = service.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "000000",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}
		// Still pending until max attempts
		if checkResp.Status != core.VerificationPending {
			t.Errorf("expected pending after 2 wrong attempts, got %s", checkResp.Status)
		}
	})

	t.Run("CheckVerification_MaxAttempts", func(t *testing.T) {
		// Create service with maxAttempts=2
		service2 := core.NewService(core.ServiceConfig{
			Store:        store,
			BlobStore:    store,
			Bus:          bus,
			Simulator:    simulator,
			Clock:        fakeClock,
			Resolver:     core.NewProjectResolver(store),
			StepDelay:    0,
			OTPFixedCode: stringPtr("123456"),
		})

		startResp, err := service2.StartVerification(ctx, prjID, core.VerificationRequest{
			To:          "+14155552671",
			Channel:     core.ChannelSMS,
			MaxAttempts: 2,
			Provider:    "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		// First wrong attempt
		_, err = service2.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "000000",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}

		// Second wrong attempt - should hit max attempts
		checkResp, err := service2.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "000000",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}
		if checkResp.Status != core.VerificationMaxAttempts {
			t.Errorf("expected max_attempts after 2 wrong attempts, got %s", checkResp.Status)
		}
	})

	t.Run("CheckVerification_Expired", func(t *testing.T) {
		resp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:         "+14155552671",
			Channel:    core.ChannelSMS,
			TTLSeconds: 60,
			Provider:   "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		fakeClock.Advance(61 * time.Second)

		checkResp, err := service.CheckVerification(ctx, prjID, resp.Verification.ID, core.CheckVerificationRequest{
			Code: "123456",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}

		if checkResp.Valid {
			t.Error("expected valid=false for expired")
		}
		if checkResp.Status != core.VerificationExpired {
			t.Errorf("expected expired, got %s", checkResp.Status)
		}
	})

	t.Run("CheckVerification_AlreadyApproved", func(t *testing.T) {
		startResp, err := service.StartVerification(ctx, prjID, core.VerificationRequest{
			To:       "+14155552671",
			Channel:  core.ChannelSMS,
			Provider: "native",
		})
		if err != nil {
			t.Fatalf("StartVerification: %v", err)
		}

		_, err = service.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "123456",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}

		checkResp, err := service.CheckVerification(ctx, prjID, startResp.Verification.ID, core.CheckVerificationRequest{
			Code: "123456",
		})
		if err != nil {
			t.Fatalf("CheckVerification: %v", err)
		}
		if checkResp.Status != core.VerificationApproved {
			t.Errorf("expected still approved, got %s", checkResp.Status)
		}
	})

	t.Run("CheckVerification_NotFound", func(t *testing.T) {
		_, err := service.CheckVerification(ctx, prjID, core.NewVerificationID(), core.CheckVerificationRequest{
			Code: "123456",
		})
		if err == nil {
			t.Error("expected error for non-existent verification")
		}
		if !core.IsError(err, core.ErrCodeVerificationNotFound) {
			t.Errorf("expected verification_not_found error, got %v", err)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
