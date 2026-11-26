package repository

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
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

// TestSessionGetReadsTheUserProfileIntegration covers spec Q-16: the same
// session query returns the user's login and registration time, and an
// anonymous session has neither.
func TestSessionGetReadsTheUserProfileIntegration(t *testing.T) {
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
	var registered time.Time
	if err = db.QueryRowContext(ctx, `SELECT created_at FROM users WHERE id = $1`, int64(id)).Scan(&registered); err != nil {
		t.Fatalf("read created_at: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	hashAuth := sha256.Sum256([]byte(rand.Text()))
	hashAnon := sha256.Sum256([]byte(rand.Text()))
	t.Cleanup(func() {
		if _, delErr := db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM sessions WHERE token_hash = $1`, hashAnon[:]); delErr != nil {
			t.Errorf("cleanup: delete anonymous session: %v", delErr)
		}
	})
	for _, s := range []domain.Session{
		{TokenHash: hashAuth[:], UserID: id, CSRFToken: "p", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Minute)},
		{TokenHash: hashAnon[:], CSRFToken: "q", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Minute)},
	} {
		if err = sessions.Create(ctx, s); err != nil {
			t.Fatalf("Create session: %v", err)
		}
	}

	got, err := sessions.Get(ctx, hashAuth[:])
	if err != nil || got.Login != login || !got.UserCreatedAt.Equal(registered) || got.UserCreatedAt.IsZero() {
		t.Fatalf("Get = login %q, created %v, %v; want %q, %v", got.Login, got.UserCreatedAt, err, login, registered)
	}
	got, err = sessions.Get(ctx, hashAnon[:])
	if err != nil || got.Login != "" || !got.UserCreatedAt.IsZero() {
		t.Fatalf("anonymous Get = login %q, created %v, %v; want neither", got.Login, got.UserCreatedAt, err)
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

// ticketsEqual compares two tickets field by field, with the dates compared
// as instants (the database may return another location).
func ticketsEqual(a, b domain.Ticket) bool {
	if !a.CreatedAt.Equal(b.CreatedAt) || !a.UpdatedAt.Equal(b.UpdatedAt) {
		return false
	}
	a.CreatedAt, a.UpdatedAt, b.CreatedAt, b.UpdatedAt = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	return a == b
}

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

	t.Run("Tickets.Create", func(t *testing.T) {
		repo := NewTickets(db)
		tickets := []domain.Ticket{
			{
				AuthorID: author, Type: domain.TicketTypeAccess, Title: "Acesso a Grafana",
				System: "Grafana", AccessLevel: domain.AccessLevelRead, Justification: "Preciso ver os painéis.",
			},
			{
				AuthorID: author, Type: domain.TicketTypeBug, Title: "Bug em Portal",
				System: "Portal", Description: "A página não carrega.", Steps: "1. Abrir\n2. Entrar", Severity: domain.SeverityHigh,
			},
			{
				AuthorID: author, Type: domain.TicketTypeDeploy, Title: "Deploy de api em Produção",
				Project: "api", Repository: "https://git.example/api", Environment: domain.EnvironmentProduction,
			},
			{
				AuthorID: author, Type: domain.TicketTypeDeploy, Title: "Deploy de api em Homologação",
				Project: "api", Repository: "https://git.example/api", Environment: domain.EnvironmentStaging, Notes: "Depois das 18h.",
			},
			// The status column is left to its default: a status in the
			// input is not stored (BR-1).
			{
				AuthorID: author, Type: domain.TicketTypeOther, Status: domain.TicketStatusClosed,
				Title: "Dúvida <b>'; DROP TABLE tickets; --", Description: "Como peço uma VPN?\n\tObrigado.",
			},
		}
		for _, in := range tickets {
			t.Run(string(in.Type)+" "+in.Title, func(t *testing.T) {
				before := time.Now()
				got, err := repo.Create(ctx, in)
				if err != nil {
					t.Fatalf("Create: %v", err)
				}
				if got.ID == 0 {
					t.Error("Create returned no id")
				}
				if got.Status != domain.TicketStatusOpen {
					t.Errorf("status = %q, want %q", got.Status, domain.TicketStatusOpen)
				}
				if got.CreatedAt.IsZero() || !got.CreatedAt.Equal(got.UpdatedAt) {
					t.Errorf("created_at %v, updated_at %v: want equal server dates", got.CreatedAt, got.UpdatedAt)
				}
				if d := got.CreatedAt.Sub(before); d < -time.Minute || d > time.Minute {
					t.Errorf("created_at = %v, not the server time (test started at %v)", got.CreatedAt, before)
				}

				// Every column as stored; NULL reads back as "".
				var stored domain.Ticket
				var status string
				var system, level, justification, description, steps, severity, project, repository, environment, notes sql.NullString
				err = db.QueryRowContext(ctx, `
					SELECT author_id, type, status, title, system, access_level, justification, description,
					       steps, severity, project, repository, environment, notes, created_at, updated_at
					FROM tickets WHERE id = $1`, got.ID).
					Scan(&stored.AuthorID, &stored.Type, &status, &stored.Title, &system, &level, &justification,
						&description, &steps, &severity, &project, &repository, &environment, &notes,
						&stored.CreatedAt, &stored.UpdatedAt)
				if err != nil {
					t.Fatalf("read back: %v", err)
				}
				stored.ID, stored.Status = got.ID, domain.TicketStatus(status)
				stored.System, stored.AccessLevel, stored.Justification = system.String, domain.AccessLevel(level.String), justification.String
				stored.Description, stored.Steps, stored.Severity = description.String, steps.String, domain.Severity(severity.String)
				stored.Project, stored.Repository, stored.Environment = project.String, repository.String, domain.Environment(environment.String)
				stored.Notes = notes.String

				want := in
				want.ID, want.Status, want.CreatedAt, want.UpdatedAt = got.ID, domain.TicketStatusOpen, got.CreatedAt, got.UpdatedAt
				if !ticketsEqual(got, want) {
					t.Errorf("returned ticket = %+v, want %+v", got, want)
				}
				if !ticketsEqual(stored, want) {
					t.Errorf("stored ticket = %+v, want %+v", stored, want)
				}
				if in.Notes == "" && notes.Valid {
					t.Error("notes stored as an empty string, want NULL")
				}
			})
		}
	})

	t.Run("Tickets.Create refused by a CHECK writes nothing", func(t *testing.T) {
		repo := NewTickets(db)
		refused := []struct {
			name       string
			ticket     domain.Ticket
			constraint string
		}{
			{"field of another type", domain.Ticket{
				AuthorID: author, Type: domain.TicketTypeOther, Title: "Dúvida",
				Description: "Como peço uma VPN?", Severity: domain.SeverityLow,
			}, "tickets_fields_by_type"},
			{"missing required field", domain.Ticket{
				AuthorID: author, Type: domain.TicketTypeBug, Title: "Bug em Portal",
				System: "Portal", Description: "A página não carrega.", Severity: domain.SeverityHigh,
			}, "tickets_fields_by_type"},
			{"enum outside its list", domain.Ticket{
				AuthorID: author, Type: domain.TicketTypeDeploy, Title: "Deploy de api",
				Project: "api", Repository: "https://git.example/api", Environment: "Produção",
			}, "tickets_environment_check"},
			{"empty title", domain.Ticket{
				AuthorID: author, Type: domain.TicketTypeOther,
				Description: "Como peço uma VPN?",
			}, "tickets_title_check"},
			{"title over 120", domain.Ticket{
				AuthorID: author, Type: domain.TicketTypeOther, Title: chars(121),
				Description: "Como peço uma VPN?",
			}, "tickets_title_check"},
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
				got, err := repo.Create(ctx, tt.ticket)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
					t.Fatalf("Create error = %v, want a wrapped CHECK violation of %s", err, tt.constraint)
				}
				if !strings.HasPrefix(err.Error(), "repository: criar ticket: ") {
					t.Errorf("error %q is not wrapped by the repository", err)
				}
				if got != (domain.Ticket{}) {
					t.Errorf("Create returned %+v with an error, want the zero ticket", got)
				}
				count(&after)
				if after != before {
					t.Errorf("row count %d → %d: a refused ticket was written", before, after)
				}
			})
		}
	})

	t.Run("Tickets.Create with an unknown author", func(t *testing.T) {
		_, err := NewTickets(db).Create(ctx, domain.Ticket{
			AuthorID: -1, Type: domain.TicketTypeOther,
			Title: "Dúvida", Description: "Como peço uma VPN?",
		})
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
			t.Fatalf("Create error = %v, want a foreign key violation", err)
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

// entryRow is a ticket_entries row as inserted by hand, so the table's own
// constraints are tested without the repository rules. nil is NULL.
type entryRow struct {
	ticketID any
	actorID  any
	kind     any
	body     *string
	from     *string
	to       *string
}

func insertEntry(ctx context.Context, db *sql.DB, r entryRow) (domain.TicketEntryID, error) {
	var id domain.TicketEntryID
	err := db.QueryRowContext(ctx, `
		INSERT INTO ticket_entries (ticket_id, actor_id, kind, body, from_status, to_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		r.ticketID, r.actorID, r.kind, r.body, r.from, r.to,
	).Scan(&id)
	return id, err
}

// TestTicketEntriesTableIntegration covers migration 00003 (spec
// 003-ticket-tracking, plan P-3 and P-15; BR-3, BR-4, BR-8; S-7): every row
// that breaks a rule of ticket_entries or tickets is refused by the
// constraint of that rule, writing nothing, and the six BR-3 transitions
// and valid messages are accepted.
func TestTicketEntriesTableIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()

	users := NewUsers(db)
	newUser := func(role domain.Role) domain.UserID {
		login := "it_" + rand.Text()[:10]
		id, err := users.Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: role}) //nolint:gosec // fake hash fixture, not a real credential
		if err != nil {
			t.Fatalf("Create user: %v", err)
		}
		return id
	}
	author, admin := newUser(domain.RoleComum), newUser(domain.RoleAdmin)
	ticket, err := insertTicket(ctx, db, ticketRow{
		authorID: author, typ: "outro", title: "Dúvida", description: ptr("Como peço uma VPN?"),
	})
	if err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	t.Cleanup(func() {
		// Nothing cascades: entries, then tickets, then users.
		bg := context.WithoutCancel(ctx)
		for _, q := range []string{
			`DELETE FROM ticket_entries WHERE ticket_id IN (SELECT id FROM tickets WHERE author_id = $1) OR actor_id IN ($1, $2)`,
			`DELETE FROM tickets WHERE author_id IN ($1, $2)`,
			`DELETE FROM users WHERE id IN ($1, $2)`,
		} {
			if _, err := db.ExecContext(bg, q, author, admin); err != nil {
				t.Errorf("cleanup %q: %v", q, err)
			}
		}
	})

	statuses := []string{"aberto", "em_andamento", "resolvido", "fechado"}
	allowed := map[[2]string]bool{
		{"aberto", "em_andamento"}: true, {"aberto", "fechado"}: true,
		{"em_andamento", "resolvido"}: true, {"em_andamento", "fechado"}: true,
		{"resolvido", "em_andamento"}: true, {"resolvido", "fechado"}: true,
	}
	message := func(body *string) entryRow {
		return entryRow{ticketID: ticket, actorID: author, kind: "message", body: body}
	}
	change := func(from, to *string) entryRow {
		return entryRow{ticketID: ticket, actorID: admin, kind: "status_change", from: from, to: to}
	}
	countEntries := func(t *testing.T) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM ticket_entries`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}

	t.Run("accepted", func(t *testing.T) {
		type accepted struct {
			name string
			row  entryRow
		}
		cases := []accepted{
			{"message of 1 character", message(ptr("a"))},
			{"message of 2000 multi-byte characters", message(ptr(chars(2000)))},
			{"message with line breaks and a tab", message(ptr("Linha 1\nLinha 2\r\n\tFim"))},
			{"message by an admin", entryRow{ticketID: ticket, actorID: admin, kind: "message", body: ptr("Olá")}},
		}
		for _, from := range statuses {
			for _, to := range statuses {
				if allowed[[2]string{from, to}] {
					cases = append(cases, accepted{"change " + from + " → " + to, change(ptr(from), ptr(to))})
				}
			}
		}
		if got := len(cases) - 4; got != 6 {
			t.Fatalf("%d BR-3 transitions in the table, want 6", got)
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				before := time.Now()
				id, err := insertEntry(ctx, db, tt.row)
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				var createdAt time.Time
				if err := db.QueryRowContext(ctx, `SELECT created_at FROM ticket_entries WHERE id = $1`, id).Scan(&createdAt); err != nil {
					t.Fatalf("read back: %v", err)
				}
				if d := createdAt.Sub(before); d < -time.Minute || d > time.Minute {
					t.Errorf("created_at = %v, not the server time (test started at %v)", createdAt, before)
				}
			})
		}
	})

	t.Run("refused", func(t *testing.T) {
		const (
			fk         = "23503"
			check      = "23514"
			notNull    = "23502"
			byKind     = "ticket_entries_fields_by_kind"
			transition = "ticket_entries_allowed_transition"
			bodyCheck  = "ticket_entries_body_check"
		)
		type refused struct {
			name       string
			row        entryRow
			code       string
			constraint string
		}
		cases := []refused{
			// Wrong set of fields for the kind. A message with a valid pair
			// passes ticket_entries_allowed_transition (it only looks at
			// status changes), and a status change with one NULL status
			// makes that CHECK NULL, which passes; so fields_by_kind is the
			// one that refuses them.
			{"message without body", message(nil), check, byKind},
			{"message with both statuses", entryRow{ticketID: ticket, actorID: author, kind: "message", body: ptr("Oi"), from: ptr("aberto"), to: ptr("em_andamento")}, check, byKind},
			{"message with a to status", entryRow{ticketID: ticket, actorID: author, kind: "message", body: ptr("Oi"), to: ptr("fechado")}, check, byKind},
			{"status change with a body", entryRow{ticketID: ticket, actorID: admin, kind: "status_change", body: ptr("Oi"), from: ptr("aberto"), to: ptr("em_andamento")}, check, byKind},
			{"status change without from", change(nil, ptr("em_andamento")), check, byKind},
			{"status change without to", change(ptr("aberto"), nil), check, byKind},
			{"status change without statuses", change(nil, nil), check, byKind},
			// An unknown kind breaks both fields_by_kind and the kind
			// CHECK; PostgreSQL checks constraints in name order.
			{"unknown kind", entryRow{ticketID: ticket, actorID: author, kind: "note", body: ptr("Oi")}, check, byKind},

			// Body length, in characters.
			{"body empty", message(ptr("")), check, bodyCheck},
			{"body 2001 multi-byte characters", message(ptr(chars(2001))), check, bodyCheck},

			// Missing or unknown references.
			{"no kind", entryRow{ticketID: ticket, actorID: author, body: ptr("Oi")}, notNull, ""},
			{"no ticket", entryRow{actorID: author, kind: "message", body: ptr("Oi")}, notNull, ""},
			{"no actor", entryRow{ticketID: ticket, kind: "message", body: ptr("Oi")}, notNull, ""},
			{"unknown ticket", entryRow{ticketID: int64(-1), actorID: author, kind: "message", body: ptr("Oi")}, fk, "ticket_entries_ticket_id_fkey"},
			{"unknown actor", entryRow{ticketID: ticket, actorID: int64(-1), kind: "message", body: ptr("Oi")}, fk, "ticket_entries_actor_id_fkey"},
		}
		// Every pair outside BR-3: the same status, anything from
		// "fechado", and the backward or skipping moves.
		outside := 0
		for _, from := range statuses {
			for _, to := range statuses {
				if !allowed[[2]string{from, to}] {
					outside++
					cases = append(cases, refused{"change " + from + " → " + to, change(ptr(from), ptr(to)), check, transition})
				}
			}
		}
		if outside != 10 {
			t.Fatalf("%d pairs outside BR-3, want 10", outside)
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				before := countEntries(t)
				_, err := insertEntry(ctx, db, tt.row)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) {
					t.Fatalf("insert error = %v, want PostgreSQL error %s", err, tt.code)
				}
				if pgErr.Code != tt.code || pgErr.ConstraintName != tt.constraint {
					t.Errorf("refused by %s %q, want %s %q", pgErr.Code, pgErr.ConstraintName, tt.code, tt.constraint)
				}
				if after := countEntries(t); after != before {
					t.Errorf("row count %d → %d: a refused entry was written", before, after)
				}
			})
		}
	})

	t.Run("updated_at is never before created_at", func(t *testing.T) {
		tests := []struct {
			name       string
			shift      string // added to created_at
			constraint string
		}{
			{"one second before", "-1 second", "tickets_updated_not_before_created"},
			{"equal", "0", ""},
			{"later", "1 minute", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var prev time.Time
				if err := db.QueryRowContext(ctx, `SELECT updated_at FROM tickets WHERE id = $1`, ticket).Scan(&prev); err != nil {
					t.Fatalf("read: %v", err)
				}
				_, err := db.ExecContext(ctx, `UPDATE tickets SET updated_at = created_at + $2::interval WHERE id = $1`, ticket, tt.shift)
				if tt.constraint == "" {
					if err != nil {
						t.Fatalf("update: %v", err)
					}
					return
				}
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
					t.Fatalf("update error = %v, want CHECK violation of %s", err, tt.constraint)
				}
				var now time.Time
				if err := db.QueryRowContext(ctx, `SELECT updated_at FROM tickets WHERE id = $1`, ticket).Scan(&now); err != nil {
					t.Fatalf("read back: %v", err)
				}
				if !now.Equal(prev) {
					t.Errorf("updated_at %v → %v: a refused update was written", prev, now)
				}
			})
		}
	})

	t.Run("a ticket or a user with entries cannot be deleted", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			id         any
			constraint string
		}{
			{"ticket", `DELETE FROM tickets WHERE id = $1`, ticket, "ticket_entries_ticket_id_fkey"},
			{"admin actor", `DELETE FROM users WHERE id = $1`, admin, "ticket_entries_actor_id_fkey"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := db.ExecContext(ctx, tt.query, tt.id)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "23503" || pgErr.ConstraintName != tt.constraint {
					t.Fatalf("delete = %v, want foreign key violation of %s", err, tt.constraint)
				}
			})
		}
	})
}

// TestTicketTrackingIndexesIntegration covers the indexes of migration 00003
// (plan section 1, S-14): the thread and both lists have their index, and
// the "Meus tickets" index of 00002, ordered by creation, is gone.
func TestTicketTrackingIndexesIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		table   string
		columns string // "" means the index must not exist
	}{
		{"ticket_entries_ticket_idx", "ticket_entries", "(ticket_id, id)"},
		{"ticket_entries_actor_idx", "ticket_entries", "(actor_id)"},
		{"tickets_author_updated_idx", "tickets", "(author_id, updated_at DESC, id DESC)"},
		{"tickets_updated_idx", "tickets", "(updated_at DESC, id DESC)"},
		{"tickets_status_updated_idx", "tickets", "(status, updated_at DESC, id DESC)"},
		{"tickets_author_created_idx", "tickets", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table, def string
			err := db.QueryRowContext(ctx,
				`SELECT tablename, indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = $1`,
				tt.name).Scan(&table, &def)
			if tt.columns == "" {
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("index %s: err = %v (definition %q), want it dropped", tt.name, err, def)
				}
				return
			}
			if err != nil {
				t.Fatalf("index %s: %v", tt.name, err)
			}
			if table != tt.table || !strings.HasSuffix(def, "USING btree "+tt.columns) {
				t.Errorf("index %s on %s: %q, want a btree on %s %s", tt.name, table, def, tt.table, tt.columns)
			}
		})
	}
}

// newTrackingUser creates a user with a random "it_" login for the spec 003
// tests and deletes, when the test ends, everything it left: the entries on
// its tickets and the entries it wrote, its tickets, then the user (nothing
// cascades).
func newTrackingUser(t *testing.T, db *sql.DB, role domain.Role) (domain.UserID, string) {
	t.Helper()
	ctx := t.Context()
	login := "it_" + strings.ToLower(rand.Text()[:12])
	id, err := NewUsers(db).Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: role}) //nolint:gosec // fake hash fixture, not a real credential
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() {
		bg := context.WithoutCancel(ctx)
		for _, q := range []string{
			`DELETE FROM ticket_entries WHERE actor_id = $1 OR ticket_id IN (SELECT id FROM tickets WHERE author_id = $1)`,
			`DELETE FROM tickets WHERE author_id = $1`,
			`DELETE FROM users WHERE id = $1`,
		} {
			if _, err := db.ExecContext(bg, q, id); err != nil {
				t.Errorf("cleanup %q for %s: %v", q, login, err)
			}
		}
	})
	return id, login
}

// sampleTickets returns one valid ticket of each type by author, with notes
// on the deploy, as Tickets.Create takes them.
func sampleTickets(author domain.UserID) []domain.Ticket {
	return []domain.Ticket{
		{
			AuthorID: author, Type: domain.TicketTypeAccess, Title: "Acesso a Grafana",
			System: "Grafana", AccessLevel: domain.AccessLevelWrite, Justification: "Preciso editar os painéis.",
		},
		{
			AuthorID: author, Type: domain.TicketTypeBug, Title: "Bug em Portal",
			System: "Portal", Description: "A página não carrega.", Steps: "1. Abrir\n2. Entrar", Severity: domain.SeverityCritical,
		},
		{
			AuthorID: author, Type: domain.TicketTypeDeploy, Title: "Deploy de api em Produção",
			Project: "api", Repository: "https://git.example/api", Environment: domain.EnvironmentProduction, Notes: "Depois das 18h.\nObrigado.",
		},
		{
			AuthorID: author, Type: domain.TicketTypeOther, Title: "Dúvida <b>'; --",
			Description: "Como peço uma VPN?\n\tObrigado.",
		},
	}
}

// TestTicketReadsIntegration covers Tickets.TicketAuthor (the store of
// security.TicketAccess, spec 003 S-2) and Tickets.Get (US-2, US-5): every
// field of each type comes back as stored, with the author's login, and a
// missing ticket is ErrNotFound for both.
func TestTicketReadsIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	repo := NewTickets(db)
	author, login := newTrackingUser(t, db, domain.RoleComum)
	other, _ := newTrackingUser(t, db, domain.RoleComum)

	for _, in := range sampleTickets(author) {
		t.Run(string(in.Type), func(t *testing.T) {
			created, err := repo.Create(ctx, in)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			gotAuthor, err := repo.TicketAuthor(ctx, created.ID)
			if err != nil || gotAuthor != author {
				t.Errorf("TicketAuthor = %d, %v; want %d, nil", gotAuthor, err, author)
			}

			got, gotLogin, err := repo.Get(ctx, created.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if !ticketsEqual(got, created) {
				t.Errorf("Get = %+v, want %+v", got, created)
			}
			if gotLogin != login {
				t.Errorf("author login = %q, want %q", gotLogin, login)
			}
		})
	}

	t.Run("someone else's ticket reads its own author", func(t *testing.T) {
		created, err := repo.Create(ctx, sampleTickets(other)[3])
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if got, err := repo.TicketAuthor(ctx, created.ID); err != nil || got != other {
			t.Errorf("TicketAuthor = %d, %v; want %d, nil", got, err, other)
		}
	})

	t.Run("missing ticket", func(t *testing.T) {
		var maxID domain.TicketID
		if err := db.QueryRowContext(ctx, `SELECT COALESCE(max(id), 0) FROM tickets`).Scan(&maxID); err != nil {
			t.Fatalf("max id: %v", err)
		}
		for _, id := range []domain.TicketID{maxID + 1000, 0, -1, 9223372036854775807} {
			if got, err := repo.TicketAuthor(ctx, id); !errors.Is(err, domain.ErrNotFound) || got != 0 {
				t.Errorf("TicketAuthor(%d) = %d, %v; want ErrNotFound", id, got, err)
			}
			got, gotLogin, err := repo.Get(ctx, id)
			if !errors.Is(err, domain.ErrNotFound) || got != (domain.Ticket{}) || gotLogin != "" {
				t.Errorf("Get(%d) = %+v, %q, %v; want ErrNotFound", id, got, gotLogin, err)
			}
		}
	})
}

// listedTicket is a ticket of the list fixtures, with what the lists show.
type listedTicket struct {
	summary domain.TicketSummary
	author  domain.UserID
}

// insertListedTicket stores a ticket of typ for author, then sets its
// status and its last update, so the list order is known.
func insertListedTicket(t *testing.T, db *sql.DB, author domain.UserID, login string, typ domain.TicketType, status domain.TicketStatus, updatedAt time.Time) listedTicket {
	t.Helper()
	ctx := t.Context()
	var in domain.Ticket
	samples := sampleTickets(author)
	for i := range samples {
		if samples[i].Type == typ {
			in = samples[i]
		}
	}
	created, err := NewTickets(db).Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE tickets SET status = $2, updated_at = $3 WHERE id = $1`,
		created.ID, string(status), updatedAt); err != nil {
		t.Fatalf("set status and updated_at: %v", err)
	}
	return listedTicket{
		author: author,
		summary: domain.TicketSummary{
			ID: created.ID, Type: typ, Status: status, Title: in.Title, AuthorLogin: login,
			CreatedAt: created.CreatedAt, UpdatedAt: updatedAt,
		},
	}
}

// expectedPage filters, orders (last update first, ties by id) and pages
// the fixture tickets the way the lists must.
func expectedPage(all []listedTicket, keep func(listedTicket) bool, page int, withLogin bool) (items []domain.TicketSummary, total int) {
	var kept []listedTicket
	for i := range all {
		if keep(all[i]) {
			kept = append(kept, all[i])
		}
	}
	slices.SortFunc(kept, func(a, b listedTicket) int {
		if c := b.summary.UpdatedAt.Compare(a.summary.UpdatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.summary.ID, a.summary.ID)
	})
	items = make([]domain.TicketSummary, 0)
	for i := (page - 1) * TicketPageSize; i < len(kept) && i < page*TicketPageSize; i++ {
		s := kept[i].summary
		if !withLogin {
			s.AuthorLogin = ""
		}
		items = append(items, s)
	}
	return items, len(kept)
}

func summariesEqual(a, b []domain.TicketSummary) bool {
	return slices.EqualFunc(a, b, func(x, y domain.TicketSummary) bool {
		if !x.CreatedAt.Equal(y.CreatedAt) || !x.UpdatedAt.Equal(y.UpdatedAt) {
			return false
		}
		x.CreatedAt, x.UpdatedAt, y.CreatedAt, y.UpdatedAt = time.Time{}, time.Time{}, time.Time{}, time.Time{}
		return x == y
	})
}

func summaryIDs(items []domain.TicketSummary) []domain.TicketID {
	ids := make([]domain.TicketID, 0, len(items))
	for _, s := range items {
		ids = append(ids, s.ID)
	}
	return ids
}

// TestTicketListsIntegration covers the lists of spec 003 (AC-1.3 to
// AC-1.7, AC-4.3 to AC-4.5; BR-8; S-3, S-14; plan P-9, P-18). The fixture
// tickets get last updates in the year 2100, so they are the newest of the
// queue whatever else the database holds, and pairs of them share a last
// update, so ties are ordered by id.
func TestTicketListsIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	repo := NewTickets(db)

	authorA, loginA := newTrackingUser(t, db, domain.RoleComum)
	authorB, loginB := newTrackingUser(t, db, domain.RoleComum)
	_, loginNone := newTrackingUser(t, db, domain.RoleAdmin) // admins write no ticket
	types := []domain.TicketType{domain.TicketTypeAccess, domain.TicketTypeBug, domain.TicketTypeDeploy, domain.TicketTypeOther}
	statuses := []domain.TicketStatus{domain.TicketStatusOpen, domain.TicketStatusInProgress, domain.TicketStatusResolved, domain.TicketStatusClosed}
	base := time.Date(2100, 1, 1, 12, 0, 0, 0, time.UTC)

	var all []listedTicket
	for i := range 25 { // A: two pages
		all = append(all, insertListedTicket(t, db, authorA, loginA, types[i%4], statuses[(i/4)%4], base.Add(time.Duration(i/2)*time.Minute)))
	}
	for i := range 3 { // B: newer than every ticket of A
		all = append(all, insertListedTicket(t, db, authorB, loginB, types[i], statuses[i], base.Add(time.Hour+time.Duration(i)*time.Minute)))
	}
	byAuthor := func(author domain.UserID) func(listedTicket) bool {
		return func(lt listedTicket) bool { return lt.author == author }
	}

	t.Run("own list pages", func(t *testing.T) {
		for _, tc := range []struct {
			page, items, pages int
		}{{1, 20, 2}, {2, 5, 2}, {3, 0, 2}, {10000, 0, 2}} {
			got, err := repo.ListOwn(ctx, authorA, domain.TicketListQuery{Page: tc.page})
			if err != nil {
				t.Fatalf("ListOwn page %d: %v", tc.page, err)
			}
			want, total := expectedPage(all, byAuthor(authorA), tc.page, false)
			if got.Total != 25 || total != 25 || got.Pages != tc.pages || got.Page != tc.page || len(got.Items) != tc.items {
				t.Errorf("page %d: total %d, pages %d, page %d, %d items; want 25, %d, %d, %d",
					tc.page, got.Total, got.Pages, got.Page, len(got.Items), tc.pages, tc.page, tc.items)
			}
			if !summariesEqual(got.Items, want) {
				t.Errorf("page %d: ids %v, want %v", tc.page, summaryIDs(got.Items), summaryIDs(want))
			}
		}
	})

	t.Run("own list holds only the author's tickets", func(t *testing.T) {
		// S-3: an author login in the query is ignored by the own list.
		got, err := repo.ListOwn(ctx, authorB, domain.TicketListQuery{AuthorLogin: loginA, Page: 1})
		if err != nil {
			t.Fatalf("ListOwn: %v", err)
		}
		want, total := expectedPage(all, byAuthor(authorB), 1, false)
		if got.Total != total || !summariesEqual(got.Items, want) {
			t.Errorf("B's list = %v (total %d), want %v (total %d)", summaryIDs(got.Items), got.Total, summaryIDs(want), total)
		}
		for _, s := range got.Items {
			if s.AuthorLogin != "" {
				t.Errorf("own list item %d carries the author login %q", s.ID, s.AuthorLogin)
			}
		}
	})

	t.Run("own list filters", func(t *testing.T) {
		var queries []domain.TicketListQuery
		for _, s := range statuses {
			queries = append(queries, domain.TicketListQuery{Status: s, Page: 1})
			for _, ty := range types {
				queries = append(queries, domain.TicketListQuery{Status: s, Type: ty, Page: 1})
			}
		}
		for _, ty := range types {
			queries = append(queries, domain.TicketListQuery{Type: ty, Page: 1})
		}
		for _, q := range queries {
			got, err := repo.ListOwn(ctx, authorA, q)
			if err != nil {
				t.Fatalf("ListOwn(%+v): %v", q, err)
			}
			want, total := expectedPage(all, func(lt listedTicket) bool {
				return lt.author == authorA &&
					(q.Status == "" || lt.summary.Status == q.Status) &&
					(q.Type == "" || lt.summary.Type == q.Type)
			}, 1, false)
			if got.Total != total || got.Pages != max(1, (total+19)/20) || !summariesEqual(got.Items, want) {
				t.Errorf("ListOwn(%+v) = %v (total %d, pages %d), want %v (total %d)",
					q, summaryIDs(got.Items), got.Total, got.Pages, summaryIDs(want), total)
			}
		}
	})

	t.Run("queue holds every author, newest first", func(t *testing.T) {
		got, _, err := repo.ListQueue(ctx, domain.TicketListQuery{Page: 1})
		if err != nil {
			t.Fatalf("ListQueue: %v", err)
		}
		var total int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM tickets`).Scan(&total); err != nil {
			t.Fatalf("count: %v", err)
		}
		// The fixture is the newest of the database, so page 1 is its 20
		// newest tickets, B's three first.
		want, _ := expectedPage(all, func(listedTicket) bool { return true }, 1, true)
		if got.Total != total || got.Pages != (total+19)/20 || !summariesEqual(got.Items, want) {
			t.Errorf("queue page 1 = %v (total %d, pages %d), want %v (total %d)",
				summaryIDs(got.Items), got.Total, got.Pages, summaryIDs(want), total)
		}
	})

	t.Run("queue filters", func(t *testing.T) {
		queries := []domain.TicketListQuery{
			{AuthorLogin: loginA, Page: 1},
			{AuthorLogin: loginA, Page: 2},
			{AuthorLogin: loginB, Page: 1},
		}
		for _, login := range []string{loginA, loginB} {
			for _, s := range statuses {
				queries = append(queries, domain.TicketListQuery{AuthorLogin: login, Status: s, Page: 1})
				for _, ty := range types {
					queries = append(queries, domain.TicketListQuery{AuthorLogin: login, Status: s, Type: ty, Page: 1})
				}
			}
			for _, ty := range types {
				queries = append(queries, domain.TicketListQuery{AuthorLogin: login, Type: ty, Page: 1})
			}
		}
		logins := map[domain.UserID]string{authorA: loginA, authorB: loginB}
		for _, q := range queries {
			got, _, err := repo.ListQueue(ctx, q)
			if err != nil {
				t.Fatalf("ListQueue(%+v): %v", q, err)
			}
			want, total := expectedPage(all, func(lt listedTicket) bool {
				return logins[lt.author] == q.AuthorLogin &&
					(q.Status == "" || lt.summary.Status == q.Status) &&
					(q.Type == "" || lt.summary.Type == q.Type)
			}, q.Page, true)
			if got.Total != total || !summariesEqual(got.Items, want) {
				t.Errorf("ListQueue(%+v) = %v (total %d), want %v (total %d)",
					q, summaryIDs(got.Items), got.Total, summaryIDs(want), total)
			}
		}

		// Status or type alone, over the whole queue: every item matches,
		// and the fixture's matching tickets come first.
		for _, q := range []domain.TicketListQuery{
			{Status: domain.TicketStatusResolved, Page: 1},
			{Type: domain.TicketTypeDeploy, Page: 1},
			{Status: domain.TicketStatusClosed, Type: domain.TicketTypeBug, Page: 1},
		} {
			got, _, err := repo.ListQueue(ctx, q)
			if err != nil {
				t.Fatalf("ListQueue(%+v): %v", q, err)
			}
			var total int
			if err := db.QueryRowContext(ctx, `SELECT count(*) FROM tickets WHERE ($1 = '' OR status = $1) AND ($2 = '' OR type = $2)`,
				string(q.Status), string(q.Type)).Scan(&total); err != nil {
				t.Fatalf("count: %v", err)
			}
			want, fixture := expectedPage(all, func(lt listedTicket) bool {
				return (q.Status == "" || lt.summary.Status == q.Status) && (q.Type == "" || lt.summary.Type == q.Type)
			}, 1, true)
			if got.Total != total || len(got.Items) < min(fixture, TicketPageSize) ||
				!summariesEqual(got.Items[:len(want)], want) {
				t.Errorf("ListQueue(%+v) = %v (total %d), want it to start with %v (total %d)",
					q, summaryIDs(got.Items), got.Total, summaryIDs(want), total)
			}
			for _, s := range got.Items {
				if (q.Status != "" && s.Status != q.Status) || (q.Type != "" && s.Type != q.Type) {
					t.Errorf("ListQueue(%+v) holds %+v", q, s)
				}
			}
		}

		// An author with no ticket, or no such login: an empty page.
		for _, login := range []string{loginNone, "it_nobody_" + strings.ToLower(rand.Text()[:8])} {
			got, _, err := repo.ListQueue(ctx, domain.TicketListQuery{AuthorLogin: login, Page: 1})
			if err != nil || got.Total != 0 || got.Pages != 1 || len(got.Items) != 0 || got.Items == nil {
				t.Errorf("ListQueue(%q) = %+v, %v; want an empty page", login, got, err)
			}
		}
	})

	t.Run("status counts cover the whole queue", func(t *testing.T) {
		var want domain.StatusCounts
		err := db.QueryRowContext(ctx, `SELECT
				count(*) FILTER (WHERE status = 'aberto'), count(*) FILTER (WHERE status = 'em_andamento'),
				count(*) FILTER (WHERE status = 'resolvido'), count(*) FILTER (WHERE status = 'fechado')
			FROM tickets`).Scan(&want.Open, &want.InProgress, &want.Resolved, &want.Closed)
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		for _, q := range []domain.TicketListQuery{
			{Page: 1},
			{Status: domain.TicketStatusClosed, Page: 1},
			{AuthorLogin: loginB, Type: domain.TicketTypeBug, Page: 3},
		} {
			_, got, err := repo.ListQueue(ctx, q)
			if err != nil || got != want {
				t.Errorf("ListQueue(%+v) counts = %+v, %v; want %+v", q, got, err, want)
			}
		}
	})

	t.Run("queue authors", func(t *testing.T) {
		got, err := repo.QueueAuthors(ctx, MaxQueueAuthors)
		if err != nil {
			t.Fatalf("QueueAuthors: %v", err)
		}
		if !slices.Contains(got, loginA) || !slices.Contains(got, loginB) {
			t.Errorf("QueueAuthors lacks %q or %q", loginA, loginB)
		}
		if slices.Contains(got, loginNone) {
			t.Errorf("QueueAuthors holds %q, who has no ticket", loginNone)
		}
		if !slices.IsSorted(got) || len(slices.Compact(slices.Clone(got))) != len(got) {
			t.Errorf("QueueAuthors is not sorted without repeats: %v", got)
		}
		var withTickets int
		if err := db.QueryRowContext(ctx, `SELECT count(DISTINCT author_id) FROM tickets`).Scan(&withTickets); err != nil {
			t.Fatalf("count: %v", err)
		}
		if len(got) != min(withTickets, MaxQueueAuthors) {
			t.Errorf("QueueAuthors returned %d logins, want %d", len(got), min(withTickets, MaxQueueAuthors))
		}
		for _, limit := range []int{1, 2} {
			got, err := repo.QueueAuthors(ctx, limit)
			if err != nil || len(got) != limit {
				t.Errorf("QueueAuthors(%d) = %d logins, %v", limit, len(got), err)
			}
		}
	})

	t.Run("summaries never select the type fields", func(t *testing.T) {
		for _, q := range []string{ownListQuery, queueQuery} {
			selected, _, _ := strings.Cut(q, "FROM")
			for _, column := range []string{"system", "access_level", "justification", "description", "steps", "severity", "project", "repository", "environment", "notes", "author_id,", "password_hash"} {
				if strings.Contains(selected, column) {
					t.Errorf("list query selects %s: %s", column, q)
				}
			}
		}
	})
}

// TestTicketListPlansIntegration checks, on 2 000 extra tickets, that the
// list queries use the indexes of migration 00003 (plan section 1, S-14)
// instead of reading the whole table.
func TestTicketListPlansIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	// 2 000 tickets of one bulk author, and 30 of the author whose plans are
	// read, like a user among many.
	bulk, _ := newTrackingUser(t, db, domain.RoleComum)
	author, login := newTrackingUser(t, db, domain.RoleComum)
	for _, fill := range []struct {
		author domain.UserID
		n      int
	}{{bulk, 2000}, {author, 30}} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO tickets (author_id, type, status, title, description)
			SELECT $1, 'outro', (ARRAY['aberto', 'em_andamento', 'resolvido', 'fechado'])[1 + g % 4],
			       'Dúvida ' || g, 'Como peço uma VPN?'
			FROM generate_series(1, $2::int) AS g`, fill.author, fill.n); err != nil {
			t.Fatalf("insert %d tickets: %v", fill.n, err)
		}
	}
	if _, err := db.ExecContext(ctx, `ANALYZE tickets`); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	// explain returns the plan of query; noSeqScan makes it the plan with
	// sequential scans disabled, which shows an index can serve the query.
	explain := func(t *testing.T, query string, args []any, noSeqScan bool) string {
		t.Helper()
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer func() { _ = tx.Rollback() }()
		if noSeqScan {
			if _, err = tx.ExecContext(ctx, `SET LOCAL enable_seqscan = off`); err != nil {
				t.Fatalf("set: %v", err)
			}
		}
		rows, err := tx.QueryContext(ctx, "EXPLAIN "+query, args...)
		if err != nil {
			t.Fatalf("explain: %v", err)
		}
		defer rows.Close()
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatalf("scan: %v", err)
			}
			plan.WriteString(line + "\n")
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows: %v", err)
		}
		return plan.String()
	}

	page := []any{TicketPageSize, 0}
	byAuthor := []string{"tickets_author_updated_idx"}
	tests := []struct {
		name    string
		query   string
		args    []any
		indexes []string // the plan uses one of them, with no sequential scan of tickets
	}{
		{"own list", ownListQuery, append([]any{author, "", ""}, page...), byAuthor},
		{"own list by status", ownListQuery, append([]any{author, "resolvido", ""}, page...), byAuthor},
		{"queue without filter", queueQuery, append([]any{"", "", ""}, page...), []string{"tickets_updated_idx"}},
		{"queue by type", queueQuery, append([]any{"", "outro", ""}, page...), []string{"tickets_updated_idx"}},
		{"queue by status", queueQuery, append([]any{"em_andamento", "", ""}, page...), []string{"tickets_status_updated_idx"}},
		{"queue by author", queueQuery, append([]any{"", "", login}, page...), byAuthor},
		{"queue authors", queueAuthorsQuery, []any{MaxQueueAuthors}, byAuthor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := explain(t, tt.query, tt.args, false)
			uses := slices.ContainsFunc(tt.indexes, func(index string) bool { return strings.Contains(plan, index) })
			if !uses || strings.Contains(plan, "Seq Scan on tickets") {
				t.Errorf("plan does not use %v without a sequential scan:\n%s", tt.indexes, plan)
			}
		})
	}

	// The status counts read every ticket, so at this size a sequential
	// scan is the cheapest plan and PostgreSQL takes it; the status index
	// can serve the count (an index-only scan), which the plan shows with
	// sequential scans disabled. Either way the counts need no sort.
	t.Run("status counts", func(t *testing.T) {
		if plan := explain(t, statusCountsQuery, nil, true); !strings.Contains(plan, "Index Only Scan using tickets_status_updated_idx") &&
			!strings.Contains(plan, "Index Scan using tickets_status_updated_idx") {
			t.Errorf("the status index cannot serve the counts:\n%s", plan)
		}
		if plan := explain(t, statusCountsQuery, nil, false); strings.Contains(plan, "Sort") {
			t.Errorf("the counts sort the table:\n%s", plan)
		}
	})
}

// insertEntries adds n entries to ticket, oldest first: messages by author
// with the text "m<i>", and every third entry a status change by admin
// (aberto → em_andamento, only the table is tested here). It returns their
// ids in order.
func insertEntries(t *testing.T, db *sql.DB, ticket domain.TicketID, author, admin domain.UserID, n int) []domain.TicketEntryID {
	t.Helper()
	ids := make([]domain.TicketEntryID, 0, n)
	for i := range n {
		row := entryRow{ticketID: ticket, actorID: author, kind: "message", body: ptr(fmt.Sprintf("m%d", i))}
		if i%3 == 2 {
			row = entryRow{ticketID: ticket, actorID: admin, kind: "status_change", from: ptr("aberto"), to: ptr("em_andamento")}
		}
		id, err := insertEntry(t.Context(), db, row)
		if err != nil {
			t.Fatalf("insert entry %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

func entryIDs(entries []domain.TicketEntry) []domain.TicketEntryID {
	ids := make([]domain.TicketEntryID, 0, len(entries))
	for i := range entries {
		ids = append(ids, entries[i].ID)
	}
	return ids
}

// TestThreadIntegration covers TicketEntries.Thread (spec 003 AC-3.1,
// AC-3.2, AC-3.4a; BR-4; S-14; plan P-4): the latest 100 entries oldest
// first, older pages with before, has_more, and only the ticket's own
// entries.
func TestThreadIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	entries := NewTicketEntries(db)
	author, authorLogin := newTrackingUser(t, db, domain.RoleComum)
	admin, adminLogin := newTrackingUser(t, db, domain.RoleAdmin)

	newTicket := func() domain.TicketID {
		created, err := NewTickets(db).Create(ctx, sampleTickets(author)[3])
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		return created.ID
	}
	long, exact, empty, other := newTicket(), newTicket(), newTicket(), newTicket()
	// 5 000 older entries on another ticket, so the thread read is one
	// ticket among many, before and after the cursor, as in use.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO ticket_entries (ticket_id, actor_id, kind, body)
		SELECT $1, $2, 'message', 'bulk ' || g FROM generate_series(1, 5000) AS g`, other, author); err != nil {
		t.Fatalf("insert bulk entries: %v", err)
	}
	ids := insertEntries(t, db, long, author, admin, 250)
	insertEntries(t, db, other, author, admin, 5)
	// Interleaved with long's last entries, so a missing ticket filter shows.
	insertEntries(t, db, other, author, admin, 3)
	exactIDs := insertEntries(t, db, exact, author, admin, ThreadPageSize)

	t.Run("pages from the latest", func(t *testing.T) {
		tests := []struct {
			name    string
			before  domain.TicketEntryID
			want    []domain.TicketEntryID
			hasMore bool
		}{
			{"latest", 0, ids[150:], true},
			{"before the latest page", ids[150], ids[50:150], true},
			{"the oldest", ids[50], ids[:50], false},
			{"before the first entry", ids[0], nil, false},
			{"before an entry in the middle", ids[10], ids[:10], false},
			{"before an id past the end", ids[249] + 1_000_000, ids[150:], true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := entries.Thread(ctx, long, tt.before, ThreadPageSize)
				if err != nil {
					t.Fatalf("Thread: %v", err)
				}
				if !slices.Equal(entryIDs(got.Entries), tt.want) && len(got.Entries)+len(tt.want) > 0 {
					t.Errorf("Thread(before %d) = %v, want %v", tt.before, entryIDs(got.Entries), tt.want)
				}
				if got.HasMore != tt.hasMore {
					t.Errorf("HasMore = %v, want %v", got.HasMore, tt.hasMore)
				}
				if got.Entries == nil {
					t.Error("Entries is nil, want an empty slice")
				}
				for _, e := range got.Entries {
					if e.TicketID != long {
						t.Errorf("entry %d of ticket %d in the thread of %d", e.ID, e.TicketID, long)
					}
				}
			})
		}
	})

	t.Run("exactly one page", func(t *testing.T) {
		got, err := entries.Thread(ctx, exact, 0, ThreadPageSize)
		if err != nil || !slices.Equal(entryIDs(got.Entries), exactIDs) || got.HasMore {
			t.Errorf("Thread = %v (has more %v), %v; want all %d entries and no more", entryIDs(got.Entries), got.HasMore, err, len(exactIDs))
		}
	})

	t.Run("no entries", func(t *testing.T) {
		got, err := entries.Thread(ctx, empty, 0, ThreadPageSize)
		if err != nil || len(got.Entries) != 0 || got.Entries == nil || got.HasMore {
			t.Errorf("Thread = %+v, %v; want an empty page", got, err)
		}
	})

	t.Run("limit is bounded", func(t *testing.T) {
		for _, limit := range []int{0, -1, ThreadPageSize + 1, 1 << 30} {
			got, err := entries.Thread(ctx, long, 0, limit)
			if err != nil || len(got.Entries) != ThreadPageSize {
				t.Errorf("Thread(limit %d) = %d entries, %v; want %d", limit, len(got.Entries), err, ThreadPageSize)
			}
		}
		got, err := entries.Thread(ctx, long, 0, 7)
		if err != nil || !slices.Equal(entryIDs(got.Entries), ids[243:]) || !got.HasMore {
			t.Errorf("Thread(limit 7) = %v, %v; want the 7 latest", entryIDs(got.Entries), err)
		}
	})

	t.Run("entry fields", func(t *testing.T) {
		got, err := entries.Thread(ctx, long, ids[3], ThreadPageSize) // ids[0..2]
		if err != nil || len(got.Entries) != 3 {
			t.Fatalf("Thread = %d entries, %v", len(got.Entries), err)
		}
		message, change := got.Entries[0], got.Entries[2]
		wantMessage := domain.TicketEntry{
			ID: ids[0], TicketID: long, ActorID: author, ActorLogin: authorLogin, ActorRole: domain.RoleComum,
			Kind: domain.EntryKindMessage, Body: "m0", CreatedAt: message.CreatedAt,
		}
		wantChange := domain.TicketEntry{
			ID: ids[2], TicketID: long, ActorID: admin, ActorLogin: adminLogin, ActorRole: domain.RoleAdmin,
			Kind: domain.EntryKindStatusChange, From: domain.TicketStatusOpen, To: domain.TicketStatusInProgress, CreatedAt: change.CreatedAt,
		}
		if message != wantMessage || message.CreatedAt.IsZero() {
			t.Errorf("message = %+v, want %+v", message, wantMessage)
		}
		if change != wantChange || change.CreatedAt.IsZero() {
			t.Errorf("status change = %+v, want %+v", change, wantChange)
		}
	})

	t.Run("plan uses the thread index", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, `ANALYZE ticket_entries`); err != nil {
			t.Fatalf("analyze: %v", err)
		}
		explain := func(before domain.TicketEntryID) string {
			rows, err := db.QueryContext(ctx, "EXPLAIN "+threadQuery, int64(long), int64(before), ThreadPageSize+1)
			if err != nil {
				t.Fatalf("explain: %v", err)
			}
			defer rows.Close()
			var plan strings.Builder
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					t.Fatalf("scan: %v", err)
				}
				plan.WriteString(line + "\n")
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("rows: %v", err)
			}
			return plan.String()
		}
		for _, before := range []domain.TicketEntryID{0, ids[150]} {
			if plan := explain(before); !strings.Contains(plan, "ticket_entries_ticket_idx") || strings.Contains(plan, "Seq Scan on ticket_entries") {
				t.Errorf("before %d: plan does not use ticket_entries_ticket_idx:\n%s", before, plan)
			}
		}
	})
}

// openConcurrentTestDB opens a pool with room for a lock holder and the
// writers that wait behind it (openTestDB keeps 2 connections).
func openConcurrentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	db, err := Open(t.Context(), url, PoolOptions{MaxOpenConns: 12, MaxIdleConns: 12})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ticketState is what a write may change on a ticket.
type ticketState struct {
	status    domain.TicketStatus
	updatedAt time.Time
	entries   int
}

func readTicketState(ctx context.Context, t *testing.T, db *sql.DB, ticket domain.TicketID) ticketState {
	t.Helper()
	var s ticketState
	err := db.QueryRowContext(ctx, `
		SELECT status, updated_at, (SELECT count(*) FROM ticket_entries WHERE ticket_id = $1)
		FROM tickets WHERE id = $1`, ticket).Scan(&s.status, &s.updatedAt, &s.entries)
	if err != nil {
		t.Fatalf("read ticket %d: %v", ticket, err)
	}
	return s
}

// lockTicket holds the row lock of ticket in an open transaction, as a
// write in progress would; release commits it.
func lockTicket(ctx context.Context, t *testing.T, db *sql.DB, ticket domain.TicketID) (release func()) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM tickets WHERE id = $1 FOR UPDATE`, ticket); err != nil {
		_ = tx.Rollback()
		t.Fatalf("lock ticket: %v", err)
	}
	done := false
	t.Cleanup(func() {
		if !done {
			_ = tx.Rollback()
		}
	})
	return func() {
		done = true
		if err := tx.Commit(); err != nil {
			t.Errorf("release lock: %v", err)
		}
	}
}

// waitForLockWaiters waits until n statements of this database are blocked
// on a lock, so the writes started by a test are known to overlap.
func waitForLockWaiters(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting int
		err := db.QueryRowContext(t.Context(), `
			SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE 'UPDATE tickets SET%'`).Scan(&waiting)
		if err != nil {
			t.Fatalf("pg_stat_activity: %v", err)
		}
		if waiting >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d writes waiting on the ticket lock after 10s, want %d", waiting, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type writeResult struct {
	entry domain.TicketEntry
	err   error
}

// TestTicketWritesIntegration covers AddMessage and ChangeStatus one at a
// time (spec 003 BR-4 to BR-8, BR-10; plan section 4).
func TestTicketWritesIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	entries := NewTicketEntries(db)
	author, authorLogin := newTrackingUser(t, db, domain.RoleComum)
	admin, adminLogin := newTrackingUser(t, db, domain.RoleAdmin)
	newTicket := func(status domain.TicketStatus) domain.TicketID {
		created, err := NewTickets(db).Create(ctx, sampleTickets(author)[3])
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE tickets SET status = $2 WHERE id = $1`, created.ID, string(status)); err != nil {
			t.Fatalf("set status: %v", err)
		}
		return created.ID
	}
	var missing domain.TicketID
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(max(id), 0) + 1000 FROM tickets`).Scan(&missing); err != nil {
		t.Fatalf("max id: %v", err)
	}

	t.Run("a message moves updated_at, not the status", func(t *testing.T) {
		for _, status := range []domain.TicketStatus{domain.TicketStatusOpen, domain.TicketStatusInProgress, domain.TicketStatusResolved} {
			ticket := newTicket(status)
			before := readTicketState(ctx, t, db, ticket)
			for _, actor := range []struct {
				id    domain.UserID
				login string
				role  domain.Role
			}{{author, authorLogin, domain.RoleComum}, {admin, adminLogin, domain.RoleAdmin}} {
				got, err := entries.AddMessage(ctx, ticket, actor.id, "Linha 1\nLinha 2")
				if err != nil {
					t.Fatalf("AddMessage on %s: %v", status, err)
				}
				want := domain.TicketEntry{
					ID: got.ID, TicketID: ticket, ActorID: actor.id, ActorLogin: actor.login, ActorRole: actor.role,
					Kind: domain.EntryKindMessage, Body: "Linha 1\nLinha 2", CreatedAt: got.CreatedAt,
				}
				if got != want || got.ID == 0 {
					t.Errorf("AddMessage = %+v, want %+v", got, want)
				}
				after := readTicketState(ctx, t, db, ticket)
				if after.status != status || after.entries != before.entries+1 {
					t.Errorf("after a message: %+v, want status %s and one more entry than %+v", after, status, before)
				}
				if !after.updatedAt.Equal(got.CreatedAt) || !after.updatedAt.After(before.updatedAt) {
					t.Errorf("updated_at %v, entry %v, before %v: want the entry's time, later", after.updatedAt, got.CreatedAt, before.updatedAt)
				}
				before = after
			}
		}
	})

	t.Run("no message on a closed ticket", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusClosed)
		before := readTicketState(ctx, t, db, ticket)
		_, err := entries.AddMessage(ctx, ticket, author, "Ainda está acontecendo.")
		if !errors.Is(err, domain.ErrTicketClosed) || !errors.Is(err, domain.ErrConflict) {
			t.Errorf("AddMessage on fechado = %v, want ErrTicketClosed", err)
		}
		if after := readTicketState(ctx, t, db, ticket); after != before {
			t.Errorf("ticket %+v → %+v: a refused message changed it", before, after)
		}
	})

	t.Run("status changes along BR-3", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusOpen)
		path := []domain.TicketStatus{domain.TicketStatusInProgress, domain.TicketStatusResolved, domain.TicketStatusInProgress, domain.TicketStatusResolved, domain.TicketStatusClosed}
		from := domain.TicketStatusOpen
		for _, to := range path {
			before := readTicketState(ctx, t, db, ticket)
			got, err := entries.ChangeStatus(ctx, ticket, admin, from, to)
			if err != nil {
				t.Fatalf("ChangeStatus %s → %s: %v", from, to, err)
			}
			want := domain.TicketEntry{
				ID: got.ID, TicketID: ticket, ActorID: admin, ActorLogin: adminLogin, ActorRole: domain.RoleAdmin,
				Kind: domain.EntryKindStatusChange, From: from, To: to, CreatedAt: got.CreatedAt,
			}
			if got != want {
				t.Errorf("ChangeStatus = %+v, want %+v", got, want)
			}
			after := readTicketState(ctx, t, db, ticket)
			if after.status != to || after.entries != before.entries+1 || !after.updatedAt.Equal(got.CreatedAt) || !after.updatedAt.After(before.updatedAt) {
				t.Errorf("after %s → %s: %+v (before %+v, entry at %v)", from, to, after, before, got.CreatedAt)
			}
			from = to
		}
	})

	t.Run("stale status writes nothing", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusInProgress)
		before := readTicketState(ctx, t, db, ticket)
		for _, from := range []domain.TicketStatus{domain.TicketStatusOpen, domain.TicketStatusResolved, domain.TicketStatusClosed} {
			_, err := entries.ChangeStatus(ctx, ticket, admin, from, domain.TicketStatusClosed)
			if !errors.Is(err, domain.ErrStaleStatus) || !errors.Is(err, domain.ErrConflict) {
				t.Errorf("ChangeStatus from %s on em_andamento = %v, want ErrStaleStatus", from, err)
			}
		}
		if after := readTicketState(ctx, t, db, ticket); after != before {
			t.Errorf("ticket %+v → %+v: a refused change wrote", before, after)
		}
	})

	t.Run("missing ticket", func(t *testing.T) {
		if _, err := entries.AddMessage(ctx, missing, author, "Oi"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("AddMessage on a missing ticket = %v, want ErrNotFound", err)
		}
		if _, err := entries.ChangeStatus(ctx, missing, admin, domain.TicketStatusOpen, domain.TicketStatusClosed); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("ChangeStatus on a missing ticket = %v, want ErrNotFound", err)
		}
	})

	// BR-10: when the second statement fails, the first one (the ticket's
	// UPDATE, which already ran) is rolled back too.
	t.Run("a failure after the ticket update rolls back both", func(t *testing.T) {
		tests := []struct {
			name  string
			write func(ticket domain.TicketID) error
		}{
			{"message by an unknown actor", func(ticket domain.TicketID) error {
				_, err := entries.AddMessage(ctx, ticket, -1, "Oi")
				return err
			}},
			{"message with an empty body", func(ticket domain.TicketID) error {
				_, err := entries.AddMessage(ctx, ticket, author, "")
				return err
			}},
			{"message over 2000 characters", func(ticket domain.TicketID) error {
				_, err := entries.AddMessage(ctx, ticket, author, chars(2001))
				return err
			}},
			{"change BR-3 forbids, past the service", func(ticket domain.TicketID) error {
				_, err := entries.ChangeStatus(ctx, ticket, admin, domain.TicketStatusOpen, domain.TicketStatusResolved)
				return err
			}},
			{"change by an unknown actor", func(ticket domain.TicketID) error {
				_, err := entries.ChangeStatus(ctx, ticket, -1, domain.TicketStatusOpen, domain.TicketStatusInProgress)
				return err
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ticket := newTicket(domain.TicketStatusOpen)
				before := readTicketState(ctx, t, db, ticket)
				var pgErr *pgconn.PgError
				if err := tt.write(ticket); !errors.As(err, &pgErr) {
					t.Fatalf("write = %v, want the database to refuse the entry", err)
				}
				if after := readTicketState(ctx, t, db, ticket); after != before {
					t.Errorf("ticket %+v → %+v: the UPDATE was not rolled back", before, after)
				}
			})
		}
	})
}

// TestTicketWritesConcurrencyIntegration runs the concurrent cases of plan
// section 4 with real transactions: a test transaction holds the ticket's
// row lock, the writes start and wait on it (checked in pg_stat_activity),
// then the lock is released and PostgreSQL serializes them.
func TestTicketWritesConcurrencyIntegration(t *testing.T) {
	db := openConcurrentTestDB(t)
	ctx := t.Context()
	entries := NewTicketEntries(db)
	author, _ := newTrackingUser(t, db, domain.RoleComum)
	admin, _ := newTrackingUser(t, db, domain.RoleAdmin)
	other, _ := newTrackingUser(t, db, domain.RoleAdmin)
	newTicket := func(status domain.TicketStatus) domain.TicketID {
		created, err := NewTickets(db).Create(ctx, sampleTickets(author)[3])
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE tickets SET status = $2 WHERE id = $1`, created.ID, string(status)); err != nil {
			t.Fatalf("set status: %v", err)
		}
		return created.ID
	}
	// start runs write in a goroutine and waits until it is blocked on the
	// lock, behind waiting writes already started.
	start := func(t *testing.T, waiting int, write func() (domain.TicketEntry, error)) <-chan writeResult {
		t.Helper()
		out := make(chan writeResult, 1)
		go func() {
			e, err := write()
			out <- writeResult{e, err}
		}()
		waitForLockWaiters(t, db, waiting)
		return out
	}
	statusChanges := func(t *testing.T, ticket domain.TicketID) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM ticket_entries WHERE ticket_id = $1 AND kind = 'status_change'`, ticket).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}

	t.Run("two admins change from the same status: one applies", func(t *testing.T) {
		for round := range 5 {
			ticket := newTicket(domain.TicketStatusOpen)
			release := lockTicket(ctx, t, db, ticket)
			first := start(t, 1, func() (domain.TicketEntry, error) {
				return entries.ChangeStatus(ctx, ticket, admin, domain.TicketStatusOpen, domain.TicketStatusInProgress)
			})
			second := start(t, 2, func() (domain.TicketEntry, error) {
				return entries.ChangeStatus(ctx, ticket, other, domain.TicketStatusOpen, domain.TicketStatusClosed)
			})
			release()
			a, b := <-first, <-second
			if a.err != nil || !errors.Is(b.err, domain.ErrStaleStatus) {
				t.Errorf("round %d: first = %v, second = %v; want the first applied and the second ErrStaleStatus", round, a.err, b.err)
			}
			if s := readTicketState(ctx, t, db, ticket); s.status != domain.TicketStatusInProgress || statusChanges(t, ticket) != 1 {
				t.Errorf("round %d: status %s with %d history entries, want em_andamento and 1", round, s.status, statusChanges(t, ticket))
			}
		}
	})

	t.Run("eight admins at once: exactly one applies", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusResolved)
		release := lockTicket(ctx, t, db, ticket)
		results := make([]<-chan writeResult, 0, 8)
		for i := range 8 {
			to := domain.TicketStatusInProgress
			if i%2 == 1 {
				to = domain.TicketStatusClosed
			}
			results = append(results, start(t, i+1, func() (domain.TicketEntry, error) {
				return entries.ChangeStatus(ctx, ticket, admin, domain.TicketStatusResolved, to)
			}))
		}
		release()
		applied, stale := 0, 0
		var winner domain.TicketEntry
		for _, ch := range results {
			r := <-ch
			switch {
			case r.err == nil:
				applied++
				winner = r.entry
			case errors.Is(r.err, domain.ErrStaleStatus):
				stale++
			default:
				t.Errorf("unexpected error: %v", r.err)
			}
		}
		s := readTicketState(ctx, t, db, ticket)
		if applied != 1 || stale != 7 || statusChanges(t, ticket) != 1 || s.status != winner.To {
			t.Errorf("%d applied, %d stale, %d history entries, status %s (winner %s); want 1, 7, 1 and the winner's", applied, stale, statusChanges(t, ticket), s.status, winner.To)
		}
	})

	t.Run("close first, then the reply: the reply is refused", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusInProgress)
		release := lockTicket(ctx, t, db, ticket)
		closing := start(t, 1, func() (domain.TicketEntry, error) {
			return entries.ChangeStatus(ctx, ticket, admin, domain.TicketStatusInProgress, domain.TicketStatusClosed)
		})
		reply := start(t, 2, func() (domain.TicketEntry, error) {
			return entries.AddMessage(ctx, ticket, author, "Ainda está acontecendo.")
		})
		release()
		c, r := <-closing, <-reply
		if c.err != nil || !errors.Is(r.err, domain.ErrTicketClosed) {
			t.Errorf("close = %v, reply = %v; want the close applied and the reply ErrTicketClosed", c.err, r.err)
		}
		if s := readTicketState(ctx, t, db, ticket); s.status != domain.TicketStatusClosed || s.entries != 1 {
			t.Errorf("ticket %+v, want fechado with only the status change", s)
		}
	})

	t.Run("reply first, then the close: both stored, the reply before", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusInProgress)
		release := lockTicket(ctx, t, db, ticket)
		reply := start(t, 1, func() (domain.TicketEntry, error) {
			return entries.AddMessage(ctx, ticket, author, "Ainda está acontecendo.")
		})
		closing := start(t, 2, func() (domain.TicketEntry, error) {
			return entries.ChangeStatus(ctx, ticket, admin, domain.TicketStatusInProgress, domain.TicketStatusClosed)
		})
		release()
		r, c := <-reply, <-closing
		if r.err != nil || c.err != nil {
			t.Fatalf("reply = %v, close = %v; want both stored", r.err, c.err)
		}
		s := readTicketState(ctx, t, db, ticket)
		if s.status != domain.TicketStatusClosed || s.entries != 2 || r.entry.ID >= c.entry.ID || r.entry.CreatedAt.After(c.entry.CreatedAt) {
			t.Errorf("ticket %+v, reply %d at %v, close %d at %v; want fechado, the reply first", s, r.entry.ID, r.entry.CreatedAt, c.entry.ID, c.entry.CreatedAt)
		}
	})

	t.Run("a reply and a status change: both stored, updated_at the later", func(t *testing.T) {
		for _, replyFirst := range []bool{true, false} {
			ticket := newTicket(domain.TicketStatusOpen)
			release := lockTicket(ctx, t, db, ticket)
			reply := func() (domain.TicketEntry, error) { return entries.AddMessage(ctx, ticket, author, "Mais um detalhe.") }
			change := func() (domain.TicketEntry, error) {
				return entries.ChangeStatus(ctx, ticket, admin, domain.TicketStatusOpen, domain.TicketStatusInProgress)
			}
			var r, c <-chan writeResult
			if replyFirst {
				r, c = start(t, 1, reply), start(t, 2, change)
			} else {
				c, r = start(t, 1, change), start(t, 2, reply)
			}
			release()
			rr, cr := <-r, <-c
			if rr.err != nil || cr.err != nil {
				t.Fatalf("reply = %v, change = %v; want both stored (a reply never makes a change fail, P-6)", rr.err, cr.err)
			}
			last := cr.entry
			if replyFirst != (rr.entry.ID < cr.entry.ID) {
				t.Errorf("reply %d, change %d: not in lock order (reply first: %v)", rr.entry.ID, cr.entry.ID, replyFirst)
			}
			if rr.entry.ID > cr.entry.ID {
				last = rr.entry
			}
			s := readTicketState(ctx, t, db, ticket)
			if s.status != domain.TicketStatusInProgress || s.entries != 2 || !s.updatedAt.Equal(last.CreatedAt) ||
				rr.entry.CreatedAt.After(s.updatedAt) || cr.entry.CreatedAt.After(s.updatedAt) {
				t.Errorf("ticket %+v, reply at %v, change at %v; want em_andamento, 2 entries, updated_at of the later", s, rr.entry.CreatedAt, cr.entry.CreatedAt)
			}
		}
	})

	t.Run("updated_at never goes back while writes wait", func(t *testing.T) {
		ticket := newTicket(domain.TicketStatusOpen)
		release := lockTicket(ctx, t, db, ticket)
		results := make([]<-chan writeResult, 0, 6)
		for i := range 6 {
			results = append(results, start(t, i+1, func() (domain.TicketEntry, error) {
				return entries.AddMessage(ctx, ticket, author, fmt.Sprintf("Mensagem %d", i))
			}))
		}
		release()
		var stored []domain.TicketEntry
		for _, ch := range results {
			r := <-ch
			if r.err != nil {
				t.Fatalf("AddMessage: %v", r.err)
			}
			stored = append(stored, r.entry)
		}
		slices.SortFunc(stored, func(a, b domain.TicketEntry) int { return cmp.Compare(a.ID, b.ID) })
		for i := 1; i < len(stored); i++ {
			if stored[i].CreatedAt.Before(stored[i-1].CreatedAt) {
				t.Errorf("entry %d at %v is before entry %d at %v", stored[i].ID, stored[i].CreatedAt, stored[i-1].ID, stored[i-1].CreatedAt)
			}
		}
		if s := readTicketState(ctx, t, db, ticket); !s.updatedAt.Equal(stored[len(stored)-1].CreatedAt) {
			t.Errorf("updated_at %v, want the last entry's %v", s.updatedAt, stored[len(stored)-1].CreatedAt)
		}
	})
}
