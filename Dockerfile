# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o urlshortener

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/urlshortener .

# Expose port
EXPOSE 8080

# Set environment variables
ENV SERVER_PORT="8080" \
    BASE_URL="http://localhost:8080" \
    API_KEY="your-api-key-here" \
    POSTGRES_URL="postgres://postgres:postgres@postgres:5432/urlshortener?sslmode=disable" \
    VALKEY_ADDR="valkey:6379" \
    VALKEY_PASSWORD="" \
    VALKEY_DB="0" \
    VALKEY_TTL="24h"

# Run the application
CMD ["./urlshortener"]
