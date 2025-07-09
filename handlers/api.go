package handlers

import (
	"net/http"
	"regexp"
	"xipe/db"
	"github.com/gin-gonic/gin"
)

func URLPostHandler(c *gin.Context) {
	key := c.Query("key")
	url := c.Query("url")

	if key == "" || url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key and url parameters are required",
		})
		return
	}

	if !isValidKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key must be 4-8 alphanumeric characters",
		})
		return
	}

	err := db.DB.PutURL(key, url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to store URL",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"key":    key,
		"url":    url,
	})
}

func isValidKey(key string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]{4,8}$", key)
	return matched
}