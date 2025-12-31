-- name: CreateURL :one
INSERT INTO urls (short_code, original_url) VALUES (?, ?) RETURNING *;

-- name: GetURL :one
SELECT * FROM urls WHERE short_code = ?;

-- name: UpdateAccessCount :exec
UPDATE urls SET access_count = access_count + 1 WHERE short_code = ?;

-- name: GetAllURLs :many
SELECT * FROM urls ORDER BY created_at DESC;
