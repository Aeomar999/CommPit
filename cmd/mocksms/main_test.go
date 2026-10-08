package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/config"
	"github.com/Aeomar999/CommPit/core"
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
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", httpPort)
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

func TestRun_YAMLProjectLinking(t *testing.T) {
	httpPort := getFreePort(t)
	smtpPort := getFreePort(t)
	projectID := core.NewProjectID()

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = httpPort
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = smtpPort
	cfg.Memory = true
	cfg.Projects = []config.ProjectLinkConfig{
		{
			ID:   projectID,
			Name: "combined",
			Credentials: []config.CredentialLinkConfig{
				{Provider: "native", Key: "YAMLKEY123"},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithContext(ctx, cfg)
	}()

	// Wait for server to become responsive
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", httpPort)
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

	// Sending with the YAML-linked credential must land in the YAML project.
	payload, _ := json.Marshal(map[string]any{"from": "+15555550100", "to": "+15005550006", "body": "Hi"})
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/api/v1/sms", httpPort), bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer YAMLKEY123")
	req.Header.Set("X-Mocksms", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send SMS: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected status 201, got %d: %v", resp.StatusCode, errBody)
	}

	// The YAML project exists (?project= validates existence without side
	// effects) and the Bearer send landed in it.
	checkReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/projects?project=%s", httpPort, projectID), nil)
	if err != nil {
		t.Fatalf("build check request: %v", err)
	}
	checkReq.Header.Set("X-Mocksms", "1")
	checkResp, err := http.DefaultClient.Do(checkReq)
	if err != nil {
		t.Fatalf("check project: %v", err)
	}
	checkResp.Body.Close()
	if checkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for linked project, got %d", checkResp.StatusCode)
	}

	msgsReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/messages?project=%s", httpPort, projectID), nil)
	if err != nil {
		t.Fatalf("build messages request: %v", err)
	}
	msgsReq.Header.Set("X-Mocksms", "1")
	msgsResp, err := http.DefaultClient.Do(msgsReq)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	defer msgsResp.Body.Close()
	var msgsBody map[string]any
	if err := json.NewDecoder(msgsResp.Body).Decode(&msgsBody); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	msgs, _ := msgsBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected the sent message in the linked project, got %v", msgs)
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

func TestRun_ApiV1Prefix(t *testing.T) {
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

	base := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	ready := false
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		resp, err := http.Get(base + "/api/v1/healthz")
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

	// The spec-canonical /api/v1 prefix serves the native API for the web UI.
	payload, _ := json.Marshal(map[string]any{"from": "+15555550100", "to": "+15005550006", "body": "Hi"})
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/sms", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mocksms", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send SMS: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
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
