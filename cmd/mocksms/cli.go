package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client subcommands talk to a running server (REQ-102). The --host and
// --port flags select its address. Usage errors exit 2; request failures
// exit 1.

func serverBaseURL(host string, port int) string {
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func apiErrorBody(body []byte) string {
	var decoded struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return strings.TrimSpace(string(body))
	}
	if decoded.Error.Message != "" {
		return decoded.Error.Message
	}
	return strings.TrimSpace(string(body))
}

func apiCall(ctx context.Context, method, rawURL string, params map[string]string) (int, []byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0, nil, err
	}
	query := parsed.Query()
	for key, value := range params {
		if value != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, parsed.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Mocksms", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

// runCLI dispatches client subcommands, writing results to stdout and
// diagnostics to stderr. It returns the process exit code.
func runCLI(host string, port int, stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: mocksms [serve] | otp latest | messages wait | reset")
		return 2
	}
	base := serverBaseURL(host, port)
	switch args[0] {
	case "otp":
		return runOTPLatest(base, stdout, stderr, args[1:])
	case "messages":
		return runMessagesWait(base, stdout, stderr, args[1:])
	case "reset":
		return runReset(base, stdout, stderr, args[1:])
	default:
		fmt.Fprintf(stderr, "unknown command %q\nusage: mocksms [serve] | otp latest | messages wait | reset\n", args[0])
		return 2
	}
}

// splitFlags partitions arguments into positionals and flags so flag order
// is free: `otp latest --to X` and `otp --to X latest` both work. Every
// flag takes a value, either `--flag value` or `--flag=value`.
func splitFlags(args []string) (positionals []string, flags []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			if strings.Contains(a, "=") {
				flags = append(flags, a)
				continue
			}
			flags = append(flags, a)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return positionals, flags
}

func runOTPLatest(base string, stdout, stderr io.Writer, args []string) int {
	positionals, flags := splitFlags(args)
	fs := flag.NewFlagSet("otp latest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	to := fs.String("to", "", "Recipient to look up (required)")
	project := fs.String("project", "", "Project ULID (default project when empty)")
	if err := fs.Parse(flags); err != nil {
		return 2
	}
	if len(positionals) == 0 || positionals[0] != "latest" {
		fmt.Fprintln(stderr, "usage: mocksms otp latest --to <recipient> [--project <id>]")
		return 2
	}
	if *to == "" {
		fmt.Fprintln(stderr, "otp latest requires --to")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	status, body, err := apiCall(ctx, http.MethodGet, base+"/api/v1/otp/latest", map[string]string{
		"to":      *to,
		"project": *project,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if status != http.StatusOK {
		fmt.Fprintf(stderr, "Error: %s\n", apiErrorBody(body))
		return 1
	}
	var resp struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Code == "" {
		fmt.Fprintln(stderr, "Error: unexpected response")
		return 1
	}
	fmt.Fprintln(stdout, resp.Code)
	return 0
}

func runMessagesWait(base string, stdout, stderr io.Writer, args []string) int {
	positionals, flags := splitFlags(args)
	fs := flag.NewFlagSet("messages wait", flag.ContinueOnError)
	fs.SetOutput(stderr)
	to := fs.String("to", "", "Only match this recipient")
	channel := fs.String("channel", "", "sms or email")
	timeout := fs.Int("timeout", 10, "Wait budget in seconds (max 60)")
	since := fs.String("since", "", "Only match messages after this RFC3339 time")
	project := fs.String("project", "", "Project ULID (default project when empty)")
	if err := fs.Parse(flags); err != nil {
		return 2
	}
	if len(positionals) == 0 || positionals[0] != "wait" {
		fmt.Fprintln(stderr, "usage: mocksms messages wait [--to <r>] [--channel <c>] [--timeout <s>] [--since <rfc3339>] [--project <id>]")
		return 2
	}
	if *since != "" {
		if _, err := time.Parse(time.RFC3339, *since); err != nil {
			fmt.Fprintln(stderr, "since must be RFC3339, e.g. 2026-10-12T00:00:00Z")
			return 2
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	status, body, err := apiCall(ctx, http.MethodGet, base+"/api/v1/messages/wait", map[string]string{
		"to":      *to,
		"channel": *channel,
		"timeout": fmt.Sprint(*timeout),
		"since":   *since,
		"project": *project,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if status != http.StatusOK {
		fmt.Fprintf(stderr, "Error: %s\n", apiErrorBody(body))
		return 1
	}
	fmt.Fprintln(stdout, strings.TrimSpace(string(body)))
	return 0
}

func runReset(base string, stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "Project ULID to reset (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *project == "" {
		fmt.Fprintln(stderr, "reset requires --project")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	status, body, err := apiCall(ctx, http.MethodDelete, base+"/api/v1/messages", map[string]string{
		"project": *project,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		fmt.Fprintf(stderr, "Error: %s\n", apiErrorBody(body))
		return 1
	}
	fmt.Fprintf(stdout, "project %s reset\n", *project)
	return 0
}
