package domain

import "time"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleComum Role = "comum"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleComum
}

type UserID int64

type User struct {
	ID           UserID
	Login        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

type Session struct {
	TokenHash  []byte
	UserID     UserID
	Role       Role
	CSRFToken  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

func (s Session) Authenticated() bool {
	return s.UserID != 0
}
