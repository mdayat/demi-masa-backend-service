package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/goccy/go-json"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
)

type UserServicer interface {
	ReverseGeocode(ctx context.Context, latitude, longitude string) (reverseGeocodeResult, error)
	ParseStringCoordinates(latitudeString, longitudeString string) (float64, float64, error)
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

func (u user) ReverseGeocode(ctx context.Context, latitude, longitude string) (reverseGeocodeResult, error) {
	url := fmt.Sprintf(
		"https://api.geoapify.com/v1/geocode/reverse?lat=%s&lon=%s&format=json&type=city&apiKey=%s",
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

		var result reverseGeocodeResult
		if len(respBody.Results) != 0 {
			result.City = respBody.Results[0].City
			result.Timezone = respBody.Results[0].Timezone.Name
		}

		if result.City == "" || result.Timezone == "" {
			return result, errors.New("empty reverse geocode result")
		}

		return result, nil
	}

	return retryutil.RetryWithData(retryableFunc)
}

func (u user) ParseStringCoordinates(latitudeString, longitudeString string) (float64, float64, error) {
	latitude, err := strconv.ParseFloat(latitudeString, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse latitude string to float64: %w", err)
	}

	longitude, err := strconv.ParseFloat(longitudeString, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse longitude string to float64: %w", err)
	}

	return latitude, longitude, nil
}
