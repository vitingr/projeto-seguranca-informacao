package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

// createTestAdmin stores an admin that is deleted when the test ends. An
// admin who removed posts can only be deleted once those posts are gone
// (removed_by has no ON DELETE action), so create it before the authors:
// cleanups run last-in first-out, and the authors' posts go with them.
func createTestAdmin(t *testing.T, users *Users) domain.UserID {
	t.Helper()
	ctx := t.Context()
	login := "it_" + strings.ToLower(rand.Text()[:10])
	id, err := users.Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: domain.RoleAdmin}) //nolint:gosec // fake hash fixture, not a real credential
	if err != nil {
		t.Fatalf("Create admin: %v", err)
	}
	t.Cleanup(func() {
		if _, err := users.db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM users WHERE id = $1`, int64(id)); err != nil {
			t.Errorf("cleanup: delete test admin %d: %v", id, err)
		}
	})
	return id
}

func countEvents(t *testing.T, db *sql.DB, event domain.AuditEventType, targetType domain.AuditTargetType, targetID int64) int {
	t.Helper()
	var n int
	err := db.QueryRowContext(t.Context(),
		`SELECT count(*) FROM audit_events WHERE event = $1 AND target_type = $2 AND target_id = $3`,
		string(event), string(targetType), targetID).Scan(&n)
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	return n
}

func findPost(posts []domain.Post, id domain.PostID) (domain.Post, bool) {
	for _, p := range posts {
		if p.ID == id {
			return p, true
		}
	}
	return domain.Post{}, false
}

// TestPostRemovalIntegration covers 006 S-6 and D-3 against PostgreSQL: the
// removal and its event are one transaction, a second removal is 404, the
// author can no longer edit or delete the post, the row is kept, and only
// the author and the admins still list it.
func TestPostRemovalIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	users, posts := NewUsers(db), NewPosts(db)
	admin := createTestAdmin(t, users)
	author, other := createTestUser(t, users), createTestUser(t, users)

	created, err := posts.Create(ctx, domain.Post{AuthorID: author, Title: "a remover", Body: "texto"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Removal != nil {
		t.Fatalf("a new post has removal %+v", created.Removal)
	}
	ip := netip.MustParseAddr("203.0.113.7")
	event := domain.AuditEvent{Type: domain.AuditPostRemoved, ActorID: admin, TargetType: domain.AuditTargetPost, TargetID: int64(created.ID), IP: ip}

	// An event the log rejects (its CHECK) rolls the removal back.
	bad := event
	bad.Type = "bogus"
	if err = posts.Remove(ctx, created.ID, admin, "spam", bad); err == nil {
		t.Fatal("Remove with an event the log rejects succeeded")
	}
	var removedAt sql.NullTime
	if err = db.QueryRowContext(ctx, `SELECT removed_at FROM posts WHERE id = $1`, int64(created.ID)).Scan(&removedAt); err != nil || removedAt.Valid {
		t.Fatalf("after a failed event: removed_at = %v, %v; want the removal rolled back", removedAt, err)
	}

	if err = posts.Remove(ctx, created.ID, admin, "Conteúdo ofensivo.", event); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	var (
		removedBy int64
		reason    string
	)
	err = db.QueryRowContext(ctx, `SELECT removed_at, removed_by, removal_reason FROM posts WHERE id = $1`, int64(created.ID)).
		Scan(&removedAt, &removedBy, &reason)
	if err != nil || !removedAt.Valid || removedBy != int64(admin) || reason != "Conteúdo ofensivo." {
		t.Errorf("stored removal = %v, %d, %q, %v", removedAt, removedBy, reason, err)
	}
	if n := countEvents(t, db, domain.AuditPostRemoved, domain.AuditTargetPost, int64(created.ID)); n != 1 {
		t.Errorf("%d post_removed events for the post, want 1", n)
	}

	if err := posts.Remove(ctx, created.ID, admin, "de novo", event); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second Remove = %v, want ErrNotFound", err)
	}
	if n := countEvents(t, db, domain.AuditPostRemoved, domain.AuditTargetPost, int64(created.ID)); n != 1 {
		t.Errorf("%d post_removed events after a second removal, want still 1", n)
	}
	if err := posts.Remove(ctx, created.ID+1_000_000, admin, "x", event); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Remove of a missing post = %v, want ErrNotFound", err)
	}
	if _, err := posts.Update(ctx, domain.Post{ID: created.ID, AuthorID: author, Title: "editado", Body: "x"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("author's Update of a removed post = %v, want ErrNotFound", err)
	}
	if err := posts.Delete(ctx, created.ID, author); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("author's Delete of a removed post = %v, want ErrNotFound", err)
	}
	var title string
	if err := db.QueryRowContext(ctx, `SELECT title FROM posts WHERE id = $1`, int64(created.ID)).Scan(&title); err != nil || title != "a remover" {
		t.Errorf("removed post after the author's attempts = %q, %v; want it kept unchanged", title, err)
	}

	for _, tt := range []struct {
		name   string
		viewer domain.UserID
		all    bool
		want   bool
	}{
		{"another comum user", other, false, false},
		{"the author", author, false, true},
		{"an admin", admin, true, true},
	} {
		got, err := posts.List(ctx, tt.viewer, tt.all, 1000, 0)
		if err != nil {
			t.Fatalf("List as %s: %v", tt.name, err)
		}
		p, ok := findPost(got, created.ID)
		if ok != tt.want {
			t.Errorf("List as %s: has the removed post = %v, want %v", tt.name, ok, tt.want)
		}
		if ok && (p.Removal == nil || p.Removal.Reason != "Conteúdo ofensivo." || p.Removal.RemovedBy != admin || p.Removal.RemovedAt.IsZero()) {
			t.Errorf("List as %s: removal = %+v", tt.name, p.Removal)
		}
	}

	// The CHECK keeps the three columns together.
	if _, err := db.ExecContext(ctx, `UPDATE posts SET removal_reason = NULL WHERE id = $1`, int64(created.ID)); err == nil {
		t.Error("a removed post without a reason passed the CHECK")
	}
}

// TestAuditAppendOnlyIntegration covers 006 S-1 and D-7 against PostgreSQL:
// UPDATE, DELETE and TRUNCATE on audit_events fail in the trigger, even
// when no row matches. Each attempt runs in a transaction that is rolled
// back, so a missing trigger cannot wipe the log.
func TestAuditAppendOnlyIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	audit := NewAudit(db)
	if err := audit.Append(ctx, domain.AuditEvent{Type: domain.AuditLogout, ActorID: 999_999_999}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	for _, stmt := range []string{
		`UPDATE audit_events SET event = 'logout' WHERE actor_user_id = 999999999`,
		`UPDATE audit_events SET ip = NULL WHERE false`,
		`DELETE FROM audit_events WHERE actor_user_id = 999999999`,
		`DELETE FROM audit_events WHERE false`,
		`TRUNCATE audit_events`,
	} {
		t.Run(stmt, func(t *testing.T) {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			_, err = tx.ExecContext(ctx, stmt)
			if err == nil || !strings.Contains(err.Error(), "append-only") {
				t.Errorf("error = %v, want the append-only trigger to reject it", err)
			}
		})
	}

	var n int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM audit_events WHERE actor_user_id = 999999999`).Scan(&n); err != nil || n < 1 {
		t.Errorf("the appended event is gone: %d, %v", n, err)
	}
}

// TestAuditListIntegration covers the audit log reads (006 D-8): newest
// first, the type filter, the logins joined from users, and the IP.
func TestAuditListIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	users, audit := NewUsers(db), NewAudit(db)
	user := createTestUser(t, users)
	var login string
	if err := db.QueryRowContext(ctx, `SELECT login FROM users WHERE id = $1`, int64(user)).Scan(&login); err != nil {
		t.Fatal(err)
	}

	appended := []domain.AuditEvent{
		{Type: domain.AuditLoginFailed, TargetType: domain.AuditTargetUser, TargetID: int64(user), IP: netip.MustParseAddr("203.0.113.9")},
		{Type: domain.AuditLoginSucceeded, ActorID: user, IP: netip.MustParseAddr("::1")},
		{Type: domain.AuditLogout, ActorID: user},
	}
	for _, ev := range appended {
		if err := audit.Append(ctx, ev); err != nil {
			t.Fatalf("Append %s: %v", ev.Type, err)
		}
	}
	if err := audit.Append(ctx, domain.AuditEvent{Type: "password_changed"}); err == nil {
		t.Error("Append of an unknown event passed the CHECK")
	}

	mine := func(evs []domain.AuditEvent) []domain.AuditEvent {
		var out []domain.AuditEvent
		for _, ev := range evs {
			if ev.ActorID == user || (ev.TargetType == domain.AuditTargetUser && ev.TargetID == int64(user)) {
				out = append(out, ev)
			}
		}
		return out
	}
	all, err := audit.List(ctx, "", 1000, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := mine(all)
	if len(got) != 3 || got[0].Type != domain.AuditLogout || got[1].Type != domain.AuditLoginSucceeded || got[2].Type != domain.AuditLoginFailed {
		t.Fatalf("List = %+v, want logout, login_succeeded, login_failed (newest first)", got)
	}
	if got[0].IP.IsValid() || got[0].ActorLogin != login || got[0].TargetType != "" {
		t.Errorf("logout = %+v, want no IP, the actor login and no target", got[0])
	}
	if got[1].IP.String() != "::1" {
		t.Errorf("login_succeeded IP = %v, want ::1", got[1].IP)
	}
	if got[2].ActorID != 0 || got[2].TargetLogin != login || got[2].IP.String() != "203.0.113.9" {
		t.Errorf("login_failed = %+v, want no actor, the target login and 203.0.113.9", got[2])
	}

	failed, err := audit.List(ctx, domain.AuditLoginFailed, 1000, 0)
	if err != nil {
		t.Fatalf("List(login_failed): %v", err)
	}
	for _, ev := range failed {
		if ev.Type != domain.AuditLoginFailed {
			t.Errorf("List(login_failed) returned %s", ev.Type)
		}
	}
	if len(mine(failed)) != 1 {
		t.Errorf("List(login_failed) has %d of this test's events, want 1", len(mine(failed)))
	}
	if page, err := audit.List(ctx, "", 1, 0); err != nil || len(page) != 1 {
		t.Errorf("List(limit 1) = %d events, %v", len(page), err)
	}
}
