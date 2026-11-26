package repository

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

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
