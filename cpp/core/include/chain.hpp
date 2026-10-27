// Chain configuration: RPC endpoints, explorer URLs and native token metadata.
//
// The registry ships with the chain ids, native symbols and public explorer
// URLs for the networks the wallet supports. RPC endpoints are deliberately not
// bundled: the operator supplies them, either at construction or per chain via
// SetRpcUrl. A chain with no endpoint reports that it is unconfigured instead
// of falling back to a third-party service.

#ifndef TIGERWALLET_CHAIN_HPP
#define TIGERWALLET_CHAIN_HPP

#include <cstdint>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

namespace tigerwallet {

struct ChainConfig {
    uint64_t chain_id = 0;
    std::string name;
    std::string native_symbol;
    uint8_t native_decimals = 18;
    std::string explorer_url;   // no trailing slash
    std::string rpc_url;        // empty until the operator sets one
    bool is_evm = true;
    bool enabled = true;
};

class ChainRegistry {
public:
    ChainRegistry();

    // Looks up a built-in chain by id.
    std::optional<ChainConfig> Get(uint64_t chain_id) const;

    // Adds or replaces a chain, for custom networks.
    void Set(const ChainConfig& config);

    // Sets only the RPC endpoint, keeping the rest of the entry.
    bool SetRpcUrl(uint64_t chain_id, const std::string& rpc_url);

    // Enables or disables a chain. Disabled chains are still returned by Get()
    // but callers should refuse to transact on them.
    bool SetEnabled(uint64_t chain_id, bool enabled);

    std::vector<ChainConfig> All() const;

    // Chains that have an RPC endpoint configured.
    std::vector<ChainConfig> Configured() const;

    bool IsSupported(uint64_t chain_id) const;

    // Builds an explorer link for a transaction hash, or an empty string when
    // the chain is unknown or has no explorer.
    std::string ExplorerTxUrl(uint64_t chain_id, const std::string& tx_hash) const;
    std::string ExplorerAddressUrl(uint64_t chain_id, const std::string& address) const;

private:
    mutable std::mutex mutex_;
    std::unordered_map<uint64_t, ChainConfig> chains_;
};

// The built-in chain list, without RPC endpoints.
std::vector<ChainConfig> BuiltinChains();

}  // namespace tigerwallet

#endif  // TIGERWALLET_CHAIN_HPP
