package handlers

import (
	"net/http"
	"regexp"
	"xipe/db"
	"github.com/gin-gonic/gin"
)

func isValidCode(code string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]{4,6}$", code)
	return matched
}

func RedirectHandler(c *gin.Context) {
	code := c.Param("code")

	if !isValidCode(code) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid code format",
		})
		return
	}

	redirect, err := db.DB.GetRedirect(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve URL",
		})
		return
	}

	if redirect == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "not found",
		})
		return
	}

	c.Redirect(http.StatusMovedPermanently, redirect.Val)
}

func CatchAllHandler(c *gin.Context) {
	path := c.Request.URL.Path[1:]
	
	codePattern := regexp.MustCompile("^[a-zA-Z0-9]{4,6}$")
	if codePattern.MatchString(path) {
		c.Params = append(c.Params, gin.Param{Key: "code", Value: path})
		RedirectHandler(c)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "not found",
	})
}