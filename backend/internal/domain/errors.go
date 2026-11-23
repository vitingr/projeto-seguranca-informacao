package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// InvalidCredentialsError is a failed login (it wraps ErrInvalidCredentials).
// UserID is the user whose password did not match, or 0 when the login does
// not exist. It lets the audit log name the targeted account (006 D-4)
// without ever keeping the typed login.
type InvalidCredentialsError struct {
	UserID UserID
}

func (e *InvalidCredentialsError) Error() string { return ErrInvalidCredentials.Error() }

func (e *InvalidCredentialsError) Unwrap() error { return ErrInvalidCredentials }
