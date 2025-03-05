package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/rs/zerolog/log"
)

type PaymentHandler interface {
	GetActiveInvoice(res http.ResponseWriter, req *http.Request)
}

type payment struct {
	configs configs.Configs
	service services.PaymentServicer
}

func NewPaymentHandler(configs configs.Configs, service services.PaymentServicer) PaymentHandler {
	return &payment{
		configs: configs,
		service: service,
	}
}

func (p payment) GetActiveInvoice(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	invoice, err := p.service.SelectActiveInvoice(ctx, userId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select active invoice")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	params := httputil.SendSuccessResponseParams{StatusCode: http.StatusOK}
	if err == nil {
		resBody := struct {
			Id          string `json:"id"`
			TotalAmount int32  `json:"total_amount"`
			Status      string `json:"status"`
			ExpiresAt   string `json:"expires_at"`
			CreatedAt   string `json:"created_at"`
		}{
			Id:          invoice.ID.String(),
			TotalAmount: invoice.TotalAmount,
			Status:      invoice.Status,
			ExpiresAt:   invoice.ExpiresAt.Time.Format(time.RFC3339),
			CreatedAt:   invoice.CreatedAt.Time.Format(time.RFC3339),
		}

		params.ResBody = resBody
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got active invoice")
}
