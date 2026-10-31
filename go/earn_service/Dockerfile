# Build context is ./go (the parent directory) so the shared auth module is
# available: the replace directive points at ../sharedauth, which resolves to
# /sharedauth inside the image.
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY earn_service/go.mod earn_service/go.sum ./
COPY sharedauth /sharedauth
RUN go mod download
COPY earn_service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /earn-service .
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /earn-service /app/earn-service
USER appuser
EXPOSE 8466
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8466/api/v1/earn/products || exit 1
ENTRYPOINT ["/app/earn-service"]
