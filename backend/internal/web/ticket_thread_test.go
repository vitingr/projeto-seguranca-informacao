package web

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
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
