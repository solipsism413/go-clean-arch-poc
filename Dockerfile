# Base stage
FROM golang:1.26.2-alpine AS base

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Install development and code generation tools
RUN go install github.com/air-verse/air@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Development stage
FROM base AS dev

CMD ["air", "-c", ".air.toml"]

# Build stage
FROM base AS builder

# Run code generation
RUN sqlc generate && \
    swag init -g cmd/server/main.go -o docs

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# Production stage
FROM alpine:3.19 AS production

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./server"]
