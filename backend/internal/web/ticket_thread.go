package web

import (
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// Sides of a thread entry (plan P-11): the ticket's author, or the support
// team (any admin).
const (
	sideAuthor  = "author"
	sideSupport = "support"
)

// threadEntry is an entry of the thread as a comum user sees it (spec 003
// AC-3.1, AC-3.2): the side and whether it is theirs, never an admin's
// login or id (S-13, Q-6). A message has text; a status change has from
// and to.
type threadEntry struct {
	ID        domain.TicketEntryID `json:"id"`
	Kind      domain.EntryKind     `json:"kind"`
	Side      string               `json:"side"`
	Mine      bool                 `json:"mine"`
	Text      string               `json:"text,omitempty"`
	From      domain.TicketStatus  `json:"from,omitempty"`
	To        domain.TicketStatus  `json:"to,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// threadPage is a page of the thread, oldest first; has_more says there are
// older entries, read with ?antes=<id of the first one> (plan P-4).
type threadPage struct {
	Entries []threadEntry `json:"entries"`
	HasMore bool          `json:"has_more"`
}

func newThreadEntry(e *domain.TicketEntry, ref security.TicketRef, viewer domain.UserID) threadEntry {
	side := sideSupport
	if e.ActorID == ref.AuthorID {
		side = sideAuthor
	}
	return threadEntry{
		ID: e.ID, Kind: e.Kind, Side: side, Mine: e.ActorID == viewer,
		Text: e.Body, From: e.From, To: e.To, CreatedAt: e.CreatedAt,
	}
}

// getOwnThread answers the thread of a comum user's ticket (spec 003 US-3).
// TicketAccess already decided that the ticket is theirs (S-2); the only
// parameter is antes, and anything else is a 400.
func (h *handlers) getOwnThread(w http.ResponseWriter, r *http.Request) {
	p, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	page, ok := h.thread(w, r, p, ref)
	if !ok {
		return
	}
	out := threadPage{Entries: make([]threadEntry, 0, len(page.Entries)), HasMore: page.HasMore}
	for i := range page.Entries {
		out.Entries = append(out.Entries, newThreadEntry(&page.Entries[i], ref, p.UserID))
	}
	writeJSON(w, http.StatusOK, out)
}

// ticketCaller returns the principal and the ticket TicketAccess let
// through. Without them the route is miswired: it answers 500 (fail closed)
// or 401, and ok is false.
func (h *handlers) ticketCaller(w http.ResponseWriter, r *http.Request) (security.Principal, security.TicketRef, bool) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return security.Principal{}, security.TicketRef{}, false
	}
	ref, ok := security.TicketRefFrom(r.Context())
	if !ok {
		h.sec.RespondError(w, r, errNoTicketRef)
		return security.Principal{}, security.TicketRef{}, false
	}
	return p, ref, true
}

// thread reads the cursor of the query and a page of the ticket's thread.
// On error it writes the answer and ok is false.
func (h *handlers) thread(w http.ResponseWriter, r *http.Request, p security.Principal, ref security.TicketRef) (domain.ThreadPage, bool) {
	values, err := h.sec.ReadQuery(r)
	var before domain.TicketEntryID
	if err == nil {
		before, err = h.sec.ParseThreadCursor(values)
	}
	if err != nil {
		h.queryRefused(r, p.UserID, err)
		h.badRequest(w, r, err)
		return domain.ThreadPage{}, false
	}
	page, err := h.tickets.Thread(r.Context(), ref.ID, before)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return domain.ThreadPage{}, false
	}
	return page, true
}
