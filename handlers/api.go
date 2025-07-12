package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	DB db.DBInterface
	S3 db.S3Interface
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
	var rawData string

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
		rawData = c.PostForm("data")
	} else {
		// Default: expect JSON body
		var requestBody struct {
			Data string `json:"data" binding:"required"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "error", "Invalid JSON format or missing required field (data)")
			return
		}

		rawData = requestBody.Data
	}

	if rawData == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "data parameter is required")
		return
	}

	// Always use 1d TTL (24 hours)
	const ttl = "1d"

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

	// Check data length (2MB max)
	dataLen := len(processedData)
	log.Printf("DEBUG: processedData length = %d bytes", dataLen)
	if dataLen > 2097152 {
		utils.RespondWithError(c, http.StatusForbidden, "error", fmt.Sprintf("Data too long (%d bytes, 2MB max)", dataLen))
		return
	}

	// Determine storage type and prepare data
	var recordType string
	var finalValue string
	var s3Key string

	if dataLen <= 10240 { // 10KB or less: store in DynamoDB
		recordType = "D"
		finalValue = processedData
	} else { // Over 10KB: store in S3
		recordType = "S"
		finalValue = "" // Empty in DynamoDB, data will be in S3
	}

	// Calculate TTL (always 1d now)
	ettl, _, err := utils.CalculateTTL(ttl)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to calculate TTL")
		return
	}

	// Try 3 times with 4-character codes, then 3 times with 5-character codes
	var code string
	var insertErr error
	totalAttempts := 0

	for _, currentCodeLength := range []int{4, 5} {
		for attempts := 0; attempts < 3; attempts++ {
			totalAttempts++
			code, err = utils.GenerateUniqueCode(currentCodeLength)
			if err != nil {
				utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to generate code")
				return
			}

			// Set S3 key if using S3 storage
			if recordType == "S" {
				s3Key = "S/" + code + ".zst"
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

			log.Printf("Attempting to store data - Code: %s (%d chars), Type: %s, Size: %d bytes, Attempt: %d/6",
				code, currentCodeLength, recordType, dataLen, totalAttempts)

			// Store in S3 first if needed (before DynamoDB to ensure data is available)
			if recordType == "S" {
				s3Err := h.S3.PutObject(s3Key, []byte(processedData))
				if s3Err != nil {
					log.Printf("Failed to store data in S3: %v", s3Err)
					// Check for specific S3 errors
					errorMsg := s3Err.Error()
					if strings.Contains(errorMsg, "AccessDenied") || strings.Contains(errorMsg, "Forbidden") {
						utils.RespondWithError(c, http.StatusInternalServerError, "error", "storage service access denied")
					} else if strings.Contains(errorMsg, "ServiceUnavailable") || strings.Contains(errorMsg, "SlowDown") {
						utils.RespondWithError(c, http.StatusServiceUnavailable, "error", "storage service temporarily unavailable")
					} else if strings.Contains(errorMsg, "NoSuchBucket") {
						utils.RespondWithError(c, http.StatusInternalServerError, "error", "storage configuration error")
					} else {
						utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to store data")
					}
					return
				}
				log.Printf("Successfully stored data in S3 - Key: %s", s3Key)
			}

			insertErr = h.DB.PutRedirect(record)
			if insertErr == nil {
				log.Printf("Successfully stored metadata - Code: %s (%d chars)", code, currentCodeLength)

				// Set the owner ID cookie (30 days expiration, no HttpOnly so JS can read for delete button)
				c.SetCookie("id", ownerID, 30*24*60*60, "/", "", false, false)

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

				// Preserve html parameter if present
				if c.Request.URL.Query().Has("html") {
					redirectPath += "&html"
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
				utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to store data")
				return
			}
			log.Printf("Duplicate key error for %d-char code, retrying. Error: %v", currentCodeLength, insertErr)
			// Continue to next attempt if duplicate key
		}
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
		if utils.ShouldReturnHTML(c) {
			// For browser clients, redirect to error page
			c.HTML(http.StatusUnauthorized, "error.html", gin.H{
				"status":      "error",
				"description": "You are not authorized to delete this item",
				"statusCode":  http.StatusUnauthorized,
			})
		} else {
			utils.RespondWithError(c, http.StatusUnauthorized, "error", "unauthorized")
		}
		return
	}

	// Attempt to delete the record
	err = h.DB.DeleteRedirect(code, ownerID)
	if err != nil {
		// Check if it's a conditional check failure (wrong owner or not found)
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			log.Printf("Delete failed for code %s: unauthorized or not found", code)
			if utils.ShouldReturnHTML(c) {
				// For browser clients, redirect to error page
				c.HTML(http.StatusUnauthorized, "error.html", gin.H{
					"status":      "error",
					"description": "You are not authorized to delete this item",
					"statusCode":  http.StatusUnauthorized,
				})
			} else {
				utils.RespondWithError(c, http.StatusUnauthorized, "error", "unauthorized")
			}
			return
		}

		// Other database error
		log.Printf("Database error during delete for code %s: %v", code, err)
		if utils.ShouldReturnHTML(c) {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"status":      "error",
				"description": "Failed to delete item",
				"statusCode":  http.StatusInternalServerError,
			})
		} else {
			utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to delete")
		}
		return
	}

	log.Printf("Successfully deleted code: %s", code)

	// Always redirect for browser clients to the item URL (which will now 404)
	if utils.ShouldReturnHTML(c) {
		c.Redirect(http.StatusSeeOther, "/"+code)
	} else {
		// For API clients, return JSON success
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "deleted successfully",
		})
	}
}

// PutHandler handles raw text uploads via PUT /
func (h *Handlers) PutHandler(c *gin.Context) {
	// Read the raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "Error: Failed to read request body\n")
		return
	}

	// Validate UTF-8
	if !utf8.Valid(body) {
		c.String(http.StatusBadRequest, "Error: Input text must be UTF-8\n")
		return
	}

	// Convert to string for processing
	rawData := string(body)

	// Validate that we have content to store
	if len(rawData) == 0 {
		c.String(http.StatusBadRequest, "Error: Cannot store empty content\n")
		return
	}

	// Smart truncation to 2MB (2097152 bytes) while preserving UTF-8
	const maxBytes = 2097152
	finalData := rawData
	if len(rawData) > maxBytes {
		// Find the longest valid UTF-8 prefix within the limit
		finalData = rawData
		for len(finalData) > maxBytes {
			// Remove the last rune to ensure we don't break UTF-8
			_, size := utf8.DecodeLastRuneInString(finalData)
			if size == 0 {
				break
			}
			finalData = finalData[:len(finalData)-size]
		}
		log.Printf("Truncated input from %d bytes to %d bytes", len(rawData), len(finalData))

		// Ensure truncation didn't result in empty content
		if len(finalData) == 0 {
			c.String(http.StatusBadRequest, "Error: Content became empty after truncation\n")
			return
		}
	}

	// Get or create owner ID for this post
	ownerID, err := getOrCreateOwnerID(c)
	if err != nil {
		log.Printf("Failed to generate owner ID: %v", err)
		c.String(http.StatusInternalServerError, "Error: Failed to generate owner ID\n")
		return
	}

	// Determine storage type for PUT data
	dataLen := len(finalData)
	var recordType string
	var finalValue string
	var s3Key string

	if dataLen <= 10240 { // 10KB or less: store in DynamoDB
		recordType = "D"
		finalValue = finalData
	} else { // Over 10KB: store in S3
		recordType = "S"
		finalValue = "" // Empty in DynamoDB, data will be in S3
	}

	// Use default 1d TTL for PUT requests
	ettl, _, err := utils.CalculateTTL("1d")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: Failed to calculate TTL\n")
		return
	}

	// Try 3 times with 4-character codes, then 3 times with 5-character codes
	var code string
	var insertErr error
	totalAttempts := 0

	for _, currentCodeLength := range []int{4, 5} {
		for attempts := 0; attempts < 3; attempts++ {
			totalAttempts++
			code, err = utils.GenerateUniqueCode(currentCodeLength)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error: Failed to generate code\n")
				return
			}

			// Set S3 key if using S3 storage
			if recordType == "S" {
				s3Key = "S/" + code + ".zst"
			}

			// Get the client IP
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

			log.Printf("PUT: Attempting to store data - Code: %s (%d chars), Type: %s, Size: %d bytes, Attempt: %d/6",
				code, currentCodeLength, recordType, dataLen, totalAttempts)

			// Store in S3 first if needed
			if recordType == "S" {
				s3Err := h.S3.PutObject(s3Key, []byte(finalData))
				if s3Err != nil {
					log.Printf("PUT: Failed to store data in S3: %v", s3Err)
					// Check for specific S3 errors
					errorMsg := s3Err.Error()
					if strings.Contains(errorMsg, "AccessDenied") || strings.Contains(errorMsg, "Forbidden") {
						c.String(http.StatusInternalServerError, "Error: Storage service access denied\n")
					} else if strings.Contains(errorMsg, "ServiceUnavailable") || strings.Contains(errorMsg, "SlowDown") {
						c.String(http.StatusServiceUnavailable, "Error: Storage service temporarily unavailable\n")
					} else if strings.Contains(errorMsg, "NoSuchBucket") {
						c.String(http.StatusInternalServerError, "Error: Storage configuration error\n")
					} else {
						c.String(http.StatusInternalServerError, "Error: Failed to store data\n")
					}
					return
				}
				log.Printf("PUT: Successfully stored data in S3 - Key: %s", s3Key)
			}

			insertErr = h.DB.PutRedirect(record)
			if insertErr == nil {
				log.Printf("PUT: Successfully stored metadata - Code: %s (%d chars)", code, currentCodeLength)

				// Set the owner ID cookie (30 days expiration, no HttpOnly)
				c.SetCookie("id", ownerID, 30*24*60*60, "/", "", false, false)

				// Build the full URL
				scheme := "https"
				if c.Request.Header.Get("X-Forwarded-Proto") == "" && c.Request.TLS == nil {
					scheme = "http"
				}
				host := c.Request.Host
				if host == "" {
					host = "xi.pe"
				}
				fullURL := scheme + "://" + host + "/" + code

				// Return plain text response with just the URL
				c.String(http.StatusOK, fullURL+"\n")
				return
			}

			// Check if error is due to duplicate key
			if !isDuplicateKeyError(insertErr) {
				// Some other error occurred
				log.Printf("DynamoDB error (not duplicate key): %v", insertErr)
				c.String(http.StatusInternalServerError, "Error: Failed to store data\n")
				return
			}
			log.Printf("PUT: Duplicate key error for %d-char code, retrying. Error: %v", currentCodeLength, insertErr)
			// Continue to next attempt if duplicate key
		}
	}

	// All attempts failed
	c.String(529, "Error: Could not allocate URL in the target namespace.\n")
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var ccf *types.ConditionalCheckFailedException
	return errors.As(err, &ccf)
}
