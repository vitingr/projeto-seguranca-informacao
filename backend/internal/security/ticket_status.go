package security

import (
	"fmt"
	"net/http"
	"slices"

	"backend_si/internal/domain"
)

var allowedTransitions = map[domain.TicketStatus][]domain.TicketStatus{
	domain.TicketStatusOpen:       {domain.TicketStatusInProgress, domain.TicketStatusClosed},
	domain.TicketStatusInProgress: {domain.TicketStatusResolved, domain.TicketStatusClosed},
	domain.TicketStatusResolved:   {domain.TicketStatusInProgress, domain.TicketStatusClosed},
	domain.TicketStatusClosed:     {},
}

func AllowedNextStatuses(from domain.TicketStatus) []domain.TicketStatus {
	return slices.Clone(allowedTransitions[from])
}

const reasonTransition = "transition not allowed"

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
