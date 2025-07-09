package utils

import (
	"crypto/rand"
	"math/big"
	"time"
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

// CalculateTTL calculates the TTL timestamp based on duration string
func CalculateTTL(duration string) (int64, int, error) {
	now := time.Now()
	var expiry time.Time
	var codeLength int

	switch duration {
	case "1d":
		expiry = now.Add(24 * time.Hour)
		codeLength = 4
	case "1w":
		expiry = now.Add(7 * 24 * time.Hour)
		codeLength = 5
	case "1m":
		expiry = now.AddDate(0, 1, 0)
		codeLength = 6
	default:
		return 0, 0, nil
	}

	return expiry.Unix(), codeLength, nil
}