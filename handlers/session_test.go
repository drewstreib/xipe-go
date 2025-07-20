package handlers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/drewstreib/xipe-go/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSessionExpiration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Session expires after MaxAge", func(t *testing.T) {
		// Create config with 1-second session expiration
		cfg := &config.Config{
			SessionsKey:   "test-secret-key-32-chars-long!",
			SessionMaxAge: 1, // 1 second expiration
		}

		// Create router with session middleware using 1-second expiration
		r := gin.New()

		// Set up cookie store with key rotation support (similar to main.go)
		var store sessions.Store
		store = cookie.NewStore([]byte(cfg.SessionsKey))

		// Configure cookie options with 1-second MaxAge
		store.Options(sessions.Options{
			Path:     "/",
			MaxAge:   int(cfg.SessionMaxAge), // 1 second
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		// Fix gorilla/sessions bug: MaxAge doesn't propagate to underlying securecookie codecs
		// This is the same reflection code as in main.go to ensure consistent behavior
		if storeValue := reflect.ValueOf(store).Elem(); storeValue.IsValid() {
			if storeValue.NumField() > 0 {
				cookieStoreField := storeValue.Field(0) // Embedded field is first
				if cookieStoreField.IsValid() && !cookieStoreField.IsNil() {
					if cs := cookieStoreField.Elem(); cs.IsValid() {
						if codecsField := cs.FieldByName("Codecs"); codecsField.IsValid() {
							for i := 0; i < codecsField.Len(); i++ {
								codec := codecsField.Index(i).Interface()
								// Use reflection to call MaxAge method on *securecookie.SecureCookie
								codecValue := reflect.ValueOf(codec)
								if codecValue.IsValid() {
									maxAgeMethod := codecValue.MethodByName("MaxAge")
									if maxAgeMethod.IsValid() {
										maxAgeMethod.Call([]reflect.Value{reflect.ValueOf(int(cfg.SessionMaxAge))})
									}
								}
							}
						}
					}
				}
			}
		}

		r.Use(sessions.Sessions("xipe_session", store))

		// Handler to set session value
		r.POST("/set-session", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("test_key", "test_value")
			session.Set("timestamp", time.Now().Unix())
			if err := session.Save(); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"status": "session_set"})
		})

		// Handler to get session value
		r.GET("/get-session", func(c *gin.Context) {
			session := sessions.Default(c)

			// Try to get values and catch any errors
			testKey := session.Get("test_key")
			timestamp := session.Get("timestamp")

			// Check if this is a new session (which would indicate the old one was rejected)
			isNew := false
			if s, ok := session.(interface {
				Session() interface{ IsNew() bool }
			}); ok {
				if sess := s.Session(); sess != nil {
					isNew = sess.IsNew()
				}
			}

			if testKey == nil {
				c.JSON(200, gin.H{"session": "empty", "test_key": nil, "timestamp": nil, "is_new": isNew})
			} else {
				c.JSON(200, gin.H{"session": "exists", "test_key": testKey, "timestamp": timestamp, "is_new": isNew})
			}
		})

		// Step 1: Set session value
		setReq := httptest.NewRequest("POST", "/set-session", nil)
		setRecorder := httptest.NewRecorder()
		r.ServeHTTP(setRecorder, setReq)

		assert.Equal(t, http.StatusOK, setRecorder.Code)

		// Extract the session cookie
		var sessionCookie *http.Cookie
		for _, cookie := range setRecorder.Result().Cookies() {
			if cookie.Name == "xipe_session" {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie, "Session cookie should be set")

		// Step 2: Immediately read session value (should work)
		getReq1 := httptest.NewRequest("GET", "/get-session", nil)
		getReq1.AddCookie(sessionCookie)
		getRecorder1 := httptest.NewRecorder()
		r.ServeHTTP(getRecorder1, getReq1)

		assert.Equal(t, http.StatusOK, getRecorder1.Code)
		assert.Contains(t, getRecorder1.Body.String(), `"session":"exists"`)
		assert.Contains(t, getRecorder1.Body.String(), `"test_key":"test_value"`)

		// Step 3: Wait for session to expire (2 seconds > 1 second MaxAge)
		time.Sleep(2 * time.Second)

		// Step 4: Try to read session value again (should fail - session expired)
		getReq2 := httptest.NewRequest("GET", "/get-session", nil)
		getReq2.AddCookie(sessionCookie) // Same cookie, but now expired server-side
		getRecorder2 := httptest.NewRecorder()
		r.ServeHTTP(getRecorder2, getReq2)

		assert.Equal(t, http.StatusOK, getRecorder2.Code)
		// The session should be empty because the securecookie timestamp validation failed
		assert.Contains(t, getRecorder2.Body.String(), `"session":"empty"`)
		assert.Contains(t, getRecorder2.Body.String(), `"test_key":null`)
		assert.Contains(t, getRecorder2.Body.String(), `"timestamp":null`)
	})
}
