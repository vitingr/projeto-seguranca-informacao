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
	// Create stores a validated ticket and returns it with the id, status
	// and dates set by the database.
	Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error)
}

type TicketValidator interface {
	// ValidateTicket applies the field rules of a ticket type and returns
	// the trimmed values; errors wrap domain.ErrInvalidInput.
	ValidateTicket(d domain.TicketDraft) (domain.Ticket, error)
}

type InputValidator interface {
	// NormalizeLogin returns the canonical (lowercase) form of a login.
	NormalizeLogin(raw string) string
	ValidateLogin(login string) error
	ValidatePassword(password string) error
	ParseRole(raw string) (domain.Role, error)
}
