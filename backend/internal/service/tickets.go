package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"backend_si/internal/domain"
)

const maxTitleLen = 120 // Unicode code points

var environmentLabels = map[domain.Environment]string{
	domain.EnvironmentDevelopment: "Desenvolvimento",
	domain.EnvironmentStaging:     "Homologação",
	domain.EnvironmentProduction:  "Produção",
}

var errNoAuthor = errors.New("no author")

const (
	threadPageSize  = 100
	maxQueueAuthors = 500
)

var errNoThread = errors.New("no ticket entry repository")

type Tickets struct {
	repo      TicketRepository
	entries   TicketEntryRepository
	validator TicketValidator
}

func NewTickets(repo TicketRepository, validator TicketValidator) *Tickets {
	return &Tickets{repo: repo, validator: validator}
}

func NewTicketsWithThread(repo TicketRepository, entries TicketEntryRepository, validator TicketValidator) *Tickets {
	return &Tickets{repo: repo, entries: entries, validator: validator}
}

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

func (s *Tickets) ListQueue(ctx context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	page, counts, err := s.repo.ListQueue(ctx, q)
	if err != nil {
		return domain.TicketPage{}, domain.StatusCounts{}, fmt.Errorf("service: list queue: %w", err)
	}
	return page, counts, nil
}

func (s *Tickets) Authors(ctx context.Context) ([]string, error) {
	logins, err := s.repo.QueueAuthors(ctx, maxQueueAuthors)
	if err != nil {
		return nil, fmt.Errorf("service: queue authors: %w", err)
	}
	return logins, nil
}

func (s *Tickets) Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error) {
	t, login, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Ticket{}, "", fmt.Errorf("service: get ticket: %w", err)
	}
	return t, login, nil
}

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
