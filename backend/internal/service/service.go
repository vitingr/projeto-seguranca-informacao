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
// a post of the given author, in the same statement (005 D-3); no such
// post wraps domain.ErrNotFound.
type PostRepository interface {
	List(ctx context.Context, limit, offset int) ([]domain.Post, error)
	Create(ctx context.Context, p domain.Post) (domain.Post, error)
	Update(ctx context.Context, p domain.Post) (domain.Post, error)
	Delete(ctx context.Context, id domain.PostID, author domain.UserID) error
}

type PostValidator interface {
	// ValidatePost returns the trimmed title and body, or an error that
	// wraps domain.ErrInvalidInput.
	ValidatePost(title, body string) (string, string, error)
}
