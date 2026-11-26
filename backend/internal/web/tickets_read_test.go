package web

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// Generic messages of the error bodies (json.go).
const (
	msgBadRequest = "A requisição não pôde ser processada."
	msgNotFound   = "Recurso não encontrado."
	msgInternal   = "Ocorreu um erro inesperado. Tente novamente mais tarde."
)

var (
	summaryKeys = []string{"created_at", "id", "status", "title", "type", "updated_at"}
	pageKeys    = []string{"items", "page", "page_size", "pages", "total"}
)

// setStatus moves a stored ticket to status without an entry, to build
// list fixtures.
func (m *memTracking) setStatus(id domain.TicketID, status domain.TicketStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[m.find(id)].Status = status
}

// seedLists stores 25 tickets of maria (user A) and 4 of joao (user B), of
// every type and status, and returns the ids of each.
func seedLists(a trackingApp) (maria, joao []domain.TicketID) {
	types := []domain.TicketType{domain.TicketTypeAccess, domain.TicketTypeBug, domain.TicketTypeDeploy, domain.TicketTypeOther}
	statuses := []domain.TicketStatus{domain.TicketStatusOpen, domain.TicketStatusInProgress, domain.TicketStatusResolved, domain.TicketStatusClosed}
	for i := range 25 {
		t := a.store.seed(trackMariaID, types[i%4], fmt.Sprintf("Ticket %d de maria", i+1))
		a.store.setStatus(t.ID, statuses[(i/4)%4])
		maria = append(maria, t.ID)
		if i%8 == 0 {
			t := a.store.seed(trackJoaoID, types[i%4], "Ticket de joao")
			joao = append(joao, t.ID)
		}
	}
	return maria, joao
}

// itemIDs returns the ids of the items of a page answer.
func itemIDs(t *testing.T, body map[string]any) []domain.TicketID {
	t.Helper()
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items = %v, want a JSON array", body["items"])
	}
	ids := make([]domain.TicketID, 0, len(items))
	for _, it := range items {
		ids = append(ids, domain.TicketID(it.(map[string]any)["id"].(float64)))
	}
	return ids
}

// TestListOwnTickets covers AC-1.3, AC-1.4, AC-1.7 and S-13, S-14 on R1:
// only the session user's tickets, newest update first, 20 per page, the
// exact keys, and a page past the end with the real totals (plan P-9).
func TestListOwnTickets(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	maria, joao := seedLists(a)
	cookie, _ := a.loginAs(t, comumLogin)

	rr := a.send(t, http.MethodGet, "/api/tickets", "", cookie, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
	code, body := a.get(t, "/api/tickets", cookie)
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if got := keysOf(body); !slices.Equal(got, pageKeys) {
		t.Errorf("page keys = %v, want %v", got, pageKeys)
	}
	if body["total"] != 25.0 || body["page"] != 1.0 || body["pages"] != 2.0 || body["page_size"] != 20.0 {
		t.Errorf("page = total %v, page %v, pages %v, page_size %v; want 25, 1, 2, 20",
			body["total"], body["page"], body["pages"], body["page_size"])
	}
	for _, it := range body["items"].([]any) {
		if got := keysOf(it); !slices.Equal(got, summaryKeys) {
			t.Fatalf("item keys = %v, want %v (no author, no type fields)", got, summaryKeys)
		}
	}

	// Newest update first: the ids of maria in reverse, after a reply moved
	// the oldest one to the top (BR-8).
	if _, err := a.store.AddMessage(t.Context(), maria[0], trackMariaID, "Alguma novidade?"); err != nil {
		t.Fatal(err)
	}
	want := append([]domain.TicketID{maria[0]}, reversed(maria[1:])...)
	var got []domain.TicketID
	for page := 1; page <= 2; page++ {
		_, pageBody := a.get(t, fmt.Sprintf("/api/tickets?pagina=%d", page), cookie)
		got = append(got, itemIDs(t, pageBody)...)
	}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}

	_, body = a.get(t, "/api/tickets?pagina=3", cookie)
	if ids := itemIDs(t, body); len(ids) != 0 || body["total"] != 25.0 || body["pages"] != 2.0 || body["page"] != 3.0 {
		t.Errorf("page past the end = %v, want no items, total 25, pages 2, page 3", body)
	}

	// joao (user B) sees only his own four, newest first.
	cookieB, _ := a.loginAs(t, joaoLogin)
	_, body = a.get(t, "/api/tickets", cookieB)
	if got := itemIDs(t, body); !slices.Equal(got, reversed(joao)) {
		t.Errorf("joao's list = %v, want %v", got, reversed(joao))
	}
}

// reversed returns a reversed copy of s.
func reversed[S ~[]E, E any](s S) S {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}

// TestListOwnTicketsNeverHoldsOthers is S-3: with every filter and on every
// page, A's list holds none of B's tickets, and only tickets that match.
func TestListOwnTicketsNeverHoldsOthers(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	_, joao := seedLists(a)
	cookie, _ := a.loginAs(t, comumLogin)

	statuses := []string{"", "aberto", "em_andamento", "resolvido", "fechado"}
	types := []string{"", "acesso", "bug", "deploy", "outro"}
	for _, status := range statuses {
		for _, tipo := range types {
			for page := 1; page <= 3; page++ {
				query := fmt.Sprintf("pagina=%d", page)
				if status != "" {
					query += "&status=" + status
				}
				if tipo != "" {
					query += "&tipo=" + tipo
				}
				code, body := a.get(t, "/api/tickets?"+query, cookie)
				if code != http.StatusOK {
					t.Fatalf("%s: status = %d", query, code)
				}
				for _, it := range body["items"].([]any) {
					item := it.(map[string]any)
					id := domain.TicketID(item["id"].(float64))
					if slices.Contains(joao, id) {
						t.Errorf("%s: A's list holds B's ticket %d", query, id)
					}
					if a.store.ticket(id).AuthorID != trackMariaID {
						t.Errorf("%s: ticket %d is not A's", query, id)
					}
					if (status != "" && item["status"] != status) || (tipo != "" && item["type"] != tipo) {
						t.Errorf("%s: item %v does not match the filters", query, item)
					}
				}
			}
		}
	}
}

// TestListOwnTicketsFilters checks the totals of each filter alone and of
// both combined (AC-1.5).
func TestListOwnTicketsFilters(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	seedLists(a)
	cookie, _ := a.loginAs(t, comumLogin)

	// 25 tickets: types cycle every ticket, statuses every 4 tickets.
	tests := []struct {
		query string
		total float64
	}{
		{"", 25},
		{"status=aberto", 8},          // tickets 1-4, 17-20
		{"status=fechado", 4},         // tickets 13-16
		{"tipo=acesso", 7},            // 1, 5, 9, 13, 17, 21, 25
		{"tipo=outro", 6},             // 4, 8, 12, 16, 20, 24
		{"status=aberto&tipo=bug", 2}, // 2, 18
		{"tipo=deploy&status=fechado", 1},
	}
	for _, tt := range tests {
		_, body := a.get(t, "/api/tickets?"+tt.query, cookie)
		if body["total"] != tt.total {
			t.Errorf("%q: total = %v, want %v", tt.query, body["total"], tt.total)
		}
		if n := len(itemIDs(t, body)); float64(n) != min(tt.total, 20) {
			t.Errorf("%q: %d items, want %v", tt.query, n, min(tt.total, 20))
		}
	}
}

// TestListOwnTicketsRefusesQueries covers plan P-9 and S-3, S-6, S-12: every
// malformed, unknown, repeated or empty parameter, and usuario on the own
// list, is a 400 with the generic body; the log has a fixed reason and
// never the query text.
func TestListOwnTicketsRefusesQueries(t *testing.T) {
	t.Parallel()

	queries := []string{
		"usuario=maria", "usuario=", "status=aberto&usuario=joao",
		"x=1", "pagina=1&page=2", "status=aberto&status=fechado", "tipo=bug&tipo=bug",
		"status=", "tipo=", "pagina=",
		"status=Aberto", "status=closed", "tipo=BUG", "tipo=acess",
		"pagina=0", "pagina=-1", "pagina=01", "pagina=+1", "pagina=1.0", "pagina=1e3", "pagina=abc",
		"pagina=10001", "pagina=9223372036854775808",
		"status=%zz", "status=aberto;tipo=bug", "%",
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			var logBuf bytes.Buffer
			a := newTrackingApp(t, &logBuf)
			cookie, _ := a.loginAs(t, comumLogin)
			logBuf.Reset()

			rr := a.send(t, http.MethodGet, "/api/tickets?"+query, "", cookie, nil)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (%s)", rr.Code, rr.Body.String())
			}
			checkErrorBody(t, rr, "bad_request", msgBadRequest)
			if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", cc)
			}
			logged := logBuf.String()
			if !strings.Contains(logged, `"msg":"ticket query refused"`) {
				t.Errorf("log has no refusal:\n%s", logged)
			}
			for _, secret := range []string{"maria", "joao", "Aberto", "closed", "acess", "10001"} {
				if strings.Contains(query, secret) && strings.Contains(logged, secret) {
					t.Errorf("log holds the query value %q:\n%s", secret, logged)
				}
			}
		})
	}
}

// TestListOwnTicketsServerError is the generic 500 when the store fails.
func TestListOwnTicketsServerError(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	cookie, _ := a.loginAs(t, comumLogin)
	a.store.fail(errors.New("database is down"))

	rr := a.send(t, http.MethodGet, "/api/tickets", "", cookie, nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	checkErrorBody(t, rr, "internal", msgInternal)
}

var detailKeys = []string{"created_at", "data", "id", "status", "title", "type", "updated_at"}

// TestGetOwnTicket covers AC-2.1, AC-2.2 (data) and P-10 on R2: every
// field of the type under its request key, exactly the keys of
// security.TicketDataKeys, null for empty notes, no author data (S-13).
func TestGetOwnTicket(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	tests := []struct {
		typ  domain.TicketType
		want map[string]any
	}{
		{domain.TicketTypeAccess, map[string]any{"sistema": "Grafana", "nivel_acesso": "leitura", "justificativa": "Ver os painéis."}},
		{domain.TicketTypeBug, map[string]any{"sistema": "Portal", "descricao": "Erro 500.", "passos": "1. Abrir\n2. Salvar", "severidade": "alta"}}, //nolint:gosec // "passos" is the steps field, not a credential
		{domain.TicketTypeDeploy, map[string]any{"projeto": "api", "repositorio": "https://git.example.com/api", "ambiente": "prod", "observacoes": nil}},
		{domain.TicketTypeOther, map[string]any{"titulo": "Dúvida <b>x</b>", "descricao": "Como peço acesso à VPN?"}},
	}
	cookie, _ := a.loginAs(t, comumLogin)
	for _, tt := range tests {
		ticket := a.store.seed(trackMariaID, tt.typ, "Dúvida <b>x</b>")
		rr := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d", ticket.ID), "", cookie, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200 (%s)", tt.typ, rr.Code, rr.Body.String())
		}
		if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", tt.typ, cc)
		}
		_, body := a.get(t, fmt.Sprintf("/api/tickets/%d", ticket.ID), cookie)
		if got := keysOf(body); !slices.Equal(got, detailKeys) {
			t.Errorf("%s: keys = %v, want %v (no author data)", tt.typ, got, detailKeys)
		}
		if body["id"] != float64(ticket.ID) || body["type"] != string(tt.typ) || body["status"] != "aberto" || body["title"] != "Dúvida <b>x</b>" {
			t.Errorf("%s: header fields = %v", tt.typ, body)
		}
		data, _ := body["data"].(map[string]any)
		if got, want := sortedKeys(data), sorted(security.TicketDataKeys()[tt.typ]); !slices.Equal(got, want) {
			t.Errorf("%s: data keys = %v, want the request keys %v", tt.typ, got, want)
		}
		for key, value := range tt.want {
			if data[key] != value {
				t.Errorf("%s: data[%s] = %#v, want %#v", tt.typ, key, data[key], value)
			}
		}
	}
}

// TestTicketDataCoversEveryKey checks that ticketData knows every request
// key of every type: a fully filled ticket has no null value but the notes.
func TestTicketDataCoversEveryKey(t *testing.T) {
	t.Parallel()

	full := domain.Ticket{
		Title: "t", System: "s", AccessLevel: domain.AccessLevelRead, Justification: "j",
		Description: "d", Steps: "p", Severity: domain.SeverityLow, Project: "p",
		Repository: "https://r", Environment: domain.EnvironmentDevelopment, Notes: "n",
	}
	for typ, keys := range security.TicketDataKeys() {
		full.Type = typ
		data := ticketData(full)
		for _, key := range keys {
			if data[key] == nil {
				t.Errorf("%s: ticketData has no value for %q", typ, key)
			}
		}
		if len(data) != len(keys) {
			t.Errorf("%s: ticketData has %d keys, want %d", typ, len(data), len(keys))
		}
	}
}

// TestGetOwnTicketServerError is the generic 500 when the read fails after
// TicketAccess let the request through.
func TestGetOwnTicketServerError(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	ticket := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")
	cookie, _ := a.loginAs(t, comumLogin)
	a.store.fail(errors.New("database is down"))

	rr := a.send(t, http.MethodGet, fmt.Sprintf("/api/tickets/%d", ticket.ID), "", cookie, nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	checkErrorBody(t, rr, "internal", msgInternal)
}

var (
	adminSummaryKeys = []string{"author_login", "created_at", "id", "status", "title", "type", "updated_at"}
	queueKeys        = []string{"items", "page", "page_size", "pages", "status_counts", "total"}
)

// TestListQueue covers AC-4.2 to AC-4.5 on R5: every user's tickets with
// the author's login, the filters alone and combined (usuario included),
// and the status counts of the whole queue whatever the filters.
func TestListQueue(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	seedLists(a) // maria: aberto 8, em_andamento 8, resolvido 5, fechado 4; joao: aberto 4
	cookie, _ := a.loginAs(t, flowLogin)
	wantCounts := map[string]any{"aberto": 12.0, "em_andamento": 8.0, "resolvido": 5.0, "fechado": 4.0}

	tests := []struct {
		query  string
		total  float64
		author string // every item's author_login, when set
	}{
		{"", 29, ""},
		{"usuario=joao", 4, joaoLogin},
		{"usuario=maria&tipo=bug&status=aberto", 2, comumLogin},
		{"status=fechado", 4, comumLogin},
		{"tipo=acesso", 11, ""}, // 7 of maria and the 4 of joao
		{"usuario=ana", 0, ""},
		{"pagina=2", 29, ""},
	}
	for _, tt := range tests {
		rr := a.send(t, http.MethodGet, "/api/administradores/tickets?"+tt.query, "", cookie, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("%q: status = %d (%s)", tt.query, rr.Code, rr.Body.String())
		}
		if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("%q: Cache-Control = %q", tt.query, cc)
		}
		_, body := a.get(t, "/api/administradores/tickets?"+tt.query, cookie)
		if got := keysOf(body); !slices.Equal(got, queueKeys) {
			t.Errorf("%q: keys = %v, want %v", tt.query, got, queueKeys)
		}
		if body["total"] != tt.total || body["page_size"] != 20.0 {
			t.Errorf("%q: total %v, page_size %v; want %v, 20", tt.query, body["total"], body["page_size"], tt.total)
		}
		counts, _ := body["status_counts"].(map[string]any)
		if !maps.Equal(counts, wantCounts) {
			t.Errorf("%q: status_counts = %v, want %v (whole queue)", tt.query, counts, wantCounts)
		}
		items := body["items"].([]any)
		if len(items) > 20 {
			t.Errorf("%q: %d items, want at most 20", tt.query, len(items))
		}
		for _, it := range items {
			item := it.(map[string]any)
			if got := keysOf(item); !slices.Equal(got, adminSummaryKeys) {
				t.Fatalf("%q: item keys = %v, want %v", tt.query, got, adminSummaryKeys)
			}
			if tt.author != "" && item["author_login"] != tt.author {
				t.Errorf("%q: item of %v, want %s", tt.query, item["author_login"], tt.author)
			}
		}
	}
}

// TestListQueueRefusals covers S-4 and plan P-9 on R5: comum users get 403
// before any read, and malformed queries 400.
func TestListQueueRefusals(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	seedLists(a)
	cookie, _ := a.loginAs(t, comumLogin)
	calls := a.store.handlerCalls()
	rr := a.send(t, http.MethodGet, "/api/administradores/tickets", "", cookie, nil)
	if rr.Code != http.StatusForbidden || a.store.handlerCalls() != calls {
		t.Errorf("comum: status = %d, store calls %d; want 403 and no read", rr.Code, a.store.handlerCalls()-calls)
	}

	admin, _ := a.loginAs(t, flowLogin)
	for _, query := range []string{"usuario=", "usuario=Maria%20Souza", "usuario=ab", "usuario=maria&usuario=joao", "x=1", "status=closed", "pagina=0", "status=%zz"} {
		rr := a.send(t, http.MethodGet, "/api/administradores/tickets?"+query, "", admin, nil)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, rr.Code)
			continue
		}
		checkErrorBody(t, rr, "bad_request", msgBadRequest)
	}
}

// TestListQueueAuthors covers AC-4.4 (Q-7) and P-18 on R6: the logins of
// the users with tickets, alphabetical, nobody without tickets; any
// parameter is a 400; the literal path never goes through TicketAccess.
func TestListQueueAuthors(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	a := newTrackingApp(t, &logBuf)
	cookie, _ := a.loginAs(t, flowLogin)
	rr := a.send(t, http.MethodGet, "/api/administradores/tickets/autores", "", cookie, nil)
	if got := rr.Body.String(); rr.Code != http.StatusOK || got != `{"logins":[]}` {
		t.Errorf("no tickets: %d %s, want 200 with an empty list", rr.Code, got)
	}

	seedLists(a)
	logBuf.Reset()
	rr = a.send(t, http.MethodGet, "/api/administradores/tickets/autores", "", cookie, nil)
	if got := rr.Body.String(); rr.Code != http.StatusOK || got != `{"logins":["joao","maria"]}` {
		t.Errorf("authors = %d %s, want joao and maria only (ana and admin have no ticket)", rr.Code, got)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q", cc)
	}
	if strings.Contains(logBuf.String(), "ticket access") {
		t.Errorf("autores went through TicketAccess:\n%s", logBuf.String())
	}

	for _, query := range []string{"x=1", "pagina=2", "usuario=maria", "%zz"} {
		rr := a.send(t, http.MethodGet, "/api/administradores/tickets/autores?"+query, "", cookie, nil)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, rr.Code)
			continue
		}
		checkErrorBody(t, rr, "bad_request", msgBadRequest)
	}

	comum, _ := a.loginAs(t, comumLogin)
	if rr := a.send(t, http.MethodGet, "/api/administradores/tickets/autores", "", comum, nil); rr.Code != http.StatusForbidden {
		t.Errorf("comum: status = %d, want 403", rr.Code)
	}
}

// TestListOwnTicketsEmpty is an empty list: items is [], never null, and
// "Página 1 de 1".
func TestListOwnTicketsEmpty(t *testing.T) {
	t.Parallel()

	a := newTrackingApp(t, nil)
	cookie, _ := a.loginAs(t, comumLogin)
	rr := a.send(t, http.MethodGet, "/api/tickets", "", cookie, nil)
	if got := rr.Body.String(); !strings.Contains(got, `"items":[]`) || !strings.Contains(got, `"total":0,"page":1,"pages":1,"page_size":20`) {
		t.Errorf("empty list = %s", got)
	}
}
