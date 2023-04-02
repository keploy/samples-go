package weather

import (
	"context"
	"errors"
	"testing"

	"net/http"

	"github.com/go-test/deep"
	"github.com/keploy/go-sdk/integrations/khttpclient"
	"github.com/keploy/go-sdk/keploy"
	"github.com/keploy/go-sdk/mock"
)

func TestFetchWeatherData(t *testing.T) {
	apiKey := "56bd01265a70326b2712ed64cf1a3c5e"

	testCases := []struct {
		name           string
		city           string
		expectedResult WeatherData
		expectedError  error
	}{
		{
			name: "successful weather data fetch",
			city: "New York",
			expectedResult: WeatherData{
				Weather: []struct {
					ID          int    `json:"id"`
					Main        string `json:"main"`
					Description string `json:"description"`
					Icon        string `json:"icon"`
				}{
					{
						Description: "clear sky",
					},
				},
			},
			expectedError: nil,
		},
		{
			name:           "failed weather data fetch, invalid city",
			city:           "InvalidCity",
			expectedResult: WeatherData{},
			expectedError:  errors.New("failed to fetch weather data, status code: 404"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = mock.NewContext(mock.Config{
				Mode:      keploy.MODE_RECORD,
				Name:      testCase.name,
				CTX:       ctx,
				Path:      "./",
				OverWrite: false,
				Remove:    []string{},
				Replace:   map[string]string{},
			})

			interceptor := khttpclient.NewInterceptor(http.DefaultTransport)
			client := http.Client{
				Transport: interceptor,
			}

			weatherData, err := FetchWeatherDataWithClient(testCase.city, apiKey, &client, ctx)

			if deep.Equal(testCase.expectedError, err) != nil {
				t.Errorf("expected error: %v, got: %v", testCase.expectedError, err)
			}

			if len(weatherData.Weather) > 0 && testCase.expectedResult.Weather[0].Description != weatherData.Weather[0].Description {
				t.Errorf("expected weather description: %v, got: %v", testCase.expectedResult.Weather[0].Description, weatherData.Weather[0].Description)
			}
		})
	}
}
