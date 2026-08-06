/**
 * TigerAdmin C++ Core - Main Entry Point
 * Ultra-low latency admin operations
 */

#include <iostream>
#include <csignal>
#include <thread>
#include <chrono>
#include "admin_core.hpp"
#include "admin_logger.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Signal Handling
// ============================================================================

static std::atomic<bool> g_running{true};

static void signal_handler(int signal) {
    if (signal == SIGINT || signal == SIGTERM) {
        LOG_INFO("Received shutdown signal, stopping...");
        g_running = false;
    }
}

// ============================================================================
// Admin Application
// ============================================================================

AdminApplication& AdminApplication::instance() {
    static AdminApplication app;
    return app;
}

bool AdminApplication::initialize(const Config& config) {
    config_ = config;
    
    // Initialize logger
    Logger::instance().init(config.log_file, LogLevel::INFO);
    LOG_INFO("TigerAdmin C++ Core starting...");
    LOG_INFO("Version: " + get_version());
    
    // Initialize database
    DatabaseManager::instance().initialize(config);
    LOG_INFO("Database initialized");
    
    // Initialize services
    AuthService::instance().initialize();
    UserService::instance().initialize();
    KYCService::instance().initialize();
    TransactionService::instance().initialize();
    WithdrawalService::instance().initialize();
    TokenService::instance().initialize();
    PairService::instance().initialize();
    FeeService::instance().initialize();
    BlockchainService::instance().initialize();
    WhiteLabelService::instance().initialize();
    WebhookService::instance().initialize();
    TicketService::instance().initialize();
    AnalyticsService::instance().initialize();
    ReportService::instance().initialize();
    SLAService::instance().initialize();
    WebSocketService::instance().initialize();
    NotificationService::instance().initialize();
    CacheService::instance().initialize();
    SecurityService::instance().initialize();
    FeatureFlagService::instance().initialize();
    AuditService::instance().initialize();
    BackupService::instance().initialize();
    ArchivalService::instance().initialize();
    RateLimiterService::instance().initialize();
    IPRateLimiter::instance().initialize();
    AdminActionRateLimiter::instance().initialize();
    
    LOG_INFO("All services initialized");
    initialized_ = true;
    return true;
}

bool AdminApplication::start() {
    if (!initialized_) {
        LOG_ERROR("Application not initialized");
        return false;
    }
    
    // Create and start server
    AdminServer* server = new AdminServer(config_);
    AdminHandler::instance().set_server(server);
    AdminHandler::instance().initialize();
    
    if (!server->start()) {
        LOG_ERROR("Failed to start server");
        return false;
    }
    
    running_ = true;
    LOG_INFO("TigerAdmin C++ Core started on port " + 
             std::to_string(config_.port));
    
    return true;
}

void AdminApplication::stop() {
    if (running_) {
        LOG_INFO("Shutting down TigerAdmin C++ Core...");
        
        // Stop services
        WebSocketService::instance().shutdown();
        CacheService::instance().shutdown();
        DatabaseManager::instance().shutdown();
        
        LOG_INFO("Shutdown complete");
        running_ = false;
    }
}

void AdminApplication::run() {
    // Setup signal handlers
    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);
    
    if (!initialize(Config::load())) {
        LOG_ERROR("Failed to initialize application");
        return;
    }
    
    if (!start()) {
        LOG_ERROR("Failed to start application");
        return;
    }
    
    // Wait for shutdown
    while (g_running) {
        std::this_thread::sleep_for(std::chrono::seconds(1));
    }
    
    stop();
}

const Config& AdminApplication::get_config() const {
    return config_;
}

std::string AdminApplication::get_version() const {
    return "1.0.0";
}

} // namespace admin
} // namespace tiger

// ============================================================================
// Main
// ============================================================================

int main(int argc, char* argv[]) {
    using namespace tiger::admin;
    
    try {
        AdminApplication::instance().run();
    } catch (const std::exception& e) {
        std::cerr << "Fatal error: " << e.what() << std::endl;
        return 1;
    }
    
    return 0;
}
