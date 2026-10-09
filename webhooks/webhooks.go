// Package webhooks delivers persistent webhook callbacks to
// developer-configured URLs (spec §8.2).
//
// The worker drains WebhookDelivery rows woken by bus events, sends each
// with a timeout, and retries failures with backoff, recording every
// attempt. Producers (adapter notifiers in M3-02..M3-04, replay in M3-05)
// only create rows; delivery policy lives here.
package webhooks

import "time"

// DefaultBackoff is the wait applied after each failed attempt: the initial
// try plus these five waits make six sends at most.
var DefaultBackoff = []time.Duration{
	time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
}

// Config tunes delivery policy.
type Config struct {
	// Timeout bounds one send attempt. Default 10s.
	Timeout time.Duration
	// Backoff lists the wait after each failed attempt. Default
	// DefaultBackoff. MaxAttempts defaults to len(Backoff)+1.
	Backoff []time.Duration
	// MaxAttempts caps total sends per delivery. Default len(Backoff)+1.
	MaxAttempts int
	// BatchSize caps rows fetched per drain. Default 100.
	BatchSize int
}

// withDefaults fills zero values.
func (c Config) withDefaults() Config {
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}
	if len(c.Backoff) == 0 {
		c.Backoff = DefaultBackoff
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = len(c.Backoff) + 1
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	return c
}
