package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// memTicketRepo is an in-memory service.TicketRepository that behaves like
// the tickets table: the id, the status default and the dates are set here,
// never taken from the ticket given.
type memTicketRepo struct {
	mu      sync.Mutex
	tickets []domain.Ticket
	err     error // returned by Create when set
}

func newMemTicketRepo() *memTicketRepo { return &memTicketRepo{} }

func (m *memTicketRepo) Create(_ context.Context, t domain.Ticket) (domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.Ticket{}, m.err
	}
	now := time.Now().UTC()
	t.ID = domain.TicketID(len(m.tickets) + 1)
	t.Status = domain.TicketStatusOpen
	t.CreatedAt, t.UpdatedAt = now, now
	m.tickets = append(m.tickets, t)
	return t, nil
}

func (m *memTicketRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tickets)
}

func (m *memTicketRepo) all() []domain.Ticket {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.tickets)
}

func (m *memTicketRepo) fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// ticketBody marshals a ticket request, so values need no JSON escaping by
// hand.
func ticketBody(tipo string, dados map[string]any) string {
	b, err := json.Marshal(map[string]any{"tipo": tipo, "dados": dados})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func validAccess() map[string]any {
	return map[string]any{"sistema": "Grafana", "nivel_acesso": "leitura", "justificativa": "Preciso ver os painéis da equipe."}
}

func validBug() map[string]any {
	return map[string]any{ //nolint:gosec // "passos" (steps) is a ticket field, not a credential
		"sistema": "Portal do aluno", "descricao": "A página de notas não carrega.",
		"passos": "1. Entrar\n2. Abrir Notas", "severidade": "alta",
	}
}

func validDeploy() map[string]any {
	return map[string]any{"projeto": "api-notas", "repositorio": "https://github.com/org/api-notas", "ambiente": "prod"}
}

func validOther() map[string]any {
	return map[string]any{"titulo": "Dúvida sobre a VPN", "descricao": "Como peço acesso à VPN?"}
}

func withKey(dados map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(dados)+1)
	for k, v := range dados {
		out[k] = v
	}
	if value == nil {
		delete(out, key)
	} else {
		out[key] = value
	}
	return out
}

// postTicket sends POST /api/tickets as a signed-in comum user with the
// current CSRF token, plus headers.
func (f flowApp) postTicket(t *testing.T, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	cookie, token := f.loginAs(t, comumLogin)
	h := map[string]string{security.CSRFHeaderName: token}
	for k, v := range headers {
		h[k] = v
	}
	return f.send(t, http.MethodPost, "/api/tickets", body, cookie, h)
}

// checkErrorBody requires the generic error body: exactly error, message
// and request_id, with the given code and message.
func checkErrorBody(t *testing.T, rr *httptest.ResponseRecorder, wantError, wantMessage string) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v (%s)", err, rr.Body.String())
	}
	if got := sortedKeys(body); !slices.Equal(got, []string{"error", "message", "request_id"}) {
		t.Errorf("error body keys = %v, want [error message request_id]", got)
	}
	if body["error"] != wantError || body["message"] != wantMessage {
		t.Errorf("error body = %v, want error %q and message %q", body, wantError, wantMessage)
	}
	if body["request_id"] != rr.Header().Get("X-Request-Id") || body["request_id"] == "" {
		t.Errorf("request_id = %v, want the X-Request-Id header %q", body["request_id"], rr.Header().Get("X-Request-Id"))
	}
}

// TestCreateTicketPerType covers AC-5.2 (server side), BR-1, BR-2, BR-3 and
// BR-4: one ticket of each type is created with status aberto, the session
// user as author, the built title and trimmed values, and the response
// holds only what the confirmation shows.
func TestCreateTicketPerType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantType  domain.TicketType
		wantTitle string
		check     func(t *testing.T, stored domain.Ticket)
	}{
		{
			"acesso", ticketBody("acesso", withKey(validAccess(), "sistema", "  Grafana  ")), domain.TicketTypeAccess, "Acesso a Grafana",
			func(t *testing.T, s domain.Ticket) {
				if s.System != "Grafana" || s.AccessLevel != domain.AccessLevelRead || s.Justification != "Preciso ver os painéis da equipe." {
					t.Errorf("stored = %+v", s)
				}
			},
		},
		{
			"bug", ticketBody("bug", validBug()), domain.TicketTypeBug, "Bug em Portal do aluno",
			func(t *testing.T, s domain.Ticket) {
				if s.Steps != "1. Entrar\n2. Abrir Notas" || s.Severity != domain.SeverityHigh {
					t.Errorf("stored = %+v (line breaks must be kept)", s)
				}
			},
		},
		{
			"deploy without notes", ticketBody("deploy", validDeploy()), domain.TicketTypeDeploy, "Deploy de api-notas em Produção",
			func(t *testing.T, s domain.Ticket) {
				if s.Repository != "https://github.com/org/api-notas" || s.Environment != domain.EnvironmentProduction || s.Notes != "" {
					t.Errorf("stored = %+v", s)
				}
			},
		},
		{
			"deploy with notes", ticketBody("deploy", withKey(withKey(validDeploy(), "ambiente", "homolog"), "observacoes", " Depois das 18h. ")),
			domain.TicketTypeDeploy, "Deploy de api-notas em Homologação",
			func(t *testing.T, s domain.Ticket) {
				if s.Notes != "Depois das 18h." {
					t.Errorf("notes = %q", s.Notes)
				}
			},
		},
		{
			"deploy with null notes", `{"tipo":"deploy","dados":{"projeto":"api-notas","repositorio":"https://github.com/org/api-notas","ambiente":"dev","observacoes":null}}`,
			domain.TicketTypeDeploy, "Deploy de api-notas em Desenvolvimento",
			func(t *testing.T, s domain.Ticket) {
				if s.Notes != "" {
					t.Errorf("notes = %q, want empty", s.Notes)
				}
			},
		},
		{
			"outro with HTML kept as typed", ticketBody("outro", withKey(validOther(), "titulo", `<script>alert(1)</script>`)),
			domain.TicketTypeOther, `<script>alert(1)</script>`,
			func(t *testing.T, s domain.Ticket) {
				if s.Description != "Como peço acesso à VPN?" {
					t.Errorf("stored = %+v", s)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFlowApp(t)
			before := time.Now()
			rr := f.postTicket(t, tt.body, nil)
			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
			}

			var raw map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			if got := sortedKeys(raw); !slices.Equal(got, []string{"created_at", "id", "status", "title", "type", "updated_at"}) {
				t.Errorf("response keys = %v: it must hold only what the confirmation shows", got)
			}
			var got ticketResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.ID != 1 || got.Type != tt.wantType || got.Title != tt.wantTitle || got.Status != domain.TicketStatusOpen {
				t.Errorf("response = %+v, want id 1, type %s, title %q, status aberto", got, tt.wantType, tt.wantTitle)
			}
			if !got.CreatedAt.Equal(got.UpdatedAt) || got.CreatedAt.Before(before.Add(-time.Second)) {
				t.Errorf("dates = %v / %v, want equal server times", got.CreatedAt, got.UpdatedAt)
			}

			stored := f.tickets.all()
			if len(stored) != 1 {
				t.Fatalf("%d tickets stored, want 1", len(stored))
			}
			if stored[0].AuthorID != 2 {
				t.Errorf("author = %d, want the session user 2", stored[0].AuthorID)
			}
			tt.check(t, stored[0])
		})
	}
}

// TestCreateTicketAuthorFollowsSession covers S-4: the author is the user of
// the session. User 1 is made comum, so two different comum users create a
// ticket each.
func TestCreateTicketAuthorFollowsSession(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	f.sessions.setRole(1, domain.RoleComum)

	for _, login := range []string{comumLogin, flowLogin} {
		cookie, token := f.loginAs(t, login)
		rr := f.send(t, http.MethodPost, "/api/tickets", ticketBody("outro", validOther()), cookie,
			map[string]string{security.CSRFHeaderName: token})
		if rr.Code != http.StatusCreated {
			t.Fatalf("%s: status = %d (%s)", login, rr.Code, rr.Body.String())
		}
	}
	stored := f.tickets.all()
	if len(stored) != 2 || stored[0].AuthorID != 2 || stored[1].AuthorID != 1 {
		t.Errorf("authors = %+v, want 2 then 1", stored)
	}
}

// TestCreateTicketRefusals covers S-1, S-4, S-5, AC-6.1 and AC-6.6: every
// class of invalid request gets the generic 400 or 422 body, never naming a
// field, and nothing is stored.
func TestCreateTicketRefusals(t *testing.T) {
	t.Parallel()

	invalidByte := string([]byte{0xff})
	access := `"tipo":"acesso","dados":{"sistema":"Grafana","nivel_acesso":"leitura","justificativa":"Preciso ver os painéis."`
	bad := http.StatusBadRequest
	invalid := http.StatusUnprocessableEntity
	tests := []struct {
		name    string
		body    string
		headers map[string]string
		status  int
	}{
		// 400: the body is not well-formed for the operation.
		{"wrong content type", ticketBody("outro", validOther()), map[string]string{"Content-Type": "text/plain"}, bad},
		{"malformed json", `{"tipo":"outro","dados":{`, nil, bad},
		{"empty body", ``, map[string]string{"Content-Type": "application/json"}, bad},
		{"trailing data", ticketBody("outro", validOther()) + `{}`, nil, bad},
		{"not valid utf-8", `{"tipo":"outro","dados":{"titulo":"Dúvida ` + invalidByte + `","descricao":"Como peço acesso à VPN?"}}`, nil, bad},
		{"unknown outer key", `{` + access + `},"prioridade":"alta"}`, nil, bad},
		{"unknown key in dados", `{` + access + `,"prioridade":"alta"}}`, nil, bad},
		{"field of another type", ticketBody("outro", withKey(validOther(), "severidade", "alta")), nil, bad},
		{"notes in an access ticket", ticketBody("acesso", withKey(validAccess(), "observacoes", "x")), nil, bad},
		{"wrong JSON type", ticketBody("acesso", withKey(validAccess(), "sistema", 1)), nil, bad},
		{"tipo is a number", `{"tipo":1,"dados":{}}`, nil, bad},
		{"dados missing", `{"tipo":"outro"}`, nil, bad},
		{"dados null", `{"tipo":"outro","dados":null}`, nil, bad},
		{"dados array", `{"tipo":"outro","dados":[]}`, nil, bad},
		{"outer key in another case", `{"TIPO":"outro","dados":{"titulo":"Dúvida","descricao":"Como peço acesso à VPN?"}}`, nil, bad},
		{"dados key in another case", `{"tipo":"outro","dados":{"Titulo":"Dúvida","descricao":"Como peço acesso à VPN?"}}`, nil, bad},
		{"duplicate key", `{"tipo":"acesso","tipo":"outro","dados":{"titulo":"Dúvida","descricao":"Como peço acesso à VPN?"}}`, nil, bad},
		{"duplicate key in another case", `{` + access + `,"SISTEMA":"Outro"}}`, nil, bad},
		{"body over 64 KiB", ticketBody("outro", withKey(validOther(), "descricao", strings.Repeat("a", security.MaxBodyBytes))), nil, bad},

		// 400: keys the server owns (S-4, S-5), outside and inside dados.
		{"author outside", `{` + access + `},"author":"maria"}`, nil, bad},
		{"user_id outside", `{` + access + `},"user_id":1}`, nil, bad},
		{"author_id inside", `{` + access + `,"author_id":1}}`, nil, bad},
		{"status outside", `{` + access + `},"status":"fechado"}`, nil, bad},
		{"status inside", `{` + access + `,"status":"fechado"}}`, nil, bad},
		{"id outside", `{` + access + `},"id":99}`, nil, bad},
		{"id inside", `{` + access + `,"id":99}}`, nil, bad},
		{"created_at outside", `{` + access + `},"created_at":"2020-01-01T00:00:00Z"}`, nil, bad},
		{"updated_at inside", `{` + access + `,"updated_at":"2020-01-01T00:00:00Z"}}`, nil, bad},

		// 422: well-formed, but a value breaks a rule of spec section 4.
		{"unknown tipo", `{"tipo":"incidente","dados":{"titulo":"Dúvida"}}`, nil, invalid},
		{"tipo missing", `{"dados":{"titulo":"Dúvida","descricao":"Como peço acesso à VPN?"}}`, nil, invalid},
		{"required key missing", ticketBody("outro", withKey(validOther(), "descricao", nil)), nil, invalid},
		{"required field empty", ticketBody("outro", withKey(validOther(), "titulo", "")), nil, invalid},
		{"spaces only", ticketBody("outro", withKey(validOther(), "titulo", "   \t ")), nil, invalid},
		{"single-line too short", ticketBody("acesso", withKey(validAccess(), "sistema", "G")), nil, invalid},
		{"single-line too long", ticketBody("outro", withKey(validOther(), "titulo", strings.Repeat("é", 121))), nil, invalid},
		{"multi-line too short", ticketBody("bug", withKey(validBug(), "passos", "Abrir")), nil, invalid},
		{"multi-line too long", ticketBody("bug", withKey(validBug(), "descricao", strings.Repeat("é", 2001))), nil, invalid},
		{"notes too long", ticketBody("deploy", withKey(validDeploy(), "observacoes", strings.Repeat("a", 2001))), nil, invalid},
		{"line break in a single-line field", ticketBody("outro", withKey(validOther(), "titulo", "Dúvida\nsobre a VPN")), nil, invalid},
		{"tab in a single-line field", ticketBody("bug", withKey(validBug(), "sistema", "Portal\tdo aluno")), nil, invalid},
		{"NUL in a multi-line field", ticketBody("outro", withKey(validOther(), "descricao", "Como peço\x00 acesso à VPN?")), nil, invalid},
		{"bell in a multi-line field", ticketBody("acesso", withKey(validAccess(), "justificativa", "Preciso ver\x07 os painéis.")), nil, invalid},
		{"access level outside its list", ticketBody("acesso", withKey(validAccess(), "nivel_acesso", "root")), nil, invalid},
		{"severity in another case", ticketBody("bug", withKey(validBug(), "severidade", "Alta")), nil, invalid},
		{"environment label instead of code", ticketBody("deploy", withKey(validDeploy(), "ambiente", "Produção")), nil, invalid},
		{"http repository", ticketBody("deploy", withKey(validDeploy(), "repositorio", "http://github.com/org/api")), nil, invalid},
		{"javascript repository", ticketBody("deploy", withKey(validDeploy(), "repositorio", "javascript:alert(1)")), nil, invalid},
		{"ssh repository", ticketBody("deploy", withKey(validDeploy(), "repositorio", "git@github.com:org/api.git")), nil, invalid},
		{"repository without host", ticketBody("deploy", withKey(validDeploy(), "repositorio", "https:///org/api")), nil, invalid},
		{"repository with credentials", ticketBody("deploy", withKey(validDeploy(), "repositorio", "https://user:pass@github.com/org/api")), nil, invalid},
		{"repository over 300", ticketBody("deploy", withKey(validDeploy(), "repositorio", "https://github.com/"+strings.Repeat("a", 282))), nil, invalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFlowApp(t)
			rr := f.postTicket(t, tt.body, tt.headers)
			if rr.Code != tt.status {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.status, rr.Body.String())
			}
			if tt.status == bad {
				checkErrorBody(t, rr, "bad_request", defaultMessage(http.StatusBadRequest))
			} else {
				checkErrorBody(t, rr, "unprocessable_entity", msgDadosInvalidos)
			}
			if n := f.tickets.count(); n != 0 {
				t.Errorf("%d tickets stored, want 0", n)
			}
		})
	}
}

// TestCreateTicketCSRF covers S-3: no token, a wrong token, a cross-site
// browser request and no session at all get 403, and nothing is stored.
func TestCreateTicketCSRF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		session bool
		headers func(token string) map[string]string
	}{
		{"no token", true, func(string) map[string]string { return nil }},
		{"wrong token", true, func(string) map[string]string {
			return map[string]string{security.CSRFHeaderName: "AAAAAAAAAAAAAAAAAAAAAAAAAA"}
		}},
		{"cross-site request", true, func(tok string) map[string]string {
			return map[string]string{security.CSRFHeaderName: tok, "Sec-Fetch-Site": "cross-site"}
		}},
		{"other origin without fetch metadata", true, func(tok string) map[string]string {
			return map[string]string{security.CSRFHeaderName: tok, "Origin": "https://evil.example"}
		}},
		{"no session", false, func(tok string) map[string]string { return map[string]string{security.CSRFHeaderName: tok} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var logBuf bytes.Buffer
			f := newFlowAppWithLog(t, &logBuf)
			cookie, token := f.loginAs(t, comumLogin)
			if !tt.session {
				cookie = nil
			}
			rr := f.send(t, http.MethodPost, "/api/tickets", ticketBody("outro", validOther()), cookie, tt.headers(token))
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (%s)", rr.Code, rr.Body.String())
			}
			checkErrorBody(t, rr, "forbidden", defaultMessage(http.StatusForbidden))
			if !strings.Contains(logBuf.String(), `"reason":"csrf:`) {
				t.Errorf("the CSRF check did not refuse the request:\n%s", logBuf.String())
			}
			if n := f.tickets.count(); n != 0 {
				t.Errorf("%d tickets stored, want 0", n)
			}
		})
	}
}

// TestCreateTicketServerError covers AC-6.4 on the server: a storage failure
// is a 500 with the generic body and the request id, and nothing leaks.
func TestCreateTicketServerError(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	f := newFlowAppWithLog(t, &logBuf)
	f.tickets.fail(errors.New("insert failed: connection reset"))

	rr := f.postTicket(t, ticketBody("outro", validOther()), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (%s)", rr.Code, rr.Body.String())
	}
	checkErrorBody(t, rr, "internal", defaultMessage(http.StatusInternalServerError))
	if strings.Contains(rr.Body.String(), "connection reset") {
		t.Errorf("response leaks the error: %s", rr.Body.String())
	}
	if !strings.Contains(logBuf.String(), "connection reset") {
		t.Errorf("the error is not in the log:\n%s", logBuf.String())
	}
}

// TestCreateTicketLogsNoFieldValues covers S-7: the log of a valid, an
// invalid (422) and a malformed (400) creation holds ids, the type and fixed
// reasons, never a submitted value.
func TestCreateTicketLogsNoFieldValues(t *testing.T) {
	t.Parallel()

	const sentinel = "Sentinela-Sigilosa-4d1e"
	var logBuf bytes.Buffer
	f := newFlowAppWithLog(t, &logBuf)

	requests := []struct {
		body   string
		status int
	}{
		{ticketBody("acesso", withKey(withKey(validAccess(), "sistema", "Grafana "+sentinel), "justificativa", "Preciso de "+sentinel)), http.StatusCreated},
		{ticketBody("outro", withKey(validOther(), "titulo", sentinel+"\nsegunda linha")), http.StatusUnprocessableEntity},
		{ticketBody("acesso", withKey(validAccess(), sentinel, "x")), http.StatusBadRequest},
		{`{"tipo":"` + sentinel + `","dados":{}}`, http.StatusUnprocessableEntity},
	}
	for _, req := range requests {
		if rr := f.postTicket(t, req.body, nil); rr.Code != req.status {
			t.Fatalf("status = %d, want %d (%s)", rr.Code, req.status, rr.Body.String())
		}
	}

	logged := logBuf.String()
	if strings.Contains(logged, sentinel) {
		t.Errorf("log contains a submitted value:\n%s", logged)
	}
	var created, refused int
	for line := range strings.Lines(logged) {
		switch {
		case strings.Contains(line, `"msg":"ticket created"`):
			created++
			for _, want := range []string{`"ticket_id":1`, `"author_id":2`, `"type":"acesso"`} {
				if !strings.Contains(line, want) {
					t.Errorf("ticket created line has no %s: %s", want, line)
				}
			}
		case strings.Contains(line, `"msg":"ticket refused"`):
			refused++
			if !strings.Contains(line, `"author_id":2`) || !strings.Contains(line, `"reason":`) {
				t.Errorf("ticket refused line = %s, want author_id and reason", line)
			}
		}
	}
	if created != 1 || refused != 3 {
		t.Errorf("logged %d created and %d refused, want 1 and 3:\n%s", created, refused, logged)
	}
}

// TestCreateTicketLargestValid sends the largest valid ticket: a bug with
// 120 + 2000 + 2000 characters, each an emoji written as a JSON escape pair
// (12 bytes, the most a client can spend per character). It fits the 64 KiB
// limit and is created.
func TestCreateTicketLargestValid(t *testing.T) {
	t.Parallel()

	const emoji = `😀`
	body := `{"tipo":"bug","dados":{"sistema":"` + strings.Repeat(emoji, 120) +
		`","descricao":"` + strings.Repeat(emoji, 2000) +
		`","passos":"` + strings.Repeat(emoji, 2000) +
		`","severidade":"critica"}}`
	if len(body) > security.MaxBodyBytes {
		t.Fatalf("body has %d bytes, over the %d limit: the test is wrong", len(body), security.MaxBodyBytes)
	}

	f := newFlowApp(t)
	rr := f.postTicket(t, body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
	stored := f.tickets.all()
	if len(stored) != 1 || len([]rune(stored[0].Description)) != 2000 || len([]rune(stored[0].Title)) != 120 {
		t.Errorf("stored %d tickets; want one with a 2000-character description and a 120-character title", len(stored))
	}
}
