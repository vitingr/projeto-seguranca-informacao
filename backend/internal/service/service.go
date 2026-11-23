package service

import (
	"context"

	"backend_si/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.UserID, error)
	GetByLogin(ctx context.Context, login string) (domain.User, error)
}

type PasswordHasher interface {
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, encodedHash, password string) error
}

type InputValidator interface {
	// NormalizeLogin returns the canonical (lowercase) form of a login.
	NormalizeLogin(raw string) string
	ValidateLogin(login string) error
	ValidatePassword(password string) error
	ParseRole(raw string) (domain.Role, error)
}

// PostRepository stores the posts of the wall. Update and Delete only touch
// a post of the given author that was not removed, in the same statement
// (005 D-3, 006 D-2); no such post wraps domain.ErrNotFound.
type PostRepository interface {
	// List returns every post when all is set, otherwise the posts that are
	// not removed plus viewer's own removed posts (006 D-1, D-2).
	List(ctx context.Context, viewer domain.UserID, all bool, limit, offset int) ([]domain.Post, error)
	Create(ctx context.Context, p domain.Post) (domain.Post, error)
	Update(ctx context.Context, p domain.Post) (domain.Post, error)
	Delete(ctx context.Context, id domain.PostID, author domain.UserID) error
	// Remove marks a post that is not removed yet as removed and appends ev,
	// in one transaction (006 D-3). No such post wraps domain.ErrNotFound.
	Remove(ctx context.Context, id domain.PostID, admin domain.UserID, reason string, ev domain.AuditEvent) error
}

type PostValidator interface {
	// ValidatePost returns the trimmed title and body, or an error that
	// wraps domain.ErrInvalidInput.
	ValidatePost(title, body string) (string, string, error)
	// ValidateRemovalReason returns the trimmed reason, or an error that
	// wraps domain.ErrInvalidInput.
	ValidateRemovalReason(reason string) (string, error)
}

// AuditRepository is the append-only audit log (006 D-7): it has no method
// to change or delete an event.
type AuditRepository interface {
	Append(ctx context.Context, ev domain.AuditEvent) error
	// List returns events newest first, only of typ when typ is not empty.
	List(ctx context.Context, typ domain.AuditEventType, limit, offset int) ([]domain.AuditEvent, error)
}
