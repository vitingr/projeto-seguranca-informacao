package security

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"backend_si/internal/domain"
)

type Middleware func(http.Handler) http.Handler

type Options struct {
	Logger             *slog.Logger
	Sessions           SessionStore
	RenderError        ErrorRenderer
	SessionIdleTimeout time.Duration
	// SessionAbsoluteTimeout is the maximum lifetime of a session from its
	// creation, whatever its activity (D-18). Required (>= idle timeout).
	SessionAbsoluteTimeout time.Duration
	Argon2                 Argon2Params
	// MaxConcurrentHashes is the number of Argon2id computations that may
	// run at once (D-19). Zero means runtime.GOMAXPROCS(0).
	MaxConcurrentHashes int
	// RequestTimeout is the context deadline of every request (D-21). It
	// must be below the http.Server WriteTimeout so the error response is
	// written before the connection is cut. Required (> 0).
	RequestTimeout time.Duration
	// Tickets tells the author of a ticket to TicketAccess (spec 003 S-2).
	// Optional for New; without it TicketAccess answers 500 (fail closed).
	Tickets TicketOwnerStore
	Now     func() time.Time
}

// TicketOwnerStore reads who wrote a ticket. TicketAuthor returns an error
// wrapping domain.ErrNotFound when the ticket does not exist.
type TicketOwnerStore interface {
	TicketAuthor(ctx context.Context, id domain.TicketID) (domain.UserID, error)
}

type Security struct {
	logger      *slog.Logger
	sessions    SessionStore
	tickets     TicketOwnerStore
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

var ErrInvalidOptions = errors.New("security: invalid options")

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
		tickets:     opts.Tickets,
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

func (s *Security) deny(w http.ResponseWriter, r *http.Request, status int, reason string) {
	s.logger.WarnContext(r.Context(), "security: acesso negado",
		slog.String("reason", reason),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
	)
	s.renderError(w, r, status)
}
