package web

import (
	"errors"
	"log/slog"
	"net/http"

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
	logger *slog.Logger
	sec    *security.Security
	auth   AuthService
	users  UserService
	posts  PostService
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
	writeJSON(w, http.StatusOK, userResponse{UserID: p.UserID, Role: p.Role})
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
