-- name: InsertUser :one
INSERT INTO "user" (id) VALUES ($1) RETURNING *;

-- name: SelectUserById :one
SELECT * FROM "user" WHERE id = $1 AND deleted_at IS NULL;

-- name: SelectActiveSubscription :one
SELECT * FROM subscription WHERE user_id = $1 AND end_date > NOW();

-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (id, user_id, expires_at) VALUES ($1, $2, $3);

-- name: SelectRefreshTokenById :one
SELECT * FROM refresh_token WHERE id = $1 AND user_id = $2;

-- name: RevokeRefreshToken :exec
UPDATE refresh_token SET revoked = TRUE WHERE id = $1 AND user_id = $2;

-- name: SelectPrayers :many
SELECT * FROM prayer WHERE user_id = $1 AND year = $2 AND month = $3 AND (day = sqlc.narg('day') OR sqlc.narg('day') IS NULL);

-- name: UpdatePrayerStatus :exec
UPDATE prayer SET status = $2 WHERE id = $1;

-- name: InsertInvoice :one
INSERT INTO invoice (id, user_id, ref_id, coupon_code, total_amount, qr_url, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: SelectActiveInvoice :one
SELECT * FROM invoice WHERE user_id = $1 AND status = 'unpaid' AND expires_at > NOW();

-- name: DecrementCouponQuota :execrows
UPDATE coupon SET quota = quota - 1
WHERE code = $1 AND quota > 0 AND deleted_at IS NULL;

-- name: IncrementCouponQuota :exec
UPDATE coupon SET quota = quota + 1 WHERE code = $1;