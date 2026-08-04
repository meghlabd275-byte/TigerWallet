/**
 * TigerWallet KYC/AML Integration Service
 * Production-ready identity verification and anti-money laundering system
 */

#ifndef KYC_AML_SERVICE_HPP
#define KYC_AML_SERVICE_HPP

#include <string>
#include <vector>
#include <memory>
#include <unordered_map>
#include <functional>
#include <mutex>
#include <atomic>
#include <chrono>
#include <optional>

// JSON
#include <nlohmann/json.hpp>

using json = nlohmann::json;

namespace tigerwallet {
namespace kyc {

// ============================================================================
// CONSTANTS & CONFIGURATION
// ============================================================================

constexpr int KYC_TIMEOUT_MS = 30000;
constexpr int AML_SCREEN_TIMEOUT_MS = 15000;

// KYC Levels
enum class KYCLevel : uint8_t {
    NONE = 0,
    BASIC = 1,      
    STANDARD = 2,   
    ENHANCED = 3,   
    PREMIUM = 4     
};

// Verification Status
enum class VerificationStatus : uint8_t {
    NOT_STARTED = 0,
    PENDING = 1,
    IN_PROGRESS = 2,
    VERIFIED = 3,
    REJECTED = 4,
    EXPIRED = 5,
    NEEDS_REVIEW = 6
};

// Risk Level
enum class RiskLevel : uint8_t {
    LOW = 0,
    MEDIUM = 1,
    HIGH = 2,
    CRITICAL = 3
};

// Document Types
enum class DocumentType : uint8_t {
    PASSPORT = 0,
    NATIONAL_ID = 1,
    DRIVERS_LICENSE = 2,
    UTILITY_BILL = 3,
    BANK_STATEMENT = 4,
    SELFIE = 5
};

// ============================================================================
// DATA STRUCTURES
// ============================================================================

struct IdentityInfo {
    std::string id;
    std::string user_id;
    std::string master_wallet_id;
    std::string first_name;
    std::string last_name;
    std::string middle_name;
    std::string date_of_birth;
    std::string nationality;
    std::string email;
    std::string phone;
    std::string address;
    std::string city;
    std::string state;
    std::string postal_code;
    std::string country;
    std::string id_number;
    DocumentType id_document_type;
    std::string id_issuing_country;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
};

struct DocumentInfo {
    std::string id;
    std::string identity_id;
    DocumentType type;
    std::string document_number;
    std::string front_image_url;
    std::string back_image_url;
    std::string selfie_image_url;
    VerificationStatus status;
    bool is_verified;
    double confidence_score;
    std::string rejection_reason;
    std::chrono::system_clock::time_point uploaded_at;
    std::chrono::system_clock::time_point verified_at;
};

struct AMLScreeningResult {
    std::string id;
    std::string identity_id;
    RiskLevel overall_risk;
    double risk_score;
    bool has_sanctions_match;
    bool has_pep_match;
    bool has_adverse_media_match;
    bool has_watchlist_match;
    std::vector<std::string> matched_sanctions_lists;
    std::vector<std::string> matched_pep_lists;
    std::chrono::system_clock::time_point screened_at;
};

struct KYCVerificationResult {
    std::string id;
    std::string identity_id;
    KYCLevel level;
    VerificationStatus status;
    RiskLevel risk_level;
    bool email_verified;
    bool phone_verified;
    bool id_document_verified;
    bool selfie_verified;
    double identity_score;
    double document_score;
    double overall_score;
    std::string rejection_reason;
    std::chrono::system_clock::time_point initiated_at;
    std::chrono::system_clock::time_point completed_at;
};

struct ComplianceReport {
    std::string id;
    std::string identity_id;
    std::string master_wallet_id;
    std::string report_type;
    int total_transactions;
    double total_volume;
    std::string risk_rating;
    bool suspicious_activity_detected;
    std::vector<std::string> findings;
    std::chrono::system_clock::time_point generated_at;
};

// ============================================================================
// KYC/AML SERVICE
// ============================================================================

class KYCAMLService {
public:
    static KYCAMLService& getInstance();
    
    bool initialize(const std::string& config_json);
    void shutdown();
    
    // Identity Management
    std::string createIdentity(const IdentityInfo& identity);
    std::optional<IdentityInfo> getIdentity(const std::string& identity_id);
    std::optional<IdentityInfo> getIdentityByUserId(const std::string& user_id);
    bool updateIdentity(const IdentityInfo& identity);
    bool deleteIdentity(const std::string& identity_id);
    
    // Document Management
    std::string uploadDocument(const DocumentInfo& document);
    std::optional<DocumentInfo> getDocument(const std::string& document_id);
    KYCVerificationResult verifyDocument(const std::string& document_id);
    
    // KYC Verification
    KYCVerificationResult startVerification(const std::string& identity_id, KYCLevel target_level);
    KYCVerificationResult getVerificationStatus(const std::string& identity_id);
    std::vector<DocumentType> getRequiredDocuments(KYCLevel level);
    
    // AML Screening
    AMLScreeningResult screenIndividual(const std::string& identity_id);
    AMLScreeningResult screenEntity(const std::string& identity_id);
    std::optional<AMLScreeningResult> getAMLScreeningResult(const std::string& identity_id);
    
    // Compliance
    ComplianceReport generateComplianceReport(const std::string& master_wallet_id, 
                                             const std::string& period_start,
                                             const std::string& period_end);
    double getTransactionRiskScore(const std::string& identity_id, const std::string& amount,
                                   const std::string& currency);
    bool reportSuspiciousActivity(const std::string& identity_id, const std::string& description);

private:
    KYCAMLService() = default;
    ~KYCAMLService() = default;
    
    KYCAMLService(const KYCAMLService&) = delete;
    KYCAMLService& operator=(const KYCAMLService&) = delete;
    
    json config_;
    mutable std::mutex mutex_;
    std::atomic<bool> running_;
    
    std::unordered_map<std::string, std::shared_ptr<IdentityInfo>> identity_cache_;
    std::unordered_map<std::string, std::shared_ptr<AMLScreeningResult>> aml_cache_;
    
    std::atomic<uint64_t> total_verifications_{0};
    std::atomic<uint64_t> total_screens_{0};
    
    // Internal methods
    bool performIdentityVerification(const IdentityInfo& identity, KYCVerificationResult& result);
    bool performDocumentVerification(const DocumentInfo& document, KYCVerificationResult& result);
    bool performAMLScreening(const IdentityInfo& identity, AMLScreeningResult& result);
    
    double calculateRiskScore(const AMLScreeningResult& aml_result, const IdentityInfo& identity);
    std::string generateUUID();
    std::string hashData(const std::string& data);
    
    // Database operations (mock - in production use real DB)
    void saveIdentity(const IdentityInfo& identity);
    void saveDocument(const DocumentInfo& document);
    void saveVerificationResult(const KYCVerificationResult& result);
    void saveAMLScreeningResult(const AMLScreeningResult& result);
};

} // namespace kyc
} // namespace tigerwallet

#endif // KYC_AML_SERVICE_HPP
