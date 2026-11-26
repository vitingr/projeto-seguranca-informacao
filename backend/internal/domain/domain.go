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
	TokenHash []byte
	UserID    UserID
	Role      Role
	// Login and UserCreatedAt are the session user's login and registration
	// time, read with the role; empty for an anonymous session.
	Login         string
	UserCreatedAt time.Time
	CSRFToken     string
	CreatedAt     time.Time
	LastSeenAt    time.Time
	ExpiresAt     time.Time
}

func (s Session) Authenticated() bool {
	return s.UserID != 0
}
