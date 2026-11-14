package service

import (
	"context"
	"fmt"

	"backend_si/internal/domain"
)

// Auth é o caso de uso de autenticação por login e senha.
type Auth struct {
	users     UserRepository
	passwords PasswordHasher
}

// NewAuth cria o serviço de autenticação.
func NewAuth(users UserRepository, passwords PasswordHasher) *Auth {
	return &Auth{users: users, passwords: passwords}
}

// Authenticate verifica as credenciais e retorna o usuário. Qualquer falha
// de credencial (usuário inexistente ou senha errada) retorna erro que
// encapsula domain.ErrInvalidCredentials, sem distinguir os casos.
func (a *Auth) Authenticate(ctx context.Context, login, password string) (domain.User, error) {
	// TODO(etapa 2): buscar por login; se não existir, verificar contra um
	// hash fictício (tempo constante, evita enumeração de usuários);
	// a.passwords.VerifyPassword.
	return domain.User{}, fmt.Errorf("service: autenticar: %w: %w", domain.ErrInvalidCredentials, domain.ErrNotImplemented)
}
