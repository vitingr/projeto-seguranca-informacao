-- +goose Up
-- Moderation of the wall (spec 006 D-2): a removal is a soft delete that
-- keeps the post, who removed it, when and why. The three columns are null
-- together (not removed) or set together (removed). The reason limit
-- repeats security.ValidateRemovalReason, counted in characters.
--
-- removed_by has no ON DELETE action: an administrator who removed posts
-- cannot be deleted while those posts exist, so the evidence stays whole.
ALTER TABLE posts
    ADD COLUMN removed_at     TIMESTAMPTZ,
    ADD COLUMN removed_by     BIGINT REFERENCES users (id),
    ADD COLUMN removal_reason TEXT,
    ADD CONSTRAINT posts_removal_all_or_none CHECK (
        (removed_at IS NULL AND removed_by IS NULL AND removal_reason IS NULL)
        OR (removed_at IS NOT NULL AND removed_by IS NOT NULL AND removal_reason IS NOT NULL)
    ),
    ADD CONSTRAINT posts_removal_reason_length CHECK (char_length(removal_reason) BETWEEN 1 AND 500);

-- +goose Down
ALTER TABLE posts
    DROP CONSTRAINT posts_removal_reason_length,
    DROP CONSTRAINT posts_removal_all_or_none,
    DROP COLUMN removal_reason,
    DROP COLUMN removed_by,
    DROP COLUMN removed_at;
