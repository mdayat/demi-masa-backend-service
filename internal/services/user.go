package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/retryutil"
)

type UserServicer interface {
	ReverseGeocode(ctx context.Context, latitude, longitude float64) (reverseGeocodeResult, error)
}

type user struct {
	configs configs.Configs
}

func NewUserService(configs configs.Configs) UserServicer {
	return &user{
		configs: configs,
	}
}

type reverseGeocodeResult struct {
	City     string
	Timezone string
}

func (u user) ReverseGeocode(ctx context.Context, latitude, longitude float64) (reverseGeocodeResult, error) {
	url := fmt.Sprintf(
		"https://api.geoapify.com/v1/geocode/reverse?lat=%f&lon=%f&format=json&type=city&apiKey=%s",
		latitude,
		longitude,
		u.configs.Env.GeoapifyAPIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return reverseGeocodeResult{}, fmt.Errorf("failed to new get request with context: %w", err)
	}

	retryableFunc := func() (reverseGeocodeResult, error) {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return reverseGeocodeResult{}, fmt.Errorf("failed to send get request: %w", err)
		}
		defer resp.Body.Close()

		var respBody struct {
			Results []struct {
				City     string `json:"city"`
				Timezone struct {
					Name string `json:"name"`
				} `json:"timezone"`
			} `json:"results"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			return reverseGeocodeResult{}, fmt.Errorf("failed to decode reverse geocode result: %w", err)
		}

		if len(respBody.Results) != 0 {
			return reverseGeocodeResult{City: respBody.Results[0].City, Timezone: respBody.Results[0].Timezone.Name}, nil
		}

		return reverseGeocodeResult{}, nil
	}

	return retryutil.RetryWithData(retryableFunc)
}
