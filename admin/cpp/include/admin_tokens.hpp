/**
 * TigerAdmin C++ Core - Tokens Header
 */
#pragma once

#include "admin_security.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

using TokenID = uint64_t;
using PairID = uint64_t;

enum class TokenStatus {
    ACTIVE = 0,
    INACTIVE = 1,
    DELISTED = 2,
    PENDING = 3
};

enum class PairStatus {
    ACTIVE = 0,
    INACTIVE = 1,
    HALTED = 2,
    SUSPENDED = 3
};

struct Token {
    TokenID id = 0;
    std::string name;
    std::string symbol;
    std::string contract_address;
    int decimals = 0;
    int chain_id = 0;
    std::string total_supply;
    std::string logo_url;
    std::string website;
    std::string description;
    bool is_active = true;
    bool is_verified = false;
    TokenStatus status = TokenStatus::ACTIVE;
    int64_t created_at = 0;
};

struct TradingPair {
    PairID id = 0;
    TokenID base_token_id = 0;
    TokenID quote_token_id = 0;
    int chain_id = 0;
    std::string name;
    std::string price;
    std::string volume_24h;
    std::string liquidity;
    PairStatus status = PairStatus::ACTIVE;
    int64_t created_at = 0;
};

struct FeeStructure {
    uint64_t id = 0;
    std::string fee_type;
    std::string asset;
    std::string fee_percent;
    std::string fee_fixed;
    std::string min_fee;
    std::string max_fee;
    std::string tier;
    int chain_id = 0;
    bool is_active = true;
    int64_t created_at = 0;
};

class TokenService {
public:
    struct TokenListParams {
        int page = 1;
        int page_size = 20;
        std::optional<TokenStatus> status;
        std::optional<int> chain_id;
        std::optional<std::string> search;
        std::optional<bool> verified_only;
    };

    struct TokenListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<Token> tokens;
    };

    struct TokenStats {
        int64_t total = 0;
        int64_t active = 0;
        int64_t inactive = 0;
        int64_t delisted = 0;
    };

    static TokenService& instance();

    void initialize();

    TokenListResult list_tokens(const TokenListParams& params);
    std::optional<Token> get_token(TokenID id);
    std::optional<Token> get_token_by_symbol(const std::string& symbol, int chain_id);
    std::optional<Token> get_token_by_address(const std::string& address, int chain_id);

    Token create_token(const std::string& name, const std::string& symbol,
                       const std::string& contract_address, int decimals,
                       int chain_id, const std::string& total_supply,
                       AdminID created_by);
    bool update_token(TokenID id, const std::optional<std::string>& name,
                      const std::optional<std::string>& logo_url,
                      const std::optional<std::string>& website,
                      const std::optional<std::string>& description);
    bool activate_token(TokenID id);
    bool deactivate_token(TokenID id);
    bool verify_token(TokenID id);
    bool unverify_token(TokenID id);
    bool delete_token(TokenID id);
    TokenStats get_token_stats();
    bool validate_contract(const std::string& address, int chain_id);
};

class PairService {
public:
    struct PairListParams {
        int page = 1;
        int page_size = 20;
        std::optional<PairStatus> status;
        std::optional<int> chain_id;
        std::optional<std::string> search;
    };

    struct PairListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<TradingPair> pairs;
    };

    struct PairStats {
        int64_t total = 0;
        int64_t active = 0;
        int64_t inactive = 0;
        int64_t halted = 0;
    };

    static PairService& instance();

    void initialize();

    PairListResult list_pairs(const PairListParams& params);
    std::optional<TradingPair> get_pair(PairID id);
    std::optional<TradingPair> get_pair_by_name(const std::string& name);

    TradingPair create_pair(TokenID base_token_id, TokenID quote_token_id,
                            int chain_id, AdminID created_by);
    bool update_pair(PairID id, const std::optional<std::string>& price,
                    const std::optional<std::string>& volume_24h,
                    const std::optional<std::string>& liquidity);
    bool activate_pair(PairID id);
    bool halt_pair(PairID id);
    bool suspend_pair(PairID id);
    bool delete_pair(PairID id);
    int bulk_halt(const std::vector<PairID>& ids);
    int bulk_activate(const std::vector<PairID>& ids);
    PairStats get_pair_stats();
};

class FeeService {
public:
    static FeeService& instance();

    void initialize();

    std::vector<FeeStructure> list_fees(int chain_id);
    std::optional<FeeStructure> get_fee(uint64_t id);

    FeeStructure create_fee(const std::string& fee_type, const std::string& asset,
                            const std::string& fee_percent, const std::string& fee_fixed,
                            const std::string& min_fee, const std::string& max_fee,
                            const std::string& tier, int chain_id);
    bool update_fee(uint64_t id, const std::optional<std::string>& fee_percent,
                    const std::optional<std::string>& fee_fixed,
                    const std::optional<std::string>& min_fee,
                    const std::optional<std::string>& max_fee);
    bool activate_fee(uint64_t id);
    bool deactivate_fee(uint64_t id);
};

} // namespace admin
} // namespace tiger
