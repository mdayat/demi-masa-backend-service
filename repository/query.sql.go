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

const createRefreshToken = `-- name: CreateRefreshToken :exec
INSERT INTO refresh_token (id, user_id, expires_at) VALUES ($1, $2, $3)
`

type CreateRefreshTokenParams struct {
	ID        pgtype.UUID        `json:"id"`
	UserID    string             `json:"user_id"`
	ExpiresAt pgtype.Timestamptz `json:"expires_at"`
}

func (q *Queries) CreateRefreshToken(ctx context.Context, arg CreateRefreshTokenParams) error {
	_, err := q.db.Exec(ctx, createRefreshToken, arg.ID, arg.UserID, arg.ExpiresAt)
	return err
}

const createUser = `-- name: CreateUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING id, created_at, updated_at, deleted_at
`

func (q *Queries) CreateUser(ctx context.Context, id string) (User, error) {
	row := q.db.QueryRow(ctx, createUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}
