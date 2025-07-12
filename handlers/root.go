package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) RootHandler(c *gin.Context) {
	// Cache root page for 1 hour since it's static content
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "xi.pe pastebin service",
	})
}

func (h *Handlers) StatsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"stats": gin.H{
			"cached_items": h.DB.GetCacheSize(),
		},
	})
}
