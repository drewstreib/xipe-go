package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drewstreib/xipe-go/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDataHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		code           string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedHeader string
		expectedBody   map[string]interface{}
	}{
		{
			name: "Valid code - shows info page",
			code: "test",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test").Return(&db.RedirectRecord{
					Code:    "test",
					Typ:     "R",
					Val:     "https://example.com",
					Created: 1234567890,
					Ettl:    1234567890,
					IP:      "192.168.1.1",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "",
			expectedBody:   nil, // Returns HTML info page
		},
		{
			name: "Valid code - not found",
			code: "nope",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "nope").Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody:   nil, // Now returns HTML, don't check body
		},
		{
			name: "Database error",
			code: "test",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedHeader: "",
			expectedBody:   nil, // Now returns HTML, don't check body
		},
		{
			name:           "Invalid code format",
			code:           "test!",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
			expectedBody:   nil, // Now returns HTML, don't check body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)

			h := &Handlers{DB: mockDB}

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*") // Load templates for HTML responses
			c.Request = httptest.NewRequest("GET", "/"+tt.code, nil)
			c.Params = []gin.Param{{Key: "code", Value: tt.code}}

			h.DataHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Location"))
			}

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
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
			name: "Valid 4-char code",
			path: "/test",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test").Return(&db.RedirectRecord{
					Code:    "test",
					Typ:     "R",
					Val:     "https://example.com",
					Created: 1234567890,
					Ettl:    1234567890,
					IP:      "192.168.1.1",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "",
			expectedBody:   nil, // Returns HTML info page
		},
		{
			name: "Valid 5-char code",
			path: "/test5",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test5").Return(&db.RedirectRecord{
					Code:    "test5",
					Typ:     "R",
					Val:     "https://example.com",
					Created: 1234567890,
					Ettl:    1234567890,
					IP:      "192.168.1.1",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "",
			expectedBody:   nil, // Returns HTML info page
		},
		{
			name: "Valid 6-char code",
			path: "/test66",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test66").Return(&db.RedirectRecord{
					Code:    "test66",
					Typ:     "R",
					Val:     "https://example.com",
					Created: 1234567890,
					Ettl:    1234567890,
					IP:      "192.168.1.1",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "",
			expectedBody:   nil, // Returns HTML info page
		},
		{
			name:           "Invalid path - too short",
			path:           "/abc",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody:   nil, // Now returns HTML, don't check body
		},
		{
			name:           "Invalid path - too long",
			path:           "/abcdefg",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody:   nil, // Now returns HTML, don't check body
		},
		{
			name:           "Invalid path - special characters",
			path:           "/test!234",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody:   nil, // Now returns HTML, don't check body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)

			h := &Handlers{DB: mockDB}

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*") // Load templates for HTML responses
			c.Request = httptest.NewRequest("GET", tt.path, nil)

			h.CatchAllHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Location"))
			}

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestDataHandlerBranching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		code           string
		userAgent      string
		redirect       *db.RedirectRecord
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedBody   string
		expectHTML     bool
	}{
		{
			name:      "API client (curl) - URL redirect returns raw URL",
			code:      "test1",
			userAgent: "curl/7.68.0",
			redirect: &db.RedirectRecord{
				Code:    "test1",
				Typ:     "R",
				Val:     "https://example.com",
				Created: time.Now().Unix(),
				Owner:   "owner123",
			},
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test1").Return(&db.RedirectRecord{
					Code:    "test1",
					Typ:     "R",
					Val:     "https://example.com",
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "https://example.com",
			expectHTML:     false,
		},
		{
			name:      "API client (curl) - Data returns raw data",
			code:      "test2",
			userAgent: "curl/7.68.0",
			redirect: &db.RedirectRecord{
				Code:    "test2",
				Typ:     "D",
				Val:     "Hello, world!\nLine 2",
				Created: time.Now().Unix(),
				Owner:   "owner123",
			},
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test2").Return(&db.RedirectRecord{
					Code:    "test2",
					Typ:     "D",
					Val:     "Hello, world!\nLine 2",
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Hello, world!\nLine 2",
			expectHTML:     false,
		},
		{
			name:      "Browser client - URL redirect returns HTML",
			code:      "test3",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			redirect: &db.RedirectRecord{
				Code:    "test3",
				Typ:     "R",
				Val:     "https://example.com",
				Created: time.Now().Unix(),
				Owner:   "owner123",
			},
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test3").Return(&db.RedirectRecord{
					Code:    "test3",
					Typ:     "R",
					Val:     "https://example.com",
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "", // Don't check HTML content exactly
			expectHTML:     true,
		},
		{
			name:      "Browser client - Data returns HTML",
			code:      "test4",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			redirect: &db.RedirectRecord{
				Code:    "test4",
				Typ:     "D",
				Val:     "console.log('hello');",
				Created: time.Now().Unix(),
				Owner:   "owner123",
			},
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test4").Return(&db.RedirectRecord{
					Code:    "test4",
					Typ:     "D",
					Val:     "console.log('hello');",
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "", // Don't check HTML content exactly
			expectHTML:     true,
		},
		{
			name:      "Unknown client defaults to plain text - URL",
			code:      "test5",
			userAgent: "SomeUnknownBot/1.0",
			redirect: &db.RedirectRecord{
				Code:    "test5",
				Typ:     "R",
				Val:     "https://example.com/path",
				Created: time.Now().Unix(),
				Owner:   "owner123",
			},
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test5").Return(&db.RedirectRecord{
					Code:    "test5",
					Typ:     "R",
					Val:     "https://example.com/path",
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "https://example.com/path",
			expectHTML:     false,
		},
		{
			name:      "Unknown client defaults to plain text - Data",
			code:      "test6",
			userAgent: "SomeUnknownBot/1.0",
			redirect: &db.RedirectRecord{
				Code:    "test6",
				Typ:     "D",
				Val:     "Plain text data",
				Created: time.Now().Unix(),
				Owner:   "owner123",
			},
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test6").Return(&db.RedirectRecord{
					Code:    "test6",
					Typ:     "D",
					Val:     "Plain text data",
					Created: time.Now().Unix(),
					Owner:   "owner123",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Plain text data",
			expectHTML:     false,
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

			req := httptest.NewRequest("GET", "/"+tt.code, nil)
			req.Header.Set("User-Agent", tt.userAgent)
			c.Request = req
			c.Params = gin.Params{{Key: "code", Value: tt.code}}

			h.DataHandler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectHTML {
				// For HTML responses, just check content type and that it contains HTML
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
				body := w.Body.String()
				assert.Contains(t, body, "<html")
				assert.Contains(t, body, "</html>")
			} else {
				// For plain text responses, check exact body content
				assert.Equal(t, tt.expectedBody, w.Body.String())
				// Should not contain HTML
				body := w.Body.String()
				assert.NotContains(t, body, "<html")
				assert.NotContains(t, body, "</html>")
			}

			mockDB.AssertExpectations(t)
		})
	}
}
