/**
 * TigerAdmin C++ Core - Application Header
 */
#pragma once

#include "admin_config.hpp"
#include "admin_server.hpp"
#include "admin_logger.hpp"
#include "admin_handler.hpp"
#include "admin_auth.hpp"
#include "admin_kyc.hpp"
#include "admin_transactions.hpp"
#include "admin_tokens.hpp"
#include "admin_blockchain.hpp"
#include "admin_audit.hpp"
#include "admin_analytics.hpp"
#include "admin_security.hpp"
#include "admin_cache.hpp"
#include "admin_connection_pool.hpp"
#include "admin_rate_limiter.hpp"
#include "admin_websocket.hpp"

#include <string>
#include <atomic>

namespace tiger {
namespace admin {

class AdminApplication {
public:
    static AdminApplication& instance();

    bool initialize(const Config& config);
    bool start();
    void stop();
    void run();

    const Config& get_config() const;
    std::string get_version() const;

    void initialize();

    bool is_running() const { return running_.load(); }

private:
    AdminApplication() = default;
    Config config_;
    std::atomic<bool> running_{false};
    std::atomic<bool> initialized_{false};
};

} // namespace admin
} // namespace tiger
