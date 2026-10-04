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
	DataDir        string
	Memory         bool
	Store          StoreConfig
	Lifecycle      LifecycleConfig
	OTP            OTPConfig
	Validation     ValidationConfig
	Sim            SimConfig
	UIAuth         string
	NoDockerHostRewrite bool
	Version        bool
}

type StoreConfig struct {
	ReadPoolSize int
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
	Latency      time.Duration
	FailureRate  float64
}

func Load() *Config {
	k := koanf.New(".")

	k.Load(env.Provider("MOCKSMS_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(s), "_", ".")
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
	cfg.Lifecycle.StepDelay = getDuration(k, "lifecycle.step_delay", 300*time.Millisecond)
	cfg.OTP.FixedCode = getString(k, "otp.fixed_code", "")
	cfg.Validation.Phone = getString(k, "validation.phone", "valid")
	cfg.Sim.Latency = getDuration(k, "sim.latency", 0)
	cfg.Sim.FailureRate = getFloat64(k, "sim.failure_rate", 0)
	cfg.UIAuth = getString(k, "ui_auth", "")
	cfg.NoDockerHostRewrite = getBool(k, "no_docker_host_rewrite", false)

	return cfg
}

func getString(k *koanf.Koanf, key, def string) string {
	if k.Exists(key) {
		return k.String(key)
	}
	return def
}

func getInt(k *koanf.Koanf, key string, def int) int {
	if k.Exists(key) {
		return k.Int(key)
	}
	return def
}

func getBool(k *koanf.Koanf, key string, def bool) bool {
	if k.Exists(key) {
		return k.Bool(key)
	}
	return def
}

func getDuration(k *koanf.Koanf, key string, def time.Duration) time.Duration {
	if k.Exists(key) {
		return k.Duration(key)
	}
	return def
}

func getFloat64(k *koanf.Koanf, key string, def float64) float64 {
	if k.Exists(key) {
		return k.Float64(key)
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