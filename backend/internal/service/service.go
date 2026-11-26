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
	// Get returns a ticket and its author's login; domain.ErrNotFound when
	// it does not exist.
	Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error)
	// ListOwn returns a page of the tickets of author, whatever q.AuthorLogin.
	ListOwn(ctx context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error)
	// ListQueue returns a page of every author's tickets and the status
	// counts of the whole queue.
	ListQueue(ctx context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error)
	// QueueAuthors returns the logins of the users with tickets, at most limit.
	QueueAuthors(ctx context.Context, limit int) ([]string, error)
}

// TicketEntryRepository reads and writes the timeline of a ticket (spec 003).
type TicketEntryRepository interface {
	// Thread returns the latest limit entries older than before (0: the
	// latest), oldest first, and whether older ones remain.
	Thread(ctx context.Context, ticket domain.TicketID, before domain.TicketEntryID, limit int) (domain.ThreadPage, error)
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
