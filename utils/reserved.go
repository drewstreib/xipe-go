package utils

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed pages/*.txt
var pagesFS embed.FS

// ReservedCodes maintains a list of reserved codes that cannot be used for URL shortening
var ReservedCodes = map[string]bool{}

// InitReservedCodes initializes the reserved codes based on embedded files in the pages directory
func InitReservedCodes() error {
	// Read all files in embedded pages directory
	entries, err := pagesFS.ReadDir("pages")
	if err != nil {
		return fmt.Errorf("failed to read pages directory: %w", err)
	}

	// Add each filename (without extension) as a reserved code
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			// Remove .txt extension to get the code
			code := strings.TrimSuffix(entry.Name(), ".txt")
			ReservedCodes[code] = true
		}
	}

	return nil
}

// IsReservedCode checks if a code is reserved
func IsReservedCode(code string) bool {
	return ReservedCodes[code]
}

// GetPageContent reads the content of an embedded page file
func GetPageContent(code string) (string, error) {
	filePath := fmt.Sprintf("pages/%s.txt", code)
	content, err := pagesFS.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
