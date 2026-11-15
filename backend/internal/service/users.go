package service

import (
	"context"
	"fmt"

	"backend_si/internal/domain"
)

// RegisterInput são os dados brutos do formulário de cadastro.
type RegisterInput struct {
	Login    string
	Password string
	Role     string
}

// Users é o caso de uso de gestão de usuários.
type Users struct {
	repo      UserRepository
	passwords PasswordHasher
	validator InputValidator
}

// NewUsers cria o serviço de usuários.
func NewUsers(repo UserRepository, passwords PasswordHasher, validator InputValidator) *Users {
	return &Users{repo: repo, passwords: passwords, validator: validator}
}

// Register valida a entrada, gera o hash da senha e cria o usuário.
// Retorna erros que encapsulam domain.ErrInvalidInput ou domain.ErrConflict.
func (u *Users) Register(ctx context.Context, in RegisterInput) (domain.User, error) {
	// TODO(etapa 2): validator.ValidateLogin/ValidatePassword/ParseRole,
	// passwords.HashPassword e repo.Create.
	return domain.User{}, fmt.Errorf("service: cadastrar usuário: %w", domain.ErrNotImplemented)
}

// CreateFirstAdmin cria o administrador inicial (comando seed). Deve falhar
// com domain.ErrConflict se o login já existir.
func (u *Users) CreateFirstAdmin(ctx context.Context, login, password string) (domain.User, error) {
	user, err := u.create(ctx, login, password, domain.RoleAdmin)
	if err != nil {
		return domain.User{}, fmt.Errorf("service: create first admin: %w", err)
	}
	return user, nil
}

// create normalizes and validates the login, validates and hashes the
// password, and stores the user (D-2). Validation errors wrap
// domain.ErrInvalidInput; a taken login wraps domain.ErrConflict. The
// returned user has no PasswordHash.
func (u *Users) create(ctx context.Context, login, password string, role domain.Role) (domain.User, error) {
	if !role.Valid() {
		return domain.User{}, fmt.Errorf("invalid role: %w", domain.ErrInvalidInput)
	}
	login = u.validator.NormalizeLogin(login)
	if err := u.validator.ValidateLogin(login); err != nil {
		return domain.User{}, err
	}
	if err := u.validator.ValidatePassword(password); err != nil {
		return domain.User{}, err
	}
	hash, err := u.passwords.HashPassword(ctx, password)
	if err != nil {
		return domain.User{}, err
	}
	id, err := u.repo.Create(ctx, domain.User{Login: login, PasswordHash: hash, Role: role})
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: id, Login: login, Role: role}, nil
}
