package service

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"testing"

	"backend_si/internal/domain"
)

// memPosts is an in-memory PostRepository with the owner condition and the
// freeze of removed posts of the SQL statements (005 D-3, 006 D-2). List
// returns the newest (highest ID) first.
type memPosts struct {
	mu     sync.Mutex
	posts  map[domain.PostID]domain.Post
	nextID domain.PostID
	viewer domain.UserID // arguments of the last List call
	all    bool
	limit  int
	offset int
	events []domain.AuditEvent // appended by Remove
}

func newMemPosts() *memPosts { return &memPosts{posts: map[domain.PostID]domain.Post{}} }

func (m *memPosts) List(_ context.Context, viewer domain.UserID, everything bool, limit, offset int) ([]domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viewer, m.all, m.limit, m.offset = viewer, everything, limit, offset
	all := make([]domain.Post, 0, len(m.posts))
	for _, p := range m.posts {
		if everything || p.Removal == nil || p.AuthorID == viewer {
			all = append(all, p)
		}
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
	if !ok || old.AuthorID != p.AuthorID || old.Removal != nil {
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
	if !ok || old.AuthorID != author || old.Removal != nil {
		return domain.ErrNotFound
	}
	delete(m.posts, id)
	return nil
}

func (m *memPosts) Remove(_ context.Context, id domain.PostID, admin domain.UserID, reason string, ev domain.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.posts[id]
	if !ok || old.Removal != nil {
		return domain.ErrNotFound
	}
	old.Removal = &domain.PostRemoval{Reason: reason, RemovedBy: admin}
	m.posts[id] = old
	m.events = append(m.events, ev)
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
		got, err := posts.List(t.Context(), 2, domain.RoleComum, tt.page)
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

	if _, err := posts.List(t.Context(), 2, domain.RoleComum, 0); !errors.Is(err, domain.ErrInvalidInput) {
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

// TestPostsListByRole covers 006 D-1 and D-2 in the service: an admin reads
// every post, a comum user the posts that are not removed plus their own.
func TestPostsListByRole(t *testing.T) {
	t.Parallel()

	repo := newMemPosts()
	posts := NewPosts(repo, newSecurity(t))
	const admin, author, other domain.UserID = 1, 2, 3
	removed, err := posts.Create(t.Context(), author, PostInput{Title: "removido", Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := posts.Create(t.Context(), other, PostInput{Title: "visível", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := posts.Remove(t.Context(), admin, removed.ID, "spam", netip.Addr{}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		viewer  domain.UserID
		role    domain.Role
		wantAll bool
		wantLen int
	}{
		{"admin sees every post", admin, domain.RoleAdmin, true, 2},
		{"author sees their removed post", author, domain.RoleComum, false, 2},
		{"another comum user does not", other, domain.RoleComum, false, 1},
	}
	for _, tt := range tests {
		got, err := posts.List(t.Context(), tt.viewer, tt.role, 1)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if len(got.Items) != tt.wantLen || repo.all != tt.wantAll || repo.viewer != tt.viewer {
			t.Errorf("%s: %d items, all %v, viewer %d", tt.name, len(got.Items), repo.all, repo.viewer)
		}
	}
	if _, err := posts.List(t.Context(), 1, "root", 1); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("List with an invalid role = %v, want ErrInvalidInput", err)
	}
}

// TestPostsRemove covers 006 D-2 and D-3 in the service: the reason is
// validated and trimmed, the event names the admin, the post and the IP,
// and a removed post is frozen.
func TestPostsRemove(t *testing.T) {
	t.Parallel()

	repo := newMemPosts()
	posts := NewPosts(repo, newSecurity(t))
	const admin, author domain.UserID = 1, 2
	created, err := posts.Create(t.Context(), author, PostInput{Title: "t", Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	ip := netip.MustParseAddr("203.0.113.7")

	for _, reason := range []string{"", "   ", strings.Repeat("a", 501), "spam\x00"} {
		if err := posts.Remove(t.Context(), admin, created.ID, reason, ip); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Remove(reason %q) = %v, want ErrInvalidInput", reason, err)
		}
	}
	if p, _ := repo.get(created.ID); p.Removal != nil || len(repo.events) != 0 {
		t.Fatalf("an invalid reason removed the post or wrote an event: %+v, %v", p, repo.events)
	}

	if err := posts.Remove(t.Context(), admin, created.ID, "  Spam  ", ip); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	p, _ := repo.get(created.ID)
	if p.Removal == nil || p.Removal.Reason != "Spam" || p.Removal.RemovedBy != admin {
		t.Errorf("removal = %+v, want the trimmed reason and the admin", p.Removal)
	}
	want := domain.AuditEvent{Type: domain.AuditPostRemoved, ActorID: admin, TargetType: domain.AuditTargetPost, TargetID: int64(created.ID), IP: ip}
	if len(repo.events) != 1 || repo.events[0] != want {
		t.Errorf("events = %+v, want [%+v]", repo.events, want)
	}

	if err := posts.Remove(t.Context(), admin, created.ID, "de novo", ip); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second Remove = %v, want ErrNotFound", err)
	}
	if _, err := posts.Update(t.Context(), author, created.ID, PostInput{Title: "t2", Body: "b2"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("author's Update of a removed post = %v, want ErrNotFound", err)
	}
	if err := posts.Delete(t.Context(), author, created.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("author's Delete of a removed post = %v, want ErrNotFound", err)
	}
	if _, ok := repo.get(created.ID); !ok {
		t.Error("the removed post is gone, want it kept")
	}
}
