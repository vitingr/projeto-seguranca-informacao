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

// SessionCookieName é o nome do cookie de sessão. O prefixo __Host- obriga o
// navegador a exigir Secure, Path=/ e ausência de Domain.
const SessionCookieName = "__Host-session"

// tokenLength is the length of rand.Text(): 26 base32 characters (128 bits).
const tokenLength = 26

// SessionStore persiste sessões no servidor. É implementada pelo pacote
// repository; as chaves são sempre o hash SHA-256 do token, nunca o token.
type SessionStore interface {
	// Create grava uma nova sessão.
	Create(ctx context.Context, s domain.Session) error
	// Get retorna a sessão pelo hash do token ou domain.ErrNotFound.
	Get(ctx context.Context, tokenHash []byte) (domain.Session, error)
	// Touch renova a atividade da sessão (expiração deslizante).
	Touch(ctx context.Context, tokenHash []byte, lastSeenAt, expiresAt time.Time) error
	// Delete remove a sessão (logout ou regeneração de ID).
	Delete(ctx context.Context, tokenHash []byte) error
	// DeleteExpired remove as sessões expiradas até now e retorna quantas foram removidas.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// Principal identifica o usuário autenticado de uma requisição.
type Principal struct {
	UserID domain.UserID
	Role   domain.Role
}

type principalKey struct{}

// PrincipalFrom retorna o usuário autenticado da requisição, se houver.
// Só o middleware Session coloca um Principal no contexto.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok && p.UserID != 0 && p.Role.Valid()
}

// sessionState is the session loaded for the current request (anonymous
// or authenticated). Only the Session middleware puts it in the context.
type sessionState struct {
	tokenHash []byte
	csrfToken string
}

type sessionStateKey struct{}

func sessionFrom(ctx context.Context) (sessionState, bool) {
	st, ok := ctx.Value(sessionStateKey{}).(sessionState)
	return st, ok && len(st.tokenHash) == sha256.Size
}

// Session carrega a sessão do cookie, aplica a expiração por inatividade e
// disponibiliza o Principal e o token CSRF no contexto da requisição.
//
// It only loads existing sessions: a request without a valid session
// continues as anonymous and nothing is written (D-8). Only CSRFToken
// creates an anonymous session. A session past its idle or absolute timeout
// (D-18) is deleted and the request is anonymous (D-9). A valid session is
// touched on every request (D-10). A store failure is a 500, never
// "continue as anonymous" (constitution §2).
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
			ctx = context.WithValue(ctx, principalKey{}, Principal{UserID: sess.UserID, Role: sess.Role})
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loadSession returns the valid session for tokenHash, touching it. ok is
// false when there is no valid session.
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
		// Deleted in between (a concurrent logout): anonymous.
		return domain.Session{}, false, nil
	case err != nil:
		return domain.Session{}, false, fmt.Errorf("security: touch session: %w", err)
	}
	sess.TokenHash = tokenHash
	return sess, true, nil
}

// expired reports whether sess is past its idle or absolute timeout (D-18)
// at now. ExpiresAt is checked too, so a stored expiry is never extended.
func (s *Security) expired(sess domain.Session, now time.Time) bool {
	return !now.Before(sess.LastSeenAt.Add(s.idleTimeout)) ||
		!now.Before(sess.CreatedAt.Add(s.absTimeout)) ||
		!now.Before(sess.ExpiresAt)
}

// expiresAt is min(now + idle, createdAt + absolute) (D-18).
func (s *Security) expiresAt(createdAt, now time.Time) time.Time {
	idle := now.Add(s.idleTimeout)
	if abs := createdAt.Add(s.absTimeout); abs.Before(idle) {
		return abs
	}
	return idle
}

// StartSession deve ser chamada após um login bem-sucedido: descarta a
// sessão anônima atual e cria uma nova com token e CSRF novos (regeneração
// de ID, contra fixação de sessão).
//
// The new session starts the absolute window (D-18). Clients must fetch
// GET /api/csrf again, because the CSRF token changes (D-4).
func (s *Security) StartSession(w http.ResponseWriter, r *http.Request, user domain.User) error {
	if user.ID == 0 || !user.Role.Valid() {
		return fmt.Errorf("security: start session: invalid user: %w", domain.ErrInvalidInput)
	}
	if err := s.deleteCurrent(r.Context()); err != nil {
		return fmt.Errorf("security: start session: %w", err)
	}
	if _, err := s.newSession(r.Context(), w, user.ID); err != nil {
		return fmt.Errorf("security: start session: %w", err)
	}
	return nil
}

// EndSession encerra a sessão atual no servidor e expira o cookie.
//
// The cookie is expired even when the request has no session.
func (s *Security) EndSession(w http.ResponseWriter, r *http.Request) error {
	if err := s.deleteCurrent(r.Context()); err != nil {
		return fmt.Errorf("security: end session: %w", err)
	}
	http.SetCookie(w, sessionCookie("", -1))
	return nil
}

// CSRFToken returns the CSRF token of the request's session, delivered to
// the frontend by GET /api/csrf. Without a session it creates an anonymous
// one and sets its cookie; this is the only place that creates anonymous
// sessions (D-8).
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

// newSession stores a new session for userID (zero = anonymous), sets its
// cookie and returns its CSRF token. Both tokens come from crypto/rand
// (D-11); only the SHA-256 of the session token is stored.
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

// deleteCurrent deletes the request's session, if any.
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

// sessionCookie builds the session cookie. It has no Max-Age or Expires
// (a browser-session cookie; the server enforces expiry, D-11); maxAge -1
// expires it.
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

// validToken reports whether v has the shape of a rand.Text() token, so a
// malformed cookie never reaches the store.
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

// PurgeExpiredSessions deletes expired sessions every interval until ctx
// ends (D-9). A failure is logged and the loop goes on.
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

// internalError logs err and answers the generic 500.
func (s *Security) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.ErrorContext(r.Context(), "security: internal error",
		slog.Any("error", err),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)
	s.renderError(w, r, http.StatusInternalServerError)
}
