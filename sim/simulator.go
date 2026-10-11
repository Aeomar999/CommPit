// Package sim implements the simulation engine for message delivery lifecycles,
// pattern numbers, latency, and failure rates.
package sim

import (
	"context"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/phone"
)

func cleanPhone(s string) string {
	norm := phone.Normalize(s)
	var b strings.Builder
	for _, r := range norm {
		if (r >= '0' && r <= '9') || r == '+' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Simulator evaluates simulation rules, pattern numbers, latency, and failure rates.
type Simulator struct {
	mu          sync.RWMutex
	rules       []core.SimRule
	latency     time.Duration
	failureRate float64
	clock       core.Clock
	windows     map[string][]time.Time
	randMu      sync.Mutex
	randFloat   func() float64
}

// Option configures a Simulator.
type Option func(*Simulator)

// WithRandFloat overrides the random float generator (e.g. for deterministic unit testing).
func WithRandFloat(fn func() float64) Option {
	return func(s *Simulator) {
		s.randFloat = fn
	}
}

// WithClock overrides the clock used for rate-limit windows (e.g. the fake
// clock in tests). Defaults to the real clock.
func WithClock(clock core.Clock) Option {
	return func(s *Simulator) {
		s.clock = clock
	}
}

// New creates a new Simulator with default rules.
func New(opts ...Option) *Simulator {
	s := &Simulator{
		rules:     DefaultRules(),
		clock:     core.RealClock{},
		windows:   map[string][]time.Time{},
		randFloat: rand.Float64,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewSimulator creates a new Simulator with default rules (alias for New).
func NewSimulator(opts ...Option) *Simulator {
	return New(opts...)
}

func (s *Simulator) randomFloat() float64 {
	s.randMu.Lock()
	defer s.randMu.Unlock()
	if s.randFloat != nil {
		return s.randFloat()
	}
	//nolint:gosec // simulation random failure does not require cryptographic randomness
	return rand.Float64()
}

// DefaultRules returns the standard simulator rules.
func DefaultRules() []core.SimRule {
	return []core.SimRule{
		{
			Match:  core.SimMatch{To: "+15005550001"},
			Effect: core.SimEffect{Reject: core.NewInvalidNumber("invalid number", "to")},
		},
		{
			Match:  core.SimMatch{To: "+15005550002"},
			Effect: core.SimEffect{Reject: core.NewUnroutable("unroutable", "to")},
		},
		{
			Match:  core.SimMatch{To: "+15005550004"},
			Effect: core.SimEffect{Reject: core.NewUnsubscribed("unsubscribed", "to")},
		},
		{
			Match:  core.SimMatch{To: "+15005550009"},
			Effect: core.SimEffect{Reject: core.NewNotSMSCapable("not sms capable", "to")},
		},
		{
			Match:  core.SimMatch{To: "*@invalid.mocksms.test"},
			Effect: core.SimEffect{Reject: core.NewInvalidAddress("invalid address", "to")},
		},
		{
			Match: core.SimMatch{To: "*@bounce.mocksms.test"},
			Effect: core.SimEffect{FailAsync: &core.AsyncFail{
				ErrorCode:    "bounced",
				ErrorMessage: "mailbox unavailable",
				Status:       core.StatusFailed,
			}},
		},
	}
}

// SetRules updates the explicit simulation rules.
func (s *Simulator) SetRules(rules []core.SimRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = rules
}

// SetLatency sets the global simulated latency.
func (s *Simulator) SetLatency(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latency = latency
}

// SetFailureRate sets the global simulated random failure rate.
func (s *Simulator) SetFailureRate(rate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureRate = rate
}

// Evaluate evaluates a request against rules, pattern numbers, latency, and failure rates.
func (s *Simulator) Evaluate(_ context.Context, projectID string, req core.SendRequest) (*core.Error, *core.SimResult) {
	s.mu.RLock()
	rules := s.rules
	latency := s.latency
	failureRate := s.failureRate
	s.mu.RUnlock()

	res := &core.SimResult{}

	// 1. Evaluate explicit rules first
	for _, rule := range rules {
		if s.matchRule(rule.Match, projectID, req) {
			if rule.Effect.Reject != nil {
				return rule.Effect.Reject, res
			}
			if rule.Effect.RateLimit > 0 && !s.checkRateLimit(projectID, req.Provider, rule.Effect.RateLimit) {
				res.RateLimited = true
				return nil, res
			}
			s.applyEffectToResult(rule.Effect, res)
			if latency > 0 {
				res.Delay += latency
			}
			return nil, res
		}
	}

	// 2. Evaluate pattern rules (...999901 to ...999906 are magic numbers)
	if s.matchPatternRules(req) {
		err, patternRes := s.applyPatternEffect(req)
		if err != nil {
			return err, res
		}
		if patternRes.AsyncFail != nil {
			res.AsyncFail = patternRes.AsyncFail
		}
		if patternRes.Hang > 0 {
			res.Hang = patternRes.Hang
		}
		if patternRes.Delay > 0 {
			res.Delay += patternRes.Delay
		}
		if patternRes.RateLimited {
			res.RateLimited = true
		}
		if latency > 0 {
			res.Delay += latency
		}
		return nil, res
	}

	// 3. Global effects (latency and random failure combine)
	if latency > 0 {
		res.Delay += latency
	}

	if failureRate > 0 && s.randomFloat() < failureRate {
		res.AsyncFail = &core.AsyncFail{
			ErrorCode:    "failed",
			ErrorMessage: "random failure",
		}
	}

	return nil, res
}

func (s *Simulator) matchRule(match core.SimMatch, projectID string, req core.SendRequest) bool {
	if match.To != "" {
		reqTo := ""
		if len(req.To) > 0 {
			reqTo = req.To[0]
		}
		if !matchRecipient(match.To, reqTo) {
			return false
		}
	}
	if match.From != "" && !matchRecipient(match.From, req.From) {
		return false
	}
	if match.Provider != "" && match.Provider != req.Provider {
		return false
	}
	if match.Project != "" && match.Project != projectID {
		return false
	}
	if match.Channel != "" && match.Channel != req.Channel {
		return false
	}
	return true
}

func (s *Simulator) matchPatternRules(req core.SendRequest) bool {
	if len(req.To) == 0 {
		return false
	}
	if strings.Contains(req.To[0], "@") {
		return false
	}
	to := cleanPhone(req.To[0])
	if len(to) < 6 {
		return false
	}
	suffix := to[len(to)-6:]
	if suffix[:5] != "99990" {
		return false
	}
	digit := suffix[5]
	return digit >= '1' && digit <= '6'
}

func (s *Simulator) applyPatternEffect(req core.SendRequest) (*core.Error, *core.SimResult) {
	if len(req.To) == 0 {
		return nil, &core.SimResult{}
	}
	to := cleanPhone(req.To[0])
	if len(to) < 6 {
		return nil, &core.SimResult{}
	}
	digit := to[len(to)-1]

	switch digit {
	case '1':
		return core.NewInvalidNumber("invalid number", "to"), &core.SimResult{}
	case '2':
		return nil, &core.SimResult{
			AsyncFail: &core.AsyncFail{
				ErrorCode:    "30005",
				ErrorMessage: "unknown handset",
			},
		}
	case '3':
		return core.NewRateLimited("rate limited", "to"), &core.SimResult{}
	case '4':
		return nil, &core.SimResult{Hang: 60 * time.Second}
	case '5':
		return nil, &core.SimResult{
			AsyncFail: &core.AsyncFail{
				ErrorCode:    "30007",
				ErrorMessage: "carrier filtered",
				Status:       core.StatusFailed,
			},
		}
	case '6':
		return nil, &core.SimResult{
			AsyncFail: &core.AsyncFail{
				ErrorCode:    "30003",
				ErrorMessage: "unreachable handset",
			},
		}
	}
	return nil, &core.SimResult{}
}

func (s *Simulator) applyEffectToResult(effect core.SimEffect, res *core.SimResult) {
	if effect.FailAsync != nil {
		res.AsyncFail = effect.FailAsync
	}
	if effect.Delay > 0 {
		res.Delay += effect.Delay
	}
	if effect.Hang > 0 {
		res.Hang = effect.Hang
	}
}

// checkRateLimit enforces N sends per minute per project per provider with
// a sliding window. Allowed sends consume quota; rejected ones do not, so
// a burst cannot lock a sender out beyond one window.
func (s *Simulator) checkRateLimit(projectID, provider string, perMinute int) bool {
	if perMinute <= 0 {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	cutoff := now.Add(-time.Minute)
	key := projectID + "\x00" + provider
	kept := make([]time.Time, 0, len(s.windows[key]))
	for _, at := range s.windows[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= perMinute {
		s.windows[key] = kept
		return false
	}
	s.windows[key] = append(kept, now)
	return true
}

// matchRecipient compares a rule's To against a request recipient. Either
// side containing '@' switches to case-insensitive email matching (phone
// normalization would mangle addresses); otherwise both sides are phone
// cleaned first.
func matchRecipient(pattern, value string) bool {
	if strings.Contains(pattern, "@") || strings.Contains(value, "@") {
		return matchEmail(pattern, value)
	}
	return matchString(cleanPhone(pattern), cleanPhone(value))
}

// matchEmail compares addresses case-insensitively, with leading-'*'
// suffix wildcards (`*@invalid.mocksms.test`).
func matchEmail(pattern, value string) bool {
	pattern = strings.ToLower(pattern)
	value = strings.ToLower(value)
	if pattern == value {
		return true
	}
	if len(pattern) > 0 && pattern[0] == '*' {
		return len(value) >= len(pattern)-1 && value[len(value)-len(pattern)+1:] == pattern[1:]
	}
	return false
}

func matchString(pattern, value string) bool {
	if pattern == value {
		return true
	}
	if len(pattern) > 0 && pattern[0] == '*' {
		return len(value) >= len(pattern)-1 && value[len(value)-len(pattern)+1:] == pattern[1:]
	}
	return false
}
