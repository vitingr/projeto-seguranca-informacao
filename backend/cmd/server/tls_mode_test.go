package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"backend_si/internal/config"
)

func missingCerts(t *testing.T, mode string) config.Config {
	t.Helper()
	dir := t.TempDir()
	return config.Config{
		TLSCertFile: filepath.Join(dir, "server.crt"),
		TLSKeyFile:  filepath.Join(dir, "server.key"),
		TLSMode:     mode,
	}
}

func TestProxyModeServesPlainHTTPWithoutCertificates(t *testing.T) {
	t.Parallel()

	cfg := missingCerts(t, config.TLSModeProxy)
	var logs bytes.Buffer
	tlsConfig, err := serverTLSConfig(cfg, slog.New(slog.NewJSONHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("serverTLSConfig: %v", err)
	}
	if tlsConfig != nil {
		t.Fatal("serverTLSConfig returned a TLS config in proxy mode")
	}
	if out := logs.String(); !strings.Contains(out, `"level":"WARN"`) || !strings.Contains(out, "APP_TLS_MODE=proxy") {
		t.Errorf("want a startup warning about APP_TLS_MODE=proxy, got %q", out)
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := &http.Server{Handler: mux, TLSConfig: tlsConfig, ReadHeaderTimeout: time.Second}
	go func() { _ = serveOn(srv, ln, slog.New(slog.NewTextHandler(io.Discard, nil))) }()
	t.Cleanup(func() { _ = srv.Close() })

	cfg.Addr = ln.Addr().String()
	if err := healthcheck(t.Context(), cfg); err != nil {
		t.Errorf("healthcheck in proxy mode: %v", err)
	}
}

func TestWithoutProxyModeCertificatesAreRequired(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"", "tls", "PROXY"} {
		t.Run("mode="+mode, func(t *testing.T) {
			t.Parallel()

			_, err := serverTLSConfig(missingCerts(t, mode), slog.New(slog.NewTextHandler(io.Discard, nil)))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("serverTLSConfig error = %v, want a missing certificate error", err)
			}
		})
	}
}
