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
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/urlshortener .
USER appuser
EXPOSE 8080
CMD ["./urlshortener"]