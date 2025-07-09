package handlers

import (
	"net/http"
	"regexp"
	"xipe/db"
	"github.com/gin-gonic/gin"
)

func RedirectHandler(c *gin.Context) {
	key := c.Param("key")

	if !isValidKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid key format",
		})
		return
	}

	url, err := db.DB.GetURL(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve URL",
		})
		return
	}

	if url == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "key not found",
		})
		return
	}

	c.Redirect(http.StatusMovedPermanently, url)
}

func CatchAllHandler(c *gin.Context) {
	path := c.Request.URL.Path[1:]
	
	keyPattern := regexp.MustCompile("^[a-zA-Z0-9]{4,8}$")
	if keyPattern.MatchString(path) {
		c.Param("key")
		c.Set("key", path)
		RedirectHandler(c)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "not found",
	})
}