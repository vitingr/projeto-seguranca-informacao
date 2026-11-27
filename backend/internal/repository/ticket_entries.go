package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"backend_si/internal/domain"
)

const ThreadPageSize = 100

type TicketEntries struct {
	db *sql.DB
}

func NewTicketEntries(db *sql.DB) *TicketEntries {
	return &TicketEntries{db: db}
}

const threadQuery = `SELECT e.id, e.ticket_id, e.actor_id, u.login, u.role, e.kind,
		COALESCE(e.body, ''), COALESCE(e.from_status, ''), COALESCE(e.to_status, ''), e.created_at
	FROM ticket_entries e JOIN users u ON u.id = e.actor_id
	WHERE e.ticket_id = $1 AND ($2::bigint = 0 OR e.id < $2)
	ORDER BY e.id DESC
	LIMIT $3`

func (r *TicketEntries) Thread(ctx context.Context, ticket domain.TicketID, before domain.TicketEntryID, limit int) (domain.ThreadPage, error) {
	if limit <= 0 || limit > ThreadPageSize {
		limit = ThreadPageSize
	}
	if before < 0 {
		before = 0
	}
	rows, err := r.db.QueryContext(ctx, threadQuery, int64(ticket), int64(before), limit+1)
	if err != nil {
		return domain.ThreadPage{}, fmt.Errorf("repository: thread: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.TicketEntry, 0, limit+1)
	for rows.Next() {
		var e domain.TicketEntry
		if err := rows.Scan(&e.ID, &e.TicketID, &e.ActorID, &e.ActorLogin, &e.ActorRole, &e.Kind,
			&e.Body, &e.From, &e.To, &e.CreatedAt); err != nil {
			return domain.ThreadPage{}, fmt.Errorf("repository: thread: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return domain.ThreadPage{}, fmt.Errorf("repository: thread: %w", err)
	}

	page := domain.ThreadPage{HasMore: len(entries) > limit}
	if page.HasMore {
		entries = entries[:limit]
	}
	slices.Reverse(entries)
	page.Entries = entries
	return page, nil
}

const (
	lockForMessageQuery = `UPDATE tickets SET updated_at = GREATEST(clock_timestamp(), updated_at)
		WHERE id = $1 AND status <> 'fechado'
		RETURNING updated_at`

	lockForStatusQuery = `UPDATE tickets SET status = $3, updated_at = GREATEST(clock_timestamp(), updated_at)
		WHERE id = $1 AND status = $2
		RETURNING updated_at`

	ticketExistsQuery = `SELECT EXISTS (SELECT 1 FROM tickets WHERE id = $1)`

	insertEntryQuery = `WITH e AS (
			INSERT INTO ticket_entries (ticket_id, actor_id, kind, body, from_status, to_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, ticket_id, actor_id, kind, body, from_status, to_status, created_at)
		SELECT e.id, e.ticket_id, e.actor_id, u.login, u.role, e.kind,
			COALESCE(e.body, ''), COALESCE(e.from_status, ''), COALESCE(e.to_status, ''), e.created_at
		FROM e JOIN users u ON u.id = e.actor_id`
)

func (r *TicketEntries) AddMessage(ctx context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error) {
	e, err := r.write(ctx, lockForMessageQuery, []any{int64(ticket)}, domain.ErrTicketClosed,
		int64(ticket), int64(actor), string(domain.EntryKindMessage), text, nil, nil)
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("repository: add message: %w", err)
	}
	return e, nil
}

func (r *TicketEntries) ChangeStatus(ctx context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error) {
	e, err := r.write(ctx, lockForStatusQuery, []any{int64(ticket), string(from), string(to)}, domain.ErrStaleStatus,
		int64(ticket), int64(actor), string(domain.EntryKindStatusChange), nil, string(from), string(to))
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("repository: change status: %w", err)
	}
	return e, nil
}

func (r *TicketEntries) write(ctx context.Context, lockQuery string, lockArgs []any, refused error, entry ...any) (domain.TicketEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.TicketEntry{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var at time.Time
	switch lockErr := tx.QueryRowContext(ctx, lockQuery, lockArgs...).Scan(&at); {
	case errors.Is(lockErr, sql.ErrNoRows):
		var exists bool
		if existsErr := tx.QueryRowContext(ctx, ticketExistsQuery, lockArgs[0]).Scan(&exists); existsErr != nil {
			return domain.TicketEntry{}, existsErr
		}
		if !exists {
			return domain.TicketEntry{}, domain.ErrNotFound
		}
		return domain.TicketEntry{}, refused
	case lockErr != nil:
		return domain.TicketEntry{}, lockErr
	}

	var e domain.TicketEntry
	args := slices.Concat(entry, []any{at})
	if insertErr := tx.QueryRowContext(ctx, insertEntryQuery, args...).Scan(&e.ID, &e.TicketID, &e.ActorID,
		&e.ActorLogin, &e.ActorRole, &e.Kind, &e.Body, &e.From, &e.To, &e.CreatedAt); insertErr != nil {
		return domain.TicketEntry{}, insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return domain.TicketEntry{}, commitErr
	}
	return e, nil
}
