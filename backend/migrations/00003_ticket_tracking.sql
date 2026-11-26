-- +goose Up
-- Timeline of a ticket (spec 003-ticket-tracking): messages and status
-- changes in one table, never edited or deleted (BR-4). No cascade: entries
-- are history, so a ticket or a user with entries cannot be deleted by
-- accident. Lengths are counted in characters (char_length), like the
-- application rules.
CREATE TABLE ticket_entries (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ticket_id   BIGINT      NOT NULL REFERENCES tickets (id),
    actor_id    BIGINT      NOT NULL REFERENCES users (id),
    kind        TEXT        NOT NULL CHECK (kind IN ('message', 'status_change')),
    body        TEXT        CHECK (char_length(body) BETWEEN 1 AND 2000),
    from_status TEXT        CHECK (from_status IN ('aberto', 'em_andamento', 'resolvido', 'fechado')),
    to_status   TEXT        CHECK (to_status   IN ('aberto', 'em_andamento', 'resolvido', 'fechado')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ticket_entries_fields_by_kind CHECK (
        (kind = 'message'       AND body IS NOT NULL AND num_nonnulls(from_status, to_status) = 0)
     OR (kind = 'status_change' AND body IS NULL     AND num_nonnulls(from_status, to_status) = 2)
    ),
    -- BR-3, a second net under security.CheckStatusChange (plan P-15).
    CONSTRAINT ticket_entries_allowed_transition CHECK (
        kind = 'message' OR (from_status, to_status) IN (
            ('aberto', 'em_andamento'), ('aberto', 'fechado'),
            ('em_andamento', 'resolvido'), ('em_andamento', 'fechado'),
            ('resolvido', 'em_andamento'), ('resolvido', 'fechado'))
    )
);

-- The thread, oldest first, and its "older than" pages (plan P-4).
CREATE INDEX ticket_entries_ticket_idx ON ticket_entries (ticket_id, id);
-- The foreign key to users.
CREATE INDEX ticket_entries_actor_idx  ON ticket_entries (actor_id);

-- Both lists are ordered by the last update, most recent first (Q-8, BR-8).
DROP INDEX tickets_author_created_idx;
-- "Meus tickets", the foreign key and the author filter of the queue.
CREATE INDEX tickets_author_updated_idx ON tickets (author_id, updated_at DESC, id DESC);
-- The queue with no filter or by type only.
CREATE INDEX tickets_updated_idx        ON tickets (updated_at DESC, id DESC);
-- The queue by status and the status counts.
CREATE INDEX tickets_status_updated_idx ON tickets (status, updated_at DESC, id DESC);

ALTER TABLE tickets ADD CONSTRAINT tickets_updated_not_before_created CHECK (updated_at >= created_at);

-- +goose Down
-- For development only: the entries are lost.
ALTER TABLE tickets DROP CONSTRAINT tickets_updated_not_before_created;
DROP INDEX tickets_status_updated_idx;
DROP INDEX tickets_updated_idx;
DROP INDEX tickets_author_updated_idx;
CREATE INDEX tickets_author_created_idx ON tickets (author_id, created_at DESC, id DESC);
DROP TABLE ticket_entries;
