package services

import (
	"github.com/mdayat/demi-masa/configs"
)

type AuthServicer interface{}

type auth struct {
	configs configs.Configs
}

func NewAuthService(configs configs.Configs) AuthServicer {
	return &auth{
		configs: configs,
	}
}
