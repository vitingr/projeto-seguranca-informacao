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
	// AddMessage stores a message and moves the ticket's last update;
	// domain.ErrTicketClosed on a "fechado" ticket.
	AddMessage(ctx context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error)
	// ChangeStatus moves the ticket from status from to to and records it;
	// domain.ErrStaleStatus when from is not its status any more.
	ChangeStatus(ctx context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error)
}

type TicketValidator interface {
	// ValidateTicket applies the field rules of a ticket type and returns
	// the trimmed values; errors wrap domain.ErrInvalidInput.
	ValidateTicket(d domain.TicketDraft) (domain.Ticket, error)
	// ValidateMessage applies the message rule (spec 003 BR-6) and returns
	// the trimmed text; errors wrap domain.ErrInvalidInput.
	ValidateMessage(raw string) (string, error)
	// CheckStatusChange allows only the transitions of spec 003 BR-3:
	// domain.ErrInvalidInput for a pair BR-3 never allows,
	// domain.ErrStaleStatus for a change from "fechado".
	CheckStatusChange(from, to domain.TicketStatus) error
}

type InputValidator interface {
	// NormalizeLogin returns the canonical (lowercase) form of a login.
	NormalizeLogin(raw string) string
	ValidateLogin(login string) error
	ValidatePassword(password string) error
	ParseRole(raw string) (domain.Role, error)
}
