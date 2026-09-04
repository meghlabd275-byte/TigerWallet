# TigerWallet bot_core — the Rust execution plane for trading bots.
# Ultra-low-latency order execution (CEX HMAC-signed REST + DEX on-chain swaps),
# strategy runners (market_maker/arbitrage/sniper/grid/dca/momentum/
# mean_reversion/scalping), real PostgreSQL execution records.
# MSRV is 1.85: Cargo.lock pins time 0.3.36 / simple_asn1 0.6.2 / icu 1.5.x.
FROM rust:1.85-bookworm AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release --locked

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 10001 -m botcore
WORKDIR /app
COPY --from=builder /app/target/release/bot_core /app/bot_core
USER botcore
EXPOSE 8472
# BOT_CORE_PG: PostgreSQL connection string (fail-closed /health + /stats without it)
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD ["/app/bot_core", "--healthcheck"]
ENTRYPOINT ["/app/bot_core"]
