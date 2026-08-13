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
    std::string data; // optional calldata for the create-transaction-record endpoint
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

// Gas price snapshot from GET /api/v1/gas?chain_id=N (real backend oracle).
struct GasEstimate {
    std::string gasPrice;     // base fee / legacy gas price (wei, hex or decimal)
    std::string maxFee;        // EIP-1559 max fee (wei)
    std::string priorityFee;   // EIP-1559 priority fee (wei)
    bool success;
    std::string error;
};

// Market price from GET /api/v1/price?coin_id=... (real backend oracle).
struct PriceQuote {
    double usd;
    double usd24hChange;
    bool success;
    std::string error;
};

// A single backend transaction record (GET /api/v1/master-wallet/:id/transactions).
struct TransactionRecord {
    std::string hash;
    std::string from;
    std::string to;
    std::string amount;
    std::string token;
    std::string status;
    std::string timestamp;
    std::string blockNumber;
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
    // createTransaction POSTs a transaction RECORD to /transactions (pending),
    // distinct from signAndBroadcast which signs+broadcasts via /sign.
    TransactionResult createTransaction(const TransactionRequest& request);
    TransactionResult signAndBroadcast(const TransactionRequest& request);
    // Approve / reject a pending transaction record created via createTransaction.
    TransactionResult approveTransaction(const WalletID& masterId, const std::string& txId);
    TransactionResult rejectTransaction(const WalletID& masterId, const std::string& txId);
    std::string signMessage(const WalletID& walletId, const std::string& message);
    bool verifySignature(const std::string& message, const std::string& signature, const std::string& address);

    // Real backend queries (canonical API contract, backend :8450).
    // These return data fetched from the backend; none fabricate values. On
    // error the result carries success=false / an error string, or throws.
    std::vector<TransactionRecord> getTransactions(const WalletID& walletId);
    GasEstimate getGas(ChainID chainId);
    PriceQuote getPrice(const std::string& coinId);
    std::vector<ChainConfig> fetchChains(); // force-refresh from GET /api/v1/chains

    // Sub-wallets
    std::string getSubWallets(const WalletID& walletId);          // raw backend JSON
    BalanceResult getSubWalletBalance(const WalletID& walletId, const std::string& subId);
    TransactionResult transferSubWallet(const WalletID& walletId, const std::string& subId,
                                         const std::string& to, const std::string& amount,
                                         const std::string& password, const std::string& token = "");

    // Policies / Fees / Auto-sign / Users (raw backend JSON passthrough)
    std::string getPolicies(const WalletID& walletId);
    std::string createPolicy(const WalletID& walletId, const std::string& body);
    std::string updatePolicy(const WalletID& walletId, const std::string& policyId, const std::string& body);
    bool deletePolicy(const WalletID& walletId, const std::string& policyId);

    std::string getFees(const WalletID& walletId);
    std::string createFee(const WalletID& walletId, const std::string& body);
    bool deleteFee(const WalletID& walletId, const std::string& feeId);

    std::string getAutoSignRules(const WalletID& walletId);
    std::string createAutoSignRule(const WalletID& walletId, const std::string& body);
    bool deleteAutoSignRule(const WalletID& walletId, const std::string& ruleId);

    std::string getUsers(const WalletID& walletId);
    std::string createUser(const WalletID& walletId, const std::string& body);
    bool deleteUser(const WalletID& walletId, const std::string& userId);

    // Audit + Analytics
    std::string getAudit(const WalletID& walletId);
    std::string getAnalyticsVolume(const WalletID& walletId);
    std::string getAnalyticsTransactions(const WalletID& walletId);
    std::string getAnalyticsWallets(const WalletID& walletId);

    // Notifications + Webhooks
    std::string getNotifications(const WalletID& walletId);
    std::string createNotification(const WalletID& walletId, const std::string& body);
    std::string getWebhooks(const WalletID& walletId);
    std::string createWebhook(const WalletID& walletId, const std::string& body);
    bool deleteWebhook(const WalletID& walletId, const std::string& webhookId);

    // Treasury (real balances via backend)
    std::string getTreasury(const WalletID& walletId);
    std::string getTreasuryTransactions(const WalletID& walletId);
    TransactionResult treasuryTransfer(const WalletID& walletId, const std::string& to,
                                       const std::string& amount, const std::string& password);
    TransactionResult treasurySweep(const WalletID& walletId, const std::string& to,
                                    const std::string& password);

    // Multisig
    std::string getMultisigWallets(const WalletID& walletId);
    std::string createMultisigWallet(const WalletID& walletId, const std::string& body);
    std::string getMultisigTransactions(const WalletID& walletId, const std::string& multisigId);
    std::string createMultisigTransaction(const WalletID& walletId, const std::string& multisigId,
                                          const std::string& body);
    std::string signMultisigTransaction(const WalletID& walletId, const std::string& txId,
                                        const std::string& body);
    std::string executeMultisigTransaction(const WalletID& walletId, const std::string& txId,
                                          const std::string& body);

    // ==================== UserWallet Management (fetchers) ====================
    // Real HTTP passthroughs to /api/v1/master-wallet/:id/* sub-resources. All
    // carry Bearer JWT auth via the shared APIClient and return raw backend JSON.

    // EVM chain management
    std::string listUserEVMChains(const WalletID& walletId);
    std::string addUserEVMChain(const WalletID& walletId, const std::string& body);
    std::string updateUserEVMChain(const WalletID& walletId, const std::string& chainId,
                                   const std::string& body);
    bool removeUserEVMChain(const WalletID& walletId, const std::string& chainId);

    // Non-EVM chain management
    std::string listUserNonEVMChains(const WalletID& walletId);
    std::string addUserNonEVMChain(const WalletID& walletId, const std::string& body);
    std::string updateUserNonEVMChain(const WalletID& walletId, const std::string& chainId,
                                      const std::string& body);
    bool removeUserNonEVMChain(const WalletID& walletId, const std::string& chainId);

    // Token management
    std::string listUserTokens(const WalletID& walletId,
                               const std::optional<std::string>& chainId = std::nullopt);
    std::string addUserToken(const WalletID& walletId, const std::string& body);
    std::string updateUserToken(const WalletID& walletId, const std::string& tokenId,
                                const std::string& body);
    bool removeUserToken(const WalletID& walletId, const std::string& tokenId);

    // Address derivation
    std::string deriveUserAddress(const WalletID& walletId, const std::string& body);
    std::string listUserWalletAddresses(const WalletID& walletId);

    // Auto-sign
    std::string autoSignTransaction(const WalletID& walletId, const std::string& body);
    std::string listAutoSignLogs(const WalletID& walletId);

    // Feature flags
    std::string listFeatureFlags(const WalletID& walletId);
    std::string addFeatureFlag(const WalletID& walletId, const std::string& body);
    std::string updateFeatureFlag(const WalletID& walletId, const std::string& flagId,
                                  const std::string& body);
    bool removeFeatureFlag(const WalletID& walletId, const std::string& flagId);

    // Public endpoint helpers
    std::string getTransactionHistory(const std::string& address, ChainID chainId);
    
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
    
    // Storage (mutable so const accessors may refresh the in-memory cache)
    mutable std::map<WalletID, WalletData> wallets_;
    mutable std::map<ChainID, ChainConfig> chains_;
    mutable std::map<std::pair<TokenAddress, ChainID>, TokenConfig> tokens_;
    mutable std::map<UserID, WalletID> userWallets_;
    mutable std::map<WalletID, std::set<UserID>> walletUsers_;
    
    // Cache
    mutable std::map<WalletID, std::pair<BalanceResult, uint64_t>> balanceCache_;
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
