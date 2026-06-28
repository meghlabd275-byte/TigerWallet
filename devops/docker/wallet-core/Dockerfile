# TIGERWALLET WALLET CORE - Dockerfile
FROM rust:1.75-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache \
    musl-dev \
    openssl-dev \
    pkgconfig \
    git \
    protobuf-dev

# Copy Cargo files
COPY wallet_core/Cargo.toml wallet_core/Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/lib.rs
RUN cargo build --release --target x86_64-unknown-linux-musl 2>&1 | grep -v "warning" || true

# Copy source and build
COPY wallet_core/src ./src
RUN cargo build --release --target x86_64-unknown-linux-musl

# Runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata libssl

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /app/target/x86_64-unknown-linux-musl/release/wallet_core .
RUN chown -R appuser:appuser /app

USER appuser

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["./wallet_core", "--health"]

ENV RUST_LOG=info

EXPOSE 9090

CMD ["./wallet_core"]