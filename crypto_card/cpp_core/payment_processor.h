/**
 * TigerWallet Payment Card - High-Performance C++ Processing Engine
 * Ultra-low latency payment processing for crypto card transactions
 */

#ifndef TIGER_PAYMENT_PROCESSOR_H
#define TIGER_PAYMENT_PROCESSOR_H

#include <array>
#include <atomic>
#include <cstdint>
#include <memory>
#include <mutex>
#include <queue>
#include <string>
#include <string_view>
#include <thread>
#include <unordered_map>
#include <vector>

namespace tiger {
namespace payment {

// ============================================================================
// Constants
// ============================================================================

constexpr size_t MAX_PENDING_TRANSACTIONS = 100000;
constexpr size_t TRANSACTION_QUEUE_SIZE = 10000;
constexpr uint64_t MILLISECONDS_PER_SECOND = 1000ULL;
constexpr uint64_t NANOSECONDS_PER_MILLISECOND = 1000000ULL;

// Card network codes
constexpr uint8_t CARD_NETWORK_VISA = 0x01;
constexpr uint8_t CARD_NETWORK_MASTERCARD = 0x02;
constexpr uint8_t CARD_NETWORK_AMEX = 0x03;
constexpr uint8_t CARD_NETWORK_DISCOVER = 0x04;

// Transaction types
constexpr uint8_t TRANSACTION_TYPE_PURCHASE = 0x01;
constexpr uint8_t TRANSACTION_TYPE_WITHDRAWAL = 0x02;
constexpr uint8_t TRANSACTION_TYPE_REFUND = 0x03;
constexpr uint8_t TRANSACTION_TYPE_CHARGEBACK = 0x04;
constexpr uint8_t TRANSACTION_TYPE_LOAD = 0x05;

// Transaction status
constexpr uint8_t STATUS_PENDING = 0x01;
constexpr uint8_t STATUS_PROCESSING = 0x02;
constexpr uint8_t STATUS_APPROVED = 0x03;
constexpr uint8_t STATUS_DECLINED = 0x04;
constexpr uint8_t STATUS_FAILED = 0x05;
constexpr uint8_t STATUS_COMPLETED = 0x06;

// Currency codes (ISO 4217)
constexpr uint32_t CURRENCY_USD = 840;
constexpr uint32_t CURRENCY_EUR = 978;
constexpr uint32_t CURRENCY_GBP = 826;
constexpr uint32_t CURRENCY_JPY = 392;

// ============================================================================
// Data Structures
// ============================================================================

// Card data (never stored in plaintext)
struct CardData {
    uint8_t card_number_encrypted[16];  // AES-256 encrypted
    uint8_t cvv_encrypted[8];
    uint8_t exp_month;
    uint8_t exp_year;
    uint8_t pin_block[8];  // IBM 3624 pin block
    
    CardData() : exp_month(0), exp_year(0) {
        memset(card_number_encrypted, 0, sizeof(card_number_encrypted));
        memset(cvv_encrypted, 0, sizeof(cvv_encrypted));
        memset(pin_block, 0, sizeof(pin_block));
    }
};

// Card token (safe to store)
struct CardToken {
    std::string token_id;
    uint8_t last_four[4];
    uint8_t card_type;  // Debit/Credit
    uint8_t network;
    uint8_t exp_month;
    uint8_t exp_year;
    std::string cardholder_name;
    uint64_t created_at;
    uint64_t last_used_at;
    bool is_frozen;
    uint32_t spend_limit_daily;
    uint32_t spend_limit_monthly;
    
    CardToken() : card_type(0), network(0), exp_month(0), exp_year(0),
                  created_at(0), last_used_at(0), is_frozen(false),
                  spend_limit_daily(0), spend_limit_monthly(0) {
        last_four[0] = last_four[1] = last_four[2] = last_four[3] = 0;
    }
};

// Transaction record
struct alignas(64) Transaction {
    uint64_t transaction_id;
    uint64_t user_id;
    uint64_t card_id;
    uint8_t transaction_type;
    uint8_t status;
    uint32_t amount;
    uint32_t original_amount;
    uint32_t currency;
    uint32_t merchant_id;
    std::string merchant_name;
    std::string merchant_category;
    std::string merchant_country;
    int32_t exchange_rate;
    uint32_t fees;
    uint32_t cashback;
    uint64_t timestamp;
    uint64_t processed_at;
    uint8_t risk_score;
    std::string rejection_reason;
    uint8_t avs_response;
    uint8_t cvv_response;
    uint8_t three_ds_status;
    uint8_t network_reference[8];
    uint64_t parent_transaction;
    
    Transaction() : transaction_id(0), user_id(0), card_id(0),
                   transaction_type(0), status(STATUS_PENDING),
                   amount(0), original_amount(0), currency(CURRENCY_USD),
                   merchant_id(0), exchange_rate(10000), fees(0), cashback(0),
                   timestamp(0), processed_at(0), risk_score(0),
                   parent_transaction(0) {
        avs_response = cvv_response = three_ds_status = 0;
        memset(network_reference, 0, sizeof(network_reference));
    }
};

// Authorization request
struct AuthorizationRequest {
    uint64_t user_id;
    uint64_t card_id;
    uint32_t amount;
    uint32_t currency;
    uint32_t merchant_id;
    std::string merchant_name;
    std::string merchant_category;
    std::string merchant_country;
    std::string terminal_id;
    uint8_t terminal_type;
    double latitude;
    double longitude;
    uint8_t pin_verified;
    uint8_t cvv_verified;
};

// Authorization response
struct AuthorizationResponse {
    uint64_t transaction_id;
    uint8_t status;
    uint32_t approved_amount;
    uint32_t auth_code;
    uint32_t response_code;
    uint8_t risk_score;
    std::string response_message;
    uint64_t timestamp;
    uint8_t network_reference[8];
};

// Card limits
struct CardLimits {
    uint32_t daily_limit;
    uint32_t monthly_limit;
    uint32_t per_transaction_limit;
    uint32_t daily_spent;
    uint32_t monthly_spent;
    uint32_t transaction_count_today;
    
    CardLimits() : daily_limit(0), monthly_limit(0), per_transaction_limit(0),
                   daily_spent(0), monthly_spent(0), transaction_count_today(0) {}
};

// ============================================================================
// Risk Analysis
// ============================================================================

class RiskAnalyzer {
private:
    // ML model weights (simplified for demo)
    float velocity_weight_;
    float amount_weight_;
    float merchant_risk_weight_;
    float location_weight_;
    float time_weight_;
    
    // Merchant risk categories
    std::unordered_map<std::string, float> merchant_risk_scores_;
    std::unordered_map<std::string, uint64_t> country_risk_scores_;
    
    // Velocity tracking
    struct VelocityData {
        uint64_t last_transaction_time;
        uint32_t transaction_count_1h;
        uint32_t transaction_count_24h;
        uint32_t total_amount_1h;
        uint32_t total_amount_24h;
    };
    std::unordered_map<uint64_t, VelocityData> user_velocity_;
    mutable std::mutex velocity_mutex_;
    
public:
    RiskAnalyzer();
    ~RiskAnalyzer() = default;
    
    // Analyze transaction risk
    uint8_t analyze_risk(
        const AuthorizationRequest& request,
        const CardLimits& limits,
        const std::vector<Transaction>& recent_transactions
    );
    
    // Check velocity limits
    bool check_velocity(uint64_t user_id, uint32_t amount);
    
    // Update velocity data after transaction
    void update_velocity(uint64_t user_id, uint32_t amount);
    
    // Get merchant risk score
    float get_merchant_risk(const std::string& category, const std::string& country);
    
    // Update merchant risk score (for learning)
    void update_merchant_risk(const std::string& category, const std::string& country, bool confirmed_fraud);
};

// ============================================================================
// Fraud Detection
// ============================================================================

class FraudDetector {
private:
    // Pattern detection
    struct FraudPattern {
        std::string pattern_id;
        std::string description;
        float threshold;
        uint32_t occurrence_count;
        bool is_active;
    };
    
    std::vector<FraudPattern> patterns_;
    std::mutex patterns_mutex_;
    
    // Device fingerprinting
    struct DeviceFingerprint {
        std::string device_id;
        std::string ip_address;
        std::string user_agent;
        std::vector<std::string> recent_ips;
        uint64_t first_seen;
        uint64_t last_seen;
        uint32_t fraud_count;
    };
    std::unordered_map<std::string, DeviceFingerprint> device_fingerprints_;
    mutable std::mutex device_mutex_;
    
public:
    FraudDetector();
    ~FraudDetector() = default;
    
    // Check for fraud indicators
    struct FraudCheckResult {
        bool is_fraud;
        uint8_t risk_level;  // 0-100
        std::vector<std::string> flags;
        std::string recommendation;
    };
    
    FraudCheckResult check_fraud(
        const AuthorizationRequest& request,
        const std::string& device_id,
        const std::string& ip_address
    );
    
    // Add device fingerprint
    void add_device_fingerprint(
        const std::string& device_id,
        const std::string& ip_address,
        const std::string& user_agent
    );
    
    // Report confirmed fraud
    void report_fraud(const std::string& device_id, const std::string& ip_address);
};

// ============================================================================
// Payment Processor
// ============================================================================

class PaymentProcessor {
private:
    // Core components
    std::unique_ptr<RiskAnalyzer> risk_analyzer_;
    std::unique_ptr<FraudDetector> fraud_detector_;
    
    // Transaction storage
    std::unordered_map<uint64_t, Transaction> transactions_;
    mutable std::mutex transactions_mutex_;
    uint64_t next_transaction_id_;
    
    // User card tokens
    std::unordered_map<uint64_t, std::vector<CardToken>> user_cards_;
    mutable std::mutex cards_mutex_;
    
    // Card limits
    std::unordered_map<uint64_t, CardLimits> card_limits_;
    mutable std::mutex limits_mutex_;
    
    // Processing threads
    std::vector<std::thread> worker_threads_;
    std::queue<AuthorizationRequest> transaction_queue_;
    std::mutex queue_mutex_;
    std::condition_variable queue_cv_;
    std::atomic<bool> running_;
    
    // Statistics
    struct ProcessorStats {
        std::atomic<uint64_t> total_transactions;
        std::atomic<uint64_t> approved_transactions;
        std::atomic<uint64_t> declined_transactions;
        std::atomic<uint64_t> total_volume;
        std::atomic<uint64_t> total_fees;
        std::atomic<uint64_t> total_cashback;
        std::atomic<uint64_t> avg_processing_time_us;
        std::atomic<uint64_t> min_processing_time_us;
        std::atomic<uint64_t> max_processing_time_us;
        
        ProcessorStats() : total_transactions(0), approved_transactions(0),
                          declined_transactions(0), total_volume(0),
                          total_fees(0), total_cashback(0),
                          avg_processing_time_us(0), 
                          min_processing_time_us(UINT64_MAX),
                          max_processing_time_us(0) {}
    } stats_;
    
    // Worker thread function
    void worker_thread();
    
    // Process single authorization
    AuthorizationResponse process_authorization(const AuthorizationRequest& request);
    
    // Apply fees and cashback
    void apply_fees_and_cashback(Transaction& transaction);
    
    // Update limits
    void update_limits(uint64_t user_id, uint32_t amount);
    
    // Generate auth code
    uint32_t generate_auth_code();
    
public:
    PaymentProcessor();
    ~PaymentProcessor();
    
    // Start/stop
    void start(uint32_t num_threads = std::thread::hardware_concurrency());
    void stop();
    
    // Card management
    std::string create_card_token(
        uint64_t user_id,
        const CardData& card_data,
        const std::string& cardholder_name,
        uint8_t card_type
    );
    
    bool freeze_card(uint64_t user_id, const std::string& token_id);
    bool unfreeze_card(uint64_t user_id, const std::string& token_id);
    bool update_card_limits(uint64_t user_id, const std::string& token_id, const CardLimits& limits);
    std::vector<CardToken> get_user_cards(uint64_t user_id);
    bool delete_card(uint64_t user_id, const std::string& token_id);
    
    // Transaction processing
    AuthorizationResponse authorize(const AuthorizationRequest& request);
    bool capture(uint64_t transaction_id);
    bool refund(uint64_t transaction_id, uint32_t amount);
    bool void_transaction(uint64_t transaction_id);
    
    // Transaction queries
    std::optional<Transaction> get_transaction(uint64_t transaction_id);
    std::vector<Transaction> get_user_transactions(
        uint64_t user_id,
        uint64_t start_time = 0,
        uint64_t end_time = UINT64_MAX,
        uint32_t limit = 100
    );
    std::vector<Transaction> get_card_transactions(
        uint64_t user_id,
        const std::string& token_id,
        uint32_t limit = 100
    );
    
    // Statistics
    struct ProcessorStats get_stats() const;
    
    // Risk analyzer access
    RiskAnalyzer* get_risk_analyzer() { return risk_analyzer_.get(); }
    FraudDetector* get_fraud_detector() { return fraud_detector_.get(); }
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline uint32_t PaymentProcessor::generate_auth_code() {
    // Generate 6-digit auth code
    static std::mt19937 rng(std::random_device{}());
    static std::uniform_int_distribution<uint32_t> dist(100000, 999999);
    return dist(rng);
}

inline AuthorizationResponse PaymentProcessor::authorize(const AuthorizationRequest& request) {
    // Record start time
    auto start = std::chrono::high_resolution_clock::now();
    
    // Process authorization
    AuthorizationResponse response = process_authorization(request);
    
    // Record processing time
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start).count();
    
    // Update statistics
    uint64_t old_avg = stats_.avg_processing_time_us.load();
    uint64_t count = stats_.total_transactions.load() + 1;
    stats_.avg_processing_time_us.store((old_avg * (count - 1) + duration) / count);
    
    uint64_t old_min = stats_.min_processing_time_us.load();
    while (duration < old_min && 
           !stats_.min_processing_time_us.compare_exchange_weak(old_min, duration)) {}
    
    uint64_t old_max = stats_.max_processing_time_us.load();
    while (duration > old_max && 
           !stats_.max_processing_time_us.compare_exchange_weak(old_max, duration)) {}
    
    return response;
}

inline bool PaymentProcessor::capture(uint64_t transaction_id) {
    std::lock_guard<std::mutex> lock(transactions_mutex_);
    
    auto it = transactions_.find(transaction_id);
    if (it == transactions_.end()) return false;
    
    auto& tx = it->second;
    if (tx.status != STATUS_APPROVED) return false;
    
    tx.status = STATUS_COMPLETED;
    tx.processed_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    return true;
}

inline bool PaymentProcessor::refund(uint64_t transaction_id, uint32_t amount) {
    std::lock_guard<std::mutex> lock(transactions_mutex_);
    
    auto it = transactions_.find(transaction_id);
    if (it == transactions_.end()) return false;
    
    auto& original = it->second;
    if (original.status != STATUS_COMPLETED) return false;
    if (amount > original.amount) return false;
    
    // Create refund transaction
    uint64_t refund_id = next_transaction_id_++;
    Transaction refund;
    refund.transaction_id = refund_id;
    refund.user_id = original.user_id;
    refund.card_id = original.card_id;
    refund.transaction_type = TRANSACTION_TYPE_REFUND;
    refund.status = STATUS_COMPLETED;
    refund.amount = amount;
    refund.original_amount = amount;
    refund.currency = original.currency;
    refund.merchant_id = original.merchant_id;
    refund.merchant_name = original.merchant_name;
    refund.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    refund.processed_at = refund.timestamp;
    refund.parent_transaction = transaction_id;
    
    transactions_[refund_id] = refund;
    original.amount -= amount;
    
    return true;
}

inline bool PaymentProcessor::void_transaction(uint64_t transaction_id) {
    std::lock_guard<std::mutex> lock(transactions_mutex_);
    
    auto it = transactions_.find(transaction_id);
    if (it == transactions_.end()) return false;
    
    auto& tx = it->second;
    if (tx.status != STATUS_APPROVED && tx.status != STATUS_PENDING) return false;
    
    tx.status = STATUS_DECLINED;
    tx.rejection_reason = "VOIDED";
    tx.processed_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    // Restore limits
    {
        std::lock_guard<std::mutex> limit_lock(limits_mutex_);
        auto limit_it = card_limits_.find(tx.user_id);
        if (limit_it != card_limits_.end()) {
            limit_it->second.daily_spent -= tx.amount;
            limit_it->second.monthly_spent -= tx.amount;
            limit_it->second.transaction_count_today--;
        }
    }
    
    return true;
}

inline std::optional<Transaction> PaymentProcessor::get_transaction(uint64_t transaction_id) {
    std::shared_lock<std::mutex> lock(transactions_mutex_);
    auto it = transactions_.find(transaction_id);
    if (it != transactions_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline std::vector<Transaction> PaymentProcessor::get_user_transactions(
    uint64_t user_id,
    uint64_t start_time,
    uint64_t end_time,
    uint32_t limit
) {
    std::shared_lock<std::mutex> lock(transactions_mutex_);
    
    std::vector<Transaction> result;
    result.reserve(limit);
    
    for (const auto& [id, tx] : transactions_) {
        if (tx.user_id != user_id) continue;
        if (tx.timestamp < start_time || tx.timestamp > end_time) continue;
        if (result.size() >= limit) break;
        
        result.push_back(tx);
    }
    
    // Sort by timestamp descending
    std::sort(result.begin(), result.end(), [](const Transaction& a, const Transaction& b) {
        return a.timestamp > b.timestamp;
    });
    
    return result;
}

inline std::vector<Transaction> PaymentProcessor::get_card_transactions(
    uint64_t user_id,
    const std::string& token_id,
    uint32_t limit
) {
    std::shared_lock<std::mutex> lock(transactions_mutex_);
    
    std::vector<Transaction> result;
    result.reserve(limit);
    
    // Find card ID from token
    uint64_t card_id = 0;
    {
        std::shared_lock<std::mutex> card_lock(cards_mutex_);
        auto user_it = user_cards_.find(user_id);
        if (user_it != user_cards_.end()) {
            for (const auto& card : user_it->second) {
                if (card.token_id == token_id) {
                    card_id = std::hash<std::string>{}(token_id);
                    break;
                }
            }
        }
    }
    
    if (card_id == 0) return result;
    
    for (const auto& [id, tx] : transactions_) {
        if (tx.card_id != card_id) continue;
        if (result.size() >= limit) break;
        result.push_back(tx);
    }
    
    return result;
}

inline PaymentProcessor::ProcessorStats PaymentProcessor::get_stats() const {
    return stats_;
}

} // namespace payment
} // namespace tiger

#endif // TIGER_PAYMENT_PROCESSOR_H
