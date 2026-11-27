package main

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"backend_si/internal/config"
	"backend_si/internal/repository"
	"backend_si/internal/security"
)

func TestServeSecurityOptionsWireTheTicketStore(t *testing.T) {
	t.Parallel()

	cfg := config.Config{SessionIdleTimeout: 15 * time.Minute, SessionAbsoluteTimeout: 8 * time.Hour}
	tickets := repository.NewTickets(nil)
	opts := serveSecurityOptions(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), repository.NewSessions(nil), tickets)

	if opts.Tickets != security.TicketOwnerStore(tickets) {
		t.Errorf("Options.Tickets = %v, want the ticket repository", opts.Tickets)
	}
	if opts.Sessions == nil || opts.RenderError == nil || opts.RequestTimeout != requestTimeout {
		t.Errorf("options = %+v, want sessions, the renderer and requestTimeout", opts)
	}
	if _, err := security.New(opts); err != nil {
		t.Errorf("security.New: %v", err)
	}
}
