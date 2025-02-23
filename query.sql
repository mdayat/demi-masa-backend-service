-- name: CreateUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING *;

-- name: CheckUserExistence :one
SELECT EXISTS(SELECT 1 FROM "user" WHERE id = $1);