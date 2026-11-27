package security

import (
	"fmt"
	"net/http"
	"slices"

	"backend_si/internal/domain"
)

// Status changes (spec 003 BR-2, BR-3, BR-5, BR-7). Only admins reach the
// route; these rules decide which change may be applied.

// allowedTransitions is the table of BR-3, with the targets of each status
// in a fixed order. "fechado" is final: it has no target (BR-5).
var allowedTransitions = map[domain.TicketStatus][]domain.TicketStatus{
	domain.TicketStatusOpen:       {domain.TicketStatusInProgress, domain.TicketStatusClosed},
	domain.TicketStatusInProgress: {domain.TicketStatusResolved, domain.TicketStatusClosed},
	domain.TicketStatusResolved:   {domain.TicketStatusInProgress, domain.TicketStatusClosed},
	domain.TicketStatusClosed:     {},
}

// AllowedNextStatuses returns the statuses a ticket in from may move to,
// in the order of BR-3; none for "fechado" or an unknown status. The slice
// is a copy.
func AllowedNextStatuses(from domain.TicketStatus) []domain.TicketStatus {
	return slices.Clone(allowedTransitions[from])
}

// reasonTransition is the refusal of a pair BR-3 never allows.
const reasonTransition = "transition not allowed"

// CheckStatusChange allows only the BR-3 pairs. A pair BR-3 never allows,
// whatever the ticket's state (a backward or skipping move, the same
// status, an unknown status), is an invalid request: domain.ErrInvalidInput,
// 422 in the handler. A change from "fechado" is a conflict with the
// ticket's state instead, since "fechado" is final (BR-5):
// domain.ErrStaleStatus, 409, like a de that is not the current status.
func (s *Security) CheckStatusChange(from, to domain.TicketStatus) error {
	switch {
	case slices.Contains(allowedTransitions[from], to):
		return nil
	case from == domain.TicketStatusClosed:
		return fmt.Errorf("security: status change: %w", domain.ErrStaleStatus)
	default:
		return fmt.Errorf("security: status change: %s: %w", reasonTransition, domain.ErrInvalidInput)
	}
}

// ParseStatus reads a status code. An unknown, empty or differently cased
// code is domain.ErrInvalidInput (422 in the handler).
func (s *Security) ParseStatus(raw string) (domain.TicketStatus, error) {
	status := domain.TicketStatus(raw)
	if !status.Valid() {
		return "", fmt.Errorf("security: parse status: %w", domain.ErrInvalidInput)
	}
	return status, nil
}

const (
	reasonNoFrom = "missing de"
	reasonNoTo   = "missing para"
)

type statusChangeRequest struct {
	De   *string `json:"de"`
	Para *string `json:"para"`
}

// DecodeStatusChange reads the body of a status change, {"de": "…",
// "para": "…"}: de is the status the admin saw, para the new one. The rules
// of DecodeJSON apply (exact keys, so an actor or a date is refused: S-8);
// both keys are required and must be strings. Every refusal wraps
// domain.ErrInvalidInput (400) with a fixed reason. The codes are returned
// raw; ParseStatus checks them.
func (s *Security) DecodeStatusChange(w http.ResponseWriter, r *http.Request) (from, to string, err error) {
	var req statusChangeRequest
	if err := s.DecodeJSON(w, r, &req); err != nil {
		return "", "", err
	}
	switch {
	case req.De == nil:
		return "", "", invalidBody(reasonNoFrom)
	case req.Para == nil:
		return "", "", invalidBody(reasonNoTo)
	}
	return *req.De, *req.Para, nil
}
