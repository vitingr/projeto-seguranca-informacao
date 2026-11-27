package web

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
	"backend_si/internal/service"
)

// The users of the ticket tracking tests (spec 003). maria and joao are the
// comum users A and B of the IDOR tests; admin and ana are two admins.
const (
	trackAdminID domain.UserID = 1
	trackMariaID domain.UserID = 2
	trackJoaoID  domain.UserID = 3
	trackAnaID   domain.UserID = 4

	joaoLogin = "joao"
	anaLogin  = "ana"
)

var trackingUsers = map[string]domain.User{
	flowLogin:  {ID: trackAdminID, Login: flowLogin, Role: domain.RoleAdmin},
	comumLogin: {ID: trackMariaID, Login: comumLogin, Role: domain.RoleComum},
	joaoLogin:  {ID: trackJoaoID, Login: joaoLogin, Role: domain.RoleComum},
	anaLogin:   {ID: trackAnaID, Login: anaLogin, Role: domain.RoleAdmin},
}

// trackingAuth accepts the four tracking users, all with flowPassword.
type trackingAuth struct{}

func (trackingAuth) Authenticate(_ context.Context, login, password string) (domain.User, error) {
	if u, ok := trackingUsers[login]; ok && password == flowPassword {
		return u, nil
	}
	return domain.User{}, fmt.Errorf("fake: %w", domain.ErrInvalidCredentials)
}

// memTracking is an in-memory store of tickets and their entries. It is the
// service.TicketRepository, the service.TicketEntryRepository and the
// security.TicketOwnerStore of the tracking app, with the semantics of the
// repository (order by last update, pages of 20, compare-and-set on the
// status, no message after the close), checked against PostgreSQL by the
// repository integration tests.
type memTracking struct {
	mu      sync.Mutex
	clock   time.Time
	tickets []domain.Ticket // in id order
	entries []domain.TicketEntry
	err     error // returned by every call a handler makes, when set
	// calls counts the calls only a handler makes (every method but
	// TicketAuthor, which TicketAccess uses).
	calls int
}

// handlerCalls returns how many times a handler reached the store.
func (m *memTracking) handlerCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// lock takes the store's lock for a call a handler makes and counts it.
func (m *memTracking) lock() {
	m.mu.Lock()
	m.calls++
}

func newMemTracking() *memTracking {
	return &memTracking{clock: time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)}
}

// tick returns a new time, one second after the previous one, so the order
// by last update is deterministic.
func (m *memTracking) tick() time.Time {
	m.clock = m.clock.Add(time.Second)
	return m.clock
}

func loginOf(id domain.UserID) (string, domain.Role) {
	for _, u := range trackingUsers {
		if u.ID == id {
			return u.Login, u.Role
		}
	}
	return "", ""
}

func (m *memTracking) find(id domain.TicketID) int {
	return slices.IndexFunc(m.tickets, func(t domain.Ticket) bool { return t.ID == id })
}

func (m *memTracking) Create(_ context.Context, t domain.Ticket) (domain.Ticket, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.Ticket{}, m.err
	}
	t.ID = domain.TicketID(len(m.tickets) + 1)
	t.Status = domain.TicketStatusOpen
	t.CreatedAt = m.tick()
	t.UpdatedAt = t.CreatedAt
	m.tickets = append(m.tickets, t)
	return t, nil
}

func (m *memTracking) Get(_ context.Context, id domain.TicketID) (domain.Ticket, string, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.Ticket{}, "", m.err
	}
	i := m.find(id)
	if i < 0 {
		return domain.Ticket{}, "", fmt.Errorf("mem: %w", domain.ErrNotFound)
	}
	login, _ := loginOf(m.tickets[i].AuthorID)
	return m.tickets[i], login, nil
}

// TicketAuthor ignores fail: it lets the request reach the handler, whose
// store call then fails.
func (m *memTracking) TicketAuthor(_ context.Context, id domain.TicketID) (domain.UserID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.find(id)
	if i < 0 {
		return 0, fmt.Errorf("mem: %w", domain.ErrNotFound)
	}
	return m.tickets[i].AuthorID, nil
}

// list filters, orders by last update (ties by id, highest first) and pages
// like the repository.
func (m *memTracking) list(q domain.TicketListQuery, keep func(*domain.Ticket) bool) domain.TicketPage {
	var rows []*domain.Ticket
	for i := range m.tickets {
		t := &m.tickets[i]
		login, _ := loginOf(t.AuthorID)
		if keep(t) && (q.Status == "" || t.Status == q.Status) && (q.Type == "" || t.Type == q.Type) &&
			(q.AuthorLogin == "" || login == q.AuthorLogin) {
			rows = append(rows, t)
		}
	}
	slices.SortFunc(rows, func(a, b *domain.Ticket) int {
		return cmp.Or(b.UpdatedAt.Compare(a.UpdatedAt), cmp.Compare(b.ID, a.ID))
	})
	page := max(q.Page, 1)
	out := domain.TicketPage{
		Items: []domain.TicketSummary{}, Total: len(rows), Page: page,
		Pages: max(1, (len(rows)+listPageSize-1)/listPageSize),
	}
	for _, t := range rows[min(len(rows), (page-1)*listPageSize):min(len(rows), page*listPageSize)] {
		login, _ := loginOf(t.AuthorID)
		out.Items = append(out.Items, domain.TicketSummary{
			ID: t.ID, Type: t.Type, Status: t.Status, Title: t.Title, AuthorLogin: login,
			CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		})
	}
	return out
}

func (m *memTracking) ListOwn(_ context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.TicketPage{}, m.err
	}
	page := m.list(q, func(t *domain.Ticket) bool { return t.AuthorID == author })
	for i := range page.Items {
		page.Items[i].AuthorLogin = "" // the repository does not select it
	}
	return page, nil
}

func (m *memTracking) ListQueue(_ context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.TicketPage{}, domain.StatusCounts{}, m.err
	}
	var counts domain.StatusCounts
	for i := range m.tickets {
		switch m.tickets[i].Status {
		case domain.TicketStatusOpen:
			counts.Open++
		case domain.TicketStatusInProgress:
			counts.InProgress++
		case domain.TicketStatusResolved:
			counts.Resolved++
		case domain.TicketStatusClosed:
			counts.Closed++
		}
	}
	return m.list(q, func(*domain.Ticket) bool { return true }), counts, nil
}

func (m *memTracking) QueueAuthors(_ context.Context, limit int) ([]string, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	logins := []string{}
	for i := range m.tickets {
		if login, _ := loginOf(m.tickets[i].AuthorID); !slices.Contains(logins, login) {
			logins = append(logins, login)
		}
	}
	slices.Sort(logins)
	return logins[:min(len(logins), limit)], nil
}

func (m *memTracking) Thread(_ context.Context, ticket domain.TicketID, before domain.TicketEntryID, limit int) (domain.ThreadPage, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.ThreadPage{}, m.err
	}
	var rows []domain.TicketEntry
	for i := range m.entries {
		if e := &m.entries[i]; e.TicketID == ticket && (before == 0 || e.ID < before) {
			rows = append(rows, *e)
		}
	}
	page := domain.ThreadPage{Entries: []domain.TicketEntry{}, HasMore: len(rows) > limit}
	page.Entries = append(page.Entries, rows[max(0, len(rows)-limit):]...)
	return page, nil
}

// addEntry stores e on the ticket at index i and moves its last update.
func (m *memTracking) addEntry(i int, e domain.TicketEntry) domain.TicketEntry {
	e.ID = domain.TicketEntryID(len(m.entries) + 1)
	e.TicketID = m.tickets[i].ID
	e.ActorLogin, e.ActorRole = loginOf(e.ActorID)
	e.CreatedAt = m.tick()
	m.tickets[i].UpdatedAt = e.CreatedAt
	m.entries = append(m.entries, e)
	return e
}

func (m *memTracking) AddMessage(_ context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.TicketEntry{}, m.err
	}
	i := m.find(ticket)
	switch {
	case i < 0:
		return domain.TicketEntry{}, fmt.Errorf("mem: %w", domain.ErrNotFound)
	case m.tickets[i].Status == domain.TicketStatusClosed:
		return domain.TicketEntry{}, fmt.Errorf("mem: %w", domain.ErrTicketClosed)
	}
	return m.addEntry(i, domain.TicketEntry{ActorID: actor, Kind: domain.EntryKindMessage, Body: text}), nil
}

func (m *memTracking) ChangeStatus(_ context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error) {
	m.lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.TicketEntry{}, m.err
	}
	i := m.find(ticket)
	switch {
	case i < 0:
		return domain.TicketEntry{}, fmt.Errorf("mem: %w", domain.ErrNotFound)
	case m.tickets[i].Status != from:
		return domain.TicketEntry{}, fmt.Errorf("mem: %w", domain.ErrStaleStatus)
	}
	m.tickets[i].Status = to
	return m.addEntry(i, domain.TicketEntry{ActorID: actor, Kind: domain.EntryKindStatusChange, From: from, To: to}), nil
}

// seed stores a ticket of author directly, as if it had been created.
func (m *memTracking) seed(author domain.UserID, typ domain.TicketType, title string) domain.Ticket {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := domain.Ticket{
		ID: domain.TicketID(len(m.tickets) + 1), AuthorID: author, Type: typ,
		Status: domain.TicketStatusOpen, Title: title,
	}
	switch typ {
	case domain.TicketTypeAccess:
		t.System, t.AccessLevel, t.Justification = "Grafana", domain.AccessLevelRead, "Ver os painéis."
	case domain.TicketTypeBug:
		t.System, t.Description, t.Steps, t.Severity = "Portal", "Erro 500.", "1. Abrir\n2. Salvar", domain.SeverityHigh
	case domain.TicketTypeDeploy:
		t.Project, t.Repository, t.Environment = "api", "https://git.example.com/api", domain.EnvironmentProduction
	case domain.TicketTypeOther:
		t.Description = "Como peço acesso à VPN?"
	}
	t.CreatedAt = m.tick()
	t.UpdatedAt = t.CreatedAt
	m.tickets = append(m.tickets, t)
	return t
}

// fail makes every later call return err.
func (m *memTracking) fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// sizes returns the number of stored tickets and entries, to check that a
// refused request wrote nothing.
func (m *memTracking) sizes() (tickets, entries int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tickets), len(m.entries)
}

func (m *memTracking) entry(id domain.TicketEntryID) domain.TicketEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.entries {
		if m.entries[i].ID == id {
			return m.entries[i]
		}
	}
	return domain.TicketEntry{}
}

func (m *memTracking) ticket(id domain.TicketID) domain.Ticket {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tickets[m.find(id)]
}

// trackingApp is the full NewHandler chain with working sessions for the
// four tracking users, the real service.Tickets and security.Security, and
// memTracking as every store.
type trackingApp struct {
	flowApp
	store    *memTracking
	messages *security.RateLimiter // the reply limiter of the app
}

func newTrackingApp(t *testing.T, logOut io.Writer) trackingApp {
	t.Helper()
	if logOut == nil {
		logOut = io.Discard
	}
	logger := slog.New(NewLogHandler(slog.NewJSONHandler(logOut, nil)))
	sessions := newMemSessions()
	for _, u := range trackingUsers {
		sessions.setRole(u.ID, u.Role)
		sessions.setProfile(u.ID, u.Login, time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC))
	}
	store := newMemTracking()
	sec, err := security.New(security.Options{
		Logger:                 logger,
		Sessions:               sessions,
		Tickets:                store,
		RenderError:            WriteError,
		SessionIdleTimeout:     time.Minute,
		SessionAbsoluteTimeout: time.Hour,
		RequestTimeout:         time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	deps := trackingDeps(t, logger, sec, service.NewTicketsWithThread(store, store, sec))
	app, err := NewHandler(deps)
	if err != nil {
		t.Fatal(err)
	}
	return trackingApp{flowApp: flowApp{app: app, sessions: sessions}, store: store, messages: deps.MessageLimiter}
}

// trackingDeps is a complete Deps for the tracking app.
func trackingDeps(t *testing.T, logger *slog.Logger, sec *security.Security, tickets TicketService) Deps {
	t.Helper()
	return Deps{
		Logger: logger, Security: sec, Auth: trackingAuth{}, Users: fakeUsers{},
		Tickets: tickets, TicketLimiter: newTicketLimiter(t, sec),
		MessageLimiter: newMessageLimiter(t, sec),
		StatusLimiter:  newStatusLimiter(t, sec),
	}
}

// newStatusLimiter is the production limiter of status changes: 30 per
// admin in any 60-second window (spec 003 S-9).
func newStatusLimiter(t *testing.T, sec *security.Security) *security.RateLimiter {
	t.Helper()
	l, err := sec.NewRateLimiter(security.StatusChangeLimit, security.StatusChangeWindow)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}
	return l
}

// newMessageLimiter is the production limiter of replies: 30 per user in
// any 60-second window (spec 003 S-9).
func newMessageLimiter(t *testing.T, sec *security.Security) *security.RateLimiter {
	t.Helper()
	l, err := sec.NewRateLimiter(security.MessageCreateLimit, security.MessageCreateWindow)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}
	return l
}

// get sends a GET with the session of cookie and decodes a JSON object.
func (a trackingApp) get(t *testing.T, path string, cookie *http.Cookie) (int, map[string]any) {
	t.Helper()
	rr := a.send(t, http.MethodGet, path, "", cookie, nil)
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET %s: body is not a JSON object: %v (%s)", path, err, rr.Body.String())
	}
	return rr.Code, body
}

// keysOf returns the sorted keys of a JSON object.
func keysOf(v any) []string {
	obj, _ := v.(map[string]any)
	return sortedKeys(obj)
}
