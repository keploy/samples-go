// Package uss provides utility functions for generating short, unique URL links by hashing and encoding input URLs
package uss

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/itchyny/base58-go"
)

func GenerateShortLink(initialLink string) string {
	hash := sha256Of(initialLink)
	truncated := hash[:6]
	return base58Encoded(truncated)
}

func sha256Of(input string) []byte {
	algorithm := sha256.New()
	algorithm.Write([]byte(input))
	return algorithm.Sum(nil)
}

func base58Encoded(bytes []byte) string {
	encoding := base58.BitcoinEncoding
	encoded, _ := encoding.Encode(bytes)
	return string(encoded)
}
