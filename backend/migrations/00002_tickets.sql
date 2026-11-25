-- +goose Up
-- Support tickets (spec 001-ticket-builder). One typed column per field and
-- a CHECK per type, so the database refuses a ticket with the wrong set of
-- fields even if the application rules are bypassed. Lengths are counted in
-- characters (char_length), like the application rules.
CREATE TABLE tickets (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- No cascade: tickets are history, so a user who has tickets cannot be
    -- deleted by accident.
    author_id     BIGINT      NOT NULL REFERENCES users (id),
    type          TEXT        NOT NULL CHECK (type IN ('acesso', 'bug', 'deploy', 'outro')),
    status        TEXT        NOT NULL DEFAULT 'aberto'
                  CHECK (status IN ('aberto', 'em_andamento', 'resolvido', 'fechado')),
    title         TEXT        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    system        TEXT        CHECK (char_length(system) BETWEEN 2 AND 120),         -- acesso, bug
    access_level  TEXT        CHECK (access_level IN ('leitura', 'escrita', 'administrador')),
    justification TEXT        CHECK (char_length(justification) BETWEEN 10 AND 2000),
    description   TEXT        CHECK (char_length(description) BETWEEN 10 AND 2000),  -- bug, outro
    steps         TEXT        CHECK (char_length(steps) BETWEEN 10 AND 2000),
    severity      TEXT        CHECK (severity IN ('baixa', 'media', 'alta', 'critica')),
    project       TEXT        CHECK (char_length(project) BETWEEN 2 AND 120),
    repository    TEXT        CHECK (char_length(repository) BETWEEN 1 AND 300),
    environment   TEXT        CHECK (environment IN ('dev', 'homolog', 'prod')),
    notes         TEXT        CHECK (char_length(notes) BETWEEN 1 AND 2000),         -- NULL when empty
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tickets_fields_by_type CHECK (
        (type = 'acesso' AND num_nonnulls(system, access_level, justification) = 3
                         AND num_nonnulls(description, steps, severity, project, repository, environment, notes) = 0)
     OR (type = 'bug'    AND num_nonnulls(system, description, steps, severity) = 4
                         AND num_nonnulls(access_level, justification, project, repository, environment, notes) = 0)
     OR (type = 'deploy' AND num_nonnulls(project, repository, environment) = 3
                         AND num_nonnulls(system, access_level, justification, description, steps, severity) = 0)
     OR (type = 'outro'  AND num_nonnulls(description) = 1
                         AND num_nonnulls(system, access_level, justification, steps, severity, project, repository, environment, notes) = 0)
    )
);

-- "Meus tickets" (next spec) lists a user's tickets newest first; it also
-- serves the foreign key.
CREATE INDEX tickets_author_created_idx ON tickets (author_id, created_at DESC, id DESC);

-- +goose Down
DROP TABLE tickets;
