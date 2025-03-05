// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

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

const selectActiveSubscription = `-- name: SelectActiveSubscription :one
SELECT id, user_id, plan_id, payment_id, start_date, end_date FROM subscription WHERE user_id = $1 AND end_date > NOW()
`

func (q *Queries) SelectActiveSubscription(ctx context.Context, userID string) (Subscription, error) {
	row := q.db.QueryRow(ctx, selectActiveSubscription, userID)
	var i Subscription
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.PlanID,
		&i.PaymentID,
		&i.StartDate,
		&i.EndDate,
	)
	return i, err
}

const selectPrayers = `-- name: SelectPrayers :many
SELECT id, user_id, name, status, year, month, day FROM prayer WHERE user_id = $1 AND year = $2 AND month = $3 AND (day = $4 OR $4 IS NULL)
`

type SelectPrayersParams struct {
	UserID string      `json:"user_id"`
	Year   int16       `json:"year"`
	Month  int16       `json:"month"`
	Day    pgtype.Int2 `json:"day"`
}

func (q *Queries) SelectPrayers(ctx context.Context, arg SelectPrayersParams) ([]Prayer, error) {
	rows, err := q.db.Query(ctx, selectPrayers,
		arg.UserID,
		arg.Year,
		arg.Month,
		arg.Day,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Prayer
	for rows.Next() {
		var i Prayer
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.Name,
			&i.Status,
			&i.Year,
			&i.Month,
			&i.Day,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const selectRefreshTokenById = `-- name: SelectRefreshTokenById :one
SELECT id, user_id, revoked, expires_at FROM refresh_token WHERE id = $1 AND user_id = $2
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
		&i.ExpiresAt,
	)
	return i, err
}

const selectUserById = `-- name: SelectUserById :one
SELECT id, created_at, deleted_at FROM "user" WHERE id = $1 AND deleted_at IS NULL
`

func (q *Queries) SelectUserById(ctx context.Context, id string) (User, error) {
	row := q.db.QueryRow(ctx, selectUserById, id)
	var i User
	err := row.Scan(&i.ID, &i.CreatedAt, &i.DeletedAt)
	return i, err
}

const updatePrayerStatus = `-- name: UpdatePrayerStatus :exec
UPDATE prayer SET status = $2 WHERE id = $1
`

type UpdatePrayerStatusParams struct {
	ID     pgtype.UUID `json:"id"`
	Status string      `json:"status"`
}

func (q *Queries) UpdatePrayerStatus(ctx context.Context, arg UpdatePrayerStatusParams) error {
	_, err := q.db.Exec(ctx, updatePrayerStatus, arg.ID, arg.Status)
	return err
}
