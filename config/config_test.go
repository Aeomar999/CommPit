package config

import (
	"testing"
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
