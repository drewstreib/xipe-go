package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"xipe/db"
	"xipe/utils"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
)

func URLPostHandler(c *gin.Context) {
	ttl := c.Query("ttl")
	rawURL := c.Query("url")

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

	_, err = url.ParseRequestURI(decodedURL)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "error", "invalid URL format")
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

		insertErr = db.DB.PutRedirect(redirect)
		if insertErr == nil {
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

			// Return response based on client type
			if utils.ShouldReturnHTML(c) {
				c.HTML(http.StatusOK, "success.html", gin.H{
					"url":         fullURL,
					"originalUrl": decodedURL,
					"code":        code,
					"ttl":         ttl,
				})
			} else {
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
			utils.RespondWithError(c, http.StatusInternalServerError, "error", "failed to store URL")
			return
		}
		// Continue to next attempt if duplicate key
	}

	// All attempts failed
	utils.RespondWithError(c, 529, "error", "Could not allocate URL in the target namespace.")
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
