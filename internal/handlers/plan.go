package handlers

import (
	"net/http"
	"time"

	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/retryutil"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type PlanHandler interface {
	GetPlans(res http.ResponseWriter, req *http.Request)
}

type plan struct {
	configs configs.Configs
}

func NewPlanHandler(configs configs.Configs) PlanHandler {
	return &plan{
		configs: configs,
	}
}

type getPlansResponse struct {
	Id               string
	Name             string
	Price            int32
	DurationInMonths int16
	CreatedAt        string
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

	resBody := make([]getPlansResponse, 0, len(plans))
	for _, plan := range plans {
		resBody = append(resBody, getPlansResponse{
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
