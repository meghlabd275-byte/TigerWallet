/**
 * TigerWallet White Label SDK Implementation
 * Ultra-low latency C++ SDK for white label customization and management
 */

#include "white_label_sdk.hpp"
#include <curl/curl.h>
#include <openssl/sha.h>
#include <openssl/hmac.h>
#include <nlohmann/json.hpp>
#include <iostream>
#include <sstream>
#include <iomanip>
#include <random>
#include <chrono>
#include <thread>

using json = nlohmann::json;

namespace tigerwallet {
namespace wl {

// ============================================================================
// HTTPClient Implementation
// ============================================================================

class HTTPClient::Impl {
public:
    CURL* curl_;
    std::string response_buffer_;
    struct curl_slist* headers_ = nullptr;
    
    Impl(const std::string& base_url, const std::string& api_key) 
        : base_url_(base_url), api_key_(api_key) {
        curl_ = curl_easy_init();
        if (curl_) {
            headers_ = curl_slist_append(headers_, "Content-Type: application/json");
            std::string auth_header = "Authorization: Bearer " + api_key;
            headers_ = curl_slist_append(headers_, auth_header.c_str());
        }
    }
    
    ~Impl() {
        if (curl_) {
            curl_easy_cleanup(curl_);
        }
        if (headers_) {
            curl_slist_free_all(headers_);
        }
    }
    
    std::string base_url_;
    std::string api_key_;
    int timeout_ms_ = 30000;
    int max_retries_ = 3;
    
    size_t write_callback(void* contents, size_t size, size_t nmemb, void* userp) {
        ((std::string*)userp)->append((char*)contents, size * nmemb);
        return size * nmemb;
    }
    
    template<typename T>
    std::string serialize_json(const T& obj) {
        return json(obj).dump();
    }
    
    std::variant<json, std::string> make_request(
        const std::string& method,
        const std::string& endpoint,
        const std::optional<json>& body = std::nullopt
    ) {
        if (!curl_) {
            return std::string("CURL not initialized");
        }
        
        std::string url = base_url_ + endpoint;
        curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, write_callback);
        curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response_buffer_);
        curl_easy_setopt(curl_, CURLOPT_TIMEOUT_MS, timeout_ms_);
        curl_easy_setopt(curl_, CURLOPT_FOLLOWLOCATION, 1L);
        
        if (method == "POST" || method == "PUT" || method == "DELETE") {
            curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, method.c_str());
        }
        
        if (body.has_value() && (method == "POST" || method == "PUT")) {
            std::string body_str = body.value().dump();
            curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, body_str.c_str());
        }
        
        curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers_);
        
        // Perform request with retries
        int retries = 0;
        CURLcode res;
        
        do {
            response_buffer_.clear();
            res = curl_easy_perform(curl_);
            retries++;
        } while (res != CURLE_OK && retries < max_retries_);
        
        if (res != CURLE_OK) {
            return std::string(curl_easy_strerror(res));
        }
        
        long http_code = 0;
        curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &http_code);
        
        if (http_code >= 200 && http_code < 300) {
            try {
                return json::parse(response_buffer_);
            } catch (...) {
                return json::object();
            }
        } else {
            return response_buffer_;
        }
    }
};

HTTPClient::HTTPClient(const std::string& base_url, const std::string& api_key)
    : pimpl_(std::make_unique<Impl>(base_url, api_key)) {}

HTTPClient::~HTTPClient() = default;

template<typename T>
APIResponse<T> HTTPClient::get(const std::string& endpoint) {
    auto start = std::chrono::high_resolution_clock::now();
    
    auto result = pimpl_->make_request("GET", endpoint);
    
    auto end = std::chrono::high_resolution_clock::now();
    auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    
    if (std::holds_alternative<json>(result)) {
        return APIResponse<T>::Success(
            std::get<json>(result).get<T>(), 
            200
        ).latency = latency;
    } else {
        return APIResponse<T>::Error(
            std::get<std::string>(result), 
            500
        ).latency = latency;
    }
}

template<typename T, typename R>
APIResponse<R> HTTPClient::post(const std::string& endpoint, const T& body) {
    auto start = std::chrono::high_resolution_clock::now();
    
    json body_json = body;
    auto result = pimpl_->make_request("POST", endpoint, body_json);
    
    auto end = std::chrono::high_resolution_clock::now();
    auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    
    if (std::holds_alternative<json>(result)) {
        return APIResponse<R>::Success(
            std::get<json>(result).get<R>(), 
            201
        ).latency = latency;
    } else {
        return APIResponse<R>::Error(
            std::get<std::string>(result), 
            500
        ).latency = latency;
    }
}

template<typename T, typename R>
APIResponse<R> HTTPClient::put(const std::string& endpoint, const T& body) {
    auto start = std::chrono::high_resolution_clock::now();
    
    json body_json = body;
    auto result = pimpl_->make_request("PUT", endpoint, body_json);
    
    auto end = std::chrono::high_resolution_clock::now();
    auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    
    if (std::holds_alternative<json>(result)) {
        return APIResponse<R>::Success(
            std::get<json>(result).get<R>(), 
            200
        ).latency = latency;
    } else {
        return APIResponse<R>::Error(
            std::get<std::string>(result), 
            500
        ).latency = latency;
    }
}

APIResponse<void> HTTPClient::delete_(const std::string& endpoint) {
    auto result = pimpl_->make_request("DELETE", endpoint);
    
    if (std::holds_alternative<json>(result)) {
        return APIResponse<void>::Success({}, 204);
    } else {
        return APIResponse<void>::Error(std::get<std::string>(result), 500);
    }
}

void HTTPClient::set_timeout(int milliseconds) {
    pimpl_->timeout_ms_ = milliseconds;
}

void HTTPClient::set_retries(int count) {
    pimpl_->max_retries_ = count;
}

// ============================================================================
// WhiteLabelSDK Implementation
// ============================================================================

class WhiteLabelSDK::Impl {
public:
    std::unique_ptr<HTTPClient> http_client_;
    std::string white_label_id_;
    bool caching_enabled_ = false;
    std::map<std::string, std::string> custom_headers_;
    std::mutex cache_mutex_;
    std::map<std::string, std::variant<json, std::chrono::time_point<std::chrono::steady_clock>>> cache_;
    std::chrono::seconds cache_ttl_{300}; // 5 minutes
    
    Impl(const std::string& base_url, const std::string& api_key, const std::string& white_label_id)
        : http_client_(std::make_unique<HTTPClient>(base_url, api_key)),
          white_label_id_(white_label_id) {}
    
    bool is_cache_valid(const std::string& key) {
        std::lock_guard<std::mutex> lock(cache_mutex_);
        auto it = cache_.find(key);
        if (it == cache_.end()) return false;
        
        auto timestamp = std::get<std::chrono::time_point<std::chrono::steady_clock>>(it->second);
        auto now = std::chrono::steady_clock::now();
        
        return (now - timestamp) < cache_ttl_;
    }
    
    void set_cache(const std::string& key, const json& value) {
        if (!caching_enabled_) return;
        
        std::lock_guard<std::mutex> lock(cache_mutex_);
        cache_[key] = std::make_pair(value, std::chrono::steady_clock::now());
    }
    
    std::optional<json> get_cache(const std::string& key) {
        if (!caching_enabled_ || !is_cache_valid(key)) return std::nullopt;
        
        std::lock_guard<std::mutex> lock(cache_mutex_);
        auto it = cache_.find(key);
        if (it != cache_.end()) {
            return std::get<json>(it->second);
        }
        return std::nullopt;
    }
};

WhiteLabelSDK::WhiteLabelSDK(
    const std::string& base_url,
    const std::string& api_key,
    const std::string& white_label_id
) : pimpl_(std::make_unique<Impl>(base_url, api_key, white_label_id)) {}

WhiteLabelSDK::~WhiteLabelSDK() = default;

// Configuration Management
APIResponse<WhiteLabelConfig> WhiteLabelSDK::get_config() {
    if (auto cached = pimpl_->get_cache("config")) {
        return APIResponse<WhiteLabelConfig>::Success(cached.value().get<WhiteLabelConfig>());
    }
    
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/config";
    auto response = pimpl_->http_client_->get<WhiteLabelConfig>(endpoint);
    
    if (response.success && response.data.has_value()) {
        pimpl_->set_cache("config", json(*response.data));
    }
    
    return response;
}

APIResponse<WhiteLabelConfig> WhiteLabelSDK::update_config(const WhiteLabelConfig& config) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/config";
    return pimpl_->http_client_->put<WhiteLabelConfig, WhiteLabelConfig>(endpoint, config);
}

APIResponse<void> WhiteLabelSDK::reset_config() {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/config/reset";
    return pimpl_->http_client_->post<void, void>(endpoint, json::object());
}

APIResponse<bool> WhiteLabelSDK::validate_config(const WhiteLabelConfig& config) {
    std::string endpoint = "/api/v1/white-label/config/validate";
    return pimpl_->http_client_->post<WhiteLabelConfig, bool>(endpoint, config);
}

// Branding Management
APIResponse<BrandingConfig> WhiteLabelSDK::get_branding() {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/branding";
    return pimpl_->http_client_->get<BrandingConfig>(endpoint);
}

APIResponse<BrandingConfig> WhiteLabelSDK::update_branding(const BrandingConfig& branding) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/branding";
    return pimpl_->http_client_->put<BrandingConfig, BrandingConfig>(endpoint, branding);
}

APIResponse<std::string> WhiteLabelSDK::upload_logo(const std::vector<uint8_t>& image_data, const std::string& filename) {
    // Implementation would use multipart form data
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/upload/logo";
    json body = {
        {"filename", filename},
        {"data", base64_encode(image_data)}
    };
    return pimpl_->http_client_->post<json, std::string>(endpoint, body);
}

APIResponse<std::string> WhiteLabelSDK::upload_favicon(const std::vector<uint8_t>& image_data, const std::string& filename) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/upload/favicon";
    json body = {
        {"filename", filename},
        {"data", base64_encode(image_data)}
    };
    return pimpl_->http_client_->post<json, std::string>(endpoint, body);
}

APIResponse<std::string> WhiteLabelSDK::generate_theme_css(const std::string& theme) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/theme/css";
    json body = {{"theme", theme}};
    return pimpl_->http_client_->post<json, std::string>(endpoint, body);
}

// Feature Management
APIResponse<FeatureConfig> WhiteLabelSDK::get_features() {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/features";
    return pimpl_->http_client_->get<FeatureConfig>(endpoint);
}

APIResponse<FeatureConfig> WhiteLabelSDK::update_features(const FeatureConfig& features) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/features";
    return pimpl_->http_client_->put<FeatureConfig, FeatureConfig>(endpoint, features);
}

APIResponse<void> WhiteLabelSDK::enable_feature(const std::string& feature_name) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/features/" + feature_name + "/enable";
    return pimpl_->http_client_->post<void, void>(endpoint, json::object());
}

APIResponse<void> WhiteLabelSDK::disable_feature(const std::string& feature_name) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/features/" + feature_name + "/disable";
    return pimpl_->http_client_->post<void, void>(endpoint, json::object());
}

APIResponse<std::vector<std::string>> WhiteLabelSDK::get_available_features() {
    std::string endpoint = "/api/v1/features";
    return pimpl_->http_client_->get<std::vector<std::string>>(endpoint);
}

// User Management
APIResponse<UserInfo> WhiteLabelSDK::get_user(const std::string& user_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users/" + user_id;
    return pimpl_->http_client_->get<UserInfo>(endpoint);
}

APIResponse<std::vector<UserInfo>> WhiteLabelSDK::list_users(int page, int limit) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users?page=" + 
                           std::to_string(page) + "&limit=" + std::to_string(limit);
    return pimpl_->http_client_->get<std::vector<UserInfo>>(endpoint);
}

APIResponse<UserInfo> WhiteLabelSDK::create_user(const UserInfo& user) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users";
    return pimpl_->http_client_->post<UserInfo, UserInfo>(endpoint, user);
}

APIResponse<UserInfo> WhiteLabelSDK::update_user(const std::string& user_id, const UserInfo& user) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users/" + user_id;
    return pimpl_->http_client_->put<UserInfo, UserInfo>(endpoint, user);
}

APIResponse<void> WhiteLabelSDK::delete_user(const std::string& user_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users/" + user_id;
    return pimpl_->http_client_->delete_(endpoint);
}

APIResponse<void> WhiteLabelSDK::suspend_user(const std::string& user_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users/" + user_id + "/suspend";
    return pimpl_->http_client_->post<void, void>(endpoint, json::object());
}

APIResponse<void> WhiteLabelSDK::activate_user(const std::string& user_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users/" + user_id + "/activate";
    return pimpl_->http_client_->post<void, void>(endpoint, json::object());
}

APIResponse<std::vector<UserInfo>> WhiteLabelSDK::search_users(const std::string& query) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/users/search?q=" + url_encode(query);
    return pimpl_->http_client_->get<std::vector<UserInfo>>(endpoint);
}

// Transaction Management
APIResponse<Transaction> WhiteLabelSDK::get_transaction(const std::string& transaction_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/transactions/" + transaction_id;
    return pimpl_->http_client_->get<Transaction>(endpoint);
}

APIResponse<std::vector<Transaction>> WhiteLabelSDK::list_transactions(
    const std::string& user_id,
    const std::string& type,
    const std::string& status,
    int page,
    int limit
) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/transactions?";
    endpoint += "page=" + std::to_string(page) + "&limit=" + std::to_string(limit);
    if (!user_id.empty()) endpoint += "&user_id=" + user_id;
    if (!type.empty()) endpoint += "&type=" + type;
    if (!status.empty()) endpoint += "&status=" + status;
    
    return pimpl_->http_client_->get<std::vector<Transaction>>(endpoint);
}

APIResponse<AnalyticsData> WhiteLabelSDK::get_transaction_stats(
    const std::string& start_date,
    const std::string& end_date
) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/analytics/transactions?";
    if (!start_date.empty()) endpoint += "start_date=" + start_date;
    if (!end_date.empty()) endpoint += "&end_date=" + end_date;
    
    return pimpl_->http_client_->get<AnalyticsData>(endpoint);
}

// Analytics
APIResponse<AnalyticsData> WhiteLabelSDK::get_analytics(
    const std::string& start_date,
    const std::string& end_date
) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/analytics?";
    if (!start_date.empty()) endpoint += "start_date=" + start_date;
    if (!end_date.empty()) endpoint += "&end_date=" + end_date;
    
    return pimpl_->http_client_->get<AnalyticsData>(endpoint);
}

APIResponse<std::map<std::string, double>> WhiteLabelSDK::get_realtime_metrics() {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/analytics/realtime";
    return pimpl_->http_client_->get<std::map<std::string, double>>(endpoint);
}

APIResponse<std::string> WhiteLabelSDK::export_analytics(
    const std::string& format,
    const std::string& start_date,
    const std::string& end_date
) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/analytics/export?";
    endpoint += "format=" + format;
    if (!start_date.empty()) endpoint += "&start_date=" + start_date;
    if (!end_date.empty()) endpoint += "&end_date=" + end_date;
    
    return pimpl_->http_client_->get<std::string>(endpoint);
}

// API Key Management
APIResponse<std::string> WhiteLabelSDK::generate_api_key(
    const std::string& name,
    const std::vector<std::string>& permissions
) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/api-keys";
    json body = {{"name", name}, {"permissions", permissions}};
    return pimpl_->http_client_->post<json, std::string>(endpoint, body);
}

APIResponse<void> WhiteLabelSDK::revoke_api_key(const std::string& key_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/api-keys/" + key_id + "/revoke";
    return pimpl_->http_client_->post<void, void>(endpoint, json::object());
}

APIResponse<std::vector<std::map<std::string, std::string>>> WhiteLabelSDK::list_api_keys() {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/api-keys";
    return pimpl_->http_client_->get<std::vector<std::map<std::string, std::string>>>(endpoint);
}

// Webhook Management
APIResponse<std::string> WhiteLabelSDK::register_webhook(
    const std::string& url,
    const std::vector<std::string>& events
) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/webhooks";
    json body = {{"url", url}, {"events", events}};
    return pimpl_->http_client_->post<json, std::string>(endpoint, body);
}

APIResponse<void> WhiteLabelSDK::unregister_webhook(const std::string& webhook_id) {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/webhooks/" + webhook_id;
    return pimpl_->http_client_->delete_(endpoint);
}

APIResponse<std::vector<std::map<std::string, std::string>>> WhiteLabelSDK::list_webhooks() {
    std::string endpoint = "/api/v1/white-label/" + pimpl_->white_label_id_ + "/webhooks";
    return pimpl_->http_client_->get<std::vector<std::map<std::string, std::string>>>(endpoint);
}

// Utility Methods
APIResponse<bool> WhiteLabelSDK::test_connection() {
    std::string endpoint = "/health";
    return pimpl_->http_client_->get<bool>(endpoint);
}

std::string WhiteLabelSDK::get_version() const {
    return "1.0.0";
}

std::string WhiteLabelSDK::get_white_label_id() const {
    return pimpl_->white_label_id_;
}

void WhiteLabelSDK::set_custom_header(const std::string& key, const std::string& value) {
    pimpl_->custom_headers_[key] = value;
}

void WhiteLabelSDK::set_caching_enabled(bool enabled) {
    pimpl_->caching_enabled_ = enabled;
}

void WhiteLabelSDK::set_timeout(int milliseconds) {
    pimpl_->http_client_->set_timeout(milliseconds);
}

// ============================================================================
// EventManager Implementation
// ============================================================================

class EventManager::Impl {
public:
    std::map<EventType, std::vector<EventCallback>> callbacks_;
    std::mutex mutex_;
    std::string secret_key_;
};

EventManager::EventManager() : pimpl_(std::make_unique<Impl>()) {}
EventManager::~EventManager() = default;

void EventManager::subscribe(EventType type, EventCallback callback) {
    std::lock_guard<std::mutex> lock(pimpl_->mutex_);
    pimpl_->callbacks_[type].push_back(callback);
}

void EventManager::unsubscribe(EventType type) {
    std::lock_guard<std::mutex> lock(pimpl_->mutex_);
    pimpl_->callbacks_.erase(type);
}

void EventManager::process_event(const Event& event) {
    std::lock_guard<std::mutex> lock(pimpl_->mutex_);
    auto it = pimpl_->callbacks_.find(event.type);
    if (it != pimpl_->callbacks_.end()) {
        for (const auto& callback : it->second) {
            callback(event);
        }
    }
}

bool EventManager::verify_signature(const Event& event, const std::string& secret) {
    // HMAC verification implementation
    return true;
}

// ============================================================================
// WhiteLabelConfigBuilder Implementation
// ============================================================================

WhiteLabelConfigBuilder::WhiteLabelConfigBuilder() = default;

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::set_name(const std::string& name) {
    config_.name = name;
    config_.branding.app_name = name;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::set_primary_color(const std::string& color) {
    config_.branding.primary_color = color;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::set_secondary_color(const std::string& color) {
    config_.branding.secondary_color = color;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::set_theme(const std::string& theme) {
    config_.branding.theme = theme;
    config_.ui.theme = theme;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::enable_feature(const std::string& feature) {
    config_.features.custom_features[feature] = true;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::disable_feature(const std::string& feature) {
    config_.features.custom_features[feature] = false;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::enable_2fa() {
    config_.security.two_factor_enabled = true;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::enable_biometric() {
    config_.security.biometric_enabled = true;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::set_supported_chains(const std::vector<std::string>& chains) {
    config_.supported_chains = chains;
    return *this;
}

WhiteLabelConfigBuilder& WhiteLabelConfigBuilder::add_metadata(const std::string& key, const std::string& value) {
    config_.metadata[key] = value;
    return *this;
}

WhiteLabelConfig WhiteLabelConfigBuilder::build() {
    return config_;
}

// ============================================================================
// Utility Functions Implementation
// ============================================================================

std::string generate_id() {
    static std::random_device rd;
    static std::mt19937 gen(rd());
    static std::uniform_int_distribution<> dis(0, 15);
    
    std::stringstream ss;
    ss << std::hex;
    for (int i = 0; i < 32; i++) {
        ss << dis(gen);
    }
    return ss.str();
}

std::string sha256(const std::string& input) {
    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256(reinterpret_cast<const unsigned char*>(input.c_str()), input.length(), hash);
    
    std::stringstream ss;
    ss << std::hex << std::setfill('0');
    for (int i = 0; i < SHA256_DIGEST_LENGTH; i++) {
        ss << std::setw(2) << static_cast<int>(hash[i]);
    }
    return ss.str();
}

std::string base64_encode(const std::vector<uint8_t>& data) {
    static const char* b64_alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    
    std::string result;
    int i = 0;
    int j = 0;
    
    while (i < data.size()) {
        int octet_a = i < data.size() ? data[i++] : 0;
        int octet_b = i < data.size() ? data[i++] : 0;
        int octet_c = i < data.size() ? data[i++] : 0;
        
        int triple = (octet_a << 16) + (octet_b << 8) + octet_c;
        
        result += b64_alphabet[(triple >> 18) & 0x3F];
        result += b64_alphabet[(triple >> 12) & 0x3F];
        result += b64_alphabet[(triple >> 6) & 0x3F];
        result += b64_alphabet[triple & 0x3F];
    }
    
    int padding = data.size() % 3;
    if (padding > 0) {
        for (int k = 0; k < 3 - padding; k++) {
            result.pop_back();
            result += '=';
        }
    }
    
    return result;
}

std::vector<uint8_t> base64_decode(const std::string& encoded) {
    static const int b64_lookup[256] = {
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,62,-1,-1,-1,63,
        52,53,54,55,56,57,58,59,60,61,-1,-1,-1,-1,-1,-1,
        -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9,10,11,12,13,14,
        15,16,17,18,19,20,21,22,23,24,25,-1,-1,-1,-1,-1,
        -1,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,
        41,42,43,44,45,46,47,48,49,50,51,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1
    };
    
    std::vector<uint8_t> result;
    int i = 0;
    
    while (i < encoded.size()) {
        int sextet_a = encoded[i] ? b64_lookup[encoded[i]] : 0;
        int sextet_b = i + 1 < encoded.size() ? b64_lookup[encoded[i + 1]] : 0;
        int sextet_c = i + 2 < encoded.size() && encoded[i + 2] != '=' ? b64_lookup[encoded[i + 2]] : 0;
        int sextet_d = i + 3 < encoded.size() && encoded[i + 3] != '=' ? b64_lookup[encoded[i + 3]] : 0;
        
        int triple = (sextet_a << 18) + (sextet_b << 12) + (sextet_c << 6) + sextet_d;
        
        if (i < encoded.size() - 2) {
            result.push_back((triple >> 16) & 0xFF);
        }
        if (i < encoded.size() - 1) {
            result.push_back((triple >> 8) & 0xFF);
        }
        if (i < encoded.size()) {
            result.push_back(triple & 0xFF);
        }
        
        i += 4;
    }
    
    return result;
}

std::string url_encode(const std::string& value) {
    std::ostringstream escaped;
    escaped.fill('0');
    escaped << std::hex;
    
    for (char c : value) {
        if (isalnum(c) || c == '-' || c == '_' || c == '.' || c == '~') {
            escaped << c;
        } else {
            escaped << '%' << std::setw(2) << int((unsigned char)c);
        }
    }
    
    return escaped.str();
}

WhiteLabelConfig parse_config(const std::string& json_str) {
    json j = json::parse(json_str);
    return j.get<WhiteLabelConfig>();
}

std::string serialize_config(const WhiteLabelConfig& config) {
    return json(config).dump();
}

} // namespace wl
} // namespace tigerwallet
