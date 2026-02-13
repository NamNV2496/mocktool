# Multi-stage build for mocktool

# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary
# -ldflags="-s -w" reduces binary size by stripping debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=1.0.0" \
    -o mocktool \
    .

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 mocktool && \
    adduser -D -u 1000 -G mocktool mocktool

# Set working directory
WORKDIR /home/mocktool

# Copy binary from builder
COPY --from=builder /app/mocktool .

# Copy web assets
COPY --from=builder /app/web ./web

# Change ownership
RUN chown -R mocktool:mocktool /home/mocktool

# Switch to non-root user
USER mocktool

# Expose ports
# 8081 - Management API and Web UI
# 8082 - Mock forwarding server
EXPOSE 8081 8082

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Set entrypoint
ENTRYPOINT ["./mocktool"]

# Default command
CMD ["service"]
