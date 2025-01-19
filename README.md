# Go Ollama API v1.0.0

A Go-based API proxy for Ollama with API key management, rate limiting, and webhook support.

## Features

- API key management with rate limiting
- SQLite database for persistent storage
- Webhook notifications for API usage
- Interactive CLI for administration
- Graceful shutdown handling

## Installation

### From Source
```bash
go get github.com/erock530/go-ollama-api
```

### Package Installation

#### RPM-based systems (RHEL, CentOS, Fedora)
```bash
sudo rpm -i go-ollama-api-<version>.rpm
sudo systemctl enable --now go-ollama-api
```

#### Debian-based systems (Ubuntu, Debian)
```bash
sudo dpkg -i go-ollama-api_<version>_amd64.deb
sudo systemctl enable --now go-ollama-api
```

## Versioning

This project uses semantic versioning. Release versions are in the format `vMAJOR.MINOR.PATCH`:
- MAJOR version for incompatible API changes
- MINOR version for new functionality in a backwards compatible manner
- PATCH version for backwards compatible bug fixes

Each release includes:
- Binary packages (RPM and DEB)
- Systemd service configuration
- Version information embedded in the binary

## Building and Running

### Using Make

The project includes a Makefile with common commands:

```bash
# Build the application
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Clean build files
make clean

# Build and run
make run
```

### Building from Source

```bash
git clone https://github.com/erock530/go-ollama-api.git
cd go-ollama-api
go build ./cmd/server
```

### Using Docker

Build and run using Docker:

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Or manually:
docker build -t go-ollama-api .
docker run -p 8080:8080 go-ollama-api
```

Note: When running in Docker, the container expects Ollama to be running on the host machine at `http://host.docker.internal:11434`. You can override this by setting the `OLLAMA_URL` environment variable.

## Usage

Start the server:

```bash
# Using binary
./server -port 8080 -ollama-url http://127.0.0.1:11434

# Using make
make run
```

### Command Line Arguments

- `-port`: Port to run the server on (default: 8080)
- `-ollama-url`: URL of the Ollama server (default: http://127.0.0.1:11434)

## CLI Commands

The interactive CLI starts automatically with the server. Available commands:

| Command | Description | Example |
|---------|-------------|---------|
| `generatekey` | Generate a single API key | `generatekey` |
| `generatekeys <count>` | Generate multiple API keys | `generatekeys 5` |
| `listkeys` | List all API keys | `listkeys` |
| `removekey <key>` | Remove an API key | `removekey abc123` |
| `addwebhook <url>` | Add a webhook URL | `addwebhook http://example.com/webhook` |
| `deletewebhook <id>` | Delete a webhook | `deletewebhook 1` |
| `listwebhooks` | List all webhooks | `listwebhooks` |
| `help` | Show available commands | `help` |
| `exit` | Exit the program | `exit` |

## API Endpoints

### Health Check

```bash
# Check API health
curl "http://localhost:8081/health?apikey=your-api-key"

# Example successful response:
{
    "status": "API is healthy",
    "timestamp": "2024-02-20T10:00:00Z"
}

# Example error response (invalid API key):
{
    "error": "Invalid API key"
}
```

### Generate Text

```bash
# Basic text generation
curl -X POST http://localhost:8081/generate \
  -H "Content-Type: application/json" \
  -d '{
    "apikey": "your-api-key",
    "model": "llama2",
    "prompt": "Hello, how are you?",
    "stream": false,
    "images": [],
    "raw": false
  }'

# Using llava model with images
curl -X POST http://localhost:8081/generate \
  -H "Content-Type: application/json" \
  -d '{
    "apikey": "your-api-key",
    "model": "llava:34b",
    "prompt": "What do you see in this image?",
    "stream": false,
    "images": ["base64-encoded-image-data"],
    "raw": false
  }'

# Streaming response
curl -X POST http://localhost:8081/generate \
  -H "Content-Type: application/json" \
  -d '{
    "apikey": "your-api-key",
    "model": "llama2",
    "prompt": "Write a long story about a space adventure",
    "stream": true,
    "images": [],
    "raw": false
  }'

# Example successful response:
{
    "response": "I'm doing well, thank you for asking! How can I help you today?"
}

# Example error responses:

# Invalid API key:
{
    "error": "Invalid API key"
}

# Rate limit exceeded:
{
    "error": "Rate limit exceeded. Try again later."
}

# Model not found:
{
    "error": "model 'nonexistent-model' not found"
}
```

Note: Replace `localhost:8081` with your server's address and port, and `your-api-key` with a valid API key generated using the CLI commands.

## Rate Limiting

- Each API key has a configurable rate limit (default: 10 requests per minute)
- Rate limits are tracked per key and reset every minute
- When rate limit is exceeded, the API returns a 429 (Too Many Requests) status code

## Webhooks

Webhooks are called for each API request with the following payload:

```json
{
    "apikey": "key-used",
    "prompt": "user-prompt",
    "model": "model-name",
    "stream": false,
    "images": [],
    "raw": false,
    "timestamp": "2024-02-20T10:00:00Z"
}
```

## Database Schema

The SQLite database (apiKeys.db) contains the following tables:

### apiKeys
```sql
CREATE TABLE apiKeys (
    key TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tokens INTEGER DEFAULT 10,
    rate_limit INTEGER DEFAULT 10,
    active INTEGER DEFAULT 1,
    description TEXT
)
```

### apiUsage
```sql
CREATE TABLE apiUsage (
    key TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

### webhooks
```sql
CREATE TABLE webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL
)
```

## Error Handling

Common HTTP status codes:

- 200: Success
- 400: Bad Request (missing API key, invalid request body)
- 403: Forbidden (invalid API key, deactivated key)
- 429: Too Many Requests (rate limit exceeded)
- 500: Internal Server Error

## Testing

The project includes comprehensive test coverage for the API endpoints and middleware:

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run tests with race detection and coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Continuous Integration

The project uses GitHub Actions for continuous integration:
- Runs tests on every push and pull request to main branch
- Includes race condition detection
- Generates and displays test coverage reports
- Tests run on Ubuntu with Go 1.21 and SQLite dependencies

You can view the latest test results and coverage reports in the GitHub Actions tab under the test workflow.

### Test Coverage

The test suite includes:

- Health Check Endpoint Tests
  - Valid API key validation
  - Missing API key handling
  - Invalid API key responses

- Rate Limiting Tests
  - Requests within rate limit
  - Rate limit exceeded scenarios
  - Inactive API key handling

- Mock Implementations
  - Database mocking via DBInterface
  - Ollama API response mocking
  - Rate limit state management

Each test ensures proper error handling, response codes, and payload validation.

## Service Management

The packages install and configure a systemd service:

```bash
# Start the service
sudo systemctl start go-ollama-api

# Stop the service
sudo systemctl stop go-ollama-api

# Check service status
sudo systemctl status go-ollama-api

# View logs
sudo journalctl -u go-ollama-api
```

Default configuration can be modified in `/etc/go-ollama-api/config.yaml`

## License

This project is licensed under the MIT License - see the LICENSE file for details.
