package domain

import (
	"errors"
	"testing"
)

func TestTicketConflictErrors(t *testing.T) {
	t.Parallel()

	// Plan P-14: both are a 409 (ErrConflict) and stay distinct, so each
	// can carry its own message.
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"closed is a conflict", ErrTicketClosed, ErrConflict, true},
		{"stale is a conflict", ErrStaleStatus, ErrConflict, true},
		{"closed is not stale", ErrTicketClosed, ErrStaleStatus, false},
		{"stale is not closed", ErrStaleStatus, ErrTicketClosed, false},
		{"closed is not not found", ErrTicketClosed, ErrNotFound, false},
		{"stale is not invalid input", ErrStaleStatus, ErrInvalidInput, false},
		{"conflict is not closed", ErrConflict, ErrTicketClosed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err, tt.target, got, tt.want)
			}
		})
	}
}
