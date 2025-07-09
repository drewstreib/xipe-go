package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) RootHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "xi.pe - URL Shortener & Pastebin",
	})
}

func (h *Handlers) StatsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"stats": gin.H{
			"total_urls":   0,
			"total_pastes": 0,
		},
	})
}
