package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"backend_si/internal/domain"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	db, err := Open(t.Context(), url, PoolOptions{MaxOpenConns: 2, MaxIdleConns: 1})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestUsersAndSessionsIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	users, sessions := NewUsers(db), NewSessions(db)

	login := "it_" + rand.Text()[:10]
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanups run, so the delete
		// needs a context without that cancellation.
		if _, err := db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM users WHERE login = $1`, login); err != nil {
			t.Errorf("cleanup: delete test user %s: %v", login, err)
		}
	})

	id, err := users.Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: domain.RoleComum}) //nolint:gosec // fake hash fixture, not a real credential
	if err != nil || id == 0 {
		t.Fatalf("Create = %d, %v", id, err)
	}
	_, err = users.Create(ctx, domain.User{Login: login, PasswordHash: "x", Role: domain.RoleComum})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Create duplicado: erro = %v, want ErrConflict", err)
	}
	_, err = users.Create(ctx, domain.User{Login: login + "x", PasswordHash: "x", Role: "root"})
	if err == nil {
		t.Fatal("CHECK de role deveria rejeitar 'root'")
	}

	u, err := users.GetByLogin(ctx, login)
	if err != nil || u.ID != id || u.Role != domain.RoleComum {
		t.Fatalf("GetByLogin = %+v, %v", u, err)
	}
	_, err = users.GetByLogin(ctx, "' OR '1'='1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByLogin com payload de injeção: erro = %v, want ErrNotFound", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	hashAnon := sha256.Sum256([]byte(rand.Text()))
	hashAuth := sha256.Sum256([]byte(rand.Text()))
	anon := domain.Session{TokenHash: hashAnon[:], CSRFToken: "a", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Minute)}
	auth := domain.Session{TokenHash: hashAuth[:], UserID: id, CSRFToken: "b", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Minute)}
	for _, s := range []domain.Session{anon, auth} {
		if err = sessions.Create(ctx, s); err != nil {
			t.Fatalf("Create sessão: %v", err)
		}
	}

	got, err := sessions.Get(ctx, hashAnon[:])
	if err != nil || got.Authenticated() || got.Role != "" {
		t.Fatalf("Get anônima = %+v, %v", got, err)
	}
	got, err = sessions.Get(ctx, hashAuth[:])
	if err != nil || got.UserID != id || got.Role != domain.RoleComum || got.CSRFToken != "b" {
		t.Fatalf("Get autenticada = %+v, %v", got, err)
	}

	if err = sessions.Touch(ctx, hashAuth[:], now.Add(time.Second), now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	missing := sha256.Sum256([]byte("inexistente"))
	if err = sessions.Touch(ctx, missing[:], now, now.Add(time.Minute)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Touch inexistente: erro = %v, want ErrNotFound", err)
	}

	n, err := sessions.DeleteExpired(ctx, now.Add(90*time.Second))
	if err != nil || n < 1 {
		t.Fatalf("DeleteExpired = %d, %v", n, err)
	}
	if _, err = sessions.Get(ctx, hashAnon[:]); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sessão expirada ainda existe: %v", err)
	}
	if err = sessions.Delete(ctx, hashAuth[:]); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = sessions.Get(ctx, hashAuth[:]); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sessão removida ainda existe: %v", err)
	}
}

// TestRoleChangeIntegration covers 003 D-12: a role changed in the database
// is what the next Sessions.Get returns, in both directions, with no new
// session.
func TestRoleChangeIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	users, sessions := NewUsers(db), NewSessions(db)

	login := "it_" + rand.Text()[:10]
	t.Cleanup(func() {
		// Sessions go with the user (ON DELETE CASCADE).
		if _, err := db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM users WHERE login = $1`, login); err != nil {
			t.Errorf("cleanup: delete test user %s: %v", login, err)
		}
	})

	id, err := users.Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: domain.RoleComum}) //nolint:gosec // fake hash fixture, not a real credential
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	hash := sha256.Sum256([]byte(rand.Text()))
	sess := domain.Session{TokenHash: hash[:], UserID: id, CSRFToken: "c", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := sessions.Create(ctx, sess); err != nil {
		t.Fatalf("Create session: %v", err)
	}

	for _, role := range []domain.Role{domain.RoleComum, domain.RoleAdmin, domain.RoleComum} {
		res, err := db.ExecContext(ctx, `UPDATE users SET role = $1 WHERE id = $2`, string(role), int64(id))
		if err != nil {
			t.Fatalf("UPDATE role to %s: %v", role, err)
		}
		if n, rowsErr := res.RowsAffected(); rowsErr != nil || n != 1 {
			t.Fatalf("UPDATE role to %s affected %d rows, %v", role, n, rowsErr)
		}
		got, err := sessions.Get(ctx, hash[:])
		if err != nil || got.UserID != id || got.Role != role {
			t.Fatalf("Get after role %s = %+v, %v", role, got, err)
		}
	}
}

// ticketRow is one row of the tickets table as the integration test inserts
// it directly; nil is NULL. A nil status uses 'aberto', like the default.
type ticketRow struct {
	authorID      any
	typ           string
	status        *string
	title         string
	system        *string
	accessLevel   *string
	justification *string
	description   *string
	steps         *string
	severity      *string
	project       *string
	repository    *string
	environment   *string
	notes         *string
}

func ptr(s string) *string { return &s }

// chars returns n characters, each two bytes in UTF-8, so the CHECKs are
// shown to count characters rather than bytes.
func chars(n int) string { return strings.Repeat("é", n) }

func insertTicket(ctx context.Context, db *sql.DB, r ticketRow) (domain.TicketID, error) {
	var id domain.TicketID
	err := db.QueryRowContext(ctx, `
		INSERT INTO tickets (author_id, type, status, title, system, access_level, justification,
		                     description, steps, severity, project, repository, environment, notes)
		VALUES ($1, $2, COALESCE($3, 'aberto'), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`,
		r.authorID, r.typ, r.status, r.title, r.system, r.accessLevel, r.justification,
		r.description, r.steps, r.severity, r.project, r.repository, r.environment, r.notes,
	).Scan(&id)
	return id, err
}

// TestTicketsTableIntegration covers migration 00002 (spec 001-ticket-builder,
// S-1 second net, BR-1, BR-2, BR-6): one valid row per type is stored as
// "aberto" with server dates, and every row that breaks one rule is refused
// by the constraint of that rule, writing nothing.
func TestTicketsTableIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()

	login := "it_" + rand.Text()[:10]
	author, err := NewUsers(db).Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: domain.RoleComum}) //nolint:gosec // fake hash fixture, not a real credential
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() {
		// Tickets do not cascade: delete them before their author.
		bg := context.WithoutCancel(ctx)
		if _, err := db.ExecContext(bg, `DELETE FROM tickets WHERE author_id = $1`, author); err != nil {
			t.Errorf("cleanup: delete tickets: %v", err)
		}
		if _, err := db.ExecContext(bg, `DELETE FROM users WHERE id = $1`, author); err != nil {
			t.Errorf("cleanup: delete test user %s: %v", login, err)
		}
	})

	valid := map[string]func() ticketRow{
		"acesso": func() ticketRow {
			return ticketRow{
				authorID: author, typ: "acesso", title: "Acesso a Grafana",
				system: ptr("Grafana"), accessLevel: ptr("leitura"), justification: ptr("Preciso ver os painéis."),
			}
		},
		"bug": func() ticketRow {
			return ticketRow{
				authorID: author, typ: "bug", title: "Bug em Portal",
				system: ptr("Portal"), description: ptr("A página não carrega."), steps: ptr("1. Abrir\n2. Entrar"), severity: ptr("alta"),
			}
		},
		"deploy": func() ticketRow {
			return ticketRow{
				authorID: author, typ: "deploy", title: "Deploy de api em Produção",
				project: ptr("api"), repository: ptr("https://git.example/api"), environment: ptr("prod"),
			}
		},
		"outro": func() ticketRow {
			return ticketRow{authorID: author, typ: "outro", title: "Dúvida", description: ptr("Como peço uma VPN?")}
		},
	}
	with := func(typ string, change func(*ticketRow)) ticketRow {
		r := valid[typ]()
		change(&r)
		return r
	}

	t.Run("accepted", func(t *testing.T) {
		accepted := []struct {
			name string
			row  ticketRow
		}{
			{"acesso", valid["acesso"]()},
			{"bug", valid["bug"]()},
			{"deploy without notes", valid["deploy"]()},
			{"deploy with notes", with("deploy", func(r *ticketRow) { r.notes = ptr("Depois das 18h.") })},
			{"outro", valid["outro"]()},
			{"lengths at the minimum", with("bug", func(r *ticketRow) {
				r.title, r.system, r.description, r.steps = chars(1), ptr(chars(2)), ptr(chars(10)), ptr(chars(10))
			})},
			{"lengths at the maximum", with("acesso", func(r *ticketRow) {
				r.title, r.system, r.justification = chars(120), ptr(chars(120)), ptr(chars(2000))
			})},
			{"deploy lengths at the limits", with("deploy", func(r *ticketRow) {
				r.project, r.repository, r.notes = ptr(chars(120)), ptr(chars(300)), ptr(chars(2000))
			})},
		}
		for _, tt := range accepted {
			t.Run(tt.name, func(t *testing.T) {
				before := time.Now()
				id, err := insertTicket(ctx, db, tt.row)
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				var status string
				var createdAt, updatedAt time.Time
				err = db.QueryRowContext(ctx, `SELECT status, created_at, updated_at FROM tickets WHERE id = $1`, id).
					Scan(&status, &createdAt, &updatedAt)
				if err != nil {
					t.Fatalf("read back: %v", err)
				}
				if status != string(domain.TicketStatusOpen) {
					t.Errorf("status = %q, want %q", status, domain.TicketStatusOpen)
				}
				if !createdAt.Equal(updatedAt) {
					t.Errorf("created_at %v != updated_at %v", createdAt, updatedAt)
				}
				if d := createdAt.Sub(before); d < -time.Minute || d > time.Minute {
					t.Errorf("created_at = %v, not the server time (test started at %v)", createdAt, before)
				}
			})
		}
	})

	t.Run("refused", func(t *testing.T) {
		const (
			fk       = "23503"
			check    = "23514"
			notNull  = "23502"
			byType   = "tickets_fields_by_type"
			authorFK = "tickets_author_id_fkey"
		)
		refused := []struct {
			name       string
			row        ticketRow
			code       string
			constraint string
		}{
			// Wrong set of fields for the type.
			{"acesso without justification", with("acesso", func(r *ticketRow) { r.justification = nil }), check, byType},
			{"acesso with steps", with("acesso", func(r *ticketRow) { r.steps = ptr("1. Abrir a página") }), check, byType},
			{"acesso with notes", with("acesso", func(r *ticketRow) { r.notes = ptr("Observação") }), check, byType},
			{"bug without severity", with("bug", func(r *ticketRow) { r.severity = nil }), check, byType},
			{"bug with project", with("bug", func(r *ticketRow) { r.project = ptr("api") }), check, byType},
			{"bug with notes", with("bug", func(r *ticketRow) { r.notes = ptr("Observação") }), check, byType},
			{"deploy without repository", with("deploy", func(r *ticketRow) { r.repository = nil }), check, byType},
			{"deploy with system", with("deploy", func(r *ticketRow) { r.system = ptr("Portal") }), check, byType},
			{"outro without description", with("outro", func(r *ticketRow) { r.description = nil }), check, byType},
			{"outro with severity", with("outro", func(r *ticketRow) { r.severity = ptr("baixa") }), check, byType},
			{"outro with notes", with("outro", func(r *ticketRow) { r.notes = ptr("Observação") }), check, byType},
			{"acesso fields under type bug", with("acesso", func(r *ticketRow) { r.typ = "bug" }), check, byType},

			// Values outside their lists. An unknown type also breaks
			// tickets_fields_by_type, and PostgreSQL checks constraints in
			// name order, so that is the one it reports.
			{"unknown type", with("outro", func(r *ticketRow) { r.typ = "incidente" }), check, byType},
			{"status outside its list", with("outro", func(r *ticketRow) { r.status = ptr("cancelado") }), check, "tickets_status_check"},
			{"access level outside its list", with("acesso", func(r *ticketRow) { r.accessLevel = ptr("root") }), check, "tickets_access_level_check"},
			{"severity outside its list", with("bug", func(r *ticketRow) { r.severity = ptr("Alta") }), check, "tickets_severity_check"},
			{"environment outside its list", with("deploy", func(r *ticketRow) { r.environment = ptr("Produção") }), check, "tickets_environment_check"},

			// Lengths one past each limit.
			{"title empty", with("outro", func(r *ticketRow) { r.title = "" }), check, "tickets_title_check"},
			{"title 121", with("outro", func(r *ticketRow) { r.title = chars(121) }), check, "tickets_title_check"},
			{"system 1", with("acesso", func(r *ticketRow) { r.system = ptr(chars(1)) }), check, "tickets_system_check"},
			{"system 121", with("bug", func(r *ticketRow) { r.system = ptr(chars(121)) }), check, "tickets_system_check"},
			{"justification 9", with("acesso", func(r *ticketRow) { r.justification = ptr(chars(9)) }), check, "tickets_justification_check"},
			{"justification 2001", with("acesso", func(r *ticketRow) { r.justification = ptr(chars(2001)) }), check, "tickets_justification_check"},
			{"description 9", with("outro", func(r *ticketRow) { r.description = ptr(chars(9)) }), check, "tickets_description_check"},
			{"description 2001", with("bug", func(r *ticketRow) { r.description = ptr(chars(2001)) }), check, "tickets_description_check"},
			{"steps 9", with("bug", func(r *ticketRow) { r.steps = ptr(chars(9)) }), check, "tickets_steps_check"},
			{"steps 2001", with("bug", func(r *ticketRow) { r.steps = ptr(chars(2001)) }), check, "tickets_steps_check"},
			{"project 1", with("deploy", func(r *ticketRow) { r.project = ptr(chars(1)) }), check, "tickets_project_check"},
			{"project 121", with("deploy", func(r *ticketRow) { r.project = ptr(chars(121)) }), check, "tickets_project_check"},
			{"repository empty", with("deploy", func(r *ticketRow) { r.repository = ptr("") }), check, "tickets_repository_check"},
			{"repository 301", with("deploy", func(r *ticketRow) { r.repository = ptr(chars(301)) }), check, "tickets_repository_check"},
			{"notes empty instead of NULL", with("deploy", func(r *ticketRow) { r.notes = ptr("") }), check, "tickets_notes_check"},
			{"notes 2001", with("deploy", func(r *ticketRow) { r.notes = ptr(chars(2001)) }), check, "tickets_notes_check"},

			// Author.
			{"unknown author", with("outro", func(r *ticketRow) { r.authorID = int64(-1) }), fk, authorFK},
			{"no author", with("outro", func(r *ticketRow) { r.authorID = nil }), notNull, ""},
		}
		for _, tt := range refused {
			t.Run(tt.name, func(t *testing.T) {
				var before, after int
				count := func(dst *int) {
					if err := db.QueryRowContext(ctx, `SELECT count(*) FROM tickets`).Scan(dst); err != nil {
						t.Fatalf("count: %v", err)
					}
				}
				count(&before)
				_, err := insertTicket(ctx, db, tt.row)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) {
					t.Fatalf("insert error = %v, want PostgreSQL error %s", err, tt.code)
				}
				if pgErr.Code != tt.code || pgErr.ConstraintName != tt.constraint {
					t.Errorf("refused by %s %q, want %s %q", pgErr.Code, pgErr.ConstraintName, tt.code, tt.constraint)
				}
				count(&after)
				if after != before {
					t.Errorf("row count %d → %d: a refused row was written", before, after)
				}
			})
		}
	})

	t.Run("an author with tickets cannot be deleted", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, author)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" || pgErr.ConstraintName != "tickets_author_id_fkey" {
			t.Fatalf("delete author = %v, want foreign key violation", err)
		}
	})
}
