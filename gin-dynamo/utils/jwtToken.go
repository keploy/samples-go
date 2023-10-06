package utils

import (
	"user-onboarding/config"

	"github.com/dgrijalva/jwt-go"
)

func GenerateToken() (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	tokenString, err := token.SignedString([]byte(config.Get().Token))
	return tokenString, err
}

func VerifyToken(token string) bool {
	claims := jwt.MapClaims{}
	validToken, _ := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Get().Token), nil
	})
	return validToken.Valid
}
