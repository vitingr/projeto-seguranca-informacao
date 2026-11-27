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

var (
	ErrTicketClosed = fmt.Errorf("ticket closed: %w", ErrConflict)
	ErrStaleStatus  = fmt.Errorf("stale status: %w", ErrConflict)
)
