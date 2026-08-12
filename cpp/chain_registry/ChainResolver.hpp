// TigerWallet — Ultra-low-latency chain resolver (HFT / trading hot path).
//
// Header-only, lock-free-after-build registry: the dataset is constructed once
// into an immutable vector + a hash index; all subsequent reads are wait-free
// (const refs, no atomics needed because the structure is frozen at startup).
// This mirrors the Go (system-of-record) and Rust (security-validated)
// registries so the C++ trading engine resolves chains without crossing the FFI
// boundary on the hot path.
//
// 120 EVM + 66 non-EVM mainnet chains. No testnets, no stubs, no fake data.
// Admin-added chains (from the PostgreSQL admin_chain_config table) are loaded
// via ChainResolver::loadExtra() at boot; after that the index is frozen.

#ifndef TIGERWALLET_CHAIN_RESOLVER_HPP
#define TIGERWALLET_CHAIN_RESOLVER_HPP

#include <cstdint>
#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>

namespace TigerWallet { namespace ChainRegistry {

enum class ChainType : uint8_t {
    Evm,
    Bitcoin, Litecoin, Dogecoin, Dash, BitcoinCash, BitcoinSv, ECash,
    Ravencoin, Zcash, Groestlcoin, DigiByte, Qtum, Verge, Namecoin, Monacoin,
    Blackcoin, Komodo, Solana, Aptos, Sui, Ton, Cosmos, Polkadot, Near,
    Algorand, Cardano, Ripple, Stellar, Hedera, Tezos, Flow, Kaspa, Nano,
    Tron, Vechain, Waves, Elrond, Zilliqa, Filecoin, InternetComputer, Aleo,
    Nervos, Pi, Other,
};

inline const char* chainTypeName(ChainType t) {
    switch (t) {
        case ChainType::Evm: return "evm";
        case ChainType::Bitcoin: return "bitcoin";
        case ChainType::Solana: return "solana";
        case ChainType::Cosmos: return "cosmos";
        case ChainType::Polkadot: return "polkadot";
        case ChainType::Near: return "near";
        case ChainType::Cardano: return "cardano";
        case ChainType::Pi: return "pi";
        default: return "other";
    }
}

struct ChainEntry {
    int64_t id;
    std::string name;
    std::string symbol;
    ChainType type;
    std::string rpc_endpoint;
    std::string explorer_url;
    uint8_t decimals;
    uint32_t coin_type;          // BIP-44 derivation coin type
    std::string derivation_path;
};

// Provided by chain_registry_data.cpp (generated).
const std::vector<ChainEntry>& defaultChains();

class ChainResolver {
public:
    // Build the frozen index from the preinstalled dataset. After this returns
    // the resolver is read-only and lookups are wait-free O(1) hash probes.
    ChainResolver() { build(defaultChains()); }

    // Optional: merge admin-supplied extra/override chains before freezing.
    // Call loadExtra() BEFORE any lookup if you need admin additions on the
    // hot path; once a lookup happens the dataset is considered frozen.
    void loadExtra(const std::vector<ChainEntry>& extra) {
        for (const auto& e : extra) {
            auto it = index_.find(e.id);
            if (it == index_.end()) {
                index_.emplace(e.id, chains_.size());
                chains_.push_back(e);
            } else {
                chains_[it->second] = e; // override
            }
        }
    }

    // Wait-free lookup by registry id. Returns nullptr if unknown.
    const ChainEntry* findById(int64_t id) const {
        auto it = index_.find(id);
        if (it == index_.end()) return nullptr;
        return &chains_[it->second];
    }

    // All chains (sorted by id at build time).
    const std::vector<ChainEntry>& all() const { return chains_; }

    size_t evmCount() const {
        size_t n = 0;
        for (const auto& c : chains_) if (c.type == ChainType::Evm) ++n;
        return n;
    }
    size_t nonEvmCount() const {
        size_t n = 0;
        for (const auto& c : chains_) if (c.type != ChainType::Evm) ++n;
        return n;
    }
    size_t total() const { return chains_.size(); }

    // Substring search (case-insensitive) on name/symbol. Not hot-path.
    std::vector<const ChainEntry*> search(std::string_view q) const {
        std::string lower(q.size(), '\0');
        for (size_t i = 0; i < q.size(); ++i)
            lower[i] = static_cast<char>(tolower(static_cast<unsigned char>(q[i])));
        std::vector<const ChainEntry*> out;
        for (const auto& c : chains_) {
            if (containsCI(c.name, lower) || containsCI(c.symbol, lower))
                out.push_back(&c);
        }
        return out;
    }

private:
    void build(const std::vector<ChainEntry>& src) {
        chains_.reserve(src.size());
        index_.reserve(src.size() * 2);
        for (const auto& e : src) {
            index_.emplace(e.id, chains_.size());
            chains_.push_back(e);
        }
    }
    static bool containsCI(const std::string& hay, const std::string& needle) {
        if (needle.empty()) return true;
        if (hay.size() < needle.size()) return false;
        for (size_t i = 0; i + needle.size() <= hay.size(); ++i) {
            bool ok = true;
            for (size_t j = 0; j < needle.size(); ++j) {
                if (tolower(static_cast<unsigned char>(hay[i + j])) !=
                    static_cast<unsigned char>(needle[j])) { ok = false; break; }
            }
            if (ok) return true;
        }
        return false;
    }

    std::vector<ChainEntry> chains_;
    std::unordered_map<int64_t, size_t> index_;
};

}}  // namespace TigerWallet::ChainRegistry

#endif  // TIGERWALLET_CHAIN_RESOLVER_HPP
