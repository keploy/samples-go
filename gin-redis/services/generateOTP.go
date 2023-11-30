package services

import (
	"math/rand"
	"time"
)

func GenerateOTP() int {
	source := rand.NewSource(time.Now().UnixNano())
	randomGenerator := rand.New(source)
	randomNumber := randomGenerator.Intn(9000) + 1000
	return randomNumber
}
