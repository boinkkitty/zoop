-- name: GetWriter :one
SELECT plan, first_day, timezone
FROM users
WHERE id = @id AND deleted_at IS NULL;

-- name: GetPage :one
SELECT p.doc, p.schema_version, p.version, p.updated_at
FROM pages p
JOIN notebooks n ON n.id = p.notebook_id
WHERE p.user_id = @user_id AND n.slug = @notebook AND p.day = @day;

-- name: InsertPage :one
INSERT INTO pages (user_id, notebook_id, day, doc, schema_version, plain_text, version, created_at, updated_at)
SELECT @user_id, n.id, @day, @doc, @schema_version, @plain_text, 1, @now, @now
FROM notebooks n WHERE n.slug = @notebook
ON CONFLICT (user_id, notebook_id, day) DO NOTHING
RETURNING doc, schema_version, version, updated_at;

-- name: UpdatePage :one
UPDATE pages p
SET doc = @doc, schema_version = @schema_version, plain_text = @plain_text,
    version = p.version + 1, updated_at = @now
FROM notebooks n
WHERE n.id = p.notebook_id AND n.slug = @notebook
  AND p.user_id = @user_id AND p.day = @day AND p.version = @base_version
RETURNING p.doc, p.schema_version, p.version, p.updated_at;

-- name: ListPageTexts :many
-- The start of each written Page in from_day..to_day: enough for Titles and glimpses.
SELECT p.day, n.slug AS notebook, left(p.plain_text, 400)::text AS head
FROM pages p
JOIN notebooks n ON n.id = p.notebook_id
WHERE p.user_id = @user_id AND p.day BETWEEN @from_day AND @to_day AND p.plain_text <> ''
ORDER BY p.day DESC;

-- name: CountWrittenDaysByMonth :many
-- Written Days per month over the whole history, from the pages_written partial index.
SELECT date_trunc('month', p.day)::date AS month, count(DISTINCT p.day)::int AS written_days
FROM pages p
WHERE p.user_id = @user_id AND p.plain_text <> ''
GROUP BY 1
ORDER BY 1 DESC;
