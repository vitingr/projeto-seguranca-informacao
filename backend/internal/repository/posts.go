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
const postColumns = `p.id, p.author_id, u.login, p.title, p.body, p.created_at, p.updated_at,
	p.removed_at, p.removed_by, p.removal_reason`

// postReturning are the posts columns a CTE returns for postColumns.
const postReturning = `id, author_id, title, body, created_at, updated_at, removed_at, removed_by, removal_reason`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPost(row rowScanner) (domain.Post, error) {
	var (
		p         domain.Post
		removedAt sql.NullTime
		removedBy sql.NullInt64
		reason    sql.NullString
	)
	err := row.Scan(&p.ID, &p.AuthorID, &p.AuthorLogin, &p.Title, &p.Body, &p.CreatedAt, &p.UpdatedAt,
		&removedAt, &removedBy, &reason)
	if err == nil && removedAt.Valid {
		p.Removal = &domain.PostRemoval{
			Reason:    reason.String,
			RemovedAt: removedAt.Time,
			RemovedBy: domain.UserID(removedBy.Int64),
		}
	}
	return p, err
}

// List returns one slice of the wall, newest first. With all set (an
// administrator, 006 D-1) it returns every post, removed ones included;
// otherwise the posts that are not removed plus viewer's own removed posts
// (006 D-2).
func (r *Posts) List(ctx context.Context, viewer domain.UserID, all bool, limit, offset int) ([]domain.Post, error) {
	const q = `SELECT ` + postColumns + `
		FROM posts p JOIN users u ON u.id = p.author_id
		WHERE $1 OR p.removed_at IS NULL OR p.author_id = $2
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, q, all, int64(viewer), limit, offset)
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
			RETURNING ` + postReturning + `
		)
		SELECT ` + postColumns + ` FROM p JOIN users u ON u.id = p.author_id`

	post, err := scanPost(r.db.QueryRowContext(ctx, q, int64(p.AuthorID), p.Title, p.Body))
	if err != nil {
		return domain.Post{}, fmt.Errorf("repository: create post: %w", mapPostError(err))
	}
	return post, nil
}

// Update changes the title and body of post p.ID only if p.AuthorID wrote
// it and it was not removed: the owner check (005 D-3) and the freeze of a
// removed post (006 D-2) are part of the statement.
func (r *Posts) Update(ctx context.Context, p domain.Post) (domain.Post, error) {
	const q = `WITH p AS (
			UPDATE posts SET title = $3, body = $4, updated_at = now()
			WHERE id = $1 AND author_id = $2 AND removed_at IS NULL
			RETURNING ` + postReturning + `
		)
		SELECT ` + postColumns + ` FROM p JOIN users u ON u.id = p.author_id`

	post, err := scanPost(r.db.QueryRowContext(ctx, q, int64(p.ID), int64(p.AuthorID), p.Title, p.Body))
	if err != nil {
		return domain.Post{}, fmt.Errorf("repository: update post: %w", mapPostError(err))
	}
	return post, nil
}

// Delete removes post id only if author wrote it (005 D-2, D-3) and the
// moderation did not remove it (006 D-2).
func (r *Posts) Delete(ctx context.Context, id domain.PostID, author domain.UserID) error {
	const q = `DELETE FROM posts WHERE id = $1 AND author_id = $2 AND removed_at IS NULL`

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

// Remove marks post id as removed by admin with reason and appends ev to
// the audit log, in one transaction (006 D-3): the removal is never stored
// without its event. A missing post, or one already removed, wraps
// domain.ErrNotFound and writes nothing.
func (r *Posts) Remove(ctx context.Context, id domain.PostID, admin domain.UserID, reason string, ev domain.AuditEvent) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: remove post: begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const q = `UPDATE posts SET removed_at = now(), removed_by = $2, removal_reason = $3
		WHERE id = $1 AND removed_at IS NULL`
	res, err := tx.ExecContext(ctx, q, int64(id), int64(admin), reason)
	if err != nil {
		return fmt.Errorf("repository: remove post: %w", mapPostError(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: remove post: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repository: remove post: %w", domain.ErrNotFound)
	}
	if err = appendAuditEvent(ctx, tx, ev); err != nil {
		return fmt.Errorf("repository: remove post: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("repository: remove post: commit: %w", err)
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
