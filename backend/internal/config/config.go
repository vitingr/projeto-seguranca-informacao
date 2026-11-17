package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"time"
)

const (
	EnvAddr               = "APP_ADDR"
	EnvTLSCertFile        = "APP_TLS_CERT_FILE"
	EnvTLSKeyFile         = "APP_TLS_KEY_FILE"
	EnvDatabaseURL        = "DATABASE_URL"
	EnvDBMaxOpenConns     = "DB_MAX_OPEN_CONNS"
	EnvDBMaxIdleConns     = "DB_MAX_IDLE_CONNS"
	EnvSessionIdleTimeout = "SESSION_IDLE_TIMEOUT"
	// EnvSessionAbsoluteTimeout limits a session's total lifetime (D-18).
	EnvSessionAbsoluteTimeout = "SESSION_ABSOLUTE_TIMEOUT"
	EnvShutdownTimeout        = "SHUTDOWN_TIMEOUT"
	EnvLogLevel               = "LOG_LEVEL"
	EnvSeedAdminLogin         = "SEED_ADMIN_LOGIN"
	EnvSeedAdminPassword      = "SEED_ADMIN_PASSWORD" //nolint:gosec
)

const (
	minSessionIdleTimeout = time.Minute
	maxSessionIdleTimeout = 24 * time.Hour
	minSessionAbsTimeout  = time.Minute
	maxSessionAbsTimeout  = 168 * time.Hour
	maxShutdownTimeout    = 5 * time.Minute
	maxDBConns            = 1000
)

var ErrInvalid = errors.New("invalid configuration")

type Config struct {
	Addr               string
	TLSCertFile        string
	TLSKeyFile         string
	DatabaseURL        string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	SessionIdleTimeout time.Duration
	// SessionAbsoluteTimeout is the maximum lifetime of a session, counted
	// from its creation (login), whatever its activity. It is at least
	// SessionIdleTimeout (D-18).
	SessionAbsoluteTimeout time.Duration
	ShutdownTimeout        time.Duration
	LogLevel               slog.Level
}

func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("addr", c.Addr),
		slog.String("tls_cert_file", c.TLSCertFile),
		slog.String("tls_key_file", c.TLSKeyFile),
		slog.String("database_url", "[REDACTED]"),
		slog.Int("db_max_open_conns", c.DBMaxOpenConns),
		slog.Int("db_max_idle_conns", c.DBMaxIdleConns),
		slog.Duration("session_idle_timeout", c.SessionIdleTimeout),
		slog.Duration("session_absolute_timeout", c.SessionAbsoluteTimeout),
		slog.Duration("shutdown_timeout", c.ShutdownTimeout),
		slog.String("log_level", c.LogLevel.String()),
	)
}

func Load(getenv func(string) string) (Config, error) {
	p := parser{getenv: getenv}

	cfg := Config{
		Addr:               p.str(EnvAddr, ":8443"),
		TLSCertFile:        p.str(EnvTLSCertFile, "certs/server.crt"),
		TLSKeyFile:         p.str(EnvTLSKeyFile, "certs/server.key"),
		DatabaseURL:        p.required(EnvDatabaseURL),
		DBMaxOpenConns:     p.intInRange(EnvDBMaxOpenConns, 10, 1, maxDBConns),
		DBMaxIdleConns:     p.intInRange(EnvDBMaxIdleConns, 5, 0, maxDBConns),
		SessionIdleTimeout: p.durationInRange(EnvSessionIdleTimeout, 15*time.Minute, minSessionIdleTimeout, maxSessionIdleTimeout),
		SessionAbsoluteTimeout: p.durationInRange(EnvSessionAbsoluteTimeout, 8*time.Hour,
			minSessionAbsTimeout, maxSessionAbsTimeout),
		ShutdownTimeout: p.durationInRange(EnvShutdownTimeout, 15*time.Second, time.Second, maxShutdownTimeout),
		LogLevel:        p.logLevel(EnvLogLevel, slog.LevelInfo),
	}

	if _, _, err := net.SplitHostPort(cfg.Addr); err != nil {
		p.fail(EnvAddr, "deve estar no formato host:porta")
	}
	if cfg.DatabaseURL != "" {
		validateDatabaseURL(&p, cfg.DatabaseURL)
	}
	if cfg.SessionAbsoluteTimeout != 0 && cfg.SessionIdleTimeout != 0 &&
		cfg.SessionAbsoluteTimeout < cfg.SessionIdleTimeout {
		p.fail(EnvSessionAbsoluteTimeout, "must be at least "+EnvSessionIdleTimeout)
	}
	if cfg.DBMaxIdleConns > cfg.DBMaxOpenConns {
		p.fail(EnvDBMaxIdleConns, "não pode ser maior que "+EnvDBMaxOpenConns)
	}

	if err := errors.Join(p.errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type Seed struct {
	AdminLogin    string
	AdminPassword string
}

func LoadSeed(getenv func(string) string) (Seed, error) {
	p := parser{getenv: getenv}
	s := Seed{
		AdminLogin:    p.required(EnvSeedAdminLogin),
		AdminPassword: p.required(EnvSeedAdminPassword),
	}
	if err := errors.Join(p.errs...); err != nil {
		return Seed{}, err
	}
	return s, nil
}

func validateDatabaseURL(p *parser, raw string) {
	u, err := url.Parse(raw)
	if err != nil {
		p.fail(EnvDatabaseURL, "URL inválida")
		return
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		p.fail(EnvDatabaseURL, "o esquema deve ser postgres:// ou postgresql://")
	}
	if u.Host == "" {
		p.fail(EnvDatabaseURL, "o host é obrigatório")
	}
}

type parser struct {
	getenv func(string) string
	errs   []error
}

func (p *parser) fail(key, msg string) {
	p.errs = append(p.errs, fmt.Errorf("%w: %s: %s", ErrInvalid, key, msg))
}

func (p *parser) str(key, def string) string {
	if v := p.getenv(key); v != "" {
		return v
	}
	return def
}

func (p *parser) required(key string) string {
	v := p.getenv(key)
	if v == "" {
		p.fail(key, "obrigatória")
	}
	return v
}

func (p *parser) intInRange(key string, def, lo, hi int) int {
	raw := p.getenv(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		p.fail(key, fmt.Sprintf("deve ser um inteiro entre %d e %d", lo, hi))
		return def
	}
	return n
}

func (p *parser) durationInRange(key string, def, lo, hi time.Duration) time.Duration {
	raw := p.getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < lo || d > hi {
		p.fail(key, fmt.Sprintf("deve ser uma duração entre %s e %s (ex.: 15m)", lo, hi))
		return def
	}
	return d
}

func (p *parser) logLevel(key string, def slog.Level) slog.Level {
	raw := p.getenv(key)
	if raw == "" {
		return def
	}
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(raw)); err != nil {
		p.fail(key, "deve ser debug, info, warn ou error")
		return def
	}
	return lvl
}
