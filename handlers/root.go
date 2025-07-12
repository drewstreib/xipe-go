package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) RootHandler(c *gin.Context) {
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
