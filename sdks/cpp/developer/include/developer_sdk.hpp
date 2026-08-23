/**
 * TigerWallet Developer SDK
 * Ultra-low latency C++ SDK for developers - Paymaster, Session Keys, Agentic Wallet, Widgets
 * 
 * @version 1.0.0
 * @date 2024
 */

#ifndef TIGERWALLET_DEVELOPER_SDK_HPP
#define TIGERWALLET_DEVELOPER_SDK_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <chrono>
#include <variant>

namespace tigerwallet {
namespace sdk {

// ============================================================================
// Paymaster Types
// ============================================================================

/**
 * Paymaster configuration
 */
struct PaymasterConfig {
    std::string entry_point;
    std::string paymaster_address;
    std::string paymaster_url;
    std::string api_key;
    std::vector<std::string> supported_chains;
    bool sponsorship_enabled = true;
    double max_sponsorship_usd = 100.0;
};

/**
 * User operation for sponsorship
 */
struct UserOperation {
    std::string sender;
    uint64_t nonce;
    std::string init_code;
    std::string call_data;
    uint64_t call_gas_limit;
    uint64_t verification_gas_limit;
    uint64_t pre_verification_gas;
    std::string max_fee_per_gas;
    std::string max_priority_fee_per_gas;
    std::string signature;
    std::string hash;
};

/**
 * Sponsorship request
 */
struct SponsorshipRequest {
    std::string user_operation_hash;
    UserOperation operation;
    std::string chain_id;
    std::string dapp_address;
    std::bool sponsorship_eligible;
    std::string paymaster_data;
};

/**
 * Sponsorship response
 */
struct SponsorshipResponse {
    std::string paymaster_and_data;
    std::string pre_verification_gas;
    std::string verification_gas_limit;
    std::string call_gas_limit;
    bool approved;
    std::string reason;
};

// ============================================================================
// Session Key Types
// ============================================================================

/**
 * Session key permissions
 */
struct SessionKeyPermissions {
    bool allow_swap = false;
    bool allow_stake = false;
    bool allow_bridge = false;
    bool allow_nft = false;
    bool allow_transfer = false;
    double max_daily_spend = 0;
    std::vector<std::string> allowed_tokens;
    std::vector<std::string> allowedContracts;
    std::chrono::seconds validity_period = std::chrono::hours(24);
};

/**
 * Session key
 */
struct SessionKey {
    std::string id;
    std::string address;
    std::string public_key;
    SessionKeyPermissions permissions;
    std::string signature;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point expires_at;
    bool is_active = true;
};

/**
 * Session key request
 */
struct SessionKeyRequest {
    std::string user_address;
    SessionKeyPermissions permissions;
    std::string signature;
    std::chrono::seconds duration;
};

// ============================================================================
// Agentic Wallet Types
// ============================================================================

/**
 * Agent instruction
 */
struct AgentInstruction {
    std::string id;
    std::string type; // "swap", "transfer", "stake", "bridge", "custom"
    std::map<std::string, std::string> params;
    std::string condition; // "immediate", "price_based", "time_based"
    std::string condition_value;
    int priority = 0;
};

/**
 * Agent task
 */
struct AgentTask {
    std::string id;
    std::string agent_id;
    std::vector<AgentInstruction> instructions;
    std::string status; // pending, running, completed, failed, cancelled
    std::string result;
    double total_gas_used = 0;
    double total_value = 0;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point completed_at;
};

/**
 * Agent wallet config
 */
struct AgentWalletConfig {
    std::string agent_address;
    std::vector<std::string> allowed_actions;
    double max_daily_spend = 10000;
    double max_single_transaction = 1000;
    bool require_confirmation = false;
    std::vector<std::string> notification_urls;
    bool ai_enabled = true;
    std::string ai_model = "gpt-4";
};

/**
 * Natural language transaction request
 */
struct NLTransactionRequest {
    std::string natural_language;
    std::string from_address;
    std::string to_address;
    std::string amount;
    std::string token;
    std::string chain;
    double estimated_gas;
    double estimated_value;
    std::string parsed_intent;
    std::map<std::string, std::string> decoded_params;
};

// ============================================================================
// Widget Types
// ============================================================================

/**
 * Widget configuration
 */
struct WidgetConfig {
    std::string widget_id;
    std::string type; // "swap", "send", "buy", "stake", "bridge", "nft"
    std::string theme = "auto";
    std::string language = "en";
    std::map<std::string, std::string> customization;
    std::vector<std::string> enabled_chains;
    std::vector<std::string> enabled_tokens;
    bool show_wallet_connection = true;
    bool modal_mode = false;
    std::string position = "bottom-right";
};

/**
 * Widget event
 */
struct WidgetEvent {
    std::string event_type;
    std::map<std::string, std::string> data;
    std::string timestamp;
};

// ============================================================================
// Paymaster SDK
// ============================================================================

class PaymasterSDK {
public:
    PaymasterSDK(const PaymasterConfig& config);
    ~PaymasterSDK();
    
    // Sponsorship
    SponsorshipResponse request_sponsorship(const SponsorshipRequest& request);
    bool verify_sponsorship(const std::string& user_op_hash);
    std::string get_paymaster_data(const UserOperation& op);
    
    // Configuration
    void update_config(const PaymasterConfig& config);
    PaymasterConfig get_config() const;
    
    // Chain management
    bool is_chain_supported(const std::string& chain_id);
    std::vector<std::string> get_supported_chains();
    
    // Estimation
    uint64_t estimate_gas(const UserOperation& op);
    double estimate_sponsorship_cost(const UserOperation& op);

private:
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

// ============================================================================
// Session Key SDK
// ============================================================================

class SessionKeySDK {
public:
    SessionKeySDK(const std::string& rpc_url, const std::string& entry_point);
    ~SessionKeySDK();
    
    // Key management
    SessionKey create_session_key(const SessionKeyRequest& request);
    SessionKey get_session_key(const std::string& key_id);
    std::vector<SessionKey> list_session_keys(const std::string& user_address);
    bool revoke_session_key(const std::string& key_id);
    bool extend_session_key(const std::string& key_id, std::chrono::seconds duration);
    
    // Key usage
    bool is_key_valid(const std::string& key_id);
    bool has_permission(const std::string& key_id, const std::string& action);
    bool check_spending_limit(const std::string& key_id, double amount);
    
    // Signing
    std::string sign_user_operation(const std::string& key_id, const UserOperation& op);
    std::string sign_message(const std::string& key_id, const std::string& message);

private:
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

// ============================================================================
// Agentic Wallet SDK
// ============================================================================

class AgenticWalletSDK {
public:
    AgenticWalletSDK(const std::string& owner_address, const AgentWalletConfig& config);
    ~AgenticWalletSDK();
    
    // Agent management
    std::string deploy_agent(const AgentWalletConfig& config);
    AgentWalletConfig get_agent_config();
    void update_agent_config(const AgentWalletConfig& config);
    
    // Task management
    std::string submit_task(const AgentTask& task);
    AgentTask get_task(const std::string& task_id);
    std::vector<AgentTask> list_tasks(const std::string& status = "");
    bool cancel_task(const std::string& task_id);
    
    // Natural language processing
    NLTransactionRequest parse_natural_language(const std::string& text);
    std::string execute_nl_transaction(const std::string& text);
    
    // AI integration
    std::string ask_ai(const std::string& question);
    double get_recommended_gas_price();
    std::map<std::string, double> get_token_prices();

private:
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

// ============================================================================
// Widget SDK
// ============================================================================

class WidgetSDK {
public:
    WidgetSDK(const std::string& api_key);
    ~WidgetSDK();
    
    // Widget creation
    std::string create_widget(const WidgetConfig& config);
    WidgetConfig get_widget_config(const std::string& widget_id);
    void update_widget_config(const std::string& widget_id, const WidgetConfig& config);
    bool delete_widget(const std::string& widget_id);
    
    // Widget events
    using EventCallback = std::function<void(const WidgetEvent&)>;
    void subscribe_to_events(const std::string& widget_id, EventCallback callback);
    void unsubscribe_from_events(const std::string& widget_id);
    
    // Embedded widget
    std::string generate_embed_code(const std::string& widget_id);
    std::string generate_iframe_code(const std::string& widget_id, int width, int height);

private:
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

// ============================================================================
// Complete Developer SDK
// ============================================================================

/**
 * Complete Developer SDK combining all features
 */
class DeveloperSDK {
public:
    DeveloperSDK(
        const std::string& api_key,
        const std::string& rpc_url = "",
        const std::string& entry_point = "0x5FF137D4a0aC116f8B83B0b21b0bF9D5cF5C5e7"
    );
    ~DeveloperSDK();
    
    // Get individual SDKs
    PaymasterSDK& paymaster();
    SessionKeySDK& session_keys();
    AgenticWalletSDK& agentic();
    WidgetSDK& widgets();
    
    // Utility
    std::string get_sdk_version() const;
    bool is_initialized() const;
    void set_log_level(int level);

private:
    class Impl;
    std::unique_ptr<Impl> pimpl_;
};

} // namespace sdk
} // namespace tigerwallet

#endif // TIGERWALLET_DEVELOPER_SDK_HPP
