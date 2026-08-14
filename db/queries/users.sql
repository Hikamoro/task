-- name: CreateUser :execlastid
INSERT INTO users (email, password_hash, name)
VALUES (?, ?, ?);

-- name: GetUserByID :one
SELECT id, email, password_hash, name, created_at
FROM users
WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, created_at
FROM users
WHERE email = ?;
