package handlers

import (
	"net/http"

	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/services"
)

type AuthHandler interface {
	Register(res http.ResponseWriter, req *http.Request)
	Login(res http.ResponseWriter, req *http.Request)
}

type auth struct {
	authService services.AuthServicer
	env         configs.Env
	db          configs.Db
}

func NewAuthHandler(authService services.AuthServicer, env configs.Env, db configs.Db) AuthHandler {
	return &auth{
		authService: authService,
		env:         env,
		db:          db,
	}
}

func (u auth) Register(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Register"))
}

func (u auth) Login(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Login"))
}
