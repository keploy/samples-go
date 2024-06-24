package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

func TestVerifyToken(t *testing.T) {
	secretKey := "mySecretKey"
	value := "testValue"

	// Generate a token
	tokenString, err := GenerateToken(value, secretKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Test with an invalid token
	invalidToken := tokenString + "invalid"
	claims, err := VerifyToken(invalidToken, secretKey)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// Test with an expired token
	expiredClaims := CustomClaims{
		Value: value,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Set expiration time to the past
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString([]byte(secretKey))
	assert.NoError(t, err)

	claims, err = VerifyToken(expiredTokenString, secretKey)
	assert.Error(t, err)
	assert.Nil(t, claims)
}
