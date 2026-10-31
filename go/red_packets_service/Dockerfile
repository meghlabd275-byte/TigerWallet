# Build context is ./go (the parent directory) so the shared auth module is
# available: the replace directive points at ../sharedauth, which resolves to
# /sharedauth inside the image.
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY red_packets_service/go.mod red_packets_service/go.sum ./
RUN go mod download
COPY red_packets_service/ .
COPY sharedauth /sharedauth
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /red-packets-service .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /red-packets-service /app/red-packets-service
USER appuser
EXPOSE 8468
ENTRYPOINT ["/app/red-packets-service"]
