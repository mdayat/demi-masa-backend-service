-- name: InsertUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING *;

-- name: SelectUserById :one
SELECT * FROM "user" WHERE id = $1;

-- name: CheckUserExistence :one
SELECT EXISTS(SELECT 1 FROM "user" WHERE id = $1);

-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (id, user_id, expires_at) VALUES ($1, $2, $3);

-- name: SelectRefreshTokenById :one
SELECT * FROM refresh_token WHERE id = $1 AND user_id = $2;

-- name: RevokeRefreshToken :exec
UPDATE refresh_token SET revoked = TRUE WHERE id = $1 AND user_id = $2;

-- name: SelectPrayers :many
SELECT * FROM prayer WHERE user_id = $1 AND year = $2 AND month = $3 AND (day = sqlc.narg('day') OR sqlc.narg('day') IS NULL);