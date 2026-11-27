package security

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"backend_si/internal/domain"
)

// digits of the largest int64
const maxTicketIDDigits = 19

func ParseTicketID(raw string) (id domain.TicketID, ok bool) {
	if raw == "" || len(raw) > maxTicketIDDigits || raw[0] == '0' {
		return 0, false
	}
	for i := range len(raw) {
		if raw[i] < '0' || raw[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		return 0, false
	}
	return domain.TicketID(n), true
}

func CanAccessTicket(principal Principal, author domain.UserID) bool {
	if principal.UserID <= 0 || author <= 0 {
		return false
	}
	switch principal.Role {
	case domain.RoleAdmin:
		return true
	case domain.RoleComum:
		return principal.UserID == author
	default:
		return false
	}
}

type TicketRef struct {
	ID       domain.TicketID
	AuthorID domain.UserID
}

type ticketRefKey struct{}

func TicketRefFrom(ctx context.Context) (TicketRef, bool) {
	ref, ok := ctx.Value(ticketRefKey{}).(TicketRef)
	return ref, ok && ref.ID > 0 && ref.AuthorID > 0
}

const (
	ticketNoStore     = "ticket access: no store"
	ticketAnonymous   = "ticket access: anonymous"
	ticketMalformedID = "ticket access: malformed id"
	ticketNotFound    = "ticket access: not found"
	ticketNotAllowed  = "ticket access: not the author"
)

// Malformed, missing and foreign tickets all answer the same 404.
func (s *Security) TicketAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFrom(r.Context())
		if !ok {
			principal = Principal{}
		}
		if s.tickets == nil {
			s.ticketDenied(w, r, http.StatusInternalServerError, principal, ticketUnchecked, ticketNoStore)
			return
		}
		if !ok {
			s.ticketDenied(w, r, http.StatusUnauthorized, principal, ticketUnchecked, ticketAnonymous)
			return
		}
		// RawPath set means a percent-encoded path: refused as malformed
		id, ok := ParseTicketID(r.PathValue("id"))
		if !ok || r.URL.RawPath != "" {
			s.ticketDenied(w, r, http.StatusNotFound, principal, ticketMalformed, ticketMalformedID)
			return
		}
		author, err := s.tickets.TicketAuthor(r.Context(), id)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.ticketDenied(w, r, http.StatusNotFound, principal, formatTicketID(id), ticketNotFound)
			return
		case err != nil:
			s.internalError(w, r, err)
			return
		case !CanAccessTicket(principal, author):
			s.ticketDenied(w, r, http.StatusNotFound, principal, formatTicketID(id), ticketNotAllowed)
			return
		}
		ctx := context.WithValue(r.Context(), ticketRefKey{}, TicketRef{ID: id, AuthorID: author})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const (
	ticketMalformed = "malformed"
	ticketUnchecked = "unchecked"
)

func formatTicketID(id domain.TicketID) string {
	return strconv.FormatInt(int64(id), 10)
}

// The log never has the path, which may hold client-typed input.
func (s *Security) ticketDenied(w http.ResponseWriter, r *http.Request, status int, p Principal, ticketID, reason string) {
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	s.logger.Log(r.Context(), level, "security: acesso negado",
		slog.String("reason", reason),
		slog.Int64("user_id", int64(p.UserID)),
		slog.String("ticket_id", ticketID),
		slog.String("method", r.Method),
		slog.Int("status", status),
	)
	s.renderError(w, r, status)
}
