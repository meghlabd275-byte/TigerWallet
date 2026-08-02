/**
 * TigerWallet Desktop - Blockchain Service
 * Multi-chain blockchain integration with RPC providers
 */

#ifndef TIGER_WALLET_BLOCKCHAIN_SERVICE_H
#define TIGER_WALLET_BLOCKCHAIN_SERVICE_H

#include "models/wallet_models.h"
#include <memory>
#include <functional>
#include <curl/curl.h>
#include <string>
#include <map>
#include <vector>
#include <optional>
#include <future>

namespace tiger {
namespace wallet {

// ============================================================================
// JSON-RPC Types
// ============================================================================

struct JsonRpcRequest {
    std::string jsonrpc = "2.0";
    std::string method;
    std::vector<std::string> params;
    int id = 1;
};

struct JsonRpcResponse {
    std::optional<std::string> result;
    std::optional<std::string> error;
    int id;
};

// ============================================================================
// Blockchain Service
// ============================================================================

class BlockchainService {
public:
    static std::shared_ptr<BlockchainService> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // Chain Management
    std::vector<Chain> getSupportedChains();
    std::optional<Chain> getChain(const std::string& chainId);
    std::optional<Chain> getChainBySymbol(const std::string& symbol);

    // Balance Operations
    std::future<double> getBalance(const std::string& address, const std::string& chainId);
    std::future<double> getTokenBalance(const std::string& address, const std::string& tokenAddress, const std::string& chainId);

    // Transaction Operations
    std::future<std::string> sendTransaction(
        const std::string& from,
        const std::string& to,
        const std::string& amount,
        const std::string& chainId,
        const std::optional<std::string>& tokenAddress = std::nullopt
    );

    std::future<std::optional<Transaction>> getTransactionReceipt(const std::string& txHash, const std::string& chainId);
    std::future<std::vector<Transaction>> getTransactions(const std::string& address, const std::string& chainId, int limit = 20);

    // Gas Operations
    std::future<std::string> getGasPrice(const std::string& chainId);
    std::future<std::string> estimateGas(
        const std::string& from,
        const std::string& to,
        const std::string& value,
        const std::string& chainId,
        const std::optional<std::string>& data = std::nullopt
    );

    // Token Operations
    std::future<std::vector<Token>> getTokens(const std::string& address, const std::string& chainId);

    // Wallet Operations
    std::future<Wallet> createWallet(const Chain& chain, const std::string& name);
    std::future<Wallet> importWallet(const std::string& mnemonic, const std::string& chainId, const std::string& name);
    std::future<Wallet> importFromPrivateKey(const std::string& privateKey, const std::string& chainId, const std::string& name);

    // Address Validation
    bool isValidAddress(const std::string& address, const std::string& chainId);

    // Event Callbacks
    using BalanceUpdateCallback = std::function<void(const std::string&, double)>;
    using TransactionCallback = std::function<void(const Transaction&)>;
    
    void setBalanceUpdateCallback(BalanceUpdateCallback callback);
    void setTransactionCallback(TransactionCallback callback);

private:
    BlockchainService();
    ~BlockchainService();
    BlockchainService(const BlockchainService&) = delete;
    BlockchainService& operator=(const BlockchainService&) = delete;

    // RPC Communication
    JsonRpcResponse sendJsonRpc(const std::string& rpcUrl, const JsonRpcRequest& request);
    std::string callRpc(const std::string& url, const std::string& body);

    // EVM Operations
    double evmGetBalance(const std::string& address, const Chain& chain);
    double evmGetTokenBalance(const std::string& address, const std::string& tokenAddress, const Chain& chain);
    std::string evmSendTransaction(
        const std::string& from,
        const std::string& to,
        const std::string& amount,
        const Chain& chain,
        const std::optional<std::string>& data = std::nullopt
    );

    // Solana Operations
    double solanaGetBalance(const std::string& address, const Chain& chain);

    // Bitcoin Operations
    double bitcoinGetBalance(const std::string& address, const Chain& chain);

    // Key Derivation
    std::pair<std::string, std::string> deriveKeyFromMnemonic(const std::string& mnemonic, const Chain& chain);
    std::pair<std::string, std::string> deriveKeyFromPrivateKey(const std::string& privateKey);

    // Validation
    bool validateMnemonic(const std::string& mnemonic);

    // Members
    static std::shared_ptr<BlockchainService> instance_;
    std::map<std::string, Chain> chains_;
    CURL* curl_;
    bool initialized_;
    BalanceUpdateCallback balanceCallback_;
    TransactionCallback transactionCallback_;
};

// ============================================================================
// Exception Classes
// ============================================================================

class BlockchainException : public std::runtime_error {
public:
    enum class ErrorCode {
        InvalidAddress,
        InvalidMnemonic,
        InsufficientFunds,
        NetworkError,
        TransactionFailed,
        UnsupportedChain,
        WalletLocked,
        Unknown
    };

    BlockchainException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_BLOCKCHAIN_SERVICE_H
