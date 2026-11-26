package web

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// newTicketLimiter is the production limiter of ticket creation: 10
// requests per user in any 60-second window.
func newTicketLimiter(t *testing.T, sec *security.Security) *security.RateLimiter {
	t.Helper()
	l, err := sec.NewRateLimiter(security.TicketCreateLimit, security.TicketCreateWindow)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}
	return l
}

func TestNewHandlerRequiresTicketLimiter(t *testing.T) {
	t.Parallel()

	h, _ := newTestApp(t) // built from a complete Deps
	if _, err := NewHandler(Deps{Logger: h.logger, Security: h.sec, Auth: h.auth, Users: h.users, Tickets: h.tickets}); err == nil {
		t.Fatal("NewHandler without TicketLimiter must fail: the route would answer 500")
	}
}

// postAs sends POST /api/tickets with the given session and token.
func (f flowApp) postAs(t *testing.T, cookie *http.Cookie, token, body string) int {
	t.Helper()
	return f.send(t, http.MethodPost, "/api/tickets", body, cookie, map[string]string{security.CSRFHeaderName: token}).Code
}

// TestTicketRateLimit covers S-12, BR-9 and AC-6.7 on the server: 10
// creations in a minute are accepted, the 11th gets 429 with the generic
// body and Retry-After, and nothing more is stored.
func TestTicketRateLimit(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	f := newFlowAppWithLog(t, &logBuf)
	cookie, token := f.loginAs(t, comumLogin)
	body := ticketBody("outro", validOther())

	for i := 1; i <= security.TicketCreateLimit; i++ {
		if code := f.postAs(t, cookie, token, body); code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201", i, code)
		}
	}

	logBuf.Reset()
	rr := f.send(t, http.MethodPost, "/api/tickets", body, cookie, map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("request 11: status = %d, want 429 (%s)", rr.Code, rr.Body.String())
	}
	checkErrorBody(t, rr, "too_many_requests", "Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.")
	retry, err := strconv.Atoi(rr.Header().Get("Retry-After"))
	if err != nil || retry < 1 || retry > 60 {
		t.Errorf("Retry-After = %q, want whole seconds between 1 and 60", rr.Header().Get("Retry-After"))
	}
	if !strings.Contains(logBuf.String(), `"reason":"ratelimit: exceeded"`) {
		t.Errorf("log has no rate limit reason:\n%s", logBuf.String())
	}
	if n := f.tickets.count(); n != security.TicketCreateLimit {
		t.Errorf("%d tickets stored, want %d", n, security.TicketCreateLimit)
	}

	// The limit holds for the next request too, valid or not.
	if code := f.postAs(t, cookie, token, `{"tipo":"outro"}`); code != http.StatusTooManyRequests {
		t.Errorf("request 12 (malformed): status = %d, want 429", code)
	}
}

// TestTicketRateLimitIsPerUser shows that users do not share quotas: user 1
// (made comum) still creates tickets after user 2 used all of theirs.
func TestTicketRateLimitIsPerUser(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	f.sessions.setRole(1, domain.RoleComum)
	body := ticketBody("outro", validOther())

	cookie2, token2 := f.loginAs(t, comumLogin)
	for range security.TicketCreateLimit {
		f.postAs(t, cookie2, token2, body)
	}
	if code := f.postAs(t, cookie2, token2, body); code != http.StatusTooManyRequests {
		t.Fatalf("user 2, request 11: status = %d, want 429", code)
	}

	cookie1, token1 := f.loginAs(t, flowLogin)
	if code := f.postAs(t, cookie1, token1, body); code != http.StatusCreated {
		t.Errorf("user 1 after user 2 hit the limit: status = %d, want 201", code)
	}
}

// TestTicketRateLimitCountsOnlyWhatReachesIt shows where the limiter sits:
// after Session, CSRF and Authorize. Requests those refuse (no token, an
// admin, an anonymous session) use no quota; refused bodies (400, 422) that
// reach it do count.
func TestTicketRateLimitCountsOnlyWhatReachesIt(t *testing.T) {
	t.Parallel()

	t.Run("CSRF refusals and other callers use no quota", func(t *testing.T) {
		t.Parallel()
		f := newFlowApp(t)
		body := ticketBody("outro", validOther())

		anon, anonToken := f.csrf(t, nil)
		admin, adminToken := f.loginAs(t, flowLogin)
		cookie, token := f.loginAs(t, comumLogin)
		for range 20 {
			if code := f.send(t, http.MethodPost, "/api/tickets", body, cookie, nil).Code; code != http.StatusForbidden {
				t.Fatalf("no token: status = %d, want 403", code)
			}
			if code := f.postAs(t, admin, adminToken, body); code != http.StatusForbidden {
				t.Fatalf("admin: status = %d, want 403", code)
			}
			if code := f.postAs(t, anon, anonToken, body); code != http.StatusUnauthorized {
				t.Fatalf("anonymous: status = %d, want 401", code)
			}
		}
		for i := 1; i <= security.TicketCreateLimit; i++ {
			if code := f.postAs(t, cookie, token, body); code != http.StatusCreated {
				t.Fatalf("request %d after the refusals: status = %d, want 201", i, code)
			}
		}
	})

	t.Run("invalid bodies count", func(t *testing.T) {
		t.Parallel()
		f := newFlowApp(t)
		cookie, token := f.loginAs(t, comumLogin)
		if code := f.postAs(t, cookie, token, `{"tipo":"outro"}`); code != http.StatusBadRequest {
			t.Fatalf("malformed: status = %d, want 400", code)
		}
		if code := f.postAs(t, cookie, token, ticketBody("outro", withKey(validOther(), "titulo", ""))); code != http.StatusUnprocessableEntity {
			t.Fatalf("invalid: status = %d, want 422", code)
		}
		body := ticketBody("outro", validOther())
		for i := 3; i <= security.TicketCreateLimit; i++ {
			if code := f.postAs(t, cookie, token, body); code != http.StatusCreated {
				t.Fatalf("request %d: status = %d, want 201", i, code)
			}
		}
		if code := f.postAs(t, cookie, token, body); code != http.StatusTooManyRequests {
			t.Errorf("request 11: status = %d, want 429", code)
		}
		if n := f.tickets.count(); n != security.TicketCreateLimit-2 {
			t.Errorf("%d tickets stored, want %d", n, security.TicketCreateLimit-2)
		}
	})
}
