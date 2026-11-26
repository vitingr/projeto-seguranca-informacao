package repository

import (
	"context"
	"database/sql"
	"fmt"

	"backend_si/internal/domain"
)

type Tickets struct {
	db *sql.DB
}

func NewTickets(db *sql.DB) *Tickets {
	return &Tickets{db: db}
}

// Create stores a validated ticket in one parameterized INSERT, so it is
// stored with every field or not at all (BR-6). A field the type does not
// have, and empty notes, are "" in the ticket and stored as NULL. The id,
// the status and the dates come from the database (BR-1, BR-2, S-5): the
// status column is left to its default, so a new ticket is always "aberto"
// whatever t.Status holds. The ticket returned is the one given, with those
// values filled in.
func (r *Tickets) Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error) {
	const q = `INSERT INTO tickets (author_id, type, title, system, access_level, justification,
			description, steps, severity, project, repository, environment, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, status, created_at, updated_at`

	var status string
	err := r.db.QueryRowContext(ctx, q,
		int64(t.AuthorID), string(t.Type), t.Title,
		nullable(t.System), nullable(string(t.AccessLevel)), nullable(t.Justification),
		nullable(t.Description), nullable(t.Steps), nullable(string(t.Severity)),
		nullable(t.Project), nullable(t.Repository), nullable(string(t.Environment)), nullable(t.Notes),
	).Scan(&t.ID, &status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("repository: criar ticket: %w", err)
	}
	t.Status = domain.TicketStatus(status)
	return t, nil
}

// nullable maps "" to NULL, so a field the ticket type does not have is
// stored as NULL, as the tickets_fields_by_type CHECK requires.
func nullable(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
