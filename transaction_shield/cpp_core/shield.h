/**
 * TigerWallet Transaction Shield - Fraud Protection System
 * Enterprise-grade transaction protection with ML-based fraud detection
 */

#ifndef TIGER_TRANSACTION_SHIELD_H
#define TIGER_TRANSACTION_SHIELD_H

#include <array>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <memory>
#include <mutex>
#include <optional>
#include <shared_mutex>
#include <string>
#include <string_view>
#include <thread>
#include <unordered_map>
#include <unordered_set>
#include <vector>

namespace tiger {
namespace shield {

// ============================================================================
// Constants
// ============================================================================

constexpr uint64_t MAX_PROTECTION_AMOUNT = 100000000ULL;  // $10,000 in cents
constexpr uint64_t MIN_PROTECTION_AMOUNT = 10000ULL;    // $100 minimum
constexpr size_t MAX_RULES = 10000;
constexpr size_t MAX_WHITELIST = 100000;
constexpr size_t MAX_BLACKLIST = 100000;

// ============================================================================
// Types
// ============================================================================

enum class RiskLevel : uint8_t {
    NONE = 0,
    LOW = 1,
    MEDIUM = 2,
    HIGH = 3,
    CRITICAL = 4,
    BLOCKED = 5,
};

enum class TransactionType : uint8_t {
    TRANSFER = 0,
    SWAP = 1,
    STAKE = 2,
    UNSTAKE = 3,
    MINT = 4,
    BURN = 5,
    APPROVE = 6,
    CONTRACT_CALL = 7,
    NFT_TRANSFER = 8,
    BRIDGE = 9,
};

enum class ShieldStatus : uint8_t {
    INACTIVE = 0,
    ACTIVE = 1,
    SUSPENDED = 2,
    CLAIMED = 3,
    EXPIRED = 4,
};

enum class RuleType : uint8_t {
    AMOUNT_LIMIT = 0,
    VELOCITY_LIMIT = 1,
    TIME_RESTRICTION = 2,
    RECIPIENT_WHITELIST = 3,
    RECIPIENT_BLACKLIST = 4,
    TOKEN_BLACKLIST = 5,
    CHAIN_RESTRICTION = 6,
    DAPP_RESTRICTION = 7,
    PATTERN_DETECTION = 8,
    GEOLOCATION_CHECK = 9,
    DEVICE_FINGERPRINT = 10,
    BEHAVIORAL_ANALYSIS = 11,
};

// ============================================================================
// Data Structures
// ============================================================================

struct TransactionRequest {
    uint64_t request_id;
    uint32_t user_id;
    std::string from_address;
    std::string to_address;
    std::string token_address;
    uint64_t chain_id;
    uint64_t amount;
    uint64_t timestamp;
    TransactionType tx_type;
    std::string dapp_origin;
    std::string ip_address;
    std::string device_fingerprint;
    std::string user_agent;
    double latitude;
    double longitude;
    std::vector<std::string> metadata;
    
    TransactionRequest() : request_id(0), user_id(0), chain_id(1),
                         amount(0), timestamp(0), tx_type(TransactionType::TRANSFER),
                         latitude(0), longitude(0) {}
};

struct ShieldRule {
    uint64_t rule_id;
    RuleType rule_type;
    std::string name;
    std::string description;
    bool is_active;
    uint8_t priority;
    RiskLevel action_on_trigger;
    std::unordered_map<std::string, std::string> parameters;
    uint64_t created_at;
    uint64_t updated_at;
    
    ShieldRule() : rule_id(0), rule_type(RuleType::AMOUNT_LIMIT), is_active(true),
                   priority(128), action_on_trigger(RiskLevel::BLOCKED),
                   created_at(0), updated_at(0) {}
};

struct UserShield {
    uint32_t user_id;
    ShieldStatus status;
    uint64_t protection_limit;
    uint64_t current_covered_amount;
    uint64_t total_claims;
    uint64_t total_claimed_amount;
    uint64_t activated_at;
    uint64_t expires_at;
    bool auto_protect;
    uint8_t protection_level;  // 1-5
    
    UserShield() : user_id(0), status(ShieldStatus::INACTIVE),
                   protection_limit(0), current_covered_amount(0),
                   total_claims(0), total_claimed_amount(0),
                   activated_at(0), expires_at(0), auto_protect(false),
                   protection_level(1) {}
};

struct RiskAssessment {
    uint64_t request_id;
    RiskLevel risk_level;
    uint64_t risk_score;  // 0-10000
    std::vector<std::string> triggered_rules;
    std::string recommendation;
    bool requires_review;
    bool is_approved;
    uint64_t assessed_at;
    uint64_t reviewed_at;
    uint32_t reviewed_by;
    std::string review_notes;
    
    RiskAssessment() : request_id(0), risk_level(RiskLevel::NONE),
                      risk_score(0), requires_review(false), is_approved(false),
                      assessed_at(0), reviewed_at(0), reviewed_by(0) {}
};

struct Claim {
    uint64_t claim_id;
    uint32_t user_id;
    uint64_t request_id;
    uint64_t transaction_hash;
    uint64_t amount;
    std::string description;
    std::string status;  // pending, approved, rejected, paid
    uint64_t filed_at;
    uint64_t resolved_at;
    uint64_t paid_at;
    uint32_t resolved_by;
    
    Claim() : claim_id(0), user_id(0), request_id(0), transaction_hash(0),
              amount(0), filed_at(0), resolved_at(0), paid_at(0), resolved_by(0) {}
};

// ============================================================================
// Fraud Detection Engine
// ============================================================================

class FraudDetectionEngine {
private:
    // User behavior profiles
    struct UserProfile {
        uint32_t user_id;
        std::unordered_set<std::string> known_addresses;
        std::unordered_set<std::string> known_tokens;
        std::vector<std::string> recent_ips;
        std::vector<std::string> recent_devices;
        uint64_t avg_transaction_amount;
        uint64_t max_transaction_amount;
        uint32_t daily_transaction_count;
        uint64_t last_transaction_time;
        std::vector<std::string> common_dapps;
        
        UserProfile() : user_id(0), avg_transaction_amount(0),
                       max_transaction_amount(0), daily_transaction_count(0),
                       last_transaction_time(0) {}
    };
    
    std::unordered_map<uint32_t, UserProfile> user_profiles_;
    mutable std::shared_mutex profile_mutex_;
    
    // ML model weights (simplified)
    struct ModelWeights {
        double amount_weight;
        double velocity_weight;
        double recipient_weight;
        double time_weight;
        double device_weight;
        double geo_weight;
        double pattern_weight;
    } weights_;
    
    // Known scam patterns
    std::unordered_set<std::string> scam_addresses_;
    std::unordered_set<std::string> scam_tokens_;
    std::unordered_set<std::string> known_dapps_;
    
    // Statistics
    struct DetectionStats {
        std::atomic<uint64_t> total_scanned;
        std::atomic<uint64_t> total_flagged;
        std::atomic<uint64_t> total_blocked;
        std::atomic<uint64_t> false_positive_count;
        std::atomic<uint64_t> true_positive_count;
    } stats_;
    
public:
    FraudDetectionEngine();
    ~FraudDetectionEngine() = default;
    
    // Risk assessment
    RiskLevel assess_transaction(
        const TransactionRequest& request,
        const UserShield& shield,
        std::vector<std::string>& triggered_rules
    );
    
    // Profile management
    void update_user_profile(const TransactionRequest& request);
    std::optional<UserProfile> get_user_profile(uint32_t user_id);
    void clear_user_profile(uint32_t user_id);
    
    // Threat intelligence
    void add_scam_address(const std::string& address);
    void remove_scam_address(const std::string& address);
    bool is_scam_address(const std::string& address);
    void add_scam_token(const std::string& token);
    bool is_scam_token(const std::string& token);
    
    // Model updates
    void update_weights(const ModelWeights& weights);
    void train_on_result(const TransactionRequest& request, bool was_fraud);
    
    // Statistics
    DetectionStats get_stats() const;
    void reset_stats();
    
private:
    double calculate_amount_risk(const TransactionRequest& request);
    double calculate_velocity_risk(const TransactionRequest& request);
    double calculate_recipient_risk(const TransactionRequest& request);
    double calculate_time_risk(const TransactionRequest& request);
    double calculate_device_risk(const TransactionRequest& request);
    double calculate_geo_risk(const TransactionRequest& request);
    double calculate_pattern_risk(const TransactionRequest& request);
};

// ============================================================================
// Transaction Shield
// ============================================================================

class TransactionShield {
private:
    std::unique_ptr<FraudDetectionEngine> fraud_engine_;
    
    // Shield rules
    std::unordered_map<uint64_t, ShieldRule> rules_;
    std::atomic<uint64_t> next_rule_id_;
    
    // User shields
    std::unordered_map<uint32_t, UserShield> user_shields_;
    mutable std::shared_mutex shields_mutex_;
    
    // Claims
    std::unordered_map<uint64_t, Claim> claims_;
    std::atomic<uint64_t> next_claim_id_;
    
    // Risk assessments
    std::unordered_map<uint64_t, RiskAssessment> assessments_;
    mutable std::shared_mutex assessment_mutex_;
    
    // Statistics
    struct ShieldStats {
        std::atomic<uint64_t> total_transactions_scanned;
        std::atomic<uint64_t> transactions_approved;
        std::atomic<uint64_t> transactions_blocked;
        std::atomic<uint64_t> transactions_flagged;
        std::atomic<uint64_t> total_protection_claims;
        std::atomic<uint64_t> total_protection_paid;
    } stats_;
    
    mutable std::mutex rules_mutex_;
    
    // Configuration
    uint64_t default_protection_limit_;
    uint8_t default_protection_level_;
    bool auto_protect_enabled_;
    
public:
    TransactionShield();
    ~TransactionShield() = default;
    
    // Shield operations
    uint64_t activate_shield(
        uint32_t user_id,
        uint64_t protection_limit,
        uint8_t protection_level,
        uint64_t duration_days
    );
    
    bool deactivate_shield(uint32_t user_id);
    bool update_shield_limit(uint32_t user_id, uint64_t new_limit);
    
    std::optional<UserShield> get_user_shield(uint32_t user_id);
    
    // Transaction analysis
    RiskAssessment analyze_transaction(const TransactionRequest& request);
    
    // Rules management
    uint64_t create_rule(const ShieldRule& rule);
    bool update_rule(const ShieldRule& rule);
    bool delete_rule(uint64_t rule_id);
    bool toggle_rule(uint64_t rule_id, bool active);
    std::vector<ShieldRule> get_rules(RuleType type = RuleType::AMOUNT_LIMIT);
    
    // Claims
    uint64_t file_claim(
        uint32_t user_id,
        uint64_t request_id,
        uint64_t transaction_hash,
        uint64_t amount,
        const std::string& description
    );
    
    bool process_claim(
        uint64_t claim_id,
        uint32_t reviewer_id,
        bool approved,
        const std::string& notes
    );
    
    bool pay_claim(uint64_t claim_id, uint64_t transaction_hash);
    
    std::vector<Claim> get_user_claims(uint32_t user_id);
    std::optional<Claim> get_claim(uint64_t claim_id);
    
    // Statistics
    ShieldStats get_stats() const;
    
    // Configuration
    void set_default_protection(uint64_t limit, uint8_t level);
    void set_auto_protect(bool enabled);
};

// ============================================================================
// Rule Engine
// ============================================================================

class RuleEngine {
private:
    std::vector<ShieldRule> active_rules_;
    mutable std::shared_mutex mutex_;
    
public:
    RuleEngine();
    ~RuleEngine() = default;
    
    void add_rule(const ShieldRule& rule);
    void remove_rule(uint64_t rule_id);
    void update_rule(const ShieldRule& rule);
    
    std::vector<ShieldRule> evaluate(const TransactionRequest& request);
    
    std::vector<ShieldRule> get_rules_by_type(RuleType type);
    std::vector<ShieldRule> get_all_active_rules();
    
    bool matches_amount_rule(const TransactionRequest& request, const ShieldRule& rule);
    bool matches_velocity_rule(const TransactionRequest& request, const ShieldRule& rule);
    bool matches_whitelist_rule(const TransactionRequest& request, const ShieldRule& rule);
    bool matches_blacklist_rule(const TransactionRequest& request, const ShieldRule& rule);
    bool matches_time_rule(const TransactionRequest& request, const ShieldRule& rule);
    bool matches_geo_rule(const TransactionRequest& request, const ShieldRule& rule);
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline FraudDetectionEngine::FraudDetectionEngine() {
    // Default weights
    weights_ = {
        .amount_weight = 0.25,
        .velocity_weight = 0.20,
        .recipient_weight = 0.20,
        .time_weight = 0.10,
        .device_weight = 0.10,
        .geo_weight = 0.10,
        .pattern_weight = 0.05,
    };
    
    // Initialize with known scam addresses (simplified)
    scam_addresses_.insert("0x0000000000000000000000000000000000000000000");
}

inline RiskLevel FraudDetectionEngine::assess_transaction(
    const TransactionRequest& request,
    const UserShield& shield,
    std::vector<std::string>& triggered_rules
) {
    double total_risk = 0.0;
    
    // Calculate individual risk factors
    double amount_risk = calculate_amount_risk(request);
    double velocity_risk = calculate_velocity_risk(request);
    double recipient_risk = calculate_recipient_risk(request);
    double time_risk = calculate_time_risk(request);
    double device_risk = calculate_device_risk(request);
    double geo_risk = calculate_geo_risk(request);
    double pattern_risk = calculate_pattern_risk(request);
    
    // Weighted sum
    total_risk = amount_risk * weights_.amount_weight +
                 velocity_risk * weights_.velocity_weight +
                 recipient_risk * weights_.recipient_weight +
                 time_risk * weights_.time_weight +
                 device_risk * weights_.device_weight +
                 geo_risk * weights_.geo_weight +
                 pattern_risk * weights_.pattern_weight;
    
    // Check against user shield
    if (shield.status == ShieldStatus::ACTIVE && 
        request.amount > shield.protection_limit) {
        total_risk += 0.5;  // Increase risk for amounts above protection
    }
    
    // Update stats
    stats_.total_scanned++;
    
    // Map to risk level
    if (total_risk >= 0.9) {
        stats_.total_blocked++;
        return RiskLevel::BLOCKED;
    } else if (total_risk >= 0.7) {
        stats_.total_flagged++;
        return RiskLevel::CRITICAL;
    } else if (total_risk >= 0.5) {
        stats_.total_flagged++;
        return RiskLevel::HIGH;
    } else if (total_risk >= 0.3) {
        return RiskLevel::MEDIUM;
    } else if (total_risk >= 0.1) {
        return RiskLevel::LOW;
    }
    
    stats_.transactions_approved++;
    return RiskLevel::NONE;
}

inline double FraudDetectionEngine::calculate_amount_risk(const TransactionRequest& request) {
    // Check if amount is unusually large
    auto profile = get_user_profile(request.user_id);
    if (profile) {
        if (request.amount > profile->max_transaction_amount * 10) {
            return 0.9;
        } else if (request.amount > profile->max_transaction_amount * 5) {
            return 0.6;
        } else if (request.amount > profile->max_transaction_amount * 2) {
            return 0.3;
        }
    }
    
    // Default thresholds
    if (request.amount > 1000000000ULL) return 0.8;  // > $10,000
    if (request.amount > 100000000ULL) return 0.5;     // > $1,000
    if (request.amount > 10000000ULL) return 0.2;      // > $100
    
    return 0.0;
}

inline double FraudDetectionEngine::calculate_velocity_risk(const TransactionRequest& request) {
    auto profile = get_user_profile(request.user_id);
    if (!profile) return 0.3;
    
    // Check transaction frequency
    auto now = std::chrono::system_clock::now().time_since_epoch().count();
    auto time_diff = (now - profile->last_transaction_time) / 1000000000; // seconds
    
    if (time_diff < 1) return 0.9;  // Multiple in 1 second
    if (time_diff < 5) return 0.7;   // Multiple in 5 seconds
    if (time_diff < 30) return 0.5; // Multiple in 30 seconds
    
    return 0.0;
}

inline double FraudDetectionEngine::calculate_recipient_risk(const TransactionRequest& request) {
    // Check against blacklist
    if (is_scam_address(request.to_address)) {
        return 1.0;
    }
    
    // Check user profile
    auto profile = get_user_profile(request.user_id);
    if (profile) {
        if (profile->known_addresses.count(request.to_address) == 0) {
            return 0.5;  // New recipient
        }
    }
    
    return 0.0;
}

inline double FraudDetectionEngine::calculate_time_risk(const TransactionRequest& request) {
    auto time = std::localtime((time_t*)&request.timestamp);
    int hour = time->tm_hour;
    
    // Unusual hours (late night / early morning)
    if (hour >= 0 && hour < 6) return 0.5;
    
    return 0.0;
}

inline double FraudDetectionEngine::calculate_device_risk(const TransactionRequest& request) {
    if (request.device_fingerprint.empty()) return 0.3;
    
    auto profile = get_user_profile(request.user_id);
    if (profile && profile->recent_devices.empty()) {
        return 0.2;
    }
    
    return 0.0;
}

inline double FraudDetectionEngine::calculate_geo_risk(const TransactionRequest& request) {
    // Check for impossible travel (simplified)
    // In production, this would check against previous locations
    
    // High-risk coordinates (simplified check)
    if (request.latitude == 0 && request.longitude == 0) {
        return 0.2;
    }
    
    return 0.0;
}

inline double FraudDetectionEngine::calculate_pattern_risk(const TransactionRequest& request) {
    // Check for known scam tokens
    if (is_scam_token(request.token_address)) {
        return 1.0;
    }
    
    return 0.0;
}

inline void FraudDetectionEngine::update_user_profile(const TransactionRequest& request) {
    std::unique_lock<std::shared_mutex> lock(profile_mutex_);
    
    auto& profile = user_profiles_[request.user_id];
    profile.user_id = request.user_id;
    
    // Update known addresses
    profile.known_addresses.insert(request.to_address);
    profile.known_tokens.insert(request.token_address);
    
    // Update transaction amounts
    if (request.amount > profile.max_transaction_amount) {
        profile.max_transaction_amount = request.amount;
    }
    
    // Update average
    profile.daily_transaction_count++;
    profile.last_transaction_time = request.timestamp;
    
    // Update IP and device
    if (!request.ip_address.empty()) {
        profile.recent_ips.push_back(request.ip_address);
        if (profile.recent_ips.size() > 10) {
            profile.recent_ips.erase(profile.recent_ips.begin());
        }
    }
    
    if (!request.device_fingerprint.empty()) {
        profile.recent_devices.push_back(request.device_fingerprint);
        if (profile.recent_devices.size() > 5) {
            profile.recent_devices.erase(profile.recent_devices.begin());
        }
    }
}

// TransactionShield inline implementations
inline TransactionShield::TransactionShield()
    : next_rule_id_(1), next_claim_id_(1),
      default_protection_limit_(MAX_PROTECTION_AMOUNT),
      default_protection_level_(3), auto_protect_enabled_(true) {
    fraud_engine_ = std::make_unique<FraudDetectionEngine>();
}

inline uint64_t TransactionShield::activate_shield(
    uint32_t user_id,
    uint64_t protection_limit,
    uint8_t protection_level,
    uint64_t duration_days
) {
    std::unique_lock<std::shared_mutex> lock(shields_mutex_);
    
    UserShield& shield = user_shields_[user_id];
    shield.user_id = user_id;
    shield.status = ShieldStatus::ACTIVE;
    shield.protection_limit = std::min(protection_limit, MAX_PROTECTION_AMOUNT);
    shield.current_covered_amount = 0;
    shield.total_claims = 0;
    shield.total_claimed_amount = 0;
    shield.activated_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    shield.expires_at = shield.activated_at + duration_days * 24 * 60 * 60 * 1000;
    shield.auto_protect = auto_protect_enabled_;
    shield.protection_level = protection_level;
    
    return shield.activated_at;
}

inline RiskAssessment TransactionShield::analyze_transaction(const TransactionRequest& request) {
    RiskAssessment assessment;
    assessment.request_id = request.request_id;
    assessment.assessed_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    // Get user shield
    std::optional<UserShield> shield_opt;
    {
        std::shared_lock<std::shared_mutex> lock(shields_mutex_);
        auto it = user_shields_.find(request.user_id);
        if (it != user_shields_.end()) {
            shield_opt = it->second;
        }
    }
    
    UserShield shield = shield_opt.value_or(UserShield());
    
    // Run fraud detection
    assessment.risk_level = fraud_engine_->assess_transaction(
        request, shield, assessment.triggered_rules
    );
    
    // Calculate risk score (0-10000)
    assessment.risk_score = static_cast<uint64_t>(assessment.risk_level) * 2000;
    
    // Determine recommendation
    if (assessment.risk_level >= RiskLevel::CRITICAL) {
        assessment.recommendation = "BLOCK";
        assessment.is_approved = false;
    } else if (assessment.risk_level >= RiskLevel::HIGH) {
        assessment.recommendation = "REVIEW";
        assessment.requires_review = true;
    } else if (assessment.risk_level >= RiskLevel::MEDIUM) {
        assessment.recommendation = "WARN";
        assessment.is_approved = true;
    } else {
        assessment.recommendation = "APPROVE";
        assessment.is_approved = true;
    }
    
    // Update stats
    stats_.total_transactions_scanned++;
    if (assessment.is_approved) {
        stats_.transactions_approved++;
    } else {
        stats_.transactions_blocked++;
    }
    
    // Store assessment
    {
        std::unique_lock<std::shared_mutex> lock(assessment_mutex_);
        assessments_[request.request_id] = assessment;
    }
    
    return assessment;
}

} // namespace shield
} // namespace tiger

#endif // TIGER_TRANSACTION_SHIELD_H
