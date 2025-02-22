package configs

import "github.com/go-playground/validator/v10"

type Configs struct {
	Env      Env
	Db       Db
	Validate *validator.Validate
}
