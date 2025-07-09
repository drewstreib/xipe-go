package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"xipe/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestURLPostHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "Missing key parameter",
			query:          "?url=https://example.com",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "key and url parameters are required",
			},
		},
		{
			name:           "Missing url parameter",
			query:          "?key=test1234",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "key and url parameters are required",
			},
		},
		{
			name:           "Invalid key format - too short",
			query:          "?key=abc&url=https://example.com",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "key must be 4-8 alphanumeric characters",
			},
		},
		{
			name:           "Invalid key format - too long",
			query:          "?key=abcdefghi&url=https://example.com",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "key must be 4-8 alphanumeric characters",
			},
		},
		{
			name:           "Invalid key format - special characters",
			query:          "?key=test!234&url=https://example.com",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "key must be 4-8 alphanumeric characters",
			},
		},
		{
			name:  "Successful URL storage",
			query: "?key=test1234&url=https://example.com",
			setupMock: func(m *db.MockDB) {
				m.On("PutURL", "test1234", "https://example.com").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "success",
				"key":    "test1234",
				"url":    "https://example.com",
			},
		},
		{
			name:  "Database error",
			query: "?key=test1234&url=https://example.com",
			setupMock: func(m *db.MockDB) {
				m.On("PutURL", "test1234", "https://example.com").Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "failed to store URL",
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
			c.Request = httptest.NewRequest("GET", "/api/urlpost"+tt.query, nil)

			URLPostHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
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

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"test", true},
		{"test1234", true},
		{"ABCD1234", true},
		{"abc", false},
		{"abcdefghi", false},
		{"test!", false},
		{"test-123", false},
		{"test_123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isValidKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}