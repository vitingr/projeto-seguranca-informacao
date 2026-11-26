package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// The two conflicts of spec 003 (plan P-14). Both are ErrConflict (409);
// each has its own message in the web layer.
var (
	// ErrTicketClosed: a message for a "fechado" ticket (BR-5).
	ErrTicketClosed = fmt.Errorf("ticket closed: %w", ErrConflict)
	// ErrStaleStatus: the ticket's status is not the one the admin saw, or
	// the change is not a transition BR-3 allows (BR-7).
	ErrStaleStatus = fmt.Errorf("stale status: %w", ErrConflict)
)
