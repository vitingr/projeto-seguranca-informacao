package security

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"backend_si/internal/domain"
)

func newTestLimiter(t *testing.T, limit int) (*RateLimiter, *clock) {
	t.Helper()
	clk := newClock()
	s := newTestSecurityWith(t, func(o *Options) { o.Now = clk.Now })
	l, err := s.NewRateLimiter(limit, TicketCreateWindow)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}
	return l, clk
}

func (l *RateLimiter) tracked(user domain.UserID) (users, times int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.hits), len(l.hits[user])
}

func TestTicketCreateLimitIsTenPerMinute(t *testing.T) {
	t.Parallel()

	// S-12: the numbers are part of the spec, not a tuning knob.
	if TicketCreateLimit != 10 || TicketCreateWindow != time.Minute {
		t.Errorf("limit = %d per %v, want 10 per 1m", TicketCreateLimit, TicketCreateWindow)
	}
}

func TestNewRateLimiterRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	for _, tt := range []struct {
		limit  int
		window time.Duration
	}{{0, time.Minute}, {-1, time.Minute}, {10, 0}, {10, -time.Second}} {
		if _, err := s.NewRateLimiter(tt.limit, tt.window); !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("NewRateLimiter(%d, %v) error = %v, want ErrInvalidOptions", tt.limit, tt.window, err)
		}
	}
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	t.Parallel()

	type step struct {
		advance   time.Duration // clock move before the request
		user      domain.UserID
		wantOK    bool
		wantRetry time.Duration
	}
	burst := func(user domain.UserID, n int) []step {
		steps := make([]step, n)
		for i := range steps {
			steps[i] = step{user: user, wantOK: true}
		}
		return steps
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"10 at once, the 11th waits a whole window", append(burst(1, 10),
			step{user: 1, wantOK: false, wantRetry: time.Minute})},
		{"spread requests: the wait is until the oldest leaves", append([]step{
			{0, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
			{time.Second, 1, true, 0},
		}, step{time.Second, 1, false, 50 * time.Second})},
		// The burst shares one instant, so all of it leaves the window
		// together: ten more fit, and the next waits a whole window again.
		{"allowed again once the oldest leaves, not a moment before", append(append(burst(1, 10),
			step{59*time.Second + 999*time.Millisecond, 1, false, time.Millisecond},
			step{time.Millisecond, 1, true, 0}),
			append(burst(1, 9), step{0, 1, false, time.Minute})...)},
		{"refused requests are not counted", append(burst(1, 10),
			step{10 * time.Second, 1, false, 50 * time.Second},
			step{10 * time.Second, 1, false, 40 * time.Second},
			step{39 * time.Second, 1, false, time.Second},
			step{time.Second, 1, true, 0})},
		{"users do not share quotas", append(append(burst(1, 10), step{0, 1, false, time.Minute}),
			append(burst(2, 10), step{0, 2, false, time.Minute}, step{0, 1, false, time.Minute})...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, clk := newTestLimiter(t, TicketCreateLimit)
			for i, st := range tt.steps {
				clk.Advance(st.advance)
				ok, retry := l.Allow(st.user)
				if ok != st.wantOK || retry != st.wantRetry {
					t.Fatalf("request %d of user %d: Allow = %v, %v; want %v, %v", i+1, st.user, ok, retry, st.wantOK, st.wantRetry)
				}
			}
		})
	}
}

func TestRateLimiterKeepsAtMostLimitTimes(t *testing.T) {
	t.Parallel()

	l, clk := newTestLimiter(t, 3)
	for range 1000 {
		l.Allow(7)
		clk.Advance(5 * time.Second)
		if _, n := l.tracked(7); n > 3 {
			t.Fatalf("user keeps %d times, want at most 3", n)
		}
	}
}

func TestRateLimiterRetryAfterWithClockGoingBack(t *testing.T) {
	t.Parallel()

	l, clk := newTestLimiter(t, 1)
	l.Allow(1)
	clk.Advance(-time.Hour)
	if ok, retry := l.Allow(1); ok || retry != time.Minute {
		t.Errorf("Allow = %v, %v; want refused with a wait of at most one window", ok, retry)
	}
}

func TestRateLimiterPurge(t *testing.T) {
	t.Parallel()

	l, clk := newTestLimiter(t, TicketCreateLimit)
	l.Allow(1)
	clk.Advance(30 * time.Second)
	l.Allow(2)
	clk.Advance(30 * time.Second) // user 1 has left the window, user 2 has not

	if n := l.Purge(); n != 1 {
		t.Errorf("Purge = %d, want 1", n)
	}
	if users, n := l.tracked(2); users != 1 || n != 1 {
		t.Errorf("after purge: %d users, user 2 has %d times; want 1 and 1", users, n)
	}
	clk.Advance(30 * time.Second)
	if n := l.Purge(); n != 1 {
		t.Errorf("second Purge = %d, want 1", n)
	}
	if users, _ := l.tracked(2); users != 0 {
		t.Errorf("after second purge: %d users, want 0", users)
	}
}

func TestPurgeIdleRateLimitsRunsUntilCanceled(t *testing.T) {
	t.Parallel()

	clk := newClock()
	s := newTestSecurityWith(t, func(o *Options) { o.Now = clk.Now })
	l, err := s.NewRateLimiter(TicketCreateLimit, TicketCreateWindow)
	if err != nil {
		t.Fatal(err)
	}
	l.Allow(1)
	clk.Advance(time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		s.PurgeIdleRateLimits(ctx, l, time.Millisecond)
		close(done)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if users, _ := l.tracked(1); users == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("idle user was never purged")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("PurgeIdleRateLimits did not stop after cancel")
	}
}

func TestRateLimiterConcurrentCalls(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimiter(t, TicketCreateLimit)
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := range 200 {
		wg.Go(func() {
			if ok, _ := l.Allow(domain.UserID(i%2 + 1)); ok {
				allowed.Add(1)
			}
			l.Purge()
		})
	}
	wg.Wait()
	if got := allowed.Load(); got != 2*TicketCreateLimit {
		t.Errorf("allowed %d of 200 parallel requests by 2 users, want %d", got, 2*TicketCreateLimit)
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wait time.Duration
		want int
	}{
		{time.Nanosecond, 1},
		{time.Second, 1},
		{time.Second + time.Nanosecond, 2},
		{50 * time.Second, 50},
		{59*time.Second + time.Millisecond, 60},
		{time.Minute, 60},
	}
	for _, tt := range tests {
		if got := retryAfterSeconds(tt.wait); got != tt.want {
			t.Errorf("retryAfterSeconds(%v) = %d, want %d", tt.wait, got, tt.want)
		}
	}
}

// rateLimitFixture is the RateLimit middleware on a fake clock, with a
// captured log and a next handler that counts its calls.
type rateLimitFixture struct {
	sec     *Security
	limiter *RateLimiter
	clock   *clock
	log     *bytes.Buffer
	calls   int
	handler http.Handler
}

func newRateLimitFixture(t *testing.T) *rateLimitFixture {
	t.Helper()
	f := &rateLimitFixture{clock: newClock(), log: &bytes.Buffer{}}
	f.sec = newTestSecurityWith(t, func(o *Options) {
		o.Now = f.clock.Now
		o.Logger = slog.New(slog.NewJSONHandler(f.log, nil))
	})
	l, err := f.sec.NewRateLimiter(TicketCreateLimit, TicketCreateWindow)
	if err != nil {
		t.Fatal(err)
	}
	f.limiter = l
	f.handler = f.sec.RateLimit(l)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f.calls++
		w.WriteHeader(http.StatusCreated)
	}))
	return f
}

// send posts a body holding marker, as user (0 = no principal).
func (f *rateLimitFixture) send(t *testing.T, user domain.UserID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/tickets",
		strings.NewReader(`{"tipo":"outro","dados":{"titulo":"`+marker+`"}}`))
	if user != 0 {
		req = req.WithContext(context.WithValue(req.Context(), principalKey{}, Principal{UserID: user, Role: domain.RoleComum}))
	}
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	return rr
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("the 11th request of a user is 429 with Retry-After", func(t *testing.T) {
		t.Parallel()
		f := newRateLimitFixture(t)
		for i := range TicketCreateLimit {
			if rr := f.send(t, 1); rr.Code != http.StatusCreated {
				t.Fatalf("request %d: status %d, want 201", i+1, rr.Code)
			}
			f.clock.Advance(time.Second)
		}
		rr := f.send(t, 1)
		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("11th request: status %d, want 429", rr.Code)
		}
		if got := rr.Header().Get("Retry-After"); got != "50" {
			t.Errorf("Retry-After = %q, want %q (the oldest leaves in 50 s)", got, "50")
		}
		if f.calls != TicketCreateLimit {
			t.Errorf("next called %d times, want %d", f.calls, TicketCreateLimit)
		}
		if rr := f.send(t, 2); rr.Code != http.StatusCreated {
			t.Errorf("another user: status %d, want 201", rr.Code)
		}
		f.clock.Advance(50 * time.Second)
		if rr := f.send(t, 1); rr.Code != http.StatusCreated {
			t.Errorf("after Retry-After: status %d, want 201", rr.Code)
		}
	})

	t.Run("the refusal is logged with a fixed reason and no user input", func(t *testing.T) {
		t.Parallel()
		f := newRateLimitFixture(t)
		for range TicketCreateLimit + 1 {
			f.send(t, 1)
		}
		logged := f.log.String()
		if !strings.Contains(logged, `"reason":"ratelimit: exceeded"`) || !strings.Contains(logged, `"status":429`) {
			t.Errorf("log = %s, want the ratelimit reason and status 429", logged)
		}
		if strings.Contains(logged, marker) {
			t.Errorf("log holds the request body: %s", logged)
		}
	})

	t.Run("no principal fails closed with 401 and uses no quota", func(t *testing.T) {
		t.Parallel()
		f := newRateLimitFixture(t)
		if rr := f.send(t, 0); rr.Code != http.StatusUnauthorized {
			t.Errorf("status %d, want 401", rr.Code)
		}
		if f.calls != 0 {
			t.Error("next was called without a principal")
		}
		if users, _ := f.limiter.tracked(0); users != 0 {
			t.Errorf("limiter tracks %d users, want 0", users)
		}
	})

	t.Run("no limiter fails closed with 500", func(t *testing.T) {
		t.Parallel()
		f := newRateLimitFixture(t)
		f.handler = f.sec.RateLimit(nil)(f.handler)
		if rr := f.send(t, 1); rr.Code != http.StatusInternalServerError {
			t.Errorf("status %d, want 500", rr.Code)
		}
		if f.calls != 0 {
			t.Error("next was called without a limiter")
		}
	})
}
