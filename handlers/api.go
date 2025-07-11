package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	DB db.DBInterface
}

// generateOwnerToken generates a 128-bit random token and encodes it as base64
func generateOwnerToken() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 128 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// getOrCreateOwnerID gets existing owner ID from cookie or creates a new one
func getOrCreateOwnerID(c *gin.Context) (string, error) {
	// Check if owner ID cookie already exists
	if ownerID, err := c.Cookie("id"); err == nil && ownerID != "" {
		log.Printf("Reusing existing owner ID from cookie")
		return ownerID, nil
	}

	// Generate new owner ID
	ownerID, err := generateOwnerToken()
	if err != nil {
		return "", err
	}

	log.Printf("Generated new owner ID")
	return ownerID, nil
}

func (h *Handlers) PostHandler(c *gin.Context) {
	var ttl, rawData, typ string
	var isDataPost bool

	// Get or create owner ID for this post
	ownerID, err := getOrCreateOwnerID(c)
	if err != nil {
		log.Printf("Failed to generate owner ID: %v", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to generate owner ID")
		return
	}

	// Check if input format is specified as urlencoded
	if c.Query("input") == "urlencoded" {
		// Read from form body for URL-encoded data
		ttl = c.PostForm("ttl")
		rawData = c.PostForm("data")
		typ = c.PostForm("typ")
	} else {
		// Default: expect JSON body
		var requestBody struct {
			TTL  string `json:"ttl"`
			Data string `json:"data" binding:"required"`
			Typ  string `json:"typ"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "error", "Invalid JSON format or missing required field (data)")
			return
		}

		ttl = requestBody.TTL
		rawData = requestBody.Data
		typ = requestBody.Typ
	}

	// Determine if this is a URL or data post based on typ parameter
	// Default to data post if typ is not specified or is "Text"
	if typ == "URL" {
		isDataPost = false
	} else {
		// Default to data post for "Text" or empty typ
		isDataPost = true
	}

	if rawData == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "data parameter is required")
		return
	}

	// Default TTL to "1d" if not provided
	if ttl == "" {
		ttl = "1d"
	}

	// Validate TTL
	if ttl != "1d" && ttl != "1w" && ttl != "1mo" {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "ttl must be 1d, 1w, or 1mo")
		return
	}

	var finalValue string
	var recordType string

	if isDataPost {
		// Handle data post
		recordType = "D"

		// For data posts, use the raw data as-is (already URL-decoded by Gin for form data)
		// Only URL decode if this came from JSON and might be manually encoded
		var processedData string
		if c.Query("input") == "urlencoded" {
			// Form data is already URL-decoded by Gin
			processedData = rawData
		} else {
			// JSON data might be manually URL-encoded, try to decode
			if decodedData, err := url.QueryUnescape(rawData); err == nil {
				processedData = decodedData
			} else {
				// If decoding fails, use as-is (probably wasn't URL-encoded)
				processedData = rawData
			}
		}

		// Check data length (50KB max)
		dataLen := len(processedData)
		log.Printf("DEBUG: processedData length = %d bytes", dataLen)
		if dataLen > 51200 {
			utils.RespondWithError(c, http.StatusForbidden, "error", fmt.Sprintf("Data too long (%d bytes, 50KB max)", dataLen))
			return
		}

		finalValue = processedData
	} else {
		// Handle URL post
		recordType = "R"

		// Process URL from data field
		var processedURL string
		if c.Query("input") == "urlencoded" {
			// Form data is already URL-decoded by Gin
			processedURL = rawData
		} else {
			// JSON data might be manually URL-encoded, try to decode
			if decodedURL, err := url.QueryUnescape(rawData); err == nil {
				processedURL = decodedURL
			} else {
				// If decoding fails, use as-is (probably wasn't URL-encoded)
				processedURL = rawData
			}
		}

		// Check URL length (4KB max)
		if len(processedURL) > 4096 {
			utils.RespondWithError(c, http.StatusForbidden, "error", "URL too long (4KB max)")
			return
		}

		// Check if URL starts with http:// or https://
		if !strings.HasPrefix(processedURL, "http://") && !strings.HasPrefix(processedURL, "https://") {
			utils.RespondWithError(c, http.StatusForbidden, "error", "URL must start with http:// or https://")
			return
		}

		_, err := url.ParseRequestURI(processedURL)
		if err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "error", "invalid URL format")
			return
		}

		// Check URL against Cloudflare family DNS filter
		urlCheckResult := utils.URLCheck(processedURL)
		if !urlCheckResult.Allowed {
			utils.RespondWithError(c, urlCheckResult.Status, "error", urlCheckResult.Reason)
			return
		}

		finalValue = processedURL
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

		// Get the client IP (Gin's ClientIP handles X-Forwarded-For, X-Real-IP, etc.)
		clientIP := c.ClientIP()

		// Get current timestamp
		createdTime := time.Now().Unix()

		record := &db.RedirectRecord{
			Code:    code,
			Typ:     recordType,
			Val:     finalValue,
			Ettl:    ettl,
			Created: createdTime,
			IP:      clientIP,
			Owner:   ownerID,
		}

		log.Printf("Attempting to store %s - Code: %s, Value: %s",
			map[string]string{"R": "redirect", "D": "data"}[recordType],
			code,
			func() string {
				if len(finalValue) > 50 {
					return finalValue[:50] + "..."
				}
				return finalValue
			}())
		insertErr = h.DB.PutRedirect(record)
		if insertErr == nil {
			log.Printf("Successfully stored %s - Code: %s",
				map[string]string{"R": "redirect", "D": "data"}[recordType], code)

			// Set the owner ID cookie (30 days expiration)
			c.SetCookie("id", ownerID, 30*24*60*60, "/", "", false, true)

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
			redirectPath := fmt.Sprintf("/%s?from=success", code)

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

func (h *Handlers) DeleteHandler(c *gin.Context) {
	code := c.Param("code")

	// Get owner ID from cookie
	ownerID, err := c.Cookie("id")
	if err != nil || ownerID == "" {
		log.Printf("Delete request without valid owner ID cookie for code: %s", code)
		utils.RespondWithError(c, http.StatusUnauthorized, "error", "unauthorized")
		return
	}

	// Attempt to delete the record
	err = h.DB.DeleteRedirect(code, ownerID)
	if err != nil {
		// Check if it's a conditional check failure (wrong owner or not found)
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			log.Printf("Delete failed for code %s: unauthorized or not found", code)
			utils.RespondWithError(c, http.StatusUnauthorized, "error", "unauthorized")
			return
		}

		// Other database error
		log.Printf("Database error during delete for code %s: %v", code, err)
		utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to delete")
		return
	}

	log.Printf("Successfully deleted code: %s", code)

	// Return success response
	if utils.ShouldReturnHTML(c) {
		// For HTML clients, redirect to the original item location (which will now 404)
		c.Redirect(http.StatusMovedPermanently, "/"+code)
		return
	} else {
		// For API clients, return JSON success
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "deleted successfully",
		})
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var ccf *types.ConditionalCheckFailedException
	return errors.As(err, &ccf)
}
