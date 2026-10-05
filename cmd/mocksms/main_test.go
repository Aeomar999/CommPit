package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	port := getFreePort(t)

	cfg := config.Load()
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = port
	cfg.Memory = true

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithContext(ctx, cfg)
	}()

	// Wait for server to become responsive
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
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
