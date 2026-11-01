FROM golang:1.23-alpine AS builder
WORKDIR /app
# This module depends only on the standard library, so there is no go.sum
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bridge-service .
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /bridge-service /app/bridge-service
USER appuser
EXPOSE 8470
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8470/health || exit 1
ENTRYPOINT ["/app/bridge-service"]
