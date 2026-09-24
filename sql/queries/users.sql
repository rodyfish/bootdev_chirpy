-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    $1,
    Now(),
    Now(),
    $2,
    $3
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: UpdateUserPassword :one
UPDATE users
SET 
    hashed_password = $2,
    updated_at = Now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserEmail :one
UPDATE users
SET 
    email = $2,
    updated_at = Now()
WHERE id = $1
RETURNING *;

-- name: UpdateToChirpyRed :one
UPDATE users
SET is_chirpy_red = true
WHERE id = $1
RETURNING *;


-- name: DeleteAllUsers :exec
DELETE FROM users;