package security

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"backend_si/internal/domain"
)

// Ticket access (spec 003-ticket-tracking, S-2, BR-1). Every endpoint that
// takes a ticket id decides access with CanAccessTicket, through the
// TicketAccess middleware; no handler repeats the rule.

// maxTicketIDDigits is the number of digits of the largest int64.
const maxTicketIDDigits = 19

// ParseTicketID reads a ticket number as it appears in a path: ASCII
// digits only, no sign, no leading zero, from 1 to the largest int64. ok is
// false for anything else, which callers answer exactly like a ticket that
// does not exist (S-6).
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

// CanAccessTicket reports whether principal may read and answer the ticket
// written by author: an admin may access every ticket, a comum user only
// their own (BR-1). Anything else is refused (fail closed): no principal, a
// zero or negative user id or author, an unknown role.
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

// TicketRef is the ticket TicketAccess let through: handlers take the id
// only from here, never from the path again.
type TicketRef struct {
	ID       domain.TicketID
	AuthorID domain.UserID
}

type ticketRefKey struct{}

// TicketRefFrom returns the ticket TicketAccess put in ctx. ok is false
// when there is none (the route is not wrapped by TicketAccess).
func TicketRefFrom(ctx context.Context) (TicketRef, bool) {
	ref, ok := ctx.Value(ticketRefKey{}).(TicketRef)
	return ref, ok && ref.ID > 0 && ref.AuthorID > 0
}

// Fixed reasons of the TicketAccess denials (S-12): never user input.
const (
	ticketNoStore     = "ticket access: no store"
	ticketAnonymous   = "ticket access: anonymous"
	ticketMalformedID = "ticket access: malformed id"
	ticketNotFound    = "ticket access: not found"
	ticketNotAllowed  = "ticket access: not the author"
)

// TicketAccess guards every route with a {id} ticket path value (plan P-8,
// spec S-2). It runs inside Authorize, so a principal is always present;
// without one it fails closed with 401 (and with 500 without a store,
// Options.Tickets). A malformed id, a ticket that does
// not exist and, for a comum user, someone else's ticket all end in the
// same 404 through the same renderer, so the answer never tells them apart;
// a well-formed id always costs exactly one store call. Each denial is one
// warn log line with the user id, the ticket id (or "malformed") and a
// fixed reason, never the raw path or a body. On success the ticket goes
// into the context (TicketRefFrom).
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
		// A percent-encoded path (RawPath set) is refused as malformed: the
		// id of a ticket route is plain digits, so "%31" is not ticket 1.
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

// Values of the ticket_id log field when there is no well-formed id.
const (
	ticketMalformed = "malformed"
	ticketUnchecked = "unchecked"
)

func formatTicketID(id domain.TicketID) string {
	return strconv.FormatInt(int64(id), 10)
}

// ticketDenied logs one TicketAccess denial and answers the generic error
// of status. The log has the user id, the ticket id and a fixed reason; it
// never has the path, which may hold a malformed id typed by the client.
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
