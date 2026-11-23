package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

// createTestUser stores a comum user that is deleted, with its posts
// (ON DELETE CASCADE), when the test ends.
func createTestUser(t *testing.T, users *Users) domain.UserID {
	t.Helper()
	ctx := t.Context()
	login := "it_" + strings.ToLower(rand.Text()[:10])
	id, err := users.Create(ctx, domain.User{Login: login, PasswordHash: "$argon2id$fake", Role: domain.RoleComum}) //nolint:gosec // fake hash fixture, not a real credential
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := users.db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM users WHERE id = $1`, int64(id)); err != nil {
			t.Errorf("cleanup: delete test user %d: %v", id, err)
		}
	})
	return id
}

// TestPostsOwnerCheckIntegration covers 005 S-1 against PostgreSQL: Update
// and Delete by another author find no row, and the post stays unchanged.
func TestPostsOwnerCheckIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	users, posts := NewUsers(db), NewPosts(db)
	owner, other := createTestUser(t, users), createTestUser(t, users)

	created, err := posts.Create(ctx, domain.Post{AuthorID: owner, Title: "original", Body: "texto"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.AuthorID != owner || created.AuthorLogin == "" || created.CreatedAt.IsZero() {
		t.Fatalf("Create = %+v", created)
	}

	_, err = posts.Update(ctx, domain.Post{ID: created.ID, AuthorID: other, Title: "hacked", Body: "x"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update by another author: error = %v, want ErrNotFound", err)
	}
	if err = posts.Delete(ctx, created.ID, other); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete by another author: error = %v, want ErrNotFound", err)
	}
	if err = posts.Delete(ctx, created.ID+1_000_000, owner); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete of a missing post: error = %v, want ErrNotFound", err)
	}

	var title, body string
	if err = db.QueryRowContext(ctx, `SELECT title, body FROM posts WHERE id = $1`, int64(created.ID)).Scan(&title, &body); err != nil {
		t.Fatalf("post after another author's attempts: %v", err)
	}
	if title != "original" || body != "texto" {
		t.Errorf("post after another author's attempts = %q, %q; want it unchanged", title, body)
	}

	updated, err := posts.Update(ctx, domain.Post{ID: created.ID, AuthorID: owner, Title: "novo", Body: "outro"})
	if err != nil || updated.Title != "novo" || updated.Body != "outro" || updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Errorf("Update by the owner = %+v, %v", updated, err)
	}
	if err := posts.Delete(ctx, created.ID, owner); err != nil {
		t.Errorf("Delete by the owner: %v", err)
	}
	if err := posts.Delete(ctx, created.ID, owner); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second Delete: error = %v, want ErrNotFound", err)
	}
}

// TestPostsListAndChecksIntegration covers the order of the wall (005 D-5)
// and the CHECK constraints that back the validation (D-4).
func TestPostsListAndChecksIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	users, posts := NewUsers(db), NewPosts(db)
	author := createTestUser(t, users)

	var ids []domain.PostID
	for _, title := range []string{"first", "second", "third"} {
		p, err := posts.Create(ctx, domain.Post{AuthorID: author, Title: title, Body: "b"})
		if err != nil {
			t.Fatalf("Create %s: %v", title, err)
		}
		ids = append(ids, p.ID)
	}

	// Other tests may write posts at the same time, so only the relative
	// order of this test's posts is checked.
	got, err := posts.List(ctx, author, false, 1000, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var mine []domain.PostID
	for _, p := range got {
		if p.AuthorID == author {
			mine = append(mine, p.ID)
		}
	}
	if len(mine) != 3 || mine[0] != ids[2] || mine[1] != ids[1] || mine[2] != ids[0] {
		t.Errorf("List order = %v, want %v reversed (newest first)", mine, ids)
	}
	if page, err := posts.List(ctx, author, false, 1, 0); err != nil || len(page) != 1 {
		t.Errorf("List(limit 1) = %d posts, %v", len(page), err)
	}

	for _, p := range []domain.Post{
		{AuthorID: author, Title: strings.Repeat("é", 121), Body: "b"},
		{AuthorID: author, Title: "t", Body: strings.Repeat("b", 2001)},
		{AuthorID: author, Title: "", Body: "b"},
		{AuthorID: author, Title: "t", Body: ""},
	} {
		if _, err := posts.Create(ctx, p); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Create(title %d chars, body %d chars): error = %v, want ErrInvalidInput from the CHECK",
				len([]rune(p.Title)), len([]rune(p.Body)), err)
		}
	}
	if _, err := posts.Create(ctx, domain.Post{AuthorID: author, Title: strings.Repeat("é", 120), Body: strings.Repeat("ç", 2000)}); err != nil {
		t.Errorf("Create at the limits (in characters, not bytes): %v", err)
	}
}
