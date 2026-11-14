// Package repository implementa o acesso a dados no PostgreSQL.
//
// Regra: somente queries parametrizadas ($1, $2...). Nenhum valor vindo do
// usuário é concatenado em SQL.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	pingTimeout     = 5 * time.Second
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute

	// pgUniqueViolation é o SQLSTATE de violação de unicidade.
	pgUniqueViolation = "23505"
)

// PoolOptions dimensiona o pool de conexões.
type PoolOptions struct {
	MaxOpenConns int
	MaxIdleConns int
}

// Open cria o pool database/sql com o driver pgx e verifica a conexão com
// um ping. Erros não incluem a URL (que contém a senha).
func Open(ctx context.Context, databaseURL string, opts PoolOptions) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("repository: DATABASE_URL inválida")
	}

	db := stdlib.OpenDB(*cfg)
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("repository: ping no banco: %w", err)
	}
	return db, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
