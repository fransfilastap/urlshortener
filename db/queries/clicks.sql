-- name: StoreClick :one
INSERT INTO clicks (url_id, url_short, ip, location, browser, device, timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, url_id, url_short, ip, location, browser, device, timestamp;

-- name: GetClicksByShort :many
SELECT id, url_id, url_short, ip, location, browser, device, timestamp
FROM clicks
WHERE url_short = $1
ORDER BY timestamp DESC;

-- name: HasRecentClick :one
SELECT EXISTS(
    SELECT 1 FROM clicks
    WHERE url_short = $1
    AND ip = $2
    AND browser = $3
    AND device = $4
    AND timestamp > NOW() - INTERVAL '1 hour'
);

-- name: GetTotalClicks :one
SELECT COUNT(*) FROM clicks WHERE url_short = $1;

-- name: GetClicksByBrowser :many
SELECT browser, COUNT(*) AS count FROM clicks
WHERE url_short = $1
GROUP BY browser;

-- name: GetClicksByDevice :many
SELECT device, COUNT(*) AS count FROM clicks
WHERE url_short = $1
GROUP BY device;

-- name: GetClicksByLocation :many
SELECT location, COUNT(*) AS count FROM clicks
WHERE url_short = $1
GROUP BY location;