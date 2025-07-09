package utils

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestGenerateCode(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"4 char code", 4},
		{"5 char code", 5},
		{"6 char code", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateCode(tt.length)
			assert.NoError(t, err)
			assert.Len(t, code, tt.length)
			
			// Verify all characters are alphanumeric
			for _, c := range code {
				assert.True(t, (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))
			}
		})
	}

	// Test uniqueness
	t.Run("generates unique codes", func(t *testing.T) {
		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code, err := GenerateCode(4)
			assert.NoError(t, err)
			assert.False(t, codes[code], "Generated duplicate code: %s", code)
			codes[code] = true
		}
	})
}

func TestCalculateTTL(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		duration     string
		expectedLen  int
		minDiff      time.Duration
		maxDiff      time.Duration
	}{
		{
			name:        "1 day TTL",
			duration:    "1d",
			expectedLen: 4,
			minDiff:     23 * time.Hour,
			maxDiff:     25 * time.Hour,
		},
		{
			name:        "1 week TTL",
			duration:    "1w",
			expectedLen: 5,
			minDiff:     6 * 24 * time.Hour,
			maxDiff:     8 * 24 * time.Hour,
		},
		{
			name:        "1 month TTL",
			duration:    "1m",
			expectedLen: 6,
			minDiff:     28 * 24 * time.Hour,
			maxDiff:     32 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl, codeLen, err := CalculateTTL(tt.duration)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLen, codeLen)

			// Check TTL is within expected range
			ttlTime := time.Unix(ttl, 0)
			diff := ttlTime.Sub(now)
			assert.True(t, diff >= tt.minDiff, "TTL too short: %v", diff)
			assert.True(t, diff <= tt.maxDiff, "TTL too long: %v", diff)
		})
	}

	t.Run("invalid duration", func(t *testing.T) {
		ttl, codeLen, err := CalculateTTL("2d")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), ttl)
		assert.Equal(t, 0, codeLen)
	})
}