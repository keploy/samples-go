package main

import (
	"fmt"
	"log"
	"weather-app/pkg/weather"
)

const APIKey = "56bd01265a70326b2712ed64cf1a3c5e"

func main() {
	city := "London"
	weatherData, err := weather.FetchWeatherData(city, APIKey)
	if err != nil {
		log.Fatalf("Failed to fetch weather data: %v", err)
	}

	tempCelsius := weatherData.Main.Temp - 273.15
	fmt.Printf("Weather in %s:\n", city)
	fmt.Printf("Temperature: %.2f°C\n", tempCelsius)
	fmt.Printf("Weather: %s (%s)\n", weatherData.Weather[0].Main, weatherData.Weather[0].Description)
}
