# TigerSmartChain node image.
#
# The node links go-ethereum v1.13.15, which is the last release that still
# supports proof-of-authority block sealing. That release uses a linkname into
# runtime.stopTheWorld, which Go 1.23 and later reject unless the linker is told
# otherwise, so every build must pass -checklinkname=0. Dropping the flag does
# not fail the build - it fails the final link, after the whole module graph has
# been compiled, so it is set here rather than left to the caller.

FROM golang:1.23-bookworm AS build

WORKDIR /src

# Dependencies first so the module cache survives edits to the chain sources.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO is not required: the node uses the pure-Go secp256k1 implementation.
RUN CGO_ENABLED=0 go build -trimpath \
        -ldflags='-checklinkname=0 -s -w' \
        -o /out/tscnode ./cmd/tscnode

FROM debian:bookworm-slim

# ca-certificates is needed if an operator points the node at an HTTPS RPC or
# the chain later dials a TLS peer; tzdata keeps log timestamps correct. curl is
# used by the container healthcheck to issue a real JSON-RPC call.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata curl \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/tscnode /usr/local/bin/tscnode

# The node writes its chain database and keystore here. Declared as a volume so
# an operator's chain data survives a container replacement.
VOLUME ["/data"]

# JSON-RPC. Kept on the container network by default; the compose file decides
# whether to publish it.
EXPOSE 8545 8546 30303

# Run as a non-root user. The chain data directory is chowned so a mounted
# volume is writable without granting the container root.
RUN useradd --system --uid 10001 --create-home --home-dir /home/tsc tsc \
 && mkdir -p /data && chown -R tsc:tsc /data /home/tsc
USER tsc

WORKDIR /home/tsc

COPY --chown=tsc:tsc docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
