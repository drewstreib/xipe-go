package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize reserved codes for static page testing
	err := utils.InitReservedCodes()
	assert.NoError(t, err)

	tests := []struct {
		name                 string
		handler              string
		code                 string
		setupMock            func(*db.MockDB, *db.MockS3)
		expectedCacheControl string
		expectExpires        bool
		expectNoPragma       bool
	}{
		{
			name:                 "Root page cache headers",
			handler:              "root",
			expectedCacheControl: "public, max-age=3600",
			expectExpires:        true,
		},
		{
			name:    "Static page cache headers",
			handler: "data",
			code:    "privacy", // Assuming privacy is a static page
			setupMock: func(m *db.MockDB, s *db.MockS3) {
				// Static pages won't hit the database
			},
			expectedCacheControl: "public, max-age=3600",
			expectExpires:        true,
			expectNoPragma:       true,
		},
		{
			name:    "Dynamic data page cache headers - short TTL",
			handler: "data",
			code:    "test1",
			setupMock: func(m *db.MockDB, s *db.MockS3) {
				// TTL that expires in 30 minutes (1800 seconds)
				shortTTL := time.Now().Add(30 * time.Minute).Unix()
				m.On("GetRedirect", "test1").Return(&db.RedirectRecord{
					Code:    "test1",
					Typ:     "D",
					Val:     "Test content",
					Ettl:    shortTTL,
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedCacheControl: "public, max-age=1800", // Should use the shorter 30 minutes
			expectExpires:        true,
			expectNoPragma:       true,
		},
		{
			name:    "Dynamic data page cache headers - long TTL",
			handler: "data",
			code:    "test2",
			setupMock: func(m *db.MockDB, s *db.MockS3) {
				// TTL that expires in 2 hours
				longTTL := time.Now().Add(2 * time.Hour).Unix()
				m.On("GetRedirect", "test2").Return(&db.RedirectRecord{
					Code:    "test2",
					Typ:     "D",
					Val:     "Test content",
					Ettl:    longTTL,
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedCacheControl: "public, max-age=3600", // Should use the max 1 hour
			expectExpires:        true,
			expectNoPragma:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			mockS3 := new(db.MockS3)
			if tt.setupMock != nil {
				tt.setupMock(mockDB, mockS3)
			}

			h := &Handlers{DB: mockDB, S3: mockS3}

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*")

			var req *http.Request
			if tt.handler == "root" {
				req = httptest.NewRequest("GET", "/", nil)
				c.Request = req
				h.RootHandler(c)
			} else {
				req = httptest.NewRequest("GET", "/"+tt.code, nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 (browser)")
				c.Request = req
				c.Params = gin.Params{{Key: "code", Value: tt.code}}
				h.DataHandler(c)
			}

			// Check Cache-Control header
			assert.Equal(t, tt.expectedCacheControl, w.Header().Get("Cache-Control"))

			// Check Expires header is set
			if tt.expectExpires {
				expires := w.Header().Get("Expires")
				assert.NotEmpty(t, expires)
				// Parse and verify it's a valid future time
				expiresTime, err := time.Parse(http.TimeFormat, expires)
				assert.NoError(t, err)
				assert.True(t, expiresTime.After(time.Now()))
			}

			// Check Pragma header is cleared
			if tt.expectNoPragma {
				assert.Empty(t, w.Header().Get("Pragma"))
			}

			mockDB.AssertExpectations(t)
			mockS3.AssertExpectations(t)
		})
	}
}

func TestCacheDurationCalculation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ttlOffset      time.Duration
		expectedMaxAge int64
	}{
		{
			name:           "TTL in 30 minutes",
			ttlOffset:      30 * time.Minute,
			expectedMaxAge: 1800, // 30 * 60
		},
		{
			name:           "TTL in 2 hours",
			ttlOffset:      2 * time.Hour,
			expectedMaxAge: 3600, // Capped at 1 hour
		},
		{
			name:           "TTL in 45 minutes",
			ttlOffset:      45 * time.Minute,
			expectedMaxAge: 2700, // 45 * 60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			mockS3 := new(db.MockS3)

			// Set up mock with TTL
			mockTTL := time.Now().Add(tt.ttlOffset).Unix()
			mockDB.On("GetRedirect", "test").Return(&db.RedirectRecord{
				Code:    "test",
				Typ:     "D",
				Val:     "Test content",
				Ettl:    mockTTL,
				Created: time.Now().Unix(),
				Owner:   "owner123",
			}, nil)

			h := &Handlers{DB: mockDB, S3: mockS3}

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*")

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (browser)")
			c.Request = req
			c.Params = gin.Params{{Key: "code", Value: "test"}}

			h.DataHandler(c)

			// Extract max-age from Cache-Control header
			cacheControl := w.Header().Get("Cache-Control")
			assert.Contains(t, cacheControl, "public, max-age=")

			// Parse the max-age value
			maxAgeStr := cacheControl[len("public, max-age="):]
			maxAge, err := strconv.ParseInt(maxAgeStr, 10, 64)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMaxAge, maxAge)

			mockDB.AssertExpectations(t)
			mockS3.AssertExpectations(t)
		})
	}
}
