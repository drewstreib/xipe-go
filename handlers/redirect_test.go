package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"xipe/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRedirectHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		key            string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedHeader string
		expectedBody   map[string]interface{}
	}{
		{
			name: "Valid key - successful redirect",
			key:  "test1234",
			setupMock: func(m *db.MockDB) {
				m.On("GetURL", "test1234").Return("https://example.com", nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			expectedHeader: "https://example.com",
			expectedBody:   nil,
		},
		{
			name: "Valid key - not found",
			key:  "notfound",
			setupMock: func(m *db.MockDB) {
				m.On("GetURL", "notfound").Return("", nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: map[string]interface{}{
				"error": "key not found",
			},
		},
		{
			name: "Database error",
			key:  "test1234",
			setupMock: func(m *db.MockDB) {
				m.On("GetURL", "test1234").Return("", errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedHeader: "",
			expectedBody: map[string]interface{}{
				"error": "failed to retrieve URL",
			},
		},
		{
			name:           "Invalid key format",
			key:            "test!",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
			expectedBody: map[string]interface{}{
				"error": "invalid key format",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)
			db.DB = mockDB

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/"+tt.key, nil)
			c.Params = []gin.Param{{Key: "key", Value: tt.key}}

			RedirectHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Location"))
			}

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := c.ShouldBindJSON(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestCatchAllHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedHeader string
		expectedBody   map[string]interface{}
	}{
		{
			name: "Valid short code",
			path: "/test1234",
			setupMock: func(m *db.MockDB) {
				m.On("GetURL", "test1234").Return("https://example.com", nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			expectedHeader: "https://example.com",
			expectedBody:   nil,
		},
		{
			name: "Invalid path - too short",
			path: "/abc",
			setupMock: func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: map[string]interface{}{
				"error": "not found",
			},
		},
		{
			name: "Invalid path - too long",
			path: "/abcdefghi",
			setupMock: func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: map[string]interface{}{
				"error": "not found",
			},
		},
		{
			name: "Invalid path - special characters",
			path: "/test!234",
			setupMock: func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: map[string]interface{}{
				"error": "not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)
			db.DB = mockDB

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", tt.path, nil)

			CatchAllHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Location"))
			}

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := c.ShouldBindJSON(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			mockDB.AssertExpectations(t)
		})
	}
}