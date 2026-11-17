package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"backend_si/internal/domain"
)

type Auth struct {
	users     UserRepository
	passwords PasswordHasher
	validator InputValidator
	// dummyHash is verified when the login does not exist, so an unknown
	// user costs the same Argon2id work as a wrong password (D-13).
	dummyHash string
}

func NewAuth(ctx context.Context, users UserRepository, passwords PasswordHasher, validator InputValidator) (*Auth, error) {
	dummy, err := passwords.HashPassword(ctx, rand.Text())
	if err != nil {
		return nil, fmt.Errorf("service: compute dummy hash: %w", err)
	}
	return &Auth{users: users, passwords: passwords, validator: validator, dummyHash: dummy}, nil
}

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
