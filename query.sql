-- name: InsertUser :one
INSERT INTO "user" (id, email, password, name, coordinates, city, timezone)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: SelectUser :one
SELECT 
  u.*, 
  to_jsonb(s) AS subscription
FROM "user" u
LEFT JOIN subscription s ON s.user_id = u.id
WHERE u.id = $1;

-- name: SelectUserByEmail :one
SELECT * FROM "user" WHERE email = $1;

-- name: SelectUserByInvoiceId :one
SELECT u.* FROM invoice i JOIN "user" u ON i.user_id = u.id WHERE i.id = $1;

-- name: UpdateUser :one
UPDATE "user"
SET
  email = COALESCE(sqlc.narg(email), email),
  password = COALESCE(sqlc.narg(password), password),
  name = COALESCE(sqlc.narg(name), name),
  coordinates = COALESCE(sqlc.narg(coordinates), coordinates),
  city = COALESCE(sqlc.narg(city), city),
  timezone = COALESCE(sqlc.narg(timezone), timezone)
WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM "user" WHERE id = $1;

-- name: InsertUserSubscription :one
INSERT INTO subscription (id, user_id, plan_id, payment_id, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: SelectUserActiveSubscription :one
SELECT * FROM subscription WHERE user_id = $1 AND end_date > NOW();

-- name: InsertUserRefreshToken :one
INSERT INTO refresh_token (id, user_id, expires_at)
VALUES ($1, $2, $3) RETURNING *;

-- name: SelectUserRefreshToken :one
SELECT * FROM refresh_token WHERE id = $1 AND user_id = $2;

-- name: RevokeUserRefreshToken :one
UPDATE refresh_token SET revoked = TRUE
WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: InsertUserPrayers :copyfrom
INSERT INTO prayer (id, user_id, name, year, month, day)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: SelectUserPrayers :many
SELECT * FROM prayer
WHERE user_id = $1 AND year = $2 AND month = $3
AND (day = sqlc.narg('day') OR sqlc.narg('day') IS NULL);

-- name: UpdateUserPrayer :one
UPDATE prayer
SET status = COALESCE(sqlc.narg(status), status)
WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: InsertUserInvoice :one
INSERT INTO invoice (id, user_id, plan_id, ref_id, coupon_code, total_amount, qr_url, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: SelectUserActiveInvoice :one
SELECT i.* FROM invoice i
WHERE i.user_id = $1 AND i.expires_at > NOW()
AND NOT EXISTS (
    SELECT 1 
    FROM payment p 
    WHERE p.invoice_id = i.id
);

-- name: InsertCoupon :one
INSERT INTO coupon (code, influencer_username, quota)
VALUES ($1, $2, $3) RETURNING *;

-- name: SelectCoupon :one
SELECT * FROM coupon WHERE code = $1;

-- name: DecrementCouponQuota :execrows
UPDATE coupon SET quota = quota - 1
WHERE code = $1 AND quota > 0 AND deleted_at IS NULL;

-- name: IncrementCouponQuota :exec
UPDATE coupon SET quota = quota + 1 WHERE code = $1;

-- name: InsertUserPayment :one
INSERT INTO payment (id, user_id, invoice_id, amount_paid, status)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: SelectUserPayments :many
SELECT * FROM payment WHERE user_id = $1;

-- name: InsertPlan :one
INSERT INTO plan (id, type, name, price, duration_in_months)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: SelectPlanByInvoiceId :one
SELECT p.* FROM invoice i JOIN plan p ON i.plan_id = p.id WHERE i.id = $1;

-- name: SelectPlans :many
SELECT * FROM plan WHERE deleted_at IS NULL;

-- name: SelectPlan :one
SELECT * FROM plan WHERE id = $1 AND deleted_at IS NULL;

-- name: SelectUserTasks :many
SELECT * FROM task WHERE user_id = $1;

-- name: InsertUserTask :one
INSERT INTO task (id, user_id, name, description)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateUserTask :one
UPDATE task
SET
  name = COALESCE(sqlc.narg(name), name),
  description = COALESCE(sqlc.narg(description), description),
  checked = COALESCE(sqlc.narg(checked), checked)
WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: DeleteUserTask :execrows
DELETE FROM task WHERE id = $1 AND user_id = $2;