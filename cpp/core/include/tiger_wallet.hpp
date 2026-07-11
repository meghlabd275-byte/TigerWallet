#ifndef TIGER_WALLET_HPP
#define TIGER_WALLET_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <array>
#include <cstdint>

namespace tiger {

// Forward declarations
class Transaction;
class KeyPair;
class ChainConfig;

// Chain IDs for supported blockchains
enum class ChainType : uint32_t {
    ETHEREUM = 1,
    POLYGON = 137,
    ARBITRUM = 42161,
    OPTIMISM = 10,
    BASE = 8453,
    AVALANCHE = 43114,
    BINANCE_SMART_CHAIN = 56,
    TRON = 728126428,
    SOLANA = 101,
    PULSECHAIN = 369,
    APTOS = 637,
    TON = -239,
    COSMOS = 118,
    BITCOIN = 0,
    LITECOIN = 2,
    DOGECOIN = 3,
    RIPPLE = 144,
    CARDANO = 3009,
    POLKADOT = 0,
    NEAR = 1313161554,
    ALGORAND = 4163,
    NEON = 245022934,
    METIS = 1088,
    KAVA = 2222,
    CELO = 42220,
    GNOSIS = 100,
    FANTOM = 250,
    HARMONY = 1666600000,
    CRONOS = 25,
    AURORA = 1313161554,
    REI = 47856,
    TERRA = 0,
    OASIS = 42262,
    VELAS = 106,
    WAX = 23001,
    IOTEX = 4689,
    KLAYTN = 8217,
    ONTOLOGY = 2000198,
    TOMOCHAIN = 88,
    RSK = 30,
    ELASTOS = 20,
    THETA = 361,
    SOLAR = 501,
    RINKEBY = 4,
    GOERLI = 5,
    SEPOLIA = 11155111,
    BEVM = 1502,
    BITGERT = 11011,
    VICTION = 1907,
    OMAX = 3110,
    HEDERA = 295,
    MININGTUNNEL = 21000000,
    ORE = 1399811149,
    PI = 314159,
    CONFLUX = 1030,
    CUBE = 1818,
    BITCOIN_CASH = 145,
    ZCASH = 133,
    MONERO = 128,
    EOS = 17777,
    STEEM = 2017,
    TRON_SHASTA = 2494104990,
    TRON_NILE = 3448148188
};

// Token standards
enum class TokenStandard {
    ERC20,
    ERC721,
    ERC1155,
    SPL_TOKEN,
    TRC20,
    BEP20,
    ARC20,
    NATIVE
};

// Transaction status
enum class TxStatus {
    PENDING,
    CONFIRMED,
    FAILED,
    CANCELLED
};

// Wallet type
enum class WalletType {
    EVM,
    SOLANA,
    TRON,
    BITCOIN,
    COSMOS,
    MULTI_CHAIN
};

// Key derivation path structure
struct DerivationPath {
    uint32_t purpose;
    uint32_t coin_type;
    uint32_t account;
    uint32_t change;
    uint32_t address_index;
    
    DerivationPath(uint32_t p = 44, uint32_t c = 60, uint32_t a = 0, uint32_t ch = 0, uint32_t i = 0)
        : purpose(p), coin_type(c), account(a), change(ch), address_index(i) {}
    
    std::string to_string() const;
};

// Chain configuration
struct ChainConfig {
    ChainType type;
    std::string name;
    std::string symbol;
    uint32_t chain_id;
    std::string rpc_url;
    std::string explorer_url;
    std::string icon_url;
    uint8_t decimals;
    bool is_active;
    uint64_t gas_limit;
    uint64_t min_confirmations;
    
    ChainConfig() : type(ChainType::ETHEREUM), chain_id(1), decimals(18), 
                   is_active(true), gas_limit(21000), min_confirmations(12) {}
};

// Token configuration
struct TokenConfig {
    std::string address;
    std::string symbol;
    std::string name;
    uint8_t decimals;
    ChainType chain;
    TokenStandard standard;
    std::string explorer_url;
    bool is_active;
    std::string icon_url;
    uint256_t total_supply;
    bool is_listed;
    
    TokenConfig() : decimals(18), chain(ChainType::ETHEREUM), 
                   is_active(true), total_supply(0), is_listed(true) {}
};

// Address information
struct AddressInfo {
    std::string address;
    ChainType chain;
    WalletType wallet_type;
    std::string derivation_path;
    bool is_derived;
    uint64_t balance;
    std::vector<TokenBalance> tokens;
    
    AddressInfo() : chain(ChainType::ETHEREUM), wallet_type(WalletType::EVM),
                   is_derived(false), balance(0) {}
};

// Token balance
struct TokenBalance {
    std::string contract_address;
    std::string symbol;
    std::string name;
    uint256_t balance;
    uint8_t decimals;
    uint256_t usd_value;
    
    TokenBalance() : balance(0), decimals(18), usd_value(0) {}
};

// Transaction data
struct TransactionData {
    std::string from;
    std::string to;
    uint256_t value;
    std::string data;
    uint64_t nonce;
    uint64_t gas_price;
    uint64_t gas_limit;
    ChainType chain;
    uint64_t chain_id;
    
    TransactionData() : value(0), nonce(0), gas_price(0), gas_limit(21000),
                       chain(ChainType::ETHEREUM), chain_id(1) {}
};

// Signed transaction
struct SignedTransaction {
    std::vector<uint8_t> raw_tx;
    std::string tx_hash;
    std::string signature;
    TransactionData data;
    TxStatus status;
    uint64_t timestamp;
    uint64_t confirmations;
    
    SignedTransaction() : status(TxStatus::PENDING), timestamp(0), confirmations(0) {}
};

// Swap quote
struct SwapQuote {
    std::string from_token;
    std::string to_token;
    uint256_t from_amount;
    uint256_t to_amount;
    uint256_t price_impact;
    uint256_t gas_estimate;
    std::string route;
    std::string dex;
    uint64_t expires_at;
    
    SwapQuote() : from_amount(0), to_amount(0), price_impact(0), 
                  gas_estimate(0), expires_at(0) {}
};

// Perpetual position
struct PerpetualPosition {
    std::string position_id;
    std::string trader;
    std::string collateral_token;
    std::string index_token;
    bool is_long;
    uint256_t size;
    uint256_t collateral;
    uint256_t entry_price;
    uint256_t mark_price;
    uint256_t unrealized_pnl;
    uint64_t leverage;
    uint64_t timestamp;
    
    PerpetualPosition() : is_long(false), size(0), collateral(0), 
                        entry_price(0), mark_price(0), unrealized_pnl(0),
                        leverage(1), timestamp(0) {}
};

// Copy trading signal
struct TradingSignal {
    std::string signal_id;
    std::string trader_address;
    std::string token_a;
    std::string token_b;
    std::string action; // BUY, SELL
    uint256_t amount;
    uint256_t price;
    uint64_t timestamp;
    double success_rate;
    
    TradingSignal() : amount(0), price(0), timestamp(0), success_rate(0.0) {}
};

// Wallet error codes
enum class WalletError {
    SUCCESS = 0,
    INVALID_ADDRESS = 1001,
    INVALID_PARAMS = 1002,
    INSUFFICIENT_BALANCE = 1003,
    TRANSACTION_FAILED = 1004,
    SIGNATURE_INVALID = 1005,
    NETWORK_ERROR = 1006,
    CHAIN_NOT_SUPPORTED = 1007,
    TOKEN_NOT_FOUND = 1008,
    RPC_ERROR = 1009,
    TIMEOUT = 1010,
    WALLET_LOCKED = 1011,
    KEY_NOT_FOUND = 1012,
    ENCRYPTION_FAILED = 1013,
    DECRYPTION_FAILED = 1014,
    MNEMONIC_INVALID = 1015,
    DERIVATION_FAILED = 1016,
    SWAP_FAILED = 1017,
    PERPETUAL_ERROR = 1018,
    COPY_TRADING_ERROR = 1019,
    UNKNOWN_ERROR = 9999
};

// Result wrapper
template<typename T>
struct Result {
    bool success;
    T data;
    WalletError error;
    std::string error_message;
    
    Result() : success(false), error(WalletError::UNKNOWN_ERROR) {}
    
    static Result<T> ok(T&& d) {
        Result r;
        r.success = true;
        r.data = std::move(d);
        return r;
    }
    
    static Result<T> err(WalletError e, const std::string& msg) {
        Result r;
        r.success = false;
        r.error = e;
        r.error_message = msg;
        return r;
    }
};

// Callback types
using ProgressCallback = std::function<void(int, const std::string&)>;
using ConfirmationCallback = std::function<bool(const std::string&, uint64_t)>;

// Main wallet class
class TigerWallet {
public:
    TigerWallet();
    ~TigerWallet();
    
    // Wallet creation and recovery
    Result<std::vector<std::string>> generate_mnemonic();
    Result<void> recover_from_mnemonic(const std::string& mnemonic, const std::string& password = "");
    Result<void> recover_from_private_key(const std::string& private_key, ChainType chain);
    
    // Address management
    Result<std::string> get_address(ChainType chain, uint32_t index = 0);
    Result<std::vector<AddressInfo>> get_all_addresses();
    Result<void> derive_addresses(ChainType chain, uint32_t count);
    
    // Balance operations
    Result<uint256_t> get_native_balance(ChainType chain, const std::string& address);
    Result<std::vector<TokenBalance>> get_token_balances(ChainType chain, const std::string& address);
    Result<AddressInfo> get_full_balance(ChainType chain, const std::string& address);
    
    // Transaction operations
    Result<SignedTransaction> create_transaction(const TransactionData& tx_data);
    Result<SignedTransaction> sign_transaction(const TransactionData& tx_data);
    Result<std::string> broadcast_transaction(const SignedTransaction& signed_tx);
    Result<TxStatus> get_transaction_status(const std::string& tx_hash, ChainType chain);
    Result<std::vector<SignedTransaction>> get_transaction_history(
        const std::string& address, 
        ChainType chain,
        uint32_t limit = 50,
        uint32_t offset = 0
    );
    
    // Swap operations
    Result<SwapQuote> get_swap_quote(
        const std::string& from_token,
        const std::string& to_token,
        uint256_t amount,
        ChainType chain
    );
    Result<SignedTransaction> execute_swap(
        const SwapQuote& quote,
        const std::string& from_address,
        uint256_t slippage_tolerance
    );
    
    // Perpetual trading
    Result<PerpetualPosition> open_perpetual_position(
        const std::string& collateral_token,
        const std::string& index_token,
        bool is_long,
        uint256_t collateral_amount,
        uint64_t leverage
    );
    Result<PerpetualPosition> close_perpetual_position(const std::string& position_id);
    Result<PerpetualPosition> update_perpetual_position(
        const std::string& position_id,
        uint256_t new_collateral,
        uint64_t new_leverage
    );
    Result<std::vector<PerpetualPosition>> get_perpetual_positions(const std::string& trader);
    
    // Copy trading
    Result<void> follow_trader(const std::string& trader_address, uint64_t max_copy_amount);
    Result<void> unfollow_trader(const std::string& trader_address);
    Result<std::vector<TradingSignal>> get_trading_signals(
        const std::string& trader_address,
        uint32_t limit = 100
    );
    Result<void> execute_copy_trade(const TradingSignal& signal, uint256_t amount);
    Result<std::vector<TradingSignal>> get_top_traders(uint32_t limit = 50);
    
    // Chain and token management
    Result<void> add_custom_chain(const ChainConfig& config);
    Result<void> remove_chain(ChainType chain);
    Result<std::vector<ChainConfig>> get_supported_chains();
    Result<ChainConfig> get_chain_config(ChainType chain);
    Result<void> add_custom_token(const TokenConfig& config);
    Result<void> remove_token(const std::string& address, ChainType chain);
    Result<std::vector<TokenConfig>> get_supported_tokens(ChainType chain);
    
    // Admin operations
    Result<void> add_blockchain_admin(ChainType chain, const std::string& admin_address);
    Result<void> remove_blockchain_admin(ChainType chain, const std::string& admin_address);
    Result<void> add_token_admin(const std::string& token_address, ChainType chain, const std::string& admin_address);
    Result<void> remove_token_admin(const std::string& token_address, ChainType chain, const std::string& admin_address);
    Result<bool> is_super_admin(const std::string& address);
    Result<void> set_super_admin(const std::string& address, bool is_admin);
    
    // Settings
    void set_rpc_url(ChainType chain, const std::string& url);
    void set_explorer_url(ChainType chain, const std::string& url);
    void set_gas_limit(ChainType chain, uint64_t limit);
    void set_slippage_tolerance(uint8_t tolerance);
    void set_confirmation_callback(ConfirmationCallback callback);
    
private:
    class Impl;
    std::unique_ptr<Impl> pImpl;
};

} // namespace tiger

#endif // TIGER_WALLET_HPP
