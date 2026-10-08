package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/config"
)

func getFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func TestRun_PortConflict(t *testing.T) {
	// Bind a port beforehand to cause a port collision.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = port
	cfg.Memory = true

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = runWithContext(ctx, cfg)
	if err == nil {
		t.Fatal("expected error on port conflict, got nil")
	}
}

func TestRun_CleanShutdown(t *testing.T) {
	httpPort := getFreePort(t)
	smtpPort := getFreePort(t)

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = httpPort
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = smtpPort
	cfg.Memory = true

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithContext(ctx, cfg)
	}()

	// Wait for server to become responsive
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", httpPort)
	ready := false
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
	}
	if !ready {
		t.Fatal("server did not become ready in time")
	}

	// Trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected clean shutdown with nil error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shutdown within 5s timeout")
	}
}

func TestRun_AdapterDedicatedPort(t *testing.T) {
	httpPort := getFreePort(t)
	smtpPort := getFreePort(t)
	twilioPort := getFreePort(t)

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = httpPort
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = smtpPort
	cfg.Memory = true
	cfg.Adapters.Twilio.Port = twilioPort

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithContext(ctx, cfg)
	}()

	// The adapter serves its Twilio-relative routes at / on the dedicated port.
	target := fmt.Sprintf("http://127.0.0.1:%d/2010-04-01/Accounts/AC123/Messages.json", twilioPort)
	ready := false
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		form := url.Values{"To": {"+15005550006"}, "From": {"+15555550100"}, "Body": {"Hi"}}
		req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("AC123", "token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatal("dedicated adapter port did not serve requests in time")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected clean shutdown with nil error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shutdown within 5s timeout")
	}
}

func TestRun_AdapterPortConflict(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = getFreePort(t)
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = getFreePort(t)
	cfg.Memory = true
	cfg.Adapters.Termii.Port = port

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runWithContext(ctx, cfg); err == nil {
		t.Fatal("expected error on adapter port conflict, got nil")
	}
}
