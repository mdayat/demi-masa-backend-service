package services

import (
	"github.com/mdayat/demi-masa/configs"
)

type UserServicer interface{}

type user struct {
	configs configs.Configs
}

func NewUserService(configs configs.Configs) UserServicer {
	return &user{
		configs: configs,
	}
}
