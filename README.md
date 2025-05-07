# 🔮 Cosmos Upgrades Watcher 

A robust service that monitors and tracks upgrades for Cosmos-based blockchains, providing real-time notifications and comprehensive chain information through a RESTful API.

## ✨ Features

- **🔍 Chain Monitoring**
  - Tracks multiple Cosmos-based chains simultaneously
  - Supports both mainnet and testnet environments
  - Real-time upgrade tracking and notifications
  - Automatic chain registry updates

- **📢 Notifications**
  - Slack integration for upgrade notifications
  - Configurable notification thresholds
  - Custom notification formatting

- **📊 Data Sources**
  - GitHub Chain Registry integration
  - Polkachu API integration for upgrade information
  - Configurable data refresh intervals
  - Fallback mechanisms for data sources

- **💾 Caching**
  - In-memory caching for chain information
  - Configurable cache invalidation
  - Thread-safe cache operations
  - Optimized data refresh strategies

## 📋 Prerequisites

- Go 1.24 or later
- GitHub API token (recommended for higher rate limits)
- Slack webhook URL (optional, for notifications)

## 🚀 Installation

1. Clone the repository:
```bash
git clone https://github.com/p2p/devops-cosmos-watcher.git
cd devops-cosmos-watcher
```

2. Install dependencies:
```bash
go mod download
```

3. Copy and configure the sample configuration:
```bash
cp config.json.example config.json
```

4. Build the application:
```bash
go build -o cosmos-watcher ./cmd/server
```

5. Run:
```bash
go run cmd/server/main.go
```

## Makefile
The project includes a Makefile with several useful commands:

| Command | Description |
|---------|-------------|
| `make build` | Build the application |
| `make run` | Run the application |
| `make clean` | Clean build artifacts |
| `make deps` | Install dependencies |
| `make test` | Run all tests |
| `make test-coverage` | Run tests with coverage |
| `make test-integration` | Run integration tests |
| `make lint` | Run linters |
| `make setup` | Setup initial configuration |
| `make test-slack` | Test Slack notifications |
| `make help` | Show help message with all commands |

Run `make help` to see all available commands and their descriptions.

## ⚙️ Configuration

The application is configured via `config.json`:

```json
{
    "server": {
        "port": "8080",
        "read_timeout": "5s",
        "write_timeout": "10s"
    },
    "github": {
        "api_url": "https://api.github.com",
        "token": "your-github-token",
        "timeout": "30s"
    },
    "registry": {
        "url": "https://raw.githubusercontent.com/cosmos/chain-registry/master",
        "refresh_interval": "1h"
    },
    "poller": {
        "interval": "5m",
        "timeout": "30s"
    },
    "slack": {
        "webhook_url": "your-slack-webhook-url",
        "notification_threshold": "24h"
    }
}
```

## 🔌 API Reference

All endpoints are prefixed with `/api/v1`.

### 🏥 Health Check

#### GET /health
Returns the service health status.

**Response:**
```json
{
    "status": "healthy",
    "uptime": "24h10m",
    "last_update": "2024-03-20T15:04:05Z",
    "version": "1.0.0"
}
```

### ⛓️ Chain Information

#### GET /chains/{chainName}
Returns detailed information about a specific chain.

**Parameters:**
- `chainName`: The name of the chain (e.g., "cosmoshub", "osmosis")

**Response:**
```json
{
    "name": "cosmoshub",
    "network": "mainnet",
    "chain_id": "cosmoshub-4",
    "current_version": "v10.0.0",
    "binary": "gaiad",
    "last_upgrade": {
        "name": "v10",
        "height": 17343000,
        "time": "2024-02-15T14:00:00Z"
    }
}
```

### 🔄 Upgrades

#### GET /upgrades
Returns all upcoming and recent upgrades across all networks.

**Query Parameters:**
- `network`: Filter by network type (mainnet|testnet)
- `status`: Filter by status (pending|completed|failed)
- `days`: Number of days to look back for completed upgrades (default: 7)

**Response:**
```json
{
    "upgrades": [
        {
            "chain_name": "osmosis",
            "network": "mainnet",
            "name": "v20",
            "height": 11250000,
            "time": "2024-04-01T15:00:00Z",
            "status": "pending",
            "proposal_link": "https://www.mintscan.io/osmosis/proposals/1234"
        }
    ]
}
```

### 👷 Jobs Management

#### GET /jobs
Lists all configured monitoring jobs.

**Response:**
```json
{
    "jobs": [
        {
            "name": "chain-monitor",
            "type": "periodic",
            "interval": "5m",
            "last_run": "2024-03-20T15:04:05Z",
            "status": "running"
        }
    ]
}
```

#### POST /jobs
Adds a new monitoring job.

**Request Body:**
```json
{
    "name": "custom-monitor",
    "type": "periodic",
    "interval": "10m",
    "chains": ["osmosis", "juno"]
}
```

**Response:**
```json
{
    "id": "job-123",
    "status": "created",
    "message": "Job successfully added"
}
```

#### DELETE /jobs/{name}
Removes a monitoring job.

**Parameters:**
- `name`: The name of the job to remove

**Response:**
```json
{
    "status": "success",
    "message": "Job successfully removed"
}
```

### ⚙️ Scheduler Control

#### POST /scheduler/start
Starts the job scheduler.

**Response:**
```json
{
    "status": "running",
    "started_at": "2024-03-20T15:04:05Z"
}
```

#### POST /scheduler/stop
Stops the job scheduler.

**Response:**
```json
{
    "status": "stopped",
    "stopped_at": "2024-03-20T15:04:05Z"
}
```

### 📝 Response Formats

All responses follow a standard format:

- Success responses return HTTP 200/201 with the requested data
- Error responses return appropriate HTTP status codes (4xx/5xx) with error details:
```json
{
    "error": {
        "code": "ERROR_CODE",
        "message": "Human readable error message",
        "details": {}
    }
}
```

## 🧪 Testing

### 🔬 Unit Tests
Run the full test suite:
```bash
go test ./...
```

Run tests for a specific package:
```bash
go test ./internal/chain/...
```

Run tests with coverage:
```bash
go test -cover ./...
```

### 🔌 Integration Tests
Run integration tests (requires configuration):
```bash
go test -tags=integration ./...
```

### 🛠️ Manual Testing
Test Slack notifications:
```bash
go test -v ./internal/notifications -run TestSlackNotificationManual
```

### 🔍 Linting and Static Analysis
Run all linters:
```bash
golangci-lint run
```

Run specific linters:
```bash
go vet ./...
staticcheck ./...
```

## 👨‍💻 Development

### 🔗 Adding a New Chain
1. Add chain configuration to `config/chains.yaml`
2. Implement chain-specific upgrade detection if needed
3. Add relevant test cases

## 🔧 Troubleshooting

### ❗ Common Issues
1. **Rate Limiting**: Ensure GitHub token is configured
2. **Missing Upgrades**: Check chain registry data freshness
3. **Notification Delays**: Verify poller interval configuration

### 📝 Logging
- Logs are written to stdout/stderr
- Use `-v` flag for verbose logging
- Set LOG_LEVEL environment variable for custom log levels

## 📄 License

MIT License - See [](./LICENSE) file for details.
