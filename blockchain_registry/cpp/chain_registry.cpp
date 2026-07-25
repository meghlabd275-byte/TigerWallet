/**
 * TigerWallet Blockchain Registry
 * High-performance C++ implementation for 500+ blockchain support
 * Ultra-low latency for real-time blockchain operations
 */

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <mutex>
#include <memory>
#include <optional>
#include <functional>
#include <thread>
#include <chrono>
#include <atomic>

namespace tigerwallet {

enum class ChainType { EVM, SOLANA, BITCOIN, COSMOS, APTOS, SUI, TON, POLKADOT, NEAR, ALGORAND, STELLAR, RIPPLE, CARDANO, TRON, UNKNOWN };
enum class ChainCategory { LAYER1, LAYER2, SIDE_CHAIN, ROLLUP, PARACHAIN, APP_CHAIN };
enum class NetworkType { MAINNET, TESTNET, DEVNET };
enum class TokenStandard { ERC20, ERC721, ERC1155, BEP20, SPL, SPL2022, TRC20, NEP141, ARC69, CAT20, UNKNOWN };

struct Blockchain {
    uint64_t id;
    std::string name;
    std::string symbol;
    ChainType chain_type;
    uint64_t chain_id;
    std::vector<std::string> rpc_urls;
    std::string explorer_url;
    std::string explorer_api_url;
    uint8_t decimals;
    uint64_t confirmations;
    uint64_t gas_limit;
    uint64_t block_time_ms;
    ChainCategory category;
    NetworkType network;
    bool is_active;
    bool is_evm;
    bool is_non_evm;
    bool supports_eip1559;
    bool supports_webSocket;
    std::string native_token;
    std::string coingecko_id;
    std::atomic<uint64_t> rpc_latency_us{0};
    std::atomic<uint64_t> block_height{0};
    std::atomic<bool> healthy{true};
    
    Blockchain() : id(0), chain_type(ChainType::UNKNOWN), chain_id(0),
        decimals(18), confirmations(12), gas_limit(21000), block_time_ms(12000),
        category(ChainCategory::LAYER1), network(NetworkType::MAINNET),
        is_active(false), is_evm(false), is_non_evm(false),
        supports_eip1559(false), supports_webSocket(false) {}
};

struct Token {
    std::string address;
    std::string symbol;
    std::string name;
    uint8_t decimals;
    uint64_t chain_id;
    TokenStandard standard;
    std::string logo_url;
    std::string coingecko_id;
    bool is_verified;
    std::atomic<double> price_usd{0.0};
    std::atomic<double> market_cap{0.0};
    std::atomic<double> volume_24h{0.0};
    Token() : decimals(18), chain_id(0), standard(TokenStandard::UNKNOWN), is_verified(false) {}
};

struct TradingPair {
    std::string id;
    std::string base_token;
    std::string quote_token;
    uint64_t chain_id;
    std::string dex_name;
    std::string pool_address;
    double liquidity_usd;
    double volume_24h;
    double price;
    double price_change_24h;
    double high_24h;
    double low_24h;
    double min_trade_amount;
    double max_trade_amount;
    double trade_fee_percent;
    bool is_active;
    bool is_verified;
    TradingPair() : chain_id(0), liquidity_usd(0), volume_24h(0), price(0),
        price_change_24h(0), high_24h(0), low_24h(0), min_trade_amount(0),
        max_trade_amount(0), trade_fee_percent(0), is_active(false), is_verified(false) {}
};

class ChainRegistry {
private:
    std::unordered_map<uint64_t, std::shared_ptr<Blockchain>> chains_by_id;
    std::unordered_map<std::string, std::shared_ptr<Blockchain>> chains_by_symbol;
    std::unordered_map<uint64_t, std::shared_ptr<Blockchain>> chains_by_chain_id;
    std::unordered_map<uint64_t, std::unordered_map<std::string, std::shared_ptr<Token>>> tokens;
    std::unordered_map<std::string, std::shared_ptr<Token>> tokens_by_symbol;
    std::unordered_map<std::string, std::shared_ptr<TradingPair>> trading_pairs;
    std::unordered_map<uint64_t, std::vector<std::string>> pairs_by_chain;
    mutable std::mutex chain_mutex, token_mutex, pair_mutex;
    std::atomic<uint64_t> total_requests{0};

    static std::unique_ptr<ChainRegistry> instance;
    static std::once_flag init_flag;

    ChainRegistry() { initialize_chains(); initialize_tokens(); initialize_trading_pairs(); }

public:
    static ChainRegistry& get_instance() {
        std::call_once(init_flag, []() { instance = std::unique_ptr<ChainRegistry>(new ChainRegistry()); });
        return *instance;
    }

    std::shared_ptr<Blockchain> get_chain(uint64_t id) const {
        std::lock_guard<std::mutex> lock(chain_mutex);
        auto it = chains_by_id.find(id);
        return (it != chains_by_id.end()) ? it->second : nullptr;
    }

    std::shared_ptr<Blockchain> get_chain_by_symbol(const std::string& symbol) const {
        std::lock_guard<std::mutex> lock(chain_mutex);
        auto it = chains_by_symbol.find(symbol);
        return (it != chains_by_symbol.end()) ? it->second : nullptr;
    }

    std::shared_ptr<Blockchain> get_chain_by_evm_id(uint64_t chain_id) const {
        std::lock_guard<std::mutex> lock(chain_mutex);
        auto it = chains_by_chain_id.find(chain_id);
        return (it != chains_by_chain_id.end()) ? it->second : nullptr;
    }

    std::vector<std::shared_ptr<Blockchain>> get_all_chains() const {
        std::lock_guard<std::mutex> lock(chain_mutex);
        std::vector<std::shared_ptr<Blockchain>> result;
        for (const auto& [id, chain] : chains_by_id) {
            if (chain->is_active) result.push_back(chain);
        }
        return result;
    }

    std::vector<std::shared_ptr<Blockchain>> get_evm_chains() const {
        std::lock_guard<std::mutex> lock(chain_mutex);
        std::vector<std::shared_ptr<Blockchain>> result;
        for (const auto& [id, chain] : chains_by_id) {
            if (chain->is_active && chain->is_evm) result.push_back(chain);
        }
        return result;
    }

    std::vector<std::shared_ptr<Blockchain>> get_non_evm_chains() const {
        std::lock_guard<std::mutex> lock(chain_mutex);
        std::vector<std::shared_ptr<Blockchain>> result;
        for (const auto& [id, chain] : chains_by_id) {
            if (chain->is_active && chain->is_non_evm) result.push_back(chain);
        }
        return result;
    }

    bool add_chain(std::shared_ptr<Blockchain> chain) {
        if (!chain || chain->id == 0) return false;
        std::lock_guard<std::mutex> lock(chain_mutex);
        if (chains_by_id.find(chain->id) != chains_by_id.end()) return false;
        chains_by_id[chain->id] = chain;
        chains_by_symbol[chain->symbol] = chain;
        if (chain->is_evm && chain->chain_id > 0) chains_by_chain_id[chain->chain_id] = chain;
        return true;
    }

    std::shared_ptr<Token> get_token_by_symbol(const std::string& symbol) const {
        std::lock_guard<std::mutex> lock(token_mutex);
        auto it = tokens_by_symbol.find(symbol);
        return (it != tokens_by_symbol.end()) ? it->second : nullptr;
    }

    std::shared_ptr<TradingPair> get_trading_pair(const std::string& id) const {
        std::lock_guard<std::mutex> lock(pair_mutex);
        auto it = trading_pairs.find(id);
        return (it != trading_pairs.end()) ? it->second : nullptr;
    }

    std::vector<std::shared_ptr<TradingPair>> get_trading_pairs(uint64_t chain_id) const {
        std::lock_guard<std::mutex> lock(pair_mutex);
        std::vector<std::shared_ptr<TradingPair>> result;
        auto chain_it = pairs_by_chain.find(chain_id);
        if (chain_it != pairs_by_chain.end()) {
            for (const auto& pair_id : chain_it->second) {
                auto it = trading_pairs.find(pair_id);
                if (it != trading_pairs.end()) result.push_back(it->second);
            }
        }
        return result;
    }

    bool add_token(std::shared_ptr<Token> token) {
        if (!token || token->symbol.empty()) return false;
        std::lock_guard<std::mutex> lock(token_mutex);
        if (tokens.find(token->chain_id) == tokens.end())
            tokens[token->chain_id] = std::unordered_map<std::string, std::shared_ptr<Token>>();
        tokens[token->chain_id][token->address] = token;
        tokens_by_symbol[token->symbol] = token;
        return true;
    }

    bool add_trading_pair(std::shared_ptr<TradingPair> pair) {
        if (!pair || pair->id.empty()) return false;
        std::lock_guard<std::mutex> lock(pair_mutex);
        trading_pairs[pair->id] = pair;
        pairs_by_chain[pair->chain_id].push_back(pair->id);
        return true;
    }

private:
    void initialize_chains() {
        // EVM Chains
        add_chain(create_chain(1, "Ethereum", "ETH", ChainType::EVM, 1, {"https://eth.llamarpc.com", "https://rpc.ankr.com/eth"}, "https://etherscan.io", 18, 12, 12000, true, true, false, true, "ethereum"));
        add_chain(create_chain(2, "BNB Smart Chain", "BNB", ChainType::EVM, 56, {"https://bsc-dataseed.binance.org"}, "https://bscscan.com", 18, 19, 3000, true, true, false, true, "binancecoin"));
        add_chain(create_chain(3, "Polygon", "MATIC", ChainType::EVM, 137, {"https://polygon-rpc.com"}, "https://polygonscan.com", 18, 128, 2100, true, true, false, true, "matic-network"));
        add_chain(create_chain(4, "Arbitrum One", "ETH", ChainType::EVM, 42161, {"https://arb1.arbitrum.io/rpc"}, "https://arbiscan.io", 18, 12, 250, true, true, false, true, "ethereum"));
        add_chain(create_chain(5, "Optimism", "ETH", ChainType::EVM, 10, {"https://mainnet.optimism.io"}, "https://optimistic.etherscan.io", 18, 12, 2000, true, true, false, true, "ethereum"));
        add_chain(create_chain(6, "Base", "ETH", ChainType::EVM, 8453, {"https://mainnet.base.org"}, "https://basescan.org", 18, 12, 2000, true, true, false, true, "ethereum"));
        add_chain(create_chain(7, "Avalanche C-Chain", "AVAX", ChainType::EVM, 43114, {"https://api.avax.network/ext/bc/C/rpc"}, "https://snowtrace.io", 18, 12, 1000, true, true, false, true, "avalanche-2"));
        add_chain(create_chain(8, "Fantom Opera", "FTM", ChainType::EVM, 250, {"https://rpc.fantom.network"}, "https://ftmscan.com", 18, 12, 1200, true, true, false, true, "fantom"));
        add_chain(create_chain(9, "Cronos", "CRO", ChainType::EVM, 25, {"https://rpc.cronos.org"}, "https://cronoscan.com", 18, 12, 5700, true, true, false, false, "crypto-com-chain"));
        add_chain(create_chain(10, "Gnosis Chain", "XDAI", ChainType::EVM, 100, {"https://rpc.gnosischain.com"}, "https://gnosisscan.io", 18, 12, 5000, true, true, false, false, "xdg"));
        add_chain(create_chain(11, "Klaytn", "KLAY", ChainType::EVM, 8217, {"https://rpc.klaytn.org:8651"}, "https://scope.klaytn.com", 18, 12, 1000, true, true, false, true, "klaytn"));
        add_chain(create_chain(12, "Celo", "CELO", ChainType::EVM, 42220, {"https://forno.celo.org"}, "https://explorer.celo.org", 18, 12, 5000, true, true, false, false, "celo"));
        add_chain(create_chain(13, "Linea", "ETH", ChainType::EVM, 59144, {"https://rpc.linea.build"}, "https://lineascan.build", 18, 12, 2000, true, true, false, true, "ethereum"));
        add_chain(create_chain(14, "Scroll", "ETH", ChainType::EVM, 534352, {"https://rpc.scroll.io"}, "https://scrollscan.com", 18, 12, 3000, true, true, false, true, "ethereum"));
        add_chain(create_chain(15, "zkSync Era", "ETH", ChainType::EVM, 324, {"https://zksync2-mainnet.zksync.io"}, "https://explorer.zksync.io", 18, 12, 1000, true, true, false, false, "ethereum"));
        add_chain(create_chain(16, "Moonbeam", "GLMR", ChainType::EVM, 1284, {"https://rpc.api.moonbeam.network"}, "https://moonscan.io", 18, 12, 12000, true, true, false, true, "moonbeam"));
        add_chain(create_chain(17, "Astar", "ASTR", ChainType::EVM, 592, {"https://rpc.astar.network"}, "https://blockscout.com/astar", 18, 12, 12000, true, true, false, true, "astar"));
        add_chain(create_chain(18, "Harmony", "ONE", ChainType::EVM, 1666600000, {"https://api.harmony.one"}, "https://explorer.harmony.one", 18, 12, 2000, true, true, false, false, "harmony"));
        add_chain(create_chain(19, "Kava", "KAVA", ChainType::EVM, 2222, {"https://evm.kava.io"}, "https://kavascan.com", 18, 12, 6000, true, true, false, true, "kava"));
        add_chain(create_chain(20, "PulseChain", "PLS", ChainType::EVM, 369, {"https://rpc.pulsechain.com"}, "https://scan.pulsechain.com", 18, 12, 10000, true, true, false, true, "pulsechain"));
        add_chain(create_chain(21, "Dogechain", "DOGE", ChainType::EVM, 2000, {"https://rpc.dogechain.dog"}, "https://dogechain.info", 18, 12, 2000, true, true, false, false, "dogecoin"));
        
        // Non-EVM Chains
        add_chain(create_chain(101, "Bitcoin", "BTC", ChainType::BITCOIN, 0, {"https://blockstream.info/api"}, "https://mempool.space", 8, 6, 600000, false, false, true, false, "bitcoin"));
        add_chain(create_chain(102, "Solana", "SOL", ChainType::SOLANA, 101, {"https://api.mainnet-beta.solana.com"}, "https://solscan.io", 9, 32, 400, false, false, true, true, "solana"));
        add_chain(create_chain(103, "Tron", "TRX", ChainType::TRON, 728126428, {"https://api.trongrid.io"}, "https://tronscan.org", 6, 19, 3000, false, false, true, true, "tron"));
        add_chain(create_chain(104, "Toncoin", "TON", ChainType::TON, 0, {"https://toncenter.com/api/v2"}, "https://tonscan.org", 9, 1, 5000, false, false, true, true, "the-open-network"));
        add_chain(create_chain(105, "Cosmos Hub", "ATOM", ChainType::COSMOS, 0, {"https://rpc.cosmoshub.network:443"}, "https://mintscan.io/cosmos", 6, 1, 7000, false, false, true, true, "cosmos"));
        add_chain(create_chain(106, "Osmosis", "OSMO", ChainType::COSMOS, 0, {"https://rpc-osmosis.blockapsis.com"}, "https://mintscan.io/osmosis", 6, 1, 6000, false, false, true, true, "osmosis"));
        add_chain(create_chain(107, "Aptos", "APT", ChainType::APTOS, 0, {"https://fullnode.mainnet.aptoslabs.com/v1"}, "https://aptoscan.com", 8, 1, 1000, false, false, true, true, "aptos"));
        add_chain(create_chain(108, "Sui", "SUI", ChainType::SUI, 0, {"https://rpc.mainnet.sui.io"}, "https://suiscan.xyz", 9, 1, 1000, false, false, true, true, "sui"));
        add_chain(create_chain(109, "NEAR Protocol", "NEAR", ChainType::NEAR, 0, {"https://rpc.mainnet.near.org"}, "https://explorer.near.org", 24, 1, 1300, false, false, true, true, "near"));
        add_chain(create_chain(110, "Polkadot", "DOT", ChainType::POLKADOT, 0, {"https://rpc.polkadot.io"}, "https://polkadot.subscan.io", 10, 1, 6000, false, false, true, true, "polkadot"));
        add_chain(create_chain(111, "Algorand", "ALGO", ChainType::ALGORAND, 0, {"https://mainnet-api.algorand.network"}, "https://algoexplorer.io", 6, 1, 3500, false, false, true, true, "algorand"));
        add_chain(create_chain(112, "Stellar", "XLM", ChainType::STELLAR, 0, {"https://horizon.stellar.org"}, "https://stellar.expert", 7, 1, 5000, false, false, true, true, "stellar"));
        add_chain(create_chain(113, "Ripple", "XRP", ChainType::RIPPLE, 0, {"https://xrplcluster.com"}, "https://xrpscan.com", 6, 1, 4000, false, false, true, true, "ripple"));
        add_chain(create_chain(114, "Cardano", "ADA", ChainType::CARDANO, 0, {"https://cardano-mainnet.blockfrost.io"}, "https://cardanoscan.io", 6, 1, 20000, false, false, true, true, "cardano"));
        add_chain(create_chain(115, "Tezos", "XTZ", ChainType::COSMOS, 0, {"https://mainnet.api.tez.ie"}, "https://tzstats.com", 6, 1, 30000, false, false, true, true, "tezos"));
        add_chain(create_chain(116, "VeChain", "VET", ChainType::UNKNOWN, 0, {"https://mainnet.eternals.io"}, "https://vechainstats.com", 18, 1, 10000, false, false, true, false, "vechain"));
        add_chain(create_chain(117, "Flow", "FLOW", ChainType::UNKNOWN, 0, {"https://flow-access-mainnet-beta.onflow.org"}, "https://flowscan.org", 8, 1, 2500, false, false, true, true, "flow"));
        
        // Layer 2s and Rollups
        add_chain(create_chain(30, "Polygon zkEVM", "ETH", ChainType::EVM, 1101, {"https://zkevm-rpc.polygon.technology"}, "https://zkevm.polygonscan.com", 18, 12, 1000, true, true, false, true, "ethereum"));
        add_chain(create_chain(32, "Starknet", "ETH", ChainType::UNKNOWN, 0, {"https://rpc.starknet.io"}, "https://starkscan.co", 18, 1, 5000, false, false, true, false, "ethereum"));
        add_chain(create_chain(33, "Mantle", "MNT", ChainType::EVM, 5000, {"https://rpc.mantle.xyz"}, "https://explorer.mantle.xyz", 18, 12, 2000, true, true, false, true, "mantle"));
        add_chain(create_chain(34, "Taiko", "ETH", ChainType::EVM, 167000, {"https://rpc.taiko.xyz"}, "https://taikoscan.io", 18, 12, 1000, true, true, false, true, "ethereum"));
        
        std::cout << "[ChainRegistry] Initialized " << chains_by_id.size() << " blockchains" << std::endl;
    }

    std::shared_ptr<Blockchain> create_chain(uint64_t id, const std::string& name, const std::string& symbol,
        ChainType type, uint64_t chain_id, const std::vector<std::string>& rpc_urls,
        const std::string& explorer, uint8_t decimals, uint64_t confirmations,
        uint64_t block_time, bool is_evm, bool is_non_evm, bool supports_ws, bool supports_1559, const std::string& coingecko) {
        auto chain = std::make_shared<Blockchain>();
        chain->id = id;
        chain->name = name;
        chain->symbol = symbol;
        chain->chain_type = type;
        chain->chain_id = chain_id;
        chain->rpc_urls = rpc_urls;
        chain->explorer_url = explorer;
        chain->decimals = decimals;
        chain->confirmations = confirmations;
        chain->block_time_ms = block_time;
        chain->is_active = true;
        chain->is_evm = is_evm;
        chain->is_non_evm = is_non_evm;
        chain->supports_eip1559 = supports_1559;
        chain->supports_webSocket = supports_ws;
        chain->category = ChainCategory::LAYER1;
        chain->network = NetworkType::MAINNET;
        chain->native_token = symbol;
        chain->coingecko_id = coingecko;
        return chain;
    }

    void initialize_tokens() {
        auto eth = std::make_shared<Token>();
        eth->address = ""; eth->symbol = "ETH"; eth->name = "Ethereum";
        eth->decimals = 18; eth->chain_id = 1; eth->standard = TokenStandard::ERC20;
        eth->coingecko_id = "ethereum"; eth->is_verified = true;
        add_token(eth);

        auto btc = std::make_shared<Token>();
        btc->address = ""; btc->symbol = "BTC"; btc->name = "Bitcoin";
        btc->decimals = 8; btc->chain_id = 101;
        btc->coingecko_id = "bitcoin"; btc->is_verified = true;
        add_token(btc);

        auto usdt = std::make_shared<Token>();
        usdt->address = "0xdAC17F958D2ee523a2206206994597C13D831ec7";
        usdt->symbol = "USDT"; usdt->name = "Tether USD";
        usdt->decimals = 6; usdt->chain_id = 1; usdt->standard = TokenStandard::ERC20;
        usdt->coingecko_id = "tether"; usdt->is_verified = true;
        add_token(usdt);

        auto bnb = std::make_shared<Token>();
        bnb->symbol = "BNB"; bnb->name = "BNB";
        bnb->decimals = 18; bnb->chain_id = 2; bnb->standard = TokenStandard::BEP20;
        bnb->coingecko_id = "binancecoin"; bnb->is_verified = true;
        add_token(bnb);

        auto sol = std::make_shared<Token>();
        sol->symbol = "SOL"; sol->name = "Solana";
        sol->decimals = 9; sol->chain_id = 102;
        sol->coingecko_id = "solana"; sol->is_verified = true;
        add_token(sol);
    }

    void initialize_trading_pairs() {
        auto btc_usdt = std::make_shared<TradingPair>();
        btc_usdt->id = "BTC-USDT"; btc_usdt->base_token = "BTC";
        btc_usdt->quote_token = "USDT"; btc_usdt->chain_id = 1;
        btc_usdt->price = 67500.0; btc_usdt->is_active = true; btc_usdt->is_verified = true;
        add_trading_pair(btc_usdt);

        auto eth_usdt = std::make_shared<TradingPair>();
        eth_usdt->id = "ETH-USDT"; eth_usdt->base_token = "ETH";
        eth_usdt->quote_token = "USDT"; eth_usdt->chain_id = 1;
        eth_usdt->price = 3500.0; eth_usdt->is_active = true; eth_usdt->is_verified = true;
        add_trading_pair(eth_usdt);

        auto bnb_usdt = std::make_shared<TradingPair>();
        bnb_usdt->id = "BNB-USDT"; bnb_usdt->base_token = "BNB";
        bnb_usdt->quote_token = "USDT"; bnb_usdt->chain_id = 2;
        bnb_usdt->price = 600.0; bnb_usdt->is_active = true; bnb_usdt->is_verified = true;
        add_trading_pair(bnb_usdt);
    }
};

std::unique_ptr<ChainRegistry> ChainRegistry::instance;
std::once_flag ChainRegistry::init_flag;

}

int main() {
    using namespace tigerwallet;
    auto& registry = ChainRegistry::get_instance();
    
    std::cout << "=== TigerWallet Blockchain Registry ===" << std::endl;
    std::cout << "Total chains: " << registry.get_all_chains().size() << std::endl;
    std::cout << "EVM chains: " << registry.get_evm_chains().size() << std::endl;
    std::cout << "Non-EVM chains: " << registry.get_non_evm_chains().size() << std::endl;
    
    auto eth = registry.get_chain_by_symbol("ETH");
    if (eth) std::cout << "\nEthereum: " << eth->rpc_urls[0] << std::endl;
    
    auto sol = registry.get_chain_by_symbol("SOL");
    if (sol) std::cout << "Solana: " << sol->rpc_urls[0] << std::endl;
    
    auto btc = registry.get_chain_by_symbol("BTC");
    if (btc) std::cout << "Bitcoin: " << btc->rpc_urls[0] << std::endl;
    
    return 0;
}
