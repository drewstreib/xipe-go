package main

import (
	"embed"
	"html/template"
	"log"

	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/handlers"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templatesFS embed.FS

func main() {
	dbClient, err := db.NewDynamoDBClient()
	if err != nil {
		log.Fatal("Failed to create DynamoDB client:", err)
	}

	h := &handlers.Handlers{
		DB: dbClient,
	}

	r := gin.Default()

	// Load templates from embedded filesystem
	tmpl := template.Must(template.New("").ParseFS(templatesFS, "templates/*"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", h.RootHandler)

	api := r.Group("/api")
	{
		api.GET("/urlpost", h.URLPostHandler)
		api.GET("/stats", h.StatsHandler)
	}

	r.GET("/:code", h.CatchAllHandler)

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
