package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const BaseURL = "http://api.openweathermap.org/data/2.5/weather"

type WeatherData struct {
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Main struct {
		Temp     float64 `json:"temp"`
		Pressure int     `json:"pressure"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
}

func FetchWeatherData(city, apiKey string) (WeatherData, error) {
	return FetchWeatherDataWithClient(city, apiKey, http.DefaultClient, context.Background())
}

func FetchWeatherDataWithClient(city, apiKey string, client *http.Client, ctx context.Context) (WeatherData, error) {
	var weatherData WeatherData

	url := fmt.Sprintf("%s?q=%s&appid=%s", BaseURL, city, apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return weatherData, err
	}

	response, err := client.Do(req)
	if err != nil {
		return weatherData, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return weatherData, fmt.Errorf("failed to fetch weather data, status code: %d", response.StatusCode)
	}

	err = json.NewDecoder(response.Body).Decode(&weatherData)
	if err != nil {
		return weatherData, err
	}

	return weatherData, nil
}
