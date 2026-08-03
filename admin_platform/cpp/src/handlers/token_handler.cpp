/**
 * TigerWallet Admin Platform - C++ Token Handler Implementation
 * High-performance token management handler
 */

#include "../include/handlers/token_handler.h"
#include "../include/database.h"
#include "../include/redis_client.h"
#include <iostream>
#include <sstream>

namespace tiger {
namespace handlers {

TokenHandler::TokenHandler(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis)
    : db_(db), redis_(redis) {}

nlohmann::json TokenHandler::get_tokens(const nlohmann::json& request) {
    int page = request.value("page", 1);
    int limit = request.value("limit", 20);
    std::string status = request.value("status", "");
    std::string chain = request.value("chain", "");
    std::string search = request.value("search", "");
    
    std::stringstream query;
    query << "SELECT id, token_id, name, symbol, contract_addr, decimals, "
          << "chain_id, chain_name, is_active, is_verified, is_native_token, "
          << "logo_url, website, price, price_change_24h, volume_24h, created_at "
          << "FROM tokens WHERE 1=1";
    
    if (!status.empty()) {
        query << " AND is_active = " << (status == "active" ? "true" : "false");
    }
    
    if (!chain.empty()) {
        query << " AND chain_id = '" << db_->escape_string(chain) << "'";
    }
    
    if (!search.empty()) {
        query << " AND (name ILIKE '%" << db_->escape_string(search) << "%' "
              << "OR symbol ILIKE '%" << db_->escape_string(search) << "%')";
    }
    
    query << " ORDER BY created_at DESC";
    query << " LIMIT " << limit << " OFFSET " << ((page - 1) * limit);
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to fetch tokens"},
            {"success", false}
        });
    }
    
    nlohmann::json tokens = nlohmann::json::array();
    
    for (int i = 0; i < result->rows(); ++i) {
        tokens.push_back(nlohmann::json::object({
            {"id", result->get_value(i, "id")},
            {"token_id", result->get_value(i, "token_id")},
            {"name", result->get_value(i, "name")},
            {"symbol", result->get_value(i, "symbol")},
            {"contract_addr", result->get_value(i, "contract_addr")},
            {"decimals", std::stoi(result->get_value(i, "decimals"))},
            {"chain_id", result->get_value(i, "chain_id")},
            {"chain_name", result->get_value(i, "chain_name")},
            {"is_active", result->get_value(i, "is_active") == "true"},
            {"is_verified", result->get_value(i, "is_verified") == "true"},
            {"is_native_token", result->get_value(i, "is_native_token") == "true"},
            {"logo_url", result->get_value(i, "logo_url")},
            {"website", result->get_value(i, "website")},
            {"price", result->get_value(i, "price").empty() ? nullptr : std::stod(result->get_value(i, "price"))},
            {"price_change_24h", result->get_value(i, "price_change_24h").empty() ? nullptr : std::stod(result->get_value(i, "price_change_24h"))},
            {"volume_24h", result->get_value(i, "volume_24h").empty() ? nullptr : std::stod(result->get_value(i, "volume_24h"))},
            {"created_at", result->get_value(i, "created_at")}
        }));
    }
    
    // Get total count
    std::stringstream count_query;
    count_query << "SELECT COUNT(*) as total FROM tokens WHERE 1=1";
    auto count_result = db_->execute(count_query.str());
    int total = count_result && count_result->rows() > 0 ? 
        std::stoi(count_result->get_value(0, "total")) : 0;
    
    return nlohmann::json::object({
        {"success", true},
        {"data", tokens},
        {"meta", nlohmann::json::object({
            {"page", page},
            {"limit", limit},
            {"total", total},
            {"total_pages", (total + limit - 1) / limit}
        })}
    });
}

nlohmann::json TokenHandler::get_token(const std::string& token_id) {
    std::stringstream query;
    query << "SELECT id, token_id, name, symbol, contract_addr, decimals, "
          << "chain_id, chain_name, is_active, is_verified, is_native_token, "
          << "logo_url, website, price, price_change_24h, volume_24h, created_at "
          << "FROM tokens WHERE id = '" << db_->escape_string(token_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result || result->rows() == 0) {
        return nlohmann::json::object({
            {"error", "Token not found"},
            {"success", false}
        });
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"data", nlohmann::json::object({
            {"id", result->get_value(0, "id")},
            {"token_id", result->get_value(0, "token_id")},
            {"name", result->get_value(0, "name")},
            {"symbol", result->get_value(0, "symbol")},
            {"contract_addr", result->get_value(0, "contract_addr")},
            {"decimals", std::stoi(result->get_value(0, "decimals"))},
            {"chain_id", result->get_value(0, "chain_id")},
            {"chain_name", result->get_value(0, "chain_name")},
            {"is_active", result->get_value(0, "is_active") == "true"},
            {"is_verified", result->get_value(0, "is_verified") == "true"},
            {"is_native_token", result->get_value(0, "is_native_token") == "true"},
            {"logo_url", result->get_value(0, "logo_url")},
            {"website", result->get_value(0, "website")},
            {"created_at", result->get_value(0, "created_at")}
        })}
    });
}

nlohmann::json TokenHandler::create_token(const nlohmann::json& data) {
    std::string token_id = "tk_" + std::to_string(std::time(nullptr)) + "_" + std::to_string(rand() % 10000);
    
    std::stringstream query;
    query << "INSERT INTO tokens (id, token_id, name, symbol, contract_addr, decimals, "
          << "chain_id, chain_name, is_active, is_verified, is_native_token, logo_url, website, created_at) "
          << "VALUES (gen_random_uuid(), '" << db_->escape_string(token_id) << "', "
          << "'" << db_->escape_string(data.value("name", "")) << "', "
          << "'" << db_->escape_string(data.value("symbol", "")) << "', "
          << "'" << db_->escape_string(data.value("contract_addr", "")) << "', "
          << data.value("decimals", 18) << ", "
          << "'" << db_->escape_string(data.value("chain_id", "")) << "', "
          << "'" << db_->escape_string(data.value("chain_name", "")) << "', "
          << "false, false, false, "
          << "'" << db_->escape_string(data.value("logo_url", "")) << "', "
          << "'" << db_->escape_string(data.value("website", "")) << "', "
          << "NOW())";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to create token"},
            {"success", false}
        });
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "Token created successfully"},
        {"data", nlohmann::json::object({
            {"token_id", token_id}
        })}
    });
}

nlohmann::json TokenHandler::update_token(const std::string& token_id, const nlohmann::json& data) {
    std::vector<std::string> updates;
    
    if (data.contains("name")) {
        updates.push_back("name = '" + db_->escape_string(data["name"].get<std::string>()) + "'");
    }
    if (data.contains("logo_url")) {
        updates.push_back("logo_url = '" + db_->escape_string(data["logo_url"].get<std::string>()) + "'");
    }
    if (data.contains("website")) {
        updates.push_back("website = '" + db_->escape_string(data["website"].get<std::string>()) + "'");
    }
    if (data.contains("is_active")) {
        updates.push_back("is_active = " + (data["is_active"].get<bool>() ? "true" : "false"));
    }
    
    if (updates.empty()) {
        return nlohmann::json::object({
            {"error", "No fields to update"},
            {"success", false}
        });
    }
    
    std::stringstream query;
    query << "UPDATE tokens SET " << updates[0];
    for (size_t i = 1; i < updates.size(); ++i) {
        query << ", " << updates[i];
    }
    query << ", updated_at = NOW() WHERE id = '" << db_->escape_string(token_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to update token"},
            {"success", false}
        });
    }
    
    // Invalidate cache
    if (redis_) {
        redis_->del("token:" + token_id);
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "Token updated successfully"}
    });
}

nlohmann::json TokenHandler::delete_token(const std::string& token_id) {
    std::stringstream query;
    query << "DELETE FROM tokens WHERE id = '" << db_->escape_string(token_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to delete token"},
            {"success", false}
        });
    }
    
    // Invalidate cache
    if (redis_) {
        redis_->del("token:" + token_id);
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "Token deleted successfully"}
    });
}

nlohmann::json TokenHandler::verify_token(const std::string& token_id) {
    std::stringstream query;
    query << "UPDATE tokens SET is_verified = true, verified_at = NOW() "
          << "WHERE id = '" << db_->escape_string(token_id) << "'";
    
    auto result = db_->execute(query.str());
    
    if (!result) {
        return nlohmann::json::object({
            {"error", "Failed to verify token"},
            {"success", false}
        });
    }
    
    // Invalidate cache
    if (redis_) {
        redis_->del("token:" + token_id);
    }
    
    return nlohmann::json::object({
        {"success", true},
        {"message", "Token verified successfully"}
    });
}

} // namespace handlers
} // namespace tiger
