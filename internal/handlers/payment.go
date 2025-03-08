package handlers

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/goccy/go-json"

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

type PaymentHandler interface {
	GetActiveInvoice(res http.ResponseWriter, req *http.Request)
	CreateInvoice(res http.ResponseWriter, req *http.Request)
	TripayCallback(res http.ResponseWriter, req *http.Request)
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

type invoiceResponse struct {
	Id          string `json:"id"`
	RefId       string `json:"ref_id"`
	CouponCode  string `json:"coupon_code"`
	TotalAmount int32  `json:"total_amount"`
	QrUrl       string `json:"qr_url"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
}

func (p payment) GetActiveInvoice(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	invoice, err := retryutil.RetryWithData(func() (repository.Invoice, error) {
		return p.configs.Db.Queries.SelectActiveInvoice(ctx, userId)
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select active invoice")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	params := httputil.SendSuccessResponseParams{StatusCode: http.StatusOK}
	if err == nil {
		resBody := invoiceResponse{
			Id:          invoice.ID.String(),
			RefId:       invoice.RefID,
			CouponCode:  invoice.CouponCode.String,
			TotalAmount: invoice.TotalAmount,
			QrUrl:       invoice.QrUrl,
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
	var couponCode pgtype.Text

	defer func() {
		if shouldRollbackCoupon {
			err := retryutil.RetryWithoutData(func() error {
				return p.configs.Db.Queries.IncrementCouponQuota(ctx, reqBody.CouponCode)
			})

			if err != nil {
				logger.
					Error().
					Err(err).Int("status_code", http.StatusInternalServerError).
					Str("coupon_code", reqBody.CouponCode).
					Msg("failed to rollback coupon quota")
			}
		}
	}()

	if reqBody.CouponCode != "" {
		affectedRows, err := retryutil.RetryWithData(func() (int64, error) {
			return p.configs.Db.Queries.DecrementCouponQuota(ctx, reqBody.CouponCode)
		})

		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to decrement coupon quota")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if affectedRows == 1 {
			couponCode.String = reqBody.CouponCode
			couponCode.Valid = true
		}
	}

	merchantRef := uuid.New()
	merchantRefString := merchantRef.String()

	totalAmount := reqBody.Plan.Price
	if couponCode.Valid {
		totalAmount = int(math.Round(float64(reqBody.Plan.Price) * 0.7))
	}

	tripayTxRequest := p.service.CreateTripayTxRequest(services.CreateTripayTxRequestParams{
		MerchantRef:   merchantRefString,
		CustomerName:  reqBody.CustomerName,
		CustomerEmail: reqBody.CustomerEmail,
		TotalAmount:   totalAmount,
		PlanId:        reqBody.Plan.Id,
		PlanName:      reqBody.Plan.Name,
		PlanPrice:     reqBody.Plan.Price,
	})

	tripayTxResponse, err := p.service.RequestTripayTx(ctx, tripayTxRequest)
	if err != nil {
		if couponCode.Valid {
			shouldRollbackCoupon = true
		}

		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to request tripay tx")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	userId := ctx.Value(userIdKey{}).(string)
	expiresAt := time.Unix(int64(tripayTxResponse.ExpiredTime), 0)

	retryableFunc := func() (repository.Invoice, error) {
		planUUID, err := uuid.Parse(reqBody.Plan.Id)
		if err != nil {
			return repository.Invoice{}, fmt.Errorf("failed to parse plan Id to UUID: %w", err)
		}

		return p.configs.Db.Queries.InsertInvoice(ctx, repository.InsertInvoiceParams{
			ID:          pgtype.UUID{Bytes: merchantRef, Valid: true},
			UserID:      userId,
			PlanID:      pgtype.UUID{Bytes: planUUID, Valid: true},
			RefID:       tripayTxResponse.Reference,
			CouponCode:  couponCode,
			TotalAmount: int32(tripayTxResponse.Amount),
			QrUrl:       tripayTxResponse.QrURL,
			ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
	}

	invoice, err := retryutil.RetryWithData(retryableFunc)
	if err != nil {
		if couponCode.Valid {
			shouldRollbackCoupon = true
		}

		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to insert invoice")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := invoiceResponse{
		Id:          merchantRefString,
		RefId:       invoice.RefID,
		CouponCode:  invoice.CouponCode.String,
		TotalAmount: invoice.TotalAmount,
		QrUrl:       invoice.QrUrl,
		ExpiresAt:   invoice.ExpiresAt.Time.Format(time.RFC3339),
		CreatedAt:   invoice.CreatedAt.Time.Format(time.RFC3339),
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusCreated,
		ResBody:    resBody,
	}

	res.Header().Set("Location", fmt.Sprintf("%s/invoices/%s", p.configs.Env.OriginURL, merchantRefString))
	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully created invoice")
}

func (p payment) TripayCallback(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to read tripay callback request")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tripaySignature := req.Header.Get("X-Callback-Signature")
	err = p.service.ValidateCallbackSignature(tripaySignature, bytes)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusForbidden).Send()
		http.Error(res, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	var body struct {
		Reference   string `json:"reference"`
		MerchantRef string `json:"merchant_ref"`
		TotalAmount int    `json:"total_amount"`
		Status      string `json:"status"`
	}

	if err := json.Unmarshal(bytes, &body); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to unmarshal tripay callback request")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	user, err := retryutil.RetryWithData(func() (repository.User, error) {
		invoiceUUID, err := uuid.Parse(body.MerchantRef)
		if err != nil {
			return repository.User{}, fmt.Errorf("failed to parse invoice Id to UUID: %w", err)
		}

		return p.configs.Db.Queries.SelectUserByInvoiceID(ctx, pgtype.UUID{Bytes: invoiceUUID, Valid: true})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select user by invoice Id")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if body.Status == "PAID" {
		err = p.service.ProcessSuccessfulPayment(ctx, services.ProcessSuccessfulPaymentParams{
			InvoiceId:  body.MerchantRef,
			UserId:     user.ID,
			AmountPaid: int32(body.TotalAmount),
			Status:     body.Status,
		})

		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to process successful payment")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	} else {
		err = p.service.ProcessUnsuccessfulPayment(ctx, services.ProcessUnsuccessfulPaymentParams{
			InvoiceId:  body.MerchantRef,
			UserId:     user.ID,
			AmountPaid: int32(body.TotalAmount),
			Status:     body.Status,
		})

		if err != nil {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to process unsuccessful payment")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	resBody := struct {
		Status bool `json:"status"`
	}{
		Status: true,
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

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully processed tripay callback request")
}
