package webhooks

import (
	"testing"
)

func TestRewriteLoopback(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     string
		want    string
		wantHit bool
	}{
		{"localhost with port", "http://localhost:8080/hook?token=abc", "http://host.docker.internal:8080/hook?token=abc", true},
		{"127.0.0.1 with port", "http://127.0.0.1:4010/twilio-status", "http://host.docker.internal:4010/twilio-status", true},
		{"ipv6 loopback", "http://[::1]:9000/hook", "http://host.docker.internal:9000/hook", true},
		{"localhost no port", "https://localhost/hook", "https://host.docker.internal/hook", true},
		{"uppercase localhost", "http://LOCALHOST:8080/hook", "http://host.docker.internal:8080/hook", true},
		{"non-loopback untouched", "https://example.com/hook", "https://example.com/hook", false},
		{"lan host untouched", "http://192.168.1.10:8080/hook", "http://192.168.1.10:8080/hook", false},
		{"localhost-like suffix untouched", "http://localhost.example.com/hook", "http://localhost.example.com/hook", false},
		{"unparseable untouched", "://bad url", "://bad url", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, hit := RewriteLoopback(tc.raw)
			if got != tc.want || hit != tc.wantHit {
				t.Errorf("RewriteLoopback(%q) = (%q, %v), want (%q, %v)", tc.raw, got, hit, tc.want, tc.wantHit)
			}
		})
	}
}

func TestInDockerEnvOverride(t *testing.T) {
	t.Setenv("MOCKSMS_IN_DOCKER", "1")
	if !InDocker() {
		t.Errorf("expected InDocker with MOCKSMS_IN_DOCKER=1")
	}
	t.Setenv("MOCKSMS_IN_DOCKER", "0")
	if InDocker() {
		t.Errorf("expected not-InDocker with MOCKSMS_IN_DOCKER=0 (no /.dockerenv here)")
	}
}
