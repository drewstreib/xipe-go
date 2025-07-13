package utils

import (
	"testing"

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
