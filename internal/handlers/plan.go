package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/retryutil"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type PlanHandler interface {
	GetPlans(res http.ResponseWriter, req *http.Request)
	GetPlan(res http.ResponseWriter, req *http.Request)
}

type plan struct {
	configs configs.Configs
}

func NewPlanHandler(configs configs.Configs) PlanHandler {
	return &plan{
		configs: configs,
	}
}

type getPlanResponse struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Price            int32  `json:"price"`
	DurationInMonths int16  `json:"duration_in_months"`
	CreatedAt        string `json:"created_at"`
}

func (p plan) GetPlans(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	plans, err := retryutil.RetryWithData(func() ([]repository.Plan, error) {
		return p.configs.Db.Queries.SelectPlans(ctx)
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select plans")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := make([]getPlanResponse, 0, len(plans))
	for _, plan := range plans {
		resBody = append(resBody, getPlanResponse{
			Id:               plan.ID.String(),
			Name:             plan.Name,
			Price:            plan.Price,
			DurationInMonths: plan.DurationInMonths,
			CreatedAt:        plan.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusOK,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got plans")
}

func (p plan) GetPlan(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	planId := chi.URLParam(req, "planId")
	plan, err := retryutil.RetryWithData(func() (repository.Plan, error) {
		planUUID, err := uuid.Parse(planId)
		if err != nil {
			return repository.Plan{}, fmt.Errorf("failed to parse plan Id to UUID: %w", err)
		}

		return p.configs.Db.Queries.SelectPlanById(ctx, pgtype.UUID{Bytes: planUUID, Valid: true})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select plan by Id")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := getPlanResponse{
		Id:               plan.ID.String(),
		Name:             plan.Name,
		Price:            plan.Price,
		DurationInMonths: plan.DurationInMonths,
		CreatedAt:        plan.CreatedAt.Time.Format(time.RFC3339),
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusOK,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got plan by Id")
}
