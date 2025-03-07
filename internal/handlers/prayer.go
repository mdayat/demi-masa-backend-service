package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/retryutil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type PrayerHandler interface {
	GetPrayers(res http.ResponseWriter, req *http.Request)
	UpdatePrayerStatus(res http.ResponseWriter, req *http.Request)
}

type prayer struct {
	configs configs.Configs
	service services.PrayerServicer
}

func NewPrayerHandler(configs configs.Configs, service services.PrayerServicer) PrayerHandler {
	return &prayer{
		configs: configs,
		service: service,
	}
}

type getPrayersResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Year   int16  `json:"year"`
	Month  int16  `json:"month"`
	Day    int16  `json:"day"`
}

func (p prayer) GetPrayers(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	yearString := req.URL.Query().Get("year")
	monthString := req.URL.Query().Get("month")

	year, month, err := p.service.ValidateYearAndMonthParams(yearString, monthString)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid query params")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userId := ctx.Value(userIdKey{}).(string)
	selectPrayersParams := repository.SelectPrayersParams{
		UserID: userId,
		Year:   int16(year),
		Month:  int16(month),
	}

	if dayString := req.URL.Query().Get("day"); dayString != "" {
		day, err := strconv.Atoi(dayString)
		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("failed to convert day string to int")
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		selectPrayersParams.Day = pgtype.Int2{Int16: int16(day), Valid: true}
	}

	prayers, err := retryutil.RetryWithData(func() ([]repository.Prayer, error) {
		return p.configs.Db.Queries.SelectPrayers(ctx, selectPrayersParams)
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select prayers")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := make([]getPrayersResponse, 0, len(prayers))
	for _, prayer := range prayers {
		resBody = append(resBody, getPrayersResponse{
			Id:     prayer.ID.String(),
			Name:   prayer.Name,
			Status: prayer.Status,
			Year:   prayer.Year,
			Month:  prayer.Month,
			Day:    prayer.Day,
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got prayers")
}

func (p prayer) UpdatePrayerStatus(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		Id     string `json:"id" validate:"required,uuid"`
		Status string `json:"status" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, p.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	prayerId := chi.URLParam(req, "prayerId")
	err := retryutil.RetryWithoutData(func() error {
		prayerUUID, err := uuid.Parse(prayerId)
		if err != nil {
			return fmt.Errorf("failed to parse prayer Id to UUID: %w", err)
		}

		return p.configs.Db.Queries.UpdatePrayerStatus(ctx, repository.UpdatePrayerStatusParams{
			ID:     pgtype.UUID{Bytes: prayerUUID, Valid: true},
			Status: reqBody.Status,
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("prayer not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to update prayer status")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully updated prayer status")
}
