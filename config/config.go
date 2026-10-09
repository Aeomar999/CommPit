package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	HTTP struct {
		Host string
		Port int
	}
	SMTP struct {
		Host string
		Port int
	}
	DataDir             string
	Memory              bool
	Store               StoreConfig
	Projects            []ProjectLinkConfig
	Adapters            AdaptersConfig
	Lifecycle           LifecycleConfig
	OTP                 OTPConfig
	Validation          ValidationConfig
	Sim                 SimConfig
	Webhooks            WebhooksConfig
	Retention           RetentionConfig
	UIAuth              string
	NoDockerHostRewrite bool
	Version             bool
	Security            SecurityConfig
}

type SecurityConfig struct {
	AllowedHosts    []string
	RequireXMocksms bool
	UIAuth          string
}

type StoreConfig struct {
	ReadPoolSize int
}

// CredentialLinkConfig maps one provider credential to a project.
type CredentialLinkConfig struct {
	Provider string
	Key      string
}

// ProjectLinkConfig declares a project and the credentials linked to it.
type ProjectLinkConfig struct {
	ID          string
	Name        string
	Credentials []CredentialLinkConfig
}

// AdapterPortConfig holds the dedicated-port setting for one provider
// adapter. Port 0 (the default) disables the dedicated listener.
type AdapterPortConfig struct {
	Port int
}

// AdaptersConfig holds per-adapter settings.
type AdaptersConfig struct {
	Twilio AdapterPortConfig
	Termii AdapterPortConfig
}

type LifecycleConfig struct {
	StepDelay time.Duration
}

type OTPConfig struct {
	FixedCode string
}

type ValidationConfig struct {
	Phone string
}

type SimConfig struct {
	Latency     time.Duration
	FailureRate float64
}

type RetentionConfig struct {
	Enabled         bool
	Interval        time.Duration
	MessageTTL      time.Duration
	VerificationTTL time.Duration
	BatchTTL        time.Duration
	RequestLogTTL   time.Duration
	WebhookTTL      time.Duration
}

// WebhooksConfig tunes webhook delivery. Zero values select worker defaults.
type WebhooksConfig struct {
	Timeout     time.Duration
	MaxAttempts int
}

func Load() *Config {
	k := koanf.New(".")

	k.Load(env.Provider("MOCKSMS_", ".", func(s string) string {
		s = strings.TrimPrefix(s, "MOCKSMS_")
		s = strings.ToLower(s)
		// Double underscores separate nesting levels; single underscores
		// stay literal so multi-word segments (max_attempts, step_delay)
		// survive. Single-underscore names keep working through the
		// collapsed lookup fallback in the getters below.
		return strings.ReplaceAll(s, "__", ".")
	}), nil)

	configPaths := []string{
		"mocksms.yaml",
		filepath.Join(os.Getenv("HOME"), ".config", "mocksms", "mocksms.yaml"),
	}
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			k.Load(file.Provider(path), yaml.Parser())
			break
		}
	}

	cfg := &Config{}
	cfg.HTTP.Host = getString(k, "http.host", "127.0.0.1")
	cfg.HTTP.Port = getInt(k, "http.port", 4010)
	cfg.SMTP.Host = getString(k, "smtp.host", "127.0.0.1")
	cfg.SMTP.Port = getInt(k, "smtp.port", 1025)
	cfg.DataDir = getString(k, "data_dir", defaultDataDir())
	cfg.Memory = getBool(k, "memory", false)
	cfg.Store.ReadPoolSize = getInt(k, "store.read_pool_size", 4)
	if k.Exists("projects") {
		_ = k.Unmarshal("projects", &cfg.Projects)
	}
	cfg.Adapters.Twilio.Port = getInt(k, "adapters.twilio.port", 0)
	cfg.Adapters.Termii.Port = getInt(k, "adapters.termii.port", 0)
	cfg.Lifecycle.StepDelay = getDuration(k, "lifecycle.step_delay", 300*time.Millisecond)
	cfg.OTP.FixedCode = getString(k, "otp.fixed_code", "")
	cfg.Validation.Phone = getString(k, "validation.phone", "valid")
	cfg.Sim.Latency = getDuration(k, "sim.latency", 0)
	cfg.Sim.FailureRate = getFloat64(k, "sim.failure_rate", 0)
	cfg.Webhooks.Timeout = getDuration(k, "webhooks.timeout", 0)
	cfg.Webhooks.MaxAttempts = getInt(k, "webhooks.max_attempts", 0)
	cfg.UIAuth = getString(k, "ui_auth", "")
	cfg.NoDockerHostRewrite = getBool(k, "no_docker_host_rewrite", false)
	cfg.Security.AllowedHosts = getStringSlice(k, "security.allowed_hosts", []string{"127.0.0.1", "localhost"})
	cfg.Security.RequireXMocksms = getBool(k, "security.require_x_mocksms", true)
	cfg.Retention.Enabled = getBool(k, "retention.enabled", false)
	cfg.Retention.Interval = getDuration(k, "retention.interval", 1*time.Hour)
	cfg.Retention.MessageTTL = getDuration(k, "retention.message_ttl", 30*24*time.Hour)
	cfg.Retention.VerificationTTL = getDuration(k, "retention.verification_ttl", 7*24*time.Hour)
	cfg.Retention.BatchTTL = getDuration(k, "retention.batch_ttl", 30*24*time.Hour)
	cfg.Retention.RequestLogTTL = getDuration(k, "retention.request_log_ttl", 30*24*time.Hour)
	cfg.Retention.WebhookTTL = getDuration(k, "retention.webhook_ttl", 30*24*time.Hour)

	return cfg
}

func getString(k *koanf.Koanf, key, def string) string {
	for _, candidate := range keyVariants(key) {
		if k.Exists(candidate) {
			return k.String(candidate)
		}
	}
	return def
}

func getInt(k *koanf.Koanf, key string, def int) int {
	for _, candidate := range keyVariants(key) {
		if k.Exists(candidate) {
			return k.Int(candidate)
		}
	}
	return def
}

func getBool(k *koanf.Koanf, key string, def bool) bool {
	for _, candidate := range keyVariants(key) {
		if k.Exists(candidate) {
			return k.Bool(candidate)
		}
	}
	return def
}

func getDuration(k *koanf.Koanf, key string, def time.Duration) time.Duration {
	for _, candidate := range keyVariants(key) {
		if k.Exists(candidate) {
			return k.Duration(candidate)
		}
	}
	return def
}

func getFloat64(k *koanf.Koanf, key string, def float64) float64 {
	for _, candidate := range keyVariants(key) {
		if k.Exists(candidate) {
			return k.Float64(candidate)
		}
	}
	return def
}

// keyVariants returns the canonical key plus a collapsed form where dots
// become underscores. The collapsed form keeps single-underscore env names
// (MOCKSMS_ADAPTERS_TWILIO_PORT) working alongside double-underscore
// nesting (MOCKSMS_ADAPTERS__TWILIO__PORT); the canonical spelling wins.
func keyVariants(key string) []string {
	collapsed := strings.ReplaceAll(key, ".", "_")
	if collapsed == key {
		return []string{key}
	}
	return []string{key, collapsed}
}

func getStringSlice(k *koanf.Koanf, key string, def []string) []string {
	for _, candidate := range keyVariants(key) {
		if k.Exists(candidate) {
			var result []string
			k.Unmarshal(candidate, &result)
			return result
		}
	}
	return def
}

func defaultDataDir() string {
	if os.Getenv("MOCKSMS_IN_DOCKER") == "1" || exists("/.dockerenv") {
		return "/data"
	}
	if os.Getenv("LOCALAPPDATA") != "" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "mocksms")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "mocksms")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
