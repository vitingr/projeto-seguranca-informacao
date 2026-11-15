package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"backend_si/internal/domain"
)

// Auth é o caso de uso de autenticação por login e senha.
type Auth struct {
	users     UserRepository
	passwords PasswordHasher
	validator InputValidator
	// dummyHash is verified when the login does not exist, so an unknown
	// user costs the same Argon2id work as a wrong password (D-13).
	dummyHash string
}

// NewAuth cria o serviço de autenticação.
//
// It computes the dummy hash once, with the hasher's current parameters,
// from a random password nobody knows (D-13).
func NewAuth(ctx context.Context, users UserRepository, passwords PasswordHasher, validator InputValidator) (*Auth, error) {
	dummy, err := passwords.HashPassword(ctx, rand.Text())
	if err != nil {
		return nil, fmt.Errorf("service: compute dummy hash: %w", err)
	}
	return &Auth{users: users, passwords: passwords, validator: validator, dummyHash: dummy}, nil
}

// Authenticate verifica as credenciais e retorna o usuário. Qualquer falha
// de credencial (usuário inexistente ou senha errada) retorna erro que
// encapsula domain.ErrInvalidCredentials, sem distinguir os casos.
//
// The login is only normalized, never validated (D-12). Both credential
// failures run VerifyPassword exactly once (D-13). Any other failure (the
// repository, or the context ending while waiting for a hash slot) is
// returned without ErrInvalidCredentials, so it becomes a 500. The returned
// user has no PasswordHash.
func (a *Auth) Authenticate(ctx context.Context, login, password string) (domain.User, error) {
	u, err := a.users.GetByLogin(ctx, a.validator.NormalizeLogin(login))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		verr := a.passwords.VerifyPassword(ctx, a.dummyHash, password)
		if verr != nil && !errors.Is(verr, domain.ErrInvalidCredentials) {
			return domain.User{}, fmt.Errorf("service: authenticate: %w", verr)
		}
		return domain.User{}, fmt.Errorf("service: authenticate: %w", domain.ErrInvalidCredentials)
	case err != nil:
		return domain.User{}, fmt.Errorf("service: authenticate: %w", err)
	}

	if err := a.passwords.VerifyPassword(ctx, u.PasswordHash, password); err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return domain.User{}, fmt.Errorf("service: authenticate: %w", domain.ErrInvalidCredentials)
		}
		return domain.User{}, fmt.Errorf("service: authenticate: %w", err)
	}
	u.PasswordHash = ""
	return u, nil
}
