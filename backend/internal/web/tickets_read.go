package web

import (
	"log/slog"
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// listPageSize is the fixed size of a page of "Meus tickets" and of the
// queue (spec 003 S-14), the LIMIT of the repository queries. The client
// cannot change it.
const listPageSize = 20

// ticketSummary is a row of "Meus tickets" (spec 003 section 4): never the
// fields of the type nor any author data (S-13).
type ticketSummary struct {
	ID        domain.TicketID     `json:"id"`
	Type      domain.TicketType   `json:"type"`
	Status    domain.TicketStatus `json:"status"`
	Title     string              `json:"title"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// ticketPage is a page of "Meus tickets". A page past the end has no items
// and the real totals (plan P-9).
type ticketPage struct {
	Items    []ticketSummary `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	Pages    int             `json:"pages"`
	PageSize int             `json:"page_size"`
}

func newTicketSummary(s domain.TicketSummary) ticketSummary {
	return ticketSummary{
		ID: s.ID, Type: s.Type, Status: s.Status, Title: s.Title,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

// listOwnTickets answers "Meus tickets" of the comum user of the session
// (spec 003 US-1). The author is the session user, never a parameter: a
// usuario parameter is a 400 (S-3), like any unknown, repeated, empty or
// malformed one (plan P-9).
func (h *handlers) listOwnTickets(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	q, err := h.listQuery(r, security.TicketListOwn)
	if err != nil {
		h.queryRefused(r, p.UserID, err)
		h.badRequest(w, r, err)
		return
	}

	page, err := h.tickets.ListOwn(r.Context(), p.UserID, q)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	out := ticketPage{
		Items: make([]ticketSummary, 0, len(page.Items)),
		Total: page.Total, Page: page.Page, Pages: page.Pages, PageSize: listPageSize,
	}
	for _, s := range page.Items {
		out.Items = append(out.Items, newTicketSummary(s))
	}
	writeJSON(w, http.StatusOK, out)
}

// listQuery reads the query string strictly and validates it as a list
// query of scope; errors wrap domain.ErrInvalidInput (400).
func (h *handlers) listQuery(r *http.Request, scope security.TicketListScope) (domain.TicketListQuery, error) {
	values, err := h.sec.ReadQuery(r)
	if err != nil {
		return domain.TicketListQuery{}, err
	}
	return h.sec.ParseTicketListQuery(values, scope)
}

// queryRefused logs a refused list or thread query. The reason is the error
// text, built from fixed strings by the security package, never from the
// query (S-12).
func (h *handlers) queryRefused(r *http.Request, user domain.UserID, err error) {
	h.logger.InfoContext(r.Context(), "ticket query refused",
		slog.Int64("user_id", int64(user)),
		slog.String("reason", err.Error()),
	)
}
