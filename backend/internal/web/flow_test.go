package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

const (
	flowLogin    = "admin"
	flowPassword = "the right long password"
)

// comumLogin is a user with the comum role and the same password.
const comumLogin = "maria"

// credAuth accepts flowLogin (admin, ID 1) and comumLogin (comum, ID 2),
// both with flowPassword.
type credAuth struct{}

func (credAuth) Authenticate(_ context.Context, login, password string) (domain.User, error) {
	if password == flowPassword {
		switch login {
		case flowLogin:
			return domain.User{ID: 1, Login: flowLogin, Role: domain.RoleAdmin}, nil
		case comumLogin:
			return domain.User{ID: 2, Login: comumLogin, Role: domain.RoleComum}, nil
		}
	}
	return domain.User{}, fmt.Errorf("fake: %w", domain.ErrInvalidCredentials)
}

// flowApp is the full NewHandler chain with working sessions.
type flowApp struct {
	app      http.Handler
	sessions *memSessions
}

func newFlowApp(t *testing.T) flowApp {
	t.Helper()
	return newFlowAppWith(t, flowConfig{})
}

// newFlowAppWithLog is newFlowApp with the log written to logOut.
func newFlowAppWithLog(t *testing.T, logOut io.Writer) flowApp {
	t.Helper()
	return newFlowAppWith(t, flowConfig{logOut: logOut})
}

// flowConfig customizes newFlowAppWith. The zero value discards the log,
// uses the default Argon2 parameters and the fakeUsers service.
type flowConfig struct {
	logOut io.Writer
	argon2 security.Argon2Params
	// users builds the registration service from the app's Security.
	users func(sec *security.Security) UserService
}

func newFlowAppWith(t *testing.T, cfg flowConfig) flowApp {
	t.Helper()
	if cfg.logOut == nil {
		cfg.logOut = io.Discard
	}
	logger := slog.New(NewLogHandler(slog.NewJSONHandler(cfg.logOut, nil)))
	store := newMemSessions()
	sec, err := security.New(security.Options{
		Logger:                 logger,
		Sessions:               store,
		RenderError:            WriteError,
		SessionIdleTimeout:     time.Minute,
		SessionAbsoluteTimeout: time.Hour,
		RequestTimeout:         time.Minute,
		Argon2:                 cfg.argon2,
	})
	if err != nil {
		t.Fatal(err)
	}
	var users UserService = fakeUsers{}
	if cfg.users != nil {
		users = cfg.users(sec)
	}
	app, err := NewHandler(Deps{Logger: logger, Security: sec, Auth: credAuth{}, Users: users})
	if err != nil {
		t.Fatal(err)
	}
	return flowApp{app: app, sessions: store}
}

// send runs one request; cookie may be nil.
func (f flowApp) send(t *testing.T, method, path, body string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	f.app.ServeHTTP(rr, req)
	return rr
}

// csrf runs GET /api/csrf and returns the session cookie and the token.
func (f flowApp) csrf(t *testing.T, cookie *http.Cookie) (*http.Cookie, string) {
	t.Helper()
	rr := f.send(t, http.MethodGet, "/api/csrf", "", cookie, nil)
	var body csrfResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || rr.Code != http.StatusOK {
		t.Fatalf("GET /api/csrf: %d %s", rr.Code, rr.Body.String())
	}
	if c := sessionCookie(rr); c != nil {
		cookie = c
	}
	return cookie, body.CSRFToken
}

func sessionCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == security.SessionCookieName {
			return c
		}
	}
	return nil
}

func loginBody(login, password string) string {
	return fmt.Sprintf(`{"login":%q,"senha":%q}`, login, password)
}

func TestLoginFlow(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	anon, token := f.csrf(t, nil)

	rr := f.send(t, http.MethodPost, "/api/login", loginBody(flowLogin, flowPassword), anon,
		map[string]string{security.CSRFHeaderName: token, "Sec-Fetch-Site": "same-origin"})
	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d, body %s", rr.Code, rr.Body.String())
	}
	var user userResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user != (userResponse{UserID: 1, Login: flowLogin, Role: domain.RoleAdmin}) {
		t.Errorf("login body = %+v", user)
	}
	authCookie := sessionCookie(rr)
	if authCookie == nil || authCookie.Value == anon.Value {
		t.Fatalf("login must set a new session cookie, got %+v", authCookie)
	}
	if n := f.sessions.count(); n != 1 {
		t.Errorf("%d sessions stored, want 1 (the anonymous one is deleted)", n)
	}

	// D-4: the CSRF token changes at login; the old one is refused.
	_, newToken := f.csrf(t, authCookie)
	if newToken == "" || newToken == token {
		t.Errorf("token after login = %q, want a new token", newToken)
	}
	rr = f.send(t, http.MethodPost, "/api/login", loginBody(flowLogin, flowPassword), authCookie,
		map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusForbidden {
		t.Errorf("old CSRF token after login: status = %d, want 403", rr.Code)
	}

	// The anonymous cookie no longer has a session.
	rr = f.send(t, http.MethodPost, "/api/login", loginBody(flowLogin, flowPassword), anon,
		map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusForbidden {
		t.Errorf("anonymous cookie after login: status = %d, want 403", rr.Code)
	}
}

func TestLoginRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		noCookie   bool
		headers    func(token string) map[string]string
		wantStatus int
		wantError  string
	}{
		{
			name: "no CSRF token", body: loginBody(flowLogin, flowPassword),
			headers:    func(string) map[string]string { return nil },
			wantStatus: http.StatusForbidden, wantError: "forbidden",
		},
		{
			name: "wrong CSRF token", body: loginBody(flowLogin, flowPassword),
			headers: func(string) map[string]string {
				return map[string]string{security.CSRFHeaderName: "AAAAAAAAAAAAAAAAAAAAAAAAAA"}
			},
			wantStatus: http.StatusForbidden, wantError: "forbidden",
		},
		{
			name: "no session cookie", body: loginBody(flowLogin, flowPassword), noCookie: true,
			headers:    func(tok string) map[string]string { return map[string]string{security.CSRFHeaderName: tok} },
			wantStatus: http.StatusForbidden, wantError: "forbidden",
		},
		{
			name: "cross-site browser request", body: loginBody(flowLogin, flowPassword),
			headers: func(tok string) map[string]string {
				return map[string]string{security.CSRFHeaderName: tok, "Sec-Fetch-Site": "cross-site"}
			},
			wantStatus: http.StatusForbidden, wantError: "forbidden",
		},
		{
			name: "wrong password", body: loginBody(flowLogin, "a wrong long password"),
			headers:    func(tok string) map[string]string { return map[string]string{security.CSRFHeaderName: tok} },
			wantStatus: http.StatusUnauthorized, wantError: "unauthorized",
		},
		{
			name: "unknown user", body: loginBody("nobody", flowPassword),
			headers:    func(tok string) map[string]string { return map[string]string{security.CSRFHeaderName: tok} },
			wantStatus: http.StatusUnauthorized, wantError: "unauthorized",
		},
		{
			name: "unknown field", body: `{"login":"admin","senha":"x","extra":1}`,
			headers:    func(tok string) map[string]string { return map[string]string{security.CSRFHeaderName: tok} },
			wantStatus: http.StatusBadRequest, wantError: "bad_request",
		},
	}
	messages := map[string]string{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFlowApp(t)
			cookie, token := f.csrf(t, nil)
			if tt.noCookie {
				cookie = nil
			}
			rr := f.send(t, http.MethodPost, "/api/login", tt.body, cookie, tt.headers(token))
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			var body errorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body.Error != tt.wantError {
				t.Errorf("body = %+v, %v, want error %q", body, err, tt.wantError)
			}
			if sessionCookie(rr) != nil {
				t.Error("a rejected login set a session cookie")
			}
			messages[tt.name] = body.Message
		})
	}
	// Same message for an unknown user and a wrong password.
	if messages["wrong password"] != messages["unknown user"] {
		t.Errorf("401 messages differ: %q / %q", messages["wrong password"], messages["unknown user"])
	}
}

// loginAs runs csrf → login and returns the authenticated cookie and the
// new CSRF token (D-4).
func (f flowApp) loginAs(t *testing.T, login string) (*http.Cookie, string) {
	t.Helper()
	anon, token := f.csrf(t, nil)
	rr := f.send(t, http.MethodPost, "/api/login", loginBody(login, flowPassword), anon,
		map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusOK {
		t.Fatalf("login as %s: %d %s", login, rr.Code, rr.Body.String())
	}
	cookie := sessionCookie(rr)
	_, token = f.csrf(t, cookie)
	return cookie, token
}

func TestMeReturnsTheCaller(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	cookie, _ := f.loginAs(t, comumLogin)
	rr := f.send(t, http.MethodGet, "/api/me", "", cookie, nil)
	var me userResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me != (userResponse{UserID: 2, Role: domain.RoleComum}) {
		t.Errorf("me = %+v, want user 2 with role comum", me)
	}
}

// TestLogout covers the logout criterion: 204, the session row is deleted,
// the cookie is expired, and the old cookie then gets 401.
func TestLogout(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	cookie, token := f.loginAs(t, flowLogin)
	if f.sessions.count() != 1 {
		t.Fatalf("%d sessions before logout, want 1", f.sessions.count())
	}

	rr := f.send(t, http.MethodPost, "/api/logout", "", cookie, map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204 (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("Set-Cookie = %q, want Max-Age=0", rr.Header().Get("Set-Cookie"))
	}
	if n := f.sessions.count(); n != 0 {
		t.Errorf("%d sessions after logout, want 0", n)
	}

	rr = f.send(t, http.MethodGet, "/api/me", "", cookie, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("old cookie after logout: status = %d, want 401", rr.Code)
	}
}

func TestLogoutNeedsTokenAndSession(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	cookie, _ := f.loginAs(t, flowLogin)
	if rr := f.send(t, http.MethodPost, "/api/logout", "", cookie, nil); rr.Code != http.StatusForbidden {
		t.Errorf("logout without token: status = %d, want 403", rr.Code)
	}

	anon, token := f.csrf(t, nil)
	if rr := f.send(t, http.MethodPost, "/api/logout", "", anon, map[string]string{security.CSRFHeaderName: token}); rr.Code != http.StatusUnauthorized {
		t.Errorf("anonymous logout with a valid token: status = %d, want 401", rr.Code)
	}
}
