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

// HandleCloudflareTest handles the /cloudflare-test endpoint
// This endpoint serves a test page for the Cloudflare challenge verification
func (h *Handlers) HandleCloudflareTest(c *gin.Context) {
	c.HTML(http.StatusOK, "cloudflare-test.html", nil)
}
