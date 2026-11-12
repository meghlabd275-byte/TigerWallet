#!/bin/sh
# TigerSmartChain container entrypoint.
#
# The node needs a network definition before it can run, and the genesis it
# builds must be identical on every node that joins, or the peers reject each
# other. This script makes that reproducible from environment variables alone:
# on first boot it writes the config, and on every boot it re-derives the same
# genesis hash from the same seed.
#
# Set:
#   TSC_GENESIS_SEED   seed the network definition is derived from (required on
#                      first boot). Must be at least 16 characters or 32 hex
#                      digits. Changing it on a running network forks it.
#   TSC_VALIDATOR_KEY  secp256k1 private key that signs blocks. When unset the
#                      key is derived from the seed at the first validator
#                      index, which is what a single-node devnet needs.
#   TSC_CHAIN_ID       EIP-155 chain id (default 7710).
#   TSC_VALIDATORS     number of authorities in the genesis set (default 1).
#   TSC_SUPPLY         total TGR supply, in whole TGR (default 1000000000).
#   TSC_MINE           set to "false" to run a non-mining follower.

set -eu

DATADIR="${TSC_DATADIR:-/data/chain}"
CONFIG="${TSC_CONFIG:-/data/devnet.json}"
CHAIN_ID="${TSC_CHAIN_ID:-7710}"
VALIDATORS="${TSC_VALIDATORS:-1}"
SUPPLY="${TSC_SUPPLY:-1000000000}"
HTTP_PORT="${TSC_HTTP_PORT:-8545}"
WS_PORT="${TSC_WS_PORT:-8546}"
P2P_PORT="${TSC_P2P_PORT:-30303}"

log() { echo "tsc-entrypoint: $*" >&2; }

# A network definition is written once. Re-running bootstrap against an
# existing file would be harmless but pointless, and re-deriving it keeps the
# genesis hash stable across restarts, which is what the peers compare.
if [ ! -f "$CONFIG" ]; then
    if [ -z "${TSC_GENESIS_SEED:-}" ]; then
        log "TSC_GENESIS_SEED must be set on first boot: it determines the genesis block"
        log "that every node on this network must agree on."
        exit 1
    fi
    log "writing network definition to $CONFIG (chain id $CHAIN_ID, $VALIDATORS validator(s))"
    tscnode bootstrap \
        -config "$CONFIG" \
        -seed "$TSC_GENESIS_SEED" \
        -validators "$VALIDATORS" \
        -chain-id "$CHAIN_ID" \
        -supply "$SUPPLY"
fi

MINE_FLAG=""
if [ "${TSC_MINE:-true}" != "false" ]; then
    MINE_FLAG="-mine"

    # The signing key must belong to the genesis authority set, or the node
    # refuses to start. Deriving it from the same seed keeps a single-node
    # deployment self-consistent without an operator having to copy a key out
    # of `tscnode account`.
    if [ -z "${TSC_VALIDATOR_KEY:-}" ]; then
        if [ -z "${TSC_GENESIS_SEED:-}" ]; then
            log "mining requires TSC_VALIDATOR_KEY, or TSC_GENESIS_SEED to derive it"
            exit 1
        fi
        log "deriving the block-signing key from the genesis seed (first validator index)"
        # Exported, not merely assigned: the node reads this from its own
        # environment, so a shell-local variable would be invisible to it.
        TSC_VALIDATOR_KEY="$(tscnode account -seed "$TSC_GENESIS_SEED" -index 1000 | awk '/private key/{print $3}')"
        export TSC_VALIDATOR_KEY
        if [ -z "$TSC_VALIDATOR_KEY" ]; then
            log "could not derive a validator key from the seed"
            exit 1
        fi
    fi
fi

log "starting node: chain id $CHAIN_ID, rpc :$HTTP_PORT, ws :$WS_PORT, p2p :$P2P_PORT"

# exec so the node becomes PID 1 and receives SIGTERM directly; a shell in the
# middle would swallow the signal and the container would be killed on timeout.
exec tscnode run \
    -config "$CONFIG" \
    -datadir "$DATADIR" \
    -http-port "$HTTP_PORT" \
    -ws-port "$WS_PORT" \
    -p2p-port "$P2P_PORT" \
    $MINE_FLAG \
    "$@"
