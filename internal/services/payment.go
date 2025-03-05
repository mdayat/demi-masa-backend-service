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

	"github.com/goccy/go-json"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/repository"
)

type PaymentServicer interface {
	SelectActiveInvoice(ctx context.Context, userId string) (repository.Invoice, error)
	DecrementCouponQuota(ctx context.Context, code string) (int64, error)
	IncrementCouponQuota(ctx context.Context, code string) error
	CreateTripayTxRequest(arg CreateTripayTxRequestParams) tripayTxRequest
	RequestTripayTx(ctx context.Context, tripayTxRequest tripayTxRequest) (tripayTxResponse, error)
	InsertInvoice(ctx context.Context, arg repository.InsertInvoiceParams) error
}

type payment struct {
	configs configs.Configs
}

func NewPaymentService(configs configs.Configs) PaymentServicer {
	return &payment{
		configs: configs,
	}
}

func (p payment) SelectActiveInvoice(ctx context.Context, userId string) (repository.Invoice, error) {
	return retry.DoWithData(
		func() (repository.Invoice, error) {
			return p.configs.Db.Queries.SelectActiveInvoice(ctx, userId)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

func (p payment) DecrementCouponQuota(ctx context.Context, code string) (int64, error) {
	return retry.DoWithData(
		func() (int64, error) {
			return p.configs.Db.Queries.DecrementCouponQuota(ctx, code)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

func (p payment) IncrementCouponQuota(ctx context.Context, code string) error {
	return retry.Do(
		func() error {
			return p.configs.Db.Queries.IncrementCouponQuota(ctx, code)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

func createSignature(tripayPrivateKey, tripayMerchantCode, merchantRef string, amount int) string {
	key := []byte(tripayPrivateKey)
	message := fmt.Sprintf("%s%s%d", tripayMerchantCode, merchantRef, amount)

	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(message))

	return hex.EncodeToString(hash.Sum(nil))
}

var QRISPaymentChannel = "QRIS"

type orderItem struct {
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
	PlanName      string
	PlanPrice     int
}

func (p payment) CreateTripayTxRequest(arg CreateTripayTxRequestParams) tripayTxRequest {
	signature := createSignature(p.configs.Env.TripayPrivateKey, p.configs.Env.TripayMerchantCode, arg.MerchantRef, arg.TotalAmount)
	orderItems := []orderItem{
		{
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

func (p payment) InsertInvoice(ctx context.Context, arg repository.InsertInvoiceParams) error {
	return retry.Do(
		func() error {
			return p.configs.Db.Queries.InsertInvoice(ctx, arg)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}
