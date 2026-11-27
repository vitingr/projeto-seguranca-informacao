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

const (
	TicketCreateLimit  = 10
	TicketCreateWindow = time.Minute
)

const (
	MessageCreateLimit  = 30
	MessageCreateWindow = time.Minute
	StatusChangeLimit   = 30
	StatusChangeWindow  = time.Minute
)

// RateLimiter is a per-user sliding-window limiter kept in process memory.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu   sync.Mutex
	hits map[domain.UserID][]time.Time
}

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

func (l *RateLimiter) Allow(user domain.UserID) (ok bool, retryAfter time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.inWindow(l.hits[user], now)
	if len(times) >= l.limit {
		l.hits[user] = times
		wait := times[0].Add(l.window).Sub(now)
		return false, min(max(wait, time.Nanosecond), l.window)
	}
	l.hits[user] = append(times, now)
	return true, 0
}

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

func retryAfterSeconds(d time.Duration) int {
	return max(1, int((d+time.Second-1)/time.Second))
}

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
