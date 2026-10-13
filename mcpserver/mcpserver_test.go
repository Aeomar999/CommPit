package mcpserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

func setupTestMCP(t *testing.T) (*Server, core.Store, string) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	eventBus := bus.NewEventBus()
	resolver := core.NewProjectResolver(store)
	svc := core.NewService(core.ServiceConfig{
		Store:     store,
		BlobStore: store,
		Bus:       eventBus,
		Simulator: sim.NewSimulator(),
		Clock:     core.NewFakeClock(),
		Resolver:  resolver,
	})
	server := New(svc)

	projectID, err := resolver.Resolve(context.Background(), "native", "default")
	if err != nil {
		t.Fatalf("resolve default project: %v", err)
	}
	return server, store, projectID
}

func sendTestSMS(t *testing.T, svc *core.Service, projectID, to, body string) *core.Message {
	t.Helper()
	resp, err := svc.SendMessage(context.Background(), projectID, core.SendRequest{
		Channel:  core.ChannelSMS,
		From:     "+15555550100",
		To:       []string{to},
		BodyText: body,
		Provider: "native",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	return resp.Message
}

func TestMCP_ListMessages(t *testing.T) {
	server, _, projectID := setupTestMCP(t)
	svc := server.service
	sendTestSMS(t, svc, projectID, "+15005550006", "hello one")
	sendTestSMS(t, svc, projectID, "+15005550007", "hello two")

	res, out, err := server.listMessages(context.Background(), nil, listMessagesInput{Limit: 10})
	if err != nil {
		t.Fatalf("listMessages: %v", err)
	}
	messages, ok := out.([]*core.Message)
	if !ok || len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %+v", out)
	}
	if res == nil {
		t.Fatalf("expected text result")
	}

	to := "+15005550006"
	res, out, err = server.listMessages(context.Background(), nil, listMessagesInput{To: to})
	if err != nil {
		t.Fatalf("listMessages to: %v", err)
	}
	_ = res
	if messages, ok := out.([]*core.Message); !ok || len(messages) != 1 {
		t.Fatalf("expected 1 filtered message, got %+v", out)
	}
}

func TestMCP_WaitForMessage(t *testing.T) {
	server, _, projectID := setupTestMCP(t)
	svc := server.service

	t.Run("existing message with explicit since", func(t *testing.T) {
		sent := sendTestSMS(t, svc, projectID, "+15005550006", "already here")
		since := sent.CreatedAt.Add(-time.Second).Format(time.RFC3339)
		_, out, err := server.waitForMessage(context.Background(), nil, waitForMessageInput{
			To:             "+15005550006",
			Since:          since,
			TimeoutSeconds: 2,
		})
		if err != nil {
			t.Fatalf("waitForMessage: %v", err)
		}
		msg, ok := out.(*core.Message)
		if !ok || msg.ID != sent.ID {
			t.Fatalf("expected the seeded message, got %+v", out)
		}
	})

	t.Run("timeout errors", func(t *testing.T) {
		_, _, err := server.waitForMessage(context.Background(), nil, waitForMessageInput{
			To:             "+19999999999",
			TimeoutSeconds: 1,
		})
		if err == nil {
			t.Fatalf("expected timeout error, got nil")
		}
	})
}

func TestMCP_GetLatestOTP(t *testing.T) {
	server, _, projectID := setupTestMCP(t)
	svc := server.service
	sendTestSMS(t, svc, projectID, "+15005550006", "Your verification code is 482916")

	_, out, err := server.getLatestOTP(context.Background(), nil, latestOTPInput{To: "+15005550006"})
	if err != nil {
		t.Fatalf("getLatestOTP: %v", err)
	}
	otp, ok := out.(latestOTPOutput)
	if !ok || otp.Code != "482916" || otp.Source != "extracted" {
		t.Fatalf("unexpected OTP output: %+v", out)
	}

	if _, _, err := server.getLatestOTP(context.Background(), nil, latestOTPInput{To: "+19999999999"}); err == nil {
		t.Fatalf("expected not-found error, got nil")
	}
	if _, _, err := server.getLatestOTP(context.Background(), nil, latestOTPInput{}); err == nil {
		t.Fatalf("expected required-to error, got nil")
	}
}

func TestMCP_GetLatestEmail(t *testing.T) {
	server, _, projectID := setupTestMCP(t)
	svc := server.service
	resp, err := svc.SendMessage(context.Background(), projectID, core.SendRequest{
		Channel:  core.ChannelEmail,
		From:     "app@example.com",
		To:       []string{"user@example.com"},
		Subject:  "Receipt",
		BodyText: "thanks",
		Provider: "native",
	})
	if err != nil {
		t.Fatalf("SendMessage email: %v", err)
	}

	_, out, err := server.getLatestEmail(context.Background(), nil, latestEmailInput{To: "user@example.com"})
	if err != nil {
		t.Fatalf("getLatestEmail: %v", err)
	}
	msg, ok := out.(*core.Message)
	if !ok || msg.ID != resp.Message.ID {
		t.Fatalf("unexpected email output: %+v", out)
	}
}

func TestMCP_SimulateInbound(t *testing.T) {
	server, _, _ := setupTestMCP(t)
	_, out, err := server.simulateInbound(context.Background(), nil, simulateInboundInput{
		From: "+15005550006",
		To:   "+15555550100",
		Body: "STOP",
	})
	if err != nil {
		t.Fatalf("simulateInbound: %v", err)
	}
	msg, ok := out.(*core.Message)
	if !ok || msg.Direction != core.DirectionInbound || msg.Status != core.StatusReceived {
		t.Fatalf("unexpected inbound output: %+v", out)
	}
}

func TestMCP_ListRequests(t *testing.T) {
	server, store, projectID := setupTestMCP(t)
	if err := store.CreateRequestLog(context.Background(), &core.RequestLog{
		ID:             core.NewRequestLogID(),
		ProjectID:      projectID,
		Adapter:        "twilio",
		Method:         "POST",
		Path:           "/2010-04-01/Accounts/ACx/Messages.json",
		RequestHeaders: map[string]string{},
		RequestBody:    []byte("{}"),
		ResponseBody:   []byte("{}"),
		ResponseStatus: 201,
		CreatedAt:      time.Now(),
	}); err != nil {
		t.Fatalf("CreateRequestLog: %v", err)
	}
	_, out, err := server.listRequests(context.Background(), nil, listRequestsInput{})
	if err != nil {
		t.Fatalf("listRequests: %v", err)
	}
	logs, ok := out.([]*core.RequestLog)
	if !ok || len(logs) != 1 {
		t.Fatalf("expected 1 log, got %+v", out)
	}
}

func TestMCP_ResetProject(t *testing.T) {
	server, store, projectID := setupTestMCP(t)
	svc := server.service
	sendTestSMS(t, svc, projectID, "+15005550006", "bye")

	_, out, err := server.resetProject(context.Background(), nil, resetProjectInput{})
	if err != nil {
		t.Fatalf("resetProject: %v", err)
	}
	if o, ok := out.(resetProjectOutput); !ok || !o.Reset || o.Project != projectID {
		t.Fatalf("unexpected reset output: %+v", out)
	}
	msgs, _, err := store.ListMessages(context.Background(), projectID, core.MessageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected empty inbox, got %d", len(msgs))
	}
}

func TestMCP_ResolveProject(t *testing.T) {
	server, _, projectID := setupTestMCP(t)

	got, err := server.resolveProject(context.Background(), "")
	if err != nil || got != projectID {
		t.Fatalf("expected default %s, got %s %v", projectID, got, err)
	}
	if _, err := server.resolveProject(context.Background(), "prj_00000000000000000000000000"); err == nil {
		t.Fatalf("expected not-found for bogus id, got nil")
	}
}

func TestMCP_ResultsAreJSON(t *testing.T) {
	server, _, projectID := setupTestMCP(t)
	svc := server.service
	sendTestSMS(t, svc, projectID, "+15005550006", "json check")
	res, _, err := server.listMessages(context.Background(), nil, listMessagesInput{Limit: 1})
	if err != nil {
		t.Fatalf("listMessages: %v", err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
		t.Fatalf("result is not a JSON array: %v", err)
	}
	if len(decoded) != 1 || decoded[0]["body_text"] != "json check" {
		t.Fatalf("unexpected payload: %v", decoded)
	}
}
