package services

import (
	"context"

	"github.com/mdayat/demi-masa/configs"
	"google.golang.org/api/idtoken"
)

type AuthServicer interface {
	ValidateIDToken(ctx context.Context, idToken string) (*idtoken.Payload, error)
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

	payload, err := validator.Validate(ctx, idToken, a.configs.Env.ClientId)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
