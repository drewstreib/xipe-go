# xipe - URL Shortener & Pastebin Service

## Project Overview
xipe (zippy) is a high-performance URL shortener and pastebin service for xi.pe, built with Go and Gin framework. The service provides short, memorable URLs using 4-8 character alphanumeric codes.

## Architecture

### Tech Stack
- **Language**: Go 1.24.3
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **Database**: AWS DynamoDB
- **Testing**: testify/assert, testify/mock

### Project Structure
```
xipe-go/
├── main.go              # Application entry point
├── handlers/            # HTTP request handlers
│   ├── root.go         # Root page and stats endpoints
│   ├── api.go          # API endpoints (/api/*)
│   └── redirect.go     # URL redirect handler
├── db/                 # Database layer
│   ├── dynamodb.go     # DynamoDB implementation
│   └── mock.go         # Mock DB for testing
├── templates/          # HTML templates
│   └── index.html      # Landing page
└── tests files (*_test.go)
```

## Key Features

### 1. URL Shortening
- **Endpoint**: `/api/urlpost?key=<4-8 chars>&url=<target_url>`
- **Key Format**: 4-8 alphanumeric characters (a-z, A-Z, 0-9)
- **Storage**: DynamoDB table "xipe-urls"

### 2. URL Redirection
- **Pattern**: `/[a-zA-Z0-9]{4,8}`
- **Behavior**: 301 permanent redirect to stored URL
- **Fallthrough**: Catches all unmatched routes

### 3. Static Website
- **Endpoint**: `/`
- **Content**: Usage instructions and service information
- **Stats**: `/stats` endpoint for service metrics

## Database Schema
```
DynamoDB Table: xipe-urls
Primary Key: key (string)
Attributes:
  - key: string (4-8 chars)
  - url: string (target URL)
```

## Testing Strategy
- Unit tests with mocked DynamoDB interface
- Handler isolation using dependency injection
- Test coverage for:
  - Input validation
  - Error handling
  - Success scenarios
  - Edge cases

## Development Commands
```bash
make test       # Run all tests
make run        # Start the server
make build      # Build binary
make deps       # Download dependencies
```

## Configuration
- **Port**: 8080 (hardcoded in main.go)
- **AWS Region**: us-east-1 (hardcoded in db/dynamodb.go)
- **Table Name**: xipe-urls (hardcoded in db/dynamodb.go)

## Future Enhancements
- User registration and API keys
- Pastebin functionality
- Usage analytics
- Custom domains
- Rate limiting
- Configuration via environment variables

## Performance Considerations
- DynamoDB session reuse for connection pooling
- Lightweight Gin framework for minimal overhead
- Simple key-value lookups for fast redirects
- Regex validation cached at compile time

## Security Notes
- Input validation on all user-provided keys
- No user authentication (planned for future)
- SQL injection not possible (NoSQL database)
- XSS protection through Go's html/template

## Deployment Requirements
- AWS credentials with DynamoDB access
- DynamoDB table "xipe-urls" must exist
- Table must have "key" as primary key
- Go 1.24.3 or later

## API Examples
```bash
# Create short URL
curl "http://localhost:8080/api/urlpost?key=test1234&url=https://example.com"

# Use short URL
curl -L "http://localhost:8080/test1234"
```