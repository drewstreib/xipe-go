package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLCheck(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		shouldAllow    bool
		reasonContains string
	}{
		{
			name:           "Valid URL format",
			url:            "https://example.com",
			expectedStatus: 200,
			shouldAllow:    true,
			reasonContains: "allowed",
		},
		{
			name:           "Invalid URL format",
			url:            "not-a-url",
			expectedStatus: 400,
			shouldAllow:    false,
			reasonContains: "No hostname found",
		},
		{
			name:           "URL without hostname",
			url:            "https://",
			expectedStatus: 400,
			shouldAllow:    false,
			reasonContains: "No hostname found",
		},
		{
			name:           "URL with path",
			url:            "https://example.com/path/to/resource",
			expectedStatus: 200,
			shouldAllow:    true,
			reasonContains: "allowed",
		},
		{
			name:           "URL with query parameters",
			url:            "https://example.com?param=value",
			expectedStatus: 200,
			shouldAllow:    true,
			reasonContains: "allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLCheck(tt.url)

			assert.Equal(t, tt.shouldAllow, result.Allowed, "Expected allow status mismatch")
			assert.Equal(t, tt.expectedStatus, result.Status, "Expected status code mismatch")
			assert.Contains(t, result.Reason, tt.reasonContains, "Expected reason content mismatch")
		})
	}
}

// Note: These tests check the basic functionality and error handling.
// Testing actual DNS filtering would require either:
// 1. Integration tests with real DNS queries (slow and dependent on external service)
// 2. Mock HTTP client (more complex setup)
// 3. Testing against known blocked domains (not suitable for automated tests)
//
// For now, we focus on the input validation and error handling logic.
