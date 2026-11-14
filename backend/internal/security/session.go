package security

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"backend_si/internal/domain"
)

// SessionCookieName é o nome do cookie de sessão. O prefixo __Host- obriga o
// navegador a exigir Secure, Path=/ e ausência de Domain.
const SessionCookieName = "__Host-session"

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

// Session carrega a sessão do cookie, aplica a expiração por inatividade e
// disponibiliza o Principal e o token CSRF no contexto da requisição.
// Visitantes sem sessão válida recebem uma sessão anônima nova (necessária
// para o token CSRF da requisição de login).
func (s *Security) Session(next http.Handler) http.Handler {
	// TODO(etapa 2): ler o cookie, buscar por hash (SHA-256) no SessionStore,
	// rejeitar sessões expiradas (now >= ExpiresAt), renovar com Touch,
	// criar sessão anônima quando ausente e emitir o cookie com
	// HttpOnly, Secure, SameSite=Lax e Path=/ (o frontend fica na mesma
	// origem, via proxy do Next.js; o CSRF é coberto pelo token).
	//
	// Stub fail-closed: nenhuma sessão é carregada, portanto nenhuma
	// requisição é considerada autenticada.
	return next
}

// StartSession deve ser chamada após um login bem-sucedido: descarta a
// sessão anônima atual e cria uma nova com token e CSRF novos (regeneração
// de ID, contra fixação de sessão).
func (s *Security) StartSession(w http.ResponseWriter, r *http.Request, user domain.User) error {
	// TODO(etapa 2): implementar a regeneração de ID.
	return fmt.Errorf("security: iniciar sessão: %w", domain.ErrNotImplemented)
}

// EndSession encerra a sessão atual no servidor e expira o cookie.
func (s *Security) EndSession(w http.ResponseWriter, r *http.Request) error {
	// TODO(etapa 2): apagar do SessionStore e enviar cookie com MaxAge < 0.
	return fmt.Errorf("security: encerrar sessão: %w", domain.ErrNotImplemented)
}
