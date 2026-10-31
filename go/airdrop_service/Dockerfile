# Build context is ./go (the parent directory) so the shared auth module is
# available: the replace directive points at ../sharedauth, which resolves to
# /sharedauth inside the image.
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY airdrop_service/go.mod airdrop_service/go.sum ./
COPY sharedauth /sharedauth
RUN go mod download
COPY airdrop_service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /airdrop-service .
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /airdrop-service /app/airdrop-service
USER appuser
EXPOSE 8465
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8465/api/v1/airdrop/campaigns || exit 1
ENTRYPOINT ["/app/airdrop-service"]
