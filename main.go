package main

import (
	"embed"
	"html/template"
	"log"
	"xipe/db"
	"xipe/handlers"
	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templatesFS embed.FS

func main() {
	db.Init()

	r := gin.Default()
	
	// Load templates from embedded filesystem
	tmpl := template.Must(template.New("").ParseFS(templatesFS, "templates/*"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", handlers.RootHandler)
	r.GET("/stats", handlers.StatsHandler)

	api := r.Group("/api")
	{
		api.GET("/urlpost", handlers.URLPostHandler)
	}

	r.GET("/:code", handlers.CatchAllHandler)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}