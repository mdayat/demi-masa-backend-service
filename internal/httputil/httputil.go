package httputil

import (
	"net/http"

	"github.com/goccy/go-json"

	"github.com/go-playground/validator/v10"
)

func DecodeAndValidate(req *http.Request, validate *validator.Validate, v interface{}) error {
	if err := json.NewDecoder(req.Body).Decode(v); err != nil {
		return err
	}

	if err := validate.Struct(v); err != nil {
		return err
	}

	return nil
}
