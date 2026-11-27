package service

import (
	"context"

	"backend_si/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.UserID, error)
	GetByLogin(ctx context.Context, login string) (domain.User, error)
}

type PasswordHasher interface {
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, encodedHash, password string) error
}

type TicketRepository interface {
	Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error)
	Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error)
	ListOwn(ctx context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error)
	ListQueue(ctx context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error)
	QueueAuthors(ctx context.Context, limit int) ([]string, error)
}

type TicketEntryRepository interface {
	Thread(ctx context.Context, ticket domain.TicketID, before domain.TicketEntryID, limit int) (domain.ThreadPage, error)
	AddMessage(ctx context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error)
	ChangeStatus(ctx context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error)
}

type TicketValidator interface {
	ValidateTicket(d domain.TicketDraft) (domain.Ticket, error)
	ValidateMessage(raw string) (string, error)
	CheckStatusChange(from, to domain.TicketStatus) error
}

type InputValidator interface {
	NormalizeLogin(raw string) string
	ValidateLogin(login string) error
	ValidatePassword(password string) error
	ParseRole(raw string) (domain.Role, error)
}
