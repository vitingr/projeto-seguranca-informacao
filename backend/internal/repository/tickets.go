package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"backend_si/internal/domain"
)

type Tickets struct {
	db *sql.DB
}

func NewTickets(db *sql.DB) *Tickets {
	return &Tickets{db: db}
}

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

const TicketPageSize = 20

const MaxQueueAuthors = 500

const (
	ownListWhere = `WHERE author_id = $1 AND ($2 = '' OR status = $2) AND ($3 = '' OR type = $3)`

	ownListCountQuery = `SELECT count(*) FROM tickets ` + ownListWhere

	ownListQuery = `SELECT id, type, status, title, created_at, updated_at
		FROM tickets ` + ownListWhere + `
		ORDER BY updated_at DESC, id DESC
		LIMIT $4 OFFSET $5`

	queueWhere = `WHERE ($1 = '' OR t.status = $1) AND ($2 = '' OR t.type = $2) AND ($3 = '' OR u.login = $3)`

	queueCountQuery = `SELECT count(*) FROM tickets t JOIN users u ON u.id = t.author_id ` + queueWhere

	queueQuery = `SELECT t.id, t.type, t.status, t.title, t.created_at, t.updated_at, u.login
		FROM tickets t JOIN users u ON u.id = t.author_id ` + queueWhere + `
		ORDER BY t.updated_at DESC, t.id DESC
		LIMIT $4 OFFSET $5`

	statusCountsQuery = `SELECT status, count(*) FROM tickets GROUP BY status`

	queueAuthorsQuery = `SELECT u.login FROM users u
		WHERE EXISTS (SELECT 1 FROM tickets t WHERE t.author_id = u.id)
		ORDER BY u.login COLLATE "C"
		LIMIT $1`
)

func (r *Tickets) ListOwn(ctx context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error) {
	page, err := r.listPage(ctx, ownListCountQuery, ownListQuery, q.Page, false,
		int64(author), string(q.Status), string(q.Type))
	if err != nil {
		return domain.TicketPage{}, fmt.Errorf("repository: list own tickets: %w", err)
	}
	return page, nil
}

func (r *Tickets) ListQueue(ctx context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	page, err := r.listPage(ctx, queueCountQuery, queueQuery, q.Page, true,
		string(q.Status), string(q.Type), q.AuthorLogin)
	if err != nil {
		return domain.TicketPage{}, domain.StatusCounts{}, fmt.Errorf("repository: list queue: %w", err)
	}
	counts, err := r.statusCounts(ctx)
	if err != nil {
		return domain.TicketPage{}, domain.StatusCounts{}, fmt.Errorf("repository: list queue: %w", err)
	}
	return page, counts, nil
}

func (r *Tickets) QueueAuthors(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > MaxQueueAuthors {
		limit = MaxQueueAuthors
	}
	rows, err := r.db.QueryContext(ctx, queueAuthorsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("repository: queue authors: %w", err)
	}
	defer rows.Close()
	logins := make([]string, 0)
	for rows.Next() {
		var login string
		if err := rows.Scan(&login); err != nil {
			return nil, fmt.Errorf("repository: queue authors: %w", err)
		}
		logins = append(logins, login)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: queue authors: %w", err)
	}
	return logins, nil
}

func (r *Tickets) listPage(ctx context.Context, countQuery, pageQuery string, page int, withLogin bool, filters ...any) (domain.TicketPage, error) {
	if page < 1 {
		page = 1
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return domain.TicketPage{}, err
	}
	defer func() { _ = tx.Rollback() }()

	result := domain.TicketPage{Items: make([]domain.TicketSummary, 0), Page: page}
	if countErr := tx.QueryRowContext(ctx, countQuery, filters...).Scan(&result.Total); countErr != nil {
		return domain.TicketPage{}, countErr
	}
	result.Pages = max(1, (result.Total+TicketPageSize-1)/TicketPageSize)

	args := slices.Concat(filters, []any{TicketPageSize, (page - 1) * TicketPageSize})
	rows, err := tx.QueryContext(ctx, pageQuery, args...)
	if err != nil {
		return domain.TicketPage{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var s domain.TicketSummary
		dst := []any{&s.ID, &s.Type, &s.Status, &s.Title, &s.CreatedAt, &s.UpdatedAt}
		if withLogin {
			dst = append(dst, &s.AuthorLogin)
		}
		if err := rows.Scan(dst...); err != nil {
			return domain.TicketPage{}, err
		}
		result.Items = append(result.Items, s)
	}
	if err := rows.Err(); err != nil {
		return domain.TicketPage{}, err
	}
	return result, tx.Commit()
}

func (r *Tickets) statusCounts(ctx context.Context) (domain.StatusCounts, error) {
	rows, err := r.db.QueryContext(ctx, statusCountsQuery)
	if err != nil {
		return domain.StatusCounts{}, err
	}
	defer rows.Close()
	var counts domain.StatusCounts
	for rows.Next() {
		var status domain.TicketStatus
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return domain.StatusCounts{}, err
		}
		switch status {
		case domain.TicketStatusOpen:
			counts.Open = n
		case domain.TicketStatusInProgress:
			counts.InProgress = n
		case domain.TicketStatusResolved:
			counts.Resolved = n
		case domain.TicketStatusClosed:
			counts.Closed = n
		default:
			return domain.StatusCounts{}, fmt.Errorf("unknown status %q in the database", status)
		}
	}
	return counts, rows.Err()
}

func nullable(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
