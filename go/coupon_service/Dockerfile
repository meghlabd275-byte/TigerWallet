# Build context is ./go (the parent directory) so the shared auth module is
# available: the replace directive points at ../sharedauth, which resolves to
# /sharedauth inside the image.
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY coupon_service/go.mod coupon_service/go.sum ./
COPY sharedauth /sharedauth
RUN go mod download
COPY coupon_service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /coupon-service .
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /coupon-service /app/coupon-service
USER appuser
EXPOSE 8467
ENTRYPOINT ["/app/coupon-service"]
