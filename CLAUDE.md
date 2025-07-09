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
- **Endpoint**: `/api/urlpost?ttl=<1d|1w|1m>&url=<url_encoded_target>`
- **TTL Options**:
  - `1d`: 4-char code, expires in 24 hours
  - `1w`: 5-char code, expires in 1 week  
  - `1m`: 6-char code, expires in 1 month
- **Code Generation**: Cryptographically random alphanumeric
- **Retry Logic**: Up to 5 attempts on collision (returns 529 on failure)
- **Storage**: DynamoDB table "xipe_redirects" with conditional writes

### 2. URL Redirection
- **Pattern**: `/[a-zA-Z0-9]{4,6}`
- **Behavior**: 301 permanent redirect to stored URL
- **Fallthrough**: Catches all unmatched routes
- **Not Found**: Returns 404 if code doesn't exist or has expired

### 3. Static Website
- **Endpoint**: `/`
- **Content**: Usage instructions and service information
- **Stats**: `/stats` endpoint for service metrics

## Database Schema
```
DynamoDB Table: xipe_redirects
Primary Key: code (string)
Attributes:
  - code: string (4-6 chars, auto-generated)
  - typ: string (always "R" for redirects)
  - val: string (target URL)
  - ettl: number (optional, TTL in epoch seconds)

TTL Configuration:
  - Enable TTL on 'ettl' attribute in DynamoDB
  - Items automatically expire after TTL timestamp
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
make lint       # Run linting
make fmt        # Format code

# Ko build commands
make ko-build     # Build container image locally
make ko-publish   # Build and publish to registry
make ko-multiarch # Build for multiple architectures
make ko-apply     # Deploy to Kubernetes
```

## Container Building with Ko

This project uses [ko](https://ko.build/) for building minimal container images without Dockerfiles. Ko automatically:
- Builds a minimal container with just the Go binary
- Uses distroless base images for security
- Embeds static files via Go's embed directive
- Supports multi-platform builds (amd64/arm64)

### Prerequisites for Ko
```bash
# Install ko
go install github.com/google/ko@latest

# Set default container registry (optional)
export KO_DOCKER_REPO=gcr.io/my-project
# or
export KO_DOCKER_REPO=docker.io/myuser
```

### Building Images
```bash
# Build locally (for testing)
ko build --local .

# Build and push to registry
ko build .

# Deploy to Kubernetes
ko apply -f config/
```

## Configuration
- **Port**: 8080 (hardcoded in main.go)
- **AWS Region**: us-east-1 (hardcoded in db/dynamodb.go)
- **Table Name**: xipe_redirects (hardcoded in db/dynamodb.go)
- **DynamoDB Requirements**:
  - Create table with 'code' as primary key (String)
  - Enable TTL on 'ettl' attribute
  - Recommended: On-demand billing for unpredictable traffic

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

## Deployment

### Quick Start with Docker
```bash
# Pull and run with Docker Compose
git clone https://github.com/drewstreib/xipe-go.git
cd xipe-go

# Set up environment variables for AWS
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1

# Run the service
docker-compose up -d

# Check logs
docker-compose logs -f
```

### Image Authentication (if needed)
```bash
# For private repositories, authenticate with GHCR
gh auth token | docker login ghcr.io -u $(gh api user --jq .login) --password-stdin
```

### Requirements
- AWS credentials with DynamoDB access
- DynamoDB table "xipe_redirects" must exist with:
  - Primary key: "code" (String)
  - TTL enabled on "ettl" attribute
- Docker/Docker Compose for containerized deployment
- Go 1.24.3 or later for building from source

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed deployment instructions.

## CI/CD Pipeline
- **GitHub Actions**: Automated builds on main branch pushes
- **Multi-architecture**: Builds for linux/amd64 and linux/arm64
- **Container Registry**: Images published to ghcr.io/drewstreib/xipe-go
- **Security**: SBOM generation and vulnerability scanning
- **Quality**: Automated testing and linting with golangci-lint

### Multi-Architecture Build Notes
- **Ko Configuration**: Uses `--bare` flag to prevent module path appending
- **Architecture Support**: Properly builds native binaries for both AMD64 and ARM64
- **Image Naming**: Final images published as `ghcr.io/drewstreib/xipe-go:latest`
- **Docker Compose**: Uses the clean image name without module path

### Troubleshooting Multi-Arch Builds
If experiencing "exec format error" on ARM64 machines:
1. Ensure `.ko.yaml` does NOT hardcode `GOARCH=amd64` in env section
2. Verify `--bare` flag is used in GitHub Actions ko build
3. Check manifest with: `docker manifest inspect ghcr.io/drewstreib/xipe-go:latest`
4. Force pull for your architecture if needed

## API Examples
```bash
# Create short URL with 1-day TTL (4 char code)
curl "http://localhost:8080/api/urlpost?ttl=1d&url=https%3A%2F%2Fexample.com"
# Response: {"status":"success","code":"Ab3d","url":"https://example.com","ttl":"1d"}

# Create short URL with 1-week TTL (5 char code)
curl "http://localhost:8080/api/urlpost?ttl=1w&url=https%3A%2F%2Fexample.com"

# Create short URL with 1-month TTL (6 char code)
curl "http://localhost:8080/api/urlpost?ttl=1m&url=https%3A%2F%2Fexample.com"

# Use short URL
curl -L "http://localhost:8080/Ab3d"
```

## Implementation Notes

### Code Generation
- Uses crypto/rand for secure random generation
- Character set: a-z, A-Z, 0-9 (62 characters)
- Collision handling: Retries with new random code
- No sequential or predictable patterns

### DynamoDB Optimization
- Conditional writes prevent race conditions
- TTL reduces storage costs via automatic cleanup
- Single table design for simplicity
- Consider read/write capacity based on traffic

### Error Handling
- 400: Invalid parameters (ttl, url format)
- 404: Code not found or expired
- 500: Database errors
- 529: Unable to generate unique code (very rare)

### Security Considerations
- No user input in redirect codes (prevents enumeration)
- URL validation prevents open redirect vulnerabilities
- TTL limits abuse potential
- Consider rate limiting for production