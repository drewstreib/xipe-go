package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drewstreib/xipe-go/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestURLPostHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		body           string
		contentType    string
		userAgent      string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedBody   map[string]interface{}
		checkBody      bool
	}{
		// JSON format tests
		{
			name:           "JSON: Missing ttl parameter",
			query:          "",
			body:           `{"url":"https://example.com"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "Invalid JSON format or missing required fields (ttl, url)",
			},
			checkBody: true,
		},
		{
			name:           "JSON: Missing url parameter",
			query:          "",
			body:           `{"ttl":"1d"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "Invalid JSON format or missing required fields (ttl, url)",
			},
			checkBody: true,
		},
		{
			name:           "JSON: Invalid ttl format",
			query:          "",
			body:           `{"ttl":"2d","url":"https://example.com"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "ttl must be 1d, 1w, or 1mo",
			},
			checkBody: true,
		},
		{
			name:           "JSON: URL without http/https prefix",
			query:          "",
			body:           `{"ttl":"1d","url":"example.com"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "URL must start with http:// or https://",
			},
			checkBody: true,
		},
		{
			name:           "JSON: Invalid URL format",
			query:          "",
			body:           `{"ttl":"1d","url":"http://invalid url with spaces"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "invalid URL format",
			},
			checkBody: true,
		},
		{
			name:           "JSON: URL too long (exceeds 4KB)",
			query:          "",
			body:           `{"ttl":"1d","url":"https://example.com/` + strings.Repeat("a", 4100) + `"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "URL too long (4KB max)",
			},
			checkBody: true,
		},
		{
			name:        "JSON: Successful URL storage with 1d ttl (browser)",
			query:       "",
			body:        `{"ttl":"1d","url":"https://example.com"}`,
			contentType: "application/json",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:        "JSON: API client gets JSON response",
			query:       "",
			body:        `{"ttl":"1d","url":"https://example.com"}`,
			contentType: "application/json",
			userAgent:   "curl/7.68.0",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},

		// URL-encoded format tests
		{
			name:           "URLEncoded: Missing ttl parameter",
			query:          "?input=urlencoded",
			body:           "url=https%3A%2F%2Fexample.com",
			contentType:    "application/x-www-form-urlencoded",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "ttl and url parameters are required",
			},
			checkBody: true,
		},
		{
			name:           "URLEncoded: Missing url parameter",
			query:          "?input=urlencoded",
			body:           "ttl=1d",
			contentType:    "application/x-www-form-urlencoded",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "ttl and url parameters are required",
			},
			checkBody: true,
		},
		{
			name:        "URLEncoded: Successful URL storage (browser)",
			query:       "?input=urlencoded",
			body:        "ttl=1d&url=https%3A%2F%2Fexample.com&format=html",
			contentType: "application/x-www-form-urlencoded",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4
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

			h := &Handlers{DB: mockDB}

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*")

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest("POST", "/api/urlpost"+tt.query, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", tt.contentType)
			} else {
				req = httptest.NewRequest("POST", "/api/urlpost"+tt.query, nil)
			}
			req.Header.Set("User-Agent", tt.userAgent)
			c.Request = req

			h.URLPostHandler(c)

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
