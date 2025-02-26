-- name: InsertUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING *;

-- name: SelectUserById :one
SELECT * FROM "user" WHERE id = $1;

-- name: CheckUserExistence :one
SELECT EXISTS(SELECT 1 FROM "user" WHERE id = $1);

-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (id, user_id, expires_at) VALUES ($1, $2, $3);