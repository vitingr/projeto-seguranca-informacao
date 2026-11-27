package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"backend_si/internal/domain"
)

const SessionCookieName = "__Host-session"

// 26 base32 characters (128 bits), the length of rand.Text()
const tokenLength = 26

type SessionStore interface {
	Create(ctx context.Context, s domain.Session) error
	Get(ctx context.Context, tokenHash []byte) (domain.Session, error)
	Touch(ctx context.Context, tokenHash []byte, lastSeenAt, expiresAt time.Time) error
	Delete(ctx context.Context, tokenHash []byte) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type Principal struct {
	UserID    domain.UserID
	Role      domain.Role
	Login     string
	CreatedAt time.Time
}

type principalKey struct{}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok && p.UserID != 0 && p.Role.Valid()
}

type sessionState struct {
	tokenHash []byte
	csrfToken string
}

type sessionStateKey struct{}

func sessionFrom(ctx context.Context) (sessionState, bool) {
	st, ok := ctx.Value(sessionStateKey{}).(sessionState)
	return st, ok && len(st.tokenHash) == sha256.Size
}

func (s *Security) Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookieName)
		if err != nil || !validToken(c.Value) {
			next.ServeHTTP(w, r)
			return
		}

		sess, ok, err := s.loadSession(r.Context(), hashToken(c.Value))
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), sessionStateKey{}, sessionState{
			tokenHash: sess.TokenHash,
			csrfToken: sess.CSRFToken,
		})
		if sess.Authenticated() {
			ctx = context.WithValue(ctx, principalKey{}, Principal{
				UserID:    sess.UserID,
				Role:      sess.Role,
				Login:     sess.Login,
				CreatedAt: sess.UserCreatedAt,
			})
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Security) loadSession(ctx context.Context, tokenHash []byte) (domain.Session, bool, error) {
	sess, err := s.sessions.Get(ctx, tokenHash)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return domain.Session{}, false, nil
	case err != nil:
		return domain.Session{}, false, fmt.Errorf("security: load session: %w", err)
	}

	now := s.now()
	if s.expired(sess, now) {
		if delErr := s.sessions.Delete(ctx, tokenHash); delErr != nil {
			return domain.Session{}, false, fmt.Errorf("security: delete expired session: %w", delErr)
		}
		return domain.Session{}, false, nil
	}

	err = s.sessions.Touch(ctx, tokenHash, now, s.expiresAt(sess.CreatedAt, now))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return domain.Session{}, false, nil
	case err != nil:
		return domain.Session{}, false, fmt.Errorf("security: touch session: %w", err)
	}
	sess.TokenHash = tokenHash
	return sess, true, nil
}

func (s *Security) expired(sess domain.Session, now time.Time) bool {
	return !now.Before(sess.LastSeenAt.Add(s.idleTimeout)) ||
		!now.Before(sess.CreatedAt.Add(s.absTimeout)) ||
		!now.Before(sess.ExpiresAt)
}

func (s *Security) expiresAt(createdAt, now time.Time) time.Time {
	idle := now.Add(s.idleTimeout)
	if abs := createdAt.Add(s.absTimeout); abs.Before(idle) {
		return abs
	}
	return idle
}

func (s *Security) StartSession(w http.ResponseWriter, r *http.Request, user domain.User) error {
	if user.ID == 0 || !user.Role.Valid() {
		return errInvalidSessionUser
	}
	if err := s.deleteCurrent(r.Context()); err != nil {
		return fmt.Errorf("security: start session: %w", err)
	}
	if _, err := s.newSession(r.Context(), w, user.ID); err != nil {
		return fmt.Errorf("security: start session: %w", err)
	}
	return nil
}

func (s *Security) EndSession(w http.ResponseWriter, r *http.Request) error {
	if err := s.deleteCurrent(r.Context()); err != nil {
		return fmt.Errorf("security: end session: %w", err)
	}
	http.SetCookie(w, sessionCookie("", -1))
	return nil
}

func (s *Security) CSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	if st, ok := sessionFrom(r.Context()); ok {
		return st.csrfToken, nil
	}
	csrf, err := s.newSession(r.Context(), w, 0)
	if err != nil {
		return "", fmt.Errorf("security: csrf token: %w", err)
	}
	return csrf, nil
}

// Only the SHA-256 of the session token is stored.
func (s *Security) newSession(ctx context.Context, w http.ResponseWriter, userID domain.UserID) (string, error) {
	token, csrf := rand.Text(), rand.Text()
	now := s.now()
	err := s.sessions.Create(ctx, domain.Session{
		TokenHash:  hashToken(token),
		UserID:     userID,
		CSRFToken:  csrf,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  s.expiresAt(now, now),
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	http.SetCookie(w, sessionCookie(token, 0))
	return csrf, nil
}

func (s *Security) deleteCurrent(ctx context.Context) error {
	st, ok := sessionFrom(ctx)
	if !ok {
		return nil
	}
	if err := s.sessions.Delete(ctx, st.tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func sessionCookie(token string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func validToken(v string) bool {
	if len(v) != tokenLength {
		return false
	}
	for i := range len(v) {
		c := v[i]
		if (c < 'A' || c > 'Z') && (c < '2' || c > '7') {
			return false
		}
	}
	return true
}

func (s *Security) PurgeExpiredSessions(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := s.sessions.DeleteExpired(ctx, s.now())
			switch {
			case err != nil && ctx.Err() == nil:
				s.logger.ErrorContext(ctx, "security: purge expired sessions", slog.Any("error", err))
			case err == nil && n > 0:
				s.logger.InfoContext(ctx, "security: purged expired sessions", slog.Int64("count", n))
			}
		}
	}
}

func (s *Security) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.ErrorContext(r.Context(), "security: internal error",
		slog.Any("error", err),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)
	s.renderError(w, r, http.StatusInternalServerError)
}
