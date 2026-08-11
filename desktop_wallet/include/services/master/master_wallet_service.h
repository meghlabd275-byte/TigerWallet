/**
 * TigerWallet Desktop - Master Wallet Service
 * 103+ Networks, 500+ Tokens, Admin Controls
 */

#ifndef TIGER_MASTER_WALLET_SERVICE_H
#define TIGER_MASTER_WALLET_SERVICE_H

#include <memory>
#include <string>
#include <vector>
#include <map>
#include <functional>
#include <future>
#include <utility>

namespace tiger {
namespace wallet {

// ============================================================================
// Models
// ============================================================================

enum class MasterWalletType { HOT, COLD, OPERATIONS };

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
    std::chrono::system_clock::time_point created_at;
};

struct BlockchainNetwork {
    std::string id;
    std::string name;
    std::string symbol;
    int chain_id;
    std::string rpc_url;
    bool is_evm;
};

struct CryptoToken {
    std::string id;
    std::string symbol;
    std::string name;
    std::string image;
    double current_price;
    double market_cap;
    int rank;
    double price_change_24h;
};

// ============================================================================
// Master Wallet Service
// ============================================================================

class MasterWalletService {
public:
    static std::shared_ptr<MasterWalletService> getInstance();

    void initialize();
    
    // ============================================================================
    // Network Management (103+ Networks)
    // ============================================================================
    
    std::vector<BlockchainNetwork> getNetworks();
    void addNetwork(const BlockchainNetwork& network);
    void removeNetwork(const std::string& networkId);
    void updateNetwork(const BlockchainNetwork& network);
    
    // ============================================================================
    // Token Management (500+ Tokens)
    // ============================================================================
    
    std::vector<CryptoToken> getTokens();
    void addToken(const CryptoToken& token);
    void removeToken(const std::string& tokenId);
    std::vector<CryptoToken> searchTokens(const std::string& query);
    std::vector<CryptoToken> getTopTokens(int limit);
    
    // ============================================================================
    // Wallet Management
    // ============================================================================
    
    std::future<MasterWallet> createMasterWallet(
        const std::string& name,
        MasterWalletType type,
        const std::string& blockchain
    );
    
    std::vector<MasterWallet> getWallets();
    std::optional<MasterWallet> getWallet(const std::string& walletId);
    
    // ============================================================================
    // Balance Operations
    // ============================================================================
    
    std::future<void> refreshBalances();
    double getBalance(const std::string& walletId);

private:
    MasterWalletService(const MasterWalletService&) = delete;
    MasterWalletService& operator=(const MasterWalletService&) = delete;

public:
    MasterWalletService();
    ~MasterWalletService();

    void loadNetworks();
    void saveNetworks();
    void loadTokensFromAPI();
    void loadWallets();
    void saveWallets();
    
    double fetchBalanceFromChain(const std::string& address, const std::string& blockchain);
    std::string getRPCUrl(const std::string& blockchainId);
    std::string generateAddress();
    std::string generatePublicKey();
    // Delegates real wallet creation to the wallet_api backend (HTTP POST).
    // Returns {address, publicKey}; both empty on any failure (never faked).
    std::pair<std::string, std::string> createWalletViaBackend(const std::string& name, const std::string& blockchain);

    static std::shared_ptr<MasterWalletService> instance_;
    bool initialized_;
    
    std::vector<BlockchainNetwork> networks_;
    std::vector<CryptoToken> tokens_;
    std::vector<MasterWallet> wallets_;
    std::map<std::string, double> balances_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_MASTER_WALLET_SERVICE_H
