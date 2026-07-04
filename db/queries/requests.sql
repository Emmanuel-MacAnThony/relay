-- name: SaveRequest :exec
INSERT INTO requests (id, slug, method, path, headers, query_params, body, source_ip, received_at, body_size)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: SaveDeliveryAttempt :exec
INSERT INTO delivery_attempts (id, request_id, delivered, status_code, error, is_replay, attempted_at, response_body, response_headers)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetRequest :one
SELECT id, slug, method, path, headers, query_params, body, source_ip, received_at, body_size
FROM requests
WHERE id = $1;

-- name: ListRequests :many
SELECT id, slug, method, path, headers, query_params, body, source_ip, received_at, body_size
FROM requests
WHERE slug = $1
ORDER BY received_at DESC;

-- name: ListLatestAttemptsBySlug :many
SELECT DISTINCT ON (da.request_id)
    da.id, da.request_id, da.delivered, da.status_code, da.error, da.is_replay, da.attempted_at,
    da.response_body, da.response_headers
FROM delivery_attempts da
JOIN requests r ON da.request_id = r.id
WHERE r.slug = $1
ORDER BY da.request_id, da.attempted_at DESC;
