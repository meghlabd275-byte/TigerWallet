# Multi-stage build for the Notification Service (port 9004)
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY notifications/go/go.mod notifications/go/go.sum ./
RUN go mod download

# Copy source code
COPY notifications/go/ ./

# Build the canonical server binary from cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service ./cmd

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /notification-service /notification-service

EXPOSE 9004

ENTRYPOINT ["/notification-service"]
