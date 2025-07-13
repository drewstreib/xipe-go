package utils

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateCode generates a random alphanumeric code of specified length
func GenerateCode(length int) (string, error) {
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// GenerateUniqueCode generates a code that doesn't conflict with reserved codes
func GenerateUniqueCode(length int) (string, error) {
	for i := 0; i < 5; i++ { // Try up to 5 times
		code, err := GenerateCode(length)
		if err != nil {
			return "", err
		}

		// Check if this code is reserved
		if IsReservedCode(code) {
			continue // Try again
		}

		return code, nil
	}

	// If we get here, we couldn't generate a non-reserved code in 5 tries
	// This should be extremely rare, so just return a regular code
	return GenerateCode(length)
}
