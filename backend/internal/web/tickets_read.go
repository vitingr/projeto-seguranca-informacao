package web

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

const listPageSize = 20

type ticketSummary struct {
	ID        domain.TicketID     `json:"id"`
	Type      domain.TicketType   `json:"type"`
	Status    domain.TicketStatus `json:"status"`
	Title     string              `json:"title"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type ticketPage struct {
	Items    []ticketSummary `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	Pages    int             `json:"pages"`
	PageSize int             `json:"page_size"`
}

type adminTicketSummary struct {
	ID          domain.TicketID     `json:"id"`
	Type        domain.TicketType   `json:"type"`
	Status      domain.TicketStatus `json:"status"`
	Title       string              `json:"title"`
	AuthorLogin string              `json:"author_login"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type statusCounts struct {
	Open       int `json:"aberto"`
	InProgress int `json:"em_andamento"`
	Resolved   int `json:"resolvido"`
	Closed     int `json:"fechado"`
}

type queuePage struct {
	Items        []adminTicketSummary `json:"items"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	Pages        int                  `json:"pages"`
	PageSize     int                  `json:"page_size"`
	StatusCounts statusCounts         `json:"status_counts"`
}

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

type authorList struct {
	Logins []string `json:"logins"`
}

var errAuthorsQuery = fmt.Errorf("web: queue authors: unknown parameter: %w", domain.ErrInvalidInput)

func (h *handlers) listQueueAuthors(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	values, err := h.sec.ReadQuery(r)
	if err == nil && len(values) > 0 {
		err = errAuthorsQuery
	}
	if err != nil {
		h.queryRefused(r, p.UserID, err)
		h.badRequest(w, r, err)
		return
	}
	logins, err := h.tickets.Authors(r.Context())
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	out := authorList{Logins: append([]string{}, logins...)}
	writeJSON(w, http.StatusOK, out)
}

func newTicketSummary(s domain.TicketSummary) ticketSummary {
	return ticketSummary{
		ID: s.ID, Type: s.Type, Status: s.Status, Title: s.Title,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

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

type adminTicketDetail struct {
	ID          domain.TicketID     `json:"id"`
	Type        domain.TicketType   `json:"type"`
	Status      domain.TicketStatus `json:"status"`
	Title       string              `json:"title"`
	AuthorLogin string              `json:"author_login"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Data        map[string]*string  `json:"data"`
}

func (h *handlers) getAdminTicket(w http.ResponseWriter, r *http.Request) {
	_, ref, ok := h.ticketCaller(w, r)
	if !ok {
		return
	}
	t, login, err := h.tickets.Get(r.Context(), ref.ID)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, adminTicketDetail{
		ID: t.ID, Type: t.Type, Status: t.Status, Title: t.Title, AuthorLogin: login,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, Data: ticketData(t),
	})
}

var errNoTicketRef = errors.New("web: ticket route without TicketAccess")

func (h *handlers) listQuery(r *http.Request, scope security.TicketListScope) (domain.TicketListQuery, error) {
	values, err := h.sec.ReadQuery(r)
	if err != nil {
		return domain.TicketListQuery{}, err
	}
	return h.sec.ParseTicketListQuery(values, scope)
}

func (h *handlers) queryRefused(r *http.Request, user domain.UserID, err error) {
	h.logger.InfoContext(r.Context(), "ticket query refused",
		slog.Int64("user_id", int64(user)),
		slog.String("reason", err.Error()),
	)
}
