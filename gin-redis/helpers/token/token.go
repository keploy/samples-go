package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type CustomClaims struct {
	// You can define custom fields here
	Value string `json:"value"`
	jwt.StandardClaims
}

func GenerateToken(value string, secretKey string) (string, error) {
	// Create custom claims with an expiration time
	claims := CustomClaims{
		Value: value,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(4 * time.Hour).Unix(), // Set expiration time
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyToken(tokenString string, secretKey string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
