-- name: SaveEndpoint :exec
INSERT INTO endpoints (slug, created_at)
VALUES ($1, $2);

-- name: GetEndpoint :one
SELECT slug, created_at
FROM endpoints
WHERE slug = $1;
