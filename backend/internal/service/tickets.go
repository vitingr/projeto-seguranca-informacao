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

type Tickets struct {
	repo      TicketRepository
	validator TicketValidator
}

func NewTickets(repo TicketRepository, validator TicketValidator) *Tickets {
	return &Tickets{repo: repo, validator: validator}
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
