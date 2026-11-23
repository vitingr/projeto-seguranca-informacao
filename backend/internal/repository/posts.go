package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"backend_si/internal/domain"
)

const pgCheckViolation = "23514"

type Posts struct {
	db *sql.DB
}

func NewPosts(db *sql.DB) *Posts {
	return &Posts{db: db}
}

// postColumns are read from p (posts) joined with u (users), in the order
// of scanPost.
const postColumns = `p.id, p.author_id, u.login, p.title, p.body, p.created_at, p.updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPost(row rowScanner) (domain.Post, error) {
	var p domain.Post
	err := row.Scan(&p.ID, &p.AuthorID, &p.AuthorLogin, &p.Title, &p.Body, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Posts) List(ctx context.Context, limit, offset int) ([]domain.Post, error) {
	const q = `SELECT ` + postColumns + `
		FROM posts p JOIN users u ON u.id = p.author_id
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: list posts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	posts := []domain.Post{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: list posts: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list posts: %w", err)
	}
	return posts, nil
}

func (r *Posts) Create(ctx context.Context, p domain.Post) (domain.Post, error) {
	const q = `WITH p AS (
			INSERT INTO posts (author_id, title, body) VALUES ($1, $2, $3)
			RETURNING id, author_id, title, body, created_at, updated_at
		)
		SELECT ` + postColumns + ` FROM p JOIN users u ON u.id = p.author_id`

	post, err := scanPost(r.db.QueryRowContext(ctx, q, int64(p.AuthorID), p.Title, p.Body))
	if err != nil {
		return domain.Post{}, fmt.Errorf("repository: create post: %w", mapPostError(err))
	}
	return post, nil
}

// Update changes the title and body of post p.ID only if p.AuthorID wrote
// it: the owner check is part of the statement (005 D-3).
func (r *Posts) Update(ctx context.Context, p domain.Post) (domain.Post, error) {
	const q = `WITH p AS (
			UPDATE posts SET title = $3, body = $4, updated_at = now()
			WHERE id = $1 AND author_id = $2
			RETURNING id, author_id, title, body, created_at, updated_at
		)
		SELECT ` + postColumns + ` FROM p JOIN users u ON u.id = p.author_id`

	post, err := scanPost(r.db.QueryRowContext(ctx, q, int64(p.ID), int64(p.AuthorID), p.Title, p.Body))
	if err != nil {
		return domain.Post{}, fmt.Errorf("repository: update post: %w", mapPostError(err))
	}
	return post, nil
}

// Delete removes post id only if author wrote it (005 D-2, D-3).
func (r *Posts) Delete(ctx context.Context, id domain.PostID, author domain.UserID) error {
	const q = `DELETE FROM posts WHERE id = $1 AND author_id = $2`

	res, err := r.db.ExecContext(ctx, q, int64(id), int64(author))
	if err != nil {
		return fmt.Errorf("repository: delete post: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: delete post: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repository: delete post: %w", domain.ErrNotFound)
	}
	return nil
}

// mapPostError turns "no row" into domain.ErrNotFound and a CHECK
// violation (a length the validation should have caught) into
// domain.ErrInvalidInput.
func mapPostError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation:
		return domain.ErrInvalidInput
	default:
		return err
	}
}
