# URL Shortener

A URL shortening service built with Go, using Echo for the web server, PostgreSQL for storage, and Valkey for caching.

## Features

- Shorten long URLs to easy-to-share short URLs
- Support for custom short URLs
- URL expiration
- Click tracking
- Caching for improved performance

## Requirements

- Go 1.23 or higher
- PostgreSQL
- Valkey (or Redis)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/fransfilastap/urlshortener.git
   cd urlshortener
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set up environment variables (or use defaults):
   ```
   export SERVER_PORT=8080
   export BASE_URL=http://localhost:8080
   export POSTGRES_URL=postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable
   export VALKEY_ADDR=localhost:6379
   export VALKEY_PASSWORD=
   export VALKEY_DB=0
   export VALKEY_TTL=24h
   ```

4. Run the application:
   ```
   go run main.go
   ```

## Using the Makefile

This project includes a Makefile to simplify common development tasks. Here are some of the available commands:

### Building the Application

```
# Build for the current platform
make build

# Build for multiple platforms (Linux, macOS, Windows)
make build-all

# Build for a specific platform
make build-linux
make build-darwin
make build-windows
```

### Running the Application

```
# Build and run
make run
```

### Testing

```
# Run all tests
make test

# Run tests with race detection
make test-race

# Generate test coverage report
make coverage
```

### Code Quality

```
# Run linter
make lint

# Format code
make fmt
```

### Docker Operations

```
# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Start all services with Docker Compose
make docker-up

# Stop all services
make docker-down
```

### Other Commands

```
# Clean build artifacts
make clean

# Tidy dependencies
make tidy

# Show all available commands
make help
```

## API Endpoints

### Shorten a URL

```
POST /shorten
```

Request body:
```json
{
  "url": "https://example.com/very/long/url/that/needs/shortening",
  "custom_code": "custom",
  "expiry": 86400
}
```

- `url`: The original URL to shorten (required)
- `custom_code`: Custom short code (optional)
- `expiry`: Expiration time in seconds (optional)

Response:
```json
{
  "original_url": "https://example.com/very/long/url/that/needs/shortening",
  "short_url": "http://localhost:8080/custom",
  "expires_at": "2023-04-01T12:00:00Z",
  "clicks": 0
}
```

### Redirect to Original URL

```
GET /:code
```

This endpoint redirects to the original URL associated with the short code.

### Get URL Information

```
GET /api/urls/:code
```

Response:
```json
{
  "original_url": "https://example.com/very/long/url/that/needs/shortening",
  "short_url": "http://localhost:8080/custom",
  "expires_at": "2023-04-01T12:00:00Z",
  "clicks": 5
}
```

## Docker

You can run the application using Docker:

```
docker build -t urlshortener .
docker run -p 8080:8080 \
  -e POSTGRES_URL=postgres://postgres:postgres@host.docker.internal:5432/urlshortener?sslmode=disable \
  -e VALKEY_ADDR=valkey:6379 \
  -e BASE_URL=http://localhost:8080 \
  urlshortener
```

Note: This example uses `host.docker.internal` to connect to PostgreSQL running on your host machine. Make sure PostgreSQL is running on your host and is configured to accept connections.

### Using Docker Compose

The project includes a `docker-compose.yml` file that sets up the application stack with flexible database options:

1. Copy the example environment file:
   ```
   cp .env.example .env
   ```

2. Customize the `.env` file with your own values:
   - Set `USE_DOCKER_POSTGRES=true` to use PostgreSQL in Docker, or `false` to use a host PostgreSQL
   - Set `ENABLE_BACKUP=true` to enable automated backups (only works with Docker PostgreSQL)
   - Set `ENABLE_DEV_MODE=true` to enable development mode with hot reloading
   - Update `POSTGRES_URL` based on your database choice:
     - For host PostgreSQL: `postgres://user:password@host.docker.internal:5432/dbname`
     - For Docker PostgreSQL: `postgres://user:password@postgres:5432/dbname`

3. Run the application stack with the appropriate profile:

   For using host PostgreSQL (default):
   ```
   docker-compose up -d
   ```

   For using Docker PostgreSQL:
   ```
   docker-compose --profile postgres up -d
   ```

   For using Docker PostgreSQL with backups:
   ```
   docker-compose --profile postgres --profile postgres-backup up -d
   ```

   For development mode with hot reloading:
   ```
   docker-compose --profile dev up -d
   ```

   For development mode with Docker PostgreSQL:
   ```
   docker-compose --profile dev --profile postgres up -d
   ```

The `docker-compose.yml` file is configured to load environment variables from the `.env` file. If a variable is not defined in the `.env` file, it will use the default value specified in the `docker-compose.yml` file.

### Database Options

#### Using Host PostgreSQL

By default, the application is configured to connect to a PostgreSQL database running on your host machine. This is useful for development or when you already have a PostgreSQL server running.

To use host PostgreSQL:
1. Ensure PostgreSQL is installed and running on your host
2. Set `USE_DOCKER_POSTGRES=false` in your `.env` file
3. Configure `POSTGRES_URL` to point to your host PostgreSQL instance
4. Run `docker-compose up -d`

#### Using Docker PostgreSQL

Alternatively, you can use a PostgreSQL container managed by Docker Compose:

1. Set `USE_DOCKER_POSTGRES=true` in your `.env` file
2. Run `docker-compose --profile postgres up -d`

### Data Persistence and Backup

The Docker Compose configuration includes data persistence and optional automated backup for PostgreSQL:

#### Data Persistence

PostgreSQL data is stored in a named volume (`postgres_data`) that persists across container restarts and updates. This ensures your data is not lost when containers are recreated.

```yaml
volumes:
  postgres_data:
    driver: local
```

#### Automated Backups

The configuration includes an optional automated backup mechanism for PostgreSQL:

1. Set `ENABLE_BACKUP=true` in your `.env` file
2. Run with the postgres-backup profile: `docker-compose --profile postgres --profile postgres-backup up -d`

The backup system:
1. Uses a dedicated `postgres_backup` service that runs alongside the main PostgreSQL service
2. Schedules backups using cron (default: daily at midnight)
3. Stores backup files in a separate volume (`postgres_backup`)
4. Applies a retention policy to automatically remove backups older than the specified retention period (default: 7 days)

You can customize the backup behavior using these environment variables:

- `BACKUP_SCHEDULE`: Cron expression for backup schedule (default: `0 0 * * *` - daily at midnight)
- `BACKUP_RETENTION_DAYS`: Number of days to keep backups (default: 7)

Example backup schedule configurations:
- Daily at midnight: `0 0 * * *`
- Every 6 hours: `0 */6 * * *`
- Weekly on Sunday at 1am: `0 1 * * 0`

### Development Mode with Hot Reloading

The project includes a development mode with hot reloading, which automatically rebuilds and restarts the application when source code changes are detected. This is useful for rapid development and testing.

To use development mode:

1. Set `ENABLE_DEV_MODE=true` in your `.env` file
2. Run with the dev profile: `docker-compose --profile dev up -d`

The development mode:
1. Uses a dedicated `app-dev` service that runs with hot reloading enabled
2. Mounts the project directory as a volume, so changes to source code are immediately available inside the container
3. Uses the Air tool to watch for file changes and automatically rebuild/restart the application
4. Provides real-time feedback on build success or errors

You can customize the hot reloading behavior by modifying the `.air.toml` configuration file.

Benefits of development mode:
- Faster development cycles - no need to manually rebuild and restart containers
- Immediate feedback on code changes
- Preserves your database data between restarts
- Works with both host PostgreSQL and Docker PostgreSQL

## Testing

The project includes various tests for different components:

### Running Unit Tests

To run all unit tests:

```
go test ./...
```

To run tests for a specific package:

```
go test ./handlers
go test ./store
```

To run a specific test:

```
go test ./handlers -run TestAPIKeyMiddleware
```

### Integration Tests

The project uses [testcontainers-go](https://github.com/testcontainers/testcontainers-go) for integration testing. This allows the tests to spin up Docker containers for PostgreSQL and Redis/Valkey on demand, run the tests against them, and then clean up automatically.

To run the integration tests:

```
# Run all integration tests
go test ./store -run TestPostgresRepository_Integration
go test ./store -run TestCacheRepository_Integration
```

Requirements for running integration tests:
- Docker must be installed and running
- The Docker API must be accessible to the user running the tests

The testcontainers approach eliminates the need to:
- Set up separate test databases or caches
- Modify test files to enable/disable tests
- Manage test infrastructure manually

This makes the integration tests more reliable, isolated, and easier to run in CI/CD environments.

### Test Coverage

To generate a test coverage report:

```
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## License

MIT
