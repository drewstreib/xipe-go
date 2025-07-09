package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRootHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	
	router.LoadHTMLGlob("../templates/*")
	c.Request = httptest.NewRequest("GET", "/", nil)

	RootHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestStatsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stats", nil)

	StatsHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, "ok", response["status"])
	stats, ok := response["stats"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(0), stats["total_urls"])
	assert.Equal(t, float64(0), stats["total_pastes"])
}