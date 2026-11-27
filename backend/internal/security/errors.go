package security

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"backend_si/internal/domain"
)

type ErrorRenderer func(w http.ResponseWriter, r *http.Request, status int)

func (s *Security) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer s.recoverPanic(w, r)
		next.ServeHTTP(w, r)
	})
}

func (s *Security) recoverPanic(w http.ResponseWriter, r *http.Request) {
	rec := recover()
	if rec == nil {
		return
	}
	if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		panic(rec)
	}
	s.logger.ErrorContext(r.Context(), "security: panic recuperado",
		slog.Any("panic", rec),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("stack", string(debug.Stack())),
	)
	s.renderError(w, r, http.StatusInternalServerError)
}

func (s *Security) RespondError(w http.ResponseWriter, r *http.Request, err error) {
	status := statusFor(err)
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	s.logger.Log(r.Context(), level, "security: erro ao processar requisição",
		slog.Any("error", err),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
	)
	s.renderError(w, r, status)
}

func statusFor(err error) int {
	switch {
	case err == nil:
		return http.StatusInternalServerError
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

var errInvalidSessionUser = errors.New("security: start session: invalid user")
