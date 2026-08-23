/**
 * TigerWallet White Label SDK
 * Ultra-low latency C++ SDK for white label customization and management
 * 
 * @version 1.0.0
 * @date 2024
 */

#ifndef TIGERWALLET_WL_SDK_HPP
#define TIGERWALLET_WL_SDK_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <chrono>
#include <thread>
#include <mutex>
#include <atomic>
#include <queue>
#include <future>
#include <variant>
#include <any>

// Forward declarations
namespace tigerwallet {
namespace wl {

// ============================================================================
// Configuration Structures
// ============================================================================

/**
 * Branding configuration for white label
 */
struct BrandingConfig {
    std::string primary_color;
    std::string secondary_color;
    std::string accent_color;
    std::string background_color;
    std::string text_color;
    std::optional<std::string> logo_url;
    std::optional<std::string> favicon_url;
    std::optional<std::string> font_family;
    std::optional<std::string> custom_css;
    std::string app_name;
    std::optional<std::string> app_description;
    std::optional<std::string> slogan;
    std::string theme; // "light", "dark", "auto"
    
    BrandingConfig() {
        primary_color = "#6366F1";
        secondary_color = "#8B5CF6";
        accent_color = "#EC4899";
        background_color = "#FFFFFF";
        text_color = "#1F2937";
        app_name = "TigerWallet";
        theme = "light";
    }
};

/**
 * Feature configuration
 */
struct FeatureConfig {
    bool swap_enabled = true;
    bool staking_enabled = true;
    bool nft_enabled = true;
    bool bridge_enabled = true;
    bool defi_enabled = true;
    bool analytics_enabled = true;
    bool kyc_enabled = false;
    bool multi_sig_enabled = false;
    bool privacy_enabled = false;
    bool hardware_wallet_enabled = true;
    bool session_keys_enabled = true;
    bool paymaster_enabled = false;
    bool gas_optimization = true;
    bool mev_protection = false;
    bool price_alerts = true;
    bool tax_integration = false;
    
    std::map<std::string, bool> custom_features;
};

/**
 * Security configuration
 */
struct SecurityConfig {
    bool biometric_enabled = true;
    bool pin_enabled = true;
    bool two_factor_enabled = false;
    bool passkey_enabled = false;
    std::vector<std::string> whitelisted_ips;
    int max_login_attempts = 5;
    int session_timeout = 3600; // seconds
    bool force_2fa = false;
    bool ip_restriction = false;
};

/**
 * API configuration
 */
struct APIConfig {
    bool api_key_required = true;
    int rate_limit = 1000;
    std::optional<std::string> webhook_url;
    std::vector<std::string> webhook_events;
    std::string api_version = "v1";
};

/**
 * UI configuration
 */
struct UIConfig {
    std::string theme = "light";
    std::string language = "en";
    std::string direction = "ltr"; // "ltr" or "rtl"
    std::string layout = "default";
    bool show_navigation = true;
    bool show_footer = true;
    bool compact_mode = false;
    std::vector<std::string> enabled_widgets;
};

/**
 * Complete white label configuration
 */
struct WhiteLabelConfig {
    std::string id;
    std::string name;
    BrandingConfig branding;
    FeatureConfig features;
    SecurityConfig security;
    APIConfig api;
    UIConfig ui;
    std::vector<std::string> supported_chains;
    std::map<std::string, std::string> metadata;
    
    WhiteLabelConfig() {
        id = "";
        name = "TigerWallet";
    }
};

// ============================================================================
// Response Types
// ============================================================================

/**
 * API response wrapper
 */
template<typename T>
struct APIResponse {
    bool success;
    std::optional<T> data;
    std::optional<std::string> error;
    int status_code;
    std::chrono::milliseconds latency;
    
    static APIResponse<T> Success(T&& data, int status = 200) {
        return APIResponse<T>{
            .success = true,
            .data = std::forward<T>(data),
            .error = std::nullopt,
            .status_code = status,
            .latency = std::chrono::milliseconds(0)
        };
    }
    
    static APIResponse<T> Error(const std::string& err, int status = 500) {
        return APIResponse<T>{
            .success = false,
            .data = std::nullopt,
            .error = err,
            .status_code = status,
            .latency = std::chrono::milliseconds(0)
        };
    }
};

/**
 * White label client information
 */
struct WhiteLabelClient {
    std::string id;
    std::string name;
    std::string domain;
    std::string status; // "active", "suspended", "pending"
    std::string created_at;
    std::string updated_at;
    WhiteLabelConfig config;
    int user_count = 0;
    int transaction_count = 0;
    double revenue = 0.0;
};

/**
 * User information
 */
struct UserInfo {
    std::string id;
    std::string email;
    std::string username;
    std::string role; // "admin", "manager", "support", "user"
    std::string status;
    std::string created_at;
    std::string last_login;
    bool two_factor_enabled = false;
    std::map<std::string, std::string> metadata;
};

/**
 * Transaction information
 */
struct Transaction {
    std::string id;
    std::string user_id;
    std::string type; // "swap", "transfer", "stake", "unstake"
    std::string status; // "pending", "completed", "failed"
    std::string amount;
    std::string currency;
    std::string from_address;
    std::string to_address;
    std::string hash;
    std::string timestamp;
    std::map<std::string, std::string> metadata;
};

/**
 * Analytics data
 */
struct AnalyticsData {
    int total_users = 0;
    int active_users = 0;
    int new_users_today = 0;
    double total_volume = 0.0;
    double total_revenue = 0.0;
    int total_transactions = 0;
    double avg_transaction_size = 0.0;
    std::map<std::string, double> volume_by_chain;
    std::map<std::string, int> users_by_country;
    std::map<std::string, int> transactions_by_type;
};

// ============================================================================
// SDK Core
// ============================================================================

/**
 * HTTP client for API calls
 */
class HTTPClient {
public:
    HTTPClient(const std::string& base_url, const std::string& api_key);
    ~HTTPClient();
    
    template<typename T>
    APIResponse<T> get(const std::string& endpoint);
    
    template<typename T, typename R>
    APIResponse<R> post(const std::string& endpoint, const T& body);
    
    template<typename T, typename R>
    APIResponse<R> put(const std::string& endpoint, const T& body);
    
    APIResponse<void> delete_(const std::string& endpoint);
    
    void set_timeout(int milliseconds);
    void set_retries(int count);
    
private:
    std::string base_url_;
    std::string api_key_;
    int timeout_ms_ = 30000;
    int max_retries_ = 3;
    
    // Internal implementation
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

/**
 * White Label SDK
 */
class WhiteLabelSDK {
public:
    /**
     * Constructor with configuration
     */
    WhiteLabelSDK(
        const std::string& base_url,
        const std::string& api_key,
        const std::string& white_label_id = ""
    );
    
    /**
     * Destructor
     */
    ~WhiteLabelSDK();
    
    // =========================================================================
    // Configuration Management
    // =========================================================================
    
    /**
     * Get current white label configuration
     */
    APIResponse<WhiteLabelConfig> get_config();
    
    /**
     * Update white label configuration
     */
    APIResponse<WhiteLabelConfig> update_config(const WhiteLabelConfig& config);
    
    /**
     * Reset configuration to defaults
     */
    APIResponse<void> reset_config();
    
    /**
     * Validate configuration
     */
    APIResponse<bool> validate_config(const WhiteLabelConfig& config);
    
    // =========================================================================
    // Branding Management
    // =========================================================================
    
    /**
     * Get branding configuration
     */
    APIResponse<BrandingConfig> get_branding();
    
    /**
     * Update branding configuration
     */
    APIResponse<BrandingConfig> update_branding(const BrandingConfig& branding);
    
    /**
     * Upload logo image
     */
    APIResponse<std::string> upload_logo(const std::vector<uint8_t>& image_data, const std::string& filename);
    
    /**
     * Upload favicon
     */
    APIResponse<std::string> upload_favicon(const std::vector<uint8_t>& image_data, const std::string& filename);
    
    /**
     * Generate theme CSS
     */
    APIResponse<std::string> generate_theme_css(const std::string& theme);
    
    // =========================================================================
    // Feature Management
    // =========================================================================
    
    /**
     * Get feature configuration
     */
    APIResponse<FeatureConfig> get_features();
    
    /**
     * Update feature configuration
     */
    APIResponse<FeatureConfig> update_features(const FeatureConfig& features);
    
    /**
     * Enable a specific feature
     */
    APIResponse<void> enable_feature(const std::string& feature_name);
    
    /**
     * Disable a specific feature
     */
    APIResponse<void> disable_feature(const std::string& feature_name);
    
    /**
     * Get list of available features
     */
    APIResponse<std::vector<std::string>> get_available_features();
    
    // =========================================================================
    // User Management
    // =========================================================================
    
    /**
     * Get user by ID
     */
    APIResponse<UserInfo> get_user(const std::string& user_id);
    
    /**
     * List users with pagination
     */
    APIResponse<std::vector<UserInfo>> list_users(int page = 1, int limit = 50);
    
    /**
     * Create new user
     */
    APIResponse<UserInfo> create_user(const UserInfo& user);
    
    /**
     * Update user
     */
    APIResponse<UserInfo> update_user(const std::string& user_id, const UserInfo& user);
    
    /**
     * Delete user
     */
    APIResponse<void> delete_user(const std::string& user_id);
    
    /**
     * Suspend user
     */
    APIResponse<void> suspend_user(const std::string& user_id);
    
    /**
     * Activate user
     */
    APIResponse<void> activate_user(const std::string& user_id);
    
    /**
     * Search users
     */
    APIResponse<std::vector<UserInfo>> search_users(const std::string& query);
    
    // =========================================================================
    // Transaction Management
    // =========================================================================
    
    /**
     * Get transaction by ID
     */
    APIResponse<Transaction> get_transaction(const std::string& transaction_id);
    
    /**
     * List transactions with filters
     */
    APIResponse<std::vector<Transaction>> list_transactions(
        const std::string& user_id = "",
        const std::string& type = "",
        const std::string& status = "",
        int page = 1,
        int limit = 50
    );
    
    /**
     * Get transaction statistics
     */
    APIResponse<AnalyticsData> get_transaction_stats(
        const std::string& start_date = "",
        const std::string& end_date = ""
    );
    
    // =========================================================================
    // Analytics
    // =========================================================================
    
    /**
     * Get analytics data
     */
    APIResponse<AnalyticsData> get_analytics(
        const std::string& start_date = "",
        const std::string& end_date = ""
    );
    
    /**
     * Get real-time metrics
     */
    APIResponse<std::map<std::string, double>> get_realtime_metrics();
    
    /**
     * Export analytics data
     */
    APIResponse<std::string> export_analytics(
        const std::string& format = "json",
        const std::string& start_date = "",
        const std::string& end_date = ""
    );
    
    // =========================================================================
    // API Key Management
    // =========================================================================
    
    /**
     * Generate new API key
     */
    APIResponse<std::string> generate_api_key(const std::string& name, const std::vector<std::string>& permissions);
    
    /**
     * Revoke API key
     */
    APIResponse<void> revoke_api_key(const std::string& key_id);
    
    /**
     * List API keys
     */
    APIResponse<std::vector<std::map<std::string, std::string>>> list_api_keys();
    
    // =========================================================================
    // Webhook Management
    // =========================================================================
    
    /**
     * Register webhook
     */
    APIResponse<std::string> register_webhook(
        const std::string& url,
        const std::vector<std::string>& events
    );
    
    /**
     * Unregister webhook
     */
    APIResponse<void> unregister_webhook(const std::string& webhook_id);
    
    /**
     * List webhooks
     */
    APIResponse<std::vector<std::map<std::string, std::string>>> list_webhooks();
    
    // =========================================================================
    // Utility Methods
    // =========================================================================
    
    /**
     * Test API connection
     */
    APIResponse<bool> test_connection();
    
    /**
     * Get SDK version
     */
    std::string get_version() const;
    
    /**
     * Get white label ID
     */
    std::string get_white_label_id() const;
    
    /**
     * Set custom headers
     */
    void set_custom_header(const std::string& key, const std::string& value);
    
    /**
     * Enable/disable caching
     */
    void set_caching_enabled(bool enabled);
    
    /**
     * Set request timeout
     */
    void set_timeout(int milliseconds);

private:
    // Private implementation
    class Impl;
    std::unique_ptr<Impl> pimpl_;
    
    // Prevent copying
    WhiteLabelSDK(const WhiteLabelSDK&) = delete;
    WhiteLabelSDK& operator=(const WhiteLabelSDK&) = delete;
};

// ============================================================================
// Event System
// ============================================================================

/**
 * Event types for webhooks and callbacks
 */
enum class EventType {
    UserCreated,
    UserUpdated,
    UserDeleted,
    UserSuspended,
    TransactionCreated,
    TransactionCompleted,
    TransactionFailed,
    DepositReceived,
    WithdrawalProcessed,
    KYCApproved,
    KYCrejected,
    SecurityAlert,
    ConfigUpdated
};

/**
 * Event data structure
 */
struct Event {
    EventType type;
    std::string white_label_id;
    std::string resource_id;
    std::string resource_type;
    std::map<std::string, std::string> data;
    std::string timestamp;
    std::string signature;
};

/**
 * Event handler callback type
 */
using EventCallback = std::function<void(const Event&)>;

/**
 * Event manager for handling webhooks and events
 */
class EventManager {
public:
    EventManager();
    ~EventManager();
    
    /**
     * Subscribe to events
     */
    void subscribe(EventType type, EventCallback callback);
    
    /**
     * Unsubscribe from events
     */
    void unsubscribe(EventType type);
    
    /**
     * Process incoming event
     */
    void process_event(const Event& event);
    
    /**
     * Verify event signature
     */
    bool verify_signature(const Event& event, const std::string& secret);

private:
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

// ============================================================================
// Builder Pattern for Configuration
// ============================================================================

/**
 * Configuration builder for easy setup
 */
class WhiteLabelConfigBuilder {
public:
    WhiteLabelConfigBuilder();
    
    WhiteLabelConfigBuilder& set_name(const std::string& name);
    WhiteLabelConfigBuilder& set_primary_color(const std::string& color);
    WhiteLabelConfigBuilder& set_secondary_color(const std::string& color);
    WhiteLabelConfigBuilder& set_theme(const std::string& theme);
    WhiteLabelConfigBuilder& enable_feature(const std::string& feature);
    WhiteLabelConfigBuilder& disable_feature(const std::string& feature);
    WhiteLabelConfigBuilder& enable_2fa();
    WhiteLabelConfigBuilder& enable_biometric();
    WhiteLabelConfigBuilder& set_supported_chains(const std::vector<std::string>& chains);
    WhiteLabelConfigBuilder& add_metadata(const std::string& key, const std::string& value);
    
    WhiteLabelConfig build();

private:
    WhiteLabelConfig config_;
};

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Generate random ID
 */
std::string generate_id();

/**
 * Hash string with SHA-256
 */
std::string sha256(const std::string& input);

/**
 * Base64 encode
 */
std::string base64_encode(const std::vector<uint8_t>& data);

/**
 * Base64 decode
 */
std::vector<uint8_t> base64_decode(const std::string& encoded);

/**
 * URL encode
 */
std::string url_encode(const std::string& value);

/**
 * Parse JSON string to config
 */
WhiteLabelConfig parse_config(const std::string& json);

/**
 * Serialize config to JSON
 */
std::string serialize_config(const WhiteLabelConfig& config);

} // namespace wl
} // namespace tigerwallet

#endif // TIGERWALLET_WL_SDK_HPP
