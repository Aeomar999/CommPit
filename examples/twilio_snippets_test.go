//go:build e2e

package examples_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// This harness runs every Twilio redirect snippet in examples/twilio
// against a live mocksms binary (see examples/README.md). Start one first:
//
//	bin/mocksms --memory --otp-code 123456 &
//	OTP_CODE=123456 go test -v -tags e2e ./examples/...
//
// Snippets self-assert and print E2E-OK <lang>; a missing runtime skips
// that snippet instead of failing.

type snippet struct {
	name  string
	dir   string
	look  []string
	probe []string
	cmd   []string
}

func twilioSnippets() []snippet {
	return []snippet{
		{name: "node", dir: "twilio/node", look: []string{"node"}, probe: []string{"--version"}, cmd: []string{"node", "send-sms.js"}},
		{name: "python", dir: "twilio/python", look: []string{"python3", "python"}, probe: []string{"--version"}, cmd: nil},
		{name: "php", dir: "twilio/php", look: []string{"php"}, probe: []string{"--version"}, cmd: []string{"php", "send_sms.php"}},
		{name: "go", dir: "twilio/go", look: []string{"go"}, probe: []string{"version"}, cmd: []string{"go", "run", "."}},
		{name: "csharp", dir: "twilio/csharp", look: []string{"dotnet"}, probe: []string{"--list-sdks"}, cmd: []string{"dotnet", "run"}},
	}
}

// lookPath finds the first usable runtime. The binary must exist and its
// probe command must succeed: bare LookPath is not enough (Windows ships a
// Store stub for python3, and a runtime-only dotnet answers LookPath but
// cannot run anything).
func lookPath(names, probe []string) string {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		out, err := exec.CommandContext(ctx, path, probe...).CombinedOutput()
		cancel()
		if err != nil || (name == "dotnet" && len(strings.TrimSpace(string(out))) == 0) {
			continue
		}
		return path
	}
	return ""
}

func TestTwilioSnippets(t *testing.T) {
	for _, snip := range twilioSnippets() {
		t.Run(snip.name, func(t *testing.T) {
			runtime := lookPath(snip.look, snip.probe)
			if runtime == "" {
				t.Skipf("runtime %q not installed", snip.look[0])
			}
			cmd := snip.cmd
			if cmd == nil {
				// Python: use whichever interpreter was found.
				cmd = []string{runtime, "send_sms.py"}
			} else {
				cmd[0] = runtime
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
			c.Dir = snip.dir
			c.Env = append(os.Environ(),
				"MOCKSMS_URL="+envOr("MOCKSMS_URL", "http://127.0.0.1:4010"),
				"OTP_CODE="+envOr("OTP_CODE", "123456"),
			)
			out, err := c.CombinedOutput()
			t.Logf("%s output:\n%s", snip.name, out)
			if err != nil {
				t.Fatalf("%s failed: %v\n%s", snip.name, err, out)
			}
			if !strings.Contains(string(out), "E2E-OK") {
				t.Fatalf("%s missing E2E-OK marker\n%s", snip.name, out)
			}
		})
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
