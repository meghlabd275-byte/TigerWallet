# TIGERWALLET API GATEWAY - UPGRADED
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go modules
COPY backend_services/api_gateway/go.mod backend_services/api_gateway/go.sum ./
RUN go mod download

# Copy source
COPY backend_services/api_gateway/ ./

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /api-gateway

# Multi-stage runtime - minimal image
FROM alpine:3.20

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /api-gateway .
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENV PORT=8080 \
    LOG_LEVEL=info \
    DATABASE_POOL_SIZE=10 \
    CACHE_TTL=300

EXPOSE 8080
EXPOSE 6060

CMD ["./api-gateway"]