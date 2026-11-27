-- Pages: one Notebook's writing for one Day (#21). Up only; safe to apply twice.
CREATE TABLE IF NOT EXISTS pages (
    id             uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id        uuid        NOT NULL REFERENCES users,
    notebook_id    smallint    NOT NULL REFERENCES notebooks,
    day            date        NOT NULL,
    doc            jsonb       NOT NULL,
    schema_version smallint    NOT NULL,
    plain_text     text        NOT NULL,
    version        int         NOT NULL CHECK (version > 0),
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    UNIQUE (user_id, notebook_id, day)
);

-- Written Days: Pages with some text.
CREATE INDEX IF NOT EXISTS pages_written ON pages (user_id, day) WHERE plain_text <> '';
