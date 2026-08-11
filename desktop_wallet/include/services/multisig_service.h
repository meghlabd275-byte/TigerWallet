/**
 * TigerWallet Desktop - Multisig Service
 * Threshold multi-signature wallet operations via the go/multisig_service
 * backend (REST, default http://localhost:8450). The backend assembles
 * off-chain owner signatures and broadcasts the on-chain
 * `MultisigWallet.executeTransaction` call. Honest results only.
 */

#ifndef TIGER_WALLET_MULTISIG_SERVICE_H
#define TIGER_WALLET_MULTISIG_SERVICE_H

#include <memory>
#include <string>
#include <vector>
#include <future>
#include <optional>

namespace tiger {
namespace wallet {

struct MultisigWalletInfo {
    std::string id;
    std::string address;
    std::vector<std::string> owners;
    int threshold = 1;
};

struct MultisigTransaction {
    std::string id;
    std::string wallet_id;
    std::string to;
    std::string value;
    std::string data;
    bool executed = false;
    int confirmations = 0;
    int threshold = 1;
    std::string tx_hash; // populated once executed on-chain
};

class MultisigService {
public:
    static std::shared_ptr<MultisigService> getInstance();

    MultisigService();
    ~MultisigService() = default;

    void initialize(const std::string& baseUrl = "http://localhost:8450");
    void shutdown();

    // Wallet lifecycle
    std::future<MultisigWalletInfo> createWallet(const std::vector<std::string>& owners,
                                                  int threshold);
    std::future<std::vector<MultisigWalletInfo>> listWallets();
    std::future<MultisigWalletInfo> getWallet(const std::string& walletId);

    // Owner management
    std::future<bool> addOwner(const std::string& walletId, const std::string& ownerAddress);

    // Transaction lifecycle
    std::future<MultisigTransaction> createTransaction(const std::string& walletId,
                                                      const std::string& to,
                                                      const std::string& value,
                                                      const std::string& data);
    std::future<MultisigTransaction> getTransaction(const std::string& txId);
    std::future<bool> signTransaction(const std::string& txId);
    std::future<MultisigTransaction> executeTransaction(const std::string& txId);
    std::future<bool> revokeTransaction(const std::string& txId);
    std::future<std::vector<MultisigTransaction>> pendingTransactions(const std::string& walletId);

private:
    MultisigService(const MultisigService&) = delete;
    MultisigService& operator=(const MultisigService&) = delete;

    std::string post(const std::string& endpoint, const std::string& body);
    std::string get(const std::string& endpoint);
    bool initialized_ = false;
    std::string baseUrl_;
    static std::shared_ptr<MultisigService> instance_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_MULTISIG_SERVICE_H
