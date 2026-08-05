/**
 * MasterWalletService - C++ Implementation
 * Complete wallet management for Master Wallet
 * Features: HD Wallet, Multi-chain, Token Management, Transaction Signing
 * Ultra-low latency with in-memory caching
 */

#ifndef MASTER_WALLET_SERVICE_HPP
#define MASTER_WALLET_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <set>
#include <mutex>
#include <memory>
#include <atomic>
#include <functional>
#include <optional>
#include <chrono>
#include <thread>
#include <future>
#include <unordered_map>
#include <shared_mutex>

// Forward declarations
namespace tiger {
namespace master {

// Types
using WalletID = std::string;
using UserID = std::string;
using ChainID = uint64_t;
using TokenAddress = std::string;

struct WalletData {
    WalletID id;
    std::string address;
    std::string publicKey;
    std::string encryptedMnemonic;
    std::vector<ChainID> supportedChains;
    uint64_t createdAt;
    bool isActive;
};

struct ChainConfig {
    ChainID id;
    std::string name;
    std::string symbol;
    std::string rpcUrl;
    std::string explorerUrl;
    uint8_t decimals;
    bool isEVM;
};

struct TokenConfig {
    TokenAddress address;
    std::string symbol;
    std::string name;
    uint8_t decimals;
    ChainID chainId;
};

struct TransactionRequest {
    WalletID fromWallet;
    std::string toAddress;
    std::string amount;
    TokenAddress token;
    ChainID chainId;
    uint64_t gasLimit;
    uint64_t gasPrice;
};

struct TransactionResult {
    std::string txHash;
    bool success;
    std::string error;
    uint64_t timestamp;
};

struct BalanceResult {
    std::string balance;
    std::string symbol;
    uint8_t decimals;
    bool success;
    std::string error;
};

class MasterWalletService {
public:
    static MasterWalletService& getInstance();
    
    // Wallet Management
    WalletData generateWallet(const std::string& password, ChainID chainId = 1);
    std::optional<WalletData> getWallet(const WalletID& walletId) const;
    std::vector<WalletData> getAllWallets() const;
    bool deleteWallet(const WalletID& walletId);
    bool importWallet(const std::string& mnemonic, const std::string& password);
    
    // Multi-chain Support
    void addChain(const ChainConfig& config);
    void removeChain(ChainID chainId);
    std::optional<ChainConfig> getChainConfig(ChainID chainId) const;
    std::vector<ChainConfig> getAllChains() const;
    
    // Token Management
    void addToken(const TokenConfig& token);
    void removeToken(const TokenAddress& token, ChainID chainId);
    std::optional<TokenConfig> getToken(const TokenAddress& token, ChainID chainId) const;
    std::vector<TokenConfig> getAllTokens() const;
    
    // Balance Operations
    BalanceResult getBalance(const WalletID& walletId, ChainID chainId, const TokenAddress& token = "");
    std::map<std::string, BalanceResult> getAllBalances(const WalletID& walletId);
    
    // Transaction Operations
    TransactionResult createTransaction(const TransactionRequest& request);
    TransactionResult signAndBroadcast(const TransactionRequest& request);
    std::string signMessage(const WalletID& walletId, const std::string& message);
    bool verifySignature(const std::string& message, const std::string& signature, const std::string& address);
    
    // HD Wallet Operations
    std::string deriveAddress(const WalletID& walletId, ChainID chainId, uint32_t index);
    std::string derivePublicKey(const WalletID& walletId, ChainID chainId, uint32_t index);
    
    // User Wallet Management
    void registerUserWallet(const UserID& userId, const WalletID& walletId);
    void unregisterUserWallet(const UserID& userId);
    std::optional<WalletID> getUserWallet(const UserID& userId) const;
    std::vector<UserID> getUsersForWallet(const WalletID& walletId) const;
    
    // Auto-sign Configuration
    void setAutoSignEnabled(bool enabled);
    bool isAutoSignEnabled() const;
    void setAutoSignLimit(uint64_t limitInWei);
    uint64_t getAutoSignLimit() const;
    
    // Cache Management
    void clearCache();
    void setCacheTTL(uint64_t ttlMs);
    
    // Encryption
    std::string encryptData(const std::string& data, const std::string& key);
    std::string decryptData(const std::string& encryptedData, const std::string& key);
    
    // Status
    bool isHealthy() const;
    std::string getVersion() const;

private:
    MasterWalletService();
    ~MasterWalletService();
    MasterWalletService(const MasterWalletService&) = delete;
    MasterWalletService& operator=(const MasterWalletService&) = delete;
    
    // Internal methods
    std::string generateWalletId();
    std::string hashPrivateKey(const std::string& privateKey);
    std::string computeAddress(const std::string& publicKey);
    std::string computeMnemonicHash(const std::string& mnemonic);
    
    bool loadWalletsFromStorage();
    bool saveWalletsToStorage();
    bool loadChainsFromStorage();
    bool loadTokensFromStorage();
    
    // Thread-safe data access
    mutable std::shared_mutex walletMutex_;
    mutable std::shared_mutex chainMutex_;
    mutable std::shared_mutex tokenMutex_;
    mutable std::shared_mutex cacheMutex_;
    
    // Storage
    std::map<WalletID, WalletData> wallets_;
    std::map<ChainID, ChainConfig> chains_;
    std::map<std::pair<TokenAddress, ChainID>, TokenConfig> tokens_;
    std::map<UserID, WalletID> userWallets_;
    std::map<WalletID, std::set<UserID>> walletUsers_;
    
    // Cache
    std::map<WalletID, std::pair<BalanceResult, uint64_t>> balanceCache_;
    uint64_t cacheTTLMs_ = 30000; // 30 seconds
    
    // Configuration
    bool autoSignEnabled_ = true;
    uint64_t autoSignLimit_ = 1000000000000000000ULL; // 1 ETH
    
    // Security
    std::string masterEncryptionKey_;
    
    // Status
    std::atomic<bool> isHealthy_{true};
    std::string version_ = "1.0.0";
    
    // Supported chains (default)
    static constexpr ChainID CHAIN_ETHEREUM = 1;
    static constexpr ChainID CHAIN_BSC = 56;
    static constexpr ChainID CHAIN_POLYGON = 137;
    static constexpr ChainID CHAIN_ARBITRUM = 42161;
    static constexpr ChainID CHAIN_OPTIMISM = 10;
    static constexpr ChainID CHAIN_AVALANCHE = 43114;
};

// Inline implementation
inline MasterWalletService& MasterWalletService::getInstance() {
    static MasterWalletService instance;
    return instance;
}

} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_SERVICE_HPP
