-- name: SearchPages :many
-- One page of matching Pages, newest Day first then by Notebook, after the cursor if any.
-- ts_headline runs only on the rows that page returns. @query is a to_tsquery('simple', …) string.
SELECT hit.day, hit.notebook, hit.position, left(hit.plain_text, 400)::text AS head,
       ts_headline('simple', hit.plain_text, to_tsquery('simple', @query), @options)::text AS headline
FROM (
    SELECT p.day, n.slug AS notebook, n.position, p.plain_text
    FROM pages p
    JOIN notebooks n ON n.id = p.notebook_id
    WHERE p.user_id = @user_id
      AND p.search @@ to_tsquery('simple', @query)
      AND (sqlc.narg(after_day)::date IS NULL
           OR p.day < sqlc.narg(after_day)::date
           OR (p.day = sqlc.narg(after_day)::date AND n.position > sqlc.narg(after_position)::smallint))
    ORDER BY p.day DESC, n.position ASC
    LIMIT @max_results
) hit
ORDER BY hit.day DESC, hit.position ASC;
