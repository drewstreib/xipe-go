package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"xipe/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestURLPostHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedBody   map[string]interface{}
		checkBody      bool
	}{
		{
			name:           "Missing ttl parameter",
			query:          "?url=https://example.com",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "ttl and url parameters are required",
			},
			checkBody: true,
		},
		{
			name:           "Missing url parameter",
			query:          "?ttl=1d",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "ttl and url parameters are required",
			},
			checkBody: true,
		},
		{
			name:           "Invalid ttl format",
			query:          "?ttl=2d&url=https://example.com",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "ttl must be 1d, 1w, or 1m",
			},
			checkBody: true,
		},
		{
			name:           "Invalid URL format",
			query:          "?ttl=1d&url=not-a-url",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "invalid URL format",
			},
			checkBody: true,
		},
		{
			name:  "Successful URL storage with 1d ttl",
			query: "?ttl=1d&url=" + url.QueryEscape("https://example.com"),
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false, // Don't check body since code is random
		},
		{
			name:  "Successful URL storage with 1w ttl",
			query: "?ttl=1w&url=" + url.QueryEscape("https://example.com"),
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 5
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:  "Successful URL storage with 1m ttl",
			query: "?ttl=1m&url=" + url.QueryEscape("https://example.com"),
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 6
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
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
			c.Request.Header.Set("User-Agent", "curl/7.68.0") // Ensure JSON responses in tests

			URLPostHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkBody && tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			mockDB.AssertExpectations(t)
		})
	}
}
