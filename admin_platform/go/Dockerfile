# Admin Platform API Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=1 go build -o /admin-api .

# Final stage
FROM alpine:3.18

WORKDIR /app

# Install certificates and runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /admin-api .

# Expose port
EXPOSE 8081

# Run
CMD ["./admin-api"]
