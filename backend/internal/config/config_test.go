package config

import (
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

const validDatabaseURL = "postgres://app:s3cr3t@localhost:5432/app?sslmode=disable" //nolint:gosec // test fixture, not a real credential

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		wantErr []string // trechos esperados na mensagem (vazio = sucesso)
		check   func(t *testing.T, c Config)
	}{
		{
			name: "valores padrão com apenas DATABASE_URL",
			env:  map[string]string{EnvDatabaseURL: validDatabaseURL},
			check: func(t *testing.T, c Config) {
				t.Helper()
				want := Config{
					Addr:                   ":8443",
					TLSCertFile:            "certs/server.crt",
					TLSKeyFile:             "certs/server.key",
					DatabaseURL:            validDatabaseURL,
					DBMaxOpenConns:         10,
					DBMaxIdleConns:         5,
					SessionIdleTimeout:     15 * time.Minute,
					SessionAbsoluteTimeout: 8 * time.Hour,
					ShutdownTimeout:        15 * time.Second,
					LogLevel:               slog.LevelInfo,
				}
				if c != want {
					t.Errorf("Load() = %+v, want %+v", c, want)
				}
			},
		},
		{
			name: "todos os valores sobrescritos",
			env: map[string]string{ //nolint:gosec // test fixture, not a real credential
				EnvAddr:                   "127.0.0.1:9443",
				EnvTLSCertFile:            "/certs/a.crt",
				EnvTLSKeyFile:             "/certs/a.key",
				EnvDatabaseURL:            "postgresql://u:p@db/x",
				EnvDBMaxOpenConns:         "20",
				EnvDBMaxIdleConns:         "20",
				EnvSessionIdleTimeout:     "30m",
				EnvSessionAbsoluteTimeout: "12h",
				EnvShutdownTimeout:        "5s",
				EnvLogLevel:               "debug",
			},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.Addr != "127.0.0.1:9443" || c.TLSCertFile != "/certs/a.crt" || c.TLSKeyFile != "/certs/a.key" ||
					c.DBMaxOpenConns != 20 || c.DBMaxIdleConns != 20 ||
					c.SessionIdleTimeout != 30*time.Minute || c.SessionAbsoluteTimeout != 12*time.Hour ||
					c.ShutdownTimeout != 5*time.Second ||
					c.LogLevel != slog.LevelDebug {
					t.Errorf("Load() = %+v", c)
				}
			},
		},
		{
			name:    "DATABASE_URL ausente",
			env:     map[string]string{},
			wantErr: []string{EnvDatabaseURL, "obrigatória"},
		},
		{
			name:    "DATABASE_URL com esquema errado",
			env:     map[string]string{EnvDatabaseURL: "mysql://u:p@localhost/db"}, //nolint:gosec // test fixture, not a real credential
			wantErr: []string{EnvDatabaseURL, "esquema"},
		},
		{
			name:    "DATABASE_URL sem host",
			env:     map[string]string{EnvDatabaseURL: "postgres:///db"},
			wantErr: []string{EnvDatabaseURL, "host"},
		},
		{
			name:    "APP_ADDR sem porta",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvAddr: "localhost"},
			wantErr: []string{EnvAddr},
		},
		{
			name:    "timeout de sessão abaixo do mínimo",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvSessionIdleTimeout: "30s"},
			wantErr: []string{EnvSessionIdleTimeout},
		},
		{
			name:    "timeout de sessão acima do máximo",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvSessionIdleTimeout: "48h"},
			wantErr: []string{EnvSessionIdleTimeout},
		},
		{
			name:    "absolute timeout below the minimum",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvSessionAbsoluteTimeout: "30s"},
			wantErr: []string{EnvSessionAbsoluteTimeout},
		},
		{
			name:    "absolute timeout above the maximum",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvSessionAbsoluteTimeout: "169h"},
			wantErr: []string{EnvSessionAbsoluteTimeout},
		},
		{
			name:    "absolute timeout below the idle timeout",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvSessionIdleTimeout: "2h", EnvSessionAbsoluteTimeout: "1h"},
			wantErr: []string{EnvSessionAbsoluteTimeout, EnvSessionIdleTimeout},
		},
		{
			name: "absolute timeout equal to the idle timeout",
			env:  map[string]string{EnvDatabaseURL: validDatabaseURL, EnvSessionIdleTimeout: "1h", EnvSessionAbsoluteTimeout: "1h"},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.SessionAbsoluteTimeout != time.Hour {
					t.Errorf("SessionAbsoluteTimeout = %v, want 1h", c.SessionAbsoluteTimeout)
				}
			},
		},
		{
			name:    "duração malformada",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvShutdownTimeout: "dez segundos"},
			wantErr: []string{EnvShutdownTimeout},
		},
		{
			name:    "pool não numérico",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvDBMaxOpenConns: "muitas"},
			wantErr: []string{EnvDBMaxOpenConns},
		},
		{
			name:    "pool zero",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvDBMaxOpenConns: "0"},
			wantErr: []string{EnvDBMaxOpenConns},
		},
		{
			name:    "idle maior que open",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvDBMaxOpenConns: "2", EnvDBMaxIdleConns: "3"},
			wantErr: []string{EnvDBMaxIdleConns},
		},
		{
			name:    "nível de log inválido",
			env:     map[string]string{EnvDatabaseURL: validDatabaseURL, EnvLogLevel: "verbose"},
			wantErr: []string{EnvLogLevel},
		},
		{
			name: "vários erros reportados juntos",
			env: map[string]string{
				EnvSessionIdleTimeout: "1s",
				EnvLogLevel:           "x",
			},
			wantErr: []string{EnvDatabaseURL, EnvSessionIdleTimeout, EnvLogLevel},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(envFrom(tt.env))

			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("Load() erro inesperado: %v", err)
				}
				tt.check(t, cfg)
				return
			}

			if err == nil {
				t.Fatalf("Load() = %+v, esperava erro", cfg)
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("erro não encapsula ErrInvalid: %v", err)
			}
			for _, frag := range tt.wantErr {
				if !strings.Contains(err.Error(), frag) {
					t.Errorf("erro %q não contém %q", err, frag)
				}
			}
			if cfg != (Config{}) {
				t.Errorf("em caso de erro Load deve retornar Config zero, obteve %+v", cfg)
			}
		})
	}
}

func TestLoadNeverLeaksDatabasePassword(t *testing.T) {
	t.Parallel()

	const marker = "valor-que-nao-pode-vazar"
	tests := []struct {
		name string
		url  string
	}{
		{"esquema inválido", "mysql://u:" + marker + "@h/db"},
		{"url malformada", "postgres://u:" + marker + "@[::1/db"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(envFrom(map[string]string{EnvDatabaseURL: tt.url}))
			if err == nil {
				t.Fatal("esperava erro")
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("mensagem de erro vaza a senha: %v", err)
			}
		})
	}
}

func TestConfigLogValueRedactsSecrets(t *testing.T) {
	t.Parallel()

	cfg, err := Load(envFrom(map[string]string{EnvDatabaseURL: validDatabaseURL}))
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	slog.New(slog.NewTextHandler(&sb, nil)).Info("cfg", slog.Any("config", cfg))
	out := sb.String()
	if strings.Contains(out, "s3cr3t") || strings.Contains(out, validDatabaseURL) {
		t.Errorf("log expõe DATABASE_URL: %s", out)
	}
	if !strings.Contains(out, "REDACTED") {
		t.Errorf("log deveria indicar valor ocultado: %s", out)
	}
}

func TestLoadSeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		want    Seed
		wantErr []string
	}{
		{
			name: "login e senha informados",
			env:  map[string]string{EnvSeedAdminLogin: "admin", EnvSeedAdminPassword: "x"},
			want: Seed{AdminLogin: "admin", AdminPassword: "x"},
		},
		{
			name:    "login ausente",
			env:     map[string]string{EnvSeedAdminPassword: "x"},
			wantErr: []string{EnvSeedAdminLogin},
		},
		{
			name:    "ambos ausentes",
			env:     map[string]string{},
			wantErr: []string{EnvSeedAdminLogin, EnvSeedAdminPassword},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := LoadSeed(envFrom(tt.env))
			if len(tt.wantErr) == 0 {
				if err != nil || got != tt.want {
					t.Fatalf("LoadSeed() = %+v, %v; want %+v", got, err, tt.want)
				}
				return
			}
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("LoadSeed() erro = %v, want ErrInvalid", err)
			}
			for _, frag := range tt.wantErr {
				if !strings.Contains(err.Error(), frag) {
					t.Errorf("erro %q não contém %q", err, frag)
				}
			}
		})
	}
}
