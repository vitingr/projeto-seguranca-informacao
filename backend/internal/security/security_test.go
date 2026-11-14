package security

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend_si/internal/domain"
)

// fakeStore é um SessionStore em memória que não encontra nada.
type fakeStore struct{}

func (fakeStore) Create(context.Context, domain.Session) error { return nil }

func (fakeStore) Get(context.Context, []byte) (domain.Session, error) {
	return domain.Session{}, domain.ErrNotFound
}

func (fakeStore) Touch(context.Context, []byte, time.Time, time.Time) error { return nil }

func (fakeStore) Delete(context.Context, []byte) error { return nil }

func (fakeStore) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }

// renderStatus é um ErrorRenderer de teste que escreve apenas o status.
func renderStatus(w http.ResponseWriter, _ *http.Request, status int) {
	w.WriteHeader(status)
	_, _ = io.WriteString(w, http.StatusText(status))
}

func newTestSecurity(t *testing.T, logOut io.Writer) *Security {
	t.Helper()
	if logOut == nil {
		logOut = io.Discard
	}
	s, err := New(Options{
		Logger:             slog.New(slog.NewJSONHandler(logOut, nil)),
		Sessions:           fakeStore{},
		RenderError:        renderStatus,
		SessionIdleTimeout: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New() erro inesperado: %v", err)
	}
	return s
}

// okHandler marca se foi executado.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestNewValidatesOptions(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	valid := Options{Logger: logger, Sessions: fakeStore{}, RenderError: renderStatus, SessionIdleTimeout: time.Minute}

	tests := []struct {
		name    string
		mutate  func(o *Options)
		wantErr string
	}{
		{"opções válidas", func(*Options) {}, ""},
		{"sem logger", func(o *Options) { o.Logger = nil }, "Logger"},
		{"sem store", func(o *Options) { o.Sessions = nil }, "Sessions"},
		{"sem renderer", func(o *Options) { o.RenderError = nil }, "RenderError"},
		{"timeout zero", func(o *Options) { o.SessionIdleTimeout = 0 }, "SessionIdleTimeout"},
		{"timeout negativo", func(o *Options) { o.SessionIdleTimeout = -time.Second }, "SessionIdleTimeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := valid
			tt.mutate(&opts)
			s, err := New(opts)
			if tt.wantErr == "" {
				if err != nil || s == nil {
					t.Fatalf("New() = %v, %v", s, err)
				}
				if s.argon2 != DefaultArgon2Params() {
					t.Errorf("Argon2 zero deveria usar o padrão, obteve %+v", s.argon2)
				}
				return
			}
			if !errors.Is(err, ErrInvalidOptions) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("New() erro = %v, want contendo %q", err, tt.wantErr)
			}
			if s != nil {
				t.Error("New() deve retornar nil em caso de erro")
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	var called bool
	rr := httptest.NewRecorder()
	s.SecurityHeaders(okHandler(&called)).ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if !called {
		t.Fatal("handler não executado")
	}
	tests := []struct {
		header string
		want   string // trecho obrigatório
	}{
		{"Content-Security-Policy", "default-src 'none'"},
		{"Content-Security-Policy", "frame-ancestors 'none'"},
		{"Strict-Transport-Security", "max-age="},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "no-referrer"},
		{"Permissions-Policy", "camera=()"},
		{"Cross-Origin-Opener-Policy", "same-origin"},
		{"Cross-Origin-Resource-Policy", "same-origin"},
		{"Cache-Control", "no-store"},
	}
	for _, tt := range tests {
		if got := rr.Header().Get(tt.header); !strings.Contains(got, tt.want) {
			t.Errorf("%s = %q, want contendo %q", tt.header, got, tt.want)
		}
	}
}

func TestRecover(t *testing.T) {
	t.Parallel()

	const marker = "detalhe-interno-do-panic"
	tests := []struct {
		name  string
		panic any
	}{
		{"panic com string", marker},
		{"panic com erro", errors.New(marker)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var logs strings.Builder
			s := newTestSecurity(t, &logs)
			h := s.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(tt.panic) }))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", http.NoBody))

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", rr.Code)
			}
			if strings.Contains(rr.Body.String(), marker) {
				t.Error("resposta expõe o detalhe do panic ao usuário")
			}
			if !strings.Contains(logs.String(), marker) || !strings.Contains(logs.String(), "stack") {
				t.Error("o panic e a stack trace devem ir para o log")
			}
		})
	}
}

func TestRecoverRepanicsErrAbortHandler(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	h := s.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler { //nolint:errorlint // comparação de identidade do valor do panic
			t.Errorf("recover() = %v, want http.ErrAbortHandler", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))
}

func TestRespondErrorHidesDetails(t *testing.T) {
	t.Parallel()

	var logs strings.Builder
	s := newTestSecurity(t, &logs)
	rr := httptest.NewRecorder()
	s.RespondError(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody), errors.New("pq: tabela users corrompida"))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "users") {
		t.Error("resposta expõe detalhes do erro")
	}
	if !strings.Contains(logs.String(), "corrompida") {
		t.Error("o erro deve ser registrado no log")
	}
}
