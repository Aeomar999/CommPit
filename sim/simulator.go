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

// New creates a new Simulator with default rules.
func New(opts ...Option) *Simulator {
	s := &Simulator{
		rules:     DefaultRules(),
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
			s.applyEffectToResult(rule.Effect, res)
			if latency > 0 {
				res.Delay += latency
			}
			return nil, res
		}
	}

	// 2. Evaluate pattern rules (only ...999901 to ...999905 are magic numbers)
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
		matchTo := cleanPhone(match.To)
		reqTo := ""
		if len(req.To) > 0 {
			reqTo = cleanPhone(req.To[0])
		}
		if !matchString(matchTo, reqTo) {
			return false
		}
	}
	if match.From != "" && !matchString(cleanPhone(match.From), cleanPhone(req.From)) {
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
	to := cleanPhone(req.To[0])
	if len(to) < 6 {
		return false
	}
	suffix := to[len(to)-6:]
	if suffix[:5] != "99990" {
		return false
	}
	digit := suffix[5]
	return digit >= '1' && digit <= '5'
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
	if effect.RateLimit > 0 {
		res.RateLimited = true
	}
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
