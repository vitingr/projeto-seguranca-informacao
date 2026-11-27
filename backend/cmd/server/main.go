package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"backend_si/internal/config"
	"backend_si/internal/repository"
	"backend_si/internal/security"
	"backend_si/internal/service"
	"backend_si/internal/web"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	// requestTimeout is the context deadline of every request (D-21). It
	// is below writeTimeout so the JSON error is written before the
	// connection is cut; the http.Server timeouts alone do not cancel the
	// request context on HTTP/1.1.
	requestTimeout = 10 * time.Second
	maxHeaderBytes = 16 << 10

	healthcheckTimeout = 3 * time.Second
)

// The repository implements the stores security declares; it may not
// import security (depguard), so the composition root checks it.
var (
	_ security.SessionStore     = (*repository.Sessions)(nil)
	_ security.TicketOwnerStore = (*repository.Tickets)(nil)
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Getenv, os.Stdout)
	stop()
	if err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	level := new(slog.LevelVar)
	logger := slog.New(web.NewLogHandler(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: level})))

	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		logger.Error("configuração inválida", slog.Any("error", err))
		return err
	}
	level.Set(cfg.LogLevel)

	switch cmd {
	case "serve":
		err = serve(ctx, cfg, logger)
	case "healthcheck":
		err = healthcheck(ctx, cfg)
	case "seed":
		err = seed(ctx, cfg, getenv, logger)
	default:
		err = fmt.Errorf("subcomando desconhecido %q (use serve, healthcheck ou seed)", cmd)
	}
	if err != nil {
		logger.Error("encerrado com erro", slog.String("command", cmd), slog.Any("error", err))
	}
	return err
}

// serveSecurityOptions are the security options of the server. tickets is
// the owner store of TicketAccess (spec 003 S-2): without it every route
// with a ticket id answers 500.
func serveSecurityOptions(cfg config.Config, logger *slog.Logger, sessions security.SessionStore, tickets security.TicketOwnerStore) security.Options {
	return security.Options{
		Logger:                 logger,
		Sessions:               sessions,
		Tickets:                tickets,
		RenderError:            web.WriteError,
		SessionIdleTimeout:     cfg.SessionIdleTimeout,
		SessionAbsoluteTimeout: cfg.SessionAbsoluteTimeout,
		RequestTimeout:         requestTimeout,
	}
}

func serve(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	logger.Info("iniciando", slog.Any("config", cfg))

	tlsConfig, err := serverTLSConfig(cfg, logger)
	if err != nil {
		return err
	}

	db, err := repository.Open(ctx, cfg.DatabaseURL, repository.PoolOptions{
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
	})
	if err != nil {
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("falha ao fechar o banco", slog.Any("error", cerr))
		}
	}()

	tickets := repository.NewTickets(db)
	sec, err := security.New(serveSecurityOptions(cfg, logger, repository.NewSessions(db), tickets))
	if err != nil {
		return err
	}

	users := repository.NewUsers(db)
	auth, err := service.NewAuth(ctx, users, sec, sec)
	if err != nil {
		return err
	}
	ticketLimiter, err := sec.NewRateLimiter(security.TicketCreateLimit, security.TicketCreateWindow)
	if err != nil {
		return err
	}
	messageLimiter, err := sec.NewRateLimiter(security.MessageCreateLimit, security.MessageCreateWindow)
	if err != nil {
		return err
	}
	statusLimiter, err := sec.NewRateLimiter(security.StatusChangeLimit, security.StatusChangeWindow)
	if err != nil {
		return err
	}
	handler, err := web.NewHandler(web.Deps{
		Logger:        logger,
		Security:      sec,
		Auth:          auth,
		Users:         service.NewUsers(users, sec, sec),
		Tickets:       service.NewTicketsWithThread(tickets, repository.NewTicketEntries(db), sec),
		TicketLimiter: ticketLimiter,
		// Spec 003 S-9: replies of both sides share one limiter per user.
		MessageLimiter: messageLimiter,
		StatusLimiter:  statusLimiter,
	})
	if err != nil {
		return err
	}

	// D-9: purge expired sessions every idle timeout. The deferred wait runs
	// before db.Close, so the purge never uses a closed pool. Users with no
	// ticket request inside the rate limit window leave the limiter's
	// memory once per window.
	purgeCtx, stopPurge := context.WithCancel(ctx)
	var purge sync.WaitGroup
	purge.Go(func() { sec.PurgeExpiredSessions(purgeCtx, cfg.SessionIdleTimeout) })
	purge.Go(func() { sec.PurgeIdleRateLimits(purgeCtx, ticketLimiter, security.TicketCreateWindow) })
	purge.Go(func() { sec.PurgeIdleRateLimits(purgeCtx, messageLimiter, security.MessageCreateWindow) })
	purge.Go(func() { sec.PurgeIdleRateLimits(purgeCtx, statusLimiter, security.StatusChangeWindow) })
	defer func() {
		stopPurge()
		purge.Wait()
	}()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveOn(srv, ln, logger)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("servidor HTTPS: %w", err)
	case <-ctx.Done():
		logger.Info("sinal recebido, encerrando")
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("servidor encerrado")
	return nil
}

// serverTLSConfig loads the server certificate. With APP_TLS_MODE=proxy it
// loads nothing and returns nil: a proxy in front of the server terminates
// TLS, and the server speaks plain HTTP to it.
func serverTLSConfig(cfg config.Config, logger *slog.Logger) (*tls.Config, error) {
	if cfg.TLSMode == config.TLSModeProxy {
		logger.Warn("APP_TLS_MODE=proxy: serving plain HTTP, TLS must be terminated by a proxy in front of the server")
		return nil, nil
	}
	return security.NewTLSConfig(cfg.TLSCertFile, cfg.TLSKeyFile)
}

// serveOn serves srv on ln over HTTPS, or over plain HTTP when srv has no
// TLS config (APP_TLS_MODE=proxy).
func serveOn(srv *http.Server, ln net.Listener, logger *slog.Logger) error {
	if srv.TLSConfig == nil {
		logger.Info("plain HTTP server listening behind a TLS terminating proxy", slog.String("addr", srv.Addr))
		return srv.Serve(ln)
	}
	logger.Info("servidor HTTPS escutando", slog.String("addr", srv.Addr))
	return srv.ServeTLS(ln, "", "")
}

func healthcheck(ctx context.Context, cfg config.Config) error {
	_, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return fmt.Errorf("healthcheck: endereço inválido: %w", err)
	}

	scheme, transport := "http", &http.Transport{}
	if cfg.TLSMode != config.TLSModeProxy {
		pem, rerr := os.ReadFile(cfg.TLSCertFile)
		if rerr != nil {
			return fmt.Errorf("healthcheck: ler certificado: %w", rerr)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return errors.New("healthcheck: certificado PEM inválido")
		}
		tlsConfig, terr := security.NewClientTLSConfig(roots)
		if terr != nil {
			return terr
		}
		scheme, transport.TLSClientConfig = "https", tlsConfig
	}
	client := &http.Client{Timeout: healthcheckTimeout, Transport: transport}

	url := scheme + "://" + net.JoinHostPort("127.0.0.1", port) + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status inesperado %d", resp.StatusCode)
	}
	return nil
}

func seed(ctx context.Context, cfg config.Config, getenv func(string) string, logger *slog.Logger) error {
	creds, err := config.LoadSeed(getenv)
	if err != nil {
		return err
	}

	db, err := repository.Open(ctx, cfg.DatabaseURL, repository.PoolOptions{MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	sec, err := security.New(security.Options{
		Logger:                 logger,
		Sessions:               repository.NewSessions(db),
		RenderError:            web.WriteError,
		SessionIdleTimeout:     cfg.SessionIdleTimeout,
		SessionAbsoluteTimeout: cfg.SessionAbsoluteTimeout,
		RequestTimeout:         requestTimeout,
	})
	if err != nil {
		return err
	}

	users := service.NewUsers(repository.NewUsers(db), sec, sec)
	admin, err := users.CreateFirstAdmin(ctx, creds.AdminLogin, creds.AdminPassword)
	if err != nil {
		return err
	}
	logger.Info("administrador criado", slog.String("login", admin.Login))
	return nil
}
