FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o prediction-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/prediction-service .
RUN adduser -D appuser
USER appuser
EXPOSE 8455
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8455/health || exit 1
CMD ["./prediction-service"]
