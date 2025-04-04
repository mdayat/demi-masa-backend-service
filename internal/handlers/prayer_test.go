package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
)

func TestPrayerHandlers(t *testing.T) {
	ctx := context.TODO()
	now := time.Now()
	var prayer dtos.PrayerResponse

	getPrayersTable := []struct {
		name            string
		yearQueryParam  string
		monthQueryParam string
		dayQueryParam   string
		expectedStatus  int
	}{
		{
			name:            "GetPrayers/Success",
			yearQueryParam:  fmt.Sprintf("%d", now.Year()),
			monthQueryParam: fmt.Sprintf("%d", now.Month()),
			dayQueryParam:   fmt.Sprintf("%d", now.Day()),
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "GetPrayers/Bad Request (year query param)",
			yearQueryParam:  "",
			monthQueryParam: fmt.Sprintf("%d", now.Month()),
			dayQueryParam:   fmt.Sprintf("%d", now.Day()),
			expectedStatus:  http.StatusBadRequest,
		},
		{
			name:            "GetPrayers/Bad Request (month query param)",
			yearQueryParam:  fmt.Sprintf("%d", now.Year()),
			monthQueryParam: "",
			dayQueryParam:   fmt.Sprintf("%d", now.Day()),
			expectedStatus:  http.StatusBadRequest,
		},
	}

	for _, v := range getPrayersTable {
		t.Run(v.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/prayers?year=%s&month=%s&day=%s", testServer.URL, v.yearQueryParam, v.monthQueryParam, v.dayQueryParam)
			res, err := testClient.Get(url)
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != v.expectedStatus {
				t.Fatalf("expected status %d, got %d", v.expectedStatus, res.StatusCode)
			}

			if v.expectedStatus == http.StatusOK {
				var resBody []dtos.PrayerResponse
				if err = json.NewDecoder(res.Body).Decode(&resBody); err != nil {
					t.Fatalf("unexpected response body: %v", res)
				}

				if len(resBody) != 1 {
					t.Fatalf("expected 1 prayer, got %d", len(resBody))
				}

				prayer = resBody[0]
			}
		})
	}

	updatePrayerTable := []struct {
		name           string
		prayerId       string
		reqBody        string
		expectedStatus int
		expectedResult dtos.PrayerResponse
	}{
		{
			name:           "UpdatePrayer/Success",
			prayerId:       prayer.Id,
			reqBody:        `{"status": "on_time"}`,
			expectedStatus: http.StatusOK,
			expectedResult: dtos.PrayerResponse{
				Id:     prayer.Id,
				Name:   prayer.Name,
				Status: "on_time",
				Year:   prayer.Year,
				Month:  prayer.Month,
				Day:    prayer.Day,
			},
		},
		{
			name:           "UpdatePrayer/Success (no update performed)",
			prayerId:       prayer.Id,
			reqBody:        `{}`,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "UpdatePrayer/Bad Request (invalid status)",
			prayerId:       prayer.Id,
			reqBody:        `{"status": "invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "UpdatePrayer/Not Found",
			prayerId:       uuid.NewString(),
			reqBody:        `{"status": "on_time"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, v := range updatePrayerTable {
		t.Run(v.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/prayers/%s", testServer.URL, v.prayerId)
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer([]byte(v.reqBody)))
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}

			res, err := testClient.Do(req)
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != v.expectedStatus {
				t.Fatalf("expected status %d, got %d", v.expectedStatus, res.StatusCode)
			}

			if v.expectedStatus == http.StatusOK {
				var updatedPrayer dtos.PrayerResponse
				if err = json.NewDecoder(res.Body).Decode(&updatedPrayer); err != nil {
					t.Fatalf("unexpected response body: %v", res)
				}

				if diff := cmp.Diff(v.expectedResult, updatedPrayer); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
