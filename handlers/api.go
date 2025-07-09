package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	DB db.DBInterface
}

func (h *Handlers) URLPostHandler(c *gin.Context) {
	var ttl, rawURL string

	// Check if input format is specified as urlencoded
	if c.Query("input") == "urlencoded" {
		// Read from form body for URL-encoded data
		ttl = c.PostForm("ttl")
		rawURL = c.PostForm("url")
	} else {
		// Default: expect JSON body
		var requestBody struct {
			TTL string `json:"ttl" binding:"required"`
			URL string `json:"url" binding:"required"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "error", "Invalid JSON format or missing required fields (ttl, url)")
			return
		}

		ttl = requestBody.TTL
		rawURL = requestBody.URL
	}

	if ttl == "" || rawURL == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "ttl and url parameters are required")
		return
	}

	// Validate TTL
	if ttl != "1d" && ttl != "1w" && ttl != "1m" {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "ttl must be 1d, 1w, or 1m")
		return
	}

	// Decode and validate URL
	decodedURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "invalid URL encoding")
		return
	}

	// Check if URL starts with http:// or https://
	if !strings.HasPrefix(decodedURL, "http://") && !strings.HasPrefix(decodedURL, "https://") {
		utils.RespondWithError(c, http.StatusForbidden, "error", "URL must start with http:// or https://")
		return
	}

	_, err = url.ParseRequestURI(decodedURL)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "invalid URL format")
		return
	}

	// Check URL against Cloudflare family DNS filter
	urlCheckResult := utils.URLCheck(decodedURL)
	if !urlCheckResult.Allowed {
		utils.RespondWithError(c, urlCheckResult.Status, "error", urlCheckResult.Reason)
		return
	}

	// Calculate TTL and code length
	ettl, codeLength, err := utils.CalculateTTL(ttl)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to calculate TTL")
		return
	}

	// Try up to 5 times to insert a unique code
	var code string
	var insertErr error
	for attempts := 0; attempts < 5; attempts++ {
		code, err = utils.GenerateCode(codeLength)
		if err != nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to generate code")
			return
		}

		redirect := &db.RedirectRecord{
			Code: code,
			Typ:  "R",
			Val:  decodedURL,
			Ettl: ettl,
		}

		log.Printf("Attempting to store redirect - Code: %s, URL: %s", code, decodedURL)
		insertErr = h.DB.PutRedirect(redirect)
		if insertErr == nil {
			log.Printf("Successfully stored redirect - Code: %s", code)
			// Success! Build the full URL
			scheme := "https"
			if c.Request.Header.Get("X-Forwarded-Proto") == "" && c.Request.TLS == nil {
				scheme = "http"
			}
			host := c.Request.Host
			if host == "" {
				host = "xi.pe"
			}
			fullURL := scheme + "://" + host + "/" + code

			// Build redirect URL to info page with success parameter
			redirectPath := fmt.Sprintf("/%s?action=info&from=success", code)

			// Preserve format parameter if present
			if format := c.Query("format"); format != "" {
				redirectPath += "&format=" + url.QueryEscape(format)
			}

			// Return response based on client type
			if utils.ShouldReturnHTML(c) {
				// For HTML clients, redirect to info page
				c.Redirect(http.StatusSeeOther, redirectPath)
			} else {
				// For API clients, return JSON with the full URL
				c.JSON(http.StatusOK, gin.H{
					"status": "ok",
					"url":    fullURL,
				})
			}
			return
		}

		// Check if error is due to duplicate key
		if !isDuplicateKeyError(insertErr) {
			// Some other error occurred
			log.Printf("DynamoDB error (not duplicate key): %v", insertErr)
			utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to store URL")
			return
		}
		log.Printf("Duplicate key error, retrying with new code. Error: %v", insertErr)
		// Continue to next attempt if duplicate key
	}

	// All attempts failed
	utils.RespondWithError(c, 529, "error", "Could not allocate URL in the target namespace.")
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var ccf *types.ConditionalCheckFailedException
	return errors.As(err, &ccf)
}
