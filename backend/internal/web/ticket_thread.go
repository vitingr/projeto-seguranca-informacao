package web

import (
	"errors"
	"log/slog"
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

// replyOwnTicket stores a message of the comum author on their ticket (spec
// 003 US-3, AC-3.5 to AC-3.8). Inside Authorize the route runs the message
// rate limit, then TicketAccess (plan P-16), so a request on someone
// else's ticket spends the caller's quota and gets the generic 404.
func (h *handlers) replyOwnTicket(w http.ResponseWriter, r *http.Request) {
	p, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	if entry, ok := h.reply(w, r, p, ref); ok {
		writeJSON(w, http.StatusCreated, newThreadEntry(&entry, ref, p.UserID))
	}
}

// reply decodes and stores a message of p on the ticket ref. A body that is
// not well-formed is 400, a text that breaks the message rule 422, a
// "fechado" ticket 409; nothing is stored then. The log holds ids and a
// fixed reason, never the text (S-12). On error it writes the answer and
// ok is false.
func (h *handlers) reply(w http.ResponseWriter, r *http.Request, p security.Principal, ref security.TicketRef) (domain.TicketEntry, bool) {
	raw, err := h.sec.DecodeMessage(w, r)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.replyRefused(r, p, ref, err)
		}
		h.badRequest(w, r, err)
		return domain.TicketEntry{}, false
	}

	entry, err := h.tickets.Reply(r.Context(), p.UserID, ref.ID, raw)
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.replyRefused(r, p, ref, err)
		writeErrorMessage(w, r, http.StatusUnprocessableEntity, msgDadosInvalidos)
	case errors.Is(err, domain.ErrTicketClosed):
		h.replyRefused(r, p, ref, err)
		writeErrorMessage(w, r, http.StatusConflict, msgTicketClosed)
	case err != nil:
		h.sec.RespondError(w, r, err)
	default:
		h.logger.InfoContext(r.Context(), "ticket message created",
			slog.Int64("ticket_id", int64(ref.ID)),
			slog.Int64("actor_id", int64(p.UserID)),
			slog.Int64("entry_id", int64(entry.ID)),
		)
		return entry, true
	}
	return domain.TicketEntry{}, false
}

// replyRefused logs a refused message. The reason is the error text, built
// from fixed strings by the security, service and repository layers.
func (h *handlers) replyRefused(r *http.Request, p security.Principal, ref security.TicketRef, err error) {
	h.logger.InfoContext(r.Context(), "ticket message refused",
		slog.Int64("ticket_id", int64(ref.ID)),
		slog.Int64("actor_id", int64(p.UserID)),
		slog.String("reason", err.Error()),
	)
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
