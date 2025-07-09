# xipe - URL Shortener Service

A high-performance URL shortener service for xi.pe, built with Go and AWS DynamoDB. Creates short, memorable URLs using 4-8 character alphanumeric codes with automatic expiration.

## Features

- **URL Shortening**: Generate short URLs with customizable expiration times
- **Security First**: No automatic redirects - shows info page with target URL
- **Content Filtering**: Built-in malicious URL detection using Cloudflare DNS
- **High Performance**: In-memory LRU cache with TTL support
- **REST API**: JSON API with optional form-encoded input support
- **Automatic Cleanup**: URLs expire automatically based on TTL settings

## Quick Start

### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/drewstreib/xipe-go.git
cd xipe-go

# Set up AWS credentials
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

### Create Short URL

```bash
# Create short URL with 1-day expiration (4 character code)
curl -X POST "http://localhost:8080/api/urlpost" \
  -H "Content-Type: application/json" \
  -d '{"ttl":"1d","url":"https://example.com"}'

# Response:
# {"status":"ok","url":"http://localhost:8080/Ab3d"}
```

### TTL Options

- `1d` - 1 day expiration (4 character code)
- `1w` - 1 week expiration (5 character code)  
- `1mo` - 1 month expiration (6 character code)

### Access Short URL

Navigate to `http://localhost:8080/[code]` to see the info page with:
- Original URL (clickable)
- Creation timestamp
- Expiration time
- Short URL for copying

## Configuration

### Environment Variables

- `AWS_ACCESS_KEY_ID` - AWS access key for DynamoDB
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_REGION` - AWS region (default: us-east-1)
- `CACHE_SIZE` - LRU cache size (default: 10000)

### DynamoDB Setup

Create a DynamoDB table named `xipe_redirects` with:
- Primary key: `code` (String)
- Enable TTL on `ettl` attribute
- Recommended: Use on-demand billing

## Development

### Prerequisites

- Go 1.24.3 or later
- AWS credentials with DynamoDB access
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

- **No Automatic Redirects**: Users see target URL before visiting
- **Content Filtering**: URLs checked against Cloudflare family DNS
- **URL Validation**: Only accepts properly formatted http/https URLs
- **IP Tracking**: Creator IP stored for abuse prevention
- **Input Sanitization**: Protection against XSS and injection attacks

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│   Client    │────▶│  Gin HTTP   │────▶│   DynamoDB   │
└─────────────┘     │   Server    │     └──────────────┘
                    └─────────────┘              ▲
                           │                     │
                           ▼                     │
                    ┌─────────────┐              │
                    │  LRU Cache  │──────────────┘
                    └─────────────┘
```

### Key Components

- **Web Framework**: Gin (high-performance HTTP)
- **Database**: AWS DynamoDB (NoSQL)
- **Cache**: HashiCorp golang-lru/v2 (TTL-aware)
- **DNS Filter**: Cloudflare DNS over HTTPS

## API Reference

### POST /api/urlpost

Create a shortened URL.

**Request Body (JSON)**:
```json
{
  "ttl": "1d",
  "url": "https://example.com"
}
```

**Response**:
```json
{
  "status": "ok",
  "url": "http://localhost:8080/Ab3d"
}
```

**Error Responses**:
- `400` - Invalid parameters
- `403` - URL blocked by content filter
- `500` - Internal server error
- `503` - DNS service unavailable
- `529` - Unable to generate unique code

### GET /[code]

Display info page for shortened URL.

**Response**: HTML page with URL information

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