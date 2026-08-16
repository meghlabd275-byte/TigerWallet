/**
 * TigerAdmin C++ Core - Logger Header
 */
#pragma once

#include <string>
#include <atomic>
#include <mutex>
#include <condition_variable>
#include <queue>
#include <fstream>
#include <thread>

namespace tiger {
namespace admin {

enum class LogLevel { DEBUG = 0, INFO = 1, WARNING = 2, ERROR = 3, CRITICAL = 4 };

struct LogEntry {
    LogLevel level;
    std::string message;
};

class Logger {
public:
    static Logger& instance();

    void init(const std::string& log_file, LogLevel level);
    void log(LogLevel level, const std::string& message);
    void debug(const std::string& message);
    void info(const std::string& message);
    void warning(const std::string& message);
    void error(const std::string& message);
    void critical(const std::string& message);

    void log_async(LogLevel level, const std::string& message);
    void set_level(LogLevel level);
    void write_log(LogLevel level, const std::string& message);
    std::string format_message(LogLevel level, const std::string& message);
    std::string get_timestamp();
    std::string level_to_string(LogLevel level);

private:
    Logger() : level_(LogLevel::INFO), async_mode_(false) {}
    std::atomic<LogLevel> level_;
    std::mutex mutex_;
    std::ofstream file_;
    bool async_mode_;
    std::mutex async_mutex_;
    std::condition_variable async_cv_;
    std::queue<LogEntry> async_queue_;
};

} // namespace admin
} // namespace tiger

#define LOG_INFO(msg)    ::tiger::admin::Logger::instance().info(msg)
#define LOG_ERROR(msg)   ::tiger::admin::Logger::instance().error(msg)
#define LOG_WARNING(msg) ::tiger::admin::Logger::instance().warning(msg)
#define LOG_DEBUG(msg)   ::tiger::admin::Logger::instance().debug(msg)
