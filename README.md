# URL Shortener

A URL shortening service built with Go, using Echo for the web server, PostgreSQL for storage, and Valkey for caching.

## Features

- Shorten long URLs to easy-to-share short URLs
- Support for custom short URLs
- URL expiration
- Click tracking
- Caching for improved performance

## Requirements

- Go 1.21 or higher
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
  -e POSTGRES_URL=postgres://postgres:postgres@postgres:5432/urlshortener?sslmode=disable \
  -e VALKEY_ADDR=valkey:6379 \
  -e BASE_URL=http://localhost:8080 \
  urlshortener
```

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

Some tests are marked as integration tests and are skipped by default because they require a running database or cache. To run these tests, you need to:

1. Set up a test database and cache
2. Remove the `t.Skip()` line from the test
3. Run the test with the appropriate connection parameters

For example, to run the PostgreSQL repository integration test:

```
# Set up a test database
createdb urlshortener_test

# Run the test
go test ./store -run TestPostgresRepository_Integration
```

### Test Coverage

To generate a test coverage report:

```
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## License

MIT
