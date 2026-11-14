package security

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// ErrorRenderer escreve a resposta genérica de erro para o status informado.
// É fornecido pela camada web (JSON); nunca deve expor detalhes internos.
type ErrorRenderer func(w http.ResponseWriter, r *http.Request, status int)

// Recover captura panics dos handlers, registra o valor e a stack trace no
// log (com o request ID do contexto) e responde 500 com o erro genérico,
// sem expor nenhum detalhe ao usuário.
//
// http.ErrAbortHandler é repassado, preservando a semântica do net/http.
func (s *Security) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer s.recoverPanic(w, r)
		next.ServeHTTP(w, r)
	})
}

// recoverPanic precisa ser chamada diretamente por defer para que recover()
// capture o panic do handler.
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

// RespondError registra err no log e responde com o erro genérico
// correspondente, sem expor a mensagem do erro ao usuário.
func (s *Security) RespondError(w http.ResponseWriter, r *http.Request, err error) {
	// TODO(etapa 2): mapear erros de domínio para status adequados
	// (ErrInvalidInput → 400, ErrNotFound → 404, ErrConflict → 409).
	// Até lá, todo erro vira 500 genérico: falha fechada, sem vazar detalhes.
	s.logger.ErrorContext(r.Context(), "security: erro ao processar requisição",
		slog.Any("error", err),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)
	s.renderError(w, r, http.StatusInternalServerError)
}
