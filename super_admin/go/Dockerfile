# Super Admin API Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: this module has no cgo dependencies, and the alpine builder
# image ships no C toolchain, so CGO_ENABLED=1 failed with
# `cgo: C compiler "gcc" not found`. This matches every other backend image
# in this repository.
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /super-admin-api .

FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /super-admin-api .

EXPOSE 8082

CMD ["./super-admin-api"]
