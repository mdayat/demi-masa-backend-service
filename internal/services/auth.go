package services

import (
	"context"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/repository"
	"google.golang.org/api/idtoken"
)

type AuthServicer interface {
	ValidateIDToken(ctx context.Context, idToken string) (*idtoken.Payload, error)
	CheckUserExistence(ctx context.Context, userId string) (bool, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (repository.User, error)
}

type auth struct {
	configs configs.Configs
}

func NewAuthService(configs configs.Configs) AuthServicer {
	return &auth{
		configs: configs,
	}
}

func (a auth) ValidateIDToken(ctx context.Context, idToken string) (*idtoken.Payload, error) {
	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return nil, err
	}

	return validator.Validate(ctx, idToken, a.configs.Env.ClientId)
}

func (a auth) CheckUserExistence(ctx context.Context, userId string) (bool, error) {
	return retry.DoWithData(
		func() (bool, error) {
			return a.configs.Db.Queries.CheckUserExistence(ctx, userId)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

type CreateUserParams struct {
	UserId string
}

func (a auth) CreateUser(ctx context.Context, arg CreateUserParams) (repository.User, error) {
	return retry.DoWithData(
		func() (repository.User, error) {
			return a.configs.Db.Queries.CreateUser(ctx, arg.UserId)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}
