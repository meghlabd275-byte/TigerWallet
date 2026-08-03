#include <iostream>
#include <memory>
#include <string>
#include <csignal>
#include <atomic>
#include <thread>
#include <chrono>

#include "server.h"
#include "database.h"
#include "redis_client.h"
#include "session_manager.h"
#include "admin_service.h"
#include "config/config.h"
#include "utils/crypto.h"
#include "middleware/logging.h"
#include "middleware/cors.h"
#include "middleware/rate_limiter.h"
#include "handlers/auth_handler.h"
#include "handlers/admin_handler.h"
#include "handlers/user_handler.h"
#include "handlers/kyc_handler.h"
#include "handlers/token_handler.h"
#include "handlers/pair_handler.h"
#include "handlers/fee_handler.h"
#include "handlers/chain_handler.h"
#include "handlers/wallet_handler.h"
#include "handlers/whitelabel_handler.h"
#include "handlers/analytics_handler.h"

using namespace tiger;

// Global server instance
std::unique_ptr<admin::AdminService> g_admin_service;
std::unique_ptr<admin::UserService> g_user_service;
std::unique_ptr<admin::KYCService> g_kyc_service;
std::unique_ptr<admin::TokenService> g_token_service;
std::unique_ptr<admin::PairService> g_pair_service;
std::unique_ptr<admin::FeeService> g_fee_service;
std::unique_ptr<admin::ChainService> g_chain_service;
std::unique_ptr<admin::WalletService> g_wallet_service;
std::unique_ptr<admin::WhiteLabelService> g_whitelabel_service;
std::unique_ptr<admin::AnalyticsService> g_analytics_service;
std::unique_ptr<admin::WithdrawalService> g_withdrawal_service;
std::unique_ptr<admin::TransactionService> g_transaction_service;

std::atomic<bool> g_running{true};

void signal_handler(int signal) {
    std::cout << "Received signal " << signal << ", shutting down..." << std::endl;
    g_running = false;
}

void register_routes(HTTPServer& server) {
    // Health check
    server.get("/health", [](const HTTPRequest& req) {
        return HTTPResponse().json(R"({"status":"healthy","timestamp":)" + 
            std::to_string(std::chrono::system_clock::to_time_t(std::chrono::system_clock::now())) + "}");
    });
    
    server.get("/ready", [](const HTTPRequest& req) {
        return HTTPResponse().json(R"({"status":"ready"})");
    });
    
    // Auth routes
    auth_handler::register_routes(server);
    
    // Admin routes
    admin_handler::register_routes(server);
    
    // User routes
    user_handler::register_routes(server);
    
    // KYC routes
    kyc_handler::register_routes(server);
    
    // Token routes
    token_handler::register_routes(server);
    
    // Pair routes
    pair_handler::register_routes(server);
    
    // Fee routes
    fee_handler::register_routes(server);
    
    // Chain routes
    chain_handler::register_routes(server);
    
    // Wallet routes
    wallet_handler::register_routes(server);
    
    // White label routes
    whitelabel_handler::register_routes(server);
    
    // Analytics routes
    analytics_handler::register_routes(server);
    
    // API info
    server.get("/api/v1", [](const HTTPRequest& req) {
        return HTTPResponse().json(R"({
            "name": "TigerWallet Admin API",
            "version": "1.0.0",
            "description": "High-performance admin API for TigerWallet platform"
        })");
    });
}

int main(int argc, char* argv[]) {
    // Set up signal handlers
    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);
    
    std::cout << "==================================================" << std::endl;
    std::cout << "  TigerWallet Admin Platform - C++ Backend" << std::endl;
    std::cout << "  Ultra-low latency, high-performance admin API" << std::endl;
    std::cout << "==================================================" << std::endl;
    
    try {
        // Load configuration
        auto config = config::Config::load("/etc/tigeradmin/admin_config.json");
        
        std::cout << "[Config] Loaded configuration successfully" << std::endl;
        std::cout << "[Config] Server: " << config.server.host << ":" << config.server.port << std::endl;
        
        // Initialize database connection pool
        auto db = std::make_shared<Database>(
            config.database.host,
            config.database.port,
            config.database.name,
            config.database.user,
            config.database.password,
            config.database.min_connections,
            config.database.max_connections
        );
        
        if (!db->connect()) {
            std::cerr << "[Error] Failed to connect to database" << std::endl;
            return 1;
        }
        std::cout << "[Database] Connected to PostgreSQL" << std::endl;
        
        // Initialize Redis connection pool
        auto redis = std::make_shared<RedisClient>(
            config.redis.host,
            config.redis.port,
            config.redis.password,
            config.redis.db,
            config.redis.min_connections,
            config.redis.max_connections
        );
        
        if (!redis->connect()) {
            std::cerr << "[Error] Failed to connect to Redis" << std::endl;
            return 1;
        }
        std::cout << "[Redis] Connected to Redis" << std::endl;
        
        // Initialize session manager
        auto session_manager = std::make_shared<SessionManager>(redis, config.security.session_ttl);
        std::cout << "[Session] Session manager initialized" << std::endl;
        
        // Initialize services
        g_admin_service = std::make_unique<admin::AdminService>(db, redis, session_manager);
        g_user_service = std::make_unique<admin::UserService>(db, redis);
        g_kyc_service = std::make_unique<admin::KYCService>(db, redis);
        g_token_service = std::make_unique<admin::TokenService>(db, redis);
        g_pair_service = std::make_unique<admin::PairService>(db, redis);
        g_fee_service = std::make_unique<admin::FeeService>(db, redis);
        g_chain_service = std::make_unique<admin::ChainService>(db, redis);
        g_wallet_service = std::make_unique<admin::WalletService>(db, redis);
        g_whitelabel_service = std::make_unique<admin::WhiteLabelService>(db, redis);
        g_analytics_service = std::make_unique<admin::AnalyticsService>(db, redis);
        g_withdrawal_service = std::make_unique<admin::WithdrawalService>(db, redis);
        g_transaction_service = std::make_unique<admin::TransactionService>(db, redis);
        
        std::cout << "[Services] All admin services initialized" << std::endl;
        
        // Initialize HTTP server
        HTTPServer server(config.server.host, config.server.port);
        server.set_num_threads(config.server.num_threads);
        server.set_request_timeout(config.server.request_timeout);
        server.set_max_body_size(config.server.max_body_size);
        
        // Add global middleware
        server.use(middleware::logging::create_logger());
        server.use(middleware::cors::create_cors_handler(config.cors));
        server.use(middleware::rate_limiter::create_rate_limiter(redis, config.rate_limiting));
        
        // Register routes
        register_routes(server);
        
        // Set error handlers
        server.set_not_found_handler([](const HTTPRequest& req) {
            return HTTPResponse().status(HTTP_404_NOT_FOUND)
                .json(R"({"error":"Not found","path":")" + req.path + "\"");
        });
        
        server.set_internal_error_handler([](const HTTPRequest& req) {
            return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
                .json(R"({"error":"Internal server error"})");
        });
        
        std::cout << "[Server] Routes registered" << std::endl;
        
        // Start server
        if (!server.start()) {
            std::cerr << "[Error] Failed to start server" << std::endl;
            return 1;
        }
        
        std::cout << "[Server] TigerWallet Admin API started on " 
                  << config.server.host << ":" << config.server.port << std::endl;
        std::cout << "[Server] Ready to accept requests" << std::endl;
        
        // Run until interrupted
        while (g_running) {
            std::this_thread::sleep_for(std::chrono::seconds(1));
        }
        
        std::cout << "[Server] Shutting down..." << std::endl;
        server.stop();
        
    } catch (const std::exception& e) {
        std::cerr << "[Fatal] Exception: " << e.what() << std::endl;
        return 1;
    }
    
    std::cout << "[Server] Shutdown complete" << std::endl;
    return 0;
}
