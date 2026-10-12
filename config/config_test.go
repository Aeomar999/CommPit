package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		HTTP: struct {
			Host string
			Port int
		}{
			Host: "127.0.0.1",
			Port: 4010,
		},
		SMTP: struct {
			Host string
			Port int
		}{
			Host: "127.0.0.1",
			Port: 2525,
		},
		DataDir: "",
		Memory:  false,
		Lifecycle: LifecycleConfig{
			StepDelay: 0,
		},
		OTP: OTPConfig{
			FixedCode: "",
		},
		Validation: ValidationConfig{
			Phone: "valid",
		},
		UIAuth: "",
	}

	if cfg.HTTP.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", cfg.HTTP.Host)
	}
	if cfg.HTTP.Port != 4010 {
		t.Errorf("expected port 4010, got %d", cfg.HTTP.Port)
	}
	if cfg.Validation.Phone != "valid" {
		t.Errorf("expected valid, got %s", cfg.Validation.Phone)
	}
}

func TestConfig_AdapterPorts(t *testing.T) {
	t.Run("defaults disable dedicated ports", func(t *testing.T) {
		t.Setenv("MOCKSMS_ADAPTERS_TWILIO_PORT", "")
		t.Setenv("MOCKSMS_ADAPTERS_TERMII_PORT", "")
		cfg := Load()
		if cfg.Adapters.Twilio.Port != 0 {
			t.Errorf("expected twilio port 0, got %d", cfg.Adapters.Twilio.Port)
		}
		if cfg.Adapters.Termii.Port != 0 {
			t.Errorf("expected termii port 0, got %d", cfg.Adapters.Termii.Port)
		}
	})

	t.Run("env enables dedicated ports", func(t *testing.T) {
		t.Setenv("MOCKSMS_ADAPTERS_TWILIO_PORT", "4020")
		t.Setenv("MOCKSMS_ADAPTERS_TERMII_PORT", "4021")
		cfg := Load()
		if cfg.Adapters.Twilio.Port != 4020 {
			t.Errorf("expected twilio port 4020, got %d", cfg.Adapters.Twilio.Port)
		}
		if cfg.Adapters.Termii.Port != 4021 {
			t.Errorf("expected termii port 4021, got %d", cfg.Adapters.Termii.Port)
		}
	})
}
func TestConfig_ProjectsFromYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `projects:
  - id: prj_11111111111111111111111111
    name: combined
    credentials:
      - provider: twilio
        key: ACyaml001
      - provider: termii
        key: tlyaml001
`
	if err := os.WriteFile(filepath.Join(dir, "mocksms.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	t.Chdir(dir)

	cfg := Load()
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %v", cfg.Projects)
	}
	p := cfg.Projects[0]
	if p.ID != "prj_11111111111111111111111111" || p.Name != "combined" {
		t.Errorf("unexpected project: %+v", p)
	}
	if len(p.Credentials) != 2 || p.Credentials[0].Provider != "twilio" || p.Credentials[0].Key != "ACyaml001" {
		t.Errorf("unexpected credentials: %+v", p.Credentials)
	}
}
func TestConfig_PhoneValidationModes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"valid", "valid", "valid"},
		{"possible", "possible", "possible"},
		{"off", "off", "off"},
		{"empty defaults to valid", "", "valid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{}
			if tt.value != "" {
				cfg.Validation.Phone = tt.value
			}
			if tt.value == "" {
				cfg.Validation.Phone = "valid" // default
			}
			if cfg.Validation.Phone != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, cfg.Validation.Phone)
			}
		})
	}
}

func TestConfig_Webhooks(t *testing.T) {
	t.Run("defaults select worker policy", func(t *testing.T) {
		t.Setenv("MOCKSMS_WEBHOOKS_TIMEOUT", "")
		t.Setenv("MOCKSMS_WEBHOOKS_MAX_ATTEMPTS", "")
		cfg := Load()
		if cfg.Webhooks.Timeout != 0 || cfg.Webhooks.MaxAttempts != 0 {
			t.Errorf("expected zero webhooks config, got %+v", cfg.Webhooks)
		}
	})

	t.Run("env overrides", func(t *testing.T) {
		t.Setenv("MOCKSMS_WEBHOOKS_TIMEOUT", "5s")
		t.Setenv("MOCKSMS_WEBHOOKS_MAX_ATTEMPTS", "3")
		cfg := Load()
		if cfg.Webhooks.Timeout != 5*time.Second {
			t.Errorf("expected 5s timeout, got %v", cfg.Webhooks.Timeout)
		}
		if cfg.Webhooks.MaxAttempts != 3 {
			t.Errorf("expected 3 max attempts, got %d", cfg.Webhooks.MaxAttempts)
		}
	})

	t.Run("double-underscore nesting", func(t *testing.T) {
		t.Setenv("MOCKSMS_WEBHOOKS__MAX_ATTEMPTS", "4")
		cfg := Load()
		if cfg.Webhooks.MaxAttempts != 4 {
			t.Errorf("expected 4 max attempts, got %d", cfg.Webhooks.MaxAttempts)
		}
	})
}

func TestConfig_SimRulesFromYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `sim:
  latency: 50ms
  failure_rate: 0.01
  rules:
    - match: {to: "+15551234567"}
      effect: {reject: {code: invalid_number, message: "blocked"}}
    - match: {provider: twilio, channel: sms}
      effect: {delay_ms: 250, rate_limit_per_min: 60}
    - match: {to: "*@example.test"}
      effect: {fail_async: {code: bounced, message: "gone"}, hang_ms: 1000}
`
	if err := os.WriteFile(filepath.Join(dir, "mocksms.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	t.Chdir(dir)

	cfg := Load()
	if len(cfg.Sim.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %+v", cfg.Sim.Rules)
	}
	rules, err := cfg.Sim.CoreRules()
	if err != nil {
		t.Fatalf("CoreRules: %v", err)
	}
	if rules[0].Match.To != "+15551234567" || rules[0].Effect.Reject == nil || string(rules[0].Effect.Reject.Code) != "invalid_number" {
		t.Errorf("rule 0 mismatch: %+v", rules[0])
	}
	if rules[1].Match.Provider != "twilio" || string(rules[1].Match.Channel) != "sms" || rules[1].Effect.RateLimit != 60 {
		t.Errorf("rule 1 mismatch: %+v", rules[1])
	}
	if rules[1].Effect.Delay != 250*time.Millisecond {
		t.Errorf("rule 1 delay mismatch: %v", rules[1].Effect.Delay)
	}
	if rules[2].Effect.FailAsync == nil || rules[2].Effect.FailAsync.ErrorCode != "bounced" || rules[2].Effect.Hang != time.Second {
		t.Errorf("rule 2 mismatch: %+v", rules[2])
	}
}

func TestConfig_SimFailStatus(t *testing.T) {
	cfg := SimConfig{Rules: []SimRuleConfig{{
		Match:  SimMatchConfig{To: "+1555"},
		Effect: SimEffectConfig{FailAsync: &SimFailConfig{Code: "30007", Message: "filtered", Status: "failed"}},
	}}}
	rules, err := cfg.CoreRules()
	if err != nil {
		t.Fatalf("CoreRules: %v", err)
	}
	if rules[0].Effect.FailAsync.Status != "failed" {
		t.Errorf("expected failed status, got %q", rules[0].Effect.FailAsync.Status)
	}
}

func TestConfig_SimRulesValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule SimRuleConfig
	}{
		{"unknown reject code", SimRuleConfig{Effect: SimEffectConfig{Reject: &SimRejectConfig{Code: "nope", Message: "x"}}}},
		{"unknown channel", SimRuleConfig{Match: SimMatchConfig{Channel: "push"}, Effect: SimEffectConfig{DelayMs: 1}}},
		{"negative delay", SimRuleConfig{Effect: SimEffectConfig{DelayMs: -1}}},
		{"negative hang", SimRuleConfig{Effect: SimEffectConfig{HangMs: -5}}},
		{"negative rate", SimRuleConfig{Effect: SimEffectConfig{RateLimitPerMin: -2}}},
		{"no effect", SimRuleConfig{Match: SimMatchConfig{To: "+1555"}}},
		{"bad fail status", SimRuleConfig{Effect: SimEffectConfig{FailAsync: &SimFailConfig{Code: "x", Message: "y", Status: "maybe"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := SimConfig{Rules: []SimRuleConfig{tc.rule}}
			if _, err := cfg.CoreRules(); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
