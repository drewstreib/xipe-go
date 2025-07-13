package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ShouldReturnHTML determines if the response should be HTML based on query parameters or User-Agent
func ShouldReturnHTML(c *gin.Context) bool {
	// Check for ?raw parameter first (highest priority - return raw text)
	if c.Request.URL.Query().Has("raw") {
		return false
	}

	// Check for ?html parameter (second priority - return HTML)
	if c.Request.URL.Query().Has("html") {
		return true
	}

	// Fall back to User-Agent detection
	userAgent := strings.ToLower(c.GetHeader("User-Agent"))

	// Check for common browser user agents
	if strings.Contains(userAgent, "mozilla") ||
		strings.Contains(userAgent, "chrome") ||
		strings.Contains(userAgent, "safari") ||
		strings.Contains(userAgent, "firefox") ||
		strings.Contains(userAgent, "edge") ||
		strings.Contains(userAgent, "opera") {
		return true
	}

	// Check for command-line tools that should get JSON
	if strings.Contains(userAgent, "curl") ||
		strings.Contains(userAgent, "wget") ||
		strings.Contains(userAgent, "postman") ||
		strings.Contains(userAgent, "insomnia") {
		return false
	}

	// Default to JSON for unknown agents
	return false
}

// RespondWithError sends an error response in HTML or plain text format based on client type
func RespondWithError(c *gin.Context, statusCode int, status, description string) {
	if ShouldReturnHTML(c) {
		c.HTML(statusCode, "error.html", gin.H{
			"status":      status,
			"description": description,
			"statusCode":  statusCode,
		})
	} else {
		// For non-browser clients, return plain text error
		c.String(statusCode, "Error %d: %s", statusCode, description)
	}
}
