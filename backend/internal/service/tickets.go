package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"backend_si/internal/domain"
)

// maxTitleLen is the title limit of spec 001-ticket-builder (BR-3), in
// Unicode code points, the same as the tickets.title CHECK.
const maxTitleLen = 120

// environmentLabels are the labels the user sees for each environment; the
// deploy title uses them (BR-3). They match the frontend labels.
var environmentLabels = map[domain.Environment]string{
	domain.EnvironmentDevelopment: "Desenvolvimento",
	domain.EnvironmentStaging:     "Homologação",
	domain.EnvironmentProduction:  "Produção",
}

var errNoAuthor = errors.New("no author")

// Page sizes of spec 003 (S-14): the thread page (plan P-4) and the author
// filter list (plan P-18). The repository bounds them too.
const (
	threadPageSize  = 100
	maxQueueAuthors = 500
)

// errNoThread is a wiring error: a Tickets built without its entry
// repository cannot read or write threads. It is not a client error, so
// the answer is a 500 (fail closed).
var errNoThread = errors.New("no ticket entry repository")

type Tickets struct {
	repo      TicketRepository
	entries   TicketEntryRepository
	validator TicketValidator
}

func NewTickets(repo TicketRepository, validator TicketValidator) *Tickets {
	return &Tickets{repo: repo, validator: validator}
}

// NewTicketsWithThread is NewTickets with the repository of the ticket
// timelines (spec 003), which threads, replies and status changes need.
func NewTicketsWithThread(repo TicketRepository, entries TicketEntryRepository, validator TicketValidator) *Tickets {
	return &Tickets{repo: repo, entries: entries, validator: validator}
}

// ListOwn returns a page of "Meus tickets" of author, the session user
// (BR-1, S-3): the author is a parameter, never part of the query, whose
// author filter is dropped. A missing author is a server error.
func (s *Tickets) ListOwn(ctx context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error) {
	if author <= 0 {
		return domain.TicketPage{}, fmt.Errorf("service: list own tickets: %w", errNoAuthor)
	}
	q.AuthorLogin = ""
	page, err := s.repo.ListOwn(ctx, author, q)
	if err != nil {
		return domain.TicketPage{}, fmt.Errorf("service: list own tickets: %w", err)
	}
	return page, nil
}

// ListQueue returns a page of the admin queue and the status counts of the
// whole queue (AC-4.3 to AC-4.5).
func (s *Tickets) ListQueue(ctx context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	page, counts, err := s.repo.ListQueue(ctx, q)
	if err != nil {
		return domain.TicketPage{}, domain.StatusCounts{}, fmt.Errorf("service: list queue: %w", err)
	}
	return page, counts, nil
}

// Authors returns the logins of the users with at least one ticket, for the
// author filter of the queue (AC-4.4, plan P-18).
func (s *Tickets) Authors(ctx context.Context) ([]string, error) {
	logins, err := s.repo.QueueAuthors(ctx, maxQueueAuthors)
	if err != nil {
		return nil, fmt.Errorf("service: queue authors: %w", err)
	}
	return logins, nil
}

// Get returns a ticket and its author's login. Access was decided before,
// by security.TicketAccess, which gives the id.
func (s *Tickets) Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error) {
	t, login, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Ticket{}, "", fmt.Errorf("service: get ticket: %w", err)
	}
	return t, login, nil
}

// Thread returns a page of the thread of a ticket: the latest entries when
// before is 0, else the entries older than before (plan P-4).
func (s *Tickets) Thread(ctx context.Context, id domain.TicketID, before domain.TicketEntryID) (domain.ThreadPage, error) {
	if s.entries == nil {
		return domain.ThreadPage{}, fmt.Errorf("service: thread: %w", errNoThread)
	}
	page, err := s.entries.Thread(ctx, id, before, threadPageSize)
	if err != nil {
		return domain.ThreadPage{}, fmt.Errorf("service: thread: %w", err)
	}
	return page, nil
}

// Reply stores a message of actor, the session user, on a ticket TicketAccess
// let through (BR-6). The text is validated first and stored trimmed; an
// invalid text wraps domain.ErrInvalidInput and nothing is stored. The
// actor is a parameter, never part of the request (S-8). A "fechado" ticket
// is domain.ErrTicketClosed (BR-5).
func (s *Tickets) Reply(ctx context.Context, actor domain.UserID, ticket domain.TicketID, raw string) (domain.TicketEntry, error) {
	if s.entries == nil {
		return domain.TicketEntry{}, fmt.Errorf("service: reply: %w", errNoThread)
	}
	if actor <= 0 {
		return domain.TicketEntry{}, fmt.Errorf("service: reply: %w", errNoAuthor)
	}
	text, err := s.validator.ValidateMessage(raw)
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("service: reply: %w", err)
	}
	e, err := s.entries.AddMessage(ctx, ticket, actor, text)
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("service: reply: %w", err)
	}
	return e, nil
}

// ChangeStatus moves a ticket from from, the status the admin saw, to to
// (BR-2, BR-3, BR-7). Only admins reach it (route policy). The pair is
// checked first, and a refused pair reaches no repository:
// domain.ErrInvalidInput (422) for a pair BR-3 never allows,
// domain.ErrStaleStatus (409) for a change from "fechado". The repository
// then applies it only if from is still the status (domain.ErrStaleStatus
// otherwise).
func (s *Tickets) ChangeStatus(ctx context.Context, actor domain.UserID, ticket domain.TicketID, from, to domain.TicketStatus) (domain.TicketEntry, error) {
	if s.entries == nil {
		return domain.TicketEntry{}, fmt.Errorf("service: change status: %w", errNoThread)
	}
	if actor <= 0 {
		return domain.TicketEntry{}, fmt.Errorf("service: change status: %w", errNoAuthor)
	}
	if err := s.validator.CheckStatusChange(from, to); err != nil {
		return domain.TicketEntry{}, fmt.Errorf("service: change status: %w", err)
	}
	e, err := s.entries.ChangeStatus(ctx, ticket, actor, from, to)
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("service: change status: %w", err)
	}
	return e, nil
}

// Create validates a decoded ticket request, gives it its title (BR-3), the
// author of the session and the status "aberto" (BR-1, S-4, S-5), and
// stores it. The author is a parameter, never part of the request. Invalid
// values wrap domain.ErrInvalidInput and nothing is stored. A missing author
// is a server error (fail closed), not invalid input.
func (s *Tickets) Create(ctx context.Context, author domain.UserID, draft domain.TicketDraft) (domain.Ticket, error) {
	if author <= 0 {
		return domain.Ticket{}, fmt.Errorf("service: create ticket: %w", errNoAuthor)
	}
	t, err := s.validator.ValidateTicket(draft)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("service: create ticket: %w", err)
	}
	title, err := ticketTitle(t)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("service: create ticket: %w", err)
	}
	t.ID, t.AuthorID, t.Status, t.Title = 0, author, domain.TicketStatusOpen, title

	stored, err := s.repo.Create(ctx, t)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("service: create ticket: %w", err)
	}
	return stored, nil
}

// ticketTitle builds the title of a validated ticket (BR-3): "outro" keeps
// the typed title; the other types get "Acesso a <sistema>", "Bug em
// <sistema>" or "Deploy de <projeto> em <ambiente>", with the environment
// label, cut to maxTitleLen code points. Whitespace left at the end by the
// cut is removed, with the same notion of whitespace as the trimming of the
// fields (unicode.IsSpace).
func ticketTitle(t domain.Ticket) (string, error) {
	var title string
	switch t.Type {
	case domain.TicketTypeAccess:
		title = "Acesso a " + t.System
	case domain.TicketTypeBug:
		title = "Bug em " + t.System
	case domain.TicketTypeDeploy:
		label, ok := environmentLabels[t.Environment]
		if !ok {
			return "", fmt.Errorf("title: unknown environment: %w", domain.ErrInvalidInput)
		}
		title = "Deploy de " + t.Project + " em " + label
	case domain.TicketTypeOther:
		title = t.Title
	default:
		return "", fmt.Errorf("title: unknown type: %w", domain.ErrInvalidInput)
	}
	return strings.TrimRightFunc(cutCodePoints(title, maxTitleLen), unicode.IsSpace), nil
}

// cutCodePoints keeps the first n code points of s, so a multi-byte
// character is never split.
func cutCodePoints(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
