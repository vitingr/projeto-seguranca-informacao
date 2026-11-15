package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
	"backend_si/internal/service"
)

// cadastroPassword is long enough for the password policy and easy to spot
// in a log.
const cadastroPassword = "Cadastro-Sentinel-Pa55word" //nolint:gosec // test fixture, not a real credential

// memUserRepo is an in-memory service.UserRepository with the UNIQUE login
// constraint of the users table. IDs start at 100, far from the session
// users (1 and 2).
type memUserRepo struct {
	mu     sync.Mutex
	byName map[string]domain.User
	nextID domain.UserID
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{byName: map[string]domain.User{}, nextID: 99}
}

func (m *memUserRepo) Create(_ context.Context, u domain.User) (domain.UserID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byName[u.Login]; ok {
		return 0, domain.ErrConflict
	}
	m.nextID++
	u.ID = m.nextID
	m.byName[u.Login] = u
	return u.ID, nil
}

func (m *memUserRepo) GetByLogin(_ context.Context, login string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byName[login]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (m *memUserRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byName)
}

// testArgon2 keeps the real hasher fast in tests.
var testArgon2 = security.Argon2Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

// newCadastroApp is the full chain with the real service.Users over repo.
func newCadastroApp(t *testing.T, logOut *bytes.Buffer, repo *memUserRepo) flowApp {
	t.Helper()
	return newFlowAppWith(t, flowConfig{
		logOut: logOut,
		argon2: testArgon2,
		users: func(sec *security.Security) UserService {
			return service.NewUsers(repo, sec, sec)
		},
	})
}

func cadastroBody(login, senha, papel string) string {
	b, err := json.Marshal(cadastroRequest{Login: login, Senha: senha, Papel: papel})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// auditLine returns the "user registered" log line, or "" if there is none.
func auditLine(logged string) string {
	for line := range strings.Lines(logged) {
		if strings.Contains(line, `"msg":"user registered"`) {
			return line
		}
	}
	return ""
}

func TestCadastroCreatesUser(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	repo := newMemUserRepo()
	f := newCadastroApp(t, &logBuf, repo)
	cookie, token := f.loginAs(t, flowLogin)

	rr := f.send(t, http.MethodPost, "/api/cadastro", cadastroBody("Maria.Silva", cadastroPassword, "comum"), cookie,
		map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
	var got userResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	// D-10: the response carries the normalized login.
	if got.UserID < 100 || got.Login != "maria.silva" || got.Role != domain.RoleComum {
		t.Errorf("body = %+v, want a new user maria.silva with role comum", got)
	}
	if strings.Contains(rr.Body.String(), cadastroPassword) || strings.Contains(rr.Body.String(), "argon2") {
		t.Errorf("response leaks the password or its hash: %s", rr.Body.String())
	}

	stored, err := repo.GetByLogin(t.Context(), "maria.silva")
	if err != nil {
		t.Fatalf("stored user: %v", err)
	}
	if !strings.HasPrefix(stored.PasswordHash, "$argon2id$v=19$") || stored.Role != domain.RoleComum {
		t.Errorf("stored user = %+v, want an Argon2id hash and role comum", stored)
	}

	// D-11: the audit line has the IDs and the role, never the credentials.
	audit := auditLine(logBuf.String())
	if audit == "" {
		t.Fatalf("no audit line in the log:\n%s", logBuf.String())
	}
	var entry struct {
		Level       string `json:"level"`
		ActorUserID int64  `json:"actor_user_id"`
		UserID      int64  `json:"user_id"`
		Role        string `json:"role"`
	}
	if err := json.Unmarshal([]byte(audit), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Level != "INFO" || entry.ActorUserID != 1 || entry.UserID != int64(got.UserID) || entry.Role != "comum" {
		t.Errorf("audit line = %s", audit)
	}
	if strings.Contains(logBuf.String(), cadastroPassword) || strings.Contains(strings.ToLower(audit), "maria") {
		t.Errorf("the log has the login or the password:\n%s", logBuf.String())
	}
}

func TestCadastroRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		existing    string // registered before the request, when set
		body        string
		wantStatus  int
		wantError   string
		wantMessage string
	}{
		{
			name: "login taken in another case", existing: "maria.silva",
			body:       cadastroBody("MARIA.SILVA", cadastroPassword, "comum"),
			wantStatus: http.StatusConflict, wantError: "conflict", wantMessage: msgLoginEmUso,
		},
		{
			name:       "invalid login",
			body:       cadastroBody("maria silva", cadastroPassword, "comum"),
			wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMessage: msgDadosInvalidos,
		},
		{
			name:       "short password",
			body:       cadastroBody("maria.silva", "short pw", "comum"),
			wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMessage: msgDadosInvalidos,
		},
		{
			name:       "role outside admin and comum",
			body:       cadastroBody("maria.silva", cadastroPassword, "root"),
			wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMessage: msgDadosInvalidos,
		},
		{
			name:       "missing role",
			body:       `{"login":"maria.silva","senha":"` + cadastroPassword + `"}`,
			wantStatus: http.StatusUnprocessableEntity, wantError: "unprocessable_entity", wantMessage: msgDadosInvalidos,
		},
		{
			name:       "role of the wrong JSON type",
			body:       `{"login":"maria.silva","senha":"` + cadastroPassword + `","papel":1}`,
			wantStatus: http.StatusBadRequest, wantError: "bad_request",
		},
		{
			name:       "unknown field",
			body:       `{"login":"maria.silva","senha":"` + cadastroPassword + `","papel":"comum","extra":1}`,
			wantStatus: http.StatusBadRequest, wantError: "bad_request",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			repo := newMemUserRepo()
			f := newCadastroApp(t, &logBuf, repo)
			cookie, token := f.loginAs(t, flowLogin)
			csrf := map[string]string{security.CSRFHeaderName: token}
			want := 0
			if tt.existing != "" {
				rr := f.send(t, http.MethodPost, "/api/cadastro", cadastroBody(tt.existing, cadastroPassword, "comum"), cookie, csrf)
				if rr.Code != http.StatusCreated {
					t.Fatalf("setup: status = %d (%s)", rr.Code, rr.Body.String())
				}
				want = 1
			}
			logBuf.Reset()

			rr := f.send(t, http.MethodPost, "/api/cadastro", tt.body, cookie, csrf)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			var body errorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tt.wantError || (tt.wantMessage != "" && body.Message != tt.wantMessage) {
				t.Errorf("body = %+v, want error %q, message %q", body, tt.wantError, tt.wantMessage)
			}
			if auditLine(logBuf.String()) != "" {
				t.Error("a rejected registration was audited")
			}
			if strings.Contains(logBuf.String(), cadastroPassword) {
				t.Errorf("the log has the password:\n%s", logBuf.String())
			}
			if n := repo.count(); n != want {
				t.Errorf("repository holds %d users, want %d", n, want)
			}
		})
	}
}
