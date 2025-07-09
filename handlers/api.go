package handlers

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"xipe/db"
	"xipe/utils"
)

func URLPostHandler(c *gin.Context) {
	ttl := c.Query("ttl")
	rawURL := c.Query("url")

	if ttl == "" || rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ttl and url parameters are required",
		})
		return
	}

	// Validate TTL
	if ttl != "1d" && ttl != "1w" && ttl != "1m" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ttl must be 1d, 1w, or 1m",
		})
		return
	}

	// Decode and validate URL
	decodedURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid URL encoding",
		})
		return
	}

	_, err = url.ParseRequestURI(decodedURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid URL format",
		})
		return
	}

	// Calculate TTL and code length
	ettl, codeLength, err := utils.CalculateTTL(ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to calculate TTL",
		})
		return
	}

	// Try up to 5 times to insert a unique code
	var code string
	var insertErr error
	for attempts := 0; attempts < 5; attempts++ {
		code, err = utils.GenerateCode(codeLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to generate code",
			})
			return
		}

		redirect := &db.RedirectRecord{
			Code: code,
			Typ:  "R",
			Val:  decodedURL,
			Ettl: ettl,
		}

		insertErr = db.DB.PutRedirect(redirect)
		if insertErr == nil {
			// Success!
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"code":   code,
				"url":    decodedURL,
				"ttl":    ttl,
			})
			return
		}

		// Check if error is due to duplicate key
		if !isDuplicateKeyError(insertErr) {
			// Some other error occurred
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to store URL",
			})
			return
		}
		// Continue to next attempt if duplicate key
	}

	// All attempts failed
	c.JSON(529, gin.H{
		"error": "could not find appropriate insertion slot",
	})
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		return awsErr.Code() == dynamodb.ErrCodeConditionalCheckFailedException
	}
	return false
}
