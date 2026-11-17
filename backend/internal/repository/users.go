package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend_si/internal/domain"
)

type Users struct {
	db *sql.DB
}

func NewUsers(db *sql.DB) *Users {
	return &Users{db: db}
}

func (r *Users) Create(ctx context.Context, u domain.User) (domain.UserID, error) {
	const q = `INSERT INTO users (login, password_hash, role) VALUES ($1, $2, $3) RETURNING id`

	var id domain.UserID
	err := r.db.QueryRowContext(ctx, q, u.Login, u.PasswordHash, string(u.Role)).Scan(&id)
	switch {
	case isUniqueViolation(err):
		return 0, fmt.Errorf("repository: criar usuário: %w", domain.ErrConflict)
	case err != nil:
		return 0, fmt.Errorf("repository: criar usuário: %w", err)
	}
	return id, nil
}

func (r *Users) GetByLogin(ctx context.Context, login string) (domain.User, error) {
	const q = `SELECT id, login, password_hash, role, created_at FROM users WHERE login = $1`

	var u domain.User
	err := r.db.QueryRowContext(ctx, q, login).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.Role, &u.CreatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.User{}, fmt.Errorf("repository: buscar usuário: %w", domain.ErrNotFound)
	case err != nil:
		return domain.User{}, fmt.Errorf("repository: buscar usuário: %w", err)
	}
	return u, nil
}
