package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	Database  DatabaseConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	Auth      AuthConfig
	Log       LogConfig
}

type AppConfig struct {
	Name string
	Env  string
}

type HTTPConfig struct {
	Host              string
	Port              string
	MaxBodyBytes      int64
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

type DatabaseConfig struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
}

type LogConfig struct {
	Level string
}

type RateLimitConfig struct {
	Enabled  bool
	Requests int
	Window   time.Duration
}

type AuthConfig struct {
	Enabled   bool
	Secret    string
	Issuer    string
	Audience  string
	ClockSkew time.Duration
}

func Default() Config {
	return Config{
		App: AppConfig{
			Name: "gin-api",
			Env:  "local",
		},
		HTTP: HTTPConfig{
			Host:              "0.0.0.0",
			Port:              "8080",
			MaxBodyBytes:      1 << 20,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   10 * time.Second,
		},
		Database: DatabaseConfig{
			DSN:               "postgres://app:app@localhost:5432/app?sslmode=disable",
			MaxConns:          10,
			MinConns:          1,
			MaxConnLifetime:   time.Hour,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: time.Minute,
			ConnectTimeout:    5 * time.Second,
		},
		CORS: CORSConfig{
			AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-Id"},
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		},
		RateLimit: RateLimitConfig{
			Enabled:  true,
			Requests: 120,
			Window:   time.Minute,
		},
		Auth: AuthConfig{
			Enabled:   false,
			ClockSkew: 30 * time.Second,
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

func Load(path string) (Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Config{}, fmt.Errorf("missing config path; pass -c configs/local.json")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var file fileConfig
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg := Default()
	if err := cfg.apply(file); err != nil {
		return Config{}, fmt.Errorf("apply config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %s: %w", path, err)
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.App.Name) == "" {
		return fmt.Errorf("app.name is required")
	}
	if strings.TrimSpace(c.HTTP.Host) == "" {
		return fmt.Errorf("http.host is required")
	}
	if strings.TrimSpace(c.HTTP.Port) == "" {
		return fmt.Errorf("http.port is required")
	}
	if c.HTTP.MaxBodyBytes < 0 {
		return fmt.Errorf("http.max_body_bytes must be greater than or equal to 0")
	}
	if strings.TrimSpace(c.Database.DSN) == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if c.RateLimit.Enabled {
		if c.RateLimit.Requests <= 0 {
			return fmt.Errorf("rate_limit.requests must be greater than 0 when rate limit is enabled")
		}
		if c.RateLimit.Window <= 0 {
			return fmt.Errorf("rate_limit.window must be greater than 0 when rate limit is enabled")
		}
	}
	if c.Auth.Enabled {
		if strings.TrimSpace(c.Auth.Secret) == "" {
			return fmt.Errorf("auth.secret is required when auth is enabled")
		}
		if c.Auth.ClockSkew < 0 {
			return fmt.Errorf("auth.clock_skew must be greater than or equal to 0")
		}
	}
	return nil
}

func (c AppConfig) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

func (c AppConfig) IsTest() bool {
	return strings.EqualFold(c.Env, "test")
}

func (c HTTPConfig) Addr() string {
	return c.Host + ":" + c.Port
}

func (c LogConfig) SlogLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(c.Level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c *Config) apply(file fileConfig) error {
	if file.App != nil {
		setString(&c.App.Name, file.App.Name)
		setString(&c.App.Env, file.App.Env)
	}

	if file.HTTP != nil {
		setString(&c.HTTP.Host, file.HTTP.Host)
		setString(&c.HTTP.Port, file.HTTP.Port)
		setInt64(&c.HTTP.MaxBodyBytes, file.HTTP.MaxBodyBytes)
		if err := setDuration(&c.HTTP.ReadHeaderTimeout, "http.read_header_timeout", file.HTTP.ReadHeaderTimeout); err != nil {
			return err
		}
		if err := setDuration(&c.HTTP.ReadTimeout, "http.read_timeout", file.HTTP.ReadTimeout); err != nil {
			return err
		}
		if err := setDuration(&c.HTTP.WriteTimeout, "http.write_timeout", file.HTTP.WriteTimeout); err != nil {
			return err
		}
		if err := setDuration(&c.HTTP.IdleTimeout, "http.idle_timeout", file.HTTP.IdleTimeout); err != nil {
			return err
		}
		if err := setDuration(&c.HTTP.ShutdownTimeout, "http.shutdown_timeout", file.HTTP.ShutdownTimeout); err != nil {
			return err
		}
	}

	if file.Database != nil {
		setString(&c.Database.DSN, file.Database.DSN)
		setInt32(&c.Database.MaxConns, file.Database.MaxConns)
		setInt32(&c.Database.MinConns, file.Database.MinConns)
		if err := setDuration(&c.Database.MaxConnLifetime, "database.max_conn_lifetime", file.Database.MaxConnLifetime); err != nil {
			return err
		}
		if err := setDuration(&c.Database.MaxConnIdleTime, "database.max_conn_idle_time", file.Database.MaxConnIdleTime); err != nil {
			return err
		}
		if err := setDuration(&c.Database.HealthCheckPeriod, "database.health_check_period", file.Database.HealthCheckPeriod); err != nil {
			return err
		}
		if err := setDuration(&c.Database.ConnectTimeout, "database.connect_timeout", file.Database.ConnectTimeout); err != nil {
			return err
		}
	}

	if file.CORS != nil {
		if file.CORS.AllowedOrigins != nil {
			c.CORS.AllowedOrigins = file.CORS.AllowedOrigins
		}
		if file.CORS.AllowedMethods != nil {
			c.CORS.AllowedMethods = file.CORS.AllowedMethods
		}
		if file.CORS.AllowedHeaders != nil {
			c.CORS.AllowedHeaders = file.CORS.AllowedHeaders
		}
		if file.CORS.AllowCredentials != nil {
			c.CORS.AllowCredentials = *file.CORS.AllowCredentials
		}
		if err := setDuration(&c.CORS.MaxAge, "cors.max_age", file.CORS.MaxAge); err != nil {
			return err
		}
	}

	if file.RateLimit != nil {
		if file.RateLimit.Enabled != nil {
			c.RateLimit.Enabled = *file.RateLimit.Enabled
		}
		setInt(&c.RateLimit.Requests, file.RateLimit.Requests)
		if err := setDuration(&c.RateLimit.Window, "rate_limit.window", file.RateLimit.Window); err != nil {
			return err
		}
	}

	if file.Auth != nil {
		if file.Auth.Enabled != nil {
			c.Auth.Enabled = *file.Auth.Enabled
		}
		setString(&c.Auth.Secret, file.Auth.Secret)
		setString(&c.Auth.Issuer, file.Auth.Issuer)
		setString(&c.Auth.Audience, file.Auth.Audience)
		if err := setDuration(&c.Auth.ClockSkew, "auth.clock_skew", file.Auth.ClockSkew); err != nil {
			return err
		}
	}

	if file.Log != nil {
		setString(&c.Log.Level, file.Log.Level)
	}

	return nil
}

type fileConfig struct {
	App       *fileAppConfig       `json:"app"`
	HTTP      *fileHTTPConfig      `json:"http"`
	Database  *fileDatabaseConfig  `json:"database"`
	CORS      *fileCORSConfig      `json:"cors"`
	RateLimit *fileRateLimitConfig `json:"rate_limit"`
	Auth      *fileAuthConfig      `json:"auth"`
	Log       *fileLogConfig       `json:"log"`
}

type fileAppConfig struct {
	Name string `json:"name"`
	Env  string `json:"env"`
}

type fileHTTPConfig struct {
	Host              string `json:"host"`
	Port              string `json:"port"`
	MaxBodyBytes      *int64 `json:"max_body_bytes"`
	ReadHeaderTimeout string `json:"read_header_timeout"`
	ReadTimeout       string `json:"read_timeout"`
	WriteTimeout      string `json:"write_timeout"`
	IdleTimeout       string `json:"idle_timeout"`
	ShutdownTimeout   string `json:"shutdown_timeout"`
}

type fileDatabaseConfig struct {
	DSN               string `json:"dsn"`
	MaxConns          *int32 `json:"max_conns"`
	MinConns          *int32 `json:"min_conns"`
	MaxConnLifetime   string `json:"max_conn_lifetime"`
	MaxConnIdleTime   string `json:"max_conn_idle_time"`
	HealthCheckPeriod string `json:"health_check_period"`
	ConnectTimeout    string `json:"connect_timeout"`
}

type fileCORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	AllowCredentials *bool    `json:"allow_credentials"`
	MaxAge           string   `json:"max_age"`
}

type fileLogConfig struct {
	Level string `json:"level"`
}

type fileRateLimitConfig struct {
	Enabled  *bool  `json:"enabled"`
	Requests *int   `json:"requests"`
	Window   string `json:"window"`
}

type fileAuthConfig struct {
	Enabled   *bool  `json:"enabled"`
	Secret    string `json:"secret"`
	Issuer    string `json:"issuer"`
	Audience  string `json:"audience"`
	ClockSkew string `json:"clock_skew"`
}

func setString(target *string, value string) {
	if strings.TrimSpace(value) != "" {
		*target = value
	}
}

func setInt32(target *int32, value *int32) {
	if value != nil {
		*target = *value
	}
}

func setInt(target *int, value *int) {
	if value != nil {
		*target = *value
	}
}

func setInt64(target *int64, value *int64) {
	if value != nil {
		*target = *value
	}
}

func setDuration(target *time.Duration, name string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid %s=%q: %w", name, value, err)
	}
	*target = duration
	return nil
}
