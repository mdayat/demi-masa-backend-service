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

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM "user" WHERE id = $1
`

func (q *Queries) DeleteUser(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteUser, id)
	return err
}

const deleteUserTask = `-- name: DeleteUserTask :execrows
DELETE FROM task WHERE id = $1 AND user_id = $2
`

type DeleteUserTaskParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID pgtype.UUID `json:"user_id"`
}

func (q *Queries) DeleteUserTask(ctx context.Context, arg DeleteUserTaskParams) (int64, error) {
	result, err := q.db.Exec(ctx, deleteUserTask, arg.ID, arg.UserID)
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

const insertCoupon = `-- name: InsertCoupon :one
INSERT INTO coupon (code, influencer_username, quota)
VALUES ($1, $2, $3) RETURNING code, influencer_username, quota, created_at, deleted_at
`

type InsertCouponParams struct {
	Code               string `json:"code"`
	InfluencerUsername string `json:"influencer_username"`
	Quota              int16  `json:"quota"`
}

func (q *Queries) InsertCoupon(ctx context.Context, arg InsertCouponParams) (Coupon, error) {
	row := q.db.QueryRow(ctx, insertCoupon, arg.Code, arg.InfluencerUsername, arg.Quota)
	var i Coupon
	err := row.Scan(
		&i.Code,
		&i.InfluencerUsername,
		&i.Quota,
		&i.CreatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const insertPlan = `-- name: InsertPlan :one
INSERT INTO plan (id, type, name, price, duration_in_months)
VALUES ($1, $2, $3, $4, $5) RETURNING id, type, name, price, duration_in_months, created_at, deleted_at
`

type InsertPlanParams struct {
	ID               pgtype.UUID `json:"id"`
	Type             string      `json:"type"`
	Name             string      `json:"name"`
	Price            int32       `json:"price"`
	DurationInMonths int16       `json:"duration_in_months"`
}

func (q *Queries) InsertPlan(ctx context.Context, arg InsertPlanParams) (Plan, error) {
	row := q.db.QueryRow(ctx, insertPlan,
		arg.ID,
		arg.Type,
		arg.Name,
		arg.Price,
		arg.DurationInMonths,
	)
	var i Plan
	err := row.Scan(
		&i.ID,
		&i.Type,
		&i.Name,
		&i.Price,
		&i.DurationInMonths,
		&i.CreatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const insertUser = `-- name: InsertUser :one
INSERT INTO "user" (id, email, password, name, coordinates, city, timezone)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, email, password, name, coordinates, city, timezone, created_at
`

type InsertUserParams struct {
	ID          pgtype.UUID  `json:"id"`
	Email       string       `json:"email"`
	Password    string       `json:"password"`
	Name        string       `json:"name"`
	Coordinates pgtype.Point `json:"coordinates"`
	City        string       `json:"city"`
	Timezone    string       `json:"timezone"`
}

func (q *Queries) InsertUser(ctx context.Context, arg InsertUserParams) (User, error) {
	row := q.db.QueryRow(ctx, insertUser,
		arg.ID,
		arg.Email,
		arg.Password,
		arg.Name,
		arg.Coordinates,
		arg.City,
		arg.Timezone,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.Name,
		&i.Coordinates,
		&i.City,
		&i.Timezone,
		&i.CreatedAt,
	)
	return i, err
}

const insertUserInvoice = `-- name: InsertUserInvoice :one
INSERT INTO invoice (id, user_id, plan_id, ref_id, coupon_code, total_amount, qr_url, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, user_id, plan_id, ref_id, coupon_code, total_amount, qr_url, expires_at, created_at
`

type InsertUserInvoiceParams struct {
	ID          pgtype.UUID        `json:"id"`
	UserID      pgtype.UUID        `json:"user_id"`
	PlanID      pgtype.UUID        `json:"plan_id"`
	RefID       string             `json:"ref_id"`
	CouponCode  pgtype.Text        `json:"coupon_code"`
	TotalAmount int32              `json:"total_amount"`
	QrUrl       string             `json:"qr_url"`
	ExpiresAt   pgtype.Timestamptz `json:"expires_at"`
}

func (q *Queries) InsertUserInvoice(ctx context.Context, arg InsertUserInvoiceParams) (Invoice, error) {
	row := q.db.QueryRow(ctx, insertUserInvoice,
		arg.ID,
		arg.UserID,
		arg.PlanID,
		arg.RefID,
		arg.CouponCode,
		arg.TotalAmount,
		arg.QrUrl,
		arg.ExpiresAt,
	)
	var i Invoice
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.PlanID,
		&i.RefID,
		&i.CouponCode,
		&i.TotalAmount,
		&i.QrUrl,
		&i.ExpiresAt,
		&i.CreatedAt,
	)
	return i, err
}

const insertUserPayment = `-- name: InsertUserPayment :one
INSERT INTO payment (id, user_id, invoice_id, amount_paid, status)
VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, invoice_id, amount_paid, status, created_at
`

type InsertUserPaymentParams struct {
	ID         pgtype.UUID `json:"id"`
	UserID     pgtype.UUID `json:"user_id"`
	InvoiceID  pgtype.UUID `json:"invoice_id"`
	AmountPaid int32       `json:"amount_paid"`
	Status     string      `json:"status"`
}

func (q *Queries) InsertUserPayment(ctx context.Context, arg InsertUserPaymentParams) (Payment, error) {
	row := q.db.QueryRow(ctx, insertUserPayment,
		arg.ID,
		arg.UserID,
		arg.InvoiceID,
		arg.AmountPaid,
		arg.Status,
	)
	var i Payment
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.InvoiceID,
		&i.AmountPaid,
		&i.Status,
		&i.CreatedAt,
	)
	return i, err
}

type InsertUserPrayersParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID pgtype.UUID `json:"user_id"`
	Name   string      `json:"name"`
	Year   int16       `json:"year"`
	Month  int16       `json:"month"`
	Day    int16       `json:"day"`
}

const insertUserRefreshToken = `-- name: InsertUserRefreshToken :one
INSERT INTO refresh_token (id, user_id, expires_at)
VALUES ($1, $2, $3) RETURNING id, user_id, revoked, expires_at
`

type InsertUserRefreshTokenParams struct {
	ID        pgtype.UUID        `json:"id"`
	UserID    pgtype.UUID        `json:"user_id"`
	ExpiresAt pgtype.Timestamptz `json:"expires_at"`
}

func (q *Queries) InsertUserRefreshToken(ctx context.Context, arg InsertUserRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRow(ctx, insertUserRefreshToken, arg.ID, arg.UserID, arg.ExpiresAt)
	var i RefreshToken
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Revoked,
		&i.ExpiresAt,
	)
	return i, err
}

const insertUserSubscription = `-- name: InsertUserSubscription :one
INSERT INTO subscription (id, user_id, plan_id, payment_id, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, plan_id, payment_id, start_date, end_date
`

type InsertUserSubscriptionParams struct {
	ID        pgtype.UUID        `json:"id"`
	UserID    pgtype.UUID        `json:"user_id"`
	PlanID    pgtype.UUID        `json:"plan_id"`
	PaymentID pgtype.UUID        `json:"payment_id"`
	StartDate pgtype.Timestamptz `json:"start_date"`
	EndDate   pgtype.Timestamptz `json:"end_date"`
}

func (q *Queries) InsertUserSubscription(ctx context.Context, arg InsertUserSubscriptionParams) (Subscription, error) {
	row := q.db.QueryRow(ctx, insertUserSubscription,
		arg.ID,
		arg.UserID,
		arg.PlanID,
		arg.PaymentID,
		arg.StartDate,
		arg.EndDate,
	)
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

const insertUserTask = `-- name: InsertUserTask :one
INSERT INTO task (id, user_id, name, description)
VALUES ($1, $2, $3, $4) RETURNING id, user_id, name, description, checked
`

type InsertUserTaskParams struct {
	ID          pgtype.UUID `json:"id"`
	UserID      pgtype.UUID `json:"user_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
}

func (q *Queries) InsertUserTask(ctx context.Context, arg InsertUserTaskParams) (Task, error) {
	row := q.db.QueryRow(ctx, insertUserTask,
		arg.ID,
		arg.UserID,
		arg.Name,
		arg.Description,
	)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Description,
		&i.Checked,
	)
	return i, err
}

const revokeUserRefreshToken = `-- name: RevokeUserRefreshToken :one
UPDATE refresh_token SET revoked = TRUE
WHERE id = $1 AND user_id = $2 RETURNING id, user_id, revoked, expires_at
`

type RevokeUserRefreshTokenParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID pgtype.UUID `json:"user_id"`
}

func (q *Queries) RevokeUserRefreshToken(ctx context.Context, arg RevokeUserRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRow(ctx, revokeUserRefreshToken, arg.ID, arg.UserID)
	var i RefreshToken
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Revoked,
		&i.ExpiresAt,
	)
	return i, err
}

const selectCoupon = `-- name: SelectCoupon :one
SELECT code, influencer_username, quota, created_at, deleted_at FROM coupon WHERE code = $1
`

func (q *Queries) SelectCoupon(ctx context.Context, code string) (Coupon, error) {
	row := q.db.QueryRow(ctx, selectCoupon, code)
	var i Coupon
	err := row.Scan(
		&i.Code,
		&i.InfluencerUsername,
		&i.Quota,
		&i.CreatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const selectPlan = `-- name: SelectPlan :one
SELECT id, type, name, price, duration_in_months, created_at, deleted_at FROM plan WHERE id = $1 AND deleted_at IS NULL
`

func (q *Queries) SelectPlan(ctx context.Context, id pgtype.UUID) (Plan, error) {
	row := q.db.QueryRow(ctx, selectPlan, id)
	var i Plan
	err := row.Scan(
		&i.ID,
		&i.Type,
		&i.Name,
		&i.Price,
		&i.DurationInMonths,
		&i.CreatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const selectPlanByInvoiceId = `-- name: SelectPlanByInvoiceId :one
SELECT p.id, p.type, p.name, p.price, p.duration_in_months, p.created_at, p.deleted_at FROM invoice i JOIN plan p ON i.plan_id = p.id WHERE i.id = $1
`

func (q *Queries) SelectPlanByInvoiceId(ctx context.Context, id pgtype.UUID) (Plan, error) {
	row := q.db.QueryRow(ctx, selectPlanByInvoiceId, id)
	var i Plan
	err := row.Scan(
		&i.ID,
		&i.Type,
		&i.Name,
		&i.Price,
		&i.DurationInMonths,
		&i.CreatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const selectPlans = `-- name: SelectPlans :many
SELECT id, type, name, price, duration_in_months, created_at, deleted_at FROM plan WHERE deleted_at IS NULL
`

func (q *Queries) SelectPlans(ctx context.Context) ([]Plan, error) {
	rows, err := q.db.Query(ctx, selectPlans)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Plan
	for rows.Next() {
		var i Plan
		if err := rows.Scan(
			&i.ID,
			&i.Type,
			&i.Name,
			&i.Price,
			&i.DurationInMonths,
			&i.CreatedAt,
			&i.DeletedAt,
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

const selectUser = `-- name: SelectUser :one
SELECT id, email, password, name, coordinates, city, timezone, created_at FROM "user" WHERE id = $1
`

func (q *Queries) SelectUser(ctx context.Context, id pgtype.UUID) (User, error) {
	row := q.db.QueryRow(ctx, selectUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.Name,
		&i.Coordinates,
		&i.City,
		&i.Timezone,
		&i.CreatedAt,
	)
	return i, err
}

const selectUserActiveInvoice = `-- name: SelectUserActiveInvoice :one
SELECT i.id, i.user_id, i.plan_id, i.ref_id, i.coupon_code, i.total_amount, i.qr_url, i.expires_at, i.created_at FROM invoice i
WHERE i.user_id = $1 AND i.expires_at > NOW()
AND NOT EXISTS (
    SELECT 1 
    FROM payment p 
    WHERE p.invoice_id = i.id
)
`

func (q *Queries) SelectUserActiveInvoice(ctx context.Context, userID pgtype.UUID) (Invoice, error) {
	row := q.db.QueryRow(ctx, selectUserActiveInvoice, userID)
	var i Invoice
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.PlanID,
		&i.RefID,
		&i.CouponCode,
		&i.TotalAmount,
		&i.QrUrl,
		&i.ExpiresAt,
		&i.CreatedAt,
	)
	return i, err
}

const selectUserActiveSubscription = `-- name: SelectUserActiveSubscription :one
SELECT id, user_id, plan_id, payment_id, start_date, end_date FROM subscription WHERE user_id = $1 AND end_date > NOW()
`

func (q *Queries) SelectUserActiveSubscription(ctx context.Context, userID pgtype.UUID) (Subscription, error) {
	row := q.db.QueryRow(ctx, selectUserActiveSubscription, userID)
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

const selectUserByEmail = `-- name: SelectUserByEmail :one
SELECT id, email, password, name, coordinates, city, timezone, created_at FROM "user" WHERE email = $1
`

func (q *Queries) SelectUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRow(ctx, selectUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.Name,
		&i.Coordinates,
		&i.City,
		&i.Timezone,
		&i.CreatedAt,
	)
	return i, err
}

const selectUserByInvoiceId = `-- name: SelectUserByInvoiceId :one
SELECT u.id, u.email, u.password, u.name, u.coordinates, u.city, u.timezone, u.created_at FROM invoice i JOIN "user" u ON i.user_id = u.id WHERE i.id = $1
`

func (q *Queries) SelectUserByInvoiceId(ctx context.Context, id pgtype.UUID) (User, error) {
	row := q.db.QueryRow(ctx, selectUserByInvoiceId, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.Name,
		&i.Coordinates,
		&i.City,
		&i.Timezone,
		&i.CreatedAt,
	)
	return i, err
}

const selectUserPayments = `-- name: SelectUserPayments :many
SELECT id, user_id, invoice_id, amount_paid, status, created_at FROM payment WHERE user_id = $1
`

func (q *Queries) SelectUserPayments(ctx context.Context, userID pgtype.UUID) ([]Payment, error) {
	rows, err := q.db.Query(ctx, selectUserPayments, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Payment
	for rows.Next() {
		var i Payment
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.InvoiceID,
			&i.AmountPaid,
			&i.Status,
			&i.CreatedAt,
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

const selectUserPrayers = `-- name: SelectUserPrayers :many
SELECT id, user_id, name, status, year, month, day FROM prayer
WHERE user_id = $1 AND year = $2 AND month = $3
AND (day = $4 OR $4 IS NULL)
`

type SelectUserPrayersParams struct {
	UserID pgtype.UUID `json:"user_id"`
	Year   int16       `json:"year"`
	Month  int16       `json:"month"`
	Day    pgtype.Int2 `json:"day"`
}

func (q *Queries) SelectUserPrayers(ctx context.Context, arg SelectUserPrayersParams) ([]Prayer, error) {
	rows, err := q.db.Query(ctx, selectUserPrayers,
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

const selectUserRefreshToken = `-- name: SelectUserRefreshToken :one
SELECT id, user_id, revoked, expires_at FROM refresh_token WHERE id = $1 AND user_id = $2
`

type SelectUserRefreshTokenParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID pgtype.UUID `json:"user_id"`
}

func (q *Queries) SelectUserRefreshToken(ctx context.Context, arg SelectUserRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRow(ctx, selectUserRefreshToken, arg.ID, arg.UserID)
	var i RefreshToken
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Revoked,
		&i.ExpiresAt,
	)
	return i, err
}

const selectUserTasks = `-- name: SelectUserTasks :many
SELECT id, user_id, name, description, checked FROM task WHERE user_id = $1
`

func (q *Queries) SelectUserTasks(ctx context.Context, userID pgtype.UUID) ([]Task, error) {
	rows, err := q.db.Query(ctx, selectUserTasks, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Task
	for rows.Next() {
		var i Task
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.Name,
			&i.Description,
			&i.Checked,
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

const updateUser = `-- name: UpdateUser :one
UPDATE "user"
SET
  email = COALESCE($2, email),
  password = COALESCE($3, password),
  name = COALESCE($4, name),
  coordinates = COALESCE($5, coordinates),
  city = COALESCE($6, city),
  timezone = COALESCE($7, timezone)
WHERE id = $1 RETURNING id, email, password, name, coordinates, city, timezone, created_at
`

type UpdateUserParams struct {
	ID          pgtype.UUID  `json:"id"`
	Email       pgtype.Text  `json:"email"`
	Password    pgtype.Text  `json:"password"`
	Name        pgtype.Text  `json:"name"`
	Coordinates pgtype.Point `json:"coordinates"`
	City        pgtype.Text  `json:"city"`
	Timezone    pgtype.Text  `json:"timezone"`
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUser,
		arg.ID,
		arg.Email,
		arg.Password,
		arg.Name,
		arg.Coordinates,
		arg.City,
		arg.Timezone,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.Name,
		&i.Coordinates,
		&i.City,
		&i.Timezone,
		&i.CreatedAt,
	)
	return i, err
}

const updateUserPrayer = `-- name: UpdateUserPrayer :one
UPDATE prayer
SET status = COALESCE($3, status)
WHERE id = $1 AND user_id = $2 RETURNING id, user_id, name, status, year, month, day
`

type UpdateUserPrayerParams struct {
	ID     pgtype.UUID `json:"id"`
	UserID pgtype.UUID `json:"user_id"`
	Status pgtype.Text `json:"status"`
}

func (q *Queries) UpdateUserPrayer(ctx context.Context, arg UpdateUserPrayerParams) (Prayer, error) {
	row := q.db.QueryRow(ctx, updateUserPrayer, arg.ID, arg.UserID, arg.Status)
	var i Prayer
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Status,
		&i.Year,
		&i.Month,
		&i.Day,
	)
	return i, err
}

const updateUserTask = `-- name: UpdateUserTask :one
UPDATE task
SET
  name = COALESCE($3, name),
  description = COALESCE($4, description),
  checked = COALESCE($5, checked)
WHERE id = $1 AND user_id = $2 RETURNING id, user_id, name, description, checked
`

type UpdateUserTaskParams struct {
	ID          pgtype.UUID `json:"id"`
	UserID      pgtype.UUID `json:"user_id"`
	Name        pgtype.Text `json:"name"`
	Description pgtype.Text `json:"description"`
	Checked     pgtype.Bool `json:"checked"`
}

func (q *Queries) UpdateUserTask(ctx context.Context, arg UpdateUserTaskParams) (Task, error) {
	row := q.db.QueryRow(ctx, updateUserTask,
		arg.ID,
		arg.UserID,
		arg.Name,
		arg.Description,
		arg.Checked,
	)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Description,
		&i.Checked,
	)
	return i, err
}
