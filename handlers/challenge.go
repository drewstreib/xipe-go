package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleChallengeCheck handles the /challenge-check endpoint
// This endpoint returns "1" as plain text and will be protected by Cloudflare managed challenge
func (h *Handlers) HandleChallengeCheck(c *gin.Context) {
	c.String(http.StatusOK, "1")
}
