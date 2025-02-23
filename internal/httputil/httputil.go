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

	return validate.Struct(v)
}

type SendSuccessResponseParams struct {
	StatusCode int
	ResBody    interface{}
}

func SendSuccessResponse(res http.ResponseWriter, params SendSuccessResponseParams) error {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(params.StatusCode)
	return json.NewEncoder(res).Encode(params.ResBody)
}
