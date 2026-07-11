#ifndef TIGERWALLET_WALLET_H
#define TIGERWALLET_WALLET_H

#include <string>
#include <vector>
#include <optional>
#include <array>
#include <memory>
#include <map>

namespace tigerwallet {

enum class BlockchainType {
    EVM, SOLANA, BITCOIN, TRON, COSMOS, APTOS, SUI, NEAR, ALGORAND, TEZOS, CARDANO, STELLAR, XRPL, CUSTOM
};

enum class WalletType {
    USER, MASTER, WHITELABEL
};

struct ChainConfig {
    std::string id;
    std::string name;
    std::string symbol;
    int64_t chain_id;
    BlockchainType type;
    std::string rpc_url;
    std::string explorer_url;
    std::string logo_url;
    uint8_t decimals;
    std::string gas_token;
    uint32_t avg_block_time_ms;
    uint64_t max_gas_price;
    bool supports_eip1559;
};

struct TokenConfig {
    std::string id;
    std::string blockchain_id;
    std::string symbol;
    std::string name;
    uint8_t decimals;
    std::optional<std::string> contract_address;
    std::string type;
    std::string total_supply;
    std::string logo_url;
    bool is_active;
    bool is_popular;
    double price_usd;
    int64_t market_cap;
    int64_t volume_24h;
};

struct Wallet {
    std::string id;
    std::string user_id;
    WalletType type;
    std::string address;
    std::string blockchain_id;
    std::vector<uint8_t> public_key;
    std::vector<uint8_t> encrypted_private_key;
    std::string derivation_path;
    std::string created_at;
    std::string updated_at;
    bool is_active;
    std::optional<std::string> label;
};

struct Transaction {
    std::string id;
    std::string wallet_id;
    std::string blockchain_id;
    std::string type;
    std::string status;
    std::string from;
    std::string to;
    std::string token_symbol;
    std::optional<std::string> token_address;
    std::string amount;
    double amount_usd;
    std::string fee;
    double fee_usd;
    std::optional<std::string> gas_price;
    std::optional<std::string> gas_used;
    std::optional<uint64_t> nonce;
    std::string hash;
    std::optional<uint64_t> block_number;
    std::string timestamp;
    std::optional<std::string> error;
};

struct Balance {
    std::string wallet_id;
    std::string token_symbol;
    std::string balance;
    double balance_usd;
    std::string frozen_balance;
    std::string available_balance;
    std::string last_updated;
};

class WalletManager {
public:
    WalletManager();
    ~WalletManager();
    void initialize(const std::vector<ChainConfig>& chains);
    std::optional<Wallet> create_wallet(const std::string& user_id, WalletType type, const std::string& blockchain_id, const std::string& password);
    std::optional<Wallet> import_wallet(const std::string& user_id, WalletType type, const std::string& mnemonic, const std::string& blockchain_id, const std::string& password);
    std::optional<Wallet> import_from_private_key(const std::string& user_id, WalletType type, const std::vector<uint8_t>& private_key, const std::string& blockchain_id, const std::string& password);
    std::optional<Wallet> get_wallet(const std::string& wallet_id);
    std::optional<Wallet> get_wallet_by_address(const std::string& address);
    std::vector<Wallet> get_user_wallets(const std::string& user_id);
    bool delete_wallet(const std::string& wallet_id, const std::string& password);
    std::optional<Balance> get_balance(const std::string& wallet_id, const std::string& token_symbol);
    std::map<std::string, Balance> get_all_balances(const std::string& wallet_id);
    std::optional<std::vector<uint8_t>> sign_transaction(const std::string& wallet_id, const std::vector<uint8_t>& tx_data, const std::string& password);
    std::string get_address(const std::string& wallet_id);
    std::optional<std::string> export_wallet(const std::string& wallet_id, const std::string& password);
    bool verify_password(const std::string& wallet_id, const std::string& password);
    std::vector<ChainConfig> get_supported_chains() const;
    bool add_chain(const ChainConfig& config);
    bool update_chain(const std::string& chain_id, const ChainConfig& config);
    bool delete_chain(const std::string& chain_id);
    bool add_token(const TokenConfig& config);
    bool update_token(const std::string& token_id, const TokenConfig& config);
    bool delete_token(const std::string& token_id);
private:
    class Impl;
    std::unique_ptr<Impl> pImpl;
};

} // namespace tigerwallet

#endif
