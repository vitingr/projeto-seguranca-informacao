-- +goose Up
-- Posts of the wall (spec 005). The length limits repeat the ones checked
-- by security.ValidatePost (D-4), counted in characters like the backend.
CREATE TABLE posts (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id  BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT posts_title_length CHECK (char_length(title) BETWEEN 1 AND 120),
    CONSTRAINT posts_body_length CHECK (char_length(body) BETWEEN 1 AND 2000)
);

-- The wall reads the newest posts first (D-5).
CREATE INDEX posts_created_at_id_idx ON posts (created_at DESC, id DESC);
CREATE INDEX posts_author_id_idx ON posts (author_id);

-- +goose Down
DROP TABLE posts;
