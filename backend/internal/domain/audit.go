package domain

import (
	"net/netip"
	"time"
)

// AuditEventType is one of the security events of the audit log (006 D-4).
type AuditEventType string

const (
	AuditLoginSucceeded AuditEventType = "login_succeeded"
	AuditLoginFailed    AuditEventType = "login_failed"
	AuditLogout         AuditEventType = "logout"
	AuditUserCreated    AuditEventType = "user_created"
	AuditPostRemoved    AuditEventType = "post_removed"
)

// AuditEventTypes returns the five events, in the order of the contract.
func AuditEventTypes() []AuditEventType {
	return []AuditEventType{AuditLoginSucceeded, AuditLoginFailed, AuditLogout, AuditUserCreated, AuditPostRemoved}
}

func (t AuditEventType) Valid() bool {
	switch t {
	case AuditLoginSucceeded, AuditLoginFailed, AuditLogout, AuditUserCreated, AuditPostRemoved:
		return true
	default:
		return false
	}
}

// AuditTargetType names what an event is about. The zero value means the
// event has no target.
type AuditTargetType string

const (
	AuditTargetUser AuditTargetType = "user"
	AuditTargetPost AuditTargetType = "post"
)

type AuditEventID int64

// AuditEvent is one row of the append-only audit log. It has no free-text
// field on purpose: nothing typed by a user is ever stored (006 D-4).
//
// ActorID 0 means no actor (a failed login). TargetType "" means no target.
// An invalid IP means the peer address did not parse (D-5). The logins are
// read from users when the log is listed and are empty when the user no
// longer exists; they are never stored in the log.
type AuditEvent struct {
	ID          AuditEventID
	Type        AuditEventType
	ActorID     UserID
	ActorLogin  string
	TargetType  AuditTargetType
	TargetID    int64
	TargetLogin string
	IP          netip.Addr
	CreatedAt   time.Time
}

// AuditPage is one page of the audit log, newest first (006 D-8).
type AuditPage struct {
	Items    []AuditEvent
	Page     int
	PageSize int
	HasNext  bool
}
