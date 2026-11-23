package service

import (
	"context"
	"fmt"

	"backend_si/internal/domain"
)

// AuditPageSize is the number of events on each page of the audit log
// (006 D-8, like 005 D-5).
const AuditPageSize = 20

// Audit records and lists the security events of the audit log (spec 006).
type Audit struct {
	repo AuditRepository
}

func NewAudit(repo AuditRepository) *Audit {
	return &Audit{repo: repo}
}

// Record appends ev to the log. An event of an unknown type wraps
// domain.ErrInvalidInput and is not written.
func (a *Audit) Record(ctx context.Context, ev domain.AuditEvent) error {
	if !ev.Type.Valid() {
		return fmt.Errorf("service: record audit event %q: %w", ev.Type, domain.ErrInvalidInput)
	}
	if err := a.repo.Append(ctx, ev); err != nil {
		return fmt.Errorf("service: record audit event %s: %w", ev.Type, err)
	}
	return nil
}

// List returns one page of the log, newest first, with only the events of
// typ when typ is not empty. It reads one event more than the page size to
// tell whether there is a next page, with no COUNT.
func (a *Audit) List(ctx context.Context, typ domain.AuditEventType, page int) (domain.AuditPage, error) {
	if page < 1 {
		return domain.AuditPage{}, fmt.Errorf("service: list audit events: page %d: %w", page, domain.ErrInvalidInput)
	}
	if typ != "" && !typ.Valid() {
		return domain.AuditPage{}, fmt.Errorf("service: list audit events: type %q: %w", typ, domain.ErrInvalidInput)
	}
	items, err := a.repo.List(ctx, typ, AuditPageSize+1, (page-1)*AuditPageSize)
	if err != nil {
		return domain.AuditPage{}, fmt.Errorf("service: list audit events: %w", err)
	}
	hasNext := len(items) > AuditPageSize
	if hasNext {
		items = items[:AuditPageSize]
	}
	return domain.AuditPage{Items: items, Page: page, PageSize: AuditPageSize, HasNext: hasNext}, nil
}
