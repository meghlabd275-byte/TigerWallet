/**
 * TigerWallet Desktop Multi-Sig UI Components - C++ Implementation
 * Production-ready multi-signature wallet user interface
 */

#ifndef MULTISIG_UI_HPP
#define MULTISIG_UI_HPP

#include <string>
#include <vector>
#include <memory>
#include <functional>
#include <unordered_map>
#include <optional>

#include <nlohmann/json.hpp>

#include "ui/theme.hpp"

using json = nlohmann::json;

namespace tigerwallet {
namespace ui {
namespace multisig {

// ============================================================================
// DATA STRUCTURES
// ============================================================================

struct MultiSigWallet {
    std::string id;
    std::string name;
    std::string address;
    std::string blockchain;
    uint32_t threshold;
    uint32_t total_signers;
    std::vector<SignerInfo> signers;
    std::vector<TransactionInfo> pending_transactions;
    std::vector<TransactionInfo> confirmed_transactions;
    std::string balance;
    std::string balance_usd;
    bool is_active;
    std::chrono::system_clock::time_point created_at;
};

struct SignerInfo {
    std::string id;
    std::string name;
    std::string address;
    std::string role; // initiator, approver, admin
    std::string status; // active, pending, removed
    bool has_approved;
    std::chrono::system_clock::time_point approved_at;
    std::string avatar_url;
};

struct TransactionInfo {
    std::string id;
    std::string tx_hash;
    std::string from;
    std::string to;
    std::string amount;
    std::string symbol;
    std::string fee;
    std::string status; // pending, approved, rejected, executed, failed
    std::vector<ApprovalInfo> approvals;
    uint32_t required_approvals;
    uint32_t current_approvals;
    std::string description;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point executed_at;
    std::chrono::system_clock::time_point expires_at;
};

struct ApprovalInfo {
    std::string signer_id;
    std::string signer_name;
    std::string signature;
    std::string status; // pending, approved, rejected
    std::chrono::system_clock::time_point timestamp;
};

struct CreateWalletRequest {
    std::string name;
    std::string blockchain;
    uint32_t threshold;
    std::vector<SignerRequest> signers;
    std::string description;
};

struct SignerRequest {
    std::string name;
    std::string email;
    std::string address;
    std::string role;
};

struct CreateTransactionRequest {
    std::string wallet_id;
    std::string to_address;
    std::string amount;
    std::string symbol;
    std::string gas_limit;
    std::string gas_price;
    std::string data;
    std::string description;
    uint64_t expiration_hours;
};

// ============================================================================
// MULTI-SIG SERVICE
// ============================================================================

class MultiSigService {
public:
    static MultiSigService& getInstance();
    
    bool initialize();
    void shutdown();
    
    // Wallet management
    std::string createWallet(const CreateWalletRequest& request);
    std::optional<MultiSigWallet> getWallet(const std::string& wallet_id);
    std::vector<MultiSigWallet> getAllWallets(const std::string& user_id);
    bool updateWallet(const std::string& wallet_id, const MultiSigWallet& wallet);
    bool deleteWallet(const std::string& wallet_id);
    
    // Signer management
    std::string addSigner(const std::string& wallet_id, const SignerRequest& signer);
    bool removeSigner(const std::string& wallet_id, const std::string& signer_id);
    bool updateSigner(const std::string& wallet_id, const SignerInfo& signer);
    std::vector<SignerInfo> getSigners(const std::string& wallet_id);
    
    // Transaction management
    std::string createTransaction(const CreateTransactionRequest& request);
    std::optional<TransactionInfo> getTransaction(const std::string& tx_id);
    std::vector<TransactionInfo> getPendingTransactions(const std::string& wallet_id);
    std::vector<TransactionInfo> getTransactionHistory(const std::string& wallet_id, int limit = 50);
    
    // Approval workflow
    bool approveTransaction(const std::string& tx_id, const std::string& signer_id, 
                         const std::string& signature);
    bool rejectTransaction(const std::string& tx_id, const std::string& signer_id,
                         const std::string& reason);
    bool executeTransaction(const std::string& tx_id);
    bool cancelTransaction(const std::string& tx_id);
    
    // Signer actions
    std::vector<TransactionInfo> getPendingApprovals(const std::string& signer_id);
    bool signTransaction(const std::string& tx_id, const std::string& signer_id);
    
    // Events
    using WalletEventCallback = std::function<void(const MultiSigWallet&)>;
    using TransactionEventCallback = std::function<void(const TransactionInfo&)>;
    
    void registerWalletCallback(WalletEventCallback callback);
    void registerTransactionCallback(TransactionEventCallback callback);

private:
    MultiSigService() = default;
    ~MultiSigService() = default;
    
    MultiSigService(const MultiSigService&) = delete;
    MultiSigService& operator=(const MultiSigService&) = delete;
    
    std::mutex mutex_;
    bool initialized_;
    
    std::unordered_map<std::string, MultiSigWallet> wallets_;
    std::unordered_map<std::string, std::vector<TransactionInfo>> transactions_;
    
    std::vector<WalletEventCallback> wallet_callbacks_;
    std::vector<TransactionEventCallback> tx_callbacks_;
    
    void notifyWalletEvent(const MultiSigWallet& wallet);
    void notifyTransactionEvent(const TransactionInfo& tx);
    
    bool validateThreshold(uint32_t threshold, uint32_t total_signers);
    bool validateTransaction(const TransactionInfo& tx);
    bool hasRequiredApprovals(const TransactionInfo& tx);
};

// ============================================================================
// MULTI-SIG UI WIDGET
// ============================================================================

class MultiSigUIWidget {
public:
    MultiSigUIWidget();
    ~MultiSigUIWidget() = default;
    
    // Render methods
    std::string renderWalletList(const std::vector<MultiSigWallet>& wallets);
    std::string renderWalletDetails(const MultiSigWallet& wallet);
    std::string renderCreateWalletForm();
    std::string renderTransactionList(const std::vector<TransactionInfo>& transactions);
    std::string renderTransactionDetails(const TransactionInfo& tx);
    std::string renderSignerList(const std::vector<SignerInfo>& signers);
    std::string renderApprovalPanel(const TransactionInfo& tx);
    std::string renderCreateTransactionForm(const MultiSigWallet& wallet);
    
    // Interactive elements
    std::string renderApproveButton(const std::string& tx_id, bool enabled);
    std::string renderRejectButton(const std::string& tx_id, bool enabled);
    std::string renderExecuteButton(const std::string& tx_id, bool enabled);
    std::string renderSignerAvatar(const SignerInfo& signer);
    std::string renderStatusBadge(const std::string& status);
    std::string renderProgressBar(uint32_t current, uint32_t required);
    
    // Dialogs
    std::string renderAddSignerDialog(const std::string& wallet_id);
    std::string renderRemoveSignerDialog(const std::string& wallet_id, const std::string& signer_id);
    std::string renderRejectDialog(const std::string& tx_id);
    
    // Dashboard
    std::string renderDashboard(const MultiSigWallet& wallet);
    std::string renderPendingApprovals(const std::vector<TransactionInfo>& txs);
    std::string renderRecentTransactions(const std::vector<TransactionInfo>& txs);
    std::string renderSignerOverview(const std::vector<SignerInfo>& signers);
    
    // Statistics
    std::string renderStatistics(const MultiSigWallet& wallet);
    std::string renderTransactionVolume(const std::string& wallet_id, const std::string& period);
    std::string renderApprovalRate(const std::string& wallet_id);

private:
    std::mutex mutex_;
    std::string theme_; // CSS class suffix derived from ThemeManager ("dark"/"light")

    std::string renderButton(const std::string& id, const std::string& text, 
                           const std::string& style, bool enabled);
    std::string renderCard(const std::string& title, const std::string& content);
    std::string renderModal(const std::string& id, const std::string& title, 
                          const std::string& content);
    std::string renderTable(const std::vector<std::string>& headers, 
                          const std::vector<std::vector<std::string>>& rows);
    
    // Status/severity colors resolve through ThemeManager so badges match the
    // active palette (dark/light). Semantic colors come from ThemeColors.
    std::string getStatusColor(const std::string& status) {
        const TigerWallet::ThemeColors& c =
            TigerWallet::ThemeManager::getInstance().getColors();
        if (status == "active" || status == "confirmed" || status == "executed")
            return c.successColor;
        if (status == "pending" || status == "waiting")
            return c.warningColor;
        if (status == "rejected" || status == "failed" || status == "removed")
            return c.errorColor;
        return c.textSecondaryColor;
    }
    std::string getSeverityColor(const std::string& severity) {
        const TigerWallet::ThemeColors& c =
            TigerWallet::ThemeManager::getInstance().getColors();
        if (severity == "error" || severity == "critical") return c.errorColor;
        if (severity == "warning") return c.warningColor;
        if (severity == "success" || severity == "info") return c.successColor;
        return c.accentColor;
    }
};

// ============================================================================
// MULTI-SIG EVENT HANDLER
// ============================================================================

class MultiSigEventHandler {
public:
    MultiSigEventHandler();
    ~MultiSigEventHandler() = default;
    
    void setService(MultiSigService* service);
    
    void onWalletCreated(const std::string& wallet_id);
    void onWalletUpdated(const std::string& wallet_id);
    void onWalletDeleted(const std::string& wallet_id);
    
    void onTransactionCreated(const std::string& tx_id);
    void onTransactionApproved(const std::string& tx_id, const std::string& signer_id);
    void onTransactionRejected(const std::string& tx_id, const std::string& signer_id);
    void onTransactionExecuted(const std::string& tx_id);
    void onTransactionFailed(const std::string& tx_id, const std::string& error);
    
    void onSignerAdded(const std::string& wallet_id, const std::string& signer_id);
    void onSignerRemoved(const std::string& wallet_id, const std::string& signer_id);
    
    void onNotification(const std::string& title, const std::string& message, 
                       const std::string& severity);

private:
    MultiSigService* service_;
    std::mutex mutex_;
};

} // namespace multisig
} // namespace ui
} // namespace tigerwallet

#endif // MULTISIG_UI_HPP
