-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    $1,
    Now(),
    Now(),
    $2,
    $3
)
RETURNING *;

-- name: GetChirp :one
SELECT *
FROM chirps
WHERE id = $1;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;

-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE id = $1;

-- name: GetAllChirps :many
SELECT *
FROM chirps
ORDER BY created_at ASC;

-- name: GetAllChirpsFromUser :many
SELECT *
FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;