/**
 * TigerAdmin C++ Core - Main Header
 */

#ifndef TIGER_ADMIN_CORE_HPP
#define TIGER_ADMIN_CORE_HPP

#include "admin_config.hpp"
#include "admin_logger.hpp"
#include "admin_models.hpp"
#include "admin_connection_pool.hpp"
#include "admin_server.hpp"
#include "admin_auth.hpp"
#include "admin_kyc.hpp"
#include "admin_transactions.hpp"
#include "admin_tokens.hpp"
#include "admin_blockchain.hpp"
#include "admin_analytics.hpp"
#include "admin_websocket.hpp"
#include "admin_cache.hpp"
#include "admin_security.hpp"
#include "admin_audit.hpp"
#include "admin_rate_limiter.hpp"
#include "admin_handler.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Main Application
// ============================================================================

class AdminApplication {
public:
    static AdminApplication& instance();
    
    // Initialize all components
    bool initialize(const Config& config);
    
    // Start the application
    bool start();
    
    // Stop the application
    void stop();
    
    // Run (blocks)
    void run();
    
    // Get config
    const Config& get_config() const;
    
    // Get version
    std::string get_version() const;
    
private:
    AdminApplication() = default;
    ~AdminApplication() = default;
    AdminApplication(const AdminApplication&) = delete;
    AdminApplication& operator=(const AdminApplication&) = delete;
    
    Config config_;
    bool initialized_ = false;
    bool running_ = false;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_CORE_HPP
