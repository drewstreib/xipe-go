package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/drewstreib/xipe-go/config"
	"github.com/drewstreib/xipe-go/db"
	"github.com/drewstreib/xipe-go/utils"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	DB  db.DBInterface
	S3  db.S3Interface
	Cfg *config.Config
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
	var isFormInput bool

	// Get or create owner ID for this post
	ownerID, err := getOrCreateOwnerID(c)
	if err != nil {
		log.Printf("Failed to generate owner ID: %v", err)
		c.String(http.StatusInternalServerError, "Error: Failed to generate owner ID\n")
		return
	}

	// Check if input format is specified as form
	if c.Query("input") == "form" {
		// Read from form body for URL-encoded data
		rawData = c.PostForm("data")
		isFormInput = true

		if rawData == "" {
			c.String(http.StatusBadRequest, "Error: data parameter is required\n")
			return
		}
	} else {
		// Default: read raw body like old PUT
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
		rawData = string(body)

		// Validate that we have content to store
		if len(rawData) == 0 {
			c.String(http.StatusBadRequest, "Error: Cannot store empty content\n")
			return
		}
	}

	// Smart truncation to configured max size while preserving UTF-8
	maxBytes := h.Cfg.PasteMaxSize
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

	// Determine storage type for POST data
	dataLen := len(finalData)
	var recordType string
	var finalValue string
	var s3Key string

	if dataLen <= h.Cfg.PasteDynamoDBCutoffSize { // Configurable size threshold: store in DynamoDB
		recordType = "D"
		finalValue = finalData
	} else { // Over cutoff size: store in S3
		recordType = "S"
		finalValue = "" // Empty in DynamoDB, data will be in S3
	}

	// POST TTL from configuration
	ettl := time.Now().Add(time.Duration(h.Cfg.PasteTTL) * time.Second).Unix()

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

			log.Printf("POST: Attempting to store data - Code: %s (%d chars), Type: %s, Size: %d bytes, Attempt: %d/6",
				code, currentCodeLength, recordType, dataLen, totalAttempts)

			// Store in S3 first if needed
			if recordType == "S" {
				s3Err := h.S3.PutObject(s3Key, []byte(finalData))
				if s3Err != nil {
					log.Printf("POST: Failed to store data in S3: %v", s3Err)
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
				log.Printf("POST: Successfully stored data in S3 - Key: %s", s3Key)
			}

			insertErr = h.DB.PutRedirect(record)
			if insertErr == nil {
				log.Printf("POST: Successfully stored metadata - Code: %s (%d chars)", code, currentCodeLength)

				// Set the owner ID cookie (30 days expiration, no HttpOnly)
				c.SetCookie("id", ownerID, 30*24*60*60, "/", "", false, false)

				// Get session and set test value
				session := sessions.Default(c)

				// Check if session already exists (has any values)
				existingTest := session.Get("test")
				if existingTest != nil {
					log.Printf("Extending existing session with test=%v", existingTest)
				}

				// Set/update test value - this marks session as modified
				session.Set("test", "a")

				// Save session - this automatically:
				// 1. Preserves all existing session values
				// 2. Re-signs the cookie with the current key
				// 3. Sets a new expiration 30 days from now (using store's MaxAge)
				if err := session.Save(); err != nil {
					log.Printf("Failed to save session: %v", err)
				}

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

				// Return response based on whether this was form input
				if isFormInput {
					// For form input, redirect to info page like old POST behavior
					redirectPath := fmt.Sprintf("/%s?from=success", code)
					// Preserve html parameter if present
					if c.Request.URL.Query().Has("html") {
						redirectPath += "&html"
					}
					c.Redirect(http.StatusSeeOther, redirectPath)
				} else {
					// For raw input, return plain text URL like PUT
					c.String(http.StatusOK, fullURL+"\n")
				}
				return
			}

			// Check if error is due to duplicate key
			if !isDuplicateKeyError(insertErr) {
				// Some other error occurred
				log.Printf("DynamoDB error (not duplicate key): %v", insertErr)
				c.String(http.StatusInternalServerError, "Error: Failed to store data\n")
				return
			}
			log.Printf("POST: Duplicate key error for %d-char code, retrying. Error: %v", currentCodeLength, insertErr)
			// Continue to next attempt if duplicate key
		}
	}

	// All attempts failed
	c.String(529, "Error: Could not allocate URL in the target namespace.\n")
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

	// Sleep for 500ms to allow DynamoDB to sync the delete
	time.Sleep(500 * time.Millisecond)

	// Always redirect for browser clients to the item URL (which will now 404)
	if utils.ShouldReturnHTML(c) {
		c.Redirect(http.StatusSeeOther, "/"+code+"?from=delete")
	} else {
		// For API clients, return plain text success
		c.String(http.StatusOK, "Deleted successfully")
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var ccf *types.ConditionalCheckFailedException
	return errors.As(err, &ccf)
}
