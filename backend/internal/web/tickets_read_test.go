package web

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
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
