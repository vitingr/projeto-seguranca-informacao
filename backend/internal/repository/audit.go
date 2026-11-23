package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"

	"backend_si/internal/domain"
)

// Audit is the append-only audit log (006 D-7). It only appends and lists:
// there is no method to change or delete an event, and a trigger rejects
// UPDATE, DELETE and TRUNCATE on the table.
type Audit struct {
	db *sql.DB
}

func NewAudit(db *sql.DB) *Audit {
	return &Audit{db: db}
}

// execer is what appendAuditEvent needs from *sql.DB or *sql.Tx.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (r *Audit) Append(ctx context.Context, ev domain.AuditEvent) error {
	if err := appendAuditEvent(ctx, r.db, ev); err != nil {
		return fmt.Errorf("repository: append audit event: %w", err)
	}
	return nil
}

// appendAuditEvent inserts ev. Only the event, the actor, the target and
// the IP are stored; logins are never copied into the log (006 D-4).
func appendAuditEvent(ctx context.Context, db execer, ev domain.AuditEvent) error {
	const q = `INSERT INTO audit_events (event, actor_user_id, target_type, target_id, ip)
		VALUES ($1, $2, $3, $4, $5::inet)`

	actor := sql.NullInt64{Int64: int64(ev.ActorID), Valid: ev.ActorID != 0}
	targetType := sql.NullString{String: string(ev.TargetType), Valid: ev.TargetType != ""}
	targetID := sql.NullInt64{Int64: ev.TargetID, Valid: ev.TargetType != ""}
	ip := sql.NullString{String: ev.IP.String(), Valid: ev.IP.IsValid()}

	if _, err := db.ExecContext(ctx, q, string(ev.Type), actor, targetType, targetID, ip); err != nil {
		return fmt.Errorf("insert audit event %s: %w", ev.Type, err)
	}
	return nil
}

// List returns one slice of the log, newest first, with only the events of
// typ when typ is not empty (006 D-8). The logins of the actor and of a user
// target are read from users, and are empty when the user no longer exists.
func (r *Audit) List(ctx context.Context, typ domain.AuditEventType, limit, offset int) ([]domain.AuditEvent, error) {
	const q = `SELECT a.id, a.event, a.actor_user_id, actor.login, a.target_type, a.target_id,
			target.login, host(a.ip), a.created_at
		FROM audit_events a
		LEFT JOIN users actor ON actor.id = a.actor_user_id
		LEFT JOIN users target ON a.target_type = 'user' AND target.id = a.target_id
		WHERE $1 = '' OR a.event = $1
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, q, string(typ), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: list audit events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	events := []domain.AuditEvent{}
	for rows.Next() {
		ev, err := scanAuditEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: list audit events: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list audit events: %w", err)
	}
	return events, nil
}

func scanAuditEvent(row rowScanner) (domain.AuditEvent, error) {
	var (
		ev                      domain.AuditEvent
		event                   string
		actor, targetID         sql.NullInt64
		actorLogin, targetLogin sql.NullString
		targetType, ip          sql.NullString
	)
	if err := row.Scan(&ev.ID, &event, &actor, &actorLogin, &targetType, &targetID, &targetLogin, &ip, &ev.CreatedAt); err != nil {
		return domain.AuditEvent{}, err
	}
	ev.Type = domain.AuditEventType(event)
	ev.ActorID = domain.UserID(actor.Int64)
	ev.ActorLogin = actorLogin.String
	if targetType.Valid {
		ev.TargetType = domain.AuditTargetType(targetType.String)
		ev.TargetID = targetID.Int64
		ev.TargetLogin = targetLogin.String
	}
	if ip.Valid {
		addr, err := netip.ParseAddr(ip.String)
		if err != nil {
			return domain.AuditEvent{}, fmt.Errorf("parse ip: %w", err)
		}
		ev.IP = addr
	}
	return ev, nil
}
