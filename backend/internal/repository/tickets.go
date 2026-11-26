package repository

import (
	"context"
	"database/sql"
	"errors"
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

// TicketAuthor returns the author of a ticket, for the ownership check of
// security.TicketAccess (spec 003 S-2). A ticket that does not exist is
// domain.ErrNotFound. It reads one indexed column, so a missing ticket and
// someone else's ticket cost the same.
func (r *Tickets) TicketAuthor(ctx context.Context, id domain.TicketID) (domain.UserID, error) {
	const q = `SELECT author_id FROM tickets WHERE id = $1`

	var author domain.UserID
	err := r.db.QueryRowContext(ctx, q, int64(id)).Scan(&author)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("repository: ticket author: %w", domain.ErrNotFound)
	case err != nil:
		return 0, fmt.Errorf("repository: ticket author: %w", err)
	}
	return author, nil
}

// Get returns a ticket with every field of its type, as stored (NULL reads
// back as ""), and its author's login. A ticket that does not exist is
// domain.ErrNotFound.
func (r *Tickets) Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error) {
	const q = `SELECT t.id, t.author_id, t.type, t.status, t.title, t.system, t.access_level,
			t.justification, t.description, t.steps, t.severity, t.project, t.repository,
			t.environment, t.notes, t.created_at, t.updated_at, u.login
		FROM tickets t JOIN users u ON u.id = t.author_id
		WHERE t.id = $1`

	var (
		t                                                 domain.Ticket
		login                                             string
		system, level, justification, description, steps  sql.NullString
		severity, project, repository, environment, notes sql.NullString
	)
	err := r.db.QueryRowContext(ctx, q, int64(id)).Scan(
		&t.ID, &t.AuthorID, &t.Type, &t.Status, &t.Title, &system, &level,
		&justification, &description, &steps, &severity, &project, &repository,
		&environment, &notes, &t.CreatedAt, &t.UpdatedAt, &login,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.Ticket{}, "", fmt.Errorf("repository: get ticket: %w", domain.ErrNotFound)
	case err != nil:
		return domain.Ticket{}, "", fmt.Errorf("repository: get ticket: %w", err)
	}
	t.System, t.AccessLevel, t.Justification = system.String, domain.AccessLevel(level.String), justification.String
	t.Description, t.Steps, t.Severity = description.String, steps.String, domain.Severity(severity.String)
	t.Project, t.Repository, t.Environment = project.String, repository.String, domain.Environment(environment.String)
	t.Notes = notes.String
	return t, login, nil
}

// nullable maps "" to NULL, so a field the ticket type does not have is
// stored as NULL, as the tickets_fields_by_type CHECK requires.
func nullable(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
