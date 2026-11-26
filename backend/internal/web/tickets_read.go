package web

import (
	"errors"
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

// adminTicketSummary is a row of the admin queue: the summary plus the
// author's login (spec 003 AC-4.2), never the author's id.
type adminTicketSummary struct {
	ID          domain.TicketID     `json:"id"`
	Type        domain.TicketType   `json:"type"`
	Status      domain.TicketStatus `json:"status"`
	Title       string              `json:"title"`
	AuthorLogin string              `json:"author_login"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// statusCounts is the summary above the queue (AC-4.5), over the whole
// queue whatever the filters.
type statusCounts struct {
	Open       int `json:"aberto"`
	InProgress int `json:"em_andamento"`
	Resolved   int `json:"resolvido"`
	Closed     int `json:"fechado"`
}

// queuePage is a page of the admin queue.
type queuePage struct {
	Items        []adminTicketSummary `json:"items"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	Pages        int                  `json:"pages"`
	PageSize     int                  `json:"page_size"`
	StatusCounts statusCounts         `json:"status_counts"`
}

// listQueue answers the queue of every user's tickets (spec 003 US-4),
// filtered by status, type and author login, with the status counts of
// the whole queue. Only admins reach it (route policy, S-4).
func (h *handlers) listQueue(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	q, err := h.listQuery(r, security.TicketListQueue)
	if err != nil {
		h.queryRefused(r, p.UserID, err)
		h.badRequest(w, r, err)
		return
	}
	page, counts, err := h.tickets.ListQueue(r.Context(), q)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	out := queuePage{
		Items: make([]adminTicketSummary, 0, len(page.Items)),
		Total: page.Total, Page: page.Page, Pages: page.Pages, PageSize: listPageSize,
		StatusCounts: statusCounts{Open: counts.Open, InProgress: counts.InProgress, Resolved: counts.Resolved, Closed: counts.Closed},
	}
	for _, s := range page.Items {
		out.Items = append(out.Items, adminTicketSummary{
			ID: s.ID, Type: s.Type, Status: s.Status, Title: s.Title, AuthorLogin: s.AuthorLogin,
			CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
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

// ticketDetail is the ticket page (spec 003 US-2): the summary plus every
// field of the type in data, with the keys of the creation request
// (security.TicketDataKeys) and null for an empty optional field (plan
// P-10). It carries no author data (S-13).
type ticketDetail struct {
	ID        domain.TicketID     `json:"id"`
	Type      domain.TicketType   `json:"type"`
	Status    domain.TicketStatus `json:"status"`
	Title     string              `json:"title"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Data      map[string]*string  `json:"data"`
}

func newTicketDetail(t domain.Ticket) ticketDetail {
	return ticketDetail{
		ID: t.ID, Type: t.Type, Status: t.Status, Title: t.Title,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, Data: ticketData(t),
	}
}

// ticketData returns the fields of the ticket's type by their request key.
// A key the type has but the ticket left empty (only the deploy notes may
// be) is null.
func ticketData(t domain.Ticket) map[string]*string {
	values := map[string]string{
		"titulo":        t.Title,
		"sistema":       t.System,
		"nivel_acesso":  string(t.AccessLevel),
		"justificativa": t.Justification,
		"descricao":     t.Description,
		"passos":        t.Steps,
		"severidade":    string(t.Severity),
		"projeto":       t.Project,
		"repositorio":   t.Repository,
		"ambiente":      string(t.Environment),
		"observacoes":   t.Notes,
	}
	keys := security.TicketDataKeys()[t.Type]
	data := make(map[string]*string, len(keys))
	for _, key := range keys {
		data[key] = nil
		if v := values[key]; v != "" {
			data[key] = &v
		}
	}
	return data
}

// getOwnTicket answers the ticket page of a comum user (spec 003 US-2).
// TicketAccess already decided that the ticket exists and is the session
// user's (S-2); the id comes only from the TicketRef it stored.
func (h *handlers) getOwnTicket(w http.ResponseWriter, r *http.Request) {
	ref, ok := security.TicketRefFrom(r.Context())
	if !ok {
		h.sec.RespondError(w, r, errNoTicketRef)
		return
	}
	t, _, err := h.tickets.Get(r.Context(), ref.ID)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newTicketDetail(t))
}

// errNoTicketRef is a wiring error: a ticket route without TicketAccess.
// It is not a client error, so the answer is a 500 (fail closed).
var errNoTicketRef = errors.New("web: ticket route without TicketAccess")

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
