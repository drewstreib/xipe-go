# xipe - Pastebin Service

A high-performance pastebin service for xi.pe, built with Go, AWS DynamoDB, and S3. Creates short, memorable codes using 4-5 character alphanumeric identifiers with 24-hour automatic expiration.

## Features

- **Pastebin Service**: Store and share text/code snippets with 24-hour expiration
- **Hybrid Storage**: Small files (≤10KB) in DynamoDB, large files (>10KB, ≤2MB) in S3 with zstd compression
- **Syntax Highlighting**: Automatic code syntax highlighting with highlight.js
- **High Performance**: In-memory LRU cache with TTL support
- **REST API**: JSON API with optional form-encoded input support
- **Static Pages**: Built-in support for static content pages
- **Owner Authentication**: Delete functionality with secure 128-bit tokens
- **Automatic Cleanup**: All pastes expire after 24 hours

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
# Create paste (24-hour expiration, 4-5 character code)
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{"data":"Hello, world!"}'  # ttl field no longer needed

# Response:
# {"status":"ok","url":"http://localhost:8080/Ab3d"}

# Form-encoded alternative:
curl -X POST "http://localhost:8080/?input=urlencoded" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "data=Hello%20world%21"

# Raw text alternative using PUT:
curl -X PUT "http://localhost:8080/" \
  -H "Content-Type: text/plain; charset=utf-8" \
  -d "Hello, world!"
```

### Storage Architecture

xipe uses a hybrid storage approach for optimal performance:

- **Small Files (≤10KB)**: Stored directly in DynamoDB for fast access
- **Large Files (>10KB, ≤2MB)**: Content stored in S3 with zstd compression, metadata in DynamoDB
- **All files**: 24-hour expiration (no user-selectable TTL)
- **Code length**: 4-5 characters (cryptographically random, collision-resistant)

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

- `AWS_ACCESS_KEY_ID` - AWS access key (needs DynamoDB and S3 permissions)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_REGION` - AWS region (default: us-east-1)
- `CACHE_SIZE` - LRU cache size (default: 10000)

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

**Request Body (JSON)**:
```json
{
  "data": "Your text or code here"
}
```

**Alternative (Form-encoded)**:
```bash
POST /?input=urlencoded
Content-Type: application/x-www-form-urlencoded

data=Your%20text%20here
```

**Response**:
```json
{
  "status": "ok",
  "url": "http://localhost:8080/Ab3d"
}
```

### PUT /

Create a new paste with raw UTF-8 text directly in the body.

**Request**:
```bash
PUT /
Content-Type: text/plain; charset=utf-8

Your raw text or code content here
```

**Response**:
```json
{
  "status": "ok",
  "url": "http://localhost:8080/Ab3d"
}
```

**Notes**:
- Accepts raw UTF-8 text up to 2MB
- Invalid UTF-8 sequences will return 400 error
- Empty content will return 400 error
- Same 24-hour expiration as POST endpoint

**Error Responses**:
- `400` - Invalid parameters or malformed JSON
- `403` - Data too long (2MB max)
- `500` - Database/S3 errors
- `503` - S3 service temporarily unavailable
- `529` - Unable to generate unique code (very rare)

### GET /[code]

Display paste content.

**Response**: HTML page with paste content and syntax highlighting

**Query Parameters**:
- `?raw` - Return plain text instead of HTML
- `?html` - Force HTML response (default for browsers)

### DELETE /[code]

Delete paste (requires owner cookie).

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