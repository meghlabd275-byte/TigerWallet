# TigerWallet in-house KYC service — the HTTP and PostgreSQL layer over the
# tiger-kyc-engine verification engine. Runs self-hosted with no external
# vendor: identity verification, document checks and sanctions screening all
# happen inside this deployment.
#
# The engine crate is dependency-free, so the auditable decision logic carries
# no third-party code; only this service pulls in axum/sqlx.
FROM rust:1.85-bookworm AS builder
WORKDIR /app

# The service depends on the engine by relative path, so both crates are
# copied. They are built together because the engine is a path dependency.
COPY kyc_engine /app/kyc_engine
COPY kyc_service /app/kyc_service
WORKDIR /app/kyc_service
RUN cargo build --release --locked

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 10002 -m kycsvc

WORKDIR /app
COPY --from=builder /app/kyc_service/target/release/tiger-kyc-service /app/tiger-kyc-service

# The service holds personal data, so it runs unprivileged and writes nothing
# to its own image.
USER kycsvc
EXPOSE 8470

# The service refuses to start without KYC_DATABASE_URL and both tokens, so a
# container that is up has a working database connection. /readyz is the probe
# that additionally confirms the schema is reachable.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8470/health || exit 1

ENTRYPOINT ["/app/tiger-kyc-service"]
