package sim

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

const errInvalidNumber = "invalid_number"

func TestSimulator_MagicNumbersTable(t *testing.T) {
	s := New()
	ctx := context.Background()

	tests := []struct {
		name         string
		to           string
		expectReject string
		expectAsync  string
		expectHang   time.Duration
	}{
		{
			name:         "Default rule 0001",
			to:           "+15005550001",
			expectReject: errInvalidNumber,
		},
		{
			name:         "Default rule 0002",
			to:           "+15005550002",
			expectReject: "unroutable",
		},
		{
			name:         "Default rule 0004",
			to:           "+15005550004",
			expectReject: "unsubscribed",
		},
		{
			name:         "Default rule 0009",
			to:           "+15005550009",
			expectReject: "not_sms_capable",
		},
		{
			name:         "Pattern 999901 - Invalid number",
			to:           "+1555999901",
			expectReject: errInvalidNumber,
		},
		{
			name:         "Pattern 999901 formatted with spaces",
			to:           "+1 555 999 901",
			expectReject: errInvalidNumber,
		},
		{
			name:        "Pattern 999902 - Unknown handset",
			to:          "+1555999902",
			expectAsync: "30005",
		},
		{
			name:         "Pattern 999903 - Rate limited",
			to:           "+1555999903",
			expectReject: "rate_limited",
		},
		{
			name:       "Pattern 999904 - Hang 60s",
			to:         "+1555999904",
			expectHang: 60 * time.Second,
		},
		{
			name:        "Pattern 999905 - Carrier filtered",
			to:          "+1555999905",
			expectAsync: "30007",
		},
		{
			name: "Non-magic suffix 999906 should not trigger pattern",
			to:   "+1555999906",
		},
		{
			name: "Non-magic suffix 999900 should not trigger pattern",
			to:   "+1555999900",
		},
		{
			name: "Normal number",
			to:   "+15555550123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := core.SendRequest{
				Channel: core.ChannelSMS,
				From:    "+15005550000",
				To:      []string{tc.to},
			}
			err, res := s.Evaluate(ctx, "prj_test", req)

			if tc.expectReject != "" {
				if err == nil {
					t.Fatalf("expected reject %q, got nil error", tc.expectReject)
				}
				if string(err.Code) != tc.expectReject {
					t.Errorf("expected reject code %q, got %q", tc.expectReject, err.Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected reject error: %v", err)
			}

			if tc.expectAsync != "" {
				if res.AsyncFail == nil {
					t.Fatalf("expected AsyncFail %q, got nil", tc.expectAsync)
				}
				if res.AsyncFail.ErrorCode != tc.expectAsync {
					t.Errorf("expected AsyncFail code %q, got %q", tc.expectAsync, res.AsyncFail.ErrorCode)
				}
			} else if res.AsyncFail != nil {
				t.Errorf("unexpected AsyncFail: %v", res.AsyncFail)
			}

			if tc.expectHang > 0 {
				if res.Hang != tc.expectHang {
					t.Errorf("expected Hang %v, got %v", tc.expectHang, res.Hang)
				}
			} else if res.Hang > 0 {
				t.Errorf("unexpected Hang: %v", res.Hang)
			}
		})
	}
}

func TestSimulator_LatencyPlusFailureCombination(t *testing.T) {
	ctx := context.Background()
	latency := 250 * time.Millisecond

	// Injected rand function that always returns 0.0 (< failureRate) to trigger failure
	s := New(WithRandFloat(func() float64 {
		return 0.0
	}))
	s.SetLatency(latency)
	s.SetFailureRate(0.5)

	// 1. Normal number: should receive BOTH latency delay AND async failure
	req := core.SendRequest{
		Channel: core.ChannelSMS,
		From:    "+15005550000",
		To:      []string{"+15555550123"},
	}
	err, res := s.Evaluate(ctx, "prj_test", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Delay != latency {
		t.Errorf("expected Delay %v, got %v", latency, res.Delay)
	}
	if res.AsyncFail == nil {
		t.Fatal("expected AsyncFail to be set from failure rate, got nil")
	}
	if res.AsyncFail.ErrorCode != "failed" {
		t.Errorf("expected ErrorCode 'failed', got %q", res.AsyncFail.ErrorCode)
	}

	// 2. Pattern 999902: should receive BOTH pattern failure AND global latency
	reqPattern := core.SendRequest{
		Channel: core.ChannelSMS,
		From:    "+15005550000",
		To:      []string{"+1555999902"},
	}
	err2, res2 := s.Evaluate(ctx, "prj_test", reqPattern)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if res2.Delay != latency {
		t.Errorf("expected Delay %v, got %v", latency, res2.Delay)
	}
	if res2.AsyncFail == nil || res2.AsyncFail.ErrorCode != "30005" {
		t.Errorf("expected pattern AsyncFail 30005, got %v", res2.AsyncFail)
	}
}

func TestSimulator_ConcurrentEvaluateUnderRace(t *testing.T) {
	s := New()
	s.SetLatency(10 * time.Millisecond)
	s.SetFailureRate(0.2)
	ctx := context.Background()

	var wg sync.WaitGroup
	numbers := []string{
		"+15005550001",
		"+15005550002",
		"+1555999901",
		"+1555999902",
		"+1555999903",
		"+1555999905",
		"+15555550123",
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		to := numbers[i%len(numbers)]
		go func(target string) {
			defer wg.Done()
			req := core.SendRequest{
				Channel: core.ChannelSMS,
				From:    "+15005550000",
				To:      []string{target},
			}
			err, res := s.Evaluate(ctx, "prj_concurrent", req)
			if target == "+15005550001" && err == nil {
				t.Errorf("expected rejection for invalid number")
			}
			if target == "+1555999902" && (res == nil || res.AsyncFail == nil) {
				t.Errorf("expected async fail for 999902")
			}
		}(to)
	}

	wg.Wait()
}
