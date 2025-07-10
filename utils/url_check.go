package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DNSResponse represents the response from Cloudflare DNS over HTTPS
type DNSResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// URLCheckResult represents the result of URL checking
type URLCheckResult struct {
	Allowed bool
	Reason  string
	Status  int // HTTP status code to return
}

// URLCheck validates a URL against Cloudflare's family DNS filter
// Returns:
// - Allowed: true if URL is safe to shorten
// - Status: HTTP status code (503 for DNS unavailable, 403 for blocked content)
// - Reason: human-readable explanation
func URLCheck(targetURL string) URLCheckResult {
	// Parse the URL to extract the hostname
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return URLCheckResult{
			Allowed: false,
			Reason:  "Invalid URL format",
			Status:  400,
		}
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return URLCheckResult{
			Allowed: false,
			Reason:  "No hostname found in URL",
			Status:  400,
		}
	}

	// Query Cloudflare Family DNS over HTTPS
	dnsURL := fmt.Sprintf("https://family.cloudflare-dns.com/dns-query?name=%s&type=A", url.QueryEscape(hostname))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", dnsURL, nil)
	if err != nil {
		return URLCheckResult{
			Allowed: false,
			Reason:  "Failed to create DNS request",
			Status:  503,
		}
	}

	// Set Accept header for JSON response
	req.Header.Set("Accept", "application/dns-json")

	resp, err := client.Do(req)
	if err != nil {
		return URLCheckResult{
			Allowed: false,
			Reason:  "DNS service unavailable",
			Status:  503,
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		return URLCheckResult{
			Allowed: false,
			Reason:  "DNS service unavailable",
			Status:  503,
		}
	}

	var dnsResp DNSResponse
	if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
		return URLCheckResult{
			Allowed: false,
			Reason:  "Failed to parse DNS response",
			Status:  503,
		}
	}

	// Check if DNS query was successful
	if dnsResp.Status != 0 {
		return URLCheckResult{
			Allowed: false,
			Reason:  "DNS resolution failed",
			Status:  503,
		}
	}

	// Check if there are any A records
	if len(dnsResp.Answer) == 0 {
		return URLCheckResult{
			Allowed: false,
			Reason:  "No DNS records found",
			Status:  400,
		}
	}

	// Check if any A record points to 0.0.0.0 (Cloudflare family filter block)
	for _, answer := range dnsResp.Answer {
		if answer.Type == 1 { // A record
			if strings.TrimSpace(answer.Data) == "0.0.0.0" {
				return URLCheckResult{
					Allowed: false,
					Reason:  "URL blocked by content filter",
					Status:  403,
				}
			}
		}
	}

	// URL passed all checks
	return URLCheckResult{
		Allowed: true,
		Reason:  "URL is allowed",
		Status:  200,
	}
}
