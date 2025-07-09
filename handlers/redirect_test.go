package handlers

import (
	"encoding/json"
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
		code           string
		setupMock      func(*db.MockDB)
		expectedStatus int
		expectedHeader string
		expectedBody   map[string]interface{}
	}{
		{
			name: "Valid code - successful redirect",
			code: "test",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test").Return(&db.RedirectRecord{
					Code: "test",
					Typ:  "R",
					Val:  "https://example.com",
				}, nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			expectedHeader: "https://example.com",
			expectedBody:   nil,
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
			expectedBody: nil, // Now returns HTML, don't check body
		},
		{
			name:           "Invalid code format",
			code:           "test!",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
			expectedBody: nil, // Now returns HTML, don't check body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)
			db.DB = mockDB

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*") // Load templates for HTML responses
			c.Request = httptest.NewRequest("GET", "/"+tt.code, nil)
			c.Params = []gin.Param{{Key: "code", Value: tt.code}}

			RedirectHandler(c)

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
					Code: "test",
					Typ:  "R",
					Val:  "https://example.com",
				}, nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			expectedHeader: "https://example.com",
			expectedBody:   nil,
		},
		{
			name: "Valid 5-char code",
			path: "/test5",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test5").Return(&db.RedirectRecord{
					Code: "test5",
					Typ:  "R",
					Val:  "https://example.com",
				}, nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			expectedHeader: "https://example.com",
			expectedBody:   nil,
		},
		{
			name: "Valid 6-char code",
			path: "/test66",
			setupMock: func(m *db.MockDB) {
				m.On("GetRedirect", "test66").Return(&db.RedirectRecord{
					Code: "test66",
					Typ:  "R",
					Val:  "https://example.com",
				}, nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			expectedHeader: "https://example.com",
			expectedBody:   nil,
		},
		{
			name:           "Invalid path - too short",
			path:           "/abc",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: nil, // Now returns HTML, don't check body
		},
		{
			name:           "Invalid path - too long",
			path:           "/abcdefg",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: nil, // Now returns HTML, don't check body
		},
		{
			name:           "Invalid path - special characters",
			path:           "/test!234",
			setupMock:      func(m *db.MockDB) {},
			expectedStatus: http.StatusNotFound,
			expectedHeader: "",
			expectedBody: nil, // Now returns HTML, don't check body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(db.MockDB)
			tt.setupMock(mockDB)
			db.DB = mockDB

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.LoadHTMLGlob("../templates/*") // Load templates for HTML responses
			c.Request = httptest.NewRequest("GET", tt.path, nil)

			CatchAllHandler(c)

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
