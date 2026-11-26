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

// ThreadPageSize is the number of entries of one page of a thread (plan
// P-4, spec S-14).
const ThreadPageSize = 100

// TicketEntries stores the timeline of a ticket (spec 003, plan P-3):
// messages and status changes, never edited or deleted (BR-4).
type TicketEntries struct {
	db *sql.DB
}

func NewTicketEntries(db *sql.DB) *TicketEntries {
	return &TicketEntries{db: db}
}

// threadQuery reads one page of a thread, newest first, so the LIMIT keeps
// the latest entries; before = 0 means "from the latest". It reads one more
// entry than the page holds, to tell whether older entries remain.
const threadQuery = `SELECT e.id, e.ticket_id, e.actor_id, u.login, u.role, e.kind,
		COALESCE(e.body, ''), COALESCE(e.from_status, ''), COALESCE(e.to_status, ''), e.created_at
	FROM ticket_entries e JOIN users u ON u.id = e.actor_id
	WHERE e.ticket_id = $1 AND ($2::bigint = 0 OR e.id < $2)
	ORDER BY e.id DESC
	LIMIT $3`

// Thread returns the entries of ticket older than before (all of them when
// before is 0), the latest limit of them, oldest first, with each actor's
// login and role. HasMore is true when older entries remain. A limit
// outside 1 to ThreadPageSize is ThreadPageSize. Entries are ordered by id:
// every write locks the ticket first, so ids of one ticket follow the
// commit order.
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

// The writes of a thread (plan section 4). Each one is one transaction that
// starts by locking the ticket row with its UPDATE, so writes on a ticket
// are serialized and checked against the row as it is once the lock is
// held (READ COMMITTED re-reads a row it waited for). The time of the write
// is taken in that UPDATE, after the lock, and never goes back on a ticket:
// GREATEST(clock_timestamp(), updated_at). now() would be the start of the
// transaction, earlier than the write it waited for. The entry gets the
// same time, so the ticket's updated_at is the time of its last entry
// (BR-8), and a failure anywhere rolls back both statements (BR-10).
const (
	lockForMessageQuery = `UPDATE tickets SET updated_at = GREATEST(clock_timestamp(), updated_at)
		WHERE id = $1 AND status <> 'fechado'
		RETURNING updated_at`

	lockForStatusQuery = `UPDATE tickets SET status = $3, updated_at = GREATEST(clock_timestamp(), updated_at)
		WHERE id = $1 AND status = $2
		RETURNING updated_at`

	ticketExistsQuery = `SELECT EXISTS (SELECT 1 FROM tickets WHERE id = $1)`

	// insertEntryQuery stores an entry and returns it as Thread reads it.
	insertEntryQuery = `WITH e AS (
			INSERT INTO ticket_entries (ticket_id, actor_id, kind, body, from_status, to_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, ticket_id, actor_id, kind, body, from_status, to_status, created_at)
		SELECT e.id, e.ticket_id, e.actor_id, u.login, u.role, e.kind,
			COALESCE(e.body, ''), COALESCE(e.from_status, ''), COALESCE(e.to_status, ''), e.created_at
		FROM e JOIN users u ON u.id = e.actor_id`
)

// AddMessage stores a message of actor on ticket and moves the ticket's
// updated_at to the message's time; the status does not change (BR-6). A
// "fechado" ticket is domain.ErrTicketClosed and nothing is written (BR-5);
// a ticket that does not exist is domain.ErrNotFound. text is stored as
// given: the service validates it.
func (r *TicketEntries) AddMessage(ctx context.Context, ticket domain.TicketID, actor domain.UserID, text string) (domain.TicketEntry, error) {
	e, err := r.write(ctx, lockForMessageQuery, []any{int64(ticket)}, domain.ErrTicketClosed,
		int64(ticket), int64(actor), string(domain.EntryKindMessage), text, nil, nil)
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("repository: add message: %w", err)
	}
	return e, nil
}

// ChangeStatus moves ticket from status from to status to, if from is
// still its status, and records the change of actor in its history (BR-4,
// BR-7). Otherwise nothing is written and the error is
// domain.ErrStaleStatus; a ticket that does not exist is
// domain.ErrNotFound. Only the compare-and-set on the status decides, so a
// reply written meanwhile never makes a status change fail (plan P-6). The
// pair itself is checked by the service (security.CheckStatusChange) and,
// as a second net, by the ticket_entries_allowed_transition CHECK.
func (r *TicketEntries) ChangeStatus(ctx context.Context, ticket domain.TicketID, actor domain.UserID, from, to domain.TicketStatus) (domain.TicketEntry, error) {
	e, err := r.write(ctx, lockForStatusQuery, []any{int64(ticket), string(from), string(to)}, domain.ErrStaleStatus,
		int64(ticket), int64(actor), string(domain.EntryKindStatusChange), nil, string(from), string(to))
	if err != nil {
		return domain.TicketEntry{}, fmt.Errorf("repository: change status: %w", err)
	}
	return e, nil
}

// write runs lockQuery (an UPDATE of the ticket that returns the time of
// the write), then inserts the entry with that time, in one transaction.
// When lockQuery updates no row, the ticket is either missing
// (domain.ErrNotFound) or in a state that refuses the write (refused).
// entry holds ticket_id, actor_id, kind, body, from_status and to_status.
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
