package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/integrations/kgin/v1"
	"github.com/keploy/go-sdk/keploy"
)

type Weather struct {
	City    string `json:"city"`
	Temp    int    `json:"temp"`
	Weather string `json:"weather"`
}

func fuzzCityName() string {
	cities := []string{
		"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose",
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(cities))

	return cities[index]
}

func randomTemperature(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

func main() {
	port := "8080"

	// Initialize Keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "weather-app",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})

	router := gin.Default()

	// Integrate Keploy with Gin router
	kgin.GinV1(k, router)

	router.GET("/weather", func(c *gin.Context) {
		city := fuzzCityName()
		temp := randomTemperature(30, 100)

		weather := Weather{
			City:    city,
			Temp:    temp,
			Weather: "Partly Cloudy",
		}

		c.JSON(http.StatusOK, weather)
	})

	// Run the server
	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
