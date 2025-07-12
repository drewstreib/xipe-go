package handlers

import (
	"net/http"
	"regexp"

	"github.com/drewstreib/xipe-go/utils"
	"github.com/gin-gonic/gin"
)

func isValidCode(code string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]{4,6}$", code)
	return matched
}

func (h *Handlers) DataHandler(c *gin.Context) {
	code := c.Param("code")

	// Check if this is a reserved code (static page)
	if utils.IsReservedCode(code) {
		content, err := utils.GetPageContent(code)
		if err != nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "error", "Failed to load page")
			return
		}

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

		// Return response based on client type
		if utils.ShouldReturnHTML(c) {
			// Browser clients get HTML template (same as data.html)
			c.HTML(http.StatusOK, "data.html", gin.H{
				"code":         code,
				"url":          fullURL,
				"data":         content,
				"fromSuccess":  false,
				"created":      0,    // Static pages have no creation time
				"expires":      0,    // Static pages don't expire
				"ownerPrefix":  "",   // Static pages have no owner
				"isStaticPage": true, // Flag to indicate this is a static page
			})
		} else {
			// API clients get raw content as plain text
			c.String(http.StatusOK, content)
		}
		return
	}

	// For regular codes, validate format
	if !isValidCode(code) {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "Invalid code format")
		return
	}

	redirect, err := h.DB.GetRedirect(code)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "error", "Failed to retrieve URL")
		return
	}

	if redirect == nil {
		utils.RespondWithError(c, http.StatusNotFound, "error", "Short URL not found or has expired")
		return
	}

	// Default behavior: show info page (no automatic redirects for security)
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

	// Check if this is from a successful creation
	fromSuccess := c.Query("from") == "success"

	// Return response based on client type
	if utils.ShouldReturnHTML(c) {
		// Browser clients get HTML templates
		if redirect.Typ == "D" {
			// Data/pastebin type - only pass first 6 chars of owner for security
			var ownerPrefix string
			if len(redirect.Owner) >= 6 {
				ownerPrefix = redirect.Owner[:6]
			}
			c.HTML(http.StatusOK, "data.html", gin.H{
				"code":         code,
				"url":          fullURL,
				"data":         redirect.Val,
				"fromSuccess":  fromSuccess,
				"created":      redirect.Created,
				"expires":      redirect.Ettl,
				"ownerPrefix":  ownerPrefix,
				"isStaticPage": false, // Flag to indicate this is user data
			})
		} else {
			// URL redirect type (default)
			c.HTML(http.StatusOK, "url.html", gin.H{
				"code":        code,
				"url":         fullURL,
				"originalUrl": redirect.Val,
				"redirectUrl": redirect.Val,
				"fromSuccess": fromSuccess,
				"created":     redirect.Created,
				"expires":     redirect.Ettl,
			})
		}
	} else {
		// API clients get raw content as plain text
		if redirect.Typ == "D" {
			// Return raw data content
			c.String(http.StatusOK, redirect.Val)
		} else {
			// Return raw URL
			c.String(http.StatusOK, redirect.Val)
		}
	}
}

func (h *Handlers) CatchAllHandler(c *gin.Context) {
	path := c.Request.URL.Path[1:]

	// Check if this is a reserved code (static page) first
	if utils.IsReservedCode(path) {
		c.Params = append(c.Params, gin.Param{Key: "code", Value: path})
		h.DataHandler(c)
		return
	}

	// Then check if it matches the standard code pattern for generated codes
	codePattern := regexp.MustCompile("^[a-zA-Z0-9]{4,6}$")
	if codePattern.MatchString(path) {
		c.Params = append(c.Params, gin.Param{Key: "code", Value: path})
		h.DataHandler(c)
		return
	}

	utils.RespondWithError(c, http.StatusNotFound, "error", "Page not found")
}
