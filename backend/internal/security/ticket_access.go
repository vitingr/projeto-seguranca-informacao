package security

import (
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
