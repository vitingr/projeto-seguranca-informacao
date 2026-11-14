// Command server é o composition root da aplicação: lê a configuração,
// constrói as dependências e inicia o servidor HTTPS.
//
// Subcomandos:
//
//	server              inicia o servidor (padrão)
//	server serve        idem
//	server healthcheck  consulta GET /healthz local (usado pelo Docker)
//	server seed         cria o primeiro administrador a partir do ambiente
package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend_si/internal/config"
	"backend_si/internal/repository"
	"backend_si/internal/security"
	"backend_si/internal/service"
	"backend_si/internal/web"
)

// Timeouts do servidor HTTP: explícitos para mitigar Slowloris e conexões presas.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 16 << 10

	healthcheckTimeout = 3 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Getenv, os.Stdout)
	stop()
	if err != nil {
		os.Exit(1)
	}
}

// run despacha o subcomando. Todos os erros já são registrados no log;
// o retorno serve apenas para definir o código de saída.
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

func serve(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	logger.Info("iniciando", slog.Any("config", cfg))

	tlsConfig, err := security.NewTLSConfig(cfg.TLSCertFile, cfg.TLSKeyFile)
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

	sec, err := security.New(security.Options{
		Logger:             logger,
		Sessions:           repository.NewSessions(db),
		RenderError:        web.WriteError,
		SessionIdleTimeout: cfg.SessionIdleTimeout,
	})
	if err != nil {
		return err
	}

	users := repository.NewUsers(db)
	handler, err := web.NewHandler(web.Deps{
		Logger:   logger,
		Security: sec,
		Auth:     service.NewAuth(users, sec),
		Users:    service.NewUsers(users, sec, sec),
	})
	if err != nil {
		return err
	}

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

	errCh := make(chan error, 1)
	go func() {
		logger.Info("servidor HTTPS escutando", slog.String("addr", cfg.Addr))
		// Certificado e chave já estão em tlsConfig.Certificates.
		errCh <- srv.ListenAndServeTLS("", "")
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

// healthcheck consulta https://127.0.0.1:<porta>/healthz validando o
// certificado do próprio servidor (sem InsecureSkipVerify). Usado pelo
// HEALTHCHECK do Docker, já que a imagem distroless não tem curl.
func healthcheck(ctx context.Context, cfg config.Config) error {
	_, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return fmt.Errorf("healthcheck: endereço inválido: %w", err)
	}
	pem, err := os.ReadFile(cfg.TLSCertFile)
	if err != nil {
		return fmt.Errorf("healthcheck: ler certificado: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return errors.New("healthcheck: certificado PEM inválido")
	}

	tlsConfig, err := security.NewClientTLSConfig(roots)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout:   healthcheckTimeout,
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}

	url := "https://" + net.JoinHostPort("127.0.0.1", port) + "/healthz"
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

// seed cria o primeiro administrador a partir de SEED_ADMIN_LOGIN e
// SEED_ADMIN_PASSWORD. Na etapa 1 o serviço é um stub e falha fechado.
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
		Logger:             logger,
		Sessions:           repository.NewSessions(db),
		RenderError:        web.WriteError,
		SessionIdleTimeout: cfg.SessionIdleTimeout,
	})
	if err != nil {
		return err
	}

	users := service.NewUsers(repository.NewUsers(db), sec, sec)
	//nolint:staticcheck // SA4023: the stage-1 stub always fails. TODO(etapa 2): remove.
	admin, err := users.CreateFirstAdmin(ctx, creds.AdminLogin, creds.AdminPassword)
	if err != nil { //nolint:staticcheck // SA4023: the stage-1 stub always fails. TODO(etapa 2): remove.
		return err
	}
	logger.Info("administrador criado", slog.String("login", admin.Login))
	return nil
}
