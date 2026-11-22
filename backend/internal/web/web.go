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

// PostService runs the wall operations (spec 005) for the calling user.
type PostService interface {
	List(ctx context.Context, page int) (domain.PostPage, error)
	Create(ctx context.Context, author domain.UserID, in service.PostInput) (domain.Post, error)
	Update(ctx context.Context, author domain.UserID, id domain.PostID, in service.PostInput) (domain.Post, error)
	Delete(ctx context.Context, author domain.UserID, id domain.PostID) error
}

type Deps struct {
	Logger   *slog.Logger
	Security *security.Security
	Auth     AuthService
	Users    UserService
	Posts    PostService
}

func NewHandler(d Deps) (http.Handler, error) {
	if d.Logger == nil || d.Security == nil || d.Auth == nil || d.Users == nil || d.Posts == nil {
		return nil, errors.New("web: dependências obrigatórias ausentes")
	}

	h := &handlers{logger: d.Logger, sec: d.Security, auth: d.Auth, users: d.Users, posts: d.Posts}

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
