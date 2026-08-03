#pragma once

#include <string>
#include <vector>
#include <optional>
#include <chrono>
#include <nlohmann/json.hpp>

namespace tiger::models {

using json = nlohmann::json;

// Transaction model
struct Transaction {
    std::string id;
    std::string tx_hash;
    std::string user_id;
    std::string wallet_id;
    admin::TransactionType type;
    admin::TransactionStatus status;
    std::string chain_id;
    std::string token_address;
    std::string from_address;
    std::string to_address;
    std::string amount;
    std::string fee;
    std::optional<int> block_number;
    std::optional<int> confirmations;
    std::optional<std::string> metadata;
    std::optional<std::string> error_message;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point completed_at;
    
    json to_json() const;
    static Transaction from_json(const json& j);
    
    bool is_completed() const;
    bool is_pending() const;
    bool is_failed() const;
};

// Withdrawal model
struct Withdrawal {
    std::string id;
    std::string user_id;
    std::string wallet_address;
    std::string chain_id;
    std::string token;
    std::string amount;
    std::string fee;
    std::string total;
    admin::WithdrawalStatus status;
    std::string type;
    std::optional<std::string> tx_hash;
    std::optional<std::string> approved_by;
    std::optional<std::string> rejected_by;
    std::optional<std::string> rejection_reason;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point processed_at;
    
    json to_json() const;
    static Withdrawal from_json(const json& j);
    
    bool is_pending() const;
    bool is_completed() const;
    bool is_rejected() const;
};

// Market maker bot model
struct MarketMakerBot {
    std::string id;
    std::string bot_id;
    std::string name;
    std::optional<std::string> description;
    std::string owner_id;
    std::optional<std::string> white_label_id;
    std::string status;
    std::string strategy_type;
    double base_spread;
    double max_spread;
    double order_size;
    std::vector<std::string> trading_pairs;
    double allocated_capital;
    double used_capital;
    double total_volume_24h;
    double profit_loss_24h;
    double max_slippage;
    int max_open_orders;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static MarketMakerBot from_json(const json& j);
    
    bool is_active() const;
};

} // namespace tiger::models
