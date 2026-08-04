/**
 * TigerWallet Desktop Real-Time Alerts Service - C++ Implementation
 * Production-ready real-time notifications and alerts
 */

#ifndef REALTIME_ALERTS_SERVICE_HPP
#define REALTIME_ALERTS_SERVICE_HPP

#include <string>
#include <vector>
#include <memory>
#include <unordered_map>
#include <queue>
#include <mutex>
#include <atomic>
#include <chrono>
#include <functional>
#include <optional>

#include <nlohmann/json.hpp>

using json = nlohmann::json;

namespace tigerwallet {
namespace alerts {

// ============================================================================
// CONSTANTS
// ============================================================================

constexpr size_t MAX_ALERT_QUEUE = 10000;
constexpr size_t MAX_NOTIFICATION_HISTORY = 1000;
constexpr auto ALERT_CHECK_INTERVAL = std::chrono::seconds(1);

// Alert types
enum class AlertType : uint8_t {
    TRANSACTION = 0,
    PRICE = 1,
    BALANCE = 2,
    SECURITY = 3,
    SYSTEM = 4,
    NETWORK = 5,
    COMPLIANCE = 6,
    KYC = 7
};

// Alert severity
enum class AlertSeverity : uint8_t {
    INFO = 0,
    WARNING = 1,
    ERROR = 2,
    CRITICAL = 3
};

// Alert status
enum class AlertStatus : uint8_t {
    PENDING = 0,
    SENT = 1,
    READ = 2,
    DISMISSED = 3,
    ACKNOWLEDGED = 4
};

// Delivery channels
enum class DeliveryChannel : uint8_t {
    IN_APP = 0,
    EMAIL = 1,
    SMS = 2,
    PUSH = 3,
    WEBHOOK = 4
};

// ============================================================================
// DATA STRUCTURES
// ============================================================================

struct Alert {
    std::string id;
    std::string user_id;
    AlertType type;
    AlertSeverity severity;
    AlertStatus status;
    std::string title;
    std::string message;
    std::string category;
    json data;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point sent_at;
    std::chrono::system_clock::time_point read_at;
};

struct PriceAlert {
    std::string id;
    std::string user_id;
    std::string symbol;
    std::string target_price;
    std::string condition; // above, below, cross
    bool is_active;
    bool is_triggered;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point triggered_at;
};

struct TransactionAlert {
    std::string id;
    std::string user_id;
    std::string tx_hash;
    std::string from_address;
    std::string to_address;
    std::string amount;
    std::string symbol;
    std::string status;
    bool is_incoming;
    std::chrono::system_clock::time_point created_at;
};

struct SecurityAlert {
    std::string id;
    std::string user_id;
    std::string alert_type; // login, withdrawal, etc.
    std::string severity;
    std::string description;
    std::string ip_address;
    std::string location;
    json metadata;
    std::chrono::system_clock::time_point created_at;
    bool is_resolved;
};

struct NotificationPreferences {
    std::string user_id;
    bool email_enabled;
    bool sms_enabled;
    bool push_enabled;
    bool in_app_enabled;
    std::vector<AlertType> enabled_types;
    std::vector<AlertSeverity> enabled_severities;
    std::string email;
    std::string phone;
    std::string webhook_url;
};

struct WebhookConfig {
    std::string id;
    std::string url;
    std::string secret;
    std::vector<AlertType> events;
    bool is_active;
    uint32_t retry_count;
    uint32_t timeout_ms;
};

// ============================================================================
// ALERT MANAGER
// ============================================================================

class AlertManager {
public:
    static AlertManager& getInstance();
    
    bool initialize();
    void shutdown();
    
    // Alert creation
    std::string createAlert(const Alert& alert);
    std::string createPriceAlert(const PriceAlert& alert);
    std::string createTransactionAlert(const TransactionAlert& alert);
    std::string createSecurityAlert(const SecurityAlert& alert);
    
    // Alert retrieval
    std::optional<Alert> getAlert(const std::string& alert_id);
    std::vector<Alert> getUserAlerts(const std::string& user_id, int limit = 50);
    std::vector<Alert> getUnreadAlerts(const std::string& user_id);
    std::vector<PriceAlert> getPriceAlerts(const std::string& user_id);
    std::vector<SecurityAlert> getSecurityAlerts(const std::string& user_id);
    
    // Alert actions
    bool markAsRead(const std::string& alert_id);
    bool markAsDismissed(const std::string& alert_id);
    bool acknowledgeAlert(const std::string& alert_id);
    bool deleteAlert(const std::string& alert_id);
    
    // Price alerts
    bool createPriceAlert(const std::string& user_id, const std::string& symbol,
                         const std::string& target_price, const std::string& condition);
    bool cancelPriceAlert(const std::string& alert_id);
    void checkPriceAlerts();
    
    // Transaction monitoring
    void monitorTransaction(const std::string& tx_hash, const std::string& user_id);
    void checkPendingTransactions();
    
    // Security alerts
    std::string createSecurityAlert(const std::string& user_id, const std::string& alert_type,
                                   const std::string& severity, const std::string& description);
    bool resolveSecurityAlert(const std::string& alert_id);
    
    // Notification preferences
    bool updatePreferences(const NotificationPreferences& prefs);
    std::optional<NotificationPreferences> getPreferences(const std::string& user_id);
    
    // Webhooks
    std::string registerWebhook(const WebhookConfig& config);
    bool removeWebhook(const std::string& webhook_id);
    std::vector<WebhookConfig> getWebhooks(const std::string& user_id);
    
    // Statistics
    int getUnreadCount(const std::string& user_id);
    int getTotalAlertCount(const std::string& user_id);
    
    // Queue management
    void processAlertQueue();
    void sendAlert(const Alert& alert);

private:
    AlertManager() = default;
    ~AlertManager() = default;
    
    AlertManager(const AlertManager&) = delete;
    AlertManager& operator=(const AlertManager&) = delete;
    
    std::mutex mutex_;
    std::atomic<bool> running_;
    std::thread processing_thread_;
    
    // Storage
    std::unordered_map<std::string, Alert> alerts_;
    std::unordered_map<std::string, PriceAlert> price_alerts_;
    std::unordered_map<std::string, TransactionAlert> transaction_alerts_;
    std::unordered_map<std::string, SecurityAlert> security_alerts_;
    std::unordered_map<std::string, NotificationPreferences> preferences_;
    std::unordered_map<std::string, WebhookConfig> webhooks_;
    
    // Queue
    std::queue<Alert> alert_queue_;
    
    // Metrics
    std::atomic<uint64_t> total_alerts_created_{0};
    std::atomic<uint64_t> total_alerts_sent_{0};
    std::atomic<uint64_t> total_webhooks_delivered_{0};
    
    // Internal methods
    void loadAlertsFromStorage();
    void saveAlertsToStorage();
    void sendToChannels(const Alert& alert);
    void sendEmail(const Alert& alert);
    void sendSMS(const Alert& alert);
    void sendPush(const Alert& alert);
    void sendWebhook(const Alert& alert, const WebhookConfig& webhook);
    void sendInApp(const Alert& alert);
};

// ============================================================================
// PRICE MONITOR
// ============================================================================

class PriceMonitor {
public:
    PriceMonitor();
    ~PriceMonitor();
    
    void initialize(const std::string& price_feed_url);
    void shutdown();
    
    void startMonitoring();
    void stopMonitoring();
    
    void addSymbol(const std::string& symbol);
    void removeSymbol(const std::string& symbol);
    
    std::optional<std::string> getCurrentPrice(const std::string& symbol);
    std::map<std::string, std::string> getAllPrices();
    
    void checkAlerts();

private:
    bool initialized_;
    std::string price_feed_url_;
    std::atomic<bool> running_;
    std::thread monitor_thread_;
    
    std::mutex symbols_mutex_;
    std::vector<std::string> monitored_symbols_;
    
    std::mutex prices_mutex_;
    std::unordered_map<std::string, std::string> prices_;
    
    void fetchPrices();
    void updatePrice(const std::string& symbol, const std::string& price);
};

// ============================================================================
// TRANSACTION MONITOR
// ============================================================================

class TransactionMonitor {
public:
    TransactionMonitor();
    ~TransactionMonitor();
    
    void initialize(const std::string& rpc_url, const std::string& api_key);
    void shutdown();
    
    void watchTransaction(const std::string& tx_hash, const std::string& user_id);
    void unwatchTransaction(const std::string& tx_hash);
    
    void startMonitoring();
    void stopMonitoring();
    
private:
    bool initialized_;
    std::string rpc_url_;
    std::string api_key_;
    std::atomic<bool> running_;
    std::thread monitor_thread_;
    
    std::mutex watched_mutex_;
    std::unordered_map<std::string, std::string> watched_txs_; // tx_hash -> user_id
    
    void checkTransactions();
    void onTransactionConfirmed(const std::string& tx_hash, const std::string& user_id);
    void onTransactionFailed(const std::string& tx_hash, const std::string& user_id);
};

// ============================================================================
// SECURITY MONITOR
// ============================================================================

class SecurityMonitor {
public:
    SecurityMonitor();
    ~SecurityMonitor();
    
    void initialize();
    void shutdown();
    
    void logLoginAttempt(const std::string& user_id, const std::string& ip_address,
                        bool success, const std::string& location);
    void logWithdrawal(const std::string& user_id, const std::string& amount,
                      const std::string& address, const std::string& ip_address);
    void logSettingsChange(const std::string& user_id, const std::string& setting,
                          const std::string& old_value, const std::string& new_value);
    
    void checkSuspiciousActivity(const std::string& user_id);
    bool isAccountCompromised(const std::string& user_id);
    
private:
    bool initialized_;
    std::mutex login_history_mutex_;
    std::mutex withdrawal_history_mutex_;
    
    std::unordered_map<std::string, std::vector<json>> login_history_;
    std::unordered_map<std::string, std::vector<json>> withdrawal_history_;
    
    bool detectAnomalousLogin(const std::string& user_id);
    bool detectAnomalousWithdrawal(const std::string& user_id);
    void createSecurityAlert(const std::string& user_id, const std::string& alert_type,
                           const std::string& severity, const std::string& description);
};

} // namespace alerts
} // namespace tigerwallet

#endif // REALTIME_ALERTS_SERVICE_HPP
