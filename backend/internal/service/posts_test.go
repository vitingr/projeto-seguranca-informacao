package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"backend_si/internal/domain"
)

// memPosts is an in-memory PostRepository with the owner condition of the
// SQL statements (005 D-3). List returns the newest (highest ID) first.
type memPosts struct {
	mu     sync.Mutex
	posts  map[domain.PostID]domain.Post
	nextID domain.PostID
	limit  int // arguments of the last List call
	offset int
}

func newMemPosts() *memPosts { return &memPosts{posts: map[domain.PostID]domain.Post{}} }

func (m *memPosts) List(_ context.Context, limit, offset int) ([]domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limit, m.offset = limit, offset
	all := make([]domain.Post, 0, len(m.posts))
	for _, p := range m.posts {
		all = append(all, p)
	}
	slices.SortFunc(all, func(a, b domain.Post) int { return int(b.ID - a.ID) })
	if offset >= len(all) {
		return []domain.Post{}, nil
	}
	return all[offset:min(offset+limit, len(all))], nil
}

func (m *memPosts) Create(_ context.Context, p domain.Post) (domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	p.ID = m.nextID
	m.posts[p.ID] = p
	return p, nil
}

func (m *memPosts) Update(_ context.Context, p domain.Post) (domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.posts[p.ID]
	if !ok || old.AuthorID != p.AuthorID {
		return domain.Post{}, domain.ErrNotFound
	}
	old.Title, old.Body = p.Title, p.Body
	m.posts[p.ID] = old
	return old, nil
}

func (m *memPosts) Delete(_ context.Context, id domain.PostID, author domain.UserID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.posts[id]
	if !ok || old.AuthorID != author {
		return domain.ErrNotFound
	}
	delete(m.posts, id)
	return nil
}

func (m *memPosts) get(id domain.PostID) (domain.Post, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.posts[id]
	return p, ok
}

func TestPostsList(t *testing.T) {
	t.Parallel()

	repo := newMemPosts()
	posts := NewPosts(repo, newSecurity(t))
	for range PostPageSize*2 + 1 {
		if _, err := posts.Create(t.Context(), 1, PostInput{Title: "t", Body: "b"}); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		page      int
		wantLen   int
		wantFirst domain.PostID
		wantNext  bool
	}{
		{1, PostPageSize, 41, true},
		{2, PostPageSize, 21, true},
		{3, 1, 1, false},
		{4, 0, 0, false},
	}
	for _, tt := range tests {
		got, err := posts.List(t.Context(), tt.page)
		if err != nil {
			t.Fatalf("List(%d): %v", tt.page, err)
		}
		if len(got.Items) != tt.wantLen || got.HasNext != tt.wantNext || got.Page != tt.page || got.PageSize != PostPageSize {
			t.Errorf("List(%d) = %d items, has_next %v, page %d, size %d; want %d items, has_next %v",
				tt.page, len(got.Items), got.HasNext, got.Page, got.PageSize, tt.wantLen, tt.wantNext)
		}
		if tt.wantLen > 0 && got.Items[0].ID != tt.wantFirst {
			t.Errorf("List(%d) starts at post %d, want %d (newest first)", tt.page, got.Items[0].ID, tt.wantFirst)
		}
		if repo.limit != PostPageSize+1 || repo.offset != (tt.page-1)*PostPageSize {
			t.Errorf("List(%d) read limit %d offset %d", tt.page, repo.limit, repo.offset)
		}
	}

	if _, err := posts.List(t.Context(), 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("List(0) = %v, want ErrInvalidInput", err)
	}
}

func TestPostsCreateValidatesAndTrims(t *testing.T) {
	t.Parallel()

	repo := newMemPosts()
	posts := NewPosts(repo, newSecurity(t))

	got, err := posts.Create(t.Context(), 7, PostInput{Title: "  Título  ", Body: "\ncorpo\n"})
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthorID != 7 || got.Title != "Título" || got.Body != "corpo" {
		t.Errorf("Create = %+v", got)
	}

	for _, in := range []PostInput{
		{Title: "", Body: "b"},
		{Title: "t", Body: ""},
		{Title: strings.Repeat("a", 121), Body: "b"},
		{Title: "t", Body: strings.Repeat("b", 2001)},
		{Title: "t", Body: "b\x00"},
	} {
		if _, err := posts.Create(t.Context(), 7, in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Create(%q, %q) = %v, want ErrInvalidInput", in.Title, in.Body, err)
		}
	}
	if len(repo.posts) != 1 {
		t.Errorf("%d posts stored, want 1", len(repo.posts))
	}
}

// TestPostsOwnerOnly covers the owner check of 005 S-1 in the service: a
// post of another author is not found and stays unchanged.
func TestPostsOwnerOnly(t *testing.T) {
	t.Parallel()

	const owner, other domain.UserID = 1, 2
	repo := newMemPosts()
	posts := NewPosts(repo, newSecurity(t))
	created, err := posts.Create(t.Context(), owner, PostInput{Title: "original", Body: "texto"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = posts.Update(t.Context(), other, created.ID, PostInput{Title: "hacked", Body: "x"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update by another author = %v, want ErrNotFound", err)
	}
	if err = posts.Delete(t.Context(), other, created.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete by another author = %v, want ErrNotFound", err)
	}
	if p, ok := repo.get(created.ID); !ok || p.Title != "original" {
		t.Errorf("post after another author's attempts = %+v, %v; want unchanged", p, ok)
	}

	updated, err := posts.Update(t.Context(), owner, created.ID, PostInput{Title: " novo ", Body: "outro"})
	if err != nil || updated.Title != "novo" || updated.Body != "outro" {
		t.Errorf("Update by owner = %+v, %v", updated, err)
	}
	if err := posts.Delete(t.Context(), owner, created.ID); err != nil {
		t.Errorf("Delete by owner = %v", err)
	}
	if _, ok := repo.get(created.ID); ok {
		t.Error("post still stored after the owner deleted it")
	}
}
