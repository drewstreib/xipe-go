package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drewstreib/xipe-go/config"
	"github.com/drewstreib/xipe-go/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostHandlerSimple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Raw body success", func(t *testing.T) {
		mockDB := &db.MockDB{}
		mockS3 := &db.MockS3{}
		cfg := &config.Config{
			PasteTTL:                86400 * 7, // 7 days
			PasteDynamoDBCutoffSize: 10240,     // 10KB
			PasteMaxSize:            2097152,   // 2MB
			CacheMaxItems:           10000,     // 10K items
		}
		h := &Handlers{DB: mockDB, S3: mockS3, Cfg: cfg}

		// Mock successful storage
		mockDB.On("PutRedirect", mock.AnythingOfType("*db.RedirectRecord")).Return(nil)

		// Create router
		r := gin.New()
		r.POST("/", h.PostHandler)

		// Create request with raw body
		req := httptest.NewRequest("POST", "/", strings.NewReader("Hello, world!"))
		req.Header.Set("Content-Type", "text/plain")

		// Create response recorder
		w := httptest.NewRecorder()

		// Serve the request
		r.ServeHTTP(w, req)

		// Check status
		assert.Equal(t, http.StatusOK, w.Code)

		// Check response contains a URL
		body := w.Body.String()
		assert.Contains(t, body, "http://")
		assert.Contains(t, body, "/")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
		mockS3.AssertExpectations(t)
	})

	t.Run("Form input success", func(t *testing.T) {
		mockDB := &db.MockDB{}
		mockS3 := &db.MockS3{}
		cfg := &config.Config{
			PasteTTL:                86400 * 7, // 7 days
			PasteDynamoDBCutoffSize: 10240,     // 10KB
			PasteMaxSize:            2097152,   // 2MB
			CacheMaxItems:           10000,     // 10K items
		}
		h := &Handlers{DB: mockDB, S3: mockS3, Cfg: cfg}

		// Mock successful storage
		mockDB.On("PutRedirect", mock.AnythingOfType("*db.RedirectRecord")).Return(nil)

		// Create router
		r := gin.New()
		r.POST("/", h.PostHandler)

		// Create request with form data
		req := httptest.NewRequest("POST", "/?input=form&html", strings.NewReader("data=Hello%2C+world%21"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Create response recorder
		w := httptest.NewRecorder()

		// Serve the request
		r.ServeHTTP(w, req)

		// Check status (should be redirect)
		assert.Equal(t, http.StatusSeeOther, w.Code)

		// Check response contains redirect location
		location := w.Header().Get("Location")
		assert.Contains(t, location, "/")
		assert.Contains(t, location, "from=success")
		assert.Contains(t, location, "html")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
		mockS3.AssertExpectations(t)
	})
}
