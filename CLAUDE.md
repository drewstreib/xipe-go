# xipe - Pastebin Service

## Project Overview
xipe (zippy) is a high-performance pastebin service for xi.pe, built with Go and Gin framework. The service provides short, memorable codes using 4-6 character alphanumeric identifiers.

## Architecture

### Tech Stack
- **Language**: Go 1.24.3
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **Database**: AWS DynamoDB
- **Object Storage**: AWS S3 (for files >configurable cutoff, default: 10KB)
- **Cache**: HashiCorp golang-lru/v2/expirable (LRU cache with TTL)
- **Testing**: testify/assert, testify/mock

### Project Structure
```
xipe-go/
├── main.go              # Application entry point
├── config/              # Configuration management
│   └── config.go       # Environment variable loading and defaults
├── handlers/            # HTTP request handlers
│   ├── root.go         # Root page and stats endpoints
│   ├── api.go          # API endpoints
│   └── data.go         # Data display handler
├── db/                 # Database layer
│   ├── dynamodb.go     # DynamoDB implementation
│   └── mock.go         # Mock DB for testing
├── utils/              # Utility functions
│   ├── codegen.go      # Code generation utilities
│   └── response.go     # HTTP response utilities
├── templates/          # HTML templates
│   ├── index.html      # Landing page
│   └── data.html       # Pastebin data display page
└── test files (*_test.go)
```

## Key Features

### 1. Pastebin Service
- **Endpoint**: `POST /`
- **Method**: POST (required)
- **Input Format**: JSON body (default) or URL-encoded form data with `?input=urlencoded`
- **TTL**: Configurable via environment variable (default: 7 days)
- **Code Generation**: Cryptographically random alphanumeric (4-5 characters)
- **Retry Logic**: 3 attempts with 4-character codes, then 3 attempts with 5-character codes on collision (returns 529 on failure)
- **Storage**: DynamoDB table "xipe_redirects" with conditional writes
- **Owner Authentication**: 128-bit random tokens for deletion access
- **Features**:
  - Store up to configurable size of text data (configurable storage cutoff for DynamoDB vs S3, defaults: 10KB cutoff, 2MB max)
  - Syntax highlighting with highlight.js
  - Dynamic line numbers toggle
  - Optional syntax highlighting toggle
  - Clean copy functionality regardless of display mode

### 2. Data Display
- **Pattern**: `/[a-zA-Z0-9]{4,5}` (or static page names)
- **Behavior**: 
  - **Static Pages**: Reserved codes (e.g., `/privacy`) serve embedded content from `utils/pages/*.txt`
  - Shows data page with syntax highlighting and copy options
  - **URL Parameters**: `?noh` disables syntax highlighting, `?raw` returns plain text
- **Fallthrough**: Catches all unmatched routes
- **Not Found**: Returns 404 if code doesn't exist or has expired

### 3. Deletion Functionality
- **Endpoint**: `DELETE /:code`
- **Authentication**: Owner ID cookie (`id=<128-bit-token>`) required
- **Cookie Management**: 
  - Generated on first post, reused for subsequent posts
  - 30-day expiration, refreshed on each post
  - HttpOnly flag for security
- **Security**: Same error response for both "not found" and "wrong owner" (401 Unauthorized)
- **Database**: Conditional delete with owner verification
- **Cache Invalidation**: Automatic removal from LRU cache on successful delete


### 4. Static Pages System
- **Purpose**: Serve static content pages using short URL codes (e.g., `/privacy`)
- **Implementation**: 
  - Files stored in `utils/pages/*.txt` and embedded at build time
  - Reserved codes automatically loaded from filenames (without .txt extension)
  - Code generation avoids reserved codes (with 5 retry attempts)
- **Display**: Uses same data.html template as pastebin content
- **Access**: Both browser (HTML) and API (plain text) access supported
- **Management**: Manual - add/remove .txt files in pages directory and rebuild

### 5. Static Website
- **Endpoint**: `/`
- **Content**: Usage instructions and service information
- **Stats**: `/api/stats` endpoint for service metrics

## Storage Architecture

### Hybrid Storage System
xipe uses a hybrid storage approach to efficiently handle files of different sizes:

- **Small Files (≤configurable cutoff, default: 10KB)**: Stored directly in DynamoDB for fast access
- **Large Files (>cutoff, ≤configurable max, default: 2MB)**: Content stored in S3, metadata in DynamoDB

### Storage Decision Logic
1. **Data ≤PASTE_DYNAMODB_CUTOFF_SIZE**: Record type "D", content stored in DynamoDB `val` field
2. **Data >PASTE_DYNAMODB_CUTOFF_SIZE**: Record type "S", content stored in S3 bucket `xipe-data` with key `S/{code}`, DynamoDB `val` field empty

### S3 Configuration
- **Bucket**: `xipe-data`
- **Region**: `us-east-1` 
- **Object Key Format**: `S/{code}.zst` (e.g., `S/AbC4D.zst`)
- **Compression**: All objects stored with zstd level 3 compression for optimal storage efficiency
- **Access**: Same AWS credentials as DynamoDB
- **Expiration**: Objects expire after 30 days via S3 lifecycle policy (independent of DynamoDB TTL)
- **Overwriting**: Tolerant of overwriting existing objects when codes are reused after expiration
- **Error Handling**: Specific HTTP status codes for different S3 errors (access denied, service unavailable, object not found)

## Database Schema
```
DynamoDB Table: xipe_redirects
Primary Key: code (string)
Attributes:
  - code: string (4-6 chars, auto-generated)
  - typ: string ("D" for DynamoDB storage, "S" for S3 storage)
  - val: string (data content for type "D", empty for type "S")
  - ettl: number (optional, TTL in epoch seconds)
  - created: number (creation timestamp in epoch seconds)
  - ip: string (creator's IP address)
  - owner: string (128-bit base64-encoded token for deletion auth)

TTL Configuration:
  - Enable TTL on 'ettl' attribute in DynamoDB
  - Items automatically expire after TTL timestamp
  - S3 objects cleaned up separately via lifecycle policy
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

### Environment Variables
All configuration is handled via environment variables with sensible defaults:

**Application Settings:**
- `PASTE_TTL` - Paste expiration time in seconds (default: 604800 = 7 days)
- `PASTE_DYNAMODB_CUTOFF_SIZE` - Size threshold for DynamoDB vs S3 storage in bytes (default: 10240 = 10KB)
- `PASTE_MAX_SIZE` - Maximum paste size in bytes (default: 2097152 = 2MB)
- `CACHE_MAX_ITEMS` - LRU cache maximum number of items (default: 10000)

**AWS Settings:**
- `AWS_ACCESS_KEY_ID` - AWS access key (required)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key (required)
- `AWS_REGION` - AWS region (default: us-east-1)

**Hardcoded Settings:**
- **Port**: 8080 (hardcoded in main.go)
- **Table Name**: xipe_redirects (hardcoded in db/dynamodb.go)
- **S3 Bucket**: xipe-data (hardcoded in db/s3.go)
- **Cache TTL**: 1 hour (hardcoded)

**Configuration Example:**
```bash
# Custom configuration
export PASTE_TTL=86400              # 1 day expiration
export PASTE_MAX_SIZE=1048576       # 1MB max size
export PASTE_DYNAMODB_CUTOFF_SIZE=5120  # 5KB DynamoDB cutoff
export CACHE_MAX_ITEMS=5000         # 5K cache items
```

**DynamoDB Requirements:**
- Create table with 'code' as primary key (String)
- Enable TTL on 'ettl' attribute
- Recommended: On-demand billing for unpredictable traffic

## Future Enhancements
- User registration and API keys
- ✅ Pastebin functionality (completed)
- ✅ Configuration via environment variables (completed)
- Usage analytics
- Custom domains
- Rate limiting

## Performance Considerations
- **In-Memory LRU Cache**: Configurable max items (default: 10K items) with 1-hour TTL reduces DynamoDB load
- **Cache Logic**: Honors DynamoDB TTL by checking expiration before serving cached results
- **DynamoDB session reuse** for connection pooling
- **S3 Compression**: zstd level 3 compression reduces storage costs and transfer times
- **Lightweight Gin framework** for minimal overhead
- **Simple key-value lookups** for fast retrieval
- Regex validation cached at compile time

## Security Notes
- Input validation on all user-provided keys
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
- AWS credentials with DynamoDB and S3 access
- DynamoDB table "xipe_redirects" must exist with:
  - Primary key: "code" (String)
  - TTL enabled on "ettl" attribute
- S3 bucket "xipe-data" must exist with:
  - Public access blocked (private bucket)
  - 30-day lifecycle policy for object expiration (optional)
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

### Raw Text (Default)
```bash
# Store pastebin data with 1-day TTL (4-5 char code)
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: text/plain" \
  -d "Hello, world!"
# Response: http://localhost:8080/XyZ9
```

### Form-Encoded Data (for HTML forms)
```bash
# Store data using form data (for HTML forms)
curl -X POST "http://localhost:8080/?input=form" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "data=Hello%20world%21"
```

### Retrieving Pastes
```bash
# Get paste content
curl "http://localhost:8080/Ab3d"
```

### Deleting Posts
```bash
# Delete a post (requires owner cookie from creation)
curl -X DELETE "http://localhost:8080/Ab3d" \
  -b "id=<owner-token>" \
  -H "Content-Type: application/json"
# Response: {"status":"ok","message":"deleted successfully"}

# Delete attempt without cookie (fails)
curl -X DELETE "http://localhost:8080/Ab3d"
# Response: {"status":"error","description":"unauthorized"}
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
- **Raw Text (Default)**: `POST /` with raw text content in body
- **Form-encoded**: `POST /?input=form` with form data (`data` field)
- **TTL**: Configurable for all pastes (default: 7 days)
- **Size Limit**: Configurable max size for data (default: 2MB, auto-truncated with UTF-8 preservation)

### Error Handling
- 400: Invalid parameters (ttl, malformed JSON)
- 401: Unauthorized (delete without valid owner cookie, or wrong owner)
- 403: Data too long (configurable max, default: 2MB)
- 404: Code not found or expired, S3 object not found
- 500: Database errors, S3 access denied, storage configuration errors
- 503: S3 service temporarily unavailable
- 529: Unable to generate unique code (very rare)

### Security Considerations
- No user input in codes (prevents enumeration)
- TTL limits abuse potential
- Owner-based deletion prevents unauthorized access
- Same error response for "not found" vs "wrong owner" (prevents enumeration)
- HttpOnly cookies for owner tokens
- 128-bit cryptographically secure owner tokens
- Consider rate limiting for production


### Pastebin Display Features
- **Syntax Highlighting**: Uses highlight.js for code syntax highlighting
- **Line Numbers**: Optional line numbers with highlightjs-line-numbers.js plugin
- **Display Modes**: 4 combinations - plain/highlighted × with/without line numbers
- **Copy Functionality**: Always copies original plain text regardless of display formatting
- **Text Trimming**: Client-side trimming to configurable max size (default: 2MB) with proper UTF-8 byte counting
- **Line Ending Handling**: Accounts for \n → \r\n conversion during form submission
- **URL Parameter Control**: `?noh` parameter disables syntax highlighting (e.g., `/abc123?noh`)
- **URL Synchronization**: Toggling syntax highlighting checkbox updates URL bar and copy functionality
- **Shareable Preferences**: URLs preserve syntax highlighting state when shared