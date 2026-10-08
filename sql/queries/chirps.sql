-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: RemoveAllChirps :exec
DELETE FROM chirps;

-- name: ReturnAllChirps :many
SELECT *
FROM chirps
ORDER BY created_at ASC;

-- name: ReturnOneChirp :one
SELECT *
FROM chirps
WHERE id = $1;
