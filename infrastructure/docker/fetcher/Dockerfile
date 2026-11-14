# Multi-stage build for the Fetcher Gateway (Rust, port 9003 by default).
# Build context is the repository root:
#   docker build -f infrastructure/docker/fetcher/Dockerfile .
FROM rust:1.85-alpine AS builder

RUN apk add --no-cache musl-dev openssl-dev openssl-libs-static pkgconfig

WORKDIR /app

COPY fetcher_gateway/rust/Cargo.toml fetcher_gateway/rust/Cargo.lock ./
COPY fetcher_gateway/rust/src ./src

RUN cargo build --release --locked

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/target/release/fetcher-gateway /app/fetcher-gateway

USER appuser

EXPOSE 9003

CMD ["./fetcher-gateway"]
