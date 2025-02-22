package services

import "github.com/mdayat/demi-masa/configs"

type AuthServicer interface{}

type auth struct {
	env configs.Env
	db  configs.Db
}

func NewAuthService(env configs.Env, db configs.Db) AuthServicer {
	return &auth{
		env: env,
		db:  db,
	}
}
