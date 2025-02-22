-- name: CreateUser :exec
INSERT INTO "user" (id, first_name) VALUES ($1, $2);