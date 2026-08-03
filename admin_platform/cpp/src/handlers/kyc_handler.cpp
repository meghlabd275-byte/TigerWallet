/**
 * TigerWallet Admin Platform - C++ KYC Handler Implementation
 * High-performance KYC verification handler
 */

#include "../include/handlers/kyc_handler.h"
#include "../include/database.h"
#include "../include/redis_client.h"
#include <iostream>
#include <sstream>

namespace tiger {
namespace handlers {

KYCHandler::KYCHandler(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis)
    : db_(db), redis_(redis) {}

nlohmann::json KYCHandler::get_kyc_submissions(const nlohmann::json& request) {
    int page = request.value("page", 1);
    int limit = request.value("limit", 20);
    std::string status = request.value("status", "");
    int level = request.value("level", 0);
    
    std::stringstream query;
    query << "SELECT id, user_id, level, document_type, document_number, "
          << "first_name, last_name, country, address, status, reject_reason, "
          << "reviewed_by, reviewed_at, created_at FROM kyc_submissions WHERE 1=1";
    
    if (!status.empty()) {
        query << " AND status = '" << db_->escape_string(status) << "'";
    }
    
    if (level > 0) {
        query << " AND level = " << level;
    }
    
    query << " ORDER BY created_at DESC";
    query << " LIMIT " << limit << " OFFSET " << ((page - 1) * limit);
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to fetch KYC submissions"},
            {"success", false}
        });
    }
    
    nlohmann::json submissions = nlohmann::json::array();
    
    for (int i = 0; i < result->rows(); ++i) {
        submissions.push_back(nlohmann::json::object({
            {"id", result->get_value(i, "id")},
            {"user_id", result->get_value(i, "user_id")},
            {"level", std::stoi(result->get_value(i, "level"))},
            {"document_type", result->get_value(i, "document_type")},
            {"document_number", result->get_value(i, "document_number")},
            {"first_name", result->get_value(i, "first_name")},
            {"last_name", result->get_value(i, "last_name")},
            {"country", result->get_value(i, "country")},
            {"address", result->get_value(i, "address")},
            {"status", result->get_value(i, "status")},
            {"reject_reason", result->get_value(i, "reject_reason")},
            {"reviewed_by", result->get_value(i, "reviewed_by")},
            {"reviewed_at", result->get_value(i, "reviewed_at")},
            {"created_at", result->get_value(i, "created_at")}
        }));
    }
    
    // Get total count
    std::stringstream count_query;
    count_query << "SELECT COUNT(*) as total FROM kyc_submissions WHERE 1=1";
    if (!status.empty()) {
        count_query << " AND status = '" << db_->escape_string(status) << "'";
    }
    if (level > 0) {
        count_query << " AND level = " << level;
    }
    auto count_result = db_->execute(count_query.str());
    int total = count_result && count_result->rows() > 0 ? 
        std::stoi(count_result->get_value(0, "total")) : 0;
    
    return nlohmann::json::object({
        {"success", true},
        {"data", submissions},
        {"meta", nlohmann::json::object({
            {"page", page},
            {"limit", limit},
            {"total", total},
            {"total_pages", (total + limit - 1) / limit}
        })}
    });
}

nlohmann::json KYCHandler::get_kyc(const std::string& kyc_id) {
    std::stringstream query;
    query << "SELECT id, user_id, level, document_type, document_number, "
          << "first_name, last_name, country, address, status, reject_reason, "
          << "reviewed_by, reviewed_at, created_at, documents "
          << "FROM kyc_submissions WHERE id = '" << db_->escape_string(kyc_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result || result->rows() == 0) {
        return nlohmann::json::object({
            {"error", "KYC submission not found"},
            {"success", false}
        });
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"data", nlohmann::json::object({
            {"id", result->get_value(0, "id")},
            {"user_id", result->get_value(0, "user_id")},
            {"level", std::stoi(result->get_value(0, "level"))},
            {"document_type", result->get_value(0, "document_type")},
            {"document_number", result->get_value(0, "document_number")},
            {"first_name", result->get_value(0, "first_name")},
            {"last_name", result->get_value(0, "last_name")},
            {"country", result->get_value(0, "country")},
            {"address", result->get_value(0, "address")},
            {"status", result->get_value(0, "status")},
            {"reject_reason", result->get_value(0, "reject_reason")},
            {"reviewed_by", result->get_value(0, "reviewed_by")},
            {"reviewed_at", result->get_value(0, "reviewed_at")},
            {"created_at", result->get_value(0, "created_at")},
            {"documents", result->get_value(0, "documents")}
        })}
    });
}

nlohmann::json KYCHandler::approve_kyc(const std::string& kyc_id, const std::string& admin_id, const std::string& notes) {
    db_->begin();
    
    try {
        // Update KYC status
        std::stringstream query;
        query << "UPDATE kyc_submissions SET status = 'approved', "
              << "reviewed_by = '" << db_->escape_string(admin_id) << "', "
              << "reviewed_at = NOW(), "
              << "notes = '" << db_->escape_string(notes) << "' "
              << "WHERE id = '" << db_->escape_string(kyc_id) << "'";
        
        auto result = db_->execute(query.str());
        
        if (!result) {
            db_->rollback();
            return nlohmann::json::object({
                {"error", "Failed to approve KYC"},
                {"success", false}
            });
        }
        
        // Get user_id from KYC submission
        std::stringstream user_query;
        user_query << "SELECT user_id FROM kyc_submissions WHERE id = '" << db_->escape_string(kyc_id) << "'";
        auto user_result = db_->execute(user_query.str());
        
        if (user_result && user_result->rows() > 0) {
            std::string user_id = user_result->get_value(0, "user_id");
            
            // Update user KYC status
            std::stringstream update_user;
            update_user << "UPDATE users SET kyc_status = 'approved', "
                        << "kyc_level = (SELECT level FROM kyc_submissions WHERE id = '" 
                        << db_->escape_string(kyc_id) << "') "
                        << "WHERE id = '" << db_->escape_string(user_id) << "'";
            db_->execute(update_user.str());
            
            // Invalidate cache
            if (redis_) {
                redis_->del("kyc:" + kyc_id);
                redis_->del("user:" + user_id);
            }
        }
        
        db_->commit();
        
        return nlohmann::json::object({
            {"success", true},
            {"message", "KYC approved successfully"}
        });
        
    } catch (const std::exception& e) {
        db_->rollback();
        return nlohmann::json::object({
            {"error", e.what()},
            {"success", false}
        });
    }
}

nlohmann::json KYCHandler::reject_kyc(const std::string& kyc_id, const std::string& admin_id, const std::string& reason) {
    db_->begin();
    
    try {
        // Update KYC status
        std::stringstream query;
        query << "UPDATE kyc_submissions SET status = 'rejected', "
              << "reviewed_by = '" << db_->escape_string(admin_id) << "', "
              << "reviewed_at = NOW(), "
              << "reject_reason = '" << db_->escape_string(reason) << "' "
              << "WHERE id = '" << db_->escape_string(kyc_id) << "'";
        
        auto result = db_->execute(query.str());
        
        if (!result) {
            db_->rollback();
            return nlohmann::json::object({
                {"error", "Failed to reject KYC"},
                {"success", false}
            });
        }
        
        // Get user_id from KYC submission
        std::stringstream user_query;
        user_query << "SELECT user_id FROM kyc_submissions WHERE id = '" << db_->escape_string(kyc_id) << "'";
        auto user_result = db_->execute(user_query.str());
        
        if (user_result && user_result->rows() > 0) {
            std::string user_id = user_result->get_value(0, "user_id");
            
            // Update user KYC status
            std::stringstream update_user;
            update_user << "UPDATE users SET kyc_status = 'rejected' "
                        << "WHERE id = '" << db_->escape_string(user_id) << "'";
            db_->execute(update_user.str());
            
            // Invalidate cache
            if (redis_) {
                redis_->del("kyc:" + kyc_id);
                redis_->del("user:" + user_id);
            }
        }
        
        db_->commit();
        
        return nlohmann::json::object({
            {"success", true},
            {"message", "KYC rejected successfully"}
        });
        
    } catch (const std::exception& e) {
        db_->rollback();
        return nlohmann::json::object({
            {"error", e.what()},
            {"success", false}
        });
    }
}

} // namespace handlers
} // namespace tiger
