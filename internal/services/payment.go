package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/dbutil"
	"github.com/mdayat/demi-masa/internal/retryutil"
	"github.com/mdayat/demi-masa/repository"
)

type PaymentServicer interface {
	CreateTripayTxRequest(arg CreateTripayTxRequestParams) tripayTxRequest
	RequestTripayTx(ctx context.Context, tripayTxRequest tripayTxRequest) (tripayTxResponse, error)
	ValidateCallbackSignature(tripaySignature string, reqBody []byte) error
	ProcessSuccessfulPayment(ctx context.Context, arg ProcessSuccessfulPaymentParams) error
	ProcessUnsuccessfulPayment(ctx context.Context, arg ProcessUnsuccessfulPaymentParams) error
}

type payment struct {
	configs configs.Configs
}

func NewPaymentService(configs configs.Configs) PaymentServicer {
	return &payment{
		configs: configs,
	}
}

func (p payment) createRequestSignature(merchantRef string, amount int) string {
	key := []byte(p.configs.Env.TripayPrivateKey)
	message := fmt.Sprintf("%s%s%d", p.configs.Env.TripayMerchantCode, merchantRef, amount)

	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(message))

	return hex.EncodeToString(hash.Sum(nil))
}

var QRISPaymentChannel = "QRIS"

type orderItem struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

type tripayTxRequest struct {
	Method        string      `json:"method"`
	MerchantRef   string      `json:"merchant_ref"`
	Amount        int         `json:"amount"`
	CustomerName  string      `json:"customer_name"`
	CustomerEmail string      `json:"customer_email"`
	OrderItems    []orderItem `json:"order_items"`
	Signature     string      `json:"signature"`
}

type tripayTxResponse struct {
	Reference   string `json:"reference"`
	Amount      int    `json:"amount"`
	ExpiredTime int    `json:"expired_time"`
	QrURL       string `json:"qr_url"`
}

type CreateTripayTxRequestParams struct {
	MerchantRef   string
	CustomerName  string
	CustomerEmail string
	TotalAmount   int
	PlanId        string
	PlanName      string
	PlanPrice     int
}

func (p payment) CreateTripayTxRequest(arg CreateTripayTxRequestParams) tripayTxRequest {
	signature := p.createRequestSignature(arg.MerchantRef, arg.TotalAmount)
	orderItems := []orderItem{
		{
			Id:       arg.PlanId,
			Name:     arg.PlanName,
			Price:    arg.PlanPrice,
			Quantity: 1,
		},
	}

	return tripayTxRequest{
		Method:        QRISPaymentChannel,
		MerchantRef:   arg.MerchantRef,
		Amount:        arg.TotalAmount,
		CustomerName:  arg.CustomerName,
		CustomerEmail: arg.CustomerEmail,
		Signature:     signature,
		OrderItems:    orderItems,
	}
}

func (p payment) RequestTripayTx(ctx context.Context, tripayTxRequest tripayTxRequest) (tripayTxResponse, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(tripayTxRequest); err != nil {
		return tripayTxResponse{}, fmt.Errorf("failed to encode tripay tx request to json: %w", err)
	}

	tripayURL := "https://tripay.co.id/api-sandbox/transaction/create"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tripayURL, &buf)
	if err != nil {
		return tripayTxResponse{}, fmt.Errorf("failed to new http post request with context: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.configs.Env.TripayAPIKey))

	retryableFunc := func() (tripayTxResponse, error) {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return tripayTxResponse{}, fmt.Errorf("failed to send http post request: %w", err)
		}
		defer res.Body.Close()

		var resBody struct {
			Success bool            `json:"success"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}

		if err = json.NewDecoder(res.Body).Decode(&resBody); err != nil {
			return tripayTxResponse{}, fmt.Errorf("failed to decode tripay tx response: %w", err)
		}

		if !resBody.Success {
			return tripayTxResponse{}, errors.New(resBody.Message)
		}

		var data tripayTxResponse
		if err = json.Unmarshal(resBody.Data, &data); err != nil {
			return tripayTxResponse{}, fmt.Errorf("failed to unmarshal successful tripay tx request: %w", err)
		}

		return data, nil
	}

	return retry.DoWithData(retryableFunc, retry.Attempts(3), retry.LastErrorOnly(true))
}

func (p payment) createCallbackSignature(bytes []byte) string {
	key := []byte(p.configs.Env.TripayPrivateKey)
	hash := hmac.New(sha256.New, key)
	hash.Write(bytes)

	return hex.EncodeToString(hash.Sum(nil))
}

func (p payment) ValidateCallbackSignature(tripaySignature string, reqBody []byte) error {
	signature := p.createCallbackSignature(reqBody)
	if tripaySignature != signature {
		return errors.New("invalid callback signature")
	}
	return nil
}

type ProcessSuccessfulPaymentParams struct {
	InvoiceId  string
	UserId     string
	AmountPaid int32
	Status     string
}

func (p payment) ProcessSuccessfulPayment(ctx context.Context, arg ProcessSuccessfulPaymentParams) error {
	paymentUUID := uuid.New()
	subscriptionUUID := uuid.New()
	invoiceUUID, err := uuid.Parse(arg.InvoiceId)
	if err != nil {
		return fmt.Errorf("failed to parse invoice Id to UUID: %w", err)
	}

	retryableFunc := func(qtx *repository.Queries) error {
		err = qtx.InsertPayment(ctx, repository.InsertPaymentParams{
			ID:         pgtype.UUID{Bytes: paymentUUID, Valid: true},
			UserID:     arg.UserId,
			InvoiceID:  pgtype.UUID{Bytes: invoiceUUID, Valid: true},
			AmountPaid: arg.AmountPaid,
			Status:     strings.ToLower(arg.Status),
		})

		if err != nil {
			return fmt.Errorf("failed to insert payment: %w", err)
		}

		plan, err := qtx.SelectPlanByInvoiceId(ctx, pgtype.UUID{Bytes: invoiceUUID, Valid: true})
		if err != nil {
			return fmt.Errorf("failed to select plan by invoice Id: %w", err)
		}

		startDate := time.Now()
		endDate := startDate.AddDate(0, int(plan.DurationInMonths), 0)

		err = qtx.InsertSubscription(ctx, repository.InsertSubscriptionParams{
			ID:        pgtype.UUID{Bytes: subscriptionUUID, Valid: true},
			UserID:    arg.UserId,
			PlanID:    plan.ID,
			PaymentID: pgtype.UUID{Bytes: paymentUUID, Valid: true},
			StartDate: pgtype.Timestamptz{Time: startDate, Valid: true},
			EndDate:   pgtype.Timestamptz{Time: endDate, Valid: true},
		})

		if err != nil {
			return fmt.Errorf("failed to insert subscription: %w", err)
		}

		return nil
	}

	return dbutil.RetryableTxWithoutData(ctx, p.configs.Db.Conn, p.configs.Db.Queries, retryableFunc)
}

type ProcessUnsuccessfulPaymentParams struct {
	InvoiceId  string
	UserId     string
	AmountPaid int32
	Status     string
}

func (p payment) ProcessUnsuccessfulPayment(ctx context.Context, arg ProcessUnsuccessfulPaymentParams) error {
	paymentUUID := uuid.New()
	invoiceUUID, err := uuid.Parse(arg.InvoiceId)
	if err != nil {
		return fmt.Errorf("failed to parse invoice Id to UUID: %w", err)
	}

	return retryutil.RetryWithoutData(func() error {
		return p.configs.Db.Queries.InsertPayment(ctx, repository.InsertPaymentParams{
			ID:         pgtype.UUID{Bytes: paymentUUID, Valid: true},
			UserID:     arg.UserId,
			InvoiceID:  pgtype.UUID{Bytes: invoiceUUID, Valid: true},
			AmountPaid: arg.AmountPaid,
			Status:     strings.ToLower(arg.Status),
		})
	})
}
