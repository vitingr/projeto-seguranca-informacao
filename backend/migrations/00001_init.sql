-- +goose Up
CREATE TABLE users (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_login_key UNIQUE (login),
    CONSTRAINT users_role_check CHECK (role IN ('admin', 'comum'))
);

CREATE TABLE sessions (
    token_hash   BYTEA       PRIMARY KEY,
    user_id      BIGINT      REFERENCES users (id) ON DELETE CASCADE,
    csrf_token   TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    CONSTRAINT sessions_token_hash_len CHECK (octet_length(token_hash) = 32),
    CONSTRAINT sessions_expiry_order CHECK (expires_at > last_seen_at)
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;
