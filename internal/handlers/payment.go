package handlers

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/httputil"
	"github.com/mdayat/demi-masa/internal/services"
	"github.com/mdayat/demi-masa/repository"
	"github.com/rs/zerolog/log"
)

type PaymentHandler interface {
	GetActiveInvoice(res http.ResponseWriter, req *http.Request)
	CreateInvoice(res http.ResponseWriter, req *http.Request)
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

type createInvoiceRequest struct {
	CouponCode    string `json:"coupon_code" validate:"required"`
	CustomerName  string `json:"customer_name" validate:"required"`
	CustomerEmail string `json:"customer_email" validate:"required,email"`
	Plan          struct {
		Id               string `json:"id" validate:"required,uuid"`
		Name             string `json:"name" validate:"required"`
		Price            int    `json:"price" validate:"required"`
		DurationInMonths int    `json:"duration_in_months" validate:"required"`
	}
}

func (p payment) CreateInvoice(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody createInvoiceRequest
	if err := httputil.DecodeAndValidate(req, p.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var shouldRollbackCoupon bool
	var isCouponCodeValid bool

	defer func() {
		if shouldRollbackCoupon {
			if err := p.service.IncrementCouponQuota(ctx, reqBody.CouponCode); err != nil {
				logger.
					Error().
					Err(err).Int("status_code", http.StatusInternalServerError).
					Str("coupon_code", reqBody.CouponCode).
					Msg("failed to rollback coupon quota")
			}
		}
	}()

	if reqBody.CouponCode != "" {
		affectedRows, err := p.service.DecrementCouponQuota(ctx, reqBody.CouponCode)
		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to decrement coupon quota")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if affectedRows == 1 {
			isCouponCodeValid = true
		}
	}

	merchantRef := uuid.New()
	merchantRefString := merchantRef.String()

	totalAmount := reqBody.Plan.Price
	if isCouponCodeValid {
		totalAmount = int(math.Round(float64(reqBody.Plan.Price) * 0.7))
	}

	tripayTxRequest := p.service.CreateTripayTxRequest(services.CreateTripayTxRequestParams{
		MerchantRef:   merchantRefString,
		CustomerName:  reqBody.CustomerName,
		CustomerEmail: reqBody.CustomerEmail,
		TotalAmount:   totalAmount,
		PlanName:      reqBody.Plan.Name,
		PlanPrice:     reqBody.Plan.Price,
	})

	tripayTxResponse, err := p.service.RequestTripayTx(ctx, tripayTxRequest)
	if err != nil {
		if isCouponCodeValid {
			shouldRollbackCoupon = true
		}

		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to request tripay tx")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	userId := ctx.Value(userIdKey{}).(string)
	expiresAt := time.Unix(int64(tripayTxResponse.ExpiredTime), 0)

	err = p.service.InsertInvoice(ctx, repository.InsertInvoiceParams{
		ID:          pgtype.UUID{Bytes: merchantRef, Valid: true},
		UserID:      userId,
		RefID:       tripayTxResponse.Reference,
		TotalAmount: int32(tripayTxResponse.Amount),
		QrUrl:       tripayTxResponse.QrURL,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})

	if err != nil {
		if isCouponCodeValid {
			shouldRollbackCoupon = true
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusConflict).Msg("active invoice already exist")
			http.Error(res, http.StatusText(http.StatusConflict), http.StatusConflict)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to insert invoice")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	resBody := struct {
		Id          string `json:"id"`
		TotalAmount int32  `json:"total_amount"`
		ExpiresAt   string `json:"expires_at"`
	}{
		Id:          merchantRefString,
		TotalAmount: int32(tripayTxResponse.Amount),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusCreated,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully created invoice")
}
