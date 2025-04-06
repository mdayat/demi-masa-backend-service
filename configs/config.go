package configs

import "github.com/go-playground/validator/v10"

type Configs struct {
	Env      Env
	Db       Db
	Validate *validator.Validate
}

func NewConfigs(env Env, db Db) Configs {
	return Configs{
		Env:      env,
		Db:       db,
		Validate: NewValidate(),
	}
}
