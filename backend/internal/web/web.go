// Package web é a API HTTP/JSON consumida pelo frontend (Next.js): tabela de
// rotas, handlers, respostas JSON e o encadeamento de middlewares. Os
// controles de segurança vêm do pacote security; este pacote apenas os
// aplica na ordem definida.
package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"backend_si/internal/domain"
	"backend_si/internal/security"
	"backend_si/internal/service"
)

// AuthService é o caso de uso de login consumido pelos handlers.
type AuthService interface {
	Authenticate(ctx context.Context, login, password string) (domain.User, error)
}

// UserService é o caso de uso de cadastro consumido pelos handlers.
type UserService interface {
	Register(ctx context.Context, in service.RegisterInput) (domain.User, error)
}

// Deps são as dependências da camada web.
type Deps struct {
	Logger   *slog.Logger
	Security *security.Security
	Auth     AuthService
	Users    UserService
}

// NewHandler monta, uma única vez, o handler completo da aplicação:
//
//	RequestID → AccessLog → Recover → SecurityHeaders → Session → CSRF → mux
//
// e, dentro do mux, cada rota é envolvida por Authorize(política da rota)
// antes do handler. RequestID e AccessLog são observabilidade e ficam por
// fora para que até respostas de panic sejam registradas com request ID.
func NewHandler(d Deps) (http.Handler, error) {
	if d.Logger == nil || d.Security == nil || d.Auth == nil || d.Users == nil {
		return nil, errors.New("web: dependências obrigatórias ausentes")
	}

	h := &handlers{sec: d.Security, auth: d.Auth, users: d.Users}

	mux := http.NewServeMux()
	for _, rt := range h.routes() {
		mux.Handle(rt.pattern(), chain(rt.handler, d.Security.Authorize(rt.policy)))
	}

	return chain(mux,
		requestID,
		accessLog(d.Logger),
		d.Security.Recover,
		d.Security.SecurityHeaders,
		d.Security.Session,
		d.Security.CSRF,
	), nil
}

// chain aplica os middlewares de modo que o primeiro da lista seja o mais
// externo: chain(h, a, b) == a(b(h)).
func chain(h http.Handler, mws ...security.Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
