package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
	"github.com/mdayat/demi-masa-backend-service/internal/httputil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog/log"
)

type SubscriptionHandler interface {
	GetActiveSubscription(res http.ResponseWriter, req *http.Request)
}

type subscription struct {
	configs configs.Configs
}

func NewSubscriptionHandler(configs configs.Configs) SubscriptionHandler {
	return &subscription{
		configs: configs,
	}
}

func (s subscription) GetActiveSubscription(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	subscription, err := retryutil.RetryWithData(func() (repository.Subscription, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.Subscription{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return s.configs.Db.Queries.SelectUserActiveSubscription(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select user active subscription")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	params := httputil.SendSuccessResponseParams{StatusCode: http.StatusOK}
	if err == nil {
		resBody := dtos.SubscriptionResponse{
			Id:        subscription.ID.String(),
			PlanId:    subscription.PlanID.String(),
			PaymentId: subscription.PaymentID.String(),
			StartDate: subscription.StartDate.Time.Format(time.RFC3339),
			EndDate:   subscription.EndDate.Time.Format(time.RFC3339),
		}

		params.ResBody = resBody
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got active subscription")
}
