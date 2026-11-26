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
