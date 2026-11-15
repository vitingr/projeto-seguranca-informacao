package security

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_si/internal/domain"
)

// renderJSON is an ErrorRenderer that writes a JSON body, like web.WriteError.
func renderJSON(w http.ResponseWriter, _ *http.Request, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, `{"error":"internal","message":"generic"}`)
}

func TestRequestTimeoutSetsDeadline(t *testing.T) {
	t.Parallel()

	const timeout = 2 * time.Second
	s := newTestSecurityWith(t, func(o *Options) { o.RequestTimeout = timeout })
	var deadline time.Time
	var hasDeadline bool
	start := time.Now()
	s.RequestTimeout(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, hasDeadline = r.Context().Deadline()
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))
	end := time.Now()

	if !hasDeadline {
		t.Fatal("request context has no deadline")
	}
	// The deadline is set between start and end, so it lies in this window.
	if deadline.Before(start.Add(timeout)) || deadline.After(end.Add(timeout)) {
		t.Errorf("deadline %v after start, want %v", deadline.Sub(start), timeout)
	}
}

// TestHashWaitEndsAtRequestDeadline is the D-21 check: with every hash slot
// busy, a request that needs a hash must end at the RequestTimeout deadline
// with a 500, before the server WriteTimeout, on HTTP/1.1 as well as on
// HTTP/2. On HTTP/1.1 the http.Server timeouts alone never cancel the
// request context, so without RequestTimeout the wait would have no limit.
func TestHashWaitEndsAtRequestDeadline(t *testing.T) {
	t.Parallel()

	const (
		requestTimeout = 300 * time.Millisecond
		writeTimeout   = 3 * time.Second
	)
	calls := []struct {
		name string
		call func(ctx context.Context, s *Security) error
	}{
		{"HashPassword", func(ctx context.Context, s *Security) error {
			_, err := s.HashPassword(ctx, refPassword)
			return err
		}},
		{"VerifyPassword", func(ctx context.Context, s *Security) error {
			return s.VerifyPassword(ctx, refHash, refPassword)
		}},
	}
	for _, http2 := range []bool{false, true} {
		for _, c := range calls {
			wantProto := "HTTP/1.1"
			if http2 {
				wantProto = "HTTP/2.0"
			}
			t.Run(wantProto+" "+c.name, func(t *testing.T) {
				t.Parallel()
				s := newTestSecurityWith(t, func(o *Options) {
					o.MaxConcurrentHashes = 1
					o.RequestTimeout = requestTimeout
					o.RenderError = renderJSON
				})
				s.hashSlots <- struct{}{} // every slot is busy for the whole test
				defer s.releaseHashSlot()

				handlerErr := make(chan error, 1)
				ts := httptest.NewUnstartedServer(s.RequestTimeout(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					err := c.call(r.Context(), s)
					handlerErr <- err
					if err != nil {
						s.RespondError(w, r, err)
						return
					}
					w.WriteHeader(http.StatusOK)
				})))
				ts.Config.ReadTimeout = writeTimeout
				ts.Config.WriteTimeout = writeTimeout
				ts.EnableHTTP2 = http2
				ts.StartTLS()
				defer ts.Close()

				client := ts.Client()
				if !http2 {
					tr := client.Transport.(*http.Transport).Clone()
					tr.ForceAttemptHTTP2 = false
					tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
					client = &http.Client{Transport: tr}
				}
				// The client itself waits much longer than the deadline.
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, http.NoBody)
				if err != nil {
					t.Fatal(err)
				}

				start := time.Now()
				resp, err := client.Do(req)
				elapsed := time.Since(start)
				if err != nil {
					t.Fatalf("request failed after %v: %v (the response was not written)", elapsed, err)
				}
				defer func() { _ = resp.Body.Close() }()

				if resp.Proto != wantProto {
					t.Fatalf("protocol = %s, want %s", resp.Proto, wantProto)
				}
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("status = %d, want 500", resp.StatusCode)
				}
				var body struct{ Error string }
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Error != "internal" {
					t.Errorf("body is not the JSON error: %+v, %v", body, err)
				}
				if elapsed < requestTimeout || elapsed >= writeTimeout {
					t.Errorf("answered after %v, want between %v and %v", elapsed, requestTimeout, writeTimeout)
				}

				herr := <-handlerErr
				if !errors.Is(herr, context.DeadlineExceeded) {
					t.Errorf("handler error = %v, want context.DeadlineExceeded", herr)
				}
				if errors.Is(herr, domain.ErrInvalidCredentials) {
					t.Errorf("handler error = %v must not be a credential error", herr)
				}
			})
		}
	}
}
