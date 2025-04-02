package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	logger := log.With().Caller().Logger()

	env, err := configs.LoadEnv()
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	db, err := configs.NewDb(ctx, env.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	config := configs.Configs{
		Env: env,
		Db:  db,
	}

	// Seed "user" table
	user, err := retryutil.RetryWithData(func() (repository.User, error) {
		return db.Queries.InsertUser(ctx, repository.InsertUserParams{
			ID:          pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Email:       "example@gmail.com",
			Name:        "example",
			Password:    "example",
			Coordinates: pgtype.Point{P: pgtype.Vec2{X: 106.865036, Y: -6.175110}, Valid: true},
			City:        "Jakarta",
			Timezone:    "Asia/Jakarta",
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed user table")
	}

	// Seed "refresh_token" table
	authService := services.NewAuthService(config)
	now := time.Now()

	_, err = retryutil.RetryWithData(func() (repository.RefreshToken, error) {
		refreshTokenClaims := services.RefreshTokenClaims{
			Type: services.Refresh,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        uuid.NewString(),
				ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    config.Env.OriginURL,
				Subject:   user.ID.String(),
			},
		}

		_, err := authService.CreateRefreshToken(refreshTokenClaims)
		if err != nil {
			return repository.RefreshToken{}, fmt.Errorf("failed to create refresh token: %w", err)
		}

		return db.Queries.InsertUserRefreshToken(ctx, repository.InsertUserRefreshTokenParams{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: refreshTokenClaims.ExpiresAt.Time, Valid: true},
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed refresh_token table")
	}

	// Seed "prayer" table
	_, err = retryutil.RetryWithData(func() (int64, error) {
		return db.Queries.InsertUserPrayers(ctx, []repository.InsertUserPrayersParams{
			{
				ID:     pgtype.UUID{Bytes: uuid.New(), Valid: true},
				UserID: user.ID,
				Name:   "subuh",
				Year:   int16(now.Year()),
				Month:  int16(now.Month()),
				Day:    int16(now.Day()),
			},
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed prayer table")
	}

	// Seed "coupon" table
	coupon, err := retryutil.RetryWithData(func() (repository.Coupon, error) {
		return db.Queries.InsertCoupon(ctx, repository.InsertCouponParams{
			Code:               "example",
			InfluencerUsername: "example",
			Quota:              1000,
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed coupon table")
	}

	// Seed "plan" table
	plan, err := retryutil.RetryWithData(func() (repository.Plan, error) {
		return db.Queries.InsertPlan(ctx, repository.InsertPlanParams{
			ID:               pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Type:             "premium",
			Name:             "example",
			Price:            100000,
			DurationInMonths: 1,
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed plan table")
	}

	// Seed "invoice" table
	invoice, err := retryutil.RetryWithData(func() (repository.Invoice, error) {
		return db.Queries.InsertUserInvoice(ctx, repository.InsertUserInvoiceParams{
			ID:          pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:      user.ID,
			PlanID:      plan.ID,
			RefID:       uuid.NewString(),
			CouponCode:  pgtype.Text{String: coupon.Code, Valid: true},
			TotalAmount: 100000,
			QrUrl:       "example.com",
			ExpiresAt:   pgtype.Timestamptz{Time: now.Add(time.Hour * 1), Valid: true},
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed invoice table")
	}

	// Seed "payment" table
	payment, err := retryutil.RetryWithData(func() (repository.Payment, error) {
		return db.Queries.InsertUserPayment(ctx, repository.InsertUserPaymentParams{
			ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:     user.ID,
			InvoiceID:  invoice.ID,
			AmountPaid: 100000,
			Status:     "paid",
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed payment table")
	}

	// Seed "subscription" table
	_, err = retryutil.RetryWithData(func() (repository.Subscription, error) {
		return db.Queries.InsertUserSubscription(ctx, repository.InsertUserSubscriptionParams{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    user.ID,
			PaymentID: payment.ID,
			PlanID:    plan.ID,
			StartDate: pgtype.Timestamptz{Time: now, Valid: true},
			EndDate:   pgtype.Timestamptz{Time: now.AddDate(0, int(plan.DurationInMonths), 0), Valid: true},
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed subscription table")
	}

	// Seed "task" table
	_, err = retryutil.RetryWithData(func() (repository.Task, error) {
		return db.Queries.InsertUserTask(ctx, repository.InsertUserTaskParams{
			ID:          pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:      user.ID,
			Name:        "example",
			Description: "example",
		})
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed task table")
	}
}
