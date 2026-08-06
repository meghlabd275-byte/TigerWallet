/**
 * TigerAdmin C++ Core - Configuration
 */

#ifndef TIGER_ADMIN_CONFIG_HPP
#define TIGER_ADMIN_CONFIG_HPP

#include <string>
#include <vector>

namespace tiger {
namespace admin {

struct Config {
    // Server
    std::string host = "0.0.0.0";
    int port = 9094;
    int worker_threads = 4;
    
    // Database
    std::string db_host = "localhost";
    int db_port = 5432;
    std::string db_name = "tigerwallet";
    std::string db_user = "admin";
    std::string db_password = "";
    int db_pool_size = 20;
    
    // Redis
    std::string redis_host = "localhost";
    int redis_port = 6379;
    std::string redis_password = "";
    int redis_db = 0;
    
    // Security
    std::string jwt_secret = "";
    std::string encryption_key = "";
    std::string password_pepper = "";
    int max_login_attempts = 5;
    int lockout_duration_seconds = 900;
    bool require_2fa = true;
    
    // Rate Limiting
    int rate_limit_requests = 100;
    int rate_limit_window_seconds = 60;
    
    // Logging
    std::string log_level = "info";
    std::string log_file = "/var/log/tigeradmin.log";
    
    // SSL
    bool use_ssl = false;
    std::string ssl_cert = "";
    std::string ssl_key = "";
    
    // CORS
    std::vector<std::string> cors_origins = {"*"};
    
    // Environment
    std::string env = "development";
};

Config load_config(const std::string& config_file = "");

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_CONFIG_HPP
