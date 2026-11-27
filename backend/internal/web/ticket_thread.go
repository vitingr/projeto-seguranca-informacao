package web

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

const (
	sideAuthor  = "author"
	sideSupport = "support"
)

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

type adminThreadEntry struct {
	ID         domain.TicketEntryID `json:"id"`
	Kind       domain.EntryKind     `json:"kind"`
	Side       string               `json:"side"`
	Mine       bool                 `json:"mine"`
	ActorLogin string               `json:"actor_login"`
	Text       string               `json:"text,omitempty"`
	From       domain.TicketStatus  `json:"from,omitempty"`
	To         domain.TicketStatus  `json:"to,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
}

type adminThreadPage struct {
	Entries []adminThreadEntry `json:"entries"`
	HasMore bool               `json:"has_more"`
}

func newAdminThreadEntry(e *domain.TicketEntry, ref security.TicketRef, viewer domain.UserID) adminThreadEntry {
	c := newThreadEntry(e, ref, viewer)
	return adminThreadEntry{
		ID: c.ID, Kind: c.Kind, Side: c.Side, Mine: c.Mine, ActorLogin: e.ActorLogin,
		Text: c.Text, From: c.From, To: c.To, CreatedAt: c.CreatedAt,
	}
}

func (h *handlers) getAdminThread(w http.ResponseWriter, r *http.Request) {
	p, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	page, ok := h.thread(w, r, p, ref)
	if !ok {
		return
	}
	out := adminThreadPage{Entries: make([]adminThreadEntry, 0, len(page.Entries)), HasMore: page.HasMore}
	for i := range page.Entries {
		out.Entries = append(out.Entries, newAdminThreadEntry(&page.Entries[i], ref, p.UserID))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) replyOwnTicket(w http.ResponseWriter, r *http.Request) {
	p, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	if entry, ok := h.reply(w, r, p, ref); ok {
		writeJSON(w, http.StatusCreated, newThreadEntry(&entry, ref, p.UserID))
	}
}

func (h *handlers) replyAdminTicket(w http.ResponseWriter, r *http.Request) {
	p, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	if entry, ok := h.reply(w, r, p, ref); ok {
		writeJSON(w, http.StatusCreated, newAdminThreadEntry(&entry, ref, p.UserID))
	}
}

type statusChangeResponse struct {
	ID        domain.TicketID     `json:"id"`
	Status    domain.TicketStatus `json:"status"`
	UpdatedAt time.Time           `json:"updated_at"`
}

func (h *handlers) changeTicketStatus(w http.ResponseWriter, r *http.Request) {
	p, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	rawFrom, rawTo, err := h.sec.DecodeStatusChange(w, r)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.statusRefused(r, p, ref, err)
		}
		h.badRequest(w, r, err)
		return
	}
	from, err := h.sec.ParseStatus(rawFrom)
	var to domain.TicketStatus
	if err == nil {
		to, err = h.sec.ParseStatus(rawTo)
	}
	if err == nil {
		var entry domain.TicketEntry
		entry, err = h.tickets.ChangeStatus(r.Context(), p.UserID, ref.ID, from, to)
		if err == nil {
			h.logger.InfoContext(r.Context(), "ticket status changed",
				slog.Int64("ticket_id", int64(ref.ID)),
				slog.Int64("actor_id", int64(p.UserID)),
				slog.String("from", string(entry.From)),
				slog.String("to", string(entry.To)),
			)
			writeJSON(w, http.StatusOK, statusChangeResponse{ID: ref.ID, Status: entry.To, UpdatedAt: entry.CreatedAt})
			return
		}
	}
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.statusRefused(r, p, ref, err)
		writeErrorMessage(w, r, http.StatusUnprocessableEntity, msgDadosInvalidos)
	case errors.Is(err, domain.ErrStaleStatus):
		h.statusRefused(r, p, ref, err)
		writeErrorMessage(w, r, http.StatusConflict, msgStaleStatus)
	default:
		h.sec.RespondError(w, r, err)
	}
}

func (h *handlers) statusRefused(r *http.Request, p security.Principal, ref security.TicketRef, err error) {
	h.logger.InfoContext(r.Context(), "ticket status refused",
		slog.Int64("ticket_id", int64(ref.ID)),
		slog.Int64("actor_id", int64(p.UserID)),
		slog.String("reason", err.Error()),
	)
}

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

func (h *handlers) replyRefused(r *http.Request, p security.Principal, ref security.TicketRef, err error) {
	h.logger.InfoContext(r.Context(), "ticket message refused",
		slog.Int64("ticket_id", int64(ref.ID)),
		slog.Int64("actor_id", int64(p.UserID)),
		slog.String("reason", err.Error()),
	)
}

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
