// Package security é o ÚNICO ponto de segurança da aplicação.
//
// Todos os controles ficam aqui: TLS (tls.go), cabeçalhos HTTP (headers.go),
// sessão (session.go), CSRF (csrf.go), autorização por papéis (authz.go),
// hash de senha (password.go), validação de entrada (validation.go) e
// tratamento de erros (errors.go). Os demais pacotes apenas consomem esta API.
//
// Princípio geral: falhar fechado. Tudo o que não for explicitamente
// permitido é negado, inclusive o valor zero dos tipos de política e todos
// os stubs ainda não implementados (marcados com TODO(etapa 2)).
package security

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Middleware decora um http.Handler.
type Middleware func(http.Handler) http.Handler

// Options reúne as dependências e os parâmetros de Security.
type Options struct {
	// Logger recebe os eventos de segurança. Obrigatório.
	Logger *slog.Logger
	// Sessions persiste as sessões no servidor. Obrigatório.
	Sessions SessionStore
	// RenderError escreve a resposta genérica de erro. Obrigatório.
	RenderError ErrorRenderer
	// SessionIdleTimeout é o tempo máximo de inatividade. Obrigatório (> 0).
	SessionIdleTimeout time.Duration
	// Argon2 define os parâmetros de hash; o valor zero usa DefaultArgon2Params.
	Argon2 Argon2Params
	// Now é o relógio usado nas decisões de expiração; nil usa time.Now.
	Now func() time.Time
}

// Security é a fachada que expõe todos os mecanismos de segurança.
// Crie com New; o valor zero não é utilizável.
type Security struct {
	logger      *slog.Logger
	sessions    SessionStore
	renderError ErrorRenderer
	idleTimeout time.Duration
	argon2      Argon2Params
	now         func() time.Time
}

// ErrInvalidOptions é encapsulado pelos erros de validação de New.
var ErrInvalidOptions = errors.New("security: invalid options")

// New valida as opções e constrói a fachada de segurança.
func New(opts Options) (*Security, error) {
	var errs []error
	if opts.Logger == nil {
		errs = append(errs, fmt.Errorf("%w: Logger é obrigatório", ErrInvalidOptions))
	}
	if opts.Sessions == nil {
		errs = append(errs, fmt.Errorf("%w: Sessions é obrigatório", ErrInvalidOptions))
	}
	if opts.RenderError == nil {
		errs = append(errs, fmt.Errorf("%w: RenderError é obrigatório", ErrInvalidOptions))
	}
	if opts.SessionIdleTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%w: SessionIdleTimeout deve ser positivo", ErrInvalidOptions))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	params := opts.Argon2
	if params == (Argon2Params{}) {
		params = DefaultArgon2Params()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	return &Security{
		logger:      opts.Logger,
		sessions:    opts.Sessions,
		renderError: opts.RenderError,
		idleTimeout: opts.SessionIdleTimeout,
		argon2:      params,
		now:         now,
	}, nil
}

// deny registra a negação e responde com o erro genérico.
func (s *Security) deny(w http.ResponseWriter, r *http.Request, status int, reason string) {
	s.logger.WarnContext(r.Context(), "security: acesso negado",
		slog.String("reason", reason),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
	)
	s.renderError(w, r, status)
}
