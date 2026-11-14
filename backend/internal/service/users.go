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
	// TODO(etapa 2): reutilizar Register com papel admin.
	return domain.User{}, fmt.Errorf("service: criar administrador inicial: %w", domain.ErrNotImplemented)
}
