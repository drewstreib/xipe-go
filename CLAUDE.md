# xipe - URL Shortener & Pastebin Service

## Project Overview
xipe (zippy) is a high-performance URL shortener and pastebin service for xi.pe, built with Go and Gin framework. The service provides short, memorable URLs using 4-8 character alphanumeric codes.

## Architecture

### Tech Stack
- **Language**: Go 1.24.3
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **Database**: AWS DynamoDB
- **Cache**: HashiCorp golang-lru/v2/expirable (LRU cache with TTL)
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
├── utils/              # Utility functions
│   ├── codegen.go      # Code generation utilities
│   ├── response.go     # HTTP response utilities
│   └── url_check.go    # URL content filtering
├── templates/          # HTML templates
│   ├── index.html      # Landing page
│   ├── info.html       # URL redirect info page
│   └── data.html       # Pastebin data display page
└── test files (*_test.go)
```

## Key Features

### 1. URL Shortening & Pastebin Service
- **Endpoint**: `POST /` (previously /api/post, /api/urlpost)
- **Method**: POST (required)
- **Method**: POST (required)
- **Input Format**: JSON body (default) or URL-encoded form data with `?input=urlencoded`
- **Supports**: Both URL shortening and pastebin/data storage
- **TTL Options**:
  - `1d`: 4-char code, expires in 24 hours
  - `1w`: 5-char code, expires in 1 week  
  - `1mo`: 6-char code, expires in 1 month
- **Code Generation**: Cryptographically random alphanumeric
- **Retry Logic**: Up to 5 attempts on collision (returns 529 on failure)
- **Storage**: DynamoDB table "xipe_redirects" with conditional writes
- **Content Filtering**: DNS-based URL filtering using Cloudflare family DNS
- **Pastebin Features**:
  - Store up to 50KB of text data (vs 4KB for URLs)
  - Syntax highlighting with highlight.js
  - Dynamic line numbers toggle
  - Optional syntax highlighting toggle
  - Clean copy functionality regardless of display mode

### 2. URL Redirection & Data Display
- **Pattern**: `/[a-zA-Z0-9]{4,6}`
- **Behavior**: 
  - For URLs (typ="R"): Shows info page with target URL and metadata
  - For Data (typ="D"): Shows data page with syntax highlighting and copy options
- **Fallthrough**: Catches all unmatched routes
- **Not Found**: Returns 404 if code doesn't exist or has expired

### 3. URL Content Filtering
- **Method**: DNS over HTTPS queries to Cloudflare family DNS
- **Endpoint**: `https://family.cloudflare-dns.com/dns-query`
- **Detection**: URLs returning 0.0.0.0 are blocked (malicious/inappropriate content)
- **Error Handling**: 503 for DNS unavailable, 403 for blocked content
- **Timeout**: 10-second timeout for DNS queries

### 4. Static Website
- **Endpoint**: `/`
- **Content**: Usage instructions and service information
- **Stats**: `/api/stats` endpoint for service metrics

## Database Schema
```
DynamoDB Table: xipe_redirects
Primary Key: code (string)
Attributes:
  - code: string (4-6 chars, auto-generated)
  - typ: string ("R" for redirects, "D" for data/pastebin)
  - val: string (target URL)
  - ettl: number (optional, TTL in epoch seconds)
  - created: number (creation timestamp in epoch seconds)
  - ip: string (creator's IP address)

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
make pre-commit # Run all pre-commit checks (fmt + test + lint)

# Ko build commands
make ko-build     # Build container image locally
make ko-publish   # Build and publish to registry
make ko-multiarch # Build for multiple architectures
make ko-apply     # Deploy to Kubernetes

# Development setup
make install-hooks # Install pre-commit hooks
```

## Pre-commit Hooks
To prevent CI failures, install pre-commit hooks:
```bash
make install-hooks
```

This will automatically run `gofmt`, `goimports`, and tests before each commit.

**IMPORTANT**: Always run `make pre-commit` before committing to ensure CI passes.

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
- **Cache Size**: 10000 (configurable via CACHE_SIZE environment variable)
- **Cache TTL**: 1 hour (hardcoded)
- **DynamoDB Requirements**:
  - Create table with 'code' as primary key (String)
  - Enable TTL on 'ettl' attribute
  - Recommended: On-demand billing for unpredictable traffic

## Future Enhancements
- User registration and API keys
- ✅ Pastebin functionality (completed)
- Usage analytics
- Custom domains
- Rate limiting
- Configuration via environment variables

## Performance Considerations
- **In-Memory LRU Cache**: 10K item cache with 1-hour TTL reduces DynamoDB load
- **Cache Logic**: Honors DynamoDB TTL by checking expiration before serving cached results
- **DynamoDB session reuse** for connection pooling
- **Lightweight Gin framework** for minimal overhead
- **Simple key-value lookups** for fast redirects
- Regex validation cached at compile time

## Security Notes
- Input validation on all user-provided keys
- URL content filtering via Cloudflare family DNS
- Protocol validation (requires http:// or https://)
- No user authentication (planned for future)
- SQL injection not possible (NoSQL database)
- XSS protection through Go's html/template
- Content Security Policy (CSP) headers with 'unsafe-inline' for required functionality
- Proper HTML escaping for pastebin data display

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
- **Architecture**: Builds for linux/arm64 only (AMD64 disabled for faster builds)
- **Container Registry**: Images published to ghcr.io/drewstreib/xipe-go
- **Security**: SBOM generation and vulnerability scanning
- **Quality**: Automated testing and linting with golangci-lint

### Build Architecture Notes
- **Ko Configuration**: Uses `--base-import-paths` flag for clean module naming
- **Architecture Support**: Currently ARM64 only (AMD64 commented out for speed)
- **Image Naming**: Final images published as `ghcr.io/drewstreib/xipe-go/xipe:latest`
- **Docker Compose**: Uses the full image path including module name
- **Re-enabling AMD64**: Uncomment platform in `.ko.yaml` and add `linux/amd64` to build commands

### Troubleshooting Multi-Arch Builds
If experiencing "exec format error" on ARM64 machines:
1. Ensure `.ko.yaml` does NOT hardcode `GOARCH=amd64` in env section
2. Verify `--base-import-paths` flag is used in GitHub Actions ko build
3. Check manifest with: `docker manifest inspect ghcr.io/drewstreib/xipe-go/xipe:latest`
4. Force pull for your architecture if needed

### Common CI Failure Prevention
The most common CI failures are due to formatting issues. To prevent these:

1. **Install pre-commit hooks**: `make install-hooks`
2. **Always run before committing**: `make pre-commit`
3. **Or run individual checks**:
   - `make fmt` - Fix formatting
   - `make test` - Run tests
   - `make lint` - Run linting

**Pro tip**: The pre-commit hooks will automatically format your code and run tests, preventing most CI failures.

## API Examples

### JSON Format (Default)
```bash
# Create short URL with 1-day TTL (4 char code)
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{"ttl":"1d","url":"https://example.com"}'
# Response: {"status":"ok","url":"http://localhost:8080/Ab3d"}

# Store pastebin data
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{"ttl":"1d","data":"Hello, world!"}'
# Response: {"status":"ok","url":"http://localhost:8080/XyZ9"}
```

### URL-Encoded Form Data (Legacy)
```bash
# Create short URL using form data (for HTML forms)
curl -X POST "http://localhost:8080/?input=urlencoded" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ttl=1d&url=https%3A%2F%2Fexample.com"

# Store data using form data
curl -X POST "http://localhost:8080/?input=urlencoded" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ttl=1d&data=Hello%20world%21"
```

### Using Short URLs
```bash
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

### Input Formats
- **JSON (Default)**: `POST /` with `{"ttl":"1d","url":"https://example.com"}` or `{"ttl":"1d","data":"content"}` in body
- **URL-encoded**: `POST /?input=urlencoded` with form data
- **Required Fields**: `ttl` (1d|1w|1mo) and either `url` or `data` (not both)
- **Size Limits**: 4KB for URLs, 50KB for data

### Error Handling
- 400: Invalid parameters (ttl, url format, missing hostname, malformed JSON)
- 403: URL blocked by content filter, URL too long (4KB max), data too long (50KB max), or missing protocol
- 404: Code not found or expired
- 500: Database errors
- 503: DNS service unavailable
- 529: Unable to generate unique code (very rare)

### Security Considerations
- No user input in redirect codes (prevents enumeration)
- URL validation prevents open redirect vulnerabilities
- DNS-based content filtering blocks malicious URLs
- TTL limits abuse potential
- Consider rate limiting for production

### URL Content Filtering
- **Implementation**: `utils.URLCheck()` function in `utils/url_check.go`
- **DNS Provider**: Cloudflare family DNS (family.cloudflare-dns.com)
- **Query Method**: DNS over HTTPS with JSON responses
- **Blocking Logic**: URLs resolving to 0.0.0.0 are considered blocked
- **Performance**: 10-second timeout prevents hanging requests
- **Error Handling**: Graceful degradation on DNS service unavailability

### Pastebin Display Features
- **Syntax Highlighting**: Uses highlight.js for code syntax highlighting
- **Line Numbers**: Optional line numbers with highlightjs-line-numbers.js plugin
- **Display Modes**: 4 combinations - plain/highlighted × with/without line numbers
- **Copy Functionality**: Always copies original plain text regardless of display formatting
- **Text Trimming**: Client-side trimming to 50KB with proper UTF-8 byte counting
- **Line Ending Handling**: Accounts for \n → \r\n conversion during form submission