# xipe - Pastebin Service

A high-performance pastebin service for xi.pe, built with Go, AWS DynamoDB, and S3. Creates short, memorable codes using 4-5 character alphanumeric identifiers with 7-day automatic expiration.

## Features

- **Pastebin Service**: Store and share text/code snippets with configurable expiration (default: 7 days)
- **Hybrid Storage**: Small files (≤10KB) in DynamoDB, large files (>10KB, ≤2MB) in S3 with zstd compression (thresholds configurable)
- **Syntax Highlighting**: Automatic code syntax highlighting with highlight.js
- **High Performance**: In-memory LRU cache with TTL support
- **REST API**: JSON API with optional form-encoded input support
- **Static Pages**: Built-in support for static content pages
- **Owner Authentication**: Delete functionality with secure 128-bit tokens
- **Automatic Cleanup**: All pastes expire after configurable TTL (default: 7 days)

## Quick Start

### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/drewstreib/xipe-go.git
cd xipe-go

# Set up AWS credentials (needs DynamoDB and S3 access)
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1

# Optional: Configure application settings
export PASTE_TTL=604800              # 7 days (default)
export PASTE_MAX_SIZE=2097152        # 2MB (default)

# Run the service
docker-compose up -d

# Service will be available at http://localhost:8080
```

### Building from Source

```bash
# Requirements: Go 1.24.3 or later
go mod download
go build -o xipe .
./xipe
```

## API Usage

### Create Paste

```bash
# Create paste with plain text (default, 7-day expiration, 4-5 character code)
curl -X POST "http://localhost:8080/" \
  -d "Hello, world!"
# Response: http://localhost:8080/Ab3d

# Form-encoded alternative (for HTML forms only):
curl -X POST "http://localhost:8080/?input=form" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "data=Hello%20world%21"
# Response: http://localhost:8080/XyZ9
```

### Storage Architecture

xipe uses a hybrid storage approach for optimal performance:

- **Small Files (≤configurable size, default: 10KB)**: Stored directly in DynamoDB for fast access
- **Large Files (>cutoff size, ≤configurable max, default: 2MB)**: Content stored in S3 with zstd compression, metadata in DynamoDB
- **All files**: Configurable expiration (default: 7 days)
- **Code length**: 4-5 characters (randomly generated with multiple allocation attempts before failing)

### Access Paste

Navigate to `http://localhost:8080/[code]` to see the paste with:
- Syntax highlighting (toggleable)
- Line numbers (toggleable)
- Copy URL and Copy Text buttons
- Delete button (if you're the owner)
- Creation timestamp and expiration countdown
- Raw text access via `?raw` parameter

## Configuration

### Environment Variables

**AWS Configuration:**
- `AWS_ACCESS_KEY_ID` - AWS access key (needs DynamoDB and S3 permissions)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_REGION` - AWS region (default: us-east-1)

**Application Configuration:**
- `PASTE_TTL` - Paste expiration time in seconds (default: 604800 = 7 days)
- `PASTE_DYNAMODB_CUTOFF_SIZE` - Size threshold for DynamoDB vs S3 storage in bytes (default: 10240 = 10KB)
- `PASTE_MAX_SIZE` - Maximum paste size in bytes (default: 2097152 = 2MB)
- `CACHE_MAX_ITEMS` - LRU cache maximum number of items (default: 10000)

**Example Configuration:**
```bash
# Override defaults
export PASTE_TTL=86400              # 1 day instead of 7 days
export PASTE_MAX_SIZE=1048576       # 1MB instead of 2MB
export PASTE_DYNAMODB_CUTOFF_SIZE=5120  # 5KB instead of 10KB
export CACHE_MAX_ITEMS=5000         # 5K items instead of 10K
```

### AWS Setup

**DynamoDB Table**: Create `xipe_redirects` with:
- Primary key: `code` (String)
- Enable TTL on `ettl` attribute
- Recommended: Use on-demand billing

**S3 Bucket**: Create `xipe-data` with:
- Private access (public access blocked)
- 30-day lifecycle policy for automatic cleanup
- Same region as DynamoDB for optimal performance

## Development

### Prerequisites

- Go 1.24.3 or later
- AWS credentials with DynamoDB and S3 access
- Make (for build commands)

### Build Commands

```bash
make build      # Build binary
make test       # Run tests
make run        # Run locally
make fmt        # Format code
make lint       # Run linting
make pre-commit # Run all checks before committing
```

### Install Pre-commit Hooks

```bash
make install-hooks
```

This ensures code formatting and tests pass before commits.

### Container Building with Ko

```bash
# Install ko
go install github.com/google/ko@latest

# Build and push container
export KO_DOCKER_REPO=docker.io/yourusername
ko build .
```

## Security Features

- **Owner-based Deletion**: Only creators can delete their pastes (128-bit secure tokens)
- **Size Limits**: 2MB maximum paste size
- **IP Tracking**: Creator IP stored for abuse prevention
- **Input Sanitization**: Protection against XSS and injection attacks
- **Secure Code Generation**: Cryptographically random codes with collision handling
- **Same-error Responses**: Prevents enumeration attacks on deletion endpoints

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│   Client    │────▶│  Gin HTTP   │────▶│   DynamoDB   │
└─────────────┘     │   Server    │     │  (metadata)  │
                    └─────────────┘     └──────────────┘
                           │                     ▲
                           ▼                     │
                    ┌─────────────┐              │
                    │  LRU Cache  │──────────────┘
                    │ (1hr TTL)   │
                    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │     S3      │
                    │ (large files│
                    │ + compress) │
                    └─────────────┘
```

### Key Components

- **Web Framework**: Gin (high-performance HTTP)
- **Database**: AWS DynamoDB (metadata and small files)
- **Object Storage**: AWS S3 (large files with zstd compression)
- **Cache**: HashiCorp golang-lru/v2 (1-hour TTL, respects DynamoDB expiration)
- **Compression**: klauspost/compress (zstd level 3)
- **Syntax Highlighting**: highlight.js with line numbers plugin

## API Reference

### POST /

Create a new paste.

**Default: Plain Text Body**:
```bash
POST /
Content-Type: text/plain

Your text or code here
```

**Alternative (Form-encoded for HTML forms)**:
```bash
POST /?input=form
Content-Type: application/x-www-form-urlencoded

data=Your%20text%20here
```

**Response** (plain text):
```
http://localhost:8080/Ab3d
```

### GET /:code

Retrieve a paste.

**Request**:
```bash
GET /Ab3d
```

**Response** (plain text):
```
Your stored text or code content
```

**Browser Response**: HTML page with syntax highlighting

### DELETE /:code

Delete a paste (requires owner cookie).

**Request**:
```bash
DELETE /Ab3d
Cookie: id=<owner-token>
```

**Response** (plain text):
```
Deleted successfully
```

### Error Responses

All errors return plain text for non-browser clients:

```
Error 404: Short URL not found or has expired
Error 401: unauthorized
Error 403: Data too long
Error 500: Internal server error
```

Browser clients receive styled HTML error pages.

### API Response Format Summary

- **Plain text is the default**: All API responses use plain text (URLs for success, "Error {code}: {message}" for errors)
- **HTML for browsers**: Browser clients (detected by User-Agent) receive HTML pages
- **No JSON responses**: The API uses plain text exclusively for programmatic access
- **Form support**: The `?input=form` parameter exists solely for HTML form compatibility

**Response**: 
- `200` - Successfully deleted
- `401` - Unauthorized (no cookie or wrong owner)
- `404` - Paste not found

### GET /api/stats

Get service statistics (cache size, etc).

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run pre-commit checks (`make pre-commit`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/drewstreib/xipe-go/issues)
- **Abuse Contact**: abuse@xi.pe

## Acknowledgments

- Built with [Gin Web Framework](https://github.com/gin-gonic/gin)
- Container images built with [ko](https://ko.build/)
- Hosted at [alt.org](https://alt.org)