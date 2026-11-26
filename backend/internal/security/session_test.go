package security

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"backend_si/internal/domain"
)

const (
	testIdle     = 15 * time.Minute
	testAbsolute = 8 * time.Hour
)

var errStore = errors.New("store is down")

// memStore is an in-memory SessionStore. Get fills Role from roles, like
// the repository JOIN. Each *Err field makes that method fail.
type memStore struct {
	mu                                  sync.Mutex
	rows                                map[string]domain.Session
	roles                               map[domain.UserID]domain.Role
	calls                               []string
	createErr, getErr, touchErr, delErr error
	touchNotFound                       bool
	purgeErr                            error
	purgeAt                             []time.Time
}

func newMemStore() *memStore {
	return &memStore{
		rows:  map[string]domain.Session{},
		roles: map[domain.UserID]domain.Role{1: domain.RoleAdmin, 2: domain.RoleComum},
	}
}

func (m *memStore) Create(_ context.Context, s domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "Create")
	if m.createErr != nil {
		return m.createErr
	}
	if len(s.TokenHash) != sha256.Size {
		return errors.New("token hash must have 32 bytes")
	}
	if !s.ExpiresAt.After(s.LastSeenAt) {
		return errors.New("violates sessions_expiry_order")
	}
	m.rows[string(s.TokenHash)] = s
	return nil
}

func (m *memStore) Get(_ context.Context, h []byte) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "Get")
	if m.getErr != nil {
		return domain.Session{}, m.getErr
	}
	s, ok := m.rows[string(h)]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	s.Role = m.roles[s.UserID]
	return s, nil
}

func (m *memStore) Touch(_ context.Context, h []byte, lastSeen, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "Touch")
	if m.touchErr != nil {
		return m.touchErr
	}
	s, ok := m.rows[string(h)]
	if !ok || m.touchNotFound {
		return domain.ErrNotFound
	}
	if !expires.After(lastSeen) {
		return errors.New("violates sessions_expiry_order")
	}
	s.LastSeenAt, s.ExpiresAt = lastSeen, expires
	m.rows[string(h)] = s
	return nil
}

func (m *memStore) Delete(_ context.Context, h []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "Delete")
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.rows, string(h))
	return nil
}

func (m *memStore) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.purgeAt = append(m.purgeAt, now)
	if m.purgeErr != nil {
		return 0, m.purgeErr
	}
	var n int64
	for k := range m.rows {
		if !m.rows[k].ExpiresAt.After(now) {
			delete(m.rows, k)
			n++
		}
	}
	return n, nil
}

func (m *memStore) row(token string) (domain.Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.rows[string(hashToken(token))]
	return s, ok
}

func (m *memStore) size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rows)
}

func (m *memStore) callLog() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.calls...)
}

func (m *memStore) resetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
}

// clock is an adjustable time source for Options.Now.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *clock { return &clock{now: time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)} }

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newSessionSecurity(t *testing.T, store *memStore, clk *clock, logOut *bytes.Buffer) *Security {
	t.Helper()
	return newTestSecurityWith(t, func(o *Options) {
		o.Sessions = store
		o.Now = clk.Now
		o.SessionIdleTimeout = testIdle
		o.SessionAbsoluteTimeout = testAbsolute
		if logOut != nil {
			o.Logger = newJSONLogger(logOut)
		}
	})
}

// observed is what a handler behind Session saw.
type observed struct {
	called    bool
	principal Principal
	auth      bool
	csrf      string
	session   bool
}

// serve sends a request (with the session cookie when token is set)
// through Session and returns the response and what the handler saw.
func serve(t *testing.T, s *Security, token string) (*httptest.ResponseRecorder, observed) {
	t.Helper()
	var o observed
	h := s.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.called = true
		o.principal, o.auth = PrincipalFrom(r.Context())
		var st sessionState
		st, o.session = sessionFrom(r.Context())
		o.csrf = st.csrfToken
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/me", http.NoBody)
	if token != "" {
		req.AddCookie(reqCookie(token))
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr, o
}

// reqCookie is the session cookie a browser sends back.
func reqCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// sessionCookieFrom returns the session cookie set in rr.
func sessionCookieFrom(t *testing.T, rr *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == SessionCookieName {
			return c
		}
	}
	t.Fatalf("no %s cookie in %v", SessionCookieName, rr.Header().Values("Set-Cookie"))
	return nil
}

func hasSessionCookie(rr *httptest.ResponseRecorder) bool {
	for _, v := range rr.Header().Values("Set-Cookie") {
		if strings.HasPrefix(v, SessionCookieName+"=") {
			return true
		}
	}
	return false
}

// login creates an authenticated session for userID through StartSession
// and returns its token.
func login(t *testing.T, s *Security, userID domain.UserID, role domain.Role) string {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/login", http.NoBody)
	if err := s.StartSession(rr, req, domain.User{ID: userID, Role: role}); err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	return sessionCookieFrom(t, rr).Value
}

func TestSessionWithoutCookieWritesNothing(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, cookie string }{
		{"no cookie", ""},
		{"malformed cookie", "qualquer-valor"},
		{"lowercase token", strings.ToLower("ABCDEFGHIJKLMNOPQRSTUVWXYZ")},
		{"too long", "ABCDEFGHIJKLMNOPQRSTUVWXYZA"},
		{"invalid base32 digit", "ABCDEFGHIJKLMNOPQRSTUVWXY1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMemStore()
			s := newSessionSecurity(t, store, newClock(), nil)
			rr, o := serve(t, s, tt.cookie)

			if !o.called || o.auth || o.session {
				t.Errorf("handler saw %+v, want an anonymous request without session", o)
			}
			if calls := store.callLog(); len(calls) != 0 {
				t.Errorf("store calls = %v, want none (D-8)", calls)
			}
			if hasSessionCookie(rr) {
				t.Error("Session set a cookie; only CSRFToken creates sessions (D-8)")
			}
		})
	}
}

func TestSessionUnknownTokenIsAnonymous(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	rr, o := serve(t, s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	if !o.called || o.auth || o.session {
		t.Errorf("handler saw %+v, want anonymous", o)
	}
	if hasSessionCookie(rr) || store.size() != 0 {
		t.Error("an unknown token must not create a session")
	}
}

func TestSessionLoadsAndTouches(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	clk := newClock()
	s := newSessionSecurity(t, store, clk, nil)
	token := login(t, s, 1, domain.RoleAdmin)
	created, _ := store.row(token)

	clk.Advance(5 * time.Minute)
	rr, o := serve(t, s, token)
	if !o.called || !o.auth || o.principal != (Principal{UserID: 1, Role: domain.RoleAdmin}) {
		t.Fatalf("handler saw %+v, want admin 1", o)
	}
	if !o.session || o.csrf != created.CSRFToken {
		t.Errorf("csrf in context = %q, want the session's", o.csrf)
	}
	if hasSessionCookie(rr) {
		t.Error("loading a session must not set the cookie again")
	}
	row, _ := store.row(token)
	if !row.LastSeenAt.Equal(clk.Now()) || !row.ExpiresAt.Equal(clk.Now().Add(testIdle)) {
		t.Errorf("after touch: last_seen %v, expires %v (D-10)", row.LastSeenAt, row.ExpiresAt)
	}
}

// TestSessionPrincipalCarriesTheProfile covers spec Q-16: the login and the
// registration time the store reads with the role reach the principal, for
// GET /api/me.
func TestSessionPrincipalCarriesTheProfile(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	token := login(t, s, 2, domain.RoleComum)
	registered := time.Date(2026, 11, 20, 12, 15, 0, 0, time.UTC)
	store.mu.Lock()
	key := string(hashToken(token))
	row := store.rows[key]
	row.Login, row.UserCreatedAt = "maria.silva", registered
	store.rows[key] = row
	store.mu.Unlock()

	_, o := serve(t, s, token)
	want := Principal{UserID: 2, Role: domain.RoleComum, Login: "maria.silva", CreatedAt: registered}
	if !o.auth || o.principal != want {
		t.Errorf("principal = %+v (auth %v), want %+v", o.principal, o.auth, want)
	}
}

func TestSessionAnonymousHasNoPrincipal(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	rr := httptest.NewRecorder()
	csrf, err := s.CSRFToken(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}

	_, o := serve(t, s, sessionCookieFrom(t, rr).Value)
	if o.auth || !o.session || o.csrf != csrf {
		t.Errorf("handler saw %+v, want an anonymous session with csrf %q", o, csrf)
	}
}

func TestSessionRoleComesFromStore(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	token := login(t, s, 2, domain.RoleComum)

	// The user's role changes (or the user row is gone): the session follows.
	store.mu.Lock()
	store.roles[2] = ""
	store.mu.Unlock()
	if _, o := serve(t, s, token); o.auth {
		t.Errorf("handler saw principal %+v for a user without a valid role", o.principal)
	}
}

func TestSessionTimeouts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		activity time.Duration // interval between requests
		elapsed  time.Duration // since login, before the last request
		valid    bool
	}{
		{"idle just below the limit", 0, testIdle - time.Second, true},
		{"idle at the limit", 0, testIdle, false},
		{"idle above the limit", 0, testIdle + time.Hour, false},
		{"active, just below the absolute limit", 10 * time.Minute, testAbsolute - time.Second, true},
		{"active, at the absolute limit", 10 * time.Minute, testAbsolute, false},
		{"active, above the absolute limit", 10 * time.Minute, testAbsolute + time.Minute, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMemStore()
			clk := newClock()
			s := newSessionSecurity(t, store, clk, nil)
			token := login(t, s, 1, domain.RoleAdmin)

			// Continuous activity (always before the absolute limit) keeps
			// the idle timeout from firing.
			var spent time.Duration
			for tt.activity > 0 && spent+tt.activity < min(tt.elapsed, testAbsolute) {
				clk.Advance(tt.activity)
				spent += tt.activity
				if _, o := serve(t, s, token); !o.auth {
					t.Fatalf("session lost after %v of activity", spent)
				}
			}
			clk.Advance(tt.elapsed - spent)
			store.resetCalls()

			_, o := serve(t, s, token)
			if o.auth != tt.valid {
				t.Fatalf("authenticated = %v, want %v", o.auth, tt.valid)
			}
			_, stored := store.row(token)
			if !tt.valid {
				if o.session || stored {
					t.Error("an expired session must be deleted and the request anonymous (D-9)")
				}
				if calls := store.callLog(); len(calls) != 2 || calls[1] != "Delete" {
					t.Errorf("store calls = %v, want Get then Delete", calls)
				}
			}
		})
	}
}

func TestSessionStoredExpiryIsNeverExtended(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	clk := newClock()
	s := newSessionSecurity(t, store, clk, nil)
	token := login(t, s, 1, domain.RoleAdmin)

	// A row whose stored expiry is earlier than both timeouts allow.
	store.mu.Lock()
	h := string(hashToken(token))
	row := store.rows[h]
	row.ExpiresAt = clk.Now().Add(time.Minute)
	store.rows[h] = row
	store.mu.Unlock()

	clk.Advance(time.Minute)
	if _, o := serve(t, s, token); o.auth {
		t.Error("a session past its stored expires_at was accepted")
	}
}

func TestExpiresAt(t *testing.T) {
	t.Parallel()

	s := newSessionSecurity(t, newMemStore(), newClock(), nil)
	created := time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{"at creation", created, created.Add(testIdle)},
		{"mid-life", created.Add(time.Hour), created.Add(time.Hour + testIdle)},
		{"near the absolute limit", created.Add(testAbsolute - 5*time.Minute), created.Add(testAbsolute)},
	}
	for _, tt := range tests {
		if got := s.expiresAt(created, tt.now); !got.Equal(tt.want) {
			t.Errorf("%s: expiresAt = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSessionStoreFailuresAre500(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(m *memStore)
		expire bool
		want   int
		auth   bool
	}{
		{"get fails", func(m *memStore) { m.getErr = errStore }, false, http.StatusInternalServerError, false},
		{"touch fails", func(m *memStore) { m.touchErr = errStore }, false, http.StatusInternalServerError, false},
		{"delete of expired session fails", func(m *memStore) { m.delErr = errStore }, true, http.StatusInternalServerError, false},
		{"touch races with logout", func(m *memStore) { m.touchNotFound = true }, false, http.StatusOK, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMemStore()
			clk := newClock()
			var logs bytes.Buffer
			s := newSessionSecurity(t, store, clk, &logs)
			token := login(t, s, 1, domain.RoleAdmin)
			if tt.expire {
				clk.Advance(testIdle)
			}
			tt.mutate(store)

			rr, o := serve(t, s, token)
			if rr.Code != tt.want {
				t.Errorf("status = %d, want %d", rr.Code, tt.want)
			}
			if tt.want == http.StatusInternalServerError {
				if o.called {
					t.Error("a store failure must not continue as anonymous (constitution 2)")
				}
				if !strings.Contains(logs.String(), errStore.Error()) {
					t.Error("the store failure was not logged")
				}
				if strings.Contains(logs.String(), token) {
					t.Error("the log contains the session token")
				}
			}
			if o.auth != tt.auth {
				t.Errorf("authenticated = %v, want %v", o.auth, tt.auth)
			}
		})
	}
}

func TestStartSession(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	clk := newClock()
	s := newSessionSecurity(t, store, clk, nil)

	// An anonymous session first, as GET /api/csrf creates.
	rr := httptest.NewRecorder()
	anonCSRF, err := s.CSRFToken(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}
	anonToken := sessionCookieFrom(t, rr).Value

	// Login runs behind Session, which loads the anonymous session.
	clk.Advance(time.Minute)
	rr = httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/login", http.NoBody)
	req.AddCookie(reqCookie(anonToken))
	var startErr error
	s.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startErr = s.StartSession(w, r, domain.User{ID: 1, Role: domain.RoleAdmin})
	})).ServeHTTP(rr, req)
	if startErr != nil {
		t.Fatalf("StartSession: %v", startErr)
	}

	c := sessionCookieFrom(t, rr)
	if c.Value == anonToken || !validToken(c.Value) {
		t.Errorf("new token %q must be new and well-formed (session fixation)", c.Value)
	}
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Domain != "" {
		t.Errorf("cookie attributes = %+v, want HttpOnly, Secure, SameSite=Lax, Path=/, no Domain", c)
	}
	if c.MaxAge != 0 || !c.Expires.IsZero() {
		t.Errorf("cookie has MaxAge %d / Expires %v, want a browser-session cookie (D-11)", c.MaxAge, c.Expires)
	}

	if _, ok := store.row(anonToken); ok {
		t.Error("the anonymous session was not deleted")
	}
	row, ok := store.row(c.Value)
	if !ok {
		t.Fatal("the new session is not stored under the SHA-256 of its token")
	}
	if row.UserID != 1 || row.CSRFToken == anonCSRF || !validToken(row.CSRFToken) {
		t.Errorf("row = %+v, want user 1 and a new CSRF token (D-4)", row)
	}
	if !row.CreatedAt.Equal(clk.Now()) || !row.ExpiresAt.Equal(clk.Now().Add(testIdle)) {
		t.Errorf("created %v, expires %v: the absolute window must restart at login", row.CreatedAt, row.ExpiresAt)
	}
	if store.size() != 1 {
		t.Errorf("store holds %d sessions, want 1", store.size())
	}
	for _, r := range store.rows {
		if bytes.Contains(r.TokenHash, []byte(c.Value)) {
			t.Error("the token itself is stored")
		}
	}
}

func TestStartSessionFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		user   domain.User
		mutate func(m *memStore)
	}{
		{"zero user", domain.User{Role: domain.RoleAdmin}, func(*memStore) {}},
		{"invalid role", domain.User{ID: 1, Role: "root"}, func(*memStore) {}},
		{"create fails", domain.User{ID: 1, Role: domain.RoleAdmin}, func(m *memStore) { m.createErr = errStore }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMemStore()
			tt.mutate(store)
			s := newSessionSecurity(t, store, newClock(), nil)
			rr := httptest.NewRecorder()
			err := s.StartSession(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/login", http.NoBody), tt.user)
			if err == nil {
				t.Fatal("StartSession succeeded")
			}
			if hasSessionCookie(rr) || store.size() != 0 {
				t.Error("a failed StartSession must not set a cookie or store a session")
			}
			// D-4: an invalid user is an internal error, so login answers 500
			// after the RespondError mapping, never 400.
			if errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("StartSession() = %v, must not wrap ErrInvalidInput", err)
			}
			if got := statusFor(err); got != http.StatusInternalServerError {
				t.Errorf("statusFor(StartSession error) = %d, want 500", got)
			}
		})
	}
}

func TestEndSession(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	token := login(t, s, 1, domain.RoleAdmin)

	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/logout", http.NoBody)
	req.AddCookie(reqCookie(token))
	var endErr error
	s.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endErr = s.EndSession(w, r)
	})).ServeHTTP(rr, req)
	if endErr != nil {
		t.Fatalf("EndSession: %v", endErr)
	}

	if _, ok := store.row(token); ok {
		t.Error("the session row was not deleted")
	}
	c := sessionCookieFrom(t, rr)
	if c.MaxAge >= 0 || c.Value != "" {
		t.Errorf("cookie = %+v, want an expired empty cookie", c)
	}
	if !strings.Contains(rr.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("Set-Cookie = %q, want Max-Age=0", rr.Header().Get("Set-Cookie"))
	}

	// The old cookie is now anonymous.
	if _, o := serve(t, s, token); o.auth || o.session {
		t.Errorf("old cookie after logout: %+v, want anonymous", o)
	}
}

func TestEndSessionWithoutSession(t *testing.T) {
	t.Parallel()

	s := newSessionSecurity(t, newMemStore(), newClock(), nil)
	rr := httptest.NewRecorder()
	if err := s.EndSession(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/logout", http.NoBody)); err != nil {
		t.Fatalf("EndSession: %v", err)
	}
	if c := sessionCookieFrom(t, rr); c.MaxAge >= 0 {
		t.Errorf("cookie = %+v, want expired", c)
	}
}

func TestEndSessionDeleteFails(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	token := login(t, s, 1, domain.RoleAdmin)
	store.delErr = errStore

	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/logout", http.NoBody)
	req.AddCookie(reqCookie(token))
	var endErr error
	s.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endErr = s.EndSession(w, r)
	})).ServeHTTP(rr, req)
	if !errors.Is(endErr, errStore) {
		t.Fatalf("EndSession = %v, want the store error", endErr)
	}
}

func TestCSRFToken(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	clk := newClock()
	s := newSessionSecurity(t, store, clk, nil)

	rr := httptest.NewRecorder()
	csrf, err := s.CSRFToken(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody))
	if err != nil {
		t.Fatalf("CSRFToken: %v", err)
	}
	if !validToken(csrf) {
		t.Errorf("csrf token %q is not a rand.Text() token (D-11)", csrf)
	}
	c := sessionCookieFrom(t, rr)
	row, ok := store.row(c.Value)
	if !ok || row.UserID != 0 || row.CSRFToken != csrf {
		t.Fatalf("anonymous session row = %+v, %v", row, ok)
	}
	if c.Value == csrf {
		t.Error("the session token and the CSRF token must differ")
	}

	// With the session loaded, the same token comes back and nothing is created.
	store.resetCalls()
	rr = httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody)
	req.AddCookie(c)
	var again string
	s.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		again, err = s.CSRFToken(w, r)
	})).ServeHTTP(rr, req)
	if err != nil || again != csrf {
		t.Errorf("second CSRFToken = %q, %v, want %q", again, err, csrf)
	}
	if hasSessionCookie(rr) || store.size() != 1 {
		t.Error("a second CSRFToken created another session")
	}
	for _, call := range store.callLog() {
		if call == "Create" {
			t.Error("a second CSRFToken called Create")
		}
	}
}

func TestCSRFTokenStoreFailure(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	store.createErr = errStore
	s := newSessionSecurity(t, store, newClock(), nil)
	rr := httptest.NewRecorder()
	if _, err := s.CSRFToken(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody)); !errors.Is(err, errStore) {
		t.Fatalf("CSRFToken = %v, want the store error", err)
	}
	if hasSessionCookie(rr) {
		t.Error("a cookie was set for a session that was not stored")
	}
}

func TestPurgeExpiredSessions(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	clk := newClock()
	var logs syncBuffer
	s := newTestSecurityWith(t, func(o *Options) {
		o.Sessions = store
		o.Now = clk.Now
		o.SessionIdleTimeout = testIdle
		o.SessionAbsoluteTimeout = testAbsolute
		o.Logger = newJSONLogger(&logs)
	})
	expired := login(t, s, 1, domain.RoleAdmin)
	clk.Advance(testIdle)
	live := login(t, s, 2, domain.RoleComum)
	store.mu.Lock()
	store.purgeErr = errStore // the first run fails
	store.mu.Unlock()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		s.PurgeExpiredSessions(ctx, 5*time.Millisecond)
		close(done)
	}()

	waitFor(t, func() bool { return strings.Contains(logs.String(), errStore.Error()) })
	store.mu.Lock()
	store.purgeErr = nil // the loop goes on after a failure
	store.mu.Unlock()
	waitFor(t, func() bool { _, ok := store.row(expired); return !ok })

	if _, ok := store.row(live); !ok {
		t.Error("the purge deleted a live session")
	}
	store.mu.Lock()
	for _, at := range store.purgeAt {
		if !at.Equal(clk.Now()) {
			t.Errorf("purge used %v, want the injected clock %v", at, clk.Now())
		}
	}
	store.mu.Unlock()

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("PurgeExpiredSessions did not stop with the context")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in 5s")
		}
		time.Sleep(time.Millisecond)
	}
}

// syncBuffer is a bytes.Buffer safe for a logger in another goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
