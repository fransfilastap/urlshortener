# Frontend build stage
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Go build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o urlshortener ./cmd/urlshortener

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/urlshortener .
EXPOSE 8080
ENV SERVER_PORT="8080" \
    BASE_URL="http://localhost:8080" \
    API_KEY="your-api-key-here" \
    POSTGRES_URL="postgres://postgres:postgres@postgres:5432/urlshortener?sslmode=disable" \
    VALKEY_ADDR="valkey:6379" \
    VALKEY_PASSWORD="" \
    VALKEY_DB="0" \
    VALKEY_TTL="24h" \
    SESSION_SECRET="change-me-in-production" \
    SESSION_MAX_AGE="86400"
CMD ["./urlshortener"]