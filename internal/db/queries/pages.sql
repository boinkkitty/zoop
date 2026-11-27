-- name: GetWriter :one
SELECT plan, first_day, timezone, next_timezone, timezone_switch_day
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
