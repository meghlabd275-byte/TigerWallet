/**
 * TigerWallet Admin Platform - C++ User Handler Implementation
 * High-performance user management handler
 */

#include "../include/handlers/user_handler.h"
#include "../include/database.h"
#include "../include/redis_client.h"
#include <iostream>
#include <sstream>
#include <regex>

namespace tiger {
namespace handlers {

UserHandler::UserHandler(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis)
    : db_(db), redis_(redis) {}

nlohmann::json UserHandler::get_users(const nlohmann::json& request) {
    int page = request.value("page", 1);
    int limit = request.value("limit", 20);
    std::string status = request.value("status", "");
    std::string search = request.value("search", "");
    
    std::stringstream query;
    query << "SELECT id, user_id, username, email, phone, status, tier, kyc_status, "
          << "kyc_level, is_email_verified, is_phone_verified, white_label_id, "
          << "referral_code, created_at, last_login FROM users WHERE 1=1";
    
    if (!status.empty()) {
        query << " AND status = '" << db_->escape_string(status) << "'";
    }
    
    if (!search.empty()) {
        query << " AND (username ILIKE '%" << db_->escape_string(search) << "%' "
              << "OR email ILIKE '%" << db_->escape_string(search) << "%')";
    }
    
    query << " ORDER BY created_at DESC";
    query << " LIMIT " << limit << " OFFSET " << ((page - 1) * limit);
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to fetch users"},
            {"success", false}
        });
    }
    
    nlohmann::json users = nlohmann::json::array();
    
    for (int i = 0; i < result->rows(); ++i) {
        users.push_back(nlohmann::json::object({
            {"id", result->get_value(i, "id")},
            {"user_id", result->get_value(i, "user_id")},
            {"username", result->get_value(i, "username")},
            {"email", result->get_value(i, "email")},
            {"phone", result->get_value(i, "phone")},
            {"status", result->get_value(i, "status")},
            {"tier", std::stoi(result->get_value(i, "tier"))},
            {"kyc_status", result->get_value(i, "kyc_status")},
            {"kyc_level", std::stoi(result->get_value(i, "kyc_level"))},
            {"is_email_verified", result->get_value(i, "is_email_verified") == "true"},
            {"is_phone_verified", result->get_value(i, "is_phone_verified") == "true"},
            {"white_label_id", result->get_value(i, "white_label_id")},
            {"referral_code", result->get_value(i, "referral_code")},
            {"created_at", result->get_value(i, "created_at")},
            {"last_login", result->get_value(i, "last_login")}
        }));
    }
    
    // Get total count
    std::stringstream count_query;
    count_query << "SELECT COUNT(*) as total FROM users WHERE 1=1";
    if (!status.empty()) {
        count_query << " AND status = '" << db_->escape_string(status) << "'";
    }
    auto count_result = db_->execute(count_query.str());
    int total = count_result && count_result->rows() > 0 ? 
        std::stoi(count_result->get_value(0, "total")) : 0;
    
    return nlohmann::json::object({
        {"success", true},
        {"data", users},
        {"meta", nlohmann::json::object({
            {"page", page},
            {"limit", limit},
            {"total", total},
            {"total_pages", (total + limit - 1) / limit}
        })}
    });
}

nlohmann::json UserHandler::get_user(const std::string& user_id) {
    std::stringstream query;
    query << "SELECT id, user_id, username, email, phone, status, tier, kyc_status, "
          << "kyc_level, is_email_verified, is_phone_verified, white_label_id, "
          << "referral_code, created_at, last_login FROM users "
          << "WHERE id = '" << db_->escape_string(user_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result || result->rows() == 0) {
        return nlohmann::json::object({
            {"error", "User not found"},
            {"success", false}
        });
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"data", nlohmann::json::object({
            {"id", result->get_value(0, "id")},
            {"user_id", result->get_value(0, "user_id")},
            {"username", result->get_value(0, "username")},
            {"email", result->get_value(0, "email")},
            {"phone", result->get_value(0, "phone")},
            {"status", result->get_value(0, "status")},
            {"tier", std::stoi(result->get_value(0, "tier"))},
            {"kyc_status", result->get_value(0, "kyc_status")},
            {"kyc_level", std::stoi(result->get_value(0, "kyc_level"))},
            {"is_email_verified", result->get_value(0, "is_email_verified") == "true"},
            {"is_phone_verified", result->get_value(0, "is_phone_verified") == "true"},
            {"white_label_id", result->get_value(0, "white_label_id")},
            {"referral_code", result->get_value(0, "referral_code")},
            {"created_at", result->get_value(0, "created_at")},
            {"last_login", result->get_value(0, "last_login")}
        })}
    });
}

nlohmann::json UserHandler::update_user(const std::string& user_id, const nlohmann::json& data) {
    std::vector<std::string> updates;
    
    if (data.contains("username")) {
        updates.push_back("username = '" + db_->escape_string(data["username"].get<std::string>()) + "'");
    }
    if (data.contains("email")) {
        updates.push_back("email = '" + db_->escape_string(data["email"].get<std::string>()) + "'");
    }
    if (data.contains("phone")) {
        updates.push_back("phone = '" + db_->escape_string(data["phone"].get<std::string>()) + "'");
    }
    if (data.contains("status")) {
        updates.push_back("status = '" + db_->escape_string(data["status"].get<std::string>()) + "'");
    }
    if (data.contains("kyc_status")) {
        updates.push_back("kyc_status = '" + db_->escape_string(data["kyc_status"].get<std::string>()) + "'");
    }
    
    if (updates.empty()) {
        return nlohmann::json::object({
            {"error", "No fields to update"},
            {"success", false}
        });
    }
    
    std::stringstream query;
    query << "UPDATE users SET " << updates[0];
    for (size_t i = 1; i < updates.size(); ++i) {
        query << ", " << updates[i];
    }
    query << ", updated_at = NOW() WHERE id = '" << db_->escape_string(user_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to update user"},
            {"success", false}
        });
    }
    
    // Invalidate cache
    if (redis_) {
        redis_->del("user:" + user_id);
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "User updated successfully"}
    });
}

nlohmann::json UserHandler::suspend_user(const std::string& user_id, const std::string& reason) {
    std::stringstream query;
    query << "UPDATE users SET status = 'suspended', "
          << "suspend_reason = '" << db_->escape_string(reason) << "', "
          << "suspended_at = NOW() WHERE id = '" << db_->escape_string(user_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to suspend user"},
            {"success", false}
        });
    }
    
    // Invalidate all user sessions
    if (redis_) {
        redis_->del("session:user:" + user_id);
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "User suspended successfully"}
    });
}

nlohmann::json UserHandler::ban_user(const std::string& user_id, const std::string& reason) {
    std::stringstream query;
    query << "UPDATE users SET status = 'banned', "
          << "ban_reason = '" << db_->escape_string(reason) << "', "
          << "banned_at = NOW() WHERE id = '" << db_->escape_string(user_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to ban user"},
            {"success", false}
        });
    }
    
    // Invalidate all user sessions and tokens
    if (redis_) {
        redis_->del("session:user:" + user_id);
        redis_->del("token:user:" + user_id);
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "User banned successfully"}
    });
}

nlohmann::json UserHandler::activate_user(const std::string& user_id) {
    std::stringstream query;
    query << "UPDATE users SET status = 'active', "
          << "suspend_reason = NULL, suspended_at = NULL, "
          << "ban_reason = NULL, banned_at = NULL "
          << "WHERE id = '" << db_->escape_string(user_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to activate user"},
            {"success", false}
        });
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "User activated successfully"}
    });
}

} // namespace handlers
} // namespace tiger
