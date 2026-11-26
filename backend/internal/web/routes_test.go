package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
	"backend_si/internal/service"
)

type fakeAuth struct{}

func (fakeAuth) Authenticate(context.Context, string, string) (domain.User, error) {
	return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
}

type fakeUsers struct{}

func (fakeUsers) Register(context.Context, service.RegisterInput) (domain.User, error) {
	return domain.User{ID: 2, Role: domain.RoleComum}, nil
}

type nopStore struct{}

func (nopStore) Create(context.Context, domain.Session) error { return nil }

func (nopStore) Get(context.Context, []byte) (domain.Session, error) {
	return domain.Session{}, domain.ErrNotFound
}

func (nopStore) Touch(context.Context, []byte, time.Time, time.Time) error { return nil }

func (nopStore) Delete(context.Context, []byte) error { return nil }

func (nopStore) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }

// failingCreateStore is a nopStore whose Create fails.
type failingCreateStore struct{ nopStore }

var errCreateSession = errors.New("create session failed")

func (failingCreateStore) Create(context.Context, domain.Session) error { return errCreateSession }

func newTestApp(t *testing.T) (*handlers, http.Handler) {
	t.Helper()
	return newTestAppWithLog(t, io.Discard)
}

// newTestAppWithLog is newTestApp with the log written to logOut.
func newTestAppWithLog(t *testing.T, logOut io.Writer) (*handlers, http.Handler) {
	t.Helper()
	return newTestAppWith(t, logOut, nopStore{})
}

// newTestAppWith is newTestApp with the log written to logOut and the
// given session store.
func newTestAppWith(t *testing.T, logOut io.Writer, store security.SessionStore) (*handlers, http.Handler) {
	t.Helper()
	logger := slog.New(NewLogHandler(slog.NewJSONHandler(logOut, nil)))
	sec, err := security.New(security.Options{
		Logger:                 logger,
		Sessions:               store,
		RenderError:            WriteError,
		SessionIdleTimeout:     time.Minute,
		SessionAbsoluteTimeout: time.Hour,
		RequestTimeout:         time.Minute,
	})
	if err != nil {
		t.Fatalf("security.New: %v", err)
	}
	deps := Deps{
		Logger: logger, Security: sec, Auth: fakeAuth{}, Users: fakeUsers{},
		Tickets: service.NewTickets(newMemTicketRepo(), sec), TicketLimiter: newTicketLimiter(t, sec),
		MessageLimiter: newMessageLimiter(t, sec),
	}
	h, err := NewHandler(deps)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return &handlers{
		logger: logger, sec: sec, auth: deps.Auth, users: deps.Users,
		tickets: deps.Tickets, ticketLimiter: deps.TicketLimiter,
		messageLimiter: deps.MessageLimiter,
	}, h
}

func TestEveryRouteDeclaresPolicy(t *testing.T) {
	t.Parallel()

	h, _ := newTestApp(t)
	seen := map[string]bool{}
	for _, rt := range h.routes() {
		p := rt.pattern()
		if rt.policy.IsZero() {
			t.Errorf("%s: rota sem política (valor zero nega tudo)", p)
		}
		if rt.handler == nil {
			t.Errorf("%s: handler nil", p)
		}
		if rt.path == "" || !strings.HasPrefix(rt.path, "/") {
			t.Errorf("%s: caminho inválido", p)
		}
		if seen[p] {
			t.Errorf("%s: padrão duplicado", p)
		}
		seen[p] = true
	}
}

func TestRoutePolicies(t *testing.T) {
	t.Parallel()

	h, _ := newTestApp(t)
	byPattern := map[string]security.Policy{}
	for _, rt := range h.routes() {
		byPattern[rt.pattern()] = rt.policy
	}

	admin := []domain.Role{domain.RoleAdmin}
	comum := []domain.Role{domain.RoleComum}
	ambos := []domain.Role{domain.RoleAdmin, domain.RoleComum}

	tests := []struct {
		pattern   string
		public    bool
		wantRoles []domain.Role
	}{
		{"GET /healthz", true, nil},
		{"GET /api/csrf", true, nil},
		{"POST /api/login", true, nil},
		{"/", true, nil},
		{"POST /api/logout", false, ambos},
		{"GET /api/me", false, ambos},
		{"POST /api/cadastro", false, admin},
		{"GET /api/administradores", false, admin},
		{"GET /api/usuarios", false, comum},
		{"POST /api/tickets", false, comum},
		{"GET /api/tickets", false, comum},
		{"GET /api/tickets/{id}", false, comum},
		{"GET /api/tickets/{id}/mensagens", false, comum},
		{"POST /api/tickets/{id}/mensagens", false, comum},
		{"GET /api/administradores/tickets", false, admin},
		{"GET /api/administradores/tickets/autores", false, admin},
		{"GET /api/administradores/tickets/{id}", false, admin},
	}
	if len(tests) != len(byPattern) {
		t.Errorf("tabela de rotas tem %d entradas, o teste cobre %d: atualize o teste", len(byPattern), len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			t.Parallel()
			p, ok := byPattern[tt.pattern]
			if !ok {
				t.Fatalf("rota %s não declarada", tt.pattern)
			}
			if p.IsPublic() != tt.public {
				t.Errorf("IsPublic() = %v, want %v", p.IsPublic(), tt.public)
			}
			if !slices.Equal(p.Roles(), tt.wantRoles) {
				t.Errorf("Roles() = %v, want %v", p.Roles(), tt.wantRoles)
			}
		})
	}
}

func TestHandlerDeniesByDefault(t *testing.T) {
	t.Parallel()

	_, app := newTestApp(t)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantError  string
	}{
		{"healthz público", http.MethodGet, "/healthz", "", http.StatusOK, ""},
		{"csrf público", http.MethodGet, "/api/csrf", "", http.StatusOK, ""},
		{"anonymous me gets 401", http.MethodGet, "/api/me", "", http.StatusUnauthorized, "unauthorized"},
		{"anonymous administradores gets 401", http.MethodGet, "/api/administradores", "", http.StatusUnauthorized, "unauthorized"},
		{"anonymous usuarios gets 401", http.MethodGet, "/api/usuarios", "", http.StatusUnauthorized, "unauthorized"},
		{"login barrado pelo CSRF", http.MethodPost, "/api/login", `{"login":"a","senha":"b"}`, http.StatusForbidden, "forbidden"},
		{"cadastro barrado", http.MethodPost, "/api/cadastro", `{"login":"a","senha":"b","papel":"admin"}`, http.StatusForbidden, "forbidden"},
		{"logout barrado", http.MethodPost, "/api/logout", "", http.StatusForbidden, "forbidden"},
		{"tickets blocked by CSRF", http.MethodPost, "/api/tickets", `{"tipo":"outro","dados":{}}`, http.StatusForbidden, "forbidden"},
		{"anonymous ticket list gets 401", http.MethodGet, "/api/tickets", "", http.StatusUnauthorized, "unauthorized"},
		{"anonymous ticket detail gets 401", http.MethodGet, "/api/tickets/1", "", http.StatusUnauthorized, "unauthorized"},
		{"anonymous thread gets 401", http.MethodGet, "/api/tickets/1/mensagens", "", http.StatusUnauthorized, "unauthorized"},
		{"reply blocked by CSRF", http.MethodPost, "/api/tickets/1/mensagens", `{"texto":"oi"}`, http.StatusForbidden, "forbidden"},
		{"anonymous queue gets 401", http.MethodGet, "/api/administradores/tickets", "", http.StatusUnauthorized, "unauthorized"},
		{"anonymous queue authors get 401", http.MethodGet, "/api/administradores/tickets/autores", "", http.StatusUnauthorized, "unauthorized"},
		{"anonymous admin ticket gets 401", http.MethodGet, "/api/administradores/tickets/1", "", http.StatusUnauthorized, "unauthorized"},
		{"método não previsto", http.MethodDelete, "/api/usuarios", "", http.StatusForbidden, "forbidden"},
		{"rota inexistente", http.MethodGet, "/api/nao-existe", "", http.StatusNotFound, "not_found"},
		{"páginas antigas não existem mais", http.MethodGet, "/login", "", http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			app.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", ct)
			}
			if rr.Header().Get("Content-Security-Policy") == "" {
				t.Error("cabeçalhos de segurança ausentes")
			}
			requestID := rr.Header().Get("X-Request-Id")
			if requestID == "" {
				t.Error("X-Request-Id ausente")
			}

			if tt.wantError == "" {
				return
			}
			var body errorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("corpo de erro não é JSON: %v", err)
			}
			if body.Error != tt.wantError || body.Message == "" {
				t.Errorf("erro = %+v, want código %q com mensagem", body, tt.wantError)
			}
			if body.RequestID != requestID {
				t.Errorf("request_id = %q, want %q (cabeçalho)", body.RequestID, requestID)
			}
		})
	}
}

// The frontend pages moved to the paths the assignment requires (spec
// 001-ticket-builder, AC-0.7). The backend never served page paths, so the
// old ones fall through to the catch-all and get the generic 404.
func TestOldPagePathsAnswer404(t *testing.T) {
	t.Parallel()

	_, app := newTestApp(t)
	for _, path := range []string{"/minha-conta", "/admin", "/cadastrar-usuarios"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			app.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rr.Code)
			}
			var body map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("error body is not JSON: %v", err)
			}
			want := map[string]string{
				"error":      "not_found",
				"message":    "Recurso não encontrado.",
				"request_id": rr.Header().Get("X-Request-Id"),
			}
			if body["request_id"] == "" || len(body) != len(want) {
				t.Fatalf("body = %v, want exactly the keys error, message, request_id", body)
			}
			for key, value := range want {
				if body[key] != value {
					t.Errorf("%s = %q, want %q", key, body[key], value)
				}
			}
		})
	}
}

// TestTooManyRequestsBody pins the 429 body of spec AC-6.7 and S-12: the
// generic three keys, the too_many_requests code and the Portuguese text.
func TestTooManyRequestsBody(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	WriteError(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/tickets", http.NoBody), http.StatusTooManyRequests)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	want := map[string]string{
		"error":   "too_many_requests",
		"message": "Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.",
	}
	for key, value := range want {
		if body[key] != value {
			t.Errorf("%s = %q, want %q", key, body[key], value)
		}
	}
}

func TestCSRFEndpointTellsHeaderName(t *testing.T) {
	t.Parallel()

	_, app := newTestApp(t)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody))

	var body csrfResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Header != security.CSRFHeaderName {
		t.Errorf("header = %q, want %q", body.Header, security.CSRFHeaderName)
	}
	if len(body.CSRFToken) != 26 {
		t.Errorf("csrf_token = %q, want a 26-character token", body.CSRFToken)
	}
	var cookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == security.SessionCookieName {
			cookie = c
		}
	}
	if cookie == nil || cookie.Value == "" || cookie.Value == body.CSRFToken ||
		!cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Errorf("session cookie = %+v, want a new HttpOnly, Secure, SameSite=Lax cookie (D-8)", cookie)
	}
}

// TestOnlyCSRFEndpointCreatesSessions is D-8: without a session cookie, no
// route but GET /api/csrf sets one.
func TestOnlyCSRFEndpointCreatesSessions(t *testing.T) {
	t.Parallel()

	_, app := newTestApp(t)
	tests := []struct {
		method, path string
		wantCookie   bool
	}{
		{http.MethodGet, "/api/csrf", true},
		{http.MethodGet, "/healthz", false},
		{http.MethodGet, "/api/me", false},
		{http.MethodPost, "/api/login", false},
		{http.MethodGet, "/api/nao-existe", false},
	}
	for _, tt := range tests {
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody))
		got := strings.Contains(strings.Join(rr.Header().Values("Set-Cookie"), ";"), security.SessionCookieName+"=")
		if got != tt.wantCookie {
			t.Errorf("%s %s: Set-Cookie for the session = %v, want %v", tt.method, tt.path, got, tt.wantCookie)
		}
	}
}

func TestCSRFEndpointStoreFailureIs500(t *testing.T) {
	t.Parallel()

	_, app := newTestAppWith(t, io.Discard, failingCreateStore{})
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
	if len(rr.Header().Values("Set-Cookie")) != 0 {
		t.Error("a cookie was set although the session was not stored")
	}
}

func TestWriteErrorNeverLeaksDetails(t *testing.T) {
	t.Parallel()

	statuses := []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity,
		http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusTeapot,
	}
	generic := defaultMessage(http.StatusInternalServerError)
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			WriteError(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody), status)

			if rr.Code != status {
				t.Errorf("status = %d, want %d", rr.Code, status)
			}
			var body map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("corpo não é JSON: %v", err)
			}
			for k := range body {
				if k != "error" && k != "message" && k != "request_id" {
					t.Errorf("campo inesperado na resposta de erro: %q", k)
				}
			}
			// Every mapped status has its own Portuguese message (003 D-5);
			// only unmapped ones fall back to the generic 500 text.
			mapped := status != http.StatusInternalServerError && status != http.StatusTeapot
			if mapped && body["message"] == generic {
				t.Errorf("status %d uses the generic 500 message", status)
			}
		})
	}
}

func TestNewHandlerRequiresDeps(t *testing.T) {
	t.Parallel()

	if _, err := NewHandler(Deps{}); err == nil {
		t.Fatal("NewHandler deveria falhar sem dependências")
	}
}

// TestLoginNeverLogsBodyOrPassword checks that neither the request body nor
// the password reaches the log, for rejected bodies and for a valid body
// that fails later and goes through RespondError.
func TestLoginNeverLogsBodyOrPassword(t *testing.T) {
	t.Parallel()

	const sentinel = "Sentinel-Pa55word-7f3a"
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int // from the login handler itself
	}{
		// The fakeAuth accepts it and the store fails to create the
		// session, so the error is logged by RespondError.
		{"valid body", "application/json", `{"login":"admin","senha":"` + sentinel + `"}`, http.StatusInternalServerError},
		{"wrong content type", "text/plain", `{"login":"admin","senha":"` + sentinel + `"}`, http.StatusBadRequest},
		{"unknown field", "application/json", `{"login":"admin","senha":"x","` + sentinel + `":1}`, http.StatusBadRequest},
		{"control byte in password", "application/json", `{"login":"admin","senha":"` + sentinel + "\x01" + `"}`, http.StatusBadRequest},
		{"wrong type", "application/json", `{"login":"admin","senha":` + sentinel + `}`, http.StatusBadRequest},
		{"trailing data", "application/json", `{"login":"admin","senha":"x"}` + sentinel, http.StatusBadRequest},
		{"too large", "application/json", `{"login":"admin","senha":"` + strings.Repeat(sentinel, 4000) + `"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			h, app := newTestAppWith(t, &logs, failingCreateStore{})
			newReq := func() *http.Request {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/login", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", tt.contentType)
				return req
			}

			// The handler directly, without the CSRF check.
			rr := httptest.NewRecorder()
			h.login(rr, newReq())
			if rr.Code != tt.wantStatus {
				t.Errorf("login status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusInternalServerError && logs.Len() == 0 {
				t.Fatal("RespondError did not log; the test would not cover it")
			}

			// The full chain, including the access log.
			app.ServeHTTP(httptest.NewRecorder(), newReq())

			if strings.Contains(logs.String(), sentinel) {
				t.Errorf("log contains the body or the password:\n%s", logs.String())
			}
			if strings.Contains(rr.Body.String(), sentinel) {
				t.Errorf("response contains the body or the password: %s", rr.Body.String())
			}
		})
	}
}
