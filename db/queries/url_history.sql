-- name: LogURLHistory :one
INSERT INTO url_history (url_id, url_short, action, old_value, new_value, modified_at, modified_by)
VALUES ($1, $2, $3, $4, $5, NOW(), $6)
RETURNING id, url_id, url_short, action, old_value, new_value, modified_at, modified_by;