package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

type memUsers struct {
	mu     sync.Mutex
	byName map[string]domain.User
	nextID domain.UserID
	err    error
}

func newMemUsers() *memUsers { return &memUsers{byName: map[string]domain.User{}} }

func (m *memUsers) Create(_ context.Context, u domain.User) (domain.UserID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return 0, m.err
	}
	if _, ok := m.byName[u.Login]; ok {
		return 0, domain.ErrConflict
	}
	m.nextID++
	u.ID = m.nextID
	m.byName[u.Login] = u
	return u.ID, nil
}

func (m *memUsers) GetByLogin(_ context.Context, login string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.User{}, m.err
	}
	u, ok := m.byName[login]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (m *memUsers) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byName)
}

type spyHasher struct {
	PasswordHasher
	mu           sync.Mutex
	hashCalls    int
	verifyHashes []string
	hashErr      error
	verifyErr    error
}

func (s *spyHasher) HashPassword(ctx context.Context, password string) (string, error) {
	s.mu.Lock()
	s.hashCalls++
	err := s.hashErr
	s.mu.Unlock()
	if err != nil {
		return "", err
	}
	return s.PasswordHasher.HashPassword(ctx, password)
}

func (s *spyHasher) VerifyPassword(ctx context.Context, encodedHash, password string) error {
	s.mu.Lock()
	s.verifyHashes = append(s.verifyHashes, encodedHash)
	err := s.verifyErr
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return s.PasswordHasher.VerifyPassword(ctx, encodedHash, password)
}

func (s *spyHasher) verifies() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.verifyHashes...)
}

func newSecurity(t *testing.T) *security.Security {
	t.Helper()
	sec, err := security.New(security.Options{
		Logger:                 slog.New(slog.NewJSONHandler(io.Discard, nil)),
		Sessions:               nopSessions{},
		RenderError:            func(w http.ResponseWriter, _ *http.Request, status int) { w.WriteHeader(status) },
		SessionIdleTimeout:     time.Minute,
		SessionAbsoluteTimeout: time.Hour,
		RequestTimeout:         time.Minute,
	})
	if err != nil {
		t.Fatalf("security.New: %v", err)
	}
	return sec
}

type nopSessions struct{}

func (nopSessions) Create(context.Context, domain.Session) error { return nil }

func (nopSessions) Get(context.Context, []byte) (domain.Session, error) {
	return domain.Session{}, domain.ErrNotFound
}

func (nopSessions) Touch(context.Context, []byte, time.Time, time.Time) error { return nil }

func (nopSessions) Delete(context.Context, []byte) error { return nil }

func (nopSessions) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }

var errDatabase = errors.New("database is down")
