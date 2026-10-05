package core

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/Aeomar999/CommPit/phone"
)

type simulatorImpl struct {
	mu          sync.RWMutex
	rules       []SimRule
	latency     time.Duration
	failureRate float64
	rand        *rand.Rand
}

func NewSimulator() Simulator {
	s := &simulatorImpl{
		rules: defaultSimRules(),
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return s
}

func defaultSimRules() []SimRule {
	return []SimRule{
		{
			Match:  SimMatch{To: "+15005550001"},
			Effect: SimEffect{Reject: NewInvalidNumber("invalid number", "to")},
		},
		{
			Match:  SimMatch{To: "+15005550002"},
			Effect: SimEffect{Reject: NewUnroutable("unroutable", "to")},
		},
		{
			Match:  SimMatch{To: "+15005550004"},
			Effect: SimEffect{Reject: NewUnsubscribed("unsubscribed", "to")},
		},
		{
			Match:  SimMatch{To: "+15005550009"},
			Effect: SimEffect{Reject: NewNotSMSCapable("not sms capable", "to")},
		},
	}
}

func (s *simulatorImpl) SetRules(rules []SimRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = rules
}

func (s *simulatorImpl) SetLatency(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latency = latency
}

func (s *simulatorImpl) SetFailureRate(rate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureRate = rate
}

func (s *simulatorImpl) Evaluate(ctx context.Context, projectID string, req SendRequest) (*Error, *SimResult) {
	s.mu.RLock()
	rules := s.rules
	latency := s.latency
	failureRate := s.failureRate
	s.mu.RUnlock()

	for _, rule := range rules {
		if s.matchRule(rule.Match, projectID, req) {
			return s.applyEffect(rule.Effect)
		}
	}

	// Check for pattern-based rules (99990X suffix)
	if s.matchPatternRules(req) {
		return s.applyPatternEffect(req)
	}

	// Apply global latency
	if latency > 0 {
		return nil, &SimResult{Delay: latency}
	}

	// Apply global failure rate
	if failureRate > 0 && s.rand.Float64() < failureRate {
		return nil, &SimResult{
			AsyncFail: &AsyncFail{
				ErrorCode:    "failed",
				ErrorMessage: "random failure",
			},
		}
	}

	return nil, &SimResult{}
}

func (s *simulatorImpl) matchRule(match SimMatch, projectID string, req SendRequest) bool {
	if match.To != "" {
		matchTo := phone.Normalize(match.To)
		reqTo := ""
		if len(req.To) > 0 {
			reqTo = phone.Normalize(req.To[0])
		}
		if !matchString(matchTo, reqTo) {
			return false
		}
	}
	if match.From != "" && !matchString(match.From, req.From) {
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

func (s *simulatorImpl) matchPatternRules(req SendRequest) bool {
	if len(req.To) == 0 {
		return false
	}
	to := phone.Normalize(req.To[0])
	if len(to) < 6 {
		return false
	}
	suffix := to[len(to)-6:]
	return suffix[:5] == "99990"
}

func (s *simulatorImpl) applyPatternEffect(req SendRequest) (*Error, *SimResult) {
	if len(req.To) == 0 {
		return nil, &SimResult{}
	}
	to := phone.Normalize(req.To[0])
	if len(to) < 6 {
		return nil, &SimResult{}
	}
	digit := to[len(to)-1]

	switch digit {
	case '1':
		return NewInvalidNumber("invalid number", "to"), &SimResult{}
	case '2':
		return nil, &SimResult{
			AsyncFail: &AsyncFail{
				ErrorCode:    "30005",
				ErrorMessage: "unknown handset",
			},
		}
	case '3':
		return NewRateLimited("rate limited", "to"), &SimResult{}
	case '4':
		return nil, &SimResult{Hang: 60 * time.Second}
	case '5':
		return nil, &SimResult{
			AsyncFail: &AsyncFail{
				ErrorCode:    "30007",
				ErrorMessage: "carrier filtered",
			},
		}
	}
	return nil, &SimResult{}
}

func (s *simulatorImpl) applyEffect(effect SimEffect) (*Error, *SimResult) {
	if effect.Reject != nil {
		return effect.Reject, &SimResult{}
	}
	if effect.FailAsync != nil {
		return nil, &SimResult{AsyncFail: effect.FailAsync}
	}
	if effect.Delay > 0 {
		return nil, &SimResult{Delay: effect.Delay}
	}
	if effect.Hang > 0 {
		return nil, &SimResult{Hang: effect.Hang}
	}
	if effect.RateLimit > 0 {
		return nil, &SimResult{RateLimited: true}
	}
	return nil, &SimResult{}
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
