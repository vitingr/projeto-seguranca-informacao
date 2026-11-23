-- +goose Up
-- Audit log of security events (spec 006 D-4). A row has no free-text field,
-- so nothing the user typed (a login, a password) can end up in it. There
-- are no foreign keys: deleting a user never touches the log.
CREATE TABLE audit_events (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event         TEXT        NOT NULL,
    actor_user_id BIGINT,
    target_type   TEXT,
    target_id     BIGINT,
    ip            INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT audit_events_event_check CHECK (
        event IN ('login_succeeded', 'login_failed', 'logout', 'user_created', 'post_removed')
    ),
    CONSTRAINT audit_events_target_type_check CHECK (target_type IN ('user', 'post')),
    CONSTRAINT audit_events_target_all_or_none CHECK ((target_type IS NULL) = (target_id IS NULL))
);

-- The log is read newest first, optionally filtered by event (D-8).
CREATE INDEX audit_events_event_created_at_id_idx ON audit_events (event, created_at DESC, id DESC);
CREATE INDEX audit_events_created_at_id_idx ON audit_events (created_at DESC, id DESC);

-- Append-only (D-7): any UPDATE, DELETE or TRUNCATE fails, even one that
-- matches no row, because the trigger runs once per statement.
-- +goose StatementBegin
CREATE FUNCTION audit_events_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only: % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_events_append_only
    BEFORE UPDATE OR DELETE OR TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION audit_events_append_only();

-- +goose Down
DROP TABLE audit_events;
DROP FUNCTION audit_events_append_only();
