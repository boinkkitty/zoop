-- Search (#30): each Page is searchable by its text and its Day's English date words.
-- Up only; safe to apply twice.

-- Always English, since no TM prefix is used, so IMMUTABLE is safe; "Thursday 26 November 2026".
CREATE OR REPLACE FUNCTION zoop_day_words(d date) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    RETURN to_char(d, 'FMDay FMDD FMMonth YYYY');

-- 'simple' everywhere: literal words, no stemming or stop words, so prefix queries match word starts.
ALTER TABLE pages ADD COLUMN IF NOT EXISTS search tsvector
    GENERATED ALWAYS AS (to_tsvector('simple', zoop_day_words(day) || ' ' || plain_text)) STORED;

CREATE INDEX IF NOT EXISTS pages_search ON pages USING gin (search);
