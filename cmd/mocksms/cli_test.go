package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/config"
)

func runCLIForTest(host string, port int, args []string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runCLI(host, port, &stdout, &stderr, args)
	return code, stdout.String(), stderr.String()
}

func TestCLI_Usage(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"bogus"},
		{"otp"},
		{"otp", "latest"},
		{"otp", "latest", "--to", ""},
		{"messages"},
		{"messages", "wait", "--since", "yesterday"},
		{"reset"},
	} {
		// Flag-validation failures must exit 2 without dialing out. The
		// loopback port 1 refuses everything, so any network attempt
		// would surface as exit 1 instead.
		code, _, _ := runCLIForTest("127.0.0.1", 1, args)
		if code != 2 {
			t.Errorf("%v: expected exit 2, got %d", args, code)
		}
	}
}

func TestCLI_OTPLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/otp/latest" || r.URL.Query().Get("to") != "+15005550006" {
			t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("X-Mocksms") == "" {
			t.Errorf("missing X-Mocksms header")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"482916","source":"extracted","message_id":"msg_1"}`)
	}))
	defer server.Close()
	host, port := splitHostPort(t, server.URL)

	code, stdout, _ := runCLIForTest(host, port, []string{"otp", "latest", "--to", "+15005550006"})
	if code != 0 || strings.TrimSpace(stdout) != "482916" {
		t.Fatalf("expected code on stdout, got exit=%d stdout=%q", code, stdout)
	}
}

func TestCLI_FlagOrderFree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"482916"}`)
	}))
	defer server.Close()
	host, port := splitHostPort(t, server.URL)

	for _, args := range [][]string{
		{"otp", "latest", "--to", "+15005550006"},
		{"otp", "--to", "+15005550006", "latest"},
		{"otp", "--to=+15005550006", "latest"},
	} {
		code, stdout, _ := runCLIForTest(host, port, args)
		if code != 0 || strings.TrimSpace(stdout) != "482916" {
			t.Errorf("%v: expected code, got exit=%d stdout=%q", args, code, stdout)
		}
	}
}

func TestCLI_OTPLatestNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"not_found","message":"no OTP found"}}`)
	}))
	defer server.Close()
	host, port := splitHostPort(t, server.URL)

	code, _, stderr := runCLIForTest(host, port, []string{"otp", "latest", "--to", "+19999999999"})
	if code != 1 || !strings.Contains(stderr, "no OTP found") {
		t.Fatalf("expected exit 1 with message, got exit=%d stderr=%q", code, stderr)
	}
}

func TestCLI_MessagesWait(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("timeout"); got != "5" {
			t.Errorf("expected timeout=5, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"msg_1","body_text":"hi"}`)
	}))
	defer server.Close()
	host, port := splitHostPort(t, server.URL)

	code, stdout, _ := runCLIForTest(host, port, []string{"messages", "wait", "--to", "+15005550006", "--timeout", "5"})
	if code != 0 || !strings.Contains(stdout, "msg_1") {
		t.Fatalf("expected message JSON, got exit=%d stdout=%q", code, stdout)
	}
}

func TestCLI_Reset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Query().Get("project") != "prj_1" {
			t.Errorf("unexpected reset request: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	host, port := splitHostPort(t, server.URL)

	code, stdout, _ := runCLIForTest(host, port, []string{"reset", "--project", "prj_1"})
	if code != 0 || !strings.Contains(stdout, "prj_1") {
		t.Fatalf("expected confirmation, got exit=%d stdout=%q", code, stdout)
	}
}

func splitHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	host := strings.TrimPrefix(rawURL, "http://")
	parts := strings.Split(host, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected test URL %q", rawURL)
	}
	var port int
	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return parts[0], port
}

func TestCLI_LiveServer(t *testing.T) {
	httpPort := getFreePort(t)
	smtpPort := getFreePort(t)

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = httpPort
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = smtpPort
	cfg.Memory = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = runWithContext(ctx, cfg)
	}()
	waitForServer(t, httpPort)

	// Seed one SMS carrying an OTP code through the live API.
	seedBody := `{"from":"+15555550100","to":"+15005550006","body":"Your verification code is 731946"}`
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/api/v1/sms", httpPort), strings.NewReader(seedBody))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mocksms", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("seed send: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("seed send: status %d", resp.StatusCode)
	}

	code, stdout, stderr := runCLIForTest("127.0.0.1", httpPort, []string{"otp", "latest", "--to", "+15005550006"})
	if code != 0 || strings.TrimSpace(stdout) != "731946" {
		t.Fatalf("otp latest: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	since := time.Now().Add(-time.Minute).Format(time.RFC3339)
	code, stdout, _ = runCLIForTest("127.0.0.1", httpPort, []string{"messages", "wait", "--to", "+15005550006", "--since", since, "--timeout", "5"})
	if code != 0 || !strings.Contains(stdout, "731946") {
		t.Fatalf("messages wait: exit=%d stdout=%q", code, stdout)
	}
}

func waitForServer(t *testing.T, port int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
