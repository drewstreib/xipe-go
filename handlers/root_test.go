package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drewstreib/xipe-go/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRootHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handlers{DB: nil} // No DB needed for root handler

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	router.LoadHTMLGlob("../templates/*")
	c.Request = httptest.NewRequest("GET", "/", nil)

	h.RootHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestStatsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := new(db.MockDB)
	mockDB.On("GetCacheSize").Return(5)

	h := &Handlers{DB: mockDB}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stats", nil)

	h.StatsHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	stats, ok := response["stats"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(5), stats["cached_items"])

	mockDB.AssertExpectations(t)
}
