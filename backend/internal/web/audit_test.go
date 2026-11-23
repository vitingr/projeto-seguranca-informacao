package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"backend_si/internal/domain"
)

// memAudit is an in-memory service.AuditRepository. List returns the newest
// (last appended) first. With fail set, Append fails and stores nothing.
type memAudit struct {
	mu     sync.Mutex
	events []domain.AuditEvent
	fail   bool
}

func newMemAudit() *memAudit { return &memAudit{} }

var errAuditDown = errors.New("audit store down")

func (m *memAudit) Append(_ context.Context, ev domain.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errAuditDown
	}
	ev.ID = domain.AuditEventID(len(m.events) + 1)
	ev.CreatedAt = time.Date(2026, 11, 22, 13, 0, len(m.events), 0, time.UTC)
	ev.ActorLogin = credLogins[ev.ActorID]
	if ev.TargetType == domain.AuditTargetUser {
		ev.TargetLogin = credLogins[domain.UserID(ev.TargetID)]
	}
	m.events = append(m.events, ev)
	return nil
}

func (m *memAudit) List(_ context.Context, typ domain.AuditEventType, limit, offset int) ([]domain.AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.AuditEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		if typ == "" || m.events[i].Type == typ {
			out = append(out, m.events[i])
		}
	}
	if offset >= len(out) {
		return []domain.AuditEvent{}, nil
	}
	return out[offset:min(offset+limit, len(out))], nil
}

func (m *memAudit) snapshot() []domain.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.events)
}

func (m *memAudit) setFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fail = fail
}

// since returns the events appended after the first n.
func (m *memAudit) since(n int) []domain.AuditEvent {
	return m.snapshot()[n:]
}

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

// stripEvent keeps the fields a request decides (not the ID, time or
// logins that the store fills in).
func stripEvent(ev domain.AuditEvent) domain.AuditEvent {
	return domain.AuditEvent{Type: ev.Type, ActorID: ev.ActorID, TargetType: ev.TargetType, TargetID: ev.TargetID, IP: ev.IP}
}

func stripEvents(evs []domain.AuditEvent) []domain.AuditEvent {
	out := make([]domain.AuditEvent, 0, len(evs))
	for i := range evs {
		out = append(out, stripEvent(evs[i]))
	}
	return out
}

// TestAuditEventsAreReadOnly covers 006 S-1 in the contract and over HTTP:
// the audit log has only GET, and no other method changes it.
func TestAuditEventsAreReadOnly(t *testing.T) {
	t.Parallel()

	doc := loadContract(t)
	ops, err := doc.operations()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := newTestApp(t)
	var found int
	for pattern := range routePolicies(h.routes()) {
		method, path, _ := strings.Cut(pattern, " ")
		if strings.HasPrefix(path, "/api/audit-events") {
			found++
			if method != http.MethodGet {
				t.Errorf("routes.go: %s on the audit log, want GET only", pattern)
			}
		}
	}
	for pattern := range ops {
		method, path, _ := strings.Cut(pattern, " ")
		if strings.HasPrefix(path, "/api/audit-events") && method != http.MethodGet {
			t.Errorf("api/openapi.json: %s on the audit log, want GET only", pattern)
		}
	}
	if found != 1 {
		t.Errorf("%d audit routes in routes.go, want 1 (GET /api/audit-events)", found)
	}

	f := newFlowApp(t)
	cookie, token := f.loginAs(t, flowLogin)
	before := f.audit.snapshot()
	for _, path := range []string{"/api/audit-events", "/api/audit-events/1"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			rr := f.send(t, method, path, `{"event":"logout"}`, cookie, csrfHeader(token))
			if rr.Code >= 200 && rr.Code < 300 {
				t.Errorf("%s %s: status %d, want no 2xx", method, path, rr.Code)
			}
		}
	}
	if after := f.audit.snapshot(); !slices.Equal(before, after) {
		t.Errorf("the audit log changed:\nbefore %+v\nafter  %+v", before, after)
	}
}

// TestAuditRecordsOneEventPerOutcome covers 006 D-4 and S-5: one event per
// outcome of login, logout and registration, with the right actor and
// target, and nothing for requests rejected with 400, 403, 409 or 422.
func TestAuditRecordsOneEventPerOutcome(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	f := newCadastroApp(t, &logBuf, newMemUserRepo())
	const peer = "192.0.2.1" // the httptest default RemoteAddr
	ip := mustAddr(t, peer)

	login := func(login, password string, headers func(token string) map[string]string) int {
		t.Helper()
		anon, token := f.csrf(t, nil)
		return f.send(t, http.MethodPost, "/api/login", loginBody(login, password), anon, headers(token)).Code
	}
	withToken := csrfHeader

	tests := []struct {
		name       string
		run        func() int
		wantStatus int
		want       []domain.AuditEvent
	}{
		{
			name:       "login succeeded",
			run:        func() int { return login(flowLogin, flowPassword, withToken) },
			wantStatus: http.StatusOK,
			want:       []domain.AuditEvent{{Type: domain.AuditLoginSucceeded, ActorID: 1, IP: ip}},
		},
		{
			name:       "wrong password targets the existing user",
			run:        func() int { return login(comumLogin, "a wrong long password", withToken) },
			wantStatus: http.StatusUnauthorized,
			want:       []domain.AuditEvent{{Type: domain.AuditLoginFailed, TargetType: domain.AuditTargetUser, TargetID: 2, IP: ip}},
		},
		{
			name:       "unknown login has no target",
			run:        func() int { return login("nobody", flowPassword, withToken) },
			wantStatus: http.StatusUnauthorized,
			want:       []domain.AuditEvent{{Type: domain.AuditLoginFailed, IP: ip}},
		},
		{
			name: "malformed login body records nothing",
			run: func() int {
				anon, token := f.csrf(t, nil)
				return f.send(t, http.MethodPost, "/api/login", `{"login":`, anon, csrfHeader(token)).Code
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "login without a CSRF token records nothing",
			run:        func() int { return login(flowLogin, flowPassword, func(string) map[string]string { return nil }) },
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		n := len(f.audit.snapshot())
		if got := tt.run(); got != tt.wantStatus {
			t.Fatalf("%s: status %d, want %d", tt.name, got, tt.wantStatus)
		}
		if got := stripEvents(f.audit.since(n)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: events %+v, want %+v", tt.name, got, tt.want)
		}
	}

	admin, token := f.loginAs(t, flowLogin)
	steps := []struct {
		name       string
		body       string
		wantStatus int
		want       []domain.AuditEvent
	}{
		{
			"registration", cadastroBody("novo.usuario", cadastroPassword, "comum"), http.StatusCreated,
			[]domain.AuditEvent{{Type: domain.AuditUserCreated, ActorID: 1, TargetType: domain.AuditTargetUser, TargetID: 100, IP: ip}},
		},
		{"taken login (409)", cadastroBody("novo.usuario", cadastroPassword, "comum"), http.StatusConflict, nil},
		{"invalid role (422)", cadastroBody("outro.usuario", cadastroPassword, "root"), http.StatusUnprocessableEntity, nil},
		{"malformed body (400)", `{"login":1}`, http.StatusBadRequest, nil},
	}
	for _, st := range steps {
		n := len(f.audit.snapshot())
		rr := f.send(t, http.MethodPost, "/api/cadastro", st.body, admin, csrfHeader(token))
		if rr.Code != st.wantStatus {
			t.Fatalf("%s: status %d, want %d (%s)", st.name, rr.Code, st.wantStatus, rr.Body.String())
		}
		if got := stripEvents(f.audit.since(n)); !slices.Equal(got, st.want) {
			t.Errorf("%s: events %+v, want %+v", st.name, got, st.want)
		}
	}
	if auditLine(logBuf.String()) == "" {
		t.Error(`the "user registered" log line is gone (006 D-6 keeps it)`)
	}

	n := len(f.audit.snapshot())
	if rr := f.send(t, http.MethodPost, "/api/logout", "", admin, csrfHeader(token)); rr.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rr.Code)
	}
	want := []domain.AuditEvent{{Type: domain.AuditLogout, ActorID: 1, IP: ip}}
	if got := stripEvents(f.audit.since(n)); !slices.Equal(got, want) {
		t.Errorf("logout: events %+v, want %+v", got, want)
	}
}

// TestAuditWriteFailureKeepsResponses covers 006 D-6 and S-5: when the log
// cannot be written, login, logout and registration answer as before and
// the failure is logged at ERROR, with the event and IDs only.
func TestAuditWriteFailureKeepsResponses(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	f := newCadastroApp(t, &logBuf, newMemUserRepo())
	f.audit.setFail(true)

	cookie, token := f.loginAs(t, flowLogin) // fails the test unless 200
	if cookie == nil || f.sessions.count() != 1 {
		t.Fatalf("login with the log down: cookie %v, %d sessions", cookie, f.sessions.count())
	}
	anon, anonToken := f.csrf(t, nil)
	if rr := f.send(t, http.MethodPost, "/api/login", loginBody(flowLogin, "a wrong long password"), anon, csrfHeader(anonToken)); rr.Code != http.StatusUnauthorized {
		t.Errorf("failed login with the log down: %d, want 401", rr.Code)
	}
	if rr := f.send(t, http.MethodPost, "/api/cadastro", cadastroBody("novo.usuario", cadastroPassword, "comum"), cookie, csrfHeader(token)); rr.Code != http.StatusCreated {
		t.Errorf("registration with the log down: %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
	if rr := f.send(t, http.MethodPost, "/api/logout", "", cookie, csrfHeader(token)); rr.Code != http.StatusNoContent {
		t.Errorf("logout with the log down: %d, want 204", rr.Code)
	}

	var failures []string
	for line := range strings.Lines(logBuf.String()) {
		if strings.Contains(line, `"msg":"audit: record event failed"`) {
			failures = append(failures, line)
		}
	}
	if len(failures) != 4 {
		t.Fatalf("%d audit failure lines, want 4 (login ok, login failed, registration, logout):\n%s", len(failures), logBuf.String())
	}
	for _, line := range failures {
		if !strings.Contains(line, `"level":"ERROR"`) || !strings.Contains(line, `"event":`) {
			t.Errorf("audit failure line = %s, want level ERROR with the event", line)
		}
		for _, secret := range []string{flowPassword, cadastroPassword, "novo.usuario", cookie.Value, token} {
			if strings.Contains(line, secret) {
				t.Errorf("audit failure line holds %q: %s", secret, line)
			}
		}
	}
	if len(f.audit.snapshot()) != 0 {
		t.Error("events were stored while the log was down")
	}
}

// TestAuditNeverStoresSecrets covers 006 S-2: after logins (ok, unknown
// login, wrong password), a registration and a logout with known values, no
// audit row, no log line and no audit response holds a password, a session
// cookie, a CSRF token or the typed wrong login.
func TestAuditNeverStoresSecrets(t *testing.T) {
	t.Parallel()

	const (
		typedLogin    = "Typed-Login-Sentinel-q7w" // a password typed into the login field
		wrongPassword = "Wrong-Password-Sentinel-x9z"
	)
	var logBuf bytes.Buffer
	f := newCadastroApp(t, &logBuf, newMemUserRepo())
	var secrets []string
	remember := func(c *http.Cookie, token string) {
		if c != nil {
			secrets = append(secrets, c.Value)
		}
		secrets = append(secrets, token)
	}

	for _, attempt := range []struct{ login, password string }{
		{typedLogin, wrongPassword},
		{comumLogin, wrongPassword},
	} {
		anon, token := f.csrf(t, nil)
		remember(anon, token)
		if rr := f.send(t, http.MethodPost, "/api/login", loginBody(attempt.login, attempt.password), anon, csrfHeader(token)); rr.Code != http.StatusUnauthorized {
			t.Fatalf("failed login: %d", rr.Code)
		}
	}
	admin, token := f.loginAs(t, flowLogin)
	remember(admin, token)
	if rr := f.send(t, http.MethodPost, "/api/cadastro", cadastroBody("novo.usuario", cadastroPassword, "comum"), admin, csrfHeader(token)); rr.Code != http.StatusCreated {
		t.Fatalf("registration: %d", rr.Code)
	}
	listed := f.send(t, http.MethodGet, "/api/audit-events", "", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d", listed.Code)
	}
	if rr := f.send(t, http.MethodPost, "/api/logout", "", admin, csrfHeader(token)); rr.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rr.Code)
	}

	secrets = append(secrets, typedLogin, strings.ToLower(typedLogin), wrongPassword, flowPassword, cadastroPassword)
	events := f.audit.snapshot()
	if len(events) != 5 {
		t.Fatalf("%d events, want 5: %+v", len(events), events)
	}
	stored := fmt.Sprintf("%+v", events)
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if strings.Contains(stored, secret) {
			t.Errorf("an audit row holds %q: %s", secret, stored)
		}
		if strings.Contains(logBuf.String(), secret) {
			t.Errorf("a log line holds %q", secret)
		}
		if strings.Contains(listed.Body.String(), secret) {
			t.Errorf("GET /api/audit-events holds %q", secret)
		}
	}
}

// TestAuditStoresThePeerIP covers 006 S-4 over HTTP: the IP is the TCP peer,
// whatever X-Forwarded-For and X-Real-IP say, and a peer that does not parse
// is stored as no IP without failing the request.
func TestAuditStoresThePeerIP(t *testing.T) {
	t.Parallel()

	forged := map[string]string{
		"X-Forwarded-For": "198.51.100.66, 198.51.100.67",
		"X-Real-IP":       "198.51.100.77",
		"Forwarded":       "for=198.51.100.88",
	}
	tests := []struct {
		remoteAddr string
		wantIP     string // "" means none
	}{
		{"203.0.113.7:4444", "203.0.113.7"},
		{"[::1]:8443", "::1"},
		{"not-an-address", ""},
	}
	for _, tt := range tests {
		t.Run(tt.remoteAddr, func(t *testing.T) {
			t.Parallel()
			f := newFlowApp(t)
			anon, token := f.csrf(t, nil)
			headers := map[string]string{"X-CSRF-Token": token}
			for k, v := range forged {
				headers[k] = v
			}
			rr := f.sendFrom(t, tt.remoteAddr, http.MethodPost, "/api/login", loginBody(flowLogin, flowPassword), anon, headers)
			if rr.Code != http.StatusOK {
				t.Fatalf("login from %s: %d %s", tt.remoteAddr, rr.Code, rr.Body.String())
			}
			events := f.audit.snapshot()
			if len(events) != 1 {
				t.Fatalf("events = %+v, want one", events)
			}
			got := ""
			if events[0].IP.IsValid() {
				got = events[0].IP.String()
			}
			if got != tt.wantIP {
				t.Errorf("stored IP = %q, want %q (the TCP peer, never a header)", got, tt.wantIP)
			}

			admin := sessionCookie(rr)
			list := decodeInto[auditPageResponse](t, f.send(t, http.MethodGet, "/api/audit-events", "", admin, nil).Body)
			if len(list.Items) != 1 {
				t.Fatalf("listed %d events, want 1", len(list.Items))
			}
			switch ip := list.Items[0].IP; {
			case tt.wantIP == "" && ip != nil:
				t.Errorf("listed ip = %q, want null", *ip)
			case tt.wantIP != "" && (ip == nil || *ip != tt.wantIP):
				t.Errorf("listed ip = %v, want %q", ip, tt.wantIP)
			}
		})
	}
}

// TestAuditEventsList covers the answers of GET /api/audit-events (006 D-8):
// newest first, the type filter, the logins of actor and target, and 400
// for a bad type or page.
func TestAuditEventsList(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	anon, token := f.csrf(t, nil)
	f.send(t, http.MethodPost, "/api/login", loginBody(comumLogin, "a wrong long password"), anon, csrfHeader(token))
	admin, _ := f.loginAs(t, flowLogin)

	all := decodeInto[auditPageResponse](t, f.send(t, http.MethodGet, "/api/audit-events", "", admin, nil).Body)
	if all.Page != 1 || all.PageSize != 20 || all.HasNext || len(all.Items) != 2 {
		t.Fatalf("list = %+v", all)
	}
	first, second := all.Items[0], all.Items[1]
	if first.Event != domain.AuditLoginSucceeded || first.Actor == nil || *first.Actor != (auditActorResponse{UserID: 1, Login: flowLogin}) || first.Target != nil {
		t.Errorf("newest event = %+v, want admin's login_succeeded with no target", first)
	}
	if second.Event != domain.AuditLoginFailed || second.Actor != nil || second.Target == nil ||
		*second.Target != (auditTargetResponse{Type: domain.AuditTargetUser, ID: 2, Login: comumLogin}) {
		t.Errorf("older event = %+v, want a login_failed on maria with no actor", second)
	}

	failed := decodeInto[auditPageResponse](t, f.send(t, http.MethodGet, "/api/audit-events?type=login_failed", "", admin, nil).Body)
	if len(failed.Items) != 1 || failed.Items[0].Event != domain.AuditLoginFailed {
		t.Errorf("?type=login_failed = %+v", failed.Items)
	}
	rr := f.send(t, http.MethodGet, "/api/audit-events?type=post_removed&page=2", "", admin, nil)
	if !strings.Contains(rr.Body.String(), `"items":[]`) {
		t.Errorf("empty page = %s, want items []", rr.Body.String())
	}
	for _, q := range []string{"?type=bogus", "?type=LOGOUT", "?type=logout,login_failed", "?page=0", "?page=abc", "?page=10001"} {
		rr := f.send(t, http.MethodGet, "/api/audit-events"+q, "", admin, nil)
		if rr.Code != http.StatusBadRequest || decodeInto[errorResponse](t, rr.Body).Error != "bad_request" {
			t.Errorf("GET /api/audit-events%s: %d %s, want 400", q, rr.Code, rr.Body.String())
		}
	}
}

// TestModeration covers 006 S-6 and D-1 over HTTP: an admin removes a post
// with a reason; other comum users no longer see it; the author and the
// admins see it with removal; it is frozen; bad reasons are 422 and
// malformed bodies 400; the removal needs CSRF; and the admin still cannot
// create, edit or delete posts.
func TestModeration(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	f := newFlowAppWithLog(t, &logBuf)
	mariasPost := f.posts.seed(t, 2, "da maria")
	joaosPost := f.posts.seed(t, 3, "do joão")
	admin, adminToken := f.loginAs(t, flowLogin)
	maria, mariaToken := f.loginAs(t, comumLogin)
	joao, _ := f.loginAs(t, otherComumLogin)
	removalPath := postPath(mariasPost) + "/removal"

	before := f.posts.snapshot()
	for _, tt := range []struct {
		name, body string
		want       int
	}{
		{"blank reason", `{"reason":"   "}`, http.StatusUnprocessableEntity},
		{"missing reason", `{}`, http.StatusUnprocessableEntity},
		{"reason too long", fmt.Sprintf(`{"reason":%q}`, strings.Repeat("a", 501)), http.StatusUnprocessableEntity},
		{"NUL in reason", `{"reason":"spam\u0000"}`, http.StatusUnprocessableEntity},
		{"unknown field", `{"reason":"spam","post_id":2}`, http.StatusBadRequest},
		{"wrong type", `{"reason":1}`, http.StatusBadRequest},
		{"malformed json", `{"reason":`, http.StatusBadRequest},
	} {
		if rr := f.send(t, http.MethodPost, removalPath, tt.body, admin, csrfHeader(adminToken)); rr.Code != tt.want {
			t.Errorf("%s: status %d, want %d (%s)", tt.name, rr.Code, tt.want, rr.Body.String())
		}
	}
	for _, headers := range []map[string]string{nil, csrfHeader("not-the-token")} {
		logBuf.Reset()
		rr := f.send(t, http.MethodPost, removalPath, `{"reason":"spam"}`, admin, headers)
		if rr.Code != http.StatusForbidden || !strings.Contains(logBuf.String(), `"reason":"csrf: token mismatch"`) {
			t.Errorf("removal without a valid CSRF token: %d, want 403 from the CSRF check", rr.Code)
		}
	}
	if rr := f.send(t, http.MethodPost, removalPath, `{"reason":"spam"}`, maria, csrfHeader(mariaToken)); rr.Code != http.StatusForbidden {
		t.Errorf("removal by a comum user: %d, want 403", rr.Code)
	}
	assertUnchanged(t, f, before)
	for _, ev := range f.audit.snapshot() {
		if ev.Type == domain.AuditPostRemoved {
			t.Errorf("a rejected removal recorded %+v", ev)
		}
	}

	rr := f.send(t, http.MethodPost, removalPath, `{"reason":"  Conteúdo ofensivo.\nReincidente.  "}`, admin, csrfHeader(adminToken))
	if rr.Code != http.StatusNoContent || rr.Body.Len() != 0 {
		t.Fatalf("removal: %d %q", rr.Code, rr.Body.String())
	}
	removed := f.posts.snapshot()[mariasPost]
	if removed.Removal == nil || removed.Removal.Reason != "Conteúdo ofensivo.\nReincidente." || removed.Removal.RemovedBy != 1 {
		t.Errorf("stored removal = %+v, want the trimmed reason by admin 1", removed.Removal)
	}
	events := f.audit.snapshot()
	last := stripEvent(events[len(events)-1])
	want := domain.AuditEvent{Type: domain.AuditPostRemoved, ActorID: 1, TargetType: domain.AuditTargetPost, TargetID: int64(mariasPost), IP: mustAddr(t, "192.0.2.1")}
	if last != want {
		t.Errorf("last event = %+v, want %+v", last, want)
	}

	list := func(cookie *http.Cookie) map[domain.PostID]postResponse {
		t.Helper()
		rr := f.send(t, http.MethodGet, "/api/posts", "", cookie, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: %d", rr.Code)
		}
		out := map[domain.PostID]postResponse{}
		for _, p := range decodeInto[postPageResponse](t, rr.Body).Items {
			out[p.ID] = p
		}
		return out
	}
	if _, ok := list(joao)[mariasPost]; ok {
		t.Error("another comum user still sees the removed post")
	}
	if p := list(maria)[mariasPost]; !p.Mine || p.Removal == nil || p.Removal.Reason != "Conteúdo ofensivo.\nReincidente." || p.Removal.RemovedAt.IsZero() {
		t.Errorf("the author sees %+v, want the post with its removal", p)
	}
	adminView := list(admin)
	if p := adminView[mariasPost]; p.Mine || p.Removal == nil || p.Removal.Reason == "" {
		t.Errorf("the admin sees %+v, want the post marked as removed, with the reason", p)
	}
	if p, ok := adminView[joaosPost]; !ok || p.Mine || p.Removal != nil {
		t.Errorf("the admin sees %+v, %v; want joão's post, not removed", p, ok)
	}
	if !strings.Contains(f.send(t, http.MethodGet, "/api/posts", "", joao, nil).Body.String(), `"removal":null`) {
		t.Error(`a post that is not removed must have "removal":null`)
	}

	before = f.posts.snapshot()
	for _, tt := range []struct {
		name, method, path, body string
		cookie                   *http.Cookie
		token                    string
	}{
		{"author edits the removed post", http.MethodPut, postPath(mariasPost), postBody("t", "b"), maria, mariaToken},
		{"author deletes the removed post", http.MethodDelete, postPath(mariasPost), "", maria, mariaToken},
		{"second removal", http.MethodPost, removalPath, `{"reason":"de novo"}`, admin, adminToken},
		{"removal of a missing post", http.MethodPost, postPath(999) + "/removal", `{"reason":"x"}`, admin, adminToken},
		{"removal of a malformed id", http.MethodPost, "/api/posts/abc/removal", `{"reason":"x"}`, admin, adminToken},
	} {
		rr := f.send(t, tt.method, tt.path, tt.body, tt.cookie, csrfHeader(tt.token))
		got := decodeInto[errorResponse](t, rr.Body)
		if rr.Code != http.StatusNotFound || got.Message != msgPostNaoEncontrado {
			t.Errorf("%s: %d %+v, want the post 404", tt.name, rr.Code, got)
		}
	}
	for _, tt := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/posts", postBody("do admin", "b")},
		{http.MethodPut, postPath(joaosPost), postBody("t", "b")},
		{http.MethodDelete, postPath(joaosPost), ""},
	} {
		if rr := f.send(t, tt.method, tt.path, tt.body, admin, csrfHeader(adminToken)); rr.Code != http.StatusForbidden {
			t.Errorf("admin %s %s: %d, want 403 (006 D-1)", tt.method, tt.path, rr.Code)
		}
	}
	assertUnchanged(t, f, before)
}

// TestRemovalIsAtomicWithItsEvent covers 006 D-3 over HTTP: when the event
// cannot be written, the removal fails with 500 and the post is not removed.
func TestRemovalIsAtomicWithItsEvent(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	post := f.posts.seed(t, 2, "da maria")
	admin, token := f.loginAs(t, flowLogin)
	before := f.posts.snapshot()

	f.audit.setFail(true)
	rr := f.send(t, http.MethodPost, postPath(post)+"/removal", `{"reason":"spam"}`, admin, csrfHeader(token))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("removal with the log down: %d, want 500", rr.Code)
	}
	assertUnchanged(t, f, before)
}
