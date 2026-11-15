package web

import (
	"context"
	"sync"
	"time"

	"backend_si/internal/domain"
)

// memSessions is an in-memory security.SessionStore for tests that need
// working sessions. Get fills Role from roles, like the repository JOIN.
type memSessions struct {
	mu    sync.Mutex
	rows  map[string]domain.Session
	roles map[domain.UserID]domain.Role
}

func newMemSessions() *memSessions {
	return &memSessions{
		rows:  map[string]domain.Session{},
		roles: map[domain.UserID]domain.Role{1: domain.RoleAdmin, 2: domain.RoleComum},
	}
}

func (m *memSessions) Create(_ context.Context, s domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[string(s.TokenHash)] = s
	return nil
}

func (m *memSessions) Get(_ context.Context, h []byte) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.rows[string(h)]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	s.Role = m.roles[s.UserID]
	return s, nil
}

func (m *memSessions) Touch(_ context.Context, h []byte, lastSeen, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.rows[string(h)]
	if !ok {
		return domain.ErrNotFound
	}
	s.LastSeenAt, s.ExpiresAt = lastSeen, expires
	m.rows[string(h)] = s
	return nil
}

func (m *memSessions) Delete(_ context.Context, h []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, string(h))
	return nil
}

func (m *memSessions) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }

func (m *memSessions) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rows)
}

// setRole changes a user's role, like an UPDATE on the users table: the
// next Get of any of the user's sessions returns the new role.
func (m *memSessions) setRole(id domain.UserID, role domain.Role) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roles[id] = role
}
