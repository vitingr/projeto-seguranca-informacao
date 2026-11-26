package web

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
	"backend_si/internal/service"
)

const (
	msgLoginInvalido  = "Login ou senha inválidos."
	msgDadosInvalidos = "Dados inválidos. Revise os campos e tente novamente."
	msgLoginEmUso     = "Este login já está em uso."
)

type handlers struct {
	logger  *slog.Logger
	sec     *security.Security
	auth    AuthService
	users   UserService
	tickets TicketService
	// ticketLimiter runs inside Authorize on POST /api/tickets, so only
	// comum users that passed Session and CSRF use their quota (S-12).
	ticketLimiter *security.RateLimiter
}

type loginRequest struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
}

type cadastroRequest struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
	Papel string `json:"papel"`
}

type userResponse struct {
	UserID domain.UserID `json:"user_id"`
	Login  string        `json:"login,omitempty"`
	Role   domain.Role   `json:"role"`
	// CreatedAt is the registration time, sent by GET /api/me only.
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// ticketResponse is what the confirmation shows (spec AC-5.2). It never
// echoes the field contents.
type ticketResponse struct {
	ID        domain.TicketID     `json:"id"`
	Type      domain.TicketType   `json:"type"`
	Title     string              `json:"title"`
	Status    domain.TicketStatus `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type csrfResponse struct {
	CSRFToken string `json:"csrf_token"`
	Header    string `json:"header"`
}

type areaResponse struct {
	Area string `json:"area"`
}

type statusResponse struct {
	Status string `json:"status"`
}

// --- handlers ---------------------------------------------------------------

func (h *handlers) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

func (h *handlers) csrf(w http.ResponseWriter, r *http.Request) {
	token, err := h.sec.CSRFToken(w, r)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, csrfResponse{
		CSRFToken: token,
		Header:    security.CSRFHeaderName,
	})
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := h.sec.DecodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, err)
		return
	}

	user, err := h.auth.Authenticate(r.Context(), req.Login, req.Senha)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeErrorMessage(w, r, http.StatusUnauthorized, msgLoginInvalido)
			return
		}
		h.sec.RespondError(w, r, err)
		return
	}

	if err := h.sec.StartSession(w, r, user); err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse{UserID: user.ID, Login: user.Login, Role: user.Role})
}

func (h *handlers) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sec.EndSession(w, r); err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) me(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	me := userResponse{UserID: p.UserID, Login: p.Login, Role: p.Role}
	if !p.CreatedAt.IsZero() {
		createdAt := p.CreatedAt.UTC()
		me.CreatedAt = &createdAt
	}
	writeJSON(w, http.StatusOK, me)
}

func (h *handlers) cadastro(w http.ResponseWriter, r *http.Request) {
	var req cadastroRequest
	if err := h.sec.DecodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, err)
		return
	}

	user, err := h.users.Register(r.Context(), service.RegisterInput{
		Login:    req.Login,
		Password: req.Senha,
		Role:     req.Papel,
	})
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeErrorMessage(w, r, http.StatusUnprocessableEntity, msgDadosInvalidos)
	case errors.Is(err, domain.ErrConflict):
		writeErrorMessage(w, r, http.StatusConflict, msgLoginEmUso)
	case err != nil:
		h.sec.RespondError(w, r, err)
	default:
		h.auditRegistered(r, user)
		writeJSON(w, http.StatusCreated, userResponse{UserID: user.ID, Login: user.Login, Role: user.Role})
	}
}

// auditRegistered records who registered whom (003 D-11). It logs IDs and
// the role only: never the login or the password.
func (h *handlers) auditRegistered(r *http.Request, user domain.User) {
	var actor domain.UserID
	if p, ok := security.PrincipalFrom(r.Context()); ok {
		actor = p.UserID
	}
	h.logger.InfoContext(r.Context(), "user registered",
		slog.Int64("actor_user_id", int64(actor)),
		slog.Int64("user_id", int64(user.ID)),
		slog.String("role", string(user.Role)),
	)
}

// createTicket creates a support ticket for the comum user of the session
// (spec 001-ticket-builder). A body that is not well-formed for the type is
// 400; a well-formed body with an invalid value is 422 with the generic
// message, never naming the field (Q-12). The author comes only from the
// session (S-4). The log holds ids, the type and a fixed reason, never a
// field value (S-7).
func (h *handlers) createTicket(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}

	draft, err := h.sec.DecodeTicket(w, r)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.ticketRefused(r, p.UserID, err)
		}
		h.badRequest(w, r, err)
		return
	}

	ticket, err := h.tickets.Create(r.Context(), p.UserID, draft)
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.ticketRefused(r, p.UserID, err)
		writeErrorMessage(w, r, http.StatusUnprocessableEntity, msgDadosInvalidos)
	case err != nil:
		h.sec.RespondError(w, r, err)
	default:
		h.logger.InfoContext(r.Context(), "ticket created",
			slog.Int64("ticket_id", int64(ticket.ID)),
			slog.Int64("author_id", int64(ticket.AuthorID)),
			slog.String("type", string(ticket.Type)),
		)
		writeJSON(w, http.StatusCreated, ticketResponse{
			ID:        ticket.ID,
			Type:      ticket.Type,
			Title:     ticket.Title,
			Status:    ticket.Status,
			CreatedAt: ticket.CreatedAt,
			UpdatedAt: ticket.UpdatedAt,
		})
	}
}

// ticketRefused logs a refused ticket request. The reason is the error
// text, which the security and service layers build from fixed strings and
// field names only, never from the body.
func (h *handlers) ticketRefused(r *http.Request, author domain.UserID, err error) {
	h.logger.InfoContext(r.Context(), "ticket refused",
		slog.Int64("author_id", int64(author)),
		slog.String("reason", err.Error()),
	)
}

func (h *handlers) administradores(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, areaResponse{Area: "administradores"})
}

func (h *handlers) usuarios(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, areaResponse{Area: "usuarios"})
}

func (h *handlers) notFound(w http.ResponseWriter, r *http.Request) {
	WriteError(w, r, http.StatusNotFound)
}

func (h *handlers) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, domain.ErrInvalidInput) {
		WriteError(w, r, http.StatusBadRequest)
		return
	}
	h.sec.RespondError(w, r, err)
}
