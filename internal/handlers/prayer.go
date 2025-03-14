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
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
	"github.com/mdayat/demi-masa-backend-service/internal/httputil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog/log"
)

type PrayerHandler interface {
	GetPrayers(res http.ResponseWriter, req *http.Request)
	UpdatePrayer(res http.ResponseWriter, req *http.Request)
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

	selectPrayersParams := repository.SelectUserPrayersParams{
		Year:  int16(year),
		Month: int16(month),
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

	userId := ctx.Value(userIdKey{}).(string)
	prayers, err := retryutil.RetryWithData(func() ([]repository.Prayer, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return []repository.Prayer{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		selectPrayersParams.UserID = pgtype.UUID{Bytes: userUUID, Valid: true}
		return p.configs.Db.Queries.SelectUserPrayers(ctx, selectPrayersParams)
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select prayers")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := make([]dtos.PrayerResponse, 0, len(prayers))
	for _, prayer := range prayers {
		resBody = append(resBody, dtos.PrayerResponse{
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

func (p prayer) UpdatePrayer(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody dtos.PrayerRequest
	if err := httputil.DecodeAndValidate(req, p.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if reqBody.Status == "" {
		res.WriteHeader(http.StatusNoContent)
		logger.Info().Int("status_code", http.StatusNoContent).Msg("no update performed")
		return
	}

	var status pgtype.Text
	if reqBody.Status != "" {
		status = pgtype.Text{String: reqBody.Status, Valid: true}
	}

	prayerId := chi.URLParam(req, "prayerId")
	userId := ctx.Value(userIdKey{}).(string)

	prayer, err := retryutil.RetryWithData(func() (repository.Prayer, error) {
		prayerUUID, err := uuid.Parse(prayerId)
		if err != nil {
			return repository.Prayer{}, fmt.Errorf("failed to parse prayer Id to UUID: %w", err)
		}

		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.Prayer{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return p.configs.Db.Queries.UpdateUserPrayer(ctx, repository.UpdateUserPrayerParams{
			ID:     pgtype.UUID{Bytes: prayerUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
			Status: status,
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

	resBody := dtos.PrayerResponse{
		Id:     prayer.ID.String(),
		Name:   prayer.Name,
		Status: prayer.Status,
		Year:   prayer.Year,
		Month:  prayer.Month,
		Day:    prayer.Day,
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully updated prayer status")
}
