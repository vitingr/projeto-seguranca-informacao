package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

const (
	msgForbidden = "Você não tem permissão para acessar este recurso."
	msgTooMany   = "Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente."
)

var (
	messageEntryKeys = []string{"created_at", "id", "kind", "mine", "side", "text"}
	statusEntryKeys  = []string{"created_at", "from", "id", "kind", "mine", "side", "to"}
)

// entriesOf returns the entries of a thread answer.
func entriesOf(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, ok := body["entries"].([]any)
	if !ok {
		t.Fatalf("entries = %v, want a JSON array", body["entries"])
	}
	out := make([]map[string]any, 0, len(raw))
	for _, e := range raw {
		out = append(out, e.(map[string]any))
	}
	return out
}

// addMessage stores a message directly, as if a handler had.
func (a trackingApp) addMessage(t *testing.T, ticket domain.TicketID, actor domain.UserID, text string) domain.TicketEntry {
	t.Helper()
	e, err := a.store.AddMessage(t.Context(), ticket, actor, text)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// TestOwnThread covers AC-3.1, AC-3.2 (data), P-11 and S-13 on R3: entries
// oldest first, the author's on side "author" and marked mine, the admins'
// on side "support", and no login or id of anyone.
func TestOwnThread(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeBug, "Bug em Portal")
	other := a.store.seed(trackMariaID, domain.TicketTypeOther, "Outro")
	first := a.addMessage(t, ticket.ID, trackMariaID, "Alguma <b>novidade</b>?")
	change, err := a.store.ChangeStatus(t.Context(), ticket.ID, trackAdminID, domain.TicketStatusOpen, domain.TicketStatusInProgress)
	if err != nil {
		t.Fatal(err)
	}
	reply := a.addMessage(t, ticket.ID, trackAnaID, "Estamos vendo.\nRetorno em breve.")
	a.addMessage(t, other.ID, trackMariaID, "De outro ticket")

	cookie, _ := a.loginAs(t, comumLogin)
	rr := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), "", cookie, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
	for _, leak := range []string{"actor", "login", flowLogin, anaLogin, `"author_id"`, `"user_id"`} {
		if strings.Contains(rr.Body.String(), leak) {
			t.Errorf("answer holds %q: %s", leak, rr.Body.String())
		}
	}

	_, body := a.get(t, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), cookie)
	if got := keysOf(body); !slices.Equal(got, []string{"entries", "has_more"}) || body["has_more"] != false {
		t.Errorf("page = %v, want entries and has_more false", body)
	}
	entries := entriesOf(t, body)
	if len(entries) != 3 {
		t.Fatalf("%d entries, want 3 (none of the other ticket)", len(entries))
	}
	want := []map[string]any{
		{"id": float64(first.ID), "kind": "message", "side": "author", "mine": true, "text": "Alguma <b>novidade</b>?"},
		{"id": float64(change.ID), "kind": "status_change", "side": "support", "mine": false, "from": "aberto", "to": "em_andamento"},
		{"id": float64(reply.ID), "kind": "message", "side": "support", "mine": false, "text": "Estamos vendo.\nRetorno em breve."},
	}
	for i, e := range entries {
		keys := messageEntryKeys
		if e["kind"] == "status_change" {
			keys = statusEntryKeys
		}
		if got := keysOf(e); !slices.Equal(got, keys) {
			t.Errorf("entry %d keys = %v, want %v", i, got, keys)
		}
		for k, v := range want[i] {
			if e[k] != v {
				t.Errorf("entry %d: %s = %#v, want %#v", i, k, e[k], v)
			}
		}
	}
}

// TestAdminThread covers AC-5.3 (data) and P-11 on R8: every entry adds
// actor_login; mine is true only for the requesting admin; the author's
// entries are on side "author"; antes and the 100 limit as R3.
func TestAdminThread(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeBug, "Bug em Portal")
	a.addMessage(t, ticket.ID, trackMariaID, "Alguma novidade?")
	if _, err := a.store.ChangeStatus(t.Context(), ticket.ID, trackAnaID, domain.TicketStatusOpen, domain.TicketStatusInProgress); err != nil {
		t.Fatal(err)
	}
	a.addMessage(t, ticket.ID, trackAdminID, "Estamos vendo.")
	path := fmt.Sprintf("/api/administradores/tickets/%d/mensagens", ticket.ID)

	cookie, _ := a.loginAs(t, flowLogin)
	code, body := a.get(t, path, cookie)
	if code != http.StatusOK || body["has_more"] != false {
		t.Fatalf("status = %d, page = %v", code, body)
	}
	want := []map[string]any{
		{"kind": "message", "side": "author", "mine": false, "actor_login": comumLogin},
		{"kind": "status_change", "side": "support", "mine": false, "actor_login": anaLogin, "from": "aberto", "to": "em_andamento"},
		{"kind": "message", "side": "support", "mine": true, "actor_login": flowLogin},
	}
	entries := entriesOf(t, body)
	if len(entries) != len(want) {
		t.Fatalf("%d entries, want %d", len(entries), len(want))
	}
	for i, e := range entries {
		keys := append([]string{"actor_login"}, messageEntryKeys...)
		if e["kind"] == "status_change" {
			keys = append([]string{"actor_login"}, statusEntryKeys...)
		}
		slices.Sort(keys)
		if got := keysOf(e); !slices.Equal(got, keys) {
			t.Errorf("entry %d keys = %v, want %v", i, got, keys)
		}
		for k, v := range want[i] {
			if e[k] != v {
				t.Errorf("entry %d: %s = %#v, want %#v", i, k, e[k], v)
			}
		}
	}

	// ana sees the status change as hers.
	_, body = a.get(t, path, a.mustLogin(t, anaLogin))
	if mine := entriesOf(t, body)[1]["mine"]; mine != true {
		t.Errorf("ana's status change: mine = %v, want true", mine)
	}

	for range 120 {
		a.addMessage(t, ticket.ID, trackMariaID, "mais uma")
	}
	_, body = a.get(t, path, cookie)
	page := entriesOf(t, body)
	if len(page) != 100 || body["has_more"] != true {
		t.Fatalf("%d entries, has_more %v; want 100, true", len(page), body["has_more"])
	}
	_, body = a.get(t, fmt.Sprintf("%s?antes=%v", path, page[0]["id"]), cookie)
	if older := entriesOf(t, body); len(older) != 23 || body["has_more"] != false {
		t.Errorf("older page: %d entries, has_more %v; want 23, false", len(older), body["has_more"])
	}
	if rr := a.send(t, http.MethodGet, path+"?antes=abc", "", cookie, nil); rr.Code != http.StatusBadRequest {
		t.Errorf("bad antes: status = %d, want 400", rr.Code)
	}
	if rr := a.send(t, http.MethodGet, path, "", a.mustLogin(t, comumLogin), nil); rr.Code != http.StatusForbidden {
		t.Errorf("the author on the admin route: status = %d, want 403", rr.Code)
	}
}

// TestOwnThreadPages covers P-4 and S-14: at most 100 entries, the latest
// ones first answered, older ones with antes, has_more on the first page
// only.
func TestOwnThreadPages(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	var ids []float64
	for i := range 150 {
		ids = append(ids, float64(a.addMessage(t, ticket.ID, trackMariaID, fmt.Sprintf("mensagem %d", i)).ID))
	}
	cookie, _ := a.loginAs(t, comumLogin)

	_, body := a.get(t, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), cookie)
	latest := entriesOf(t, body)
	if len(latest) != 100 || body["has_more"] != true {
		t.Fatalf("first page: %d entries, has_more %v; want 100, true", len(latest), body["has_more"])
	}
	if latest[0]["id"] != ids[50] || latest[99]["id"] != ids[149] {
		t.Errorf("first page = ids %v to %v, want %v to %v (the latest, oldest first)", latest[0]["id"], latest[99]["id"], ids[50], ids[149])
	}

	_, body = a.get(t, fmt.Sprintf("/api/tickets/%d/mensagens?antes=%v", ticket.ID, latest[0]["id"]), cookie)
	older := entriesOf(t, body)
	if len(older) != 50 || body["has_more"] != false {
		t.Fatalf("older page: %d entries, has_more %v; want 50, false", len(older), body["has_more"])
	}
	if older[0]["id"] != ids[0] || older[49]["id"] != ids[49] {
		t.Errorf("older page = ids %v to %v, want %v to %v", older[0]["id"], older[49]["id"], ids[0], ids[49])
	}

	// No entry yet: an empty page, never null.
	empty := a.store.seed(trackMariaID, domain.TicketTypeOther, "Sem mensagens")
	rr := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d/mensagens", empty.ID), "", cookie, nil)
	if got := rr.Body.String(); got != `{"entries":[],"has_more":false}` {
		t.Errorf("empty thread = %s", got)
	}
}

// replyAs sends a reply as login on ticket with a valid CSRF token, plus
// headers (which may replace the token).
func (a trackingApp) replyAs(t *testing.T, login, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	cookie, token := a.loginAs(t, login)
	h := map[string]string{security.CSRFHeaderName: token}
	maps.Copy(h, headers)
	return a.send(t, http.MethodPost, path, body, cookie, h)
}

func replyBody(text string) string {
	b, _ := json.Marshal(map[string]string{"texto": text})
	return string(b)
}

// TestReplyOwnTicket covers AC-3.6 and BR-6, BR-8, S-8 on R4: 201 with the
// stored entry, trimmed, the session user as actor, the last update moved
// and the status unchanged.
func TestReplyOwnTicket(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeBug, "Bug em Portal")
	a.store.setStatus(ticket.ID, domain.TicketStatusResolved)
	path := fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID)

	rr := a.replyAs(t, comumLogin, path, replyBody("  O erro voltou.\nHoje cedo.  "), nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if got := keysOf(body); !slices.Equal(got, messageEntryKeys) {
		t.Errorf("keys = %v, want %v", got, messageEntryKeys)
	}
	if body["kind"] != "message" || body["side"] != "author" || body["mine"] != true || body["text"] != "O erro voltou.\nHoje cedo." {
		t.Errorf("entry = %v", body)
	}

	after := a.store.ticket(ticket.ID)
	if after.Status != domain.TicketStatusResolved {
		t.Errorf("status = %s, want resolvido (a reply never changes it, Q-5)", after.Status)
	}
	if !after.UpdatedAt.After(ticket.UpdatedAt) {
		t.Errorf("updated_at = %v, want later than %v (BR-8)", after.UpdatedAt, ticket.UpdatedAt)
	}
	_, thread := a.get(t, path, a.mustLogin(t, comumLogin))
	entries := entriesOf(t, thread)
	if len(entries) != 1 || entries[0]["id"] != body["id"] {
		t.Fatalf("thread = %v, want the new message", entries)
	}
	if stored := a.store.entry(domain.TicketEntryID(body["id"].(float64))); stored.ActorID != trackMariaID {
		t.Errorf("actor = %d, want the session user %d", stored.ActorID, trackMariaID)
	}
}

// mustLogin logs in and returns the session cookie only.
func (a trackingApp) mustLogin(t *testing.T, login string) *http.Cookie {
	t.Helper()
	cookie, _ := a.loginAs(t, login)
	return cookie
}

// TestReplyRefusals covers AC-3.5, AC-3.7, AC-3.8 (server), S-5, S-6, S-8
// and BR-5, BR-10 on R4: every refused reply writes nothing and leaves the
// ticket as it was.
func TestReplyRefusals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		headers    map[string]string
		closed     bool
		wantStatus int
		wantError  string
		wantMsg    string
	}{
		{name: "not JSON", body: `texto=oi`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "empty object", body: `{}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "null texto", body: `{"texto":null}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "texto of another type", body: `{"texto":1}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "key in another case", body: `{"Texto":"oi"}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "duplicate key", body: `{"texto":"oi","texto":"tchau"}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "author", body: `{"texto":"oi","author":1}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "user_id", body: `{"texto":"oi","user_id":1}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "created_at", body: `{"texto":"oi","created_at":"2026-01-01T00:00:00Z"}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "id", body: `{"texto":"oi","id":7}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "ticket_id", body: `{"texto":"oi","ticket_id":2}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "status", body: `{"texto":"oi","status":"fechado"}`, wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "invalid UTF-8", body: "{\"texto\":\"\xff\"}", wantStatus: http.StatusBadRequest, wantError: "bad_request", wantMsg: msgBadRequest},
		{name: "empty text", body: replyBody(""), wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMsg: msgDadosInvalidos},
		{name: "spaces only", body: replyBody(" \n\t "), wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMsg: msgDadosInvalidos},
		{name: "2001 characters", body: replyBody(strings.Repeat("é", 2001)), wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMsg: msgDadosInvalidos},
		{name: "NUL", body: replyBody("oi\x00"), wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMsg: msgDadosInvalidos},
		{name: "escape character", body: replyBody("oi\x1b[31m"), wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMsg: msgDadosInvalidos},
		{name: "closed ticket", body: replyBody("Ainda preciso de ajuda."), closed: true, wantStatus: http.StatusConflict, wantError: "conflict", wantMsg: msgTicketClosed},
		{name: "no CSRF token", body: replyBody("oi"), headers: map[string]string{security.CSRFHeaderName: ""}, wantStatus: http.StatusForbidden, wantError: "forbidden", wantMsg: msgForbidden},
		{name: "wrong CSRF token", body: replyBody("oi"), headers: map[string]string{security.CSRFHeaderName: "AAAAAAAAAAAAAAAAAAAAAAAAAA"}, wantStatus: http.StatusForbidden, wantError: "forbidden", wantMsg: msgForbidden},
		{name: "cross-site", body: replyBody("oi"), headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, wantStatus: http.StatusForbidden, wantError: "forbidden", wantMsg: msgForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newTrackingApp(t, nil)
			ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
			if tt.closed {
				a.store.setStatus(ticket.ID, domain.TicketStatusClosed)
			}
			before := a.store.ticket(ticket.ID)
			_, entries := a.store.sizes()

			rr := a.replyAs(t, comumLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), tt.body, tt.headers)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			checkErrorBody(t, rr, tt.wantError, tt.wantMsg)
			if _, got := a.store.sizes(); got != entries {
				t.Errorf("%d entries, want %d (nothing stored)", got, entries)
			}
			if after := a.store.ticket(ticket.ID); after.Status != before.Status || !after.UpdatedAt.Equal(before.UpdatedAt) {
				t.Errorf("ticket changed: %+v, was %+v", after, before)
			}
		})
	}
}

// TestReplyLongestMessage is the limit itself: 2000 code points of a
// multi-byte character are accepted (S-6, BR-6).
func TestReplyLongestMessage(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	rr := a.replyAs(t, comumLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), replyBody(strings.Repeat("é", 2000)), nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
}

// TestReplyRateLimit covers S-9 and P-16 on R4: 30 replies a minute per
// user, the 31st is a 429 with Retry-After and is not stored; another user
// keeps their quota; requests refused by CSRF or by the role spend none;
// requests on someone else's ticket do.
func TestReplyRateLimit(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	ticketB := a.store.seed(trackJoaoID, domain.TicketTypeOther, "Dúvida de joao")
	path := fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID)
	cookie, token := a.loginAs(t, comumLogin)
	headers := map[string]string{security.CSRFHeaderName: token}

	for range 5 { // refused by CSRF: no quota spent
		if rr := a.send(t, http.MethodPost, path, replyBody("oi"), cookie, nil); rr.Code != http.StatusForbidden {
			t.Fatalf("no token: status = %d, want 403", rr.Code)
		}
	}
	for i := 1; i <= security.MessageCreateLimit; i++ {
		if rr := a.send(t, http.MethodPost, path, replyBody(fmt.Sprintf("mensagem %d", i)), cookie, headers); rr.Code != http.StatusCreated {
			t.Fatalf("reply %d: status = %d, want 201 (%s)", i, rr.Code, rr.Body.String())
		}
	}
	rr := a.send(t, http.MethodPost, path, replyBody("mensagem 31"), cookie, headers)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("reply 31: status = %d, want 429", rr.Code)
	}
	checkErrorBody(t, rr, "too_many_requests", msgTooMany)
	if retry, err := strconv.Atoi(rr.Header().Get("Retry-After")); err != nil || retry < 1 || retry > 60 {
		t.Errorf("Retry-After = %q, want 1 to 60 seconds", rr.Header().Get("Retry-After"))
	}
	if _, entries := a.store.sizes(); entries != security.MessageCreateLimit {
		t.Errorf("%d entries, want %d (the 31st is not stored)", entries, security.MessageCreateLimit)
	}

	// Another user keeps their quota.
	if rr := a.replyAs(t, joaoLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticketB.ID), replyBody("oi"), nil); rr.Code != http.StatusCreated {
		t.Errorf("another user: status = %d, want 201", rr.Code)
	}

	// The role check runs before the limiter: an admin refused on the
	// comum route spends nothing.
	cookieAdmin, tokenAdmin := a.loginAs(t, flowLogin)
	for range security.MessageCreateLimit + 5 {
		if rr := a.send(t, http.MethodPost, path, replyBody("oi"), cookieAdmin, map[string]string{security.CSRFHeaderName: tokenAdmin}); rr.Code != http.StatusForbidden {
			t.Fatalf("admin on the comum route: status = %d, want 403", rr.Code)
		}
	}
	for i := range security.MessageCreateLimit {
		if ok, _ := a.messages.Allow(trackAdminID); !ok {
			t.Fatalf("admin quota: request %d refused, want the full quota left", i+1)
		}
	}
}

// TestReplyRateLimitCountsOthersTickets is P-16: the limiter runs before
// TicketAccess, so B's requests on A's ticket (404) spend B's quota.
func TestReplyRateLimitCountsOthersTickets(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticketA := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	ticketB := a.store.seed(trackJoaoID, domain.TicketTypeOther, "Dúvida de joao")
	cookie, token := a.loginAs(t, joaoLogin)
	headers := map[string]string{security.CSRFHeaderName: token}
	for range security.MessageCreateLimit {
		if rr := a.send(t, http.MethodPost, fmt.Sprintf("/api/tickets/%d/mensagens", ticketA.ID), replyBody("oi"), cookie, headers); rr.Code != http.StatusNotFound {
			t.Fatalf("B on A's ticket: status = %d, want 404", rr.Code)
		}
	}
	if rr := a.send(t, http.MethodPost, fmt.Sprintf("/api/tickets/%d/mensagens", ticketB.ID), replyBody("oi"), cookie, headers); rr.Code != http.StatusTooManyRequests {
		t.Errorf("B on his own ticket after 30 probes: status = %d, want 429", rr.Code)
	}
}

// TestReplyLogsNoText is S-12: the log of accepted and refused replies has
// ids and fixed reasons, never the text.
func TestReplyLogsNoText(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	a := newTrackingApp(t, &logBuf)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	closed := a.store.seed(trackMariaID, domain.TicketTypeOther, "Fechado")
	a.store.setStatus(closed.ID, domain.TicketStatusClosed)
	const secret = "segredo-da-mensagem" //nolint:gosec // a message text to look for in the log, not a credential
	bodies := map[string]string{
		fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID): replyBody(secret),
		fmt.Sprintf("/api/tickets/%d/mensagens", closed.ID): replyBody(secret),
	}
	logBuf.Reset()
	for path, body := range bodies {
		a.replyAs(t, comumLogin, path, body, nil)
	}
	a.replyAs(t, comumLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), replyBody(secret+strings.Repeat("x", 2001)), nil)
	a.replyAs(t, comumLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), `{"texto":"`+secret+`","id":1}`, nil)

	logged := logBuf.String()
	for _, want := range []string{`"msg":"ticket message created"`, `"msg":"ticket message refused"`, `"actor_id":2`} {
		if !strings.Contains(logged, want) {
			t.Errorf("log has no %s:\n%s", want, logged)
		}
	}
	if strings.Contains(logged, secret) {
		t.Errorf("log holds the message text:\n%s", logged)
	}
}

func TestNewHandlerRequiresMessageLimiter(t *testing.T) {
	t.Parallel()

	h, _ := newTestApp(t)
	d := Deps{Logger: h.logger, Security: h.sec, Auth: h.auth, Users: h.users, Tickets: h.tickets, TicketLimiter: h.ticketLimiter}
	if _, err := NewHandler(d); err == nil {
		t.Fatal("NewHandler without MessageLimiter must fail: the reply routes would answer 500")
	}
}

// TestOwnThreadRefusesQueries is plan P-9 for the cursor: anything but one
// positive antes is a 400 with the generic body.
func TestOwnThreadRefusesQueries(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	cookie, _ := a.loginAs(t, comumLogin)
	for _, query := range []string{"antes=0", "antes=-1", "antes=01", "antes=abc", "antes=", "antes=1&antes=2", "x=1", "antes=1&pagina=2", "antes=9223372036854775808", "antes=%zz"} {
		rr := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d/mensagens?%s", ticket.ID, query), "", cookie, nil)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, rr.Code)
			continue
		}
		checkErrorBody(t, rr, "bad_request", msgBadRequest)
	}
}
