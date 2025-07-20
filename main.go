package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/drewstreib/xipe-go/config"
	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/handlers"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	gorillaSessions "github.com/gorilla/sessions"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize reserved codes from embedded pages
	if err := utils.InitReservedCodes(); err != nil {
		log.Fatal("Failed to initialize reserved codes:", err)
	}

	dbClient, err := db.NewDynamoDBClient(cfg)
	if err != nil {
		log.Fatal("Failed to create DynamoDB client:", err)
	}

	s3Client, err := db.NewS3Client()
	if err != nil {
		log.Fatal("Failed to create S3 client:", err)
	}

	h := &handlers.Handlers{
		DB:  dbClient,
		S3:  s3Client,
		Cfg: cfg,
	}

	r := gin.Default()

	// Configure session middleware with cookie store
	// Create store with key rotation support
	var store sessions.Store
	if cfg.SessionsKeyPrev != "" {
		// Support key rotation: new key for signing, both keys for verification
		store = cookie.NewStore(
			[]byte(cfg.SessionsKey),     // Current key for signing
			[]byte(cfg.SessionsKeyPrev), // Previous key for verification only
		)
	} else {
		// Single key for both signing and verification
		store = cookie.NewStore([]byte(cfg.SessionsKey))
	}

	// Configure cookie options using configurable MaxAge
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   int(cfg.SessionMaxAge), // Use configurable session max age
		HttpOnly: true,                   // Security: prevent JavaScript access
		Secure:   false,                  // Allow HTTP for development (should be true in production with HTTPS)
		SameSite: http.SameSiteLaxMode,
	})

	// Fix gorilla/sessions bug: MaxAge doesn't propagate to underlying securecookie codecs
	// We need to access the underlying store and set MaxAge on each codec
	// The CookieStore is embedded anonymously as the first field
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

	// Apply session middleware
	r.Use(sessions.Sessions("xipe_session", store))

	// Debug middleware to log session contents
	r.Use(func(c *gin.Context) {
		session := sessions.Default(c)

		// Access the underlying gorilla session to get all values
		// This requires type assertion to access the internal session
		if s, ok := session.(interface {
			Session() *gorillaSessions.Session
		}); ok {
			gorillaSession := s.Session()
			if gorillaSession != nil && len(gorillaSession.Values) > 0 {
				// Convert all values to a string-keyed map for JSON marshaling
				sessionData := make(map[string]interface{})
				for key, value := range gorillaSession.Values {
					// Convert key to string for JSON compatibility
					keyStr := fmt.Sprintf("%v", key)
					sessionData[keyStr] = value
				}

				// Convert to JSON for readable output
				jsonData, err := json.Marshal(sessionData)
				if err != nil {
					log.Printf("Session Debug: Error marshaling session data: %v", err)
				} else {
					log.Printf("Session Debug: Path=%s Method=%s SessionData=%s",
						c.Request.URL.Path, c.Request.Method, string(jsonData))
				}
			}
		}

		c.Next()
	})

	// Security headers middleware
	r.Use(func(c *gin.Context) {
		// Content Security Policy - prevent XSS and script injection
		// Allow highlight.js from cdnjs, inline styles for templates, and self for everything else
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; "+
				"style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; "+
				"font-src 'self'; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"object-src 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"frame-ancestors 'none'")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content type sniffing protection
		c.Header("X-Content-Type-Options", "nosniff")

		// Referrer policy - don't leak URLs to external sites
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// CORS - restrictive by default
		c.Header("Access-Control-Allow-Origin", "")
		c.Header("Access-Control-Allow-Methods", "GET, POST")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		// Prevent caching of sensitive pages
		if c.Request.URL.Path != "/" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	})

	// Load templates from embedded filesystem
	tmpl := template.Must(template.New("").ParseFS(templatesFS, "templates/*"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", h.RootHandler)
	r.POST("/", h.PostHandler)
	r.DELETE("/:code", h.DeleteHandler)
	r.GET("/challenge-check", h.HandleChallengeCheck)
	r.GET("/cloudflare-test", h.HandleCloudflareTest)

	api := r.Group("/api")
	{
		api.GET("/stats", h.StatsHandler)
	}

	// Helper function to serve static files with 1-day cache headers
	serveStaticWithCache := func(c *gin.Context, filePath string) {
		// Cache static files for 1 day
		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("Expires", time.Now().Add(24*time.Hour).UTC().Format(http.TimeFormat))
		c.Header("Pragma", "")
		c.FileFromFS(filePath, http.FS(staticFS))
	}

	// Serve static files
	r.GET("/favicon.ico", func(c *gin.Context) {
		serveStaticWithCache(c, "static/favicon.ico")
	})
	r.GET("/favicon-16x16.png", func(c *gin.Context) {
		serveStaticWithCache(c, "static/favicon-16x16.png")
	})
	r.GET("/favicon-32x32.png", func(c *gin.Context) {
		serveStaticWithCache(c, "static/favicon-32x32.png")
	})
	r.GET("/apple-touch-icon.png", func(c *gin.Context) {
		serveStaticWithCache(c, "static/apple-touch-icon.png")
	})
	r.GET("/android-chrome-192x192.png", func(c *gin.Context) {
		serveStaticWithCache(c, "static/android-chrome-192x192.png")
	})
	r.GET("/android-chrome-512x512.png", func(c *gin.Context) {
		serveStaticWithCache(c, "static/android-chrome-512x512.png")
	})
	r.GET("/about.txt", func(c *gin.Context) {
		serveStaticWithCache(c, "static/about.txt")
	})
	r.GET("/robots.txt", func(c *gin.Context) {
		serveStaticWithCache(c, "static/robots.txt")
	})
	r.GET("/swagger.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Header("Access-Control-Allow-Origin", "https://docs.xi.pe")
		// Cache static files for 1 day
		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("Expires", time.Now().Add(24*time.Hour).UTC().Format(http.TimeFormat))
		c.Header("Pragma", "")
		c.FileFromFS("static/swagger.json", http.FS(staticFS))
	})

	r.GET("/:code", h.CatchAllHandler)

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
