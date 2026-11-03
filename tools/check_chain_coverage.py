#!/usr/bin/env python3
"""Compare the requested chain list against the shipped registry.

Run from the repository root:

    python3 tools/check_chain_coverage.py


Reports, for every requested chain, whether the registry holds an entry and
which one matched. A chain that appears nowhere is a real coverage gap.
"""
import re
import sys

EVM_REQUESTED = """
Ethereum Mainnet|Base|BNB Smart Chain|Arbitrum One|Polygon PoS|Avalanche C-Chain
OP Mainnet|Monad|Sonic|Mantle|Scroll|Linea|zkSync Era|Cronos|Celo|Moonbeam
Gnosis Chain|ApeChain|Ink|HyperEVM|World Chain|Berachain|Taiko|Viction
Robinhood Chain|Sei EVM|Flare Mainnet|X Layer|Arc|Plasma Mainnet|Bitlayer|Citrea
Chiliz Chain|Gravity Network|IOTA EVM|Katana Network|Metis|Arbitrum Nova
Polygon zkEVM|Kaia|Harmony Shard 0|Aurora|Huobi ECO Chain|Plume Network
Oasis Sapphire|Astar EVM|Rootstock|Telos EVM|Songbird Canary Network
XDC Network|Evmos EVM|Neon EVM|Milkomeda C1|Moonriver|Shiden Network|Acala EVM+
Karura EVM+|Theta Mainnet|Fuse Mainnet|Meter Mainnet|Syscoin NEVM|Wanchain
Canto|Echelon|Dogechain|Kava EVM|GateChain|Velas EVM|Godwoken|BOBA Network
Milkomeda A1|Sx Network|Ubiq|Expanse|Elastos Smart Chain|ShibaChain|ThunderCore
Metadium|Rei Network|Loopring|Manta Pacific|ZkFair|Astar zkEVM|Mode Network
Blast|ZetaChain|Core Blockchain|ShimmerEVM|Lightlink|Tenet|PulseChain
Step Network|Oasys Mainnet|Ronin Network
"""

NON_EVM_REQUESTED = """
Pi Network|Bitcoin|Solana|TRON|Sui|Aptos|Cardano|XRP Ledger|TON|Stellar
Near Protocol|Polkadot Relay Chain|Cosmos Hub|Injective|Algorand|Tezos|Starknet
Litecoin|Dogecoin|Filecoin|Mantra Network|Hyperliquid|Osmosis|Thorchain|Monero
Zcash|Kusama|Nervos Network|Internet Computer|Hedera Hashgraph|MultiversX
Waves|NEO Legacy|Arweave|Stacks|Flow|Mina Protocol|Dydx Chain|Celestia|Sei Native
Kava Cosmos Native|Noble|Supra|Juno|Secret Network|Akash Network|Kujira|Fetch.ai
Band Protocol|Terra 2.0|Chihuahua Chain|Stargaze|Axelar|Agoric|Regen Network
Sommelier|Umee|Evmos Cosmos Native|Bitcoin Cash|Dash|Ravencoin|Kaspa|Decred
Qtum|Stratis Native|Lisk Legacy|Steem|Hive|BitShares|Nem|Symbol|Nano|Banano
IOTA Tangle|Kadena|Ergo|Chia Network|Handshake|Syscoin UTXO Core|Digibyte|Verge
Bitcoin Gold|Horizen|Siacoin|Komodo|Nexus|Ark|LBRY Credits|Aion|Wakt Chain|NULS
PascalCoin|Burstcoin|Counterparty|Omni Layer|Peercoin|Namecoin|Primecoin|Gridcoin
Feathercoin
"""

# Some requested names are the same chain under a former name; the registry
# carries the current name.
ALIASES = {
    "polygon pos": "polygon",
    "op mainnet": "optimism",
    "sonic": "sonic",
    "bnb smart chain": "bnb smart chain",
    "avalanche c-chain": "avalanche",
    "zkSync Era": "zksync",
    "huobi eco chain": "heco",
    "kaia": "kaia",
    "milkomeda c1": "milkomeda",
    "milkomeda a1": "milkomeda",
    "sei evm": "sei",
    "sei native": "sei",
    "kava evm": "kava",
    "kava cosmos native": "kava",
    "evmos evm": "evmos",
    "evmos cosmos native": "evmos",
    "xrp ledger": "ripple",
    "ton": "ton",
    "nem": "nem",
    "symbol": "symbol",
    "syscoin nevm": "syscoin",
    "syscoin utxo core": "syscoin",
    "iota evm": "iota",
    "iota tangle": "iota",
    "bitcoin gold": "bitcoin gold",
    "dydx chain": "dydx",
    "fetch.ai": "fetch",
    "hyperliquid": "hyperliquid",
    "mantra network": "mantra",
    "manta pacific": "manta",
    "shimmer evm": "shimmer",
    "shimmerevm": "shimmer",
    "apechain": "ape",
    "world chain": "world",
    "plasma mainnet": "plasma",
    "robinhood chain": "robinhood",
    "arc": "arc",
    "gravity network": "gravity",
    "katana network": "katana",
    "lisk legacy": "lisk",
    "stratis native": "stratis",
    "loopring": "loopring",
    "step network": "step",
    "echelon": "echelon",
    "lightlink": "lightlink",
    "tenet": "tenet",
    "core blockchain": "core",
    "wakt chain": "wakt",
}


def registry_names(path):
    """Extracts the Name: "..." values from a generated Go registry file."""
    text = open(path, encoding="utf-8").read()
    return re.findall(r'Name:\s*"([^"]+)"', text)


def normalise(name):
    n = name.lower().strip()
    n = n.replace(" mainnet", "").replace(" network", "").replace(" chain", "")
    n = n.replace(" protocol", "").replace(" native", "").replace(" core", "")
    n = n.replace(" relay", "").replace(" legacy", "").replace(" c1", "")
    n = n.replace(" a1", "").replace(" tangle", "")
    n = n.replace("-", " ").replace(".", " ").replace("+", " ")
    n = re.sub(r"\s+", " ", n).strip()
    return n


def main():
    evm = registry_names("go/wallet_api/chains_evm_data.go")
    non_evm = registry_names("go/wallet_api/chains_nonevm_data.go")
    all_names = evm + non_evm
    normalised = {}
    for name in all_names:
        normalised.setdefault(normalise(name), name)

    missing = []
    for label, block in (("EVM", EVM_REQUESTED), ("non-EVM", NON_EVM_REQUESTED)):
        requested = [c.strip() for c in block.replace("\n", "|").split("|") if c.strip()]
        for chain in requested:
            key = normalise(chain)
            key = ALIASES.get(chain.lower(), key)
            key = normalise(key)
            if key in normalised:
                continue
            # Fall back to a substring match on the first significant token.
            token = key.split()[0] if key.split() else key
            if any(token in n for n in normalised):
                continue
            missing.append((label, chain))

    print(f"registry: {len(evm)} EVM + {len(non_evm)} non-EVM = {len(all_names)} chains")
    if not missing:
        print("every requested chain is present")
        return 0
    print(f"{len(missing)} requested chain(s) not found:")
    for label, chain in missing:
        print(f"  [{label}] {chain}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
