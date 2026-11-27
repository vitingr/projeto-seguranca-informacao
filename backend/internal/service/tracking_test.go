package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"backend_si/internal/domain"
)

// Tests of the spec 003 methods of Tickets. The fakes of tickets_test.go
// gain here the methods the grown interfaces need, so that file stays as
// it is; they are not used by these tests.

var errNotInFake = errors.New("not implemented by this fake")

func (m *memTickets) Get(context.Context, domain.TicketID) (domain.Ticket, string, error) {
	return domain.Ticket{}, "", errNotInFake
}

func (m *memTickets) ListOwn(context.Context, domain.UserID, domain.TicketListQuery) (domain.TicketPage, error) {
	return domain.TicketPage{}, errNotInFake
}

func (m *memTickets) ListQueue(context.Context, domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	return domain.TicketPage{}, domain.StatusCounts{}, errNotInFake
}

func (m *memTickets) QueueAuthors(context.Context, int) ([]string, error) {
	return nil, errNotInFake
}

// trackingCall is one call a trackingRepo received.
type trackingCall struct {
	method string
	author domain.UserID
	ticket domain.TicketID
	before domain.TicketEntryID
	limit  int
	query  domain.TicketListQuery
	actor  domain.UserID
	text   string
	from   domain.TicketStatus
	to     domain.TicketStatus
}

// trackingRepo is a TicketRepository and a TicketEntryRepository that
// records every call and answers with fixed values, or err when set.
type trackingRepo struct {
	calls  []trackingCall
	err    error
	page   domain.TicketPage
	counts domain.StatusCounts
	ticket domain.Ticket
	login  string
	thread domain.ThreadPage
	logins []string
}

func (r *trackingRepo) record(c trackingCall) error {
	r.calls = append(r.calls, c)
	return r.err
}

func (r *trackingRepo) Create(context.Context, domain.Ticket) (domain.Ticket, error) {
	return domain.Ticket{}, r.record(trackingCall{method: "Create"})
}

func (r *trackingRepo) Get(_ context.Context, id domain.TicketID) (domain.Ticket, string, error) {
	if err := r.record(trackingCall{method: "Get", ticket: id}); err != nil {
		return domain.Ticket{}, "", err
	}
	return r.ticket, r.login, nil
}

func (r *trackingRepo) ListOwn(_ context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error) {
	if err := r.record(trackingCall{method: "ListOwn", author: author, query: q}); err != nil {
		return domain.TicketPage{}, err
	}
	return r.page, nil
}

func (r *trackingRepo) ListQueue(_ context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	if err := r.record(trackingCall{method: "ListQueue", query: q}); err != nil {
		return domain.TicketPage{}, domain.StatusCounts{}, err
	}
	return r.page, r.counts, nil
}

func (r *trackingRepo) QueueAuthors(_ context.Context, limit int) ([]string, error) {
	if err := r.record(trackingCall{method: "QueueAuthors", limit: limit}); err != nil {
		return nil, err
	}
	return r.logins, nil
}

func (r *trackingRepo) Thread(_ context.Context, ticket domain.TicketID, before domain.TicketEntryID, limit int) (domain.ThreadPage, error) {
	if err := r.record(trackingCall{method: "Thread", ticket: ticket, before: before, limit: limit}); err != nil {
		return domain.ThreadPage{}, err
	}
	return r.thread, nil
}

var trackingTime = time.Date(2026, 11, 26, 14, 2, 0, 0, time.UTC)

func newTrackingService(repo *trackingRepo) *Tickets {
	return NewTicketsWithThread(repo, repo, &stubValidator{})
}

func TestListOwnTakesTheAuthorFromTheCaller(t *testing.T) {
	t.Parallel()

	repo := &trackingRepo{page: domain.TicketPage{
		Items: []domain.TicketSummary{{ID: 7, Type: domain.TicketTypeBug, Status: domain.TicketStatusOpen, Title: "Bug em Portal", CreatedAt: trackingTime, UpdatedAt: trackingTime}},
		Total: 1, Page: 1, Pages: 1,
	}}
	s := newTrackingService(repo)
	// S-3: an author login in the query never chooses whose list it is.
	q := domain.TicketListQuery{Status: domain.TicketStatusOpen, Type: domain.TicketTypeBug, AuthorLogin: "outra.pessoa", Page: 2}
	got, err := s.ListOwn(t.Context(), author, q)
	if err != nil {
		t.Fatalf("ListOwn: %v", err)
	}
	want := trackingCall{method: "ListOwn", author: author, query: domain.TicketListQuery{Status: domain.TicketStatusOpen, Type: domain.TicketTypeBug, Page: 2}}
	if len(repo.calls) != 1 || repo.calls[0] != want {
		t.Errorf("repository calls = %+v, want [%+v]", repo.calls, want)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].ID != 7 {
		t.Errorf("ListOwn = %+v, want the repository's page", got)
	}
}

func TestListOwnWithoutAuthorIsAServerError(t *testing.T) {
	t.Parallel()

	for _, a := range []domain.UserID{0, -1} {
		repo := &trackingRepo{}
		_, err := newTrackingService(repo).ListOwn(t.Context(), a, domain.TicketListQuery{Page: 1})
		if !errors.Is(err, errNoAuthor) || errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("author %d: error = %v, want errNoAuthor (500)", a, err)
		}
		if len(repo.calls) != 0 {
			t.Errorf("author %d: repository called %+v", a, repo.calls)
		}
	}
}

func TestTrackingReadsPassThrough(t *testing.T) {
	t.Parallel()

	repo := &trackingRepo{
		page:   domain.TicketPage{Items: []domain.TicketSummary{{ID: 3, AuthorLogin: "ana.lima"}}, Total: 41, Page: 3, Pages: 3},
		counts: domain.StatusCounts{Open: 4, InProgress: 2, Resolved: 9, Closed: 1},
		ticket: domain.Ticket{ID: 9, AuthorID: author, Type: domain.TicketTypeOther, Title: "Dúvida"},
		login:  "ana.lima",
		thread: domain.ThreadPage{Entries: []domain.TicketEntry{{ID: 5, TicketID: 9, Kind: domain.EntryKindMessage, Body: "Oi"}}, HasMore: true},
		logins: []string{"ana.lima", "maria.souza"},
	}
	s := newTrackingService(repo)
	ctx := t.Context()

	q := domain.TicketListQuery{Status: domain.TicketStatusClosed, AuthorLogin: "ana.lima", Page: 3}
	page, counts, err := s.ListQueue(ctx, q)
	if err != nil || page.Total != 41 || counts != repo.counts {
		t.Errorf("ListQueue = %+v, %+v, %v", page, counts, err)
	}
	logins, err := s.Authors(ctx)
	if err != nil || strings.Join(logins, ",") != "ana.lima,maria.souza" {
		t.Errorf("Authors = %v, %v", logins, err)
	}
	ticket, login, err := s.Get(ctx, 9)
	if err != nil || ticket.ID != 9 || login != "ana.lima" {
		t.Errorf("Get = %+v, %q, %v", ticket, login, err)
	}
	thread, err := s.Thread(ctx, 9, 120)
	if err != nil || !thread.HasMore || len(thread.Entries) != 1 {
		t.Errorf("Thread = %+v, %v", thread, err)
	}

	want := []trackingCall{
		{method: "ListQueue", query: q},
		{method: "QueueAuthors", limit: 500},
		{method: "Get", ticket: 9},
		{method: "Thread", ticket: 9, before: 120, limit: 100},
	}
	if fmt.Sprint(repo.calls) != fmt.Sprint(want) {
		t.Errorf("repository calls = %+v, want %+v", repo.calls, want)
	}
}

func TestTrackingReadsWrapErrors(t *testing.T) {
	t.Parallel()

	causes := []error{
		fmt.Errorf("repository: get ticket: %w", domain.ErrNotFound),
		errors.New("connection reset"),
	}
	for _, cause := range causes {
		repo := &trackingRepo{err: cause}
		s := newTrackingService(repo)
		ctx := t.Context()
		_, err1 := s.ListOwn(ctx, author, domain.TicketListQuery{Page: 1})
		_, _, err2 := s.ListQueue(ctx, domain.TicketListQuery{Page: 1})
		_, err3 := s.Authors(ctx)
		_, _, err4 := s.Get(ctx, 9)
		_, err5 := s.Thread(ctx, 9, 0)
		for i, err := range []error{err1, err2, err3, err4, err5} {
			if !errors.Is(err, cause) || !strings.HasPrefix(err.Error(), "service: ") {
				t.Errorf("call %d with %v: error = %v, want it wrapped by the service", i+1, cause, err)
			}
		}
	}
}

func TestThreadWithoutEntryRepositoryFailsClosed(t *testing.T) {
	t.Parallel()

	s := NewTickets(&memTickets{}, &stubValidator{})
	_, err := s.Thread(t.Context(), 9, 0)
	if !errors.Is(err, errNoThread) || errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Thread = %v, want errNoThread (500)", err)
	}
}

func (s *stubValidator) ValidateMessage(string) (string, error) { return "", errNotInFake }

func (s *stubValidator) CheckStatusChange(domain.TicketStatus, domain.TicketStatus) error {
	return errNotInFake
}

func (r *trackingRepo) AddMessage(_ context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error) {
	if err := r.record(trackingCall{method: "AddMessage", ticket: ticket, actor: actor, text: text}); err != nil {
		return domain.TicketEntry{}, err
	}
	return domain.TicketEntry{ID: 77, TicketID: ticket, ActorID: actor, Kind: domain.EntryKindMessage, Body: text, CreatedAt: trackingTime}, nil
}

func (r *trackingRepo) ChangeStatus(_ context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error) {
	if err := r.record(trackingCall{method: "ChangeStatus", ticket: ticket, actor: actor, from: from, to: to}); err != nil {
		return domain.TicketEntry{}, err
	}
	return domain.TicketEntry{ID: 78, TicketID: ticket, ActorID: actor, Kind: domain.EntryKindStatusChange, From: from, To: to, CreatedAt: trackingTime}, nil
}

// orderValidator is a TicketValidator that records the order of the calls
// in log, shared with the repository fake through record.
type orderValidator struct {
	repo      *trackingRepo
	text      string
	textErr   error
	statusErr error
}

func (v *orderValidator) ValidateTicket(domain.TicketDraft) (domain.Ticket, error) {
	return domain.Ticket{}, errNotInFake
}

func (v *orderValidator) ValidateMessage(raw string) (string, error) {
	v.repo.calls = append(v.repo.calls, trackingCall{method: "ValidateMessage", text: raw})
	return v.text, v.textErr
}

func (v *orderValidator) CheckStatusChange(from, to domain.TicketStatus) error {
	v.repo.calls = append(v.repo.calls, trackingCall{method: "CheckStatusChange", from: from, to: to})
	return v.statusErr
}

func methods(calls []trackingCall) string {
	names := make([]string, 0, len(calls))
	for i := range calls {
		names = append(names, calls[i].method)
	}
	return strings.Join(names, ",")
}

const adminActor domain.UserID = 5

func TestReplyValidatesThenStores(t *testing.T) {
	t.Parallel()

	repo := &trackingRepo{}
	v := &orderValidator{repo: repo, text: "Oi, tudo bem?"}
	s := NewTicketsWithThread(repo, repo, v)
	got, err := s.Reply(t.Context(), author, 9, "  Oi, tudo bem?\n")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if methods(repo.calls) != "ValidateMessage,AddMessage" {
		t.Fatalf("calls = %s, want the rule first, then the store", methods(repo.calls))
	}
	// The trimmed text, the session actor and the ticket of TicketAccess.
	want := trackingCall{method: "AddMessage", ticket: 9, actor: author, text: "Oi, tudo bem?"}
	if repo.calls[1] != want {
		t.Errorf("AddMessage call = %+v, want %+v", repo.calls[1], want)
	}
	if got.ID != 77 || got.Body != "Oi, tudo bem?" || got.ActorID != author {
		t.Errorf("Reply = %+v, want the stored entry", got)
	}
}

func TestReplyRefusals(t *testing.T) {
	t.Parallel()

	invalid := fmt.Errorf("security: validate message: too long: %w", domain.ErrInvalidInput)
	closed := fmt.Errorf("repository: add message: %w", domain.ErrTicketClosed)
	tests := []struct {
		name    string
		actor   domain.UserID
		textErr error
		repoErr error
		want    error
		calls   string
	}{
		{"invalid text stores nothing", author, invalid, nil, domain.ErrInvalidInput, "ValidateMessage"},
		{"closed ticket", author, nil, closed, domain.ErrTicketClosed, "ValidateMessage,AddMessage"},
		{"no actor is a server error", 0, nil, nil, errNoAuthor, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &trackingRepo{}
			v := &orderValidator{repo: repo, text: "Oi", textErr: tt.textErr}
			s := NewTicketsWithThread(repo, repo, v)
			if tt.repoErr != nil {
				s = NewTicketsWithThread(repo, &failingEntries{trackingRepo: repo, err: tt.repoErr}, v)
			}
			got, err := s.Reply(t.Context(), tt.actor, 9, "Oi")
			if !errors.Is(err, tt.want) || !strings.HasPrefix(err.Error(), "service: reply: ") {
				t.Errorf("Reply = %v, want %v wrapped by the service", err, tt.want)
			}
			if got != (domain.TicketEntry{}) {
				t.Errorf("Reply = %+v with an error, want the zero entry", got)
			}
			if methods(repo.calls) != tt.calls {
				t.Errorf("calls = %q, want %q", methods(repo.calls), tt.calls)
			}
		})
	}
}

// failingEntries answers every write of the embedded fake with err, after
// recording it.
type failingEntries struct {
	*trackingRepo
	err error
}

func (f *failingEntries) AddMessage(ctx context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error) {
	_, _ = f.trackingRepo.AddMessage(ctx, ticket, actor, text)
	return domain.TicketEntry{}, f.err
}

func (f *failingEntries) ChangeStatus(ctx context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error) {
	_, _ = f.trackingRepo.ChangeStatus(ctx, ticket, actor, from, to)
	return domain.TicketEntry{}, f.err
}

func TestChangeStatusChecksThePairThenStores(t *testing.T) {
	t.Parallel()

	repo := &trackingRepo{}
	s := NewTicketsWithThread(repo, repo, &orderValidator{repo: repo})
	got, err := s.ChangeStatus(t.Context(), adminActor, 9, domain.TicketStatusOpen, domain.TicketStatusInProgress)
	if err != nil {
		t.Fatalf("ChangeStatus: %v", err)
	}
	if methods(repo.calls) != "CheckStatusChange,ChangeStatus" {
		t.Fatalf("calls = %s, want the rule first, then the store", methods(repo.calls))
	}
	want := trackingCall{method: "ChangeStatus", ticket: 9, actor: adminActor, from: domain.TicketStatusOpen, to: domain.TicketStatusInProgress}
	if repo.calls[1] != want {
		t.Errorf("ChangeStatus call = %+v, want %+v", repo.calls[1], want)
	}
	if got.ID != 78 || got.To != domain.TicketStatusInProgress || got.ActorID != adminActor {
		t.Errorf("ChangeStatus = %+v, want the stored entry", got)
	}
}

func TestChangeStatusRefusals(t *testing.T) {
	t.Parallel()

	forbidden := fmt.Errorf("security: status change: transition not allowed: %w", domain.ErrInvalidInput)
	fromClosed := fmt.Errorf("security: status change: %w", domain.ErrStaleStatus)
	stale := fmt.Errorf("repository: change status: %w", domain.ErrStaleStatus)
	tests := []struct {
		name      string
		actor     domain.UserID
		statusErr error
		repoErr   error
		want      error
		calls     string
	}{
		{"pair BR-3 never allows", adminActor, forbidden, nil, domain.ErrInvalidInput, "CheckStatusChange"},
		{"change from fechado", adminActor, fromClosed, nil, domain.ErrStaleStatus, "CheckStatusChange"},
		{"stale status in the database", adminActor, nil, stale, domain.ErrStaleStatus, "CheckStatusChange,ChangeStatus"},
		{"no actor is a server error", 0, nil, nil, errNoAuthor, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &trackingRepo{}
			v := &orderValidator{repo: repo, statusErr: tt.statusErr}
			s := NewTicketsWithThread(repo, repo, v)
			if tt.repoErr != nil {
				s = NewTicketsWithThread(repo, &failingEntries{trackingRepo: repo, err: tt.repoErr}, v)
			}
			got, err := s.ChangeStatus(t.Context(), tt.actor, 9, domain.TicketStatusOpen, domain.TicketStatusResolved)
			if !errors.Is(err, tt.want) || !strings.HasPrefix(err.Error(), "service: change status: ") {
				t.Errorf("ChangeStatus = %v, want %v wrapped by the service", err, tt.want)
			}
			if got != (domain.TicketEntry{}) {
				t.Errorf("ChangeStatus = %+v with an error, want the zero entry", got)
			}
			if methods(repo.calls) != tt.calls {
				t.Errorf("calls = %q, want %q", methods(repo.calls), tt.calls)
			}
		})
	}
}

func TestWritesWithoutEntryRepositoryFailClosed(t *testing.T) {
	t.Parallel()

	s := NewTickets(&memTickets{}, &stubValidator{})
	_, err1 := s.Reply(t.Context(), author, 9, "Oi")
	_, err2 := s.ChangeStatus(t.Context(), adminActor, 9, domain.TicketStatusOpen, domain.TicketStatusClosed)
	for _, err := range []error{err1, err2} {
		if !errors.Is(err, errNoThread) || errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("write = %v, want errNoThread (500)", err)
		}
	}
}

// TestWritesWithTheSecurityRules wires the real security rules, so the
// service and security.Security agree on the message rule and the BR-3
// codes (422 for a forbidden pair, 409 from "fechado").
func TestWritesWithTheSecurityRules(t *testing.T) {
	t.Parallel()

	repo := &trackingRepo{}
	s := NewTicketsWithThread(repo, repo, newSecurity(t))
	ctx := t.Context()

	if got, err := s.Reply(ctx, author, 9, "\n  Oi, ainda está acontecendo.  \n"); err != nil || got.Body != "Oi, ainda está acontecendo." {
		t.Errorf("Reply = %+v, %v; want the trimmed text stored", got, err)
	}
	for _, raw := range []string{"", "   \n", strings.Repeat("a", 2001), "Oi\x00"} {
		if _, err := s.Reply(ctx, author, 9, raw); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Reply(%q) = %v, want ErrInvalidInput", raw, err)
		}
	}
	if _, err := s.ChangeStatus(ctx, adminActor, 9, domain.TicketStatusOpen, domain.TicketStatusResolved); !errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrConflict) {
		t.Errorf("aberto → resolvido = %v, want ErrInvalidInput (422)", err)
	}
	if _, err := s.ChangeStatus(ctx, adminActor, 9, domain.TicketStatusOpen, domain.TicketStatusOpen); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("aberto → aberto = %v, want ErrInvalidInput (422)", err)
	}
	if _, err := s.ChangeStatus(ctx, adminActor, 9, domain.TicketStatusClosed, domain.TicketStatusOpen); !errors.Is(err, domain.ErrStaleStatus) {
		t.Errorf("fechado → aberto = %v, want ErrStaleStatus (409)", err)
	}
	if _, err := s.ChangeStatus(ctx, adminActor, 9, domain.TicketStatusResolved, domain.TicketStatusInProgress); err != nil {
		t.Errorf("resolvido → em_andamento = %v, want allowed", err)
	}
	// Only the accepted reply and the allowed change reached the store.
	if got := methods(repo.calls); got != "AddMessage,ChangeStatus" {
		t.Errorf("store calls = %s, want AddMessage,ChangeStatus", got)
	}
}
