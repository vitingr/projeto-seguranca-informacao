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

type AuthService interface {
	Authenticate(ctx context.Context, login, password string) (domain.User, error)
}

type UserService interface {
	Register(ctx context.Context, in service.RegisterInput) (domain.User, error)
}

type TicketService interface {
	Create(ctx context.Context, author domain.UserID, draft domain.TicketDraft) (domain.Ticket, error)
	// ListOwn returns a page of the tickets of author, the session user.
	ListOwn(ctx context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error)
	// Get returns a ticket and its author's login.
	Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error)
}

type Deps struct {
	Logger   *slog.Logger
	Security *security.Security
	Auth     AuthService
	Users    UserService
	Tickets  TicketService
	// TicketLimiter limits ticket creation per user (spec S-12).
	TicketLimiter *security.RateLimiter
}

func NewHandler(d Deps) (http.Handler, error) {
	if d.Logger == nil || d.Security == nil || d.Auth == nil || d.Users == nil || d.Tickets == nil || d.TicketLimiter == nil {
		return nil, errors.New("web: dependências obrigatórias ausentes")
	}

	h := &handlers{
		logger: d.Logger, sec: d.Security, auth: d.Auth, users: d.Users,
		tickets: d.Tickets, ticketLimiter: d.TicketLimiter,
	}

	mux := http.NewServeMux()
	for _, rt := range h.routes() {
		mux.Handle(rt.pattern(), chain(rt.handler, d.Security.Authorize(rt.policy)))
	}

	return chain(mux,
		requestID,
		accessLog(d.Logger),
		d.Security.Recover,
		d.Security.RequestTimeout,
		d.Security.SecurityHeaders,
		d.Security.Session,
		d.Security.CSRF,
	), nil
}

func chain(h http.Handler, mws ...security.Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
