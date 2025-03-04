package services

import (
	"context"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/repository"
)

type UserServicer interface {
	SelectActiveSubscription(ctx context.Context, userId string) (repository.Subscription, error)
}

type user struct {
	configs configs.Configs
}

func NewUserService(configs configs.Configs) UserServicer {
	return &user{
		configs: configs,
	}
}

func (u user) SelectActiveSubscription(ctx context.Context, userId string) (repository.Subscription, error) {
	return retry.DoWithData(
		func() (repository.Subscription, error) {
			return u.configs.Db.Queries.SelectActiveSubscription(ctx, userId)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}
