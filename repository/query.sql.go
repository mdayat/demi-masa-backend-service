// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const decrementCouponQuota = `-- name: DecrementCouponQuota :execrows
UPDATE coupon SET quota = quota - 1
WHERE code = $1 AND quota > 0 AND deleted_at IS NULL
`

func (q *Queries) DecrementCouponQuota(ctx context.Context, code string) (int64, error) {
	result, err := q.db.Exec(ctx, decrementCouponQuota, code)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const incrementCouponQuota = `-- name: IncrementCouponQuota :exec
UPDATE coupon SET quota = quota + 1 WHERE code = $1
`

func (q *Queries) IncrementCouponQuota(ctx context.Context, code string) error {
	_, err := q.db.Exec(ctx, incrementCouponQuota, code)
	return err
}

const insertInvoice = `-- name: InsertInvoice :exec
INSERT INTO invoice (id, user_id, ref_id, total_amount, qr_url, expires_at) VALUES ($1, $2, $3, $4, $5, $6)
`

type InsertInvoiceParams struct {
	ID          pgtype.UUID        `json:"id"`
	UserID      string             `json:"user_id"`
	RefID       string             `json:"ref_id"`
	TotalAmount int32              `json:"total_amount"`
	QrUrl       string             `json:"qr_url"`
	ExpiresAt   pgtype.Timestamptz `json:"expires_at"`
}

func (q *Queries) InsertInvoice(ctx context.Context, arg InsertInvoiceParams) error {
	_, err := q.db.Exec(ctx, insertInvoice,
		arg.ID,
		arg.UserID,
		arg.RefID,
		arg.TotalAmount,
		arg.QrUrl,
		arg.ExpiresAt,
	)
	return err
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

const selectActiveInvoice = `-- name: SelectActiveInvoice :one
SELECT id, user_id, ref_id, total_amount, status, qr_url, expires_at, created_at FROM invoice WHERE user_id = $1 AND expires_at > NOW()
`

func (q *Queries) SelectActiveInvoice(ctx context.Context, userID string) (Invoice, error) {
	row := q.db.QueryRow(ctx, selectActiveInvoice, userID)
	var i Invoice
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.RefID,
		&i.TotalAmount,
		&i.Status,
		&i.QrUrl,
		&i.ExpiresAt,
		&i.CreatedAt,
	)
	return i, err
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
