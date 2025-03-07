-- name: InsertUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING *;

-- name: SelectUserById :one
SELECT * FROM "user" WHERE id = $1 AND deleted_at IS NULL;

-- name: SelectUserByInvoiceID :one
SELECT u.* FROM invoice i JOIN "user" u ON i.user_id = u.id WHERE i.id = $1;

-- name: InsertSubscription :exec
INSERT INTO subscription (id, user_id, plan_id, payment_id, start_date, end_date) VALUES ($1, $2, $3, $4, $5, $6);

-- name: SelectActiveSubscription :one
SELECT * FROM subscription WHERE user_id = $1 AND end_date > NOW();

-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (id, user_id, expires_at) VALUES ($1, $2, $3);

-- name: SelectRefreshTokenById :one
SELECT * FROM refresh_token WHERE id = $1 AND user_id = $2;

-- name: RevokeRefreshToken :exec
UPDATE refresh_token SET revoked = TRUE WHERE id = $1 AND user_id = $2;

-- name: InsertPrayers :copyfrom
INSERT INTO prayer (id, user_id, name, year, month, day)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: SelectPrayers :many
SELECT * FROM prayer WHERE user_id = $1 AND year = $2 AND month = $3 AND (day = sqlc.narg('day') OR sqlc.narg('day') IS NULL);

-- name: UpdatePrayerStatus :exec
UPDATE prayer SET status = $2 WHERE id = $1;

-- name: InsertInvoice :one
INSERT INTO invoice (id, user_id, plan_id, ref_id, coupon_code, total_amount, qr_url, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: SelectActiveInvoice :one
SELECT i.* FROM invoice i
WHERE i.user_id = $1 AND i.expires_at > NOW()
AND NOT EXISTS (
    SELECT 1 
    FROM payment p 
    WHERE p.invoice_id = i.id
);

-- name: DecrementCouponQuota :execrows
UPDATE coupon SET quota = quota - 1
WHERE code = $1 AND quota > 0 AND deleted_at IS NULL;

-- name: IncrementCouponQuota :exec
UPDATE coupon SET quota = quota + 1 WHERE code = $1;

-- name: InsertPayment :exec
INSERT INTO payment (id, user_id, invoice_id, amount_paid, status) VALUES ($1, $2, $3, $4, $5);

-- name: SelectPlanByInvoiceId :one
SELECT p.* FROM invoice i JOIN plan p ON i.plan_id = p.id WHERE i.id = $1;