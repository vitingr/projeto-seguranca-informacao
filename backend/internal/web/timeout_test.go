package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// waitingAuth stands for an Authenticate call stuck waiting for a hash slot:
// it returns the context error when the request context ends.
type waitingAuth struct{}

func (waitingAuth) Authenticate(ctx context.Context, _, _ string) (domain.User, error) {
	<-ctx.Done()
	return domain.User{}, fmt.Errorf("service: authenticate: %w", ctx.Err())
}

// TestLoginAnswersJSON500AtRequestDeadline checks D-21 through the full
// NewHandler chain: over HTTP/1.1, a login with a valid session and CSRF
// token that is stuck in the hash wait gets the generic JSON 500 at the
// RequestTimeout deadline, before the server WriteTimeout cuts the
// connection. It fails if RequestTimeout is not in the NewHandler chain:
// the client would then wait until its own 10s limit.
func TestLoginAnswersJSON500AtRequestDeadline(t *testing.T) {
	t.Parallel()

	const (
		requestTimeout = 300 * time.Millisecond
		writeTimeout   = 3 * time.Second
	)
	logger := slog.New(NewLogHandler(slog.NewJSONHandler(io.Discard, nil)))
	sec, err := security.New(security.Options{
		Logger:                 logger,
		Sessions:               newMemSessions(),
		RenderError:            WriteError,
		SessionIdleTimeout:     time.Minute,
		SessionAbsoluteTimeout: time.Hour,
		RequestTimeout:         requestTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := NewHandler(Deps{Logger: logger, Security: sec, Auth: waitingAuth{}, Users: fakeUsers{}, Posts: fakePosts{}, Audit: fakeAudit{}})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(app)
	ts.Config.ReadTimeout = writeTimeout
	ts.Config.WriteTimeout = writeTimeout
	ts.StartTLS()
	defer ts.Close()

	tr := ts.Client().Transport.(*http.Transport).Clone()
	tr.ForceAttemptHTTP2 = false
	tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	client := &http.Client{Transport: tr}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// GET /api/csrf: anonymous session cookie and token.
	csrfReq, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/csrf", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	csrfResp, err := client.Do(csrfReq)
	if err != nil {
		t.Fatal(err)
	}
	var csrf csrfResponse
	err = json.NewDecoder(csrfResp.Body).Decode(&csrf)
	_ = csrfResp.Body.Close()
	if err != nil || csrf.CSRFToken == "" {
		t.Fatalf("GET /api/csrf: %+v, %v", csrf, err)
	}
	var session *http.Cookie
	for _, c := range csrfResp.Cookies() {
		if c.Name == security.SessionCookieName {
			session = c
		}
	}
	if session == nil {
		t.Fatal("GET /api/csrf did not set the session cookie")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/api/login",
		strings.NewReader(`{"login":"admin","senha":"a long enough password"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(security.CSRFHeaderName, csrf.CSRFToken)
	req.AddCookie(session)

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("request failed after %v: %v", elapsed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Proto != "HTTP/1.1" {
		t.Fatalf("protocol = %s, want HTTP/1.1", resp.Proto)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (403 means the session or token was not accepted)", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	var body errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body.Error != "internal" || body.Message == "" || body.RequestID != resp.Header.Get("X-Request-Id") {
		t.Errorf("body = %+v, want the generic internal error with the request ID", body)
	}
	if elapsed < requestTimeout || elapsed >= writeTimeout {
		t.Errorf("answered after %v, want between %v and %v", elapsed, requestTimeout, writeTimeout)
	}
}
