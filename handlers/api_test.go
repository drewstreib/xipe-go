package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/drewstreib/xipe-go/db"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4 && r.Owner != ""
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
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4 && r.Owner != ""
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
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4 && r.Owner != ""
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},

		// JSON format tests - Data posts
		{
			name:           "JSON: Data too long (exceeds 50KB)",
			query:          "",
			body:           `{"ttl":"1d","data":"` + strings.Repeat("a", 51201) + `"}`,
			contentType:    "application/json",
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "Data too long (51201 bytes, 50KB max)",
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
					return r.Typ == "D" && r.Val == "Hello, world!" && len(r.Code) == 4 && r.Owner != ""
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
					return r.Typ == "D" && r.Val == "<script>alert('test')</script><h1>Hello & goodbye</h1>" && len(r.Code) == 4 && r.Owner != ""
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
					return r.Typ == "D" && r.Val == "<div>Test HTML & entities</div>" && len(r.Code) == 4 && r.Owner != ""
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
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4 && r.Owner != ""
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
					return r.Typ == "R" && r.Val == "https://example.com" && len(r.Code) == 4 && r.Owner != ""
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
				req = httptest.NewRequest("POST", "/"+tt.query, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", tt.contentType)
			} else {
				req = httptest.NewRequest("POST", "/"+tt.query, nil)
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

func TestDeleteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		code           string
		ownerID        string
		setCookie      bool
		userAgent      string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedBody   map[string]interface{}
		checkBody      bool
		checkLocation  string
	}{
		{
			name:      "Successful delete with valid owner (API client)",
			code:      "abc123",
			ownerID:   "validOwner123",
			setCookie: true,
			userAgent: "curl/7.68.0",
			setupMock: func(m *db.MockDB) {
				m.On("DeleteRedirect", "abc123", "validOwner123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  "ok",
				"message": "deleted successfully",
			},
			checkBody: true,
		},
		{
			name:      "Successful delete with valid owner (browser redirect)",
			code:      "abc123",
			ownerID:   "validOwner123",
			setCookie: true,
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			setupMock: func(m *db.MockDB) {
				m.On("DeleteRedirect", "abc123", "validOwner123").Return(nil)
			},
			expectedStatus: http.StatusOK, // Gin test bug: should be 301 but records as 200
			checkBody:      false,
			checkLocation:  "/abc123",
		},
		{
			name:           "Delete without cookie",
			code:           "abc123",
			setCookie:      false,
			userAgent:      "curl/7.68.0",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "unauthorized",
			},
			checkBody: true,
		},
		{
			name:      "Delete with wrong owner",
			code:      "abc123",
			ownerID:   "wrongOwner",
			setCookie: true,
			userAgent: "curl/7.68.0",
			setupMock: func(m *db.MockDB) {
				m.On("DeleteRedirect", "abc123", "wrongOwner").Return(&types.ConditionalCheckFailedException{})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"status":      "error",
				"description": "unauthorized",
			},
			checkBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)

			h := &Handlers{DB: mockDB}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest("DELETE", "/"+tt.code, nil)
			req.Header.Set("User-Agent", tt.userAgent)

			if tt.setCookie && tt.ownerID != "" {
				req.AddCookie(&http.Cookie{Name: "id", Value: tt.ownerID})
			}

			c.Request = req
			c.Params = gin.Params{{Key: "code", Value: tt.code}}

			h.DeleteHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkLocation != "" {
				assert.Equal(t, tt.checkLocation, w.Header().Get("Location"))
			}

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

func TestPutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           string
		setupMock      func(*db.MockDB)
		expectedStatus int
		checkResponse  func(t *testing.T, response string)
	}{
		{
			name: "Successful PUT with valid UTF-8 text",
			body: "Hello, world! 👋",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && r.Val == "Hello, world! 👋" && len(r.Code) == 4 && r.Owner != ""
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response string) {
				assert.Contains(t, response, "http://")
				assert.Contains(t, response, "/")
				// Strip newline and check code length
				code := strings.TrimSpace(strings.Split(response, "/")[len(strings.Split(response, "/"))-1])
				assert.Equal(t, 4, len(code))
			},
		},
		{
			name:           "Invalid UTF-8 input",
			body:           string([]byte{0xff, 0xfe, 0xfd}), // Invalid UTF-8
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response string) {
				assert.Equal(t, "Error: Input text must be UTF-8\n", response)
			},
		},
		{
			name:           "Empty content",
			body:           "",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response string) {
				assert.Equal(t, "Error: Cannot store empty content\n", response)
			},
		},
		{
			name: "Whitespace-only content",
			body: "   \n\t   ",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && r.Val == "   \n\t   " && len(r.Code) == 4 && r.Owner != ""
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response string) {
				assert.Contains(t, response, "http://")
			},
		},
		{
			name: "Large input gets truncated",
			body: strings.Repeat("a", 60000), // 60KB
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && len(r.Val) <= 51200 && r.Owner != ""
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response string) {
				assert.Contains(t, response, "http://")
			},
		},
		{
			name: "Database error during store",
			body: "Valid content",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.AnythingOfType("*db.RedirectRecord")).Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, response string) {
				assert.Equal(t, "Error: Failed to store data\n", response)
			},
		},
		{
			name: "Multiple collision retries then success",
			body: "Test content",
			setupMock: func(m *db.MockDB) {
				// First 3 attempts fail with conditional check (collision)
				m.On("PutRedirect", mock.AnythingOfType("*db.RedirectRecord")).Return(&types.ConditionalCheckFailedException{}).Times(3)
				// 4th attempt succeeds
				m.On("PutRedirect", mock.AnythingOfType("*db.RedirectRecord")).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response string) {
				assert.Contains(t, response, "http://")
			},
		},
		{
			name: "All collision retries fail",
			body: "Test content",
			setupMock: func(m *db.MockDB) {
				// All 5 attempts fail with conditional check (collision)
				m.On("PutRedirect", mock.AnythingOfType("*db.RedirectRecord")).Return(&types.ConditionalCheckFailedException{}).Times(5)
			},
			expectedStatus: 529,
			checkResponse: func(t *testing.T, response string) {
				assert.Equal(t, "Error: Could not allocate URL in the target namespace.\n", response)
			},
		},
		{
			name: "Single character input",
			body: "a",
			setupMock: func(m *db.MockDB) {
				m.On("PutRedirect", mock.MatchedBy(func(r *db.RedirectRecord) bool {
					return r.Typ == "D" && r.Val == "a" && len(r.Code) == 4 && r.Owner != ""
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response string) {
				assert.Contains(t, response, "http://")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DB
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)

			// Create handler with mock DB
			h := &Handlers{DB: mockDB}

			// Create router
			r := gin.New()
			r.PUT("/", h.PutHandler)

			// Create request
			req := httptest.NewRequest("PUT", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "text/plain")

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve the request
			r.ServeHTTP(w, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}

			// Verify mock expectations
			mockDB.AssertExpectations(t)
		})
	}
}
