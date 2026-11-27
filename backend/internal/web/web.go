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
	ListOwn(ctx context.Context, author domain.UserID, q domain.TicketListQuery) (domain.TicketPage, error)
	ListQueue(ctx context.Context, q domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error)
	Authors(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id domain.TicketID) (domain.Ticket, string, error)
	Thread(ctx context.Context, id domain.TicketID, before domain.TicketEntryID) (domain.ThreadPage, error)
	Reply(ctx context.Context, actor domain.UserID, ticket domain.TicketID, raw string) (domain.TicketEntry, error)
	ChangeStatus(ctx context.Context, actor domain.UserID, ticket domain.TicketID, from, to domain.TicketStatus) (domain.TicketEntry, error)
}

type Deps struct {
	Logger         *slog.Logger
	Security       *security.Security
	Auth           AuthService
	Users          UserService
	Tickets        TicketService
	TicketLimiter  *security.RateLimiter
	MessageLimiter *security.RateLimiter
	StatusLimiter  *security.RateLimiter
}

func NewHandler(d Deps) (http.Handler, error) {
	if d.Logger == nil || d.Security == nil || d.Auth == nil || d.Users == nil || d.Tickets == nil || d.TicketLimiter == nil ||
		d.MessageLimiter == nil || d.StatusLimiter == nil {
		return nil, errors.New("web: dependências obrigatórias ausentes")
	}

	h := &handlers{
		logger: d.Logger, sec: d.Security, auth: d.Auth, users: d.Users,
		tickets: d.Tickets, ticketLimiter: d.TicketLimiter, messageLimiter: d.MessageLimiter,
		statusLimiter: d.StatusLimiter,
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
