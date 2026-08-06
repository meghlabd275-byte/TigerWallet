FROM rust:1.75-alpine AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN cargo fetch
COPY . .
RUN cargo build --release

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/target/release/fetcher-gateway .
EXPOSE 8093
CMD ["./fetcher-gateway"]
