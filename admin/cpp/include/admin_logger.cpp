/**
 * TigerAdmin C++ Core - Logger Implementation
 */

#include "admin_logger.hpp"
#include <iostream>
#include <sstream>
#include <iomanip>
#include <chrono>

namespace tiger {
namespace admin {

Logger& Logger::instance() {
    static Logger logger;
    return logger;
}

void Logger::init(const std::string& log_file, LogLevel level) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    level_.store(level);
    
    if (!log_file.empty()) {
        file_.open(log_file, std::ios::app);
        if (!file_.is_open()) {
            std::cerr << "Failed to open log file: " << log_file << std::endl;
        }
    }
}

void Logger::log(LogLevel level, const std::string& message) {
    if (level < level_.load()) {
        return;
    }
    
    if (async_mode_) {
        log_async(level, message);
    } else {
        write_log(level, message);
    }
}

void Logger::debug(const std::string& message) {
    log(LogLevel::DEBUG, message);
}

void Logger::info(const std::string& message) {
    log(LogLevel::INFO, message);
}

void Logger::warning(const std::string& message) {
    log(LogLevel::WARNING, message);
}

void Logger::error(const std::string& message) {
    log(LogLevel::ERROR, message);
}

void Logger::critical(const std::string& message) {
    log(LogLevel::CRITICAL, message);
}

void Logger::log_async(LogLevel level, const std::string& message) {
    {
        std::lock_guard<std::mutex> lock(async_mutex_);
        async_queue_.push({level, message});
    }
    async_cv_.notify_one();
}

void Logger::set_level(LogLevel level) {
    level_.store(level);
}

void Logger::write_log(LogLevel level, const std::string& message) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string formatted = format_message(level, message);
    
    // Write to console
    std::cout << formatted << std::endl;
    
    // Write to file
    if (file_.is_open()) {
        file_ << formatted << std::endl;
        file_.flush();
    }
}

std::string Logger::format_message(LogLevel level, const std::string& message) {
    std::ostringstream oss;
    oss << get_timestamp() << " [" << level_to_string(level) << "] " << message;
    return oss.str();
}

std::string Logger::get_timestamp() {
    auto now = std::chrono::system_clock::now();
    auto time_t = std::chrono::system_clock::to_time_t(now);
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
        now.time_since_epoch()) % 1000;
    
    std::ostringstream oss;
    oss << std::put_time(std::localtime(&time_t), "%Y-%m-%d %H:%M:%S");
    oss << "." << std::setfill('0') << std::setw(3) << ms.count();
    return oss.str();
}

std::string Logger::level_to_string(LogLevel level) {
    switch (level) {
        case LogLevel::DEBUG:    return "DEBUG";
        case LogLevel::INFO:     return "INFO";
        case LogLevel::WARNING:  return "WARN";
        case LogLevel::ERROR:    return "ERROR";
        case LogLevel::CRITICAL: return "CRITICAL";
        default:                 return "UNKNOWN";
    }
}

} // namespace admin
} // namespace tiger
