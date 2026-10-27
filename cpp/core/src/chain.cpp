// Chain registry. See include/chain.hpp.

#include "chain.hpp"

#include <algorithm>

namespace tigerwallet {

std::vector<ChainConfig> BuiltinChains() {
    // Chain ids, native symbols and public explorer base URLs. No RPC endpoint
    // is bundled; the operator configures one per chain.
    return {
        {1, "Ethereum", "ETH", 18, "https://etherscan.io", "", true, true},
        {56, "BNB Smart Chain", "BNB", 18, "https://bscscan.com", "", true, true},
        {137, "Polygon", "POL", 18, "https://polygonscan.com", "", true, true},
        {42161, "Arbitrum One", "ETH", 18, "https://arbiscan.io", "", true, true},
        {10, "OP Mainnet", "ETH", 18, "https://optimistic.etherscan.io", "", true, true},
        {8453, "Base", "ETH", 18, "https://basescan.org", "", true, true},
        {43114, "Avalanche C-Chain", "AVAX", 18, "https://snowtrace.io", "", true, true},
        {250, "Fantom Opera", "FTM", 18, "https://ftmscan.com", "", true, true},
        {59144, "Linea", "ETH", 18, "https://lineascan.build", "", true, true},
        {534352, "Scroll", "ETH", 18, "https://scrollscan.com", "", true, true},
        {324, "zkSync Era", "ETH", 18, "https://explorer.zksync.io", "", true, true},
        {5000, "Mantle", "MNT", 18, "https://mantlescan.xyz", "", true, true},
        {81457, "Blast", "ETH", 18, "https://blastscan.io", "", true, true},
        {34443, "Mode", "ETH", 18, "https://modescan.io", "", true, true},
        {1101, "Polygon zkEVM", "ETH", 18, "https://zkevm.polygonscan.com", "", true, true},
        {100, "Gnosis", "xDAI", 18, "https://gnosisscan.io", "", true, true},
        {25, "Cronos", "CRO", 18, "https://cronoscan.com", "", true, true},
        {2222, "Kava EVM", "KAVA", 18, "https://kavascan.com", "", true, true},
        {42220, "Celo", "CELO", 18, "https://celoscan.io", "", true, true},
        {1313161554, "Aurora", "ETH", 18, "https://aurorascan.dev", "", true, true},
        {1088, "Metis Andromeda", "METIS", 18, "https://andromeda-explorer.metis.io", "", true, true},
        {288, "Boba", "ETH", 18, "https://bobascan.com", "", true, true},
        {2020, "Ronin", "RON", 18, "https://app.roninchain.com", "", true, true},
        {8217, "Kaia", "KAIA", 18, "https://kaiascan.io", "", true, true},
        {204, "opBNB", "BNB", 18, "https://opbnbscan.com", "", true, true},
        {7777777, "Zora", "ETH", 18, "https://explorer.zora.energy", "", true, true},
        {480, "World Chain", "ETH", 18, "https://worldscan.org", "", true, true},
        {252, "Fraxtal", "FRAX", 18, "https://fraxscan.com", "", true, true},
        {17000, "Holesky", "ETH", 18, "https://holesky.etherscan.io", "", true, true},
        {11155111, "Sepolia", "ETH", 18, "https://sepolia.etherscan.io", "", true, true},
    };
}

ChainRegistry::ChainRegistry() {
    for (const auto& chain : BuiltinChains()) {
        chains_.emplace(chain.chain_id, chain);
    }
}

std::optional<ChainConfig> ChainRegistry::Get(uint64_t chain_id) const {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = chains_.find(chain_id);
    if (it == chains_.end()) return std::nullopt;
    return it->second;
}

void ChainRegistry::Set(const ChainConfig& config) {
    std::lock_guard<std::mutex> lock(mutex_);
    chains_[config.chain_id] = config;
}

bool ChainRegistry::SetRpcUrl(uint64_t chain_id, const std::string& rpc_url) {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = chains_.find(chain_id);
    if (it == chains_.end()) return false;
    it->second.rpc_url = rpc_url;
    return true;
}

bool ChainRegistry::SetEnabled(uint64_t chain_id, bool enabled) {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = chains_.find(chain_id);
    if (it == chains_.end()) return false;
    it->second.enabled = enabled;
    return true;
}

std::vector<ChainConfig> ChainRegistry::All() const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<ChainConfig> out;
    out.reserve(chains_.size());
    for (const auto& kv : chains_) out.push_back(kv.second);
    std::sort(out.begin(), out.end(), [](const ChainConfig& a, const ChainConfig& b) {
        return a.chain_id < b.chain_id;
    });
    return out;
}

std::vector<ChainConfig> ChainRegistry::Configured() const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<ChainConfig> out;
    for (const auto& kv : chains_) {
        if (!kv.second.rpc_url.empty()) out.push_back(kv.second);
    }
    std::sort(out.begin(), out.end(), [](const ChainConfig& a, const ChainConfig& b) {
        return a.chain_id < b.chain_id;
    });
    return out;
}

bool ChainRegistry::IsSupported(uint64_t chain_id) const {
    std::lock_guard<std::mutex> lock(mutex_);
    return chains_.count(chain_id) != 0;
}

std::string ChainRegistry::ExplorerTxUrl(uint64_t chain_id,
                                         const std::string& tx_hash) const {
    const auto chain = Get(chain_id);
    if (!chain || chain->explorer_url.empty() || tx_hash.empty()) return {};
    return chain->explorer_url + "/tx/" + tx_hash;
}

std::string ChainRegistry::ExplorerAddressUrl(uint64_t chain_id,
                                              const std::string& address) const {
    const auto chain = Get(chain_id);
    if (!chain || chain->explorer_url.empty() || address.empty()) return {};
    return chain->explorer_url + "/address/" + address;
}

}  // namespace tigerwallet
