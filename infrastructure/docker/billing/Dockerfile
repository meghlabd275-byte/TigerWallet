# Multi-stage build for Billing Service
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY billing_subscription/go/go.mod billing_subscription/go/go.sum ./
RUN go mod download

# Copy source code
COPY billing_subscription/go/ ./

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -o /billing-service ./main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install certificates and CA certificates
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /billing-service /app/billing-service
COPY --from=builder /app/config /app/config

# Create non-root user
RUN adduser -D -u 1000 appuser

USER appuser

# Expose port
EXPOSE 9001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9001/health || exit 1

# Run the service
CMD ["./billing-service"]
