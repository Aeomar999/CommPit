package smtpd

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

func setupTestSMTP(t *testing.T) (*Server, core.Store, string) {
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

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	server := NewServer(&Config{
		Host:           "127.0.0.1",
		Port:           port,
		MaxMessageSize: 25 * 1024 * 1024,
	}, svc, resolver)
	go func() {
		_ = server.Start()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("smtp server did not start: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	return server, store, addr
}

func sendMail(t *testing.T, addr, from, to, subject, body string, auth smtp.Auth) {
	t.Helper()
	msg := "To: " + to + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body + "\r\n"
	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		t.Fatalf("SendMail: %v", err)
	}
}

func messagesIn(t *testing.T, store core.Store, projectID string) []*core.Message {
	t.Helper()
	msgs, _, err := store.ListMessages(context.Background(), projectID, core.MessageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	return msgs
}

func TestSMTP_NoAuthDeliversToDefaultProject(t *testing.T) {
	_, store, addr := setupTestSMTP(t)

	sendMail(t, addr, "test@app.local", "user@example.com", "Hello SMTP", "Hi from sandbox", nil)

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "smtp", "default")
	if err != nil {
		t.Fatalf("resolve default project: %v", err)
	}
	msgs := messagesIn(t, store, projectID)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 stored message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Subject != "Hello SMTP" || m.From != "test@app.local" || m.To != "user@example.com" {
		t.Errorf("message mismatch: %+v", m)
	}
	if m.Channel != core.ChannelEmail || m.Provider != "smtp" {
		t.Errorf("expected email/smtp message, got %+v", m)
	}
}

func TestSMTP_AuthDeliversToUserProject(t *testing.T) {
	_, store, addr := setupTestSMTP(t)

	auth := smtp.PlainAuth("", "smtpuser", "anypass", "127.0.0.1")
	sendMail(t, addr, "test@app.local", "user@example.com", "Authed", "Hi authed", auth)

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "smtp", "smtpuser")
	if err != nil {
		t.Fatalf("resolve user project: %v", err)
	}
	msgs := messagesIn(t, store, projectID)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 stored message, got %d", len(msgs))
	}
}
