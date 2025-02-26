package dbutil

import (
	"context"

	"github.com/avast/retry-go/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mdayat/demi-masa/repository"
)

func RetryableTxWithData[T any](
	ctx context.Context,
	conn *pgxpool.Pool,
	queries *repository.Queries,
	f func(qtx *repository.Queries) (T, error),
) (T, error) {
	retryableFunc := func() (zero T, err error) {
		var tx pgx.Tx
		tx, err = conn.Begin(ctx)
		if err != nil {
			return zero, err
		}

		defer func() {
			if err == nil {
				err = tx.Commit(ctx)
			}

			if err != nil {
				tx.Rollback(ctx)
			}
		}()

		qtx := queries.WithTx(tx)
		return f(qtx)
	}

	return retry.DoWithData(retryableFunc, retry.Attempts(3), retry.LastErrorOnly(true))
}
