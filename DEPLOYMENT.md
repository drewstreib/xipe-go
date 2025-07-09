# Deployment Guide for xipe URL Shortener

## Prerequisites

1. **AWS DynamoDB Setup**:
   ```bash
   # Create the DynamoDB table
   aws dynamodb create-table \
     --table-name xipe_redirects \
     --attribute-definitions \
       AttributeName=code,AttributeType=S \
     --key-schema \
       AttributeName=code,KeyType=HASH \
     --billing-mode PAY_PER_REQUEST \
     --region us-east-1

   # Enable TTL on the 'ettl' attribute
   aws dynamodb update-time-to-live \
     --table-name xipe_redirects \
     --time-to-live-specification \
       Enabled=true,AttributeName=ettl \
     --region us-east-1
   ```

2. **AWS Credentials**: Ensure you have valid AWS credentials with DynamoDB access

## Pulling from GitHub Container Registry (GHCR)

### For machines with `gh auth status`

If you have GitHub CLI authenticated:

```bash
# Authenticate Docker with GHCR using GitHub CLI
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Or use GitHub CLI directly
gh auth token | docker login ghcr.io -u $(gh api user --jq .login) --password-stdin

# Pull the latest image
docker pull ghcr.io/drewstreib/xipe-go:latest
```

### Alternative authentication methods

1. **Using Personal Access Token**:
   ```bash
   # Create a GitHub Personal Access Token with 'read:packages' scope
   # Then login to GHCR
   echo "YOUR_GITHUB_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
   ```

2. **Using GitHub Actions in CI/CD**:
   ```yaml
   - name: Log in to Container Registry
     uses: docker/login-action@v3
     with:
       registry: ghcr.io
       username: ${{ github.actor }}
       password: ${{ secrets.GITHUB_TOKEN }}
   ```

## Docker Compose Deployment

1. **Clone or download the repository**:
   ```bash
   git clone https://github.com/drewstreib/xipe-go.git
   cd xipe-go
   ```

2. **Set up environment variables**:
   ```bash
   # Copy the example environment file
   cp .env.example .env
   
   # Edit .env with your AWS credentials
   vim .env
   ```

3. **Deploy with Docker Compose**:
   ```bash
   # Pull the latest image and start the service
   docker-compose pull
   docker-compose up -d
   
   # View logs
   docker-compose logs -f xipe
   
   # Check health
   curl http://localhost:8080/stats
   ```

4. **Stop the service**:
   ```bash
   docker-compose down
   ```

## Docker Run (Alternative)

```bash
# Run directly with Docker
docker run -d \
  --name xipe-shortener \
  -p 8080:8080 \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_REGION=us-east-1 \
  -e DYNAMODB_TABLE=xipe_redirects \
  --restart unless-stopped \
  ghcr.io/drewstreib/xipe-go:latest
```

## Local Development with DynamoDB Local

For local development without AWS:

```bash
# Uncomment the dynamodb-local service in docker-compose.yml
# Then run:
docker-compose up -d dynamodb-local

# Wait for DynamoDB Local to start, then create the table
aws dynamodb create-table \
  --table-name xipe_redirects \
  --attribute-definitions AttributeName=code,AttributeType=S \
  --key-schema AttributeName=code,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000 \
  --region us-east-1

# Start xipe with local endpoint
docker-compose up -d xipe
```

## Health Checks and Monitoring

The service includes health checks on the `/stats` endpoint:

```bash
# Check if service is healthy
curl http://localhost:8080/stats

# Expected response:
# {"status":"ok","stats":{"total_urls":0,"total_pastes":0}}
```

## Available Image Tags

- `ghcr.io/drewstreib/xipe-go:latest` - Latest build from main branch
- `ghcr.io/drewstreib/xipe-go:main-<sha>` - Specific commit builds
- Multi-architecture support: linux/amd64, linux/arm64

## Troubleshooting

1. **Authentication Issues**:
   ```bash
   # Check if you're logged in to GHCR
   docker system info | grep -A 20 "Registry Credentials"
   
   # Re-authenticate if needed
   gh auth token | docker login ghcr.io -u $(gh api user --jq .login) --password-stdin
   ```

2. **AWS Permissions**:
   - Ensure your AWS credentials have `dynamodb:GetItem`, `dynamodb:PutItem` permissions
   - Verify the DynamoDB table exists and TTL is enabled

3. **Container Logs**:
   ```bash
   # View container logs
   docker-compose logs xipe
   
   # Follow logs in real-time
   docker-compose logs -f xipe
   ```