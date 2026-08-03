#pragma once

#include <string>
#include <vector>
#include <optional>
#include <chrono>
#include <nlohmann/json.hpp>

namespace tiger::models {

using json = nlohmann::json;

// Token model
struct Token {
    std::string id;
    std::string token_id;
    std::string name;
    std::string symbol;
    std::string contract_addr;
    int decimals;
    std::optional<std::string> total_supply;
    std::string chain_id;
    std::string chain_name;
    bool is_active;
    bool is_verified;
    bool is_native_token;
    std::optional<std::string> logo_url;
    std::optional<std::string> website;
    std::optional<std::string> whitepaper;
    std::optional<std::string> description;
    std::optional<double> market_cap;
    std::optional<double> price;
    std::optional<double> price_change_24h;
    std::optional<double> volume_24h;
    std::string created_by;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static Token from_json(const json& j);
    
    bool is_active() const;
};

// Trading pair model
struct TradingPair {
    std::string id;
    std::string pair_id;
    std::string base_token_id;
    std::string quote_token_id;
    std::string base_symbol;
    std::string quote_symbol;
    std::string pair_name;
    std::string chain_id;
    std::string chain_name;
    admin::PairStatus status;
    bool trading_enabled;
    double min_trade_amount;
    double max_trade_amount;
    double min_trade_value;
    double maker_fee;
    double taker_fee;
    std::optional<std::string> pool_address;
    double liquidity;
    double current_price;
    double price_change_24h;
    double volume_24h;
    std::string source;
    std::optional<std::string> source_exchange;
    std::optional<std::string> white_label_id;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static TradingPair from_json(const json& j);
    
    bool is_active() const;
    bool is_trading_enabled() const;
};

// Fee structure model
struct FeeStructure {
    std::string id;
    std::string fee_type;
    std::optional<std::string> chain_id;
    std::optional<std::string> token_id;
    double fee_percent;
    double fee_fixed;
    double min_fee;
    double max_fee;
    std::optional<std::string> white_label_id;
    bool is_active;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static FeeStructure from_json(const json& j);
};

// Blockchain model
struct Blockchain {
    std::string id;
    int chain_id;
    std::string name;
    std::string symbol;
    std::string type;
    std::vector<std::string> rpc_urls;
    std::vector<std::string> explorer_urls;
    bool is_active;
    bool is_testnet;
    std::optional<std::string> coin_gecko_id;
    std::optional<std::string> coingecko_symbol;
    int confirmations;
    int block_time;
    std::optional<int> native_token_id;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static Blockchain from_json(const json& j);
    
    bool is_active() const;
};

} // namespace tiger::models
