package handlers

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/drewstreib/xipe-go/utils"
	"github.com/gin-gonic/gin"
)

func isValidCode(code string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]{4,5}$", code)
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
			// Static pages can be cached for 1 hour since they don't change
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
			// Remove the no-cache headers set by middleware
			c.Header("Pragma", "")

			// Browser clients get HTML template (same as data.html)
			c.HTML(http.StatusOK, "data.html", gin.H{
				"code":         code,
				"url":          fullURL,
				"data":         content,
				"fromSuccess":  false,
				"created":      0,     // Static pages have no creation time
				"expires":      0,     // Static pages don't expire
				"showDelete":   false, // Static pages cannot be deleted
				"isStaticPage": true,  // Flag to indicate this is a static page
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

	// Handle data/pastebin types (both D and S)
	if redirect.Typ != "D" && redirect.Typ != "S" {
		utils.RespondWithError(c, http.StatusNotFound, "error", "Content not found")
		return
	}

	// Get the actual data content
	var dataContent string
	switch redirect.Typ {
	case "D":
		// Data stored directly in DynamoDB
		dataContent = redirect.Val
	case "S":
		// Data stored in S3, need to fetch it
		s3Key := "S/" + code + ".zst"
		s3Data, err := h.S3.GetObject(s3Key)
		if err != nil {
			// Check for specific S3 errors
			errorMsg := err.Error()
			if strings.Contains(errorMsg, "NoSuchKey") || strings.Contains(errorMsg, "NotFound") {
				// S3 object not found - treat as 404 since DynamoDB record exists but S3 data is missing
				utils.RespondWithError(c, http.StatusNotFound, "error", "Content not found or has expired")
			} else {
				// Other S3 errors (access denied, service unavailable, etc.)
				log.Printf("S3 error retrieving %s: %v", s3Key, err)
				utils.RespondWithError(c, http.StatusInternalServerError, "error", "Failed to retrieve content")
			}
			return
		}
		dataContent = string(s3Data)
	}

	// Calculate cache duration: min(1 hour, time until expiration)
	now := time.Now().Unix()
	maxCacheDuration := int64(3600) // 1 hour in seconds
	var cacheDuration int64

	if redirect.Ettl > 0 && redirect.Ettl > now {
		// Item has a TTL and hasn't expired yet
		timeUntilExpiration := redirect.Ettl - now
		if timeUntilExpiration < maxCacheDuration {
			cacheDuration = timeUntilExpiration
		} else {
			cacheDuration = maxCacheDuration
		}
	} else {
		// No TTL or already expired (shouldn't happen since we got the record)
		cacheDuration = maxCacheDuration
	}

	// Set cache headers for data pages (both HTML and raw responses)
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheDuration))
	c.Header("Expires", time.Now().Add(time.Duration(cacheDuration)*time.Second).UTC().Format(http.TimeFormat))
	// Remove the no-cache headers set by middleware
	c.Header("Pragma", "")

	// Return response based on client type
	if utils.ShouldReturnHTML(c) {
		// Browser clients get HTML template
		// Check if user owns this paste by comparing full owner IDs
		showDelete := false
		if ownerCookie, err := c.Cookie("id"); err == nil && ownerCookie == redirect.Owner {
			showDelete = true
		}

		c.HTML(http.StatusOK, "data.html", gin.H{
			"code":         code,
			"url":          fullURL,
			"data":         dataContent,
			"fromSuccess":  fromSuccess,
			"created":      redirect.Created,
			"expires":      redirect.Ettl,
			"showDelete":   showDelete,
			"isStaticPage": false, // Flag to indicate this is user data
		})
	} else {
		// API clients get raw content as plain text
		c.String(http.StatusOK, dataContent)
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
	codePattern := regexp.MustCompile("^[a-zA-Z0-9]{4,5}$")
	if codePattern.MatchString(path) {
		c.Params = append(c.Params, gin.Param{Key: "code", Value: path})
		h.DataHandler(c)
		return
	}

	utils.RespondWithError(c, http.StatusNotFound, "error", "Page not found")
}
