// Package service implementa os casos de uso da aplicação.
//
// As dependências são declaradas como interfaces aqui (no pacote
// consumidor) e satisfeitas pelos pacotes repository e security.
package service

import (
	"context"

	"backend_si/internal/domain"
)

// UserRepository é o acesso a dados de usuários exigido pelos serviços.
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.UserID, error)
	GetByLogin(ctx context.Context, login string) (domain.User, error)
}

// PasswordHasher gera e verifica hashes de senha.
//
// Both methods wait for a slot of the hash semaphore and return the context
// error if ctx ends first (D-19).
type PasswordHasher interface {
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, encodedHash, password string) error
}

// InputValidator valida dados de entrada antes de qualquer persistência.
type InputValidator interface {
	ValidateLogin(login string) error
	ValidatePassword(password string) error
	ParseRole(raw string) (domain.Role, error)
}
