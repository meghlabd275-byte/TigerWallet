#pragma once

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <functional>
#include <chrono>
#include <mutex>
#include <shared_mutex>

namespace tiger {

struct Session {
    std::string id;
    std::string admin_id;
    std::string token;
    std::string ip_address;
    std::string user_agent;
    std::chrono::steady_clock::time_point created_at;
    std::chrono::steady_clock::time_point last_activity;
    std::chrono::seconds expires_in;
    bool is_active;
    std::map<std::string, std::string> metadata;
};

class SessionManager {
public:
    SessionManager(std::shared_ptr<class RedisClient> redis, int default_ttl = 3600);
    ~SessionManager();
    
    // Session creation
    std::optional<Session> create_session(const std::string& admin_id, 
                                         const std::string& ip_address = "",
                                         const std::string& user_agent = "");
    
    // Session retrieval
    std::optional<Session> get_session(const std::string& session_id);
    std::optional<Session> get_session_by_token(const std::string& token);
    
    // Session update
    bool update_session(const std::string& session_id, const std::map<std::string, std::string>& metadata);
    bool extend_session(const std::string& session_id, int ttl_seconds);
    bool refresh_activity(const std::string& session_id);
    
    // Session deletion
    bool destroy_session(const std::string& session_id);
    bool destroy_user_sessions(const std::string& admin_id);
    bool destroy_all_sessions();
    
    // Session validation
    bool is_valid_session(const std::string& session_id);
    bool is_session_expired(const std::string& session_id);
    
    // Session listing
    std::vector<Session> get_user_sessions(const std::string& admin_id);
    std::vector<Session> get_all_sessions();
    int get_active_session_count();
    
    // Cleanup
    void cleanup_expired_sessions();
    void start_cleanup_task(int interval_seconds = 60);
    void stop_cleanup_task();
    
    // Statistics
    struct SessionStats {
        int64_t total_sessions;
        int64_t active_sessions;
        int64_t expired_sessions;
        std::map<std::string, int64_t> sessions_by_ip;
    };
    SessionStats get_stats();
    
private:
    std::shared_ptr<class RedisClient> redis_;
    int default_ttl_;
    std::atomic<bool> cleanup_running_;
    std::thread cleanup_thread_;
    std::mutex mutex_;
    
    std::string generate_session_id();
    std::string generate_token();
    std::string get_session_key(const std::string& session_id);
    std::string get_token_key(const std::string& token);
    std::string get_user_sessions_key(const std::string& admin_id);
    
    Session parse_session_data(const std::string& data);
    std::string serialize_session(const Session& session);
};

// JWT Token Manager
class TokenManager {
public:
    TokenManager(const std::string& secret, int default_ttl = 3600);
    ~TokenManager() = default;
    
    // Token generation
    std::string generate_token(const std::map<std::string, std::string>& claims);
    std::string generate_access_token(const std::string& admin_id, const std::string& role);
    std::string generate_refresh_token(const std::string& admin_id);
    
    // Token validation
    bool validate_token(const std::string& token);
    std::optional<std::map<std::string, std::string>> decode_token(const std::string& token);
    std::string get_token_claim(const std::string& token, const std::string& claim);
    
    // Token refresh
    std::string refresh_token(const std::string& refresh_token);
    
    // Token blacklist
    bool blacklist_token(const std::string& token, int64_t ttl_seconds = 3600);
    bool is_blacklisted(const std::string& token);
    
    // Secret management
    void set_secret(const std::string& secret);
    void rotate_secret(const std::string& new_secret);
    
private:
    std::string secret_;
    int default_ttl_;
    std::shared_mutex secret_mutex_;
    std::string previous_secret_;
    
    std::string base64_encode(const std::string& data);
    std::string base64_decode(const std::string& data);
    std::string hmac_sha256(const std::string& data, const std::string& key);
    std::string sha256(const std::string& data);
};

// API Key Manager
class APIKeyManager {
public:
    APIKeyManager(std::shared_ptr<class Database> db, std::shared_ptr<RedisClient> redis);
    ~APIKeyManager() = default;
    
    struct APIKey {
        std::string id;
        std::string key;
        std::string name;
        std::string admin_id;
        std::vector<std::string> permissions;
        int rate_limit_minute;
        int rate_limit_day;
        std::string tier;
        bool is_active;
        std::optional<std::string> expires_at;
        std::optional<std::string> last_used;
        std::string created_at;
    };
    
    // Key management
    Result<APIKey> create_key(const std::string& admin_id, const Json& data);
    Result<APIKey> get_key(const std::string& id);
    Result<APIKey> get_key_by_value(const std::string& key_value);
    Result<bool> revoke_key(const std::string& id);
    std::vector<APIKey> list_keys(const std::string& admin_id);
    
    // Key validation
    bool validate_key(const std::string& key_value);
    bool has_permission(const std::string& key_value, const std::string& permission);
    bool check_rate_limit(const std::string& key_value, const std::string& endpoint);
    
    // Usage tracking
    void track_usage(const std::string& key_value, const std::string& endpoint);
    Result<Json> get_key_usage(const std::string& key_value, const std::string& period = "24h");
    
private:
    std::shared_ptr<class Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
    
    std::string generate_key();
    std::string hash_key(const std::string& key);
};

// 2FA Manager
class TwoFAManager {
public:
    TwoFAManager(std::shared_ptr<RedisClient> redis);
    ~TwoFAManager() = default;
    
    // 2FA setup
    struct TwoFASetup {
        std::string secret;
        std::string qr_code_url;
        std::vector<std::string> backup_codes;
    };
    
    Result<TwoFASetup> generate_secret(const std::string& admin_id);
    Result<bool> enable_2fa(const std::string& admin_id, const std::string& code);
    Result<bool> disable_2fa(const std::string& admin_id, const std::string& code);
    
    // 2FA validation
    bool validate_code(const std::string& admin_id, const std::string& code);
    bool validate_backup_code(const std::string& admin_id, const std::string& code);
    
    // 2FA recovery
    Result<std::string> generate_recovery_code(const std::string& admin_id);
    bool use_recovery_code(const std::string& admin_id, const std::string& code);
    
    // Backup codes
    std::vector<std::string> get_backup_codes(const std::string& admin_id);
    Result<bool> regenerate_backup_codes(const std::string& admin_id);
    
private:
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
    
    std::string generate_secret();
    std::vector<std::string> generate_backup_codes(int count = 8);
    std::string get_2fa_key(const std::string& admin_id);
};

} // namespace tiger
