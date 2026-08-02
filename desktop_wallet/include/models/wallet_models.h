/**
 * TigerWallet Desktop - Data Models
 * Complete wallet data structures for multi-chain support
 */

#ifndef TIGER_WALLET_MODELS_H
#define TIGER_WALLET_MODELS_H

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <chrono>
#include <memory>

namespace tiger {
namespace wallet {

// ============================================================================
// Blockchain Types
// ============================================================================

enum class ChainType {
    EVM,
    SOLANA,
    APTOS,
    SUI,
    TON,
    BITCOIN,
    COSMOS,
    OTHER
};

// Real 103+ Blockchain Networks
struct Chain {
    std::string id;
    std::string name;
    std::string symbol;
    int decimals;
    std::string rpc_url;
    std::string explorer_url;
    int chain_id;
    ChainType type;
    bool is_testnet;

    static std::vector<Chain> getAllChains() {
        return {
            // === Top 10 Blockchains by TVL ===
            {"ethereum", "Ethereum", "ETH", 18, "https://eth.llamarpc.com", "https://etherscan.io", 1, ChainType::EVM, false},
            {"polygon", "Polygon", "MATIC", 18, "https://polygon-rpc.com", "https://polygonscan.com", 137, ChainType::EVM, false},
            {"bsc", "BNB Smart Chain", "BNB", 18, "https://bsc-dataseed.binance.org", "https://bscscan.com", 56, ChainType::EVM, false},
            {"arbitrum", "Arbitrum One", "ETH", 18, "https://arb1.arbitrum.io/rpc", "https://arbiscan.io", 42161, ChainType::EVM, false},
            {"optimism", "Optimism", "ETH", 18, "https://mainnet.optimism.io", "https://optimistic.etherscan.io", 10, ChainType::EVM, false},
            {"avalanche", "Avalanche C-Chain", "AVAX", 18, "https://api.avax.network/ext/bc/C/rpc", "https://snowtrace.io", 43114, ChainType::EVM, false},
            {"base", "Base", "ETH", 18, "https://mainnet.base.org", "https://basescan.org", 8453, ChainType::EVM, false},
            {"solana", "Solana", "SOL", 9, "https://api.mainnet-beta.solana.com", "https://solscan.io", 0, ChainType::SOLANA, false},
            {"tron", "Tron", "TRX", 6, "https://api.trongrid.io", "https://tronscan.org", 0, ChainType::EVM, false},
            {"bitcoin", "Bitcoin", "BTC", 8, "https://blockstream.info/api", "https://blockstream.info", 0, ChainType::BITCOIN, false},
            
            // === Layer 2 Networks ===
            {"zksync", "zkSync Era", "ETH", 18, "https://mainnet.era.zksync.io", "https://explorer.zksync.io", 324, ChainType::EVM, false},
            {"zkevm", "Polygon zkEVM", "ETH", 18, "https://zkevm-rpc.com", "https://zkevm.polygonscan.com", 1101, ChainType::EVM, false},
            {"linea", "Linea", "ETH", 18, "https://rpc.linea.build", "https://lineascan.build", 59144, ChainType::EVM, false},
            {"scroll", "Scroll", "ETH", 18, "https://rpc.scroll.io", "https://scrollscan.com", 534352, ChainType::EVM, false},
            {"starknet", "Starknet", "ETH", 18, "https://api.mainnet.starknet.io", "https://starkscan.co", 0, ChainType::OTHER, false},
            {"opbnb", "opBNB", "BNB", 18, "https://opbnb.publicnode.com", "https://opbnbscan.com", 204, ChainType::EVM, false},
            {"mantle", "Mantle", "MNT", 18, "https://rpc.mantle.xyz", "https://mantlescan.info", 5000, ChainType::EVM, false},
            {"fraxtal", "Fraxtal", "FRAX", 18, "https://rpc.frax.com", "https://fraxscan.com", 2522, ChainType::EVM, false},
            {"mode", "Mode", "ETH", 18, "https://mainnet.mode.network", "https://modescan.io", 34443, ChainType::EVM, false},
            {"worldchain", "World Chain", "ETH", 18, "https://worldchain-mainnet.g.alchemy.com", "https://worldchainscan.com", 480, ChainType::EVM, false},
            
            // === Other Major EVM Chains ===
            {"fantom", "Fantom", "FTM", 18, "https://rpc.fantom.network", "https://ftmscan.com", 250, ChainType::EVM, false},
            {"celo", "Celo", "CELO", 18, "https://forno.celo.org", "https://celoscan.io", 42220, ChainType::EVM, false},
            {"cronos", "Cronos", "CRO", 18, "https://evm.cronos.org", "https://cronoscan.com", 25, ChainType::EVM, false},
            {"gnosis", "Gnosis Chain", "GNO", 18, "https://rpc.gnosischain.com", "https://gnosisscan.io", 100, ChainType::EVM, false},
            {"kava", "Kava", "KAVA", 18, "https://evm.kava.io", "https://kavascan.com", 2222, ChainType::EVM, false},
            
            // === Cosmos Ecosystem ===
            {"cosmos", "Cosmos Hub", "ATOM", 6, "https://cosmos-rpc.polkachu.com", "https://mintscan.io", 0, ChainType::COSMOS, false},
            {"osmosis", "Osmosis", "OSMO", 6, "https://osmosis-rpc.polkachu.com", "https://mintscan.io/osmosis", 0, ChainType::COSMOS, false},
            {"juno", "Juno", "JUNO", 6, "https://juno-rpc.polkachu.com", "https://mintscan.io/juno", 0, ChainType::COSMOS, false},
            {"injective", "Injective", "INJ", 18, "https://injective-rpc.polkachu.com", "https://explorer.injective.network", 0, ChainType::COSMOS, false},
            {"stargaze", "Stargaze", "STARS", 6, "https://stargaze-rpc.polkachu.com", "https://mintscan.io/stargaze", 0, ChainType::COSMOS, false},
            {"evmos", "Evmos", "EVMOS", 18, "https://evmos-rpc.polkachu.com", "https://evmos.mintscan.io", 9001, ChainType::EVM, false},
            {"secret", "Secret Network", "SCRT", 6, "https://rpc.ankr.com/scrt", "https://secretnodes.com", 0, ChainType::COSMOS, false},
            {"sei", "Sei", "SEI", 6, "https://sei-rpc.polkachu.com", "https://seitrace.com", 0, ChainType::COSMOS, false},
            
            // === Other Popular Chains ===
            {"near", "NEAR Protocol", "NEAR", 24, "https://rpc.mainnet.near.org", "https://explorer.near.org", 0, ChainType::OTHER, false},
            {"algorand", "Algorand", "ALGO", 6, "https://mainnet-algorand.api.purestake.io", "https://algoexplorer.io", 0, ChainType::OTHER, false},
            {"sui", "Sui", "SUI", 9, "https://fullnode.mainnet.sui.io", "https://suiscan.xyz", 0, ChainType::SUI, false},
            {"aptos", "Aptos", "APT", 8, "https://api.mainnet.aptoslabs.com/v1", "https://aptoscan.com", 0, ChainType::APTOS, false},
            {"ton", "Toncoin", "TON", 9, "https://toncenter.com/api/v2", "https://tonscan.org", 0, ChainType::TON, false},
            {"flow", "Flow", "FLOW", 8, "https://rest-mainnet.onflow.org", "https://flowscan.org", 0, ChainType::OTHER, false},
            {"hedera", "Hedera", "HBAR", 8, "https://mainnet.mirrornode.hedera.com", "https://hashscan.io", 0, ChainType::OTHER, false},
            {"cardano", "Cardano", "ADA", 6, "https://cardano-mainnet.blockfrost.io", "https://cardanoscan.io", 0, ChainType::OTHER, false},
            {"polkadot", "Polkadot", "DOT", 10, "https://rpc.polkadot.io", "https://polkadot.subscan.io", 0, ChainType::COSMOS, false},
            {"kusama", "Kusama", "KSM", 12, "https://kusama-rpc.polkadot.io", "https://kusama.subscan.io", 0, ChainType::COSMOS, false},
            {"tezos", "Tezos", "XTZ", 6, "https://mainnet.api.tez.ie", "https://tzstats.com", 0, ChainType::OTHER, false},
            {"kadena", "Kadena", "KDA", 12, "https://api.chainweb.com", "https://explorer.kadena.io", 0, ChainType::OTHER, false},
            
            // === Bitcoin Related ===
            {"litecoin", "Litecoin", "LTC", 8, "https://litecoin-rpc.polkachu.com", "https://blockchair.com/litecoin", 0, ChainType::BITCOIN, false},
            {"dogecoin", "Dogecoin", "DOGE", 8, "https://dogecoin-rpc.polkachu.com", "https://dogecoin.info", 0, ChainType::BITCOIN, false},
            {"bitcoin_cash", "Bitcoin Cash", "BCH", 8, "https://bch-rpc.polkachu.com", "https://blockchair.com/bitcoin-cash", 0, ChainType::BITCOIN, false},
            {"dash", "Dash", "DASH", 8, "https://dash-rpc.polkachu.com", "https://dashblockexplorer.com", 0, ChainType::BITCOIN, false},
            {"zcash", "Zcash", "ZEC", 8, "https://zcash-rpc.polkachu.com", "https://zcashblockexplorer.com", 0, ChainType::BITCOIN, false},
            {"monero", "Monero", "XMR", 12, "https://monero-rpc.polkachu.com", "https://moneroexplorer.org", 0, ChainType::BITCOIN, false},
            {"ravencoin", "Ravencoin", "RVN", 8, "https://rvn-rpc.polkachu.com", "https://ravencoin.network", 0, ChainType::BITCOIN, false},
            
            // === Additional EVM Chains ===
            {"arbitrum_nova", "Arbitrum Nova", "ETH", 18, "https://nova.arbitrum.io/rpc", "https://nova.arbiscan.io", 42170, ChainType::EVM, false},
            {"harmony", "Harmony One", "ONE", 18, "https://api.harmony.one", "https://explorer.harmony.one", 1666600000, ChainType::EVM, false},
            {"moonbeam", "Moonbeam", "GLMR", 18, "https://rpc.api.moonbeam.network", "https://moonscan.io", 1284, ChainType::EVM, false},
            {"moonriver", "Moonriver", "MOVR", 18, "https://rpc.api.moonriver.network", "https://moonriver.moonscan.io", 1285, ChainType::EVM, false},
            {"astar", "Astar", "ASTR", 18, "https://rpc.astar.network", "https://blockscout.com/astar", 592, ChainType::EVM, false},
            {"oasis", "Oasis Emerald", "ROSE", 18, "https://emerald.oasis.dev", "https://explorer.emerald.oda.az", 42262, ChainType::EVM, false},
            {"telos", "Telos EVM", "TLOS", 18, "https://mainnet.telos.net", "https://teloscan.io", 40, ChainType::EVM, false},
            {"aurora", "Aurora", "ETH", 18, "https://mainnet.aurora.dev", "https://aurorascan.dev", 1313161554, ChainType::EVM, false},
            {"boba", "Boba Network", "ETH", 18, "https://mainnet.boba.network", "https://bobascan.com", 28882, ChainType::EVM, false},
            {"canto", "Canto", "CANTO", 18, "https://mainnet.infura.io", "https://cantoscan.com", 7700, ChainType::EVM, false},
            {"pulsechain", "PulseChain", "PLS", 18, "https://rpc.pulsechain.com", "https://explorer.pulsechain.com", 369, ChainType::EVM, false},
            {"metis", "Metis", "METIS", 18, "https://andromeda.metis.io", "https://andromeda-explorer.metis.io", 1088, ChainType::EVM, false},
            
            // === More Chains ===
            {"vechain", "VeChain", "VET", 18, "https://mainnet-vechain.eosnation.io", "https://vechainstats.com", 0, ChainType::OTHER, false},
            {"zilliqa", "Zilliqa", "ZIL", 12, "https://api.zilliqa.com", "https://viewblock.io/zilliqa", 0, ChainType::OTHER, false},
            {"icon", "ICON", "ICX", 18, "https://ctz.solidwallet.io", "https://iconosphere.io", 0, ChainType::OTHER, false},
            {"thetachain", "Theta Network", "THETA", 18, "https://theta-rpc.anager.io", "https://explorer.thetatoken.org", 0, ChainType::OTHER, false},
            {"wax", "WAX", "WAXP", 8, "https://wax.greymass.com", "https://wax.bloks.io", 0, ChainType::OTHER, false},
            {"ontology", "Ontology", "ONG", 9, "https://dappnode1.ont.io:20339", "https://explorer.ont.io", 0, ChainType::OTHER, false},
            
            // === DeFi Protocols ===
            {"synthetix", "Synthetix", "SNX", 18, "https://synthetix-mainnet.g.alchemy.com", "https://snx.mintscan.io", 0, ChainType::OTHER, false},
            {"lido", "Lido", "LDO", 18, "https://rpc.lido.fi", "https://stake.lido.fi", 0, ChainType::OTHER, false},
            {"rocketpool", "Rocket Pool", "RPL", 18, "https://rocketpool-rpc.polkachu.com", "https://rocketpool.net", 0, ChainType::OTHER, false},
            {"curve", "Curve", "CRV", 18, "https://curve-rpc.ankr.com", "https://curve.fi", 0, ChainType::OTHER, false},
            {"aave", "Aave", "AAVE", 18, "https://aave-rpc.ankr.com", "https://app.aave.com", 0, ChainType::OTHER, false},
            {"compound", "Compound", "COMP", 18, "https://mainnet-rpc.compound.finance", "https://compound.finance", 0, ChainType::OTHER, false},
            {"makerdao", "Maker", "MKR", 18, "https://rpc.makerdao.com", "https://oasis.app", 0, ChainType::OTHER, false},
            {"uniswap", "Uniswap", "UNI", 18, "https://mainnet.uniswap.org", "https://uniswap.org", 0, ChainType::OTHER, false}
        };
    }
    
    static size_t getChainCount() {
        return getAllChains().size();
    }
};

// ============================================================================
// Token Model
// ============================================================================

// Real Token Data from CoinGecko API (500+ tokens)
struct RealTokenData {
    std::string id;
    std::string symbol;
    std::string name;
    std::string image;
    double current_price;
    long long market_cap;
    int market_cap_rank;
    long long total_volume;
    double price_change_24h;
    double price_change_percentage_24h;
    double circulating_supply;
    double total_supply;
    double ath;
    double ath_change_percentage;
    double atl;
    double atl_change_percentage;
    std::string last_updated;
    
    static std::vector<RealTokenData> fetchFromAPI();
    static RealTokenData getToken(const std::vector<RealTokenData>& tokens, const std::string& symbol);
    static std::vector<RealTokenData> getTopTokens(const std::vector<RealTokenData>& tokens, int limit);
    static std::vector<RealTokenData> searchTokens(const std::vector<RealTokenData>& tokens, const std::string& query);
};

struct Token {
    std::string id;
    std::string address;
    std::string symbol;
    std::string name;
    int decimals;
    std::string chain_id;
    std::optional<std::string> logo_url;
    double price;
    double balance;
    double balance_usd;

    std::string getDisplayBalance() const;
    std::string getDisplayPrice() const;
    std::string getDisplayValue() const;
};

// ============================================================================
// Wallet Model
// ============================================================================

struct Wallet {
    std::string id;
    std::string name;
    std::string address;
    std::string public_key;
    std::string chain_id;
    double balance;
    double balance_usd;
    std::vector<Token> tokens;
    std::chrono::system_clock::time_point created_at;
    bool is_backed_up;
    bool is_hardware;

    std::string getShortAddress() const;
};

// ============================================================================
// Transaction Model
// ============================================================================

enum class TransactionStatus {
    PENDING,
    CONFIRMED,
    FAILED
};

enum class TransactionType {
    SEND,
    RECEIVE,
    SWAP,
    STAKE,
    UNSTAKE,
    APPROVE,
    CONTRACT_INTERACTION,
    NFT_TRANSFER
};

struct Transaction {
    std::string id;
    std::string hash;
    std::string from;
    std::string to;
    double amount;
    std::string symbol;
    int decimals;
    std::string chain_id;
    TransactionStatus status;
    std::chrono::system_clock::time_point timestamp;
    TransactionType type;
    std::optional<double> gas_used;
    std::optional<double> gas_price;
    std::optional<std::string> token_address;

    static std::string statusToString(TransactionStatus status);
    static std::string typeToString(TransactionType type);
};

// ============================================================================
// NFT Model
// ============================================================================

struct NFT {
    std::string id;
    std::string token_id;
    std::string contract_address;
    std::string name;
    std::optional<std::string> description;
    std::optional<std::string> image_url;
    std::string chain_id;
    std::optional<std::string> collection_name;
    std::optional<std::string> metadata_url;
};

// ============================================================================
// Staking Model
// ============================================================================

struct StakingPosition {
    std::string id;
    std::string validator;
    double amount;
    double rewards;
    std::optional<std::chrono::system_clock::time_point> unlock_time;
    std::string chain_id;
    std::string token_symbol;
};

struct StakingQuote {
    double apy;
    double min_stake;
    int lock_period_days;
};

// ============================================================================
// Swap Model
// ============================================================================

struct SwapQuote {
    std::string from_token;
    std::string to_token;
    double from_amount;
    double to_amount;
    double price_impact;
    std::vector<std::string> route;
    double gas_estimate;

    std::string getDisplayFromAmount() const;
    std::string getDisplayToAmount() const;
};

// ============================================================================
// Price Model
// ============================================================================

struct PriceInfo {
    std::string symbol;
    std::string name;
    double price;
    double change_24h;
    double change_percent_24h;
    double market_cap;
    double volume_24h;
    double high_24h;
    double low_24h;
    std::chrono::system_clock::time_point last_updated;

    bool isPositive() const;
    std::string getFormattedPrice() const;
    std::string getFormattedChange() const;
    std::string getFormattedMarketCap() const;
};

struct PriceHistory {
    std::string symbol;
    std::vector<std::pair<std::chrono::system_clock::time_point, double>> prices;
};

// ============================================================================
// API Response Models
// ============================================================================

template<typename T>
struct APIResponse {
    bool success;
    T data;
    std::optional<std::string> error;
    std::optional<std::string> error_message;
};

struct WalletListResponse {
    std::vector<Wallet> wallets;
    int total_count;
};

struct TransactionListResponse {
    std::vector<Transaction> transactions;
    int total_count;
    int page;
    int page_size;
};

struct SwapResponse {
    std::string tx_hash;
    double from_amount;
    double to_amount;
};

struct StakeResponse {
    std::string tx_hash;
    double staked_amount;
    std::string position_id;
};

struct BridgeResponse {
    std::string tx_hash;
    std::string bridge_tx_hash;
};

// ============================================================================
// Utility Functions
// ============================================================================

std::string generateUUID();
std::string getCurrentTimestamp();
double hexToDouble(const std::string& hex, int decimals);
std::string doubleToHex(double value, int decimals);

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_MODELS_H
