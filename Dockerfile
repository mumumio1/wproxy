# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.version=docker -X main.buildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -o proxy ./cmd/proxy

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/proxy .

# Copy config example
COPY config.example.yaml /app/config.yaml

# Create non-root user
RUN adduser -D -u 1000 proxy && \
    chown -R proxy:proxy /app

USER proxy

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run proxy
ENTRYPOINT ["/app/proxy"]
CMD ["-config", "/app/config.yaml"]

