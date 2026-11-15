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
	"runtime"
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
	// SessionAbsoluteTimeout is the maximum lifetime of a session from its
	// creation, whatever its activity (D-18). Required (>= idle timeout).
	SessionAbsoluteTimeout time.Duration
	// Argon2 define os parâmetros de hash; o valor zero usa DefaultArgon2Params.
	// A custom value must be within the limits that VerifyPassword accepts.
	Argon2 Argon2Params
	// MaxConcurrentHashes is the number of Argon2id computations that may
	// run at once (D-19). Zero means runtime.GOMAXPROCS(0).
	MaxConcurrentHashes int
	// RequestTimeout is the context deadline of every request (D-21). It
	// must be below the http.Server WriteTimeout so the error response is
	// written before the connection is cut. Required (> 0).
	RequestTimeout time.Duration
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
	absTimeout  time.Duration
	argon2      Argon2Params
	hashSlots   chan struct{}
	reqTimeout  time.Duration
	now         func() time.Time
	// crossOrigin is the Fetch Metadata check of CSRF (D-5), with no
	// trusted origins and no bypass patterns.
	crossOrigin *http.CrossOriginProtection
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
	if opts.SessionAbsoluteTimeout <= 0 || opts.SessionAbsoluteTimeout < opts.SessionIdleTimeout {
		errs = append(errs, fmt.Errorf("%w: SessionAbsoluteTimeout must be positive and at least SessionIdleTimeout", ErrInvalidOptions))
	}
	if opts.RequestTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%w: RequestTimeout must be positive", ErrInvalidOptions))
	}
	if opts.MaxConcurrentHashes < 0 {
		errs = append(errs, fmt.Errorf("%w: MaxConcurrentHashes must not be negative", ErrInvalidOptions))
	}
	params := opts.Argon2
	if params == (Argon2Params{}) {
		params = DefaultArgon2Params()
	}
	if err := params.validate(); err != nil {
		errs = append(errs, fmt.Errorf("%w: Argon2: %w", ErrInvalidOptions, err))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	slots := opts.MaxConcurrentHashes
	if slots == 0 {
		slots = runtime.GOMAXPROCS(0)
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
		absTimeout:  opts.SessionAbsoluteTimeout,
		argon2:      params,
		hashSlots:   make(chan struct{}, slots),
		reqTimeout:  opts.RequestTimeout,
		now:         now,
		crossOrigin: http.NewCrossOriginProtection(),
	}, nil
}

// deny registra a negação e responde com o erro genérico.
//
//nolint:unparam // Authorize passes 401 for anonymous callers from task 6 on (D-1); remove then.
func (s *Security) deny(w http.ResponseWriter, r *http.Request, status int, reason string) {
	s.logger.WarnContext(r.Context(), "security: acesso negado",
		slog.String("reason", reason),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
	)
	s.renderError(w, r, status)
}
