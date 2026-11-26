package security

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"backend_si/internal/domain"
)

// Ticket creation limit (spec 001-ticket-builder, BR-9 and S-12): at most
// 10 requests per user in any 60-second window.
const (
	TicketCreateLimit  = 10
	TicketCreateWindow = time.Minute
)

// Message and status change limits (spec 003-ticket-tracking, S-9 and
// Q-9): at most 30 replies per user and 30 status changes per admin in any
// 60-second window, each with its own limiter.
const (
	MessageCreateLimit  = 30
	MessageCreateWindow = time.Minute
	StatusChangeLimit   = 30
	StatusChangeWindow  = time.Minute
)

// RateLimiter is a sliding-window limiter per user: it keeps the times of
// the last accepted requests of each user, at most limit of them, and
// accepts a new request only while fewer than limit fall inside the window
// ending now. Refused requests are not recorded, so a user who keeps trying
// is let in again as soon as the oldest request leaves the window.
//
// The counts live in the memory of the process: a restart resets them, and
// each backend instance counts on its own (plan P-3, "Known limitations").
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu   sync.Mutex
	hits map[domain.UserID][]time.Time
}

// NewRateLimiter returns a limiter of limit requests per window, on the
// clock of the Security (Options.Now).
func (s *Security) NewRateLimiter(limit int, window time.Duration) (*RateLimiter, error) {
	if limit <= 0 || window <= 0 {
		return nil, fmt.Errorf("%w: rate limit and window must be positive", ErrInvalidOptions)
	}
	return &RateLimiter{
		limit:  limit,
		window: window,
		now:    s.now,
		hits:   make(map[domain.UserID][]time.Time),
	}, nil
}

// Allow records a request of user and reports whether it is within the
// limit. When it is not, retryAfter is how long until the oldest recorded
// request leaves the window (always > 0 and at most the window).
func (l *RateLimiter) Allow(user domain.UserID) (ok bool, retryAfter time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.inWindow(l.hits[user], now)
	if len(times) >= l.limit {
		l.hits[user] = times
		wait := times[0].Add(l.window).Sub(now)
		// A clock that went back could make the wait longer than a window.
		return false, min(max(wait, time.Nanosecond), l.window)
	}
	l.hits[user] = append(times, now)
	return true, 0
}

// inWindow drops the times that left the window ending at now (a request
// at t counts while t + window > now). It copies when it drops, so the
// backing array never grows past limit entries.
func (l *RateLimiter) inWindow(times []time.Time, now time.Time) []time.Time {
	i := 0
	for i < len(times) && !times[i].Add(l.window).After(now) {
		i++
	}
	if i == 0 {
		return times
	}
	return append([]time.Time(nil), times[i:]...)
}

// Purge removes the users with no request inside the window and returns
// how many it removed, so idle users do not stay in memory.
func (l *RateLimiter) Purge() int {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	n := 0
	for user, times := range l.hits {
		if len(l.inWindow(times, now)) == 0 {
			delete(l.hits, user)
			n++
		}
	}
	return n
}

// RateLimit limits the requests of each authenticated user with l. It runs
// after Session, CSRF and Authorize (inside the route chain), so requests
// those refuse never use a user's quota; every request that reaches it
// counts, whatever its body. It fails closed: no principal → 401, no
// limiter → 500. Over the limit it answers 429 with Retry-After in whole
// seconds, rounded up, and the generic error body; the log reason is fixed
// and holds no user input.
func (s *Security) RateLimit(l *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l == nil {
				s.deny(w, r, http.StatusInternalServerError, "ratelimit: no limiter")
				return
			}
			principal, ok := PrincipalFrom(r.Context())
			if !ok {
				s.deny(w, r, http.StatusUnauthorized, "ratelimit: anonymous")
				return
			}
			allowed, retryAfter := l.Allow(principal.UserID)
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
				s.deny(w, r, http.StatusTooManyRequests, "ratelimit: exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// retryAfterSeconds rounds a wait up to whole seconds, at least 1, so a
// client that waits that long is let in.
func retryAfterSeconds(d time.Duration) int {
	return max(1, int((d+time.Second-1)/time.Second))
}

// PurgeIdleRateLimits runs l.Purge every interval until ctx is done, like
// PurgeExpiredSessions.
func (s *Security) PurgeIdleRateLimits(ctx context.Context, l *RateLimiter, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n := l.Purge(); n > 0 {
				s.logger.DebugContext(ctx, "security: purged idle rate limit entries", slog.Int("count", n))
			}
		}
	}
}
