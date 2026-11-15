package repository

// Testes de integração: rodam apenas com TEST_DATABASE_URL apontando para um
// banco com as migrations aplicadas. Sem a variável, são ignorados.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

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
	// Injeção clássica tratada como dado, não como SQL.
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
