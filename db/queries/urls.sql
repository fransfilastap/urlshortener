-- name: CreateURL :one
INSERT INTO urls (original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at;

-- name: GetURLByShort :one
SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at
FROM urls
WHERE short = $1 AND deleted_at IS NULL;

-- name: GetURLByOriginal :one
SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at
FROM urls
WHERE original = $1 AND deleted_at IS NULL;

-- name: GetURLsByCreator :many
SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at
FROM urls
WHERE creator_reference = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: IncrementClicks :one
UPDATE urls SET clicks = clicks + 1
WHERE short = $1 AND deleted_at IS NULL
RETURNING clicks;

-- name: SoftDeleteURL :execrows
UPDATE urls SET deleted_at = NOW()
WHERE short = $1 AND deleted_at IS NULL;

-- name: SoftDeleteURLWithCreator :execrows
UPDATE urls SET deleted_at = NOW()
WHERE short = $1 AND creator_reference = $2 AND deleted_at IS NULL;

-- name: HardDeleteURL :execrows
DELETE FROM urls WHERE short = $1;

-- name: UpdateURL :one
UPDATE urls SET original = $1, title = $2, expires_at = $3
WHERE short = $4 AND deleted_at IS NULL
RETURNING id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at;

-- name: UpdateURLWithCreator :one
UPDATE urls SET original = $1, title = $2, expires_at = $3
WHERE short = $4 AND creator_reference = $5 AND deleted_at IS NULL
RETURNING id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at;