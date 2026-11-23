package domain

import "time"

type PostID int64

// Post is a post of the wall (spec 005). AuthorLogin is read from users when
// the post is loaded; it is not stored in posts. Removal is nil unless an
// administrator removed the post (006 D-2).
type Post struct {
	ID          PostID
	AuthorID    UserID
	AuthorLogin string
	Title       string
	Body        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Removal     *PostRemoval
}

// PostRemoval is the moderation of a post (006 D-2): who removed it, when
// and why. A removed post is frozen.
type PostRemoval struct {
	Reason    string
	RemovedAt time.Time
	RemovedBy UserID
}

// PostPage is one page of the wall, newest first (D-5).
type PostPage struct {
	Items    []Post
	Page     int
	PageSize int
	HasNext  bool
}
