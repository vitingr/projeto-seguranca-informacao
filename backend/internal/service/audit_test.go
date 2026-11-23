package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"backend_si/internal/domain"
)

// memAudit is an in-memory AuditRepository; List returns the newest
// (last appended) first.
type memAudit struct {
	mu     sync.Mutex
	events []domain.AuditEvent
	typ    domain.AuditEventType // arguments of the last List call
	limit  int
	offset int
}

func (m *memAudit) Append(_ context.Context, ev domain.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ev.ID = domain.AuditEventID(len(m.events) + 1)
	m.events = append(m.events, ev)
	return nil
}

func (m *memAudit) List(_ context.Context, typ domain.AuditEventType, limit, offset int) ([]domain.AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.typ, m.limit, m.offset = typ, limit, offset
	var out []domain.AuditEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		if typ == "" || m.events[i].Type == typ {
			out = append(out, m.events[i])
		}
	}
	if offset >= len(out) {
		return []domain.AuditEvent{}, nil
	}
	return out[offset:min(offset+limit, len(out))], nil
}

func TestAuditRecord(t *testing.T) {
	t.Parallel()

	repo := &memAudit{}
	audit := NewAudit(repo)
	for _, typ := range domain.AuditEventTypes() {
		if err := audit.Record(t.Context(), domain.AuditEvent{Type: typ, ActorID: 1}); err != nil {
			t.Errorf("Record(%s): %v", typ, err)
		}
	}
	if err := audit.Record(t.Context(), domain.AuditEvent{Type: "password_changed"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Record of an unknown type = %v, want ErrInvalidInput", err)
	}
	if len(repo.events) != len(domain.AuditEventTypes()) {
		t.Errorf("%d events stored, want %d", len(repo.events), len(domain.AuditEventTypes()))
	}
}

// TestAuditList covers the paging and the type filter of 006 D-8.
func TestAuditList(t *testing.T) {
	t.Parallel()

	repo := &memAudit{}
	audit := NewAudit(repo)
	for i := range AuditPageSize*2 + 1 {
		typ := domain.AuditLoginSucceeded
		if i%2 == 0 {
			typ = domain.AuditLoginFailed
		}
		if err := audit.Record(t.Context(), domain.AuditEvent{Type: typ}); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		typ       domain.AuditEventType
		page      int
		wantLen   int
		wantFirst domain.AuditEventID
		wantNext  bool
	}{
		{"", 1, AuditPageSize, 41, true},
		{"", 3, 1, 1, false},
		{"", 4, 0, 0, false},
		{domain.AuditLoginFailed, 1, AuditPageSize, 41, true},
		{domain.AuditLoginFailed, 2, 1, 1, false},
		{domain.AuditLoginSucceeded, 1, AuditPageSize, 40, false},
		{domain.AuditPostRemoved, 1, 0, 0, false},
	}
	for _, tt := range tests {
		got, err := audit.List(t.Context(), tt.typ, tt.page)
		if err != nil {
			t.Fatalf("List(%q, %d): %v", tt.typ, tt.page, err)
		}
		if len(got.Items) != tt.wantLen || got.HasNext != tt.wantNext || got.Page != tt.page || got.PageSize != AuditPageSize {
			t.Errorf("List(%q, %d) = %d items, has_next %v; want %d, %v", tt.typ, tt.page, len(got.Items), got.HasNext, tt.wantLen, tt.wantNext)
		}
		if tt.wantLen > 0 && got.Items[0].ID != tt.wantFirst {
			t.Errorf("List(%q, %d) starts at %d, want %d", tt.typ, tt.page, got.Items[0].ID, tt.wantFirst)
		}
		if repo.typ != tt.typ || repo.limit != AuditPageSize+1 || repo.offset != (tt.page-1)*AuditPageSize {
			t.Errorf("List(%q, %d) read type %q limit %d offset %d", tt.typ, tt.page, repo.typ, repo.limit, repo.offset)
		}
	}

	if _, err := audit.List(t.Context(), "", 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("List(page 0) = %v, want ErrInvalidInput", err)
	}
	if _, err := audit.List(t.Context(), "bogus", 1); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("List(type bogus) = %v, want ErrInvalidInput", err)
	}
}
