package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleChallengeCheck handles the /challenge-check endpoint
// This endpoint will be protected by Cloudflare managed challenge
func (h *Handlers) HandleChallengeCheck(c *gin.Context) {
	// If this page loads, the challenge was passed
	// Send HTML that notifies parent window and closes tab
	html := `<!DOCTYPE html>
<html>
<head><title>Challenge Passed</title></head>
<body>
<p>Challenge completed successfully!</p>
<script>
if (window.opener) {
    window.opener.postMessage({type: 'challenge-completed'}, '*');
}
setTimeout(() => window.close(), 500);
</script>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// HandleCloudflareTest handles the /cloudflare-test endpoint
// This endpoint serves a test page for the Cloudflare challenge verification
func (h *Handlers) HandleCloudflareTest(c *gin.Context) {
	c.HTML(http.StatusOK, "cloudflare-test.html", nil)
}
