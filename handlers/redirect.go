package handlers

import (
	"net/http"
	"regexp"
	"github.com/drewstreib/xipe-go/db"

	"github.com/gin-gonic/gin"
)

func isValidCode(code string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]{4,6}$", code)
	return matched
}

func RedirectHandler(c *gin.Context) {
	code := c.Param("code")
	action := c.Query("action")

	if !isValidCode(code) {
		// Always return HTML for redirect errors since this is browser navigation
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"status":      "error",
			"description": "Invalid code format",
		})
		return
	}

	redirect, err := db.DB.GetRedirect(code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"status":      "error",
			"description": "Failed to retrieve URL",
		})
		return
	}

	if redirect == nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"status":      "error",
			"description": "Short URL not found or has expired",
		})
		return
	}

	// If action=info, show info page instead of redirecting
	if action == "info" {
		// Build the full URL for display
		scheme := "https"
		if c.Request.Header.Get("X-Forwarded-Proto") == "" && c.Request.TLS == nil {
			scheme = "http"
		}
		host := c.Request.Host
		if host == "" {
			host = "xi.pe"
		}
		fullURL := scheme + "://" + host + "/" + code

		c.HTML(http.StatusOK, "info.html", gin.H{
			"code":        code,
			"url":         fullURL,
			"originalUrl": redirect.Val,
			"redirectUrl": redirect.Val,
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

	// Always return HTML for catch-all since this is browser navigation
	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"status":      "error",
		"description": "Page not found",
	})
}
