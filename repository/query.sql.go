// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const checkUserExistence = `-- name: CheckUserExistence :one
SELECT EXISTS(SELECT 1 FROM "user" WHERE id = $1)
`

func (q *Queries) CheckUserExistence(ctx context.Context, id string) (bool, error) {
	row := q.db.QueryRow(ctx, checkUserExistence, id)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const insertRefreshToken = `-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (id, user_id, expires_at) VALUES ($1, $2, $3)
`

type InsertRefreshTokenParams struct {
	ID        pgtype.UUID        `json:"id"`
	UserID    string             `json:"user_id"`
	ExpiresAt pgtype.Timestamptz `json:"expires_at"`
}

func (q *Queries) InsertRefreshToken(ctx context.Context, arg InsertRefreshTokenParams) error {
	_, err := q.db.Exec(ctx, insertRefreshToken, arg.ID, arg.UserID, arg.ExpiresAt)
	return err
}

const insertUser = `-- name: InsertUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING id, created_at, deleted_at
`

func (q *Queries) InsertUser(ctx context.Context, id string) (User, error) {
	row := q.db.QueryRow(ctx, insertUser, id)
	var i User
	err := row.Scan(&i.ID, &i.CreatedAt, &i.DeletedAt)
	return i, err
}

const revokeRefreshToken = `-- name: RevokeRefreshToken :exec
UPDATE refresh_token SET revoked = TRUE WHERE id = $1 AND user_id = $2
`

type RevokeRefreshTokenParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID string      `json:"user_id"`
}

func (q *Queries) RevokeRefreshToken(ctx context.Context, arg RevokeRefreshTokenParams) error {
	_, err := q.db.Exec(ctx, revokeRefreshToken, arg.ID, arg.UserID)
	return err
}

const selectRefreshTokenById = `-- name: SelectRefreshTokenById :one
SELECT id, user_id, revoked, created_at, expires_at FROM refresh_token WHERE id = $1 AND user_id = $2
`

type SelectRefreshTokenByIdParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID string      `json:"user_id"`
}

func (q *Queries) SelectRefreshTokenById(ctx context.Context, arg SelectRefreshTokenByIdParams) (RefreshToken, error) {
	row := q.db.QueryRow(ctx, selectRefreshTokenById, arg.ID, arg.UserID)
	var i RefreshToken
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Revoked,
		&i.CreatedAt,
		&i.ExpiresAt,
	)
	return i, err
}

const selectUserById = `-- name: SelectUserById :one
SELECT id, created_at, deleted_at FROM "user" WHERE id = $1
`

func (q *Queries) SelectUserById(ctx context.Context, id string) (User, error) {
	row := q.db.QueryRow(ctx, selectUserById, id)
	var i User
	err := row.Scan(&i.ID, &i.CreatedAt, &i.DeletedAt)
	return i, err
}
