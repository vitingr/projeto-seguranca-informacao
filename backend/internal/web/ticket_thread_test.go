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

func (a trackingApp) addMessage(t *testing.T, ticket domain.TicketID, actor domain.UserID, text string) domain.TicketEntry {
	t.Helper()
	e, err := a.store.AddMessage(t.Context(), ticket, actor, text)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

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

	empty := a.store.seed(trackMariaID, domain.TicketTypeOther, "Sem mensagens")
	rr := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d/mensagens", empty.ID), "", cookie, nil)
	if got := rr.Body.String(); got != `{"entries":[],"has_more":false}` {
		t.Errorf("empty thread = %s", got)
	}
}

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

func (a trackingApp) mustLogin(t *testing.T, login string) *http.Cookie {
	t.Helper()
	cookie, _ := a.loginAs(t, login)
	return cookie
}

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

func TestReplyLongestMessage(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	rr := a.replyAs(t, comumLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), replyBody(strings.Repeat("é", 2000)), nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
}

func TestReplyRateLimit(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	ticketB := a.store.seed(trackJoaoID, domain.TicketTypeOther, "Dúvida de joao")
	path := fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID)
	cookie, token := a.loginAs(t, comumLogin)
	headers := map[string]string{security.CSRFHeaderName: token}

	for range 5 {
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

	if rr := a.replyAs(t, joaoLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticketB.ID), replyBody("oi"), nil); rr.Code != http.StatusCreated {
		t.Errorf("another user: status = %d, want 201", rr.Code)
	}

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

func TestReplyAdminTicket(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeBug, "Bug em Portal")
	newer := a.store.seed(trackMariaID, domain.TicketTypeOther, "Mais novo")
	rr := a.replyAs(t, anaLogin, fmt.Sprintf("/api/administradores/tickets/%d/mensagens", ticket.ID), replyBody(" Estamos vendo. "), nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["side"] != "support" || body["mine"] != true || body["actor_login"] != anaLogin || body["text"] != "Estamos vendo." {
		t.Errorf("entry = %v", body)
	}
	if a.store.ticket(ticket.ID).Status != domain.TicketStatusOpen {
		t.Error("an admin reply changed the status")
	}

	author := a.mustLogin(t, comumLogin)
	thread := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), "", author, nil)
	if strings.Contains(thread.Body.String(), anaLogin) || strings.Contains(thread.Body.String(), "actor") {
		t.Errorf("the author's thread names the admin: %s", thread.Body.String())
	}
	_, tb := a.get(t, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), author)
	if e := entriesOf(t, tb); len(e) != 1 || e[0]["side"] != "support" || e[0]["mine"] != false {
		t.Errorf("the author's thread = %v, want one support entry", e)
	}

	_, own := a.get(t, "/api/tickets", author)
	_, queue := a.get(t, "/api/administradores/tickets", a.mustLogin(t, flowLogin))
	for name, page := range map[string]map[string]any{"own list": own, "queue": queue} {
		if ids := itemIDs(t, page); len(ids) != 2 || ids[0] != ticket.ID || ids[1] != newer.ID {
			t.Errorf("%s order = %v, want the replied ticket %d first", name, ids, ticket.ID)
		}
	}
}

func TestReplyAdminRefusals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		headers    map[string]string
		closed     bool
		wantStatus int
	}{
		{name: "server key", body: `{"texto":"oi","ticket_id":1}`, wantStatus: http.StatusBadRequest},
		{name: "empty text", body: replyBody("  "), wantStatus: http.StatusUnprocessableEntity},
		{name: "closed", body: replyBody("oi"), closed: true, wantStatus: http.StatusConflict},
		{name: "no token", body: replyBody("oi"), headers: map[string]string{security.CSRFHeaderName: ""}, wantStatus: http.StatusForbidden},
		{name: "wrong token", body: replyBody("oi"), headers: map[string]string{security.CSRFHeaderName: "AAAAAAAAAAAAAAAAAAAAAAAAAA"}, wantStatus: http.StatusForbidden},
		{name: "cross-site", body: replyBody("oi"), headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newTrackingApp(t, nil)
			ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
			if tt.closed {
				a.store.setStatus(ticket.ID, domain.TicketStatusClosed)
			}
			rr := a.replyAs(t, flowLogin, fmt.Sprintf("/api/administradores/tickets/%d/mensagens", ticket.ID), tt.body, tt.headers)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if tt.closed {
				checkErrorBody(t, rr, "conflict", msgTicketClosed)
			}
			if _, entries := a.store.sizes(); entries != 0 {
				t.Errorf("%d entries stored, want 0", entries)
			}
		})
	}
}

func TestReplyAdminRateLimit(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	a := newTrackingApp(t, &logBuf)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	path := fmt.Sprintf("/api/administradores/tickets/%d/mensagens", ticket.ID)
	cookie, token := a.loginAs(t, flowLogin)
	headers := map[string]string{security.CSRFHeaderName: token}
	const secret = "texto-do-suporte" //nolint:gosec // a message text to look for in the log, not a credential
	for i := 1; i <= security.MessageCreateLimit; i++ {
		if rr := a.send(t, http.MethodPost, path, replyBody(fmt.Sprintf("%s %d", secret, i)), cookie, headers); rr.Code != http.StatusCreated {
			t.Fatalf("reply %d: status = %d", i, rr.Code)
		}
	}
	if rr := a.send(t, http.MethodPost, path, replyBody(secret), cookie, headers); rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") == "" {
		t.Errorf("reply 31: status = %d, Retry-After %q; want 429 with Retry-After", rr.Code, rr.Header().Get("Retry-After"))
	}
	if rr := a.replyAs(t, comumLogin, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), replyBody("oi"), nil); rr.Code != http.StatusCreated {
		t.Errorf("the author after the admin's 31: status = %d, want 201", rr.Code)
	}
	if strings.Contains(logBuf.String(), secret) {
		t.Error("the log holds the admin's text")
	}
}

func statusBody(from, to string) string {
	return fmt.Sprintf(`{"de":%q,"para":%q}`, from, to)
}

func (a trackingApp) patchStatus(t *testing.T, login string, ticket domain.TicketID, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	cookie, token := a.loginAs(t, login)
	h := map[string]string{security.CSRFHeaderName: token}
	maps.Copy(h, headers)
	return a.send(t, http.MethodPatch, fmt.Sprintf("/api/administradores/tickets/%d/status", ticket), body, cookie, h)
}

func TestChangeStatus(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	a := newTrackingApp(t, &logBuf)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeBug, "Bug em Portal")
	steps := [][2]string{{"aberto", "em_andamento"}, {"em_andamento", "resolvido"}, {"resolvido", "em_andamento"}, {"em_andamento", "fechado"}}
	last := ticket.UpdatedAt
	for i, s := range steps {
		rr := a.patchStatus(t, flowLogin, ticket.ID, statusBody(s[0], s[1]), nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s → %s: status = %d (%s)", s[0], s[1], rr.Code, rr.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if got := keysOf(body); !slices.Equal(got, []string{"id", "status", "updated_at"}) || body["status"] != s[1] || body["id"] != float64(ticket.ID) {
			t.Errorf("%s → %s: body = %v", s[0], s[1], body)
		}
		after := a.store.ticket(ticket.ID)
		if string(after.Status) != s[1] || !after.UpdatedAt.After(last) {
			t.Errorf("step %d: ticket = %s at %v, want %s after %v", i, after.Status, after.UpdatedAt, s[1], last)
		}
		last = after.UpdatedAt
	}
	_, thread := a.get(t, fmt.Sprintf("/api/tickets/%d/mensagens", ticket.ID), a.mustLogin(t, comumLogin))
	entries := entriesOf(t, thread)
	if len(entries) != len(steps) {
		t.Fatalf("%d entries, want %d status changes", len(entries), len(steps))
	}
	for i, e := range entries {
		if e["kind"] != "status_change" || e["from"] != steps[i][0] || e["to"] != steps[i][1] || e["side"] != "support" {
			t.Errorf("entry %d = %v", i, e)
		}
	}
	logged := logBuf.String()
	if !strings.Contains(logged, `"msg":"ticket status changed"`) || !strings.Contains(logged, `"from":"resolvido","to":"em_andamento"`) {
		t.Errorf("log has no status change with from and to:\n%s", logged)
	}
}

func TestChangeStatusRefusals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		login      string
		status     domain.TicketStatus
		body       string
		headers    map[string]string
		wantStatus int
		wantMsg    string
	}{
		{name: "stale de", login: flowLogin, status: domain.TicketStatusInProgress, body: statusBody("aberto", "fechado"), wantStatus: http.StatusConflict, wantMsg: msgStaleStatus},
		{name: "from fechado", login: flowLogin, status: domain.TicketStatusClosed, body: statusBody("fechado", "aberto"), wantStatus: http.StatusConflict, wantMsg: msgStaleStatus},
		{name: "from fechado to fechado", login: flowLogin, status: domain.TicketStatusClosed, body: statusBody("fechado", "fechado"), wantStatus: http.StatusConflict, wantMsg: msgStaleStatus},
		{name: "pair never allowed", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "resolvido"), wantStatus: http.StatusUnprocessableEntity, wantMsg: msgDadosInvalidos},
		{name: "same status", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "aberto"), wantStatus: http.StatusUnprocessableEntity, wantMsg: msgDadosInvalidos},
		{name: "back to aberto", login: flowLogin, status: domain.TicketStatusInProgress, body: statusBody("em_andamento", "aberto"), wantStatus: http.StatusUnprocessableEntity, wantMsg: msgDadosInvalidos},
		{name: "unknown de", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("open", "fechado"), wantStatus: http.StatusUnprocessableEntity, wantMsg: msgDadosInvalidos},
		{name: "unknown para", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "Fechado"), wantStatus: http.StatusUnprocessableEntity, wantMsg: msgDadosInvalidos},
		{name: "missing para", login: flowLogin, status: domain.TicketStatusOpen, body: `{"de":"aberto"}`, wantStatus: http.StatusBadRequest, wantMsg: msgBadRequest},
		{name: "actor", login: flowLogin, status: domain.TicketStatusOpen, body: `{"de":"aberto","para":"fechado","actor":1}`, wantStatus: http.StatusBadRequest, wantMsg: msgBadRequest},
		{name: "updated_at", login: flowLogin, status: domain.TicketStatusOpen, body: `{"de":"aberto","para":"fechado","updated_at":"2026-01-01T00:00:00Z"}`, wantStatus: http.StatusBadRequest, wantMsg: msgBadRequest},
		{name: "id", login: flowLogin, status: domain.TicketStatusOpen, body: `{"de":"aberto","para":"fechado","id":2}`, wantStatus: http.StatusBadRequest, wantMsg: msgBadRequest},
		{name: "the author", login: comumLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "fechado"), wantStatus: http.StatusForbidden, wantMsg: msgForbidden},
		{name: "no CSRF token", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "fechado"), headers: map[string]string{security.CSRFHeaderName: ""}, wantStatus: http.StatusForbidden, wantMsg: msgForbidden},
		{name: "wrong CSRF token", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "fechado"), headers: map[string]string{security.CSRFHeaderName: "AAAAAAAAAAAAAAAAAAAAAAAAAA"}, wantStatus: http.StatusForbidden, wantMsg: msgForbidden},
		{name: "cross-site", login: flowLogin, status: domain.TicketStatusOpen, body: statusBody("aberto", "fechado"), headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, wantStatus: http.StatusForbidden, wantMsg: msgForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var logBuf bytes.Buffer
			a := newTrackingApp(t, &logBuf)
			ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
			a.store.setStatus(ticket.ID, tt.status)
			before := a.store.ticket(ticket.ID)

			rr := a.patchStatus(t, tt.login, ticket.ID, tt.body, tt.headers)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			var body errorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body.Message != tt.wantMsg {
				t.Errorf("message = %q (%v), want %q", body.Message, err, tt.wantMsg)
			}
			after := a.store.ticket(ticket.ID)
			if after.Status != before.Status || !after.UpdatedAt.Equal(before.UpdatedAt) {
				t.Errorf("ticket changed: %s at %v", after.Status, after.UpdatedAt)
			}
			if _, entries := a.store.sizes(); entries != 0 {
				t.Errorf("%d entries written, want 0", entries)
			}
			for _, raw := range []string{"open", "Fechado"} {
				if strings.Contains(tt.body, raw) && strings.Contains(logBuf.String(), raw) {
					t.Errorf("the log holds the code sent %q", raw)
				}
			}
		})
	}
}

func TestChangeStatusRateLimit(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	cookie, token := a.loginAs(t, flowLogin)
	headers := map[string]string{security.CSRFHeaderName: token}
	path := fmt.Sprintf("/api/administradores/tickets/%d/status", ticket.ID)
	for i := 1; i <= security.StatusChangeLimit; i++ {
		if rr := a.send(t, http.MethodPatch, path, statusBody("resolvido", "fechado"), cookie, headers); rr.Code != http.StatusConflict {
			t.Fatalf("request %d: status = %d, want 409 (stale de)", i, rr.Code)
		}
	}
	rr := a.send(t, http.MethodPatch, path, statusBody("aberto", "em_andamento"), cookie, headers)
	if rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("request 31: status = %d, want 429 with Retry-After", rr.Code)
	}
	if a.store.ticket(ticket.ID).Status != domain.TicketStatusOpen {
		t.Error("the 31st request changed the status")
	}
	if rr := a.send(t, http.MethodPost, fmt.Sprintf("/api/administradores/tickets/%d/mensagens", ticket.ID), replyBody("oi"), cookie, headers); rr.Code != http.StatusCreated {
		t.Errorf("a reply after the status quota: status = %d, want 201", rr.Code)
	}
	if ok, _ := a.messages.Allow(trackAdminID); !ok {
		t.Error("the status requests used the reply quota")
	}
}

func TestNewHandlerRequiresStatusLimiter(t *testing.T) {
	t.Parallel()

	h, _ := newTestApp(t)
	d := Deps{
		Logger: h.logger, Security: h.sec, Auth: h.auth, Users: h.users, Tickets: h.tickets,
		TicketLimiter: h.ticketLimiter, MessageLimiter: h.messageLimiter,
	}
	if _, err := NewHandler(d); err == nil {
		t.Fatal("NewHandler without StatusLimiter must fail: the status route would answer 500")
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
