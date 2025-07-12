package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/handlers"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	// Initialize reserved codes from embedded pages
	if err := utils.InitReservedCodes(); err != nil {
		log.Fatal("Failed to initialize reserved codes:", err)
	}

	dbClient, err := db.NewDynamoDBClient()
	if err != nil {
		log.Fatal("Failed to create DynamoDB client:", err)
	}

	s3Client, err := db.NewS3Client()
	if err != nil {
		log.Fatal("Failed to create S3 client:", err)
	}

	h := &handlers.Handlers{
		DB: dbClient,
		S3: s3Client,
	}

	r := gin.Default()

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
	r.PUT("/", h.PutHandler)
	r.DELETE("/:code", h.DeleteHandler)

	api := r.Group("/api")
	{
		api.GET("/stats", h.StatsHandler)
	}

	// Serve static files
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("static/favicon.ico", http.FS(staticFS))
	})
	r.GET("/favicon-16x16.png", func(c *gin.Context) {
		c.FileFromFS("static/favicon-16x16.png", http.FS(staticFS))
	})
	r.GET("/favicon-32x32.png", func(c *gin.Context) {
		c.FileFromFS("static/favicon-32x32.png", http.FS(staticFS))
	})
	r.GET("/apple-touch-icon.png", func(c *gin.Context) {
		c.FileFromFS("static/apple-touch-icon.png", http.FS(staticFS))
	})
	r.GET("/android-chrome-192x192.png", func(c *gin.Context) {
		c.FileFromFS("static/android-chrome-192x192.png", http.FS(staticFS))
	})
	r.GET("/android-chrome-512x512.png", func(c *gin.Context) {
		c.FileFromFS("static/android-chrome-512x512.png", http.FS(staticFS))
	})
	r.GET("/site.webmanifest", func(c *gin.Context) {
		c.FileFromFS("static/site.webmanifest", http.FS(staticFS))
	})
	r.GET("/about.txt", func(c *gin.Context) {
		c.FileFromFS("static/about.txt", http.FS(staticFS))
	})
	r.GET("/swagger.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Header("Access-Control-Allow-Origin", "https://docs.xi.pe")
		c.FileFromFS("static/swagger.json", http.FS(staticFS))
	})

	r.GET("/:code", h.CatchAllHandler)

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
