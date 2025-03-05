package services

import (
	"context"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/repository"
)

type PaymentServicer interface {
	SelectActiveInvoice(ctx context.Context, userId string) (repository.Invoice, error)
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
