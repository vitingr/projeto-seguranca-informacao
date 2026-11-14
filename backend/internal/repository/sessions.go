package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend_si/internal/domain"
)

// Sessions é o repositório da tabela sessions. Satisfaz security.SessionStore.
type Sessions struct {
	db *sql.DB
}

// NewSessions cria o repositório de sessões.
func NewSessions(db *sql.DB) *Sessions {
	return &Sessions{db: db}
}

// Create grava uma sessão. UserID zero é gravado como NULL (sessão anônima).
func (r *Sessions) Create(ctx context.Context, s domain.Session) error {
	const q = `INSERT INTO sessions (token_hash, user_id, csrf_token, created_at, last_seen_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	userID := sql.NullInt64{Int64: int64(s.UserID), Valid: s.UserID != 0}
	if _, err := r.db.ExecContext(ctx, q, s.TokenHash, userID, s.CSRFToken, s.CreatedAt, s.LastSeenAt, s.ExpiresAt); err != nil {
		return fmt.Errorf("repository: criar sessão: %w", err)
	}
	return nil
}

// Get busca a sessão pelo hash do token, trazendo o papel atual do usuário.
// A decisão sobre expiração cabe ao pacote security.
func (r *Sessions) Get(ctx context.Context, tokenHash []byte) (domain.Session, error) {
	const q = `SELECT s.token_hash, s.user_id, u.role, s.csrf_token, s.created_at, s.last_seen_at, s.expires_at
		FROM sessions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1`

	var (
		s      domain.Session
		userID sql.NullInt64
		role   sql.NullString
	)
	err := r.db.QueryRowContext(ctx, q, tokenHash).
		Scan(&s.TokenHash, &userID, &role, &s.CSRFToken, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.Session{}, fmt.Errorf("repository: buscar sessão: %w", domain.ErrNotFound)
	case err != nil:
		return domain.Session{}, fmt.Errorf("repository: buscar sessão: %w", err)
	}
	s.UserID = domain.UserID(userID.Int64)
	s.Role = domain.Role(role.String)
	return s, nil
}

// Touch atualiza a última atividade e a nova expiração da sessão.
func (r *Sessions) Touch(ctx context.Context, tokenHash []byte, lastSeenAt, expiresAt time.Time) error {
	const q = `UPDATE sessions SET last_seen_at = $2, expires_at = $3 WHERE token_hash = $1`

	res, err := r.db.ExecContext(ctx, q, tokenHash, lastSeenAt, expiresAt)
	if err != nil {
		return fmt.Errorf("repository: renovar sessão: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: renovar sessão: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repository: renovar sessão: %w", domain.ErrNotFound)
	}
	return nil
}

// Delete remove a sessão. Remover uma sessão inexistente não é erro.
func (r *Sessions) Delete(ctx context.Context, tokenHash []byte) error {
	const q = `DELETE FROM sessions WHERE token_hash = $1`

	if _, err := r.db.ExecContext(ctx, q, tokenHash); err != nil {
		return fmt.Errorf("repository: remover sessão: %w", err)
	}
	return nil
}

// DeleteExpired remove as sessões com expires_at <= now.
func (r *Sessions) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	const q = `DELETE FROM sessions WHERE expires_at <= $1`

	res, err := r.db.ExecContext(ctx, q, now)
	if err != nil {
		return 0, fmt.Errorf("repository: limpar sessões expiradas: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("repository: limpar sessões expiradas: %w", err)
	}
	return n, nil
}
