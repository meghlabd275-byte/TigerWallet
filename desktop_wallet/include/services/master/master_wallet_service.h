/**
 * TigerWallet Desktop - Master Wallet Service
 * Complete master wallet implementation with full functionality
 */

#ifndef TIGER_WALLET_MASTER_SERVICE_H
#define TIGER_WALLET_MASTER_SERVICE_H

#include <memory>
#include <string>
#include <vector>
#include <map>
#include <functional>
#include <future>
#include <chrono>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

// ============================================================================
// Enums
// ============================================================================

enum class MasterWalletType {
    HOT,
    COLD,
    OPERATIONS
};

enum class MasterTransactionType {
    DEPOSIT,
    WITHDRAWAL,
    TRANSFER,
    SWAP,
    FEE,
    AIRDROP
};

enum class MasterTransactionStatus {
    PENDING,
    CONFIRMED,
    FAILED
};

enum class FeeType {
    WITHDRAWAL,
    SWAP,
    TRANSACTION,
    LIQUIDITY,
    AIRDROP
};

// ============================================================================
// Models
// ============================================================================

struct MasterWallet {
    std::string id;
    std::string name;
    MasterWalletType type;
    std::string blockchain;
    std::string address;
    std::string public_key;
    double balance;
    bool is_active;
    bool auto_refill;
    std::string refill_threshold;
    std::string refill_amount;
    std::chrono::system_clock::time_point created_at;

    double getBalanceUSD() const;
};

struct MasterTransaction {
    std::string id;
    std::string wallet_id;
    MasterTransactionType type;
    std::string blockchain;
    std::string from_address;
    std::string to_address;
    double amount;
    double fee;
    MasterTransactionStatus status;
    std::string hash;
    std::chrono::system_clock::time_point timestamp;
};

// ============================================================================
// Master Wallet Service
// ============================================================================

class MasterWalletService {
public:
    static std::shared_ptr<MasterWalletService> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // Wallet Management
    std::future<MasterWallet> createMasterWallet(
        const std::string& name,
        MasterWalletType type,
        const std::string& blockchain,
        double initialBalance = 0.0
    );

    std::future<MasterWallet> importMasterWallet(
        const std::string& privateKey,
        const std::string& name,
        MasterWalletType type
    );

    void deleteMasterWallet(const std::string& walletId);
    std::vector<MasterWallet> getMasterWallets();
    std::optional<MasterWallet> getMasterWallet(const std::string& walletId);
    std::vector<MasterWallet> getMasterWallets(const std::string& blockchain);

    // Balance Operations
    std::future<void> refreshBalances();
    double getBalance(const std::string& walletId);

    // Transaction Operations
    std::future<std::string> sendTransaction(
        const std::string& walletId,
        const std::string& to,
        double amount,
        const std::string& blockchain
    );

    std::future<std::vector<MasterTransaction>> getTransactions(const std::string& walletId);

    // Fee Management
    void setWithdrawFee(double percent);
    void setSwapFee(double percent);
    void setTransactionFee(double percent);
    double calculateFee(double amount, FeeType type);
    std::future<double> collectFees();

    // Auto-refill
    std::future<void> setupAutoRefill(
        const std::string& walletId,
        double threshold,
        double amount
    );

    // Supported Blockchains
    std::vector<std::pair<std::string, std::string>> getSupportedBlockchains();

    // Event Callbacks
    using WalletUpdateCallback = std::function<void(const MasterWallet&)>;
    using TransactionCallback = std::function<void(const MasterTransaction&)>;
    
    void setWalletUpdateCallback(WalletUpdateCallback callback);
    void setTransactionCallback(TransactionCallback callback);

private:
    MasterWalletService();
    ~MasterWalletService();
    MasterWalletService(const MasterWalletService&) = delete;
    MasterWalletService& operator=(const MasterWalletService&) = delete;

    // Storage
    void loadWallets();
    void saveWallets();

    // Blockchain
    double fetchBalanceFromChain(const std::string& address, const std::string& blockchain);
    std::string getRPCUrl(const std::string& blockchain);

    // Key Generation
    std::string generateAddress(const std::string& blockchain);
    std::string generatePublicKey();
    std::string deriveAddressFromPrivateKey(const std::string& privateKey);
    std::string derivePublicKeyFromPrivateKey(const std::string& privateKey);

    // Transaction
    std::vector<uint8_t> buildTransaction(const MasterWallet& wallet, const std::string& to, double amount);
    std::string broadcastTransaction(const std::vector<uint8_t>& tx, const std::string& blockchain);

    // Members
    static std::shared_ptr<MasterWalletService> instance_;
    CURL* curl_;
    bool initialized_;
    std::vector<MasterWallet> wallets_;
    std::map<std::string, double> balances_;
    
    // Fee configuration
    double withdrawFeePercent_ = 1.0;
    double swapFeePercent_ = 0.3;
    double transactionFeePercent_ = 0.1;
    double liquidityFeePercent_ = 0.2;
    
    // Supported blockchains
    std::vector<std::pair<std::string, std::string>> supportedBlockchains_;
    
    // Callbacks
    WalletUpdateCallback walletCallback_;
    TransactionCallback transactionCallback_;
};

// ============================================================================
// Exception
// ============================================================================

class MasterWalletException : public std::runtime_error {
public:
    enum class ErrorCode {
        WalletNotFound,
        InsufficientFunds,
        TransactionFailed,
        NetworkError,
        InvalidAddress,
        Unknown
    };

    MasterWalletException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_MASTER_SERVICE_H
