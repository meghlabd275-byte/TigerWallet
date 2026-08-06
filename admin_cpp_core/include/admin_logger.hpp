/**
 * TigerAdmin C++ Core - Logger
 * High-performance logging with minimal latency impact
 */

#ifndef TIGER_ADMIN_LOGGER_HPP
#define TIGER_ADMIN_LOGGER_HPP

#include <string>
#include <iostream>
#include <fstream>
#include <sstream>
#include <mutex>
#include <atomic>
#include <chrono>
#include <thread>
#include <queue>
#include <functional>
#include <memory>

namespace tiger {
namespace admin {

enum class LogLevel {
    DEBUG,
    INFO,
    WARNING,
    ERROR,
    CRITICAL
};

class Logger {
public:
    static Logger& instance();
    
    void init(const std::string& log_file, LogLevel level);
    void log(LogLevel level, const std::string& message);
    void set_level(LogLevel level);
    
    void debug(const std::string& message);
    void info(const std::string& message);
    void warning(const std::string& message);
    void error(const std::string& message);
    void critical(const std::string& message);
    
    // Non-blocking logging for ultra-low latency
    void log_async(LogLevel level, const std::string& message);
    
private:
    Logger() = default;
    ~Logger();
    Logger(const Logger&) = delete;
    Logger& operator=(const Logger&) = delete;
    
    void write_log(LogLevel level, const std::string& message);
    std::string format_message(LogLevel level, const std::string& message);
    std::string get_timestamp();
    std::string level_to_string(LogLevel level);
    
    std::ofstream file_;
    std::mutex mutex_;
    std::atomic<LogLevel> level_{LogLevel::INFO};
    bool async_mode_ = false;
    std::thread async_thread_;
    std::queue<std::pair<LogLevel, std::string>> async_queue_;
    std::mutex async_mutex_;
    std::condition_variable async_cv_;
    bool running_ = true;
};

// Convenience macros
#define LOG_DEBUG(msg) tiger::admin::Logger::instance().debug(msg)
#define LOG_INFO(msg) tiger::admin::Logger::instance().info(msg)
#define LOG_WARNING(msg) tiger::admin::Logger::instance().warning(msg)
#define LOG_ERROR(msg) tiger::admin::Logger::instance().error(msg)
#define LOG_CRITICAL(msg) tiger::admin::Logger::instance().critical(msg)

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_LOGGER_HPP
