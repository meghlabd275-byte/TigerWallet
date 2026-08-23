/**
 * TigerWallet Developer SDK Implementation
 * Ultra-low latency C++ SDK for developers
 */

#include "developer_sdk.hpp"
#include <iostream>
#include <sstream>
#include <iomanip>
#include <random>
#include <chrono>
#include <thread>
#include <mutex>
#include <atomic>
#include <curl/curl.h>

namespace tigerwallet {
namespace sdk {

// ============================================================================
// Paymaster SDK Implementation
// ============================================================================

class PaymasterSDK::Impl {
public:
    PaymasterConfig config_;
    CURL* curl_;
    std::mutex mtx_;
    
    Impl(const PaymasterConfig& config) : config_(config) {
        curl_ = curl_easy_init();
    }
    
    ~Impl() {
        if (curl_) curl_easy_cleanup(curl_);
    }
    
    SponsorshipResponse request_sponsorship(const SponsorshipRequest& request) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        SponsorshipResponse response;
        
        // Check eligibility (simplified)
        response.approved = true;
        response.paymaster_and_data = config_.paymaster_address + "_" + 
                                    request.user_operation_hash.substr(0, 16);
        response.pre_verification_gas = "21000";
        response.verification_gas_limit = "50000";
        response.call_gas_limit = request.operation.call_gas_limit > 0 ? 
                                  std::to_string(request.operation.call_gas_limit) : "100000";
        
        return response;
    }
    
    bool verify_sponsorship(const std::string& user_op_hash) {
        return true;
    }
    
    std::string get_paymaster_data(const UserOperation& op) {
        return config_.paymaster_address + "_sponsored";
    }
    
    bool is_chain_supported(const std::string& chain_id) {
        for (const auto& chain : config_.supported_chains) {
            if (chain == chain_id) return true;
        }
        return false;
    }
    
    std::vector<std::string> get_supported_chains() {
        return config_.supported_chains;
    }
};

PaymasterSDK::PaymasterSDK(const PaymasterConfig& config)
    : pimpl_(std::make_unique<Impl>(config)) {}

PaymasterSDK::~PaymasterSDK() = default;

SponsorshipResponse PaymasterSDK::request_sponsorship(const SponsorshipRequest& request) {
    return pimpl_->request_sponsorship(request);
}

bool PaymasterSDK::verify_sponsorship(const std::string& user_op_hash) {
    return pimpl_->verify_sponsorship(user_op_hash);
}

std::string PaymasterSDK::get_paymaster_data(const UserOperation& op) {
    return pimpl_->get_paymaster_data(op);
}

void PaymasterSDK::update_config(const PaymasterConfig& config) {
    pimpl_->config_ = config;
}

PaymasterConfig PaymasterSDK::get_config() const {
    return pimpl_->config_;
}

bool PaymasterSDK::is_chain_supported(const std::string& chain_id) {
    return pimpl_->is_chain_supported(chain_id);
}

std::vector<std::string> PaymasterSDK::get_supported_chains() {
    return pimpl_->get_supported_chains();
}

uint64_t PaymasterSDK::estimate_gas(const UserOperation& op) {
    return op.call_gas_limit + op.verification_gas_limit + op.pre_verification_gas;
}

double PaymasterSDK::estimate_sponsorship_cost(const UserOperation& op) {
    uint64_t gas = estimate_gas(op);
    double gas_price = 0.000000025; // 25 gwei
    return gas * gas_price;
}

// ============================================================================
// Session Key SDK Implementation
// ============================================================================

class SessionKeySDK::Impl {
public:
    std::string rpc_url_;
    std::string entry_point_;
    std::map<std::string, SessionKey> session_keys_;
    std::mutex mtx_;
    
    Impl(const std::string& rpc_url, const std::string& entry_point) 
        : rpc_url_(rpc_url), entry_point_(entry_point) {}
    
    SessionKey create_session_key(const SessionKeyRequest& request) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        SessionKey key;
        key.id = "SK_" + std::to_string(time(nullptr)) + "_" + std::to_string(rand() % 10000);
        key.address = "0x" + generate_randomHex(40);
        key.public_key = "0x04" + generate_randomHex(128);
        key.permissions = request.permissions;
        key.signature = request.signature;
        key.created_at = std::chrono::system_clock::now();
        key.expires_at = key.created_at + request.duration;
        key.is_active = true;
        
        session_keys_[key.id] = key;
        
        return key;
    }
    
    SessionKey get_session_key(const std::string& key_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        return session_keys_.at(key_id);
    }
    
    std::vector<SessionKey> list_session_keys(const std::string& user_address) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        std::vector<SessionKey> result;
        for (const auto& pair : session_keys_) {
            if (pair.second.address.substr(0, 42) == user_address || 
                user_address.empty()) {
                result.push_back(pair.second);
            }
        }
        
        return result;
    }
    
    bool revoke_session_key(const std::string& key_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = session_keys_.find(key_id);
        if (it != session_keys_.end()) {
            it->second.is_active = false;
            return true;
        }
        return false;
    }
    
    bool extend_session_key(const std::string& key_id, std::chrono::seconds duration) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = session_keys_.find(key_id);
        if (it != session_keys_.end()) {
            it->second.expires_at = it->second.expires_at + duration;
            return true;
        }
        return false;
    }
    
    bool is_key_valid(const std::string& key_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = session_keys_.find(key_id);
        if (it == session_keys_.end()) return false;
        
        return it->second.is_active && 
               std::chrono::system_clock::now() < it->second.expires_at;
    }
    
    bool has_permission(const std::string& key_id, const std::string& action) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = session_keys_.find(key_id);
        if (it == session_keys_.end()) return false;
        
        const auto& perms = it->second.permissions;
        
        if (action == "swap") return perms.allow_swap;
        if (action == "stake") return perms.allow_stake;
        if (action == "bridge") return perms.allow_bridge;
        if (action == "nft") return perms.allow_nft;
        if (action == "transfer") return perms.allow_transfer;
        
        return false;
    }
    
    bool check_spending_limit(const std::string& key_id, double amount) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = session_keys_.find(key_id);
        if (it == session_keys_.end()) return false;
        
        return amount <= it->second.permissions.max_daily_spend;
    }
    
    std::string sign_user_operation(const std::string& key_id, const UserOperation& op) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = session_keys_.find(key_id);
        if (it == session_keys_.end()) return "";
        
        std::string data = op.sender + std::to_string(op.nonce) + op.call_data;
        return "SIGNATURE_" + data;
    }
    
    std::string sign_message(const std::string& key_id, const std::string& message) {
        return "SIGNATURE_" + message;
    }
    
    static std::string generate_randomHex(size_t length) {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 15);
        
        std::stringstream ss;
        for (size_t i = 0; i < length; i++) {
            ss << std::hex << dis(gen);
        }
        return ss.str();
    }
};

SessionKeySDK::SessionKeySDK(const std::string& rpc_url, const std::string& entry_point)
    : pimpl_(std::make_unique<Impl>(rpc_url, entry_point)) {}

SessionKeySDK::~SessionKeySDK() = default;

SessionKey SessionKeySDK::create_session_key(const SessionKeyRequest& request) {
    return pimpl_->create_session_key(request);
}

SessionKey SessionKeySDK::get_session_key(const std::string& key_id) {
    return pimpl_->get_session_key(key_id);
}

std::vector<SessionKey> SessionKeySDK::list_session_keys(const std::string& user_address) {
    return pimpl_->list_session_keys(user_address);
}

bool SessionKeySDK::revoke_session_key(const std::string& key_id) {
    return pimpl_->revoke_session_key(key_id);
}

bool SessionKeySDK::extend_session_key(const std::string& key_id, std::chrono::seconds duration) {
    return pimpl_->extend_session_key(key_id, duration);
}

bool SessionKeySDK::is_key_valid(const std::string& key_id) {
    return pimpl_->is_key_valid(key_id);
}

bool SessionKeySDK::has_permission(const std::string& key_id, const std::string& action) {
    return pimpl_->has_permission(key_id, action);
}

bool SessionKeySDK::check_spending_limit(const std::string& key_id, double amount) {
    return pimpl_->check_spending_limit(key_id, amount);
}

std::string SessionKeySDK::sign_user_operation(const std::string& key_id, const UserOperation& op) {
    return pimpl_->sign_user_operation(key_id, op);
}

std::string SessionKeySDK::sign_message(const std::string& key_id, const std::string& message) {
    return pimpl_->sign_message(key_id, message);
}

// ============================================================================
// Agentic Wallet SDK Implementation
// ============================================================================

class AgenticWalletSDK::Impl {
public:
    std::string owner_address_;
    AgentWalletConfig config_;
    std::map<std::string, AgentTask> tasks_;
    std::mutex mtx_;
    
    Impl(const std::string& owner_address, const AgentWalletConfig& config)
        : owner_address_(owner_address), config_(config) {}
    
    std::string deploy_agent(const AgentWalletConfig& config) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        config_ = config;
        return "AGENT_" + std::to_string(time(nullptr));
    }
    
    AgentWalletConfig get_agent_config() {
        return config_;
    }
    
    void update_agent_config(const AgentWalletConfig& config) {
        std::lock_guard<std::mutex> lock(mtx_);
        config_ = config;
    }
    
    std::string submit_task(const AgentTask& task) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        AgentTask t = task;
        t.id = "TASK_" + std::to_string(time(nullptr)) + "_" + std::to_string(rand() % 10000);
        t.status = "pending";
        t.created_at = std::chrono::system_clock::now();
        
        tasks_[t.id] = t;
        
        return t.id;
    }
    
    AgentTask get_task(const std::string& task_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        return tasks_.at(task_id);
    }
    
    std::vector<AgentTask> list_tasks(const std::string& status) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        std::vector<AgentTask> result;
        for (const auto& pair : tasks_) {
            if (status.empty() || pair.second.status == status) {
                result.push_back(pair.second);
            }
        }
        
        return result;
    }
    
    bool cancel_task(const std::string& task_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = tasks_.find(task_id);
        if (it != tasks_.end()) {
            it->second.status = "cancelled";
            return true;
        }
        return false;
    }
    
    NLTransactionRequest parse_natural_language(const std::string& text) {
        NLTransactionRequest request;
        request.natural_language = text;
        request.parsed_intent = "swap";
        request.estimated_gas = 150000;
        
        // Simplified parsing
        if (text.find("swap") != std::string::npos || text.find("exchange") != std::string::npos) {
            request.parsed_intent = "swap";
        } else if (text.find("send") != std::string::npos || text.find("transfer") != std::string::npos) {
            request.parsed_intent = "transfer";
        } else if (text.find("stake") != std::string::npos) {
            request.parsed_intent = "stake";
        }
        
        return request;
    }
    
    std::string execute_nl_transaction(const std::string& text) {
        auto request = parse_natural_language(text);
        
        AgentTask task;
        task.id = "NL_" + std::to_string(time(nullptr));
        task.status = "completed";
        task.result = "Executed: " + request.parsed_intent;
        
        return task.id;
    }
    
    std::string ask_ai(const std::string& question) {
        return "AI Response to: " + question;
    }
    
    double get_recommended_gas_price() {
        return 0.000000025; // 25 gwei
    }
    
    std::map<std::string, double> get_token_prices() {
        return {
            {"ETH", 2500.0},
            {"BTC", 45000.0},
            {"USDT", 1.0},
            {"USDC", 1.0}
        };
    }
};

AgenticWalletSDK::AgenticWalletSDK(const std::string& owner_address, const AgentWalletConfig& config)
    : pimpl_(std::make_unique<Impl>(owner_address, config)) {}

AgenticWalletSDK::~AgenticWalletSDK() = default;

std::string AgenticWalletSDK::deploy_agent(const AgentWalletConfig& config) {
    return pimpl_->deploy_agent(config);
}

AgentWalletConfig AgenticWalletSDK::get_agent_config() {
    return pimpl_->get_agent_config();
}

void AgenticWalletSDK::update_agent_config(const AgentWalletConfig& config) {
    pimpl_->update_agent_config(config);
}

std::string AgenticWalletSDK::submit_task(const AgentTask& task) {
    return pimpl_->submit_task(task);
}

AgentTask AgenticWalletSDK::get_task(const std::string& task_id) {
    return pimpl_->get_task(task_id);
}

std::vector<AgentTask> AgenticWalletSDK::list_tasks(const std::string& status) {
    return pimpl_->list_tasks(status);
}

bool AgenticWalletSDK::cancel_task(const std::string& task_id) {
    return pimpl_->cancel_task(task_id);
}

NLTransactionRequest AgenticWalletSDK::parse_natural_language(const std::string& text) {
    return pimpl_->parse_natural_language(text);
}

std::string AgenticWalletSDK::execute_nl_transaction(const std::string& text) {
    return pimpl_->execute_nl_transaction(text);
}

std::string AgenticWalletSDK::ask_ai(const std::string& question) {
    return pimpl_->ask_ai(question);
}

double AgenticWalletSDK::get_recommended_gas_price() {
    return pimpl_->get_recommended_gas_price();
}

std::map<std::string, double> AgenticWalletSDK::get_token_prices() {
    return pimpl_->get_token_prices();
}

// ============================================================================
// Widget SDK Implementation
// ============================================================================

class WidgetSDK::Impl {
public:
    std::string api_key_;
    std::map<std::string, WidgetConfig> widgets_;
    std::map<std::string, std::function<void(const WidgetEvent&)>> callbacks_;
    std::mutex mtx_;
    
    Impl(const std::string& api_key) : api_key_(api_key) {}
    
    std::string create_widget(const WidgetConfig& config) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        WidgetConfig c = config;
        c.widget_id = "WIDGET_" + std::to_string(time(nullptr)) + "_" + std::to_string(rand() % 10000);
        
        widgets_[c.widget_id] = c;
        
        return c.widget_id;
    }
    
    WidgetConfig get_widget_config(const std::string& widget_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        return widgets_.at(widget_id);
    }
    
    void update_widget_config(const std::string& widget_id, const WidgetConfig& config) {
        std::lock_guard<std::mutex> lock(mtx_);
        widgets_[widget_id] = config;
    }
    
    bool delete_widget(const std::string& widget_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        return widgets_.erase(widget_id) > 0;
    }
    
    void subscribe_to_events(const std::string& widget_id, std::function<void(const WidgetEvent&)> callback) {
        std::lock_guard<std::mutex> lock(mtx_);
        callbacks_[widget_id] = callback;
    }
    
    void unsubscribe_from_events(const std::string& widget_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        callbacks_.erase(widget_id);
    }
    
    std::string generate_embed_code(const std::string& widget_id) {
        return "<script src='https://tigerwallet.com/widget/embed.js?id=" + widget_id + "'></script>";
    }
    
    std::string generate_iframe_code(const std::string& widget_id, int width, int height) {
        std::stringstream ss;
        ss << "<iframe src='https://tigerwallet.com/widget/" << widget_id 
           << "' width='" << width << "' height='" << height << "'></iframe>";
        return ss.str();
    }
};

WidgetSDK::WidgetSDK(const std::string& api_key)
    : pimpl_(std::make_unique<Impl>(api_key)) {}

WidgetSDK::~WidgetSDK() = default;

std::string WidgetSDK::create_widget(const WidgetConfig& config) {
    return pimpl_->create_widget(config);
}

WidgetConfig WidgetSDK::get_widget_config(const std::string& widget_id) {
    return pimpl_->get_widget_config(widget_id);
}

void WidgetSDK::update_widget_config(const std::string& widget_id, const WidgetConfig& config) {
    pimpl_->update_widget_config(widget_id, config);
}

bool WidgetSDK::delete_widget(const std::string& widget_id) {
    return pimpl_->delete_widget(widget_id);
}

void WidgetSDK::subscribe_to_events(const std::string& widget_id, EventCallback callback) {
    pimpl_->subscribe_to_events(widget_id, callback);
}

void WidgetSDK::unsubscribe_from_events(const std::string& widget_id) {
    pimpl_->unsubscribe_from_events(widget_id);
}

std::string WidgetSDK::generate_embed_code(const std::string& widget_id) {
    return pimpl_->generate_embed_code(widget_id);
}

std::string WidgetSDK::generate_iframe_code(const std::string& widget_id, int width, int height) {
    return pimpl_->generate_iframe_code(widget_id, width, height);
}

// ============================================================================
// Complete Developer SDK Implementation
// ============================================================================

class DeveloperSDK::Impl {
public:
    std::string api_key_;
    PaymasterConfig paymaster_config_;
    std::unique_ptr<PaymasterSDK> paymaster_sdk_;
    std::unique_ptr<SessionKeySDK> session_key_sdk_;
    std::unique_ptr<AgenticWalletSDK> agentic_sdk_;
    std::unique_ptr<WidgetSDK> widget_sdk_;
    bool initialized_ = false;
    
    Impl(const std::string& api_key, const std::string& rpc_url, const std::string& entry_point) 
        : api_key_(api_key) {
        
        // Initialize Paymaster
        paymaster_config_.api_key = api_key;
        paymaster_config_.paymaster_address = "0x...";
        paymaster_config_.supported_chains = {"1", "137", "56", "43114"};
        paymaster_sdk_ = std::make_unique<PaymasterSDK>(paymaster_config_);
        
        // Initialize Session Keys
        session_key_sdk_ = std::make_unique<SessionKeySDK>(rpc_url, entry_point);
        
        // Initialize Agentic Wallet
        AgentWalletConfig agent_config;
        agent_config.allowed_actions = {"swap", "stake", "bridge"};
        agentic_sdk_ = std::make_unique<AgenticWalletSDK>("0x...", agent_config);
        
        // Initialize Widget SDK
        widget_sdk_ = std::make_unique<WidgetSDK>(api_key);
        
        initialized_ = true;
    }
};

DeveloperSDK::DeveloperSDK(const std::string& api_key, const std::string& rpc_url, const std::string& entry_point)
    : pimpl_(std::make_unique<Impl>(api_key, rpc_url, entry_point)) {}

DeveloperSDK::~DeveloperSDK() = default;

PaymasterSDK& DeveloperSDK::paymaster() {
    return *pimpl_->paymaster_sdk_;
}

SessionKeySDK& DeveloperSDK::session_keys() {
    return *pimpl_->session_key_sdk_;
}

AgenticWalletSDK& DeveloperSDK::agentic() {
    return *pimpl_->agentic_sdk_;
}

WidgetSDK& DeveloperSDK::widgets() {
    return *pimpl_->widget_sdk_;
}

std::string DeveloperSDK::get_sdk_version() const {
    return "1.0.0";
}

bool DeveloperSDK::is_initialized() const {
    return pimpl_->initialized_;
}

void DeveloperSDK::set_log_level(int level) {
    // Set logging level
}

} // namespace sdk
} // namespace tigerwallet
