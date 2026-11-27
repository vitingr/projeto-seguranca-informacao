-- +goose Up
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
    CONSTRAINT ticket_entries_allowed_transition CHECK (
        kind = 'message' OR (from_status, to_status) IN (
            ('aberto', 'em_andamento'), ('aberto', 'fechado'),
            ('em_andamento', 'resolvido'), ('em_andamento', 'fechado'),
            ('resolvido', 'em_andamento'), ('resolvido', 'fechado'))
    )
);

CREATE INDEX ticket_entries_ticket_idx ON ticket_entries (ticket_id, id);
CREATE INDEX ticket_entries_actor_idx  ON ticket_entries (actor_id);

DROP INDEX tickets_author_created_idx;
CREATE INDEX tickets_author_updated_idx ON tickets (author_id, updated_at DESC, id DESC);
CREATE INDEX tickets_updated_idx        ON tickets (updated_at DESC, id DESC);
CREATE INDEX tickets_status_updated_idx ON tickets (status, updated_at DESC, id DESC);

ALTER TABLE tickets ADD CONSTRAINT tickets_updated_not_before_created CHECK (updated_at >= created_at);

-- +goose Down
ALTER TABLE tickets DROP CONSTRAINT tickets_updated_not_before_created;
DROP INDEX tickets_status_updated_idx;
DROP INDEX tickets_updated_idx;
DROP INDEX tickets_author_updated_idx;
CREATE INDEX tickets_author_created_idx ON tickets (author_id, created_at DESC, id DESC);
DROP TABLE ticket_entries;
