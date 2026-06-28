package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.HTTP.Addr() != "0.0.0.0:8080" {
		t.Fatalf("unexpected addr: %s", cfg.HTTP.Addr())
	}
	if cfg.HTTP.ShutdownTimeout != 10*time.Second {
		t.Fatalf("unexpected shutdown timeout: %s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.HTTP.MaxBodyBytes != 1<<20 {
		t.Fatalf("unexpected max body bytes: %d", cfg.HTTP.MaxBodyBytes)
	}
	if cfg.Database.DSN == "" {
		t.Fatal("expected postgres dsn")
	}
	if !cfg.RateLimit.Enabled {
		t.Fatal("expected rate limit to be enabled by default")
	}
	if cfg.RateLimit.Requests != 120 {
		t.Fatalf("unexpected rate limit requests: %d", cfg.RateLimit.Requests)
	}
}

func TestLoadRequiresPath(t *testing.T) {
	_, err := Load("")
	if err == nil {
		t.Fatal("expected missing config path error")
	}
	if !strings.Contains(err.Error(), "missing config path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadReadsJSONConfig(t *testing.T) {
	path := writeConfig(t, `{
  "app": {
    "name": "orders-api",
    "env": "test"
  },
  "http": {
    "port": "9000",
    "max_body_bytes": 2048,
    "shutdown_timeout": "3s"
  },
  "database": {
    "dsn": "postgres://test:test@localhost:5432/test?sslmode=disable",
    "max_conns": 25,
    "connect_timeout": "2s"
  },
  "rate_limit": {
    "enabled": false,
    "requests": 10,
    "window": "5s"
  },
  "auth": {
    "enabled": true,
    "secret": "test-secret-from-config-32-bytes",
    "issuer": "orders-api",
    "audience": "orders-clients",
    "clock_skew": "15s"
  },
  "log": {
    "level": "debug"
  }
}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != "orders-api" {
		t.Fatalf("unexpected app name: %s", cfg.App.Name)
	}
	if cfg.App.Env != "test" {
		t.Fatalf("unexpected app env: %s", cfg.App.Env)
	}
	if cfg.HTTP.Port != "9000" {
		t.Fatalf("unexpected port: %s", cfg.HTTP.Port)
	}
	if cfg.HTTP.ShutdownTimeout != 3*time.Second {
		t.Fatalf("unexpected shutdown timeout: %s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.HTTP.MaxBodyBytes != 2048 {
		t.Fatalf("unexpected max body bytes: %d", cfg.HTTP.MaxBodyBytes)
	}
	if got := cfg.Log.SlogLevel().String(); got != "DEBUG" {
		t.Fatalf("unexpected log level: %s", got)
	}
	if cfg.Database.DSN != "postgres://test:test@localhost:5432/test?sslmode=disable" {
		t.Fatalf("unexpected database dsn: %s", cfg.Database.DSN)
	}
	if cfg.Database.MaxConns != 25 {
		t.Fatalf("unexpected database max conns: %d", cfg.Database.MaxConns)
	}
	if cfg.Database.ConnectTimeout != 2*time.Second {
		t.Fatalf("unexpected database connect timeout: %s", cfg.Database.ConnectTimeout)
	}
	if cfg.RateLimit.Enabled {
		t.Fatal("expected rate limit to be disabled")
	}
	if cfg.RateLimit.Requests != 10 {
		t.Fatalf("unexpected rate limit requests: %d", cfg.RateLimit.Requests)
	}
	if cfg.RateLimit.Window != 5*time.Second {
		t.Fatalf("unexpected rate limit window: %s", cfg.RateLimit.Window)
	}
	if !cfg.Auth.Enabled {
		t.Fatal("expected auth to be enabled")
	}
	if cfg.Auth.Secret != "test-secret-from-config-32-bytes" {
		t.Fatalf("unexpected auth secret: %s", cfg.Auth.Secret)
	}
	if cfg.Auth.Issuer != "orders-api" {
		t.Fatalf("unexpected auth issuer: %s", cfg.Auth.Issuer)
	}
	if cfg.Auth.Audience != "orders-clients" {
		t.Fatalf("unexpected auth audience: %s", cfg.Auth.Audience)
	}
	if cfg.Auth.ClockSkew != 15*time.Second {
		t.Fatalf("unexpected auth clock skew: %s", cfg.Auth.ClockSkew)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := writeConfig(t, `{"unknown": true}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAuthRequiresSecretWhenEnabled(t *testing.T) {
	cfg := Default()
	cfg.Auth.Enabled = true
	cfg.Auth.Secret = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected auth secret error")
	}
	if !strings.Contains(err.Error(), "auth.secret") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
