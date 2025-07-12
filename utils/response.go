package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ShouldReturnHTML determines if the response should be HTML based on format parameter or User-Agent
func ShouldReturnHTML(c *gin.Context) bool {
	// Check for explicit format parameter first (highest priority)
	format := strings.ToLower(c.Query("format"))
	if format == "json" {
		return false
	}
	if format == "html" {
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

// RespondWithError sends an error response in JSON or HTML format based on client type
func RespondWithError(c *gin.Context, statusCode int, status, description string) {
	if ShouldReturnHTML(c) {
		c.HTML(statusCode, "error.html", gin.H{
			"status":      status,
			"description": description,
			"statusCode":  statusCode,
		})
	} else {
		c.JSON(statusCode, gin.H{
			"status":      status,
			"description": description,
		})
	}
}
