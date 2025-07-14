package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drewstreib/xipe-go/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleChallengeCheck(t *testing.T) {
	// Set up test router
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Create handlers with minimal config
	h := &Handlers{
		Cfg: &config.Config{},
	}

	// Register route
	r.GET("/challenge-check", h.HandleChallengeCheck)

	// Create test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/challenge-check", nil)

	// Execute request
	r.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1", w.Body.String())
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
}
