# xipe - Pastebin Service

A high-performance pastebin service for xi.pe, built with Go and AWS DynamoDB. Creates short, memorable codes using 4-6 character alphanumeric identifiers with automatic expiration.

## Features

- **Pastebin Service**: Store and share text/code snippets with customizable expiration times
- **Syntax Highlighting**: Automatic code syntax highlighting with highlight.js
- **High Performance**: In-memory LRU cache with TTL support
- **REST API**: JSON API with optional form-encoded input support
- **Automatic Cleanup**: Pastes expire automatically based on TTL settings

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

### Create Paste

```bash
# Create paste with 1-day expiration (4 character code)
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{"ttl":"1d","data":"Hello, world!"}'

# Response:
# {"status":"ok","url":"http://localhost:8080/Ab3d"}
```

### TTL Options

- `1d` - 1 day expiration (4 character code)
- `1w` - 1 week expiration (5 character code)  
- `1mo` - 1 month expiration (6 character code)

### Access Paste

Navigate to `http://localhost:8080/[code]` to see the paste with:
- Syntax highlighting (if code detected)
- Line numbers toggle
- Copy button
- Creation timestamp
- Expiration time

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

- **Owner-based Deletion**: Only creators can delete their pastes
- **Size Limits**: 50KB maximum paste size
- **IP Tracking**: Creator IP stored for abuse prevention
- **Input Sanitization**: Protection against XSS and injection attacks
- **Secure Tokens**: 128-bit cryptographically secure owner tokens

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
- **Syntax Highlighting**: highlight.js

## API Reference

### POST /

Create a new paste.

**Request Body (JSON)**:
```json
{
  "ttl": "1d",
  "data": "Your text or code here"
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
- `403` - Data too long (50KB max)
- `500` - Internal server error
- `529` - Unable to generate unique code

### GET /[code]

Display paste content.

**Response**: HTML page with paste content and syntax highlighting

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