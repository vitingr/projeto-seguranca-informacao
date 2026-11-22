package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// memPostRepo is an in-memory service.PostRepository with the owner
// condition of the SQL statements (005 D-3). The author login comes from the
// users of credAuth. List returns the newest (highest ID) first.
type memPostRepo struct {
	mu     sync.Mutex
	posts  map[domain.PostID]domain.Post
	nextID domain.PostID
}

func newMemPostRepo() *memPostRepo {
	return &memPostRepo{posts: map[domain.PostID]domain.Post{}}
}

var credLogins = map[domain.UserID]string{1: flowLogin, 2: comumLogin, 3: otherComumLogin}

func (m *memPostRepo) List(_ context.Context, limit, offset int) ([]domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := slices.Collect(maps.Values(m.posts))
	slices.SortFunc(all, func(a, b domain.Post) int { return int(b.ID - a.ID) })
	if offset >= len(all) {
		return []domain.Post{}, nil
	}
	return all[offset:min(offset+limit, len(all))], nil
}

func (m *memPostRepo) Create(_ context.Context, p domain.Post) (domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	now := time.Date(2026, 11, 22, 12, 0, int(m.nextID), 0, time.UTC)
	p.ID, p.AuthorLogin, p.CreatedAt, p.UpdatedAt = m.nextID, credLogins[p.AuthorID], now, now
	m.posts[p.ID] = p
	return p, nil
}

func (m *memPostRepo) Update(_ context.Context, p domain.Post) (domain.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.posts[p.ID]
	if !ok || old.AuthorID != p.AuthorID {
		return domain.Post{}, domain.ErrNotFound
	}
	old.Title, old.Body, old.UpdatedAt = p.Title, p.Body, old.UpdatedAt.Add(time.Minute)
	m.posts[p.ID] = old
	return old, nil
}

func (m *memPostRepo) Delete(_ context.Context, id domain.PostID, author domain.UserID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.posts[id]
	if !ok || old.AuthorID != author {
		return domain.ErrNotFound
	}
	delete(m.posts, id)
	return nil
}

func (m *memPostRepo) snapshot() map[domain.PostID]domain.Post {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.posts)
}

// seed stores a post of author directly, bypassing the API.
func (m *memPostRepo) seed(t *testing.T, author domain.UserID, title string) domain.PostID {
	t.Helper()
	p, err := m.Create(t.Context(), domain.Post{AuthorID: author, Title: title, Body: "corpo"})
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func postBody(title, body string) string {
	b, err := json.Marshal(postRequest{Title: title, Body: body})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func postPath(id domain.PostID) string { return fmt.Sprintf("/api/posts/%d", id) }

func csrfHeader(token string) map[string]string {
	return map[string]string{security.CSRFHeaderName: token}
}

func decodeInto[T any](t *testing.T, body *bytes.Buffer) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", body.String(), err)
	}
	return v
}

// assertUnchanged fails when the stored posts differ from before.
func assertUnchanged(t *testing.T, f flowApp, before map[domain.PostID]domain.Post) {
	t.Helper()
	if after := f.posts.snapshot(); !maps.Equal(before, after) {
		t.Errorf("posts changed:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestPostsFlow(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	maria, mariaToken := f.loginAs(t, comumLogin)
	joao, joaoToken := f.loginAs(t, otherComumLogin)

	rr := f.send(t, http.MethodPost, "/api/posts", postBody("  Olá  ", "linha 1\nlinha 2"), maria, csrfHeader(mariaToken))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	created := decodeInto[postResponse](t, rr.Body)
	wantAuthor := postAuthorResponse{UserID: 2, Login: comumLogin}
	if created.Title != "Olá" || created.Body != "linha 1\nlinha 2" || created.Author != wantAuthor || !created.Mine {
		t.Errorf("created = %+v", created)
	}

	rr = f.send(t, http.MethodPost, "/api/posts", postBody("do joão", "b"), joao, csrfHeader(joaoToken))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create as joao: %d %s", rr.Code, rr.Body.String())
	}

	rr = f.send(t, http.MethodGet, "/api/posts", "", maria, nil)
	page := decodeInto[postPageResponse](t, rr.Body)
	if rr.Code != http.StatusOK || page.Page != 1 || page.PageSize != 20 || page.HasNext || len(page.Items) != 2 {
		t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
	}
	if page.Items[0].Title != "do joão" || page.Items[0].Mine || !page.Items[1].Mine {
		t.Errorf("items (newest first, mine only on maria's post) = %+v", page.Items)
	}

	rr = f.send(t, http.MethodPut, postPath(created.ID), postBody("Editado", "novo"), maria, csrfHeader(mariaToken))
	updated := decodeInto[postResponse](t, rr.Body)
	if rr.Code != http.StatusOK || updated.Title != "Editado" || updated.Body != "novo" || !updated.Mine {
		t.Errorf("update: %d %s", rr.Code, rr.Body.String())
	}

	rr = f.send(t, http.MethodDelete, postPath(created.ID), "", maria, csrfHeader(mariaToken))
	if rr.Code != http.StatusNoContent || rr.Body.Len() != 0 {
		t.Errorf("delete: %d %q", rr.Code, rr.Body.String())
	}
	if _, ok := f.posts.snapshot()[created.ID]; ok {
		t.Error("post still stored after delete")
	}
}

func TestPostsListPages(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	for i := range 21 {
		f.posts.seed(t, 3, fmt.Sprintf("post %d", i+1))
	}
	cookie, _ := f.loginAs(t, comumLogin)

	tests := []struct {
		query     string
		wantLen   int
		wantNext  bool
		wantFirst string
	}{
		{"", 20, true, "post 21"},
		{"?page=1", 20, true, "post 21"},
		{"?page=2", 1, false, "post 1"},
		{"?page=3", 0, false, ""},
		{"?page=10000", 0, false, ""},
	}
	for _, tt := range tests {
		rr := f.send(t, http.MethodGet, "/api/posts"+tt.query, "", cookie, nil)
		page := decodeInto[postPageResponse](t, rr.Body)
		if rr.Code != http.StatusOK || len(page.Items) != tt.wantLen || page.HasNext != tt.wantNext {
			t.Errorf("GET /api/posts%s: %d, %d items, has_next %v", tt.query, rr.Code, len(page.Items), page.HasNext)
			continue
		}
		if tt.wantLen > 0 && page.Items[0].Title != tt.wantFirst {
			t.Errorf("GET /api/posts%s starts at %q, want %q", tt.query, page.Items[0].Title, tt.wantFirst)
		}
	}

	// An empty page is [] in JSON, never null.
	rr := f.send(t, http.MethodGet, "/api/posts?page=3", "", cookie, nil)
	if !strings.Contains(rr.Body.String(), `"items":[]`) {
		t.Errorf("empty page body = %s", rr.Body.String())
	}

	for _, q := range []string{"?page=0", "?page=-1", "?page=10001", "?page=abc", "?page=1.5", "?page=01", "?page="} {
		rr := f.send(t, http.MethodGet, "/api/posts"+q, "", cookie, nil)
		if q == "?page=" {
			// An empty value is the default page, like no parameter.
			if rr.Code != http.StatusOK {
				t.Errorf("GET /api/posts%s: %d, want 200", q, rr.Code)
			}
			continue
		}
		if rr.Code != http.StatusBadRequest || decodeInto[errorResponse](t, rr.Body).Error != "bad_request" {
			t.Errorf("GET /api/posts%s: %d %s, want 400", q, rr.Code, rr.Body.String())
		}
	}
}

// TestPostsOwnerCheck covers 005 S-1: another user's post answers 404 with
// the same body as a missing or malformed ID, and stays unchanged.
func TestPostsOwnerCheck(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	mariasPost := f.posts.seed(t, 2, "da maria")
	joao, token := f.loginAs(t, otherComumLogin)
	before := f.posts.snapshot()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"update another user's post", http.MethodPut, postPath(mariasPost), postBody("hacked", "x")},
		{"delete another user's post", http.MethodDelete, postPath(mariasPost), ""},
		{"update a missing post", http.MethodPut, postPath(999), postBody("t", "b")},
		{"delete a missing post", http.MethodDelete, postPath(999), ""},
		{"update a malformed id", http.MethodPut, "/api/posts/abc", postBody("t", "b")},
		{"delete a negative id", http.MethodDelete, "/api/posts/-1", ""},
		{"delete an id that overflows", http.MethodDelete, "/api/posts/9223372036854775808", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := f.send(t, tt.method, tt.path, tt.body, joao, csrfHeader(token))
			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (%s)", rr.Code, rr.Body.String())
			}
			got := decodeInto[errorResponse](t, rr.Body)
			if got.Error != "not_found" || got.Message != msgPostNaoEncontrado {
				t.Errorf("body = %+v, want the same not_found body in every case", got)
			}
		})
	}
	assertUnchanged(t, f, before)
}

// TestPostsValidation covers 005 S-2 over HTTP: malformed bodies are 400,
// well-formed ones with invalid values are 422, and nothing is written.
func TestPostsValidation(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	own := f.posts.seed(t, 2, "minha")
	cookie, token := f.loginAs(t, comumLogin)
	before := f.posts.snapshot()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"title too long", postBody(strings.Repeat("a", 121), "b"), http.StatusUnprocessableEntity},
		{"body too long", postBody("t", strings.Repeat("b", 2001)), http.StatusUnprocessableEntity},
		{"blank title", postBody("   ", "b"), http.StatusUnprocessableEntity},
		{"empty body", postBody("t", ""), http.StatusUnprocessableEntity},
		{"missing body field", `{"title":"t"}`, http.StatusUnprocessableEntity},
		{"NUL character", postBody("t\x00", "b"), http.StatusUnprocessableEntity},
		{"newline in title", postBody("a\nb", "b"), http.StatusUnprocessableEntity},
		{"unknown field", `{"title":"t","body":"b","author_id":3}`, http.StatusBadRequest},
		{"wrong type", `{"title":1,"body":"b"}`, http.StatusBadRequest},
		{"malformed json", `{"title":`, http.StatusBadRequest},
		{"two objects", `{"title":"t","body":"b"}{}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		for _, target := range []struct{ method, path string }{
			{http.MethodPost, "/api/posts"},
			{http.MethodPut, postPath(own)},
		} {
			t.Run(target.method+" "+tt.name, func(t *testing.T) {
				rr := f.send(t, target.method, target.path, tt.body, cookie, csrfHeader(token))
				if rr.Code != tt.wantStatus {
					t.Errorf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
				}
			})
		}
	}
	assertUnchanged(t, f, before)
}

// TestPostsMutationsNeedCSRF covers 005 S-4: every mutation without the
// token, or with a wrong one, is denied by the CSRF middleware with 403 and
// writes nothing.
func TestPostsMutationsNeedCSRF(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	f := newFlowAppWithLog(t, &logBuf)
	own := f.posts.seed(t, 2, "minha")
	cookie, _ := f.loginAs(t, comumLogin)
	before := f.posts.snapshot()

	tests := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/posts", postBody("t", "b")},
		{http.MethodPut, postPath(own), postBody("t", "b")},
		{http.MethodDelete, postPath(own), ""},
	}
	tokens := []struct {
		name    string
		headers map[string]string
	}{
		{"no token", nil},
		{"wrong token", csrfHeader("not-the-token")},
	}
	for _, tt := range tests {
		for _, tok := range tokens {
			t.Run(tt.method+" "+tok.name, func(t *testing.T) {
				logBuf.Reset()
				rr := f.send(t, tt.method, tt.path, tt.body, cookie, tok.headers)
				if rr.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403 (%s)", rr.Code, rr.Body.String())
				}
				if !strings.Contains(logBuf.String(), `"reason":"csrf: token mismatch"`) {
					t.Errorf("log has no csrf: token mismatch reason:\n%s", logBuf.String())
				}
			})
		}
	}
	assertUnchanged(t, f, before)
}

// TestPostsKeepHTMLAsText covers the backend half of 005 S-3: the text is
// stored and returned exactly as sent, in a JSON response the browser does
// not sniff, with < and > escaped by encoding/json.
func TestPostsKeepHTMLAsText(t *testing.T) {
	t.Parallel()

	const payload = `<img src=x onerror="alert(1)"><script>alert(2)</script> **negrito**`
	f := newFlowApp(t)
	cookie, token := f.loginAs(t, comumLogin)
	rr := f.send(t, http.MethodPost, "/api/posts", postBody("<b>título</b>", payload), cookie, csrfHeader(token))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if strings.ContainsAny(rr.Body.String(), "<>") {
		t.Errorf("raw < or > in the JSON body: %s", rr.Body.String())
	}
	created := decodeInto[postResponse](t, rr.Body)
	if created.Body != payload || created.Title != "<b>título</b>" {
		t.Errorf("created = %+v, want the text exactly as sent", created)
	}
}
