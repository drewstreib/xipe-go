package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/drewstreib/xipe-go/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostHandler(t *testing.T) {
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
		// JSON format tests - URL posts
		{
			name:        "JSON: Missing ttl parameter defaults to 1d",
			query:       "",
			body:        `{"data":"https://example.com","typ":"URL"}`,
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
		{
			name:           "JSON: Missing data parameter",
			query:          "",
			body:           `{"ttl":"1d"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "Invalid JSON format or missing required field (data)",
			},
			checkBody: true,
		},
		{
			name:           "JSON: Invalid ttl format",
			query:          "",
			body:           `{"ttl":"2d","data":"https://example.com","typ":"URL"}`,
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
			body:           `{"ttl":"1d","data":"example.com","typ":"URL"}`,
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
			body:           `{"ttl":"1d","data":"http://invalid url with spaces","typ":"URL"}`,
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
			body:           `{"ttl":"1d","data":"https://example.com/` + strings.Repeat("a", 4100) + `","typ":"URL"}`,
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
			body:        `{"ttl":"1d","data":"https://example.com","typ":"URL"}`,
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
			body:        `{"ttl":"1d","data":"https://example.com","typ":"URL"}`,
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

		// JSON format tests - Data posts
		{
			name:           "JSON: Data too long (exceeds 10KB)",
			query:          "",
			body:           `{"ttl":"1d","data":"` + strings.Repeat("a", 10241) + `"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "Data too long (10241 bytes, 10KB max)",
			},
			checkBody: true,
		},
		{
			name:        "JSON: Successful data storage",
			query:       "",
			body:        `{"ttl":"1d","data":"Hello, world!"}`,
			contentType: "application/json",
			userAgent:   "curl/7.68.0",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && r.Val == "Hello, world!" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:        "JSON: Successful data storage with HTML content",
			query:       "",
			body:        `{"ttl":"1d","data":"<script>alert('test')</script><h1>Hello & goodbye</h1>"}`,
			contentType: "application/json",
			userAgent:   "curl/7.68.0",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && r.Val == "<script>alert('test')</script><h1>Hello & goodbye</h1>" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:        "URLEncoded: Successful data storage with HTML content",
			query:       "?input=urlencoded",
			body:        "ttl=1d&data=" + url.QueryEscape("<div>Test HTML & entities</div>"),
			contentType: "application/x-www-form-urlencoded",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && r.Val == "<div>Test HTML & entities</div>" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},

		// URL-encoded format tests
		{
			name:        "URLEncoded: Missing ttl parameter defaults to 1d",
			query:       "?input=urlencoded",
			body:        "data=https%3A%2F%2Fexample.com&typ=URL",
			contentType: "application/x-www-form-urlencoded",
			userAgent:   "curl/7.68.0",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:           "URLEncoded: Missing data parameter",
			query:          "?input=urlencoded",
			body:           "ttl=1d",
			contentType:    "application/x-www-form-urlencoded",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "data parameter is required",
			},
			checkBody: true,
		},
		{
			name:        "URLEncoded: Successful URL storage (browser)",
			query:       "?input=urlencoded",
			body:        "ttl=1d&data=https%3A%2F%2Fexample.com&typ=URL&format=html",
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
				req = httptest.NewRequest("POST", "/api/post"+tt.query, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", tt.contentType)
			} else {
				req = httptest.NewRequest("POST", "/api/post"+tt.query, nil)
			}
			req.Header.Set("User-Agent", tt.userAgent)
			c.Request = req

			h.PostHandler(c)

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
