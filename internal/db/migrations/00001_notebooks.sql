CREATE TABLE IF NOT EXISTS notebooks (
    id       smallint PRIMARY KEY,
    slug     text     NOT NULL UNIQUE,
    name     text     NOT NULL,
    position smallint NOT NULL
);

INSERT INTO notebooks (id, slug, name, position) VALUES
    (1, 'my-journal', 'My Journal', 1),
    (2, 'i-learned',  'I Learned',  2)
ON CONFLICT (id) DO NOTHING;
