/**
 * TigerAdmin C++ Core - Token & Pair Handler
 */

#ifndef TIGER_ADMIN_TOKENS_HPP
#define TIGER_ADMIN_TOKENS_HPP

#include <string>
#include <vector>
#include <optional>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Token Service
// ============================================================================

class TokenService {
public:
    static TokenService& instance();
    
    void initialize();
    
    // List tokens
    struct TokenListParams {
        std::optional<TokenStatus> status;
        std::optional<int> chain_id;
        std::string search;
        bool verified_only = false;
        int page = 1;
        int page_size = 20;
    };
    
    struct TokenListResult {
        std::vector<Token> tokens;
        int64_t total;
        int page;
        int page_size;
    };
    
    TokenListResult list_tokens(const TokenListParams& params);
    
    // Get single token
    std::optional<Token> get_token(TokenID id);
    std::optional<Token> get_token_by_symbol(const std::string& symbol,
                                              int chain_id);
    std::optional<Token> get_token_by_address(const std::string& address,
                                                int chain_id);
    
    // Create token
    Token create_token(const std::string& name,
                       const std::string& symbol,
                       const std::string& contract_address,
                       int decimals,
                       int chain_id,
                       const std::string& total_supply,
                       AdminID created_by);
    
    // Update token
    bool update_token(TokenID id, 
                     const std::optional<std::string>& name,
                     const std::optional<std::string>& logo_url,
                     const std::optional<std::string>& website,
                     const std::optional<std::string>& description);
    
    // Status
    bool activate_token(TokenID id);
    bool deactivate_token(TokenID id);
    bool verify_token(TokenID id);
    bool unverify_token(TokenID id);
    bool delete_token(TokenID id);
    
    // Stats
    struct TokenStats {
        int64_t total;
        int64_t active;
        int64_t inactive;
        int64_t verified;
    };
    
    TokenStats get_token_stats();
    
private:
    TokenService() = default;
    
    bool validate_contract(const std::string& address, int chain_id);
};

// ============================================================================
// Trading Pair Service
// ============================================================================

class PairService {
public:
    static PairService& instance();
    
    void initialize();
    
    // List pairs
    struct PairListParams {
        std::optional<PairStatus> status;
        std::optional<int> chain_id;
        std::optional<TokenID> base_token_id;
        std::optional<TokenID> quote_token_id;
        std::string search;
        int page = 1;
        int page_size = 20;
    };
    
    struct PairListResult {
        std::vector<TradingPair> pairs;
        int64_t total;
        int page;
        int page_size;
    };
    
    PairListResult list_pairs(const PairListParams& params);
    
    // Get single pair
    std::optional<TradingPair> get_pair(PairID id);
    std::optional<TradingPair> get_pair_by_name(const std::string& name);
    
    // Create pair
    TradingPair create_pair(TokenID base_token_id,
                            TokenID quote_token_id,
                            int chain_id,
                            AdminID created_by);
    
    // Update pair
    bool update_pair(PairID id, 
                    const std::optional<std::string>& price,
                    const std::optional<std::string>& volume_24h,
                    const std::optional<std::string>& liquidity);
    
    // Status
    bool activate_pair(PairID id);
    bool halt_pair(PairID id);
    bool suspend_pair(PairID id);
    bool delete_pair(PairID id);
    
    // Bulk operations
    int bulk_halt(const std::vector<PairID>& ids);
    int bulk_activate(const std::vector<PairID>& ids);
    
    // Stats
    struct PairStats {
        int64_t total;
        int64_t active;
        int64_t halted;
        int64_t suspended;
    };
    
    PairStats get_pair_stats();
    
private:
    PairService() = default;
};

// ============================================================================
// Fee Service
// ============================================================================

class FeeService {
public:
    static FeeService& instance();
    
    void initialize();
    
    // List fees
    std::vector<FeeStructure> list_fees(int chain_id = -1);
    
    // Get fee
    std::optional<FeeStructure> get_fee(uint64_t id);
    
    // Create fee
    FeeStructure create_fee(const std::string& fee_type,
                           const std::string& asset,
                           const std::string& fee_percent,
                           const std::string& fee_fixed,
                           const std::string& min_fee,
                           const std::string& max_fee,
                           const std::string& tier,
                           int chain_id);
    
    // Update fee
    bool update_fee(uint64_t id,
                   const std::optional<std::string>& fee_percent,
                   const std::optional<std::string>& fee_fixed,
                   const std::optional<std::string>& min_fee,
                   const std::optional<std::string>& max_fee);
    
    // Status
    bool activate_fee(uint64_t id);
    bool deactivate_fee(uint64_t id);
    
private:
    FeeService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_TOKENS_HPP
