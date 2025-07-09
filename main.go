package main

import (
	"log"
	"xipe/db"
	"xipe/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	db.Init()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", handlers.RootHandler)
	r.GET("/stats", handlers.StatsHandler)

	api := r.Group("/api")
	{
		api.GET("/urlpost", handlers.URLPostHandler)
	}

	r.GET("/:key", handlers.CatchAllHandler)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}