package web

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// idorRoute is a comum route with a ticket id, as user B would call it on
// user A's ticket (spec 003 S-2). %s is the ticket id in the path.
type idorRoute struct {
	name   string
	method string
	path   string
	body   string
}

// idorRoutes are the comum routes that take a ticket id.
var idorRoutes = []idorRoute{
	{"R2 detail", http.MethodGet, "/api/tickets/%s", ""},
}

// missingIDs are ids that must answer like a ticket that does not exist:
// one that was never created, and malformed ones (S-6).
var missingIDs = []string{"999999", "0", "-1", "01", "1e3", "abc", "9223372036854775808", "%31"}

// normalized is a response with the request id taken out of the headers
// and the body, so two answers can be compared byte for byte.
type normalized struct {
	status  int
	headers http.Header
	body    string
}

func normalize(rr *httptest.ResponseRecorder) normalized {
	headers := rr.Header().Clone()
	id := headers.Get("X-Request-Id")
	headers.Del("X-Request-Id")
	return normalized{status: rr.Code, headers: headers, body: strings.ReplaceAll(rr.Body.String(), id, "<request_id>")}
}

func (n normalized) equal(o normalized) bool {
	if n.status != o.status || n.body != o.body || len(n.headers) != len(o.headers) {
		return false
	}
	for k, v := range n.headers {
		if !slices.Equal(v, o.headers[k]) {
			return false
		}
	}
	return true
}

// TestTicketIDOR is S-2 over HTTP with the real handlers: user B asking for
// user A's ticket gets a 404 byte-identical (status, headers but the
// request id, body) to a ticket that does not exist and to malformed ids;
// the handler is never reached, nothing is written, and the denial is one
// log line with B's id and the ticket id.
func TestTicketIDOR(t *testing.T) {
	t.Parallel()

	for _, rt := range idorRoutes {
		t.Run(rt.name, func(t *testing.T) {
			t.Parallel()
			var logBuf bytes.Buffer
			a := newTrackingApp(t, &logBuf)
			ticketA := a.store.seed(trackMariaID, domain.TicketTypeBug, "Bug em Portal")
			cookieB, tokenB := a.loginAs(t, joaoLogin)
			headers := map[string]string{security.CSRFHeaderName: tokenB}
			tickets, entries := a.store.sizes()
			calls := a.store.handlerCalls()

			logBuf.Reset()
			got := a.send(t, rt.method, fmt.Sprintf(rt.path, fmt.Sprint(ticketA.ID)), rt.body, cookieB, headers)
			if got.Code != http.StatusNotFound {
				t.Fatalf("B on A's ticket: status = %d, want 404 (%s)", got.Code, got.Body.String())
			}
			checkErrorBody(t, got, "not_found", msgNotFound)
			logged := logBuf.String()
			for _, want := range []string{`"reason":"ticket access: not the author"`, `"user_id":3`, fmt.Sprintf(`"ticket_id":"%d"`, ticketA.ID)} {
				if !strings.Contains(logged, want) {
					t.Errorf("denial log has no %s:\n%s", want, logged)
				}
			}
			if rt.body != "" && strings.Contains(logged, rt.body) {
				t.Errorf("denial log holds the body:\n%s", logged)
			}

			for _, id := range missingIDs {
				rr := a.send(t, rt.method, fmt.Sprintf(rt.path, id), rt.body, cookieB, headers)
				if !normalize(got).equal(normalize(rr)) {
					t.Errorf("id %s: answer differs from B on A's ticket:\n%d %v %s\nwant\n%d %v %s",
						id, rr.Code, rr.Header(), rr.Body.String(), got.Code, got.Header(), got.Body.String())
				}
			}

			if n := a.store.handlerCalls(); n != calls {
				t.Errorf("the handler reached the store %d times, want never", n-calls)
			}
			if gotTickets, gotEntries := a.store.sizes(); gotTickets != tickets || gotEntries != entries {
				t.Errorf("rows = %d tickets, %d entries; want %d, %d (nothing written)", gotTickets, gotEntries, tickets, entries)
			}
			if after := a.store.ticket(ticketA.ID); after.Status != ticketA.Status || !after.UpdatedAt.Equal(ticketA.UpdatedAt) {
				t.Errorf("A's ticket changed: %+v", after)
			}
		})
	}
}

// TestEveryTicketRouteChecksAccess walks the routes table: every pattern
// with {id} must run TicketAccess before its handler, so a ticket that does
// not exist, a malformed id and (for a comum route) someone else's ticket
// never reach the handler (plan P-8).
func TestEveryTicketRouteChecksAccess(t *testing.T) {
	t.Parallel()

	h, _ := newTestApp(t)
	var checked []string
	for _, rt := range h.routes() {
		if !strings.Contains(rt.path, "{id}") {
			continue
		}
		checked = append(checked, rt.pattern())
		t.Run(rt.pattern(), func(t *testing.T) {
			t.Parallel()
			a := newTrackingApp(t, nil)
			ticketA := a.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida")

			type attempt struct{ caller, id string }
			var attempts []attempt
			for _, role := range rt.policy.Roles() {
				switch role {
				case domain.RoleComum:
					attempts = append(attempts, attempt{joaoLogin, fmt.Sprint(ticketA.ID)}, attempt{comumLogin, "999999"}, attempt{comumLogin, "abc"})
				case domain.RoleAdmin:
					attempts = append(attempts, attempt{flowLogin, "999999"}, attempt{flowLogin, "0"})
				}
			}
			if len(attempts) == 0 {
				t.Fatalf("%s: no role to try (public {id} route?)", rt.pattern())
			}
			for _, at := range attempts {
				cookie, token := a.loginAs(t, at.caller)
				calls := a.store.handlerCalls()
				path := strings.Replace(rt.path, "{id}", at.id, 1)
				rr := a.send(t, rt.method, path, "{}", cookie, map[string]string{security.CSRFHeaderName: token})
				if rr.Code != http.StatusNotFound {
					t.Errorf("%s as %s: status = %d, want 404", path, at.caller, rr.Code)
				}
				if n := a.store.handlerCalls(); n != calls {
					t.Errorf("%s as %s reached the handler", path, at.caller)
				}
			}
		})
	}
	if len(checked) == 0 {
		t.Fatal("no route with {id}: the walk checked nothing")
	}
}
