package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds all configuration values for the application
type Config struct {
	PasteTTL                int64 // TTL in seconds for pastes
	PasteDynamoDBCutoffSize int   // Size threshold for DynamoDB vs S3 storage (bytes)
	PasteMaxSize            int   // Maximum paste size (bytes)
	CacheMaxItems           int   // LRU cache maximum number of items
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	cfg := &Config{
		PasteTTL:                86400 * 7, // 7 days default
		PasteDynamoDBCutoffSize: 10240,     // 10KB default
		PasteMaxSize:            2097152,   // 2MB default
		CacheMaxItems:           10000,     // 10K items default
	}

	// Load from environment variables if present
	if val := os.Getenv("PASTE_TTL"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.PasteTTL = parsed
		} else {
			log.Printf("Warning: Invalid PASTE_TTL value '%s', using default %d", val, cfg.PasteTTL)
		}
	}

	if val := os.Getenv("PASTE_DYNAMODB_CUTOFF_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cfg.PasteDynamoDBCutoffSize = parsed
		} else {
			log.Printf("Warning: Invalid PASTE_DYNAMODB_CUTOFF_SIZE value '%s', using default %d", val, cfg.PasteDynamoDBCutoffSize)
		}
	}

	if val := os.Getenv("PASTE_MAX_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cfg.PasteMaxSize = parsed
		} else {
			log.Printf("Warning: Invalid PASTE_MAX_SIZE value '%s', using default %d", val, cfg.PasteMaxSize)
		}
	}

	if val := os.Getenv("CACHE_MAX_ITEMS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cfg.CacheMaxItems = parsed
		} else {
			log.Printf("Warning: Invalid CACHE_MAX_ITEMS value '%s', using default %d", val, cfg.CacheMaxItems)
		}
	}

	log.Printf("Config loaded - TTL: %ds, DynamoDB cutoff: %d bytes, Max size: %d bytes, Cache max items: %d",
		cfg.PasteTTL, cfg.PasteDynamoDBCutoffSize, cfg.PasteMaxSize, cfg.CacheMaxItems)

	return cfg
}
