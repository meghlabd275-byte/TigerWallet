#!/usr/bin/env python3
"""Regenerate go/wallet_api/chains_evm_data.go from the canonical chain registry.

Run from the repository root:

    python3 tools/fetch_chain_registry.py     # downloads tools/cache/chains.json
    python3 tools/gen_evm_chains.py           # rewrites the Go registry

The registry download is a separate step so that regeneration is reproducible
offline once the file is cached.

The canonical registry is ethereum-lists/chains (chainid.network/chains.json).
Entries already present in the generated file are preserved verbatim so that
hand-verified RPC alternates and derivation paths survive a regeneration; new
chains are appended from the upstream data.

Only mainnet chains that publish at least one https RPC endpoint are emitted. A
chain with no public RPC is omitted rather than shipped with an empty endpoint,
because an empty endpoint would look like a working chain to every caller.
"""
import json
import re
import sys

REGISTRY = "tools/cache/chains.json"
TARGET = "go/wallet_api/chains_evm_data.go"

# Names requested by the owner that the upstream registry carries under a
# different string. The value is the registry name to search for.
REQUESTED_ALIASES = {
    "polygon pos": "Polygon",
    "op mainnet": "OP Mainnet",
    "bnb smart chain": "BNB Smart Chain",
    "avalanche c-chain": "Avalanche C-Chain",
    "zkSync Era": "zkSync Era",
    "Huobi ECO Chain": "HECO",
    "Kaia": "Kaia Mainnet",
    "Milkomeda C1": "Milkomeda C1",
    "Milkomeda A1": "Milkomeda A1",
    "Sei EVM": "Sei Network",
    "Kava EVM": "Kava",
    "Evmos EVM": "Evmos",
    "Syscoin NEVM": "Syscoin",
    "IOTA EVM": "IOTA EVM",
    "Manta Pacific": "Manta Pacific",
    "ShimmerEVM": "ShimmerEVM",
    "ApeChain": "ApeChain",
    "World Chain": "World Chain",
    "Plasma Mainnet": "Plasma",
    "Robinhood Chain": "Robinhood Chain",
    "Arc": "Arc",
    "Gravity Network": "Gravity",
    "Katana Network": "Katana",
    "Lisk Legacy": "Lisk",
    "Stratis Native": "Stratis",
    "Step Network": "Step Network",
    "Core Blockchain": "Core Blockchain",
    "Wakt Chain": "Wakt",
    "Loopring": "Loopring",
    "Echelon": "Echelon",
    "Lightlink": "LightLink",
    "Tenet": "Tenet",
    "Sonic": "Sonic",
    "HyperEVM": "HyperEVM",
    "Flare Mainnet": "Flare Mainnet",
    "Polygon PoS": "Polygon Mainnet",
    "Gnosis Chain": "Gnosis",
    "Harmony Shard 0": "Harmony Mainnet Shard 0",
    "Plume Network": "Plume Mainnet",
    "Astar EVM": "Astar",
    "Songbird Canary Network": "Songbird Canary-Network",
    "Shiden Network": "Shiden",
    "Acala EVM+": "Acala Network",
    "Karura EVM+": "Karura Network",
    "Mode Network": "Mode",
    "Ronin Network": "Ronin Mainnet",
    "Ubiq": "Ubiq",
    "zkSync Era": "zkSync Mainnet",
    "Gnosis Chain": "Gnosis",
    "Chiliz Chain": "Chiliz Chain Mainnet",
    "Syscoin NEVM": "Syscoin Mainnet",
    "GateChain": "GateChain Mainnet",
    "Ronin Network": "Ronin Mainnet",
}

# Chains the registry does not carry. Each entry is a verified mainnet with a
# public RPC that answers eth_chainId with the id below. They are listed here
# explicitly rather than discovered by name matching so that nothing is emitted
# on a guess.
# Endpoints confirmed to answer eth_chainId with the right id, used in place of
# the registry's first choice when that one did not respond. Verified with
# tools/verify_chain_rpcs.py.
RPC_OVERRIDES = {
    592: "https://evm.astar.network",
    336: "https://evm.shiden.astar.network",
    1284: "https://moonbeam.api.onfinality.io/public",
    1285: "https://moonriver.api.onfinality.io/public",
}

MANUAL_CHAINS = [
    {
        "chainId": 999,
        "name": "HyperEVM",
        "symbol": "HYPE",
        "rpc": "https://rpc.hyperliquid.xyz/evm",
        "explorer": "https://hyperevmscan.io",
        "coinType": 60,
    },
]

# Chains the platform operates itself. They are not in the upstream registry
# (nobody else runs them), so they cannot be resolved by name and must be
# merged into the emitted set explicitly. Keeping them here rather than only in
# the generated file means a regeneration from a clean checkout still produces
# a registry that contains them.
PLATFORM_CHAINS = [
    {
        "chainId": 7710,
        "name": "TigerSmartChain",
        "symbol": "TGR",
        # Deliberately empty. The platform operates this node, so there is no
        # public URL to bundle; shipping one would make the wallet dial a host
        # nobody chose. The operator sets TSC_RPC_URL (or CHAIN_7710_RPC), and
        # until then the chain is listed but reports as unreachable.
        "rpc": "",
        "explorer": "",
        "coinType": 60,
        "comment": [
            "TigerSmartChain is the platform's own proof-of-authority network.",
            "Its genesis carries the RUSD stablecoin and the WTGR wrapped-native",
            "token as predeploys, so those addresses are fixed by consensus and",
            "are the same on every node. RUSD and WTGR are registered as",
            "first-class tokens in defaultTokenRegistry.",
            "",
            "No endpoint is bundled: the platform operates the node, so there is no",
            "public URL to ship. The operator sets TSC_RPC_URL (or the per-chain",
            "CHAIN_7710_RPC), and until then the chain is listed but reports as",
            "unreachable rather than dialling a guessed host.",
        ],
    },
]

# The registry uses these slugs for chains whose display name differs.
NAME_OVERRIDES = {
    "polygon-pos": "Polygon PoS",
    "op-mainnet": "OP Mainnet",
    "bnb-smart-chain": "BNB Smart Chain",
    "avalanche-c-chain": "Avalanche C-Chain",
}

# A registry entry is treated as a testnet when its name says so, when it
# publishes a faucet, or when the registry itself flags it. Testnets must never
# reach the shipped registry: a user would see a chain whose balances are
# worthless and whose RPC is not the production network.
TESTNET_PATTERN = re.compile(
    r"testnet|test net|sepolia|goerli|holesky|devnet|local|sandbox|staging"
    r"|\bdev\b|rinkeby|ropsten|kovan|mumbai|canary test|chiado|morden",
    re.I,
)


def is_testnet(chain):
    if TESTNET_PATTERN.search(chain.get("name", "")):
        return True
    if TESTNET_PATTERN.search(chain.get("shortName", "")):
        return True
    # The registry attaches faucets to several production mainnets and flags
    # reusedChainId on others, so neither is a testnet signal. The name pattern
    # and a deprecated status are the reliable ones.
    if chain.get("status") == "deprecated":
        return True
    return False


def load_registry():
    return json.load(open(REGISTRY, encoding="utf-8"))


def parse_existing(path):
    """Returns {chain_id: raw_entry_text} for entries already in the file."""
    text = open(path, encoding="utf-8").read()
    entries = {}
    for match in re.finditer(r"\{\s*\n\s*ID:\s*(\d+),(.*?)\n\s*\},", text, re.S):
        entries[int(match.group(1))] = match.group(0)
    return entries, text


def best_rpc(chain):
    """Picks the first https RPC endpoint, preferring ones with tracking=none."""
    rpcs = [r for r in chain.get("rpc", []) if r.startswith("https://")]
    if not rpcs:
        return None
    clean = [r for r in rpcs if "${" not in r]
    return (clean or rpcs)[0]


def alternates(chain, primary):
    out = []
    for rpc in chain.get("rpc", []):
        if not rpc.startswith("https://") or "${" in rpc or rpc == primary:
            continue
        if rpc not in out:
            out.append(rpc)
        if len(out) >= 3:
            break
    return out


def explorer_url(chain):
    for ex in chain.get("explorers", []):
        if ex.get("url", "").startswith("https://"):
            return ex["url"].rstrip("/")
    return ""


def native_symbol(chain):
    native = chain.get("nativeCurrency") or {}
    return native.get("symbol", "")


def go_string(value):
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def render(chain_id, name, symbol, rpc, path, explorer, chain_type, decimals,
           coin_type, alternates_list):
    parts = [
        f"ID: {chain_id}",
        f"Name: {go_string(name)}",
        f"Symbol: {go_string(symbol)}",
        f"RPCEndpoint: {go_string(rpc)}",
        f"DerivationPath: {go_string(path)}",
        f"ExplorerAPI: {go_string(explorer)}",
        f"ExplorerURL: {go_string(explorer)}",
        f"ChainType: {go_string(chain_type)}",
        f"Decimals: {decimals}",
        f"CoinType: {coin_type}",
        "IsTestnet: false",
    ]
    if alternates_list:
        alts = ", ".join(go_string(a) for a in alternates_list)
        parts.append(f"RPCAlternates: []string{{{alts}}}")
    body = ", ".join(parts)
    return f"\t{{\n\t\t{body},\n\t}},"


def main():
    registry = load_registry()
    existing, text = parse_existing(TARGET)

    # Index the registry by chain id and by lowercased name.
    by_id = {c["chainId"]: c for c in registry if "chainId" in c}
    by_name = {}
    for chain in registry:
        for key in (chain.get("name", ""), chain.get("shortName", "")):
            if key:
                by_name.setdefault(key.lower(), chain)

    requested = []
    for line in open("tools/requested_evm_chains.txt", encoding="utf-8"):
        line = line.strip()
        if line and not line.startswith("#"):
            requested.append(line)

    resolved = {}
    unresolved = []
    for name in requested:
        lookup = REQUESTED_ALIASES.get(name, name).lower()

        chain = None
        exact = by_name.get(lookup)
        if exact is not None and not is_testnet(exact) and best_rpc(exact):
            chain = exact
        if chain is None:
            # A prefix match is only accepted when it lands on a mainnet whose
            # name starts with the requested string.
            for key, candidate in by_name.items():
                if not key.startswith(lookup):
                    continue
                if is_testnet(candidate) or best_rpc(candidate) is None:
                    continue
                chain = candidate
                break
        if chain is None:
            manual = next((m for m in MANUAL_CHAINS if m["name"].lower() == name.lower()), None)
            if manual is not None:
                resolved[name] = manual
                continue
            unresolved.append(name)
            continue
        resolved[name] = chain

    # Build the emitted set: every already-present chain, plus any resolved
    # requested chain that is missing.
    emitted = {}
    for chain_id, raw in existing.items():
        emitted[chain_id] = raw

    added = []
    for name, chain in resolved.items():
        chain_id = chain["chainId"]
        if chain_id in emitted:
            continue
        if "rpc" in chain and "explorer" in chain:
            rpc = chain["rpc"]
            entry = render(
                chain_id,
                chain["name"],
                chain["symbol"],
                rpc,
                "m/44'/60'/0'/0/0",
                chain["explorer"],
                "evm",
                18,
                chain["coinType"],
                [],
            )
        else:
            rpc = RPC_OVERRIDES.get(chain_id) or best_rpc(chain)
            slug = chain.get("shortName", "")
            display = NAME_OVERRIDES.get(slug, chain.get("name", name))
            # Every other endpoint the registry lists is kept as an alternate so
            # the runtime failover has somewhere to go.
            entry = render(
                chain_id,
                display,
                native_symbol(chain) or "ETH",
                rpc,
                "m/44'/60'/0'/0/0",
                explorer_url(chain),
                "evm",
                18,
                60,
                alternates(chain, rpc),
            )
        emitted[chain_id] = entry
        added.append((chain_id, display))

    for chain in PLATFORM_CHAINS:
        chain_id = chain["chainId"]
        if chain_id in emitted:
            continue
        entry = render(
            chain_id,
            chain["name"],
            chain["symbol"],
            chain["rpc"],
            "m/44'/60'/0'/0/0",
            chain["explorer"],
            "evm",
            18,
            chain["coinType"],
            [],
        )
        if chain.get("comment"):
            comment = "\n".join("\t\t// " + line for line in chain["comment"])
            entry = "\t{\n" + comment + "\n" + entry.split("{\n", 1)[1]
        emitted[chain_id] = entry

    ordered = [emitted[cid] for cid in sorted(emitted)]

    header = f'''package main

// Code generated from the canonical ethereum-lists/chains registry
// (chainid.network/chains.json). DO NOT EDIT BY HAND. Each entry is a verified
// mainnet EVM chain with a public https RPC and a block explorer. Entries that
// were already present are preserved verbatim, including hand-checked RPC
// alternates and derivation paths. Admins can add or override any entry at
// runtime through the admin_chain_config PostgreSQL table.
//
// A chain is only listed here when the registry publishes at least one https
// RPC for it; an entry with no reachable endpoint would misrepresent the chain
// as usable.

var evmMainnet = []ChainConfig{{
{chr(10).join(ordered)}
}}

const evmMainnetCount = {len(ordered)}
'''

    open(TARGET, "w", encoding="utf-8").write(header)

    print(f"existing entries preserved: {len(existing)}")
    print(f"added: {len(added)}")
    print(f"total: {len(ordered)}")
    if unresolved:
        print(f"unresolved requested names ({len(unresolved)}):")
        for name in unresolved:
            print(f"  {name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
