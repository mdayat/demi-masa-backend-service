package handlers

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type PrayerHandler interface {
	GetPrayers(res http.ResponseWriter, req *http.Request)
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

	prayers, err := p.service.SelectPrayers(ctx, selectPrayersParams)
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully get prayers")
}
