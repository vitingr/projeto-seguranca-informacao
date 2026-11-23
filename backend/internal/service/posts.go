package service

import (
	"context"
	"fmt"
	"net/netip"

	"backend_si/internal/domain"
)

// PostPageSize is the number of posts on each page of the wall (005 D-5).
const PostPageSize = 20

type PostInput struct {
	Title string
	Body  string
}

type Posts struct {
	repo      PostRepository
	validator PostValidator
}

func NewPosts(repo PostRepository, validator PostValidator) *Posts {
	return &Posts{repo: repo, validator: validator}
}

// List returns one page of the wall for viewer, newest first. An
// administrator sees every post, removed ones included (006 D-1); a comum
// user sees the posts that are not removed plus their own removed ones
// (006 D-2). It reads one post more than the page size to tell whether
// there is a next page, with no COUNT.
func (p *Posts) List(ctx context.Context, viewer domain.UserID, role domain.Role, page int) (domain.PostPage, error) {
	if page < 1 {
		return domain.PostPage{}, fmt.Errorf("service: list posts: page %d: %w", page, domain.ErrInvalidInput)
	}
	if !role.Valid() {
		return domain.PostPage{}, fmt.Errorf("service: list posts: role %q: %w", role, domain.ErrInvalidInput)
	}
	items, err := p.repo.List(ctx, viewer, role == domain.RoleAdmin, PostPageSize+1, (page-1)*PostPageSize)
	if err != nil {
		return domain.PostPage{}, fmt.Errorf("service: list posts: %w", err)
	}
	hasNext := len(items) > PostPageSize
	if hasNext {
		items = items[:PostPageSize]
	}
	return domain.PostPage{Items: items, Page: page, PageSize: PostPageSize, HasNext: hasNext}, nil
}

func (p *Posts) Create(ctx context.Context, author domain.UserID, in PostInput) (domain.Post, error) {
	title, body, err := p.validator.ValidatePost(in.Title, in.Body)
	if err != nil {
		return domain.Post{}, fmt.Errorf("service: create post: %w", err)
	}
	post, err := p.repo.Create(ctx, domain.Post{AuthorID: author, Title: title, Body: body})
	if err != nil {
		return domain.Post{}, fmt.Errorf("service: create post: %w", err)
	}
	return post, nil
}

// Update changes a post of author. Another author's post, or a missing one,
// wraps domain.ErrNotFound (D-3).
func (p *Posts) Update(ctx context.Context, author domain.UserID, id domain.PostID, in PostInput) (domain.Post, error) {
	title, body, err := p.validator.ValidatePost(in.Title, in.Body)
	if err != nil {
		return domain.Post{}, fmt.Errorf("service: update post: %w", err)
	}
	post, err := p.repo.Update(ctx, domain.Post{ID: id, AuthorID: author, Title: title, Body: body})
	if err != nil {
		return domain.Post{}, fmt.Errorf("service: update post: %w", err)
	}
	return post, nil
}

// Delete removes a post of author (D-2). Another author's post, or a missing
// one, wraps domain.ErrNotFound (D-3).
func (p *Posts) Delete(ctx context.Context, author domain.UserID, id domain.PostID) error {
	if err := p.repo.Delete(ctx, id, author); err != nil {
		return fmt.Errorf("service: delete post: %w", err)
	}
	return nil
}

// Remove removes post id as admin with reason (006 D-2) and records the
// post_removed event with the caller's ip, in one transaction (D-3). An
// invalid reason wraps domain.ErrInvalidInput; a missing post, or one
// already removed, wraps domain.ErrNotFound.
func (p *Posts) Remove(ctx context.Context, admin domain.UserID, id domain.PostID, reason string, ip netip.Addr) error {
	reason, err := p.validator.ValidateRemovalReason(reason)
	if err != nil {
		return fmt.Errorf("service: remove post: %w", err)
	}
	ev := domain.AuditEvent{
		Type:       domain.AuditPostRemoved,
		ActorID:    admin,
		TargetType: domain.AuditTargetPost,
		TargetID:   int64(id),
		IP:         ip,
	}
	if err := p.repo.Remove(ctx, id, admin, reason, ev); err != nil {
		return fmt.Errorf("service: remove post: %w", err)
	}
	return nil
}
