package security

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"backend_si/internal/domain"
)

func TestParseTicketID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		want   domain.TicketID
		wantOK bool
	}{
		{"one", "1", 1, true},
		{"two digits", "42", 42, true},
		{"largest int64", "9223372036854775807", 9223372036854775807, true},

		{"zero", "0", 0, false},
		{"negative", "-1", 0, false},
		{"plus sign", "+1", 0, false},
		{"leading zero", "01", 0, false},
		{"only zeros", "00", 0, false},
		{"exponent", "1e3", 0, false},
		{"decimal", "1.0", 0, false},
		{"hex", "0x1", 0, false},
		{"letters", "abc", 0, false},
		{"leading space", " 1", 0, false},
		{"trailing space", "1 ", 0, false},
		{"trailing newline", "1\n", 0, false},
		{"empty", "", 0, false},
		{"one past int64", "9223372036854775808", 0, false},
		{"twenty digits", "10000000000000000000", 0, false},
		{"percent-encoded digit", "%31", 0, false},
		{"arabic-indic digit", "١", 0, false},
		{"fullwidth digit", "１", 0, false},
		{"digit then fullwidth digit", "1２", 0, false},
		{"underscore", "1_000", 0, false},
		{"path", "1/2", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseTicketID(tt.raw)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ParseTicketID(%q) = %d, %v; want %d, %v", tt.raw, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCanAccessTicket(t *testing.T) {
	t.Parallel()

	const owner, other domain.UserID = 10, 11
	tests := []struct {
		name      string
		principal Principal
		author    domain.UserID
		want      bool
	}{
		{"admin on any ticket", Principal{UserID: 1, Role: domain.RoleAdmin}, owner, true},
		{"admin on a ticket of another admin id", Principal{UserID: owner, Role: domain.RoleAdmin}, other, true},
		{"comum author", Principal{UserID: owner, Role: domain.RoleComum}, owner, true},

		// BR-1: for a comum user, every other ticket does not exist.
		{"other comum user", Principal{UserID: other, Role: domain.RoleComum}, owner, false},

		// Fail closed.
		{"no principal", Principal{}, owner, false},
		{"zero user id, comum", Principal{Role: domain.RoleComum}, owner, false},
		{"zero user id, admin", Principal{Role: domain.RoleAdmin}, owner, false},
		{"negative user id", Principal{UserID: -1, Role: domain.RoleAdmin}, owner, false},
		{"zero author, admin", Principal{UserID: 1, Role: domain.RoleAdmin}, 0, false},
		{"zero author and zero user, comum", Principal{Role: domain.RoleComum}, 0, false},
		{"negative author", Principal{UserID: 1, Role: domain.RoleAdmin}, -1, false},
		{"unknown role", Principal{UserID: owner, Role: "root"}, owner, false},
		{"empty role", Principal{UserID: owner}, owner, false},
		{"role in another case", Principal{UserID: owner, Role: "Admin"}, owner, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CanAccessTicket(tt.principal, tt.author); got != tt.want {
				t.Errorf("CanAccessTicket(%+v, %d) = %v, want %v", tt.principal, tt.author, got, tt.want)
			}
		})
	}
}

// fakeTicketStore is a TicketOwnerStore over a map, counting its calls.
type fakeTicketStore struct {
	authors map[domain.TicketID]domain.UserID
	err     error // returned for every id when set
	calls   atomic.Int32
}

func (f *fakeTicketStore) TicketAuthor(_ context.Context, id domain.TicketID) (domain.UserID, error) {
	f.calls.Add(1)
	if f.err != nil {
		return 0, f.err
	}
	author, ok := f.authors[id]
	if !ok {
		return 0, fmt.Errorf("repository: ticket author: %w", domain.ErrNotFound)
	}
	return author, nil
}

// renderWithRequestID mimics the web renderer: the generic JSON body and a
// new request id per answer, so tests can show two 404s are identical
// except for that id.
func renderWithRequestID() ErrorRenderer {
	var seq atomic.Int64
	return func(w http.ResponseWriter, _ *http.Request, status int) {
		id := "req-" + strconv.FormatInt(seq.Add(1), 10)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", id)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":      strings.ReplaceAll(strings.ToLower(http.StatusText(status)), " ", "_"),
			"message":    "Recurso não encontrado.",
			"request_id": id,
		})
	}
}

// Users and tickets of the TicketAccess tests: A wrote ticket 1, B wrote
// ticket 2, and ticket 999 does not exist.
const (
	userA, userB, adminID       domain.UserID   = 10, 11, 1
	ticketOfA, ticketOfB, noTkt domain.TicketID = 1, 2, 999
)

var (
	principalA     = Principal{UserID: userA, Role: domain.RoleComum}
	principalB     = Principal{UserID: userB, Role: domain.RoleComum}
	principalAdmin = Principal{UserID: adminID, Role: domain.RoleAdmin}
)

// ticketRoute is one of the routes of spec 003 that take a ticket id: read
// the ticket (R2), list its messages (R3) and reply (R4).
type ticketRoute struct {
	name, method, path string // path has "%s" where the id goes
}

var ticketRoutes = []ticketRoute{
	{"read the ticket", http.MethodGet, "/api/tickets/%s"},
	{"list the messages", http.MethodGet, "/api/tickets/%s/mensagens"},
	{"reply", http.MethodPost, "/api/tickets/%s/mensagens"},
}

type ticketAccessFixture struct {
	t       *testing.T
	store   *fakeTicketStore
	logs    *bytes.Buffer
	mux     *http.ServeMux
	reached []string    // routes whose handler ran
	refs    []TicketRef // TicketRefFrom seen by those handlers
}

func newTicketAccessFixture(t *testing.T, mutate func(o *Options)) *ticketAccessFixture {
	t.Helper()
	f := &ticketAccessFixture{
		t:     t,
		store: &fakeTicketStore{authors: map[domain.TicketID]domain.UserID{ticketOfA: userA, ticketOfB: userB}},
		logs:  &bytes.Buffer{},
		mux:   http.NewServeMux(),
	}
	sec := newTestSecurityWith(t, func(o *Options) {
		o.Logger = slog.New(slog.NewJSONHandler(f.logs, nil))
		o.RenderError = renderWithRequestID()
		o.Tickets = f.store
		if mutate != nil {
			mutate(o)
		}
	})
	for _, route := range ticketRoutes {
		pattern := route.method + " " + strings.Replace(route.path, "%s", "{id}", 1)
		f.mux.Handle(pattern, sec.TicketAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ref, ok := TicketRefFrom(r.Context())
			if !ok {
				t.Errorf("%s: handler reached without a ticket ref", route.name)
			}
			f.reached = append(f.reached, route.name)
			f.refs = append(f.refs, ref)
			w.WriteHeader(http.StatusOK)
		})))
	}
	return f
}

// do sends one request as principal (nil for none) to the fixture's mux.
func (f *ticketAccessFixture) do(route ticketRoute, id string, principal *Principal) *httptest.ResponseRecorder {
	var body *strings.Reader
	if route.method == http.MethodPost {
		body = strings.NewReader(`{"texto":"segredo do corpo"}`)
	} else {
		body = strings.NewReader("")
	}
	ctx := f.t.Context()
	if principal != nil {
		ctx = context.WithValue(ctx, principalKey{}, *principal)
	}
	req := httptest.NewRequestWithContext(ctx, route.method, fmt.Sprintf(route.path, id), body)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

// normalized is an answer with its request id blanked: the status, every
// header and the body bytes. It also checks the body is exactly
// {error, message, request_id} and that request_id is the X-Request-Id.
func normalized(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	id := rec.Header().Get("X-Request-Id")
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
	}
	if keys := slices.Sorted(maps.Keys(body)); !slices.Equal(keys, []string{"error", "message", "request_id"}) {
		t.Errorf("body keys = %v, want [error message request_id]", keys)
	}
	if id == "" || body["request_id"] != id {
		t.Errorf("request_id %v, X-Request-Id %q: want the same non-empty id", body["request_id"], id)
	}
	header := rec.Header().Clone()
	header.Del("X-Request-Id")
	var b strings.Builder
	fmt.Fprintf(&b, "%d\n", rec.Code)
	_ = header.Write(&b)
	b.Write(bytes.ReplaceAll(rec.Body.Bytes(), []byte(`"request_id":"`+id+`"`), []byte(`"request_id":""`)))
	return b.String()
}

func TestTicketAccessLetsTheAuthorAndAdminsThrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		principal Principal
		id        domain.TicketID
		want      TicketRef
	}{
		{"author A on A's ticket", principalA, ticketOfA, TicketRef{ID: ticketOfA, AuthorID: userA}},
		{"author B on B's ticket", principalB, ticketOfB, TicketRef{ID: ticketOfB, AuthorID: userB}},
		{"admin on A's ticket", principalAdmin, ticketOfA, TicketRef{ID: ticketOfA, AuthorID: userA}},
		{"admin on B's ticket", principalAdmin, ticketOfB, TicketRef{ID: ticketOfB, AuthorID: userB}},
	}
	for _, tt := range tests {
		for _, route := range ticketRoutes {
			t.Run(tt.name+", "+route.name, func(t *testing.T) {
				t.Parallel()
				f := newTicketAccessFixture(t, nil)
				rec := f.do(route, formatTicketID(tt.id), &tt.principal)
				if rec.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200", rec.Code)
				}
				if !slices.Equal(f.reached, []string{route.name}) || f.refs[0] != tt.want {
					t.Errorf("reached %v with refs %+v, want %q with %+v", f.reached, f.refs, route.name, tt.want)
				}
				if n := f.store.calls.Load(); n != 1 {
					t.Errorf("store called %d times, want 1", n)
				}
				if f.logs.Len() != 0 {
					t.Errorf("an allowed request was logged: %s", f.logs)
				}
			})
		}
	}
}

// TestTicketAccessIDOR is S-2: user A reading, listing the messages of and
// replying to B's ticket gets the same 404 as for a ticket that does not
// exist or an id that is not a number, byte for byte except the request
// id, and no handler runs (so nothing is read or written).
func TestTicketAccessIDOR(t *testing.T) {
	t.Parallel()

	for _, route := range ticketRoutes {
		t.Run(route.name, func(t *testing.T) {
			t.Parallel()
			f := newTicketAccessFixture(t, nil)
			missing := f.do(route, formatTicketID(noTkt), &principalA)
			if missing.Code != http.StatusNotFound {
				t.Fatalf("missing ticket: status = %d, want 404", missing.Code)
			}
			want := normalized(t, missing)

			cases := []struct {
				name string
				id   string
			}{
				{"A on B's ticket", formatTicketID(ticketOfB)},
				{"A on a malformed id", "abc"},
				{"A on a percent-encoded id", "%31"},
				{"A on id 0", "0"},
			}
			for _, c := range cases {
				rec := f.do(route, c.id, &principalA)
				if got := normalized(t, rec); got != want {
					t.Errorf("%s: answer differs from a missing ticket\n got: %q\nwant: %q", c.name, got, want)
				}
			}
			if len(f.reached) != 0 {
				t.Errorf("handlers reached: %v, want none", f.reached)
			}

			// The same calls from B, the author, and from an admin pass:
			// the 404 above came from the ownership rule only.
			if rec := f.do(route, formatTicketID(ticketOfB), &principalB); rec.Code != http.StatusOK {
				t.Errorf("B on B's ticket: status = %d, want 200", rec.Code)
			}
			if rec := f.do(route, formatTicketID(ticketOfB), &principalAdmin); rec.Code != http.StatusOK {
				t.Errorf("admin on B's ticket: status = %d, want 200", rec.Code)
			}
		})
	}
}

func TestTicketAccessStoreCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		id    string
		calls int32
	}{
		{"own ticket", "1", 1},
		{"someone else's ticket", "2", 1},
		{"missing ticket", "999", 1},
		{"largest int64", "9223372036854775807", 1},
		{"zero", "0", 0},
		{"negative", "-1", 0},
		{"plus sign", "+1", 0},
		{"leading zero", "01", 0},
		{"exponent", "1e3", 0},
		{"letters", "abc", 0},
		{"one past int64", "9223372036854775808", 0},
		{"percent-encoded digit", "%31", 0},
		{"encoded leading space", "%201", 0},
		{"non-ASCII digit", "%D9%A1", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newTicketAccessFixture(t, nil)
			rec := f.do(ticketRoutes[0], tt.id, &principalA)
			if n := f.store.calls.Load(); n != tt.calls {
				t.Errorf("store called %d times, want %d", n, tt.calls)
			}
			if tt.id != "1" && rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404", rec.Code)
			}
		})
	}
}

func TestTicketAccessFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(o *Options)
		principal *Principal
		want      int
	}{
		{"store error", func(o *Options) {
			o.Tickets = &fakeTicketStore{err: errors.New("connection reset")}
		}, &principalA, http.StatusInternalServerError},
		{"nil store", func(o *Options) { o.Tickets = nil }, &principalA, http.StatusInternalServerError},
		{"no principal", nil, nil, http.StatusUnauthorized},
		{"invalid principal", nil, &Principal{UserID: userA, Role: "root"}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newTicketAccessFixture(t, tt.mutate)
			rec := f.do(ticketRoutes[0], formatTicketID(ticketOfA), tt.principal)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
			if len(f.reached) != 0 {
				t.Errorf("handlers reached: %v, want none", f.reached)
			}
		})
	}
}

func TestTicketAccessLogsEachDenialOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		route     ticketRoute
		id        string
		principal *Principal
		ticketID  string
		reason    string
		level     string
	}{
		{"someone else's ticket", ticketRoutes[2], "2", &principalA, "2", ticketNotAllowed, "WARN"},
		{"missing ticket", ticketRoutes[1], "999", &principalA, "999", ticketNotFound, "WARN"},
		{"malformed id with markup", ticketRoutes[2], "%3Cscript%3E", &principalA, ticketMalformed, ticketMalformedID, "WARN"},
		{"no principal", ticketRoutes[0], "1", nil, ticketUnchecked, ticketAnonymous, "WARN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newTicketAccessFixture(t, nil)
			f.do(tt.route, tt.id, tt.principal)

			var lines []map[string]any
			scanner := bufio.NewScanner(bytes.NewReader(f.logs.Bytes()))
			for scanner.Scan() {
				var line map[string]any
				if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
					t.Fatalf("log line %q: %v", scanner.Text(), err)
				}
				lines = append(lines, line)
			}
			if len(lines) != 1 {
				t.Fatalf("%d log lines, want 1: %s", len(lines), f.logs)
			}
			line := lines[0]
			var wantUser float64
			if tt.principal != nil {
				wantUser = float64(tt.principal.UserID)
			}
			if line["level"] != tt.level || line["reason"] != tt.reason ||
				line["ticket_id"] != tt.ticketID || line["user_id"] != wantUser {
				t.Errorf("log line = %v, want level %s, reason %q, ticket_id %q, user_id %v",
					line, tt.level, tt.reason, tt.ticketID, wantUser)
			}
			if _, ok := line["path"]; ok {
				t.Errorf("log line has the path: %v", line)
			}
			for _, leak := range []string{"script", "segredo", "mensagens", "/api/"} {
				if strings.Contains(f.logs.String(), leak) {
					t.Errorf("log contains %q: %s", leak, f.logs)
				}
			}
		})
	}
}

func TestTicketRefFromWithoutRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"no value", context.Background()},
		{"zero ref", context.WithValue(context.Background(), ticketRefKey{}, TicketRef{})},
		{"no author", context.WithValue(context.Background(), ticketRefKey{}, TicketRef{ID: 1})},
		{"no id", context.WithValue(context.Background(), ticketRefKey{}, TicketRef{AuthorID: 1})},
		{"wrong type", context.WithValue(context.Background(), ticketRefKey{}, "1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if ref, ok := TicketRefFrom(tt.ctx); ok {
				t.Errorf("TicketRefFrom = %+v, true; want absent", ref)
			}
		})
	}
}
