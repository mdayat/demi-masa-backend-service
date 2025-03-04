package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/repository"
)

type PrayerServicer interface {
	ValidateYearAndMonthParams(yearString, monthString string) (int, int, error)
	SelectPrayers(ctx context.Context, arg repository.SelectPrayersParams) ([]repository.Prayer, error)
}

type prayer struct {
	configs configs.Configs
}

func NewPrayerService(configs configs.Configs) PrayerServicer {
	return &prayer{
		configs: configs,
	}
}

func (p prayer) ValidateYearAndMonthParams(yearString, monthString string) (int, int, error) {
	if yearString == "" {
		return 0, 0, errors.New("empty year query params")
	}

	if monthString == "" {
		return 0, 0, errors.New("empty month query params")
	}

	year, err := strconv.Atoi(yearString)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert year string to int: %w", err)
	}

	month, err := strconv.Atoi(monthString)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert month string to int: %w", err)
	}

	return year, month, nil
}

func (p prayer) SelectPrayers(ctx context.Context, arg repository.SelectPrayersParams) ([]repository.Prayer, error) {
	return retry.DoWithData(
		func() ([]repository.Prayer, error) {
			return p.configs.Db.Queries.SelectPrayers(ctx, arg)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}
