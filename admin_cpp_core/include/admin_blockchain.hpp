/**
 * TigerAdmin C++ Core - Blockchain & WhiteLabel Handler
 */

#ifndef TIGER_ADMIN_BLOCKCHAIN_HPP
#define TIGER_ADMIN_BLOCKCHAIN_HPP

#include <string>
#include <vector>
#include <optional>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Blockchain Service
// ============================================================================

class BlockchainService {
public:
    static BlockchainService& instance();
    
    void initialize();
    
    // List blockchains
    std::vector<Blockchain> list_blockchains(bool active_only = false);
    
    // Get blockchain
    std::optional<Blockchain> get_blockchain(BlockchainID id);
    std::optional<Blockchain> get_blockchain_by_chain_id(int chain_id);
    
    // Create blockchain
    Blockchain create_blockchain(const std::string& name,
                                const std::string& symbol,
                                int chain_id,
                                bool is_evm,
                                const std::string& rpc_url,
                                const std::string& explorer_url,
                                const std::string& native_token,
                                int decimals);
    
    // Update blockchain
    bool update_blockchain(BlockchainID id,
                          const std::optional<std::string>& rpc_url,
                          const std::optional<std::string>& explorer_url,
                          const std::optional<std::string>& avg_gas_price_gwei);
    
    // Status
    bool activate_blockchain(BlockchainID id);
    bool deactivate_blockchain(BlockchainID id);
    bool set_maintenance(BlockchainID id, bool maintenance);
    
    // RPC health check
    bool check_rpc_health(BlockchainID id);
    double get_gas_price(BlockchainID id);
    
    // Stats
    struct BlockchainStats {
        int64_t total;
        int64_t active;
        int64_t inactive;
        int64_t maintenance;
    };
    
    BlockchainStats get_blockchain_stats();
    
private:
    BlockchainService() = default;
};

// ============================================================================
// White Label Service
// ============================================================================

class WhiteLabelService {
public:
    static WhiteLabelService& instance();
    
    void initialize();
    
    // List white labels
    struct WhiteLabelListParams {
        std::optional<WhiteLabelStatus> status;
        std::string search;
        int page = 1;
        int page_size = 20;
    };
    
    struct WhiteLabelListResult {
        std::vector<WhiteLabel> white_labels;
        int64_t total;
        int page;
        int page_size;
    };
    
    WhiteLabelListResult list_white_labels(const WhiteLabelListParams& params);
    
    // Get white label
    std::optional<WhiteLabel> get_white_label(WhiteLabelID id);
    std::optional<WhiteLabel> get_white_label_by_domain(const std::string& domain);
    std::optional<WhiteLabel> get_white_label_by_client_id(const std::string& client_id);
    
    // Create white label
    WhiteLabel create_white_label(const std::string& company_name,
                                   const std::string& domain,
                                   AdminID admin_user_id,
                                   const std::string& contact_email);
    
    // Update white label
    bool update_white_label(WhiteLabelID id,
                           const std::optional<std::string>& company_name,
                           const std::optional<std::string>& logo_url,
                           const std::optional<std::string>& primary_color,
                           const std::optional<std::string>& secondary_color,
                           const std::optional<std::string>& theme_mode,
                           const std::optional<JSON>& features);
    
    // Limits
    bool update_limits(WhiteLabelID id, int max_users, double max_daily_volume);
    
    // Fees
    bool update_fees(WhiteLabelID id, double platform_fee_percent,
                     double custom_fee_percent);
    
    // Status
    bool activate_white_label(WhiteLabelID id);
    bool suspend_white_label(WhiteLabelID id);
    bool terminate_white_label(WhiteLabelID id);
    bool verify_domain(WhiteLabelID id);
    
    // Stats
    struct WhiteLabelStats {
        int64_t total;
        int64_t active;
        int64_t suspended;
        int64_t pending;
    };
    
    WhiteLabelStats get_white_label_stats();
    
private:
    WhiteLabelService() = default;
};

// ============================================================================
// Webhook Service
// ============================================================================

class WebhookService {
public:
    static WebhookService& instance();
    
    void initialize();
    
    // List webhooks
    std::vector<Webhook> list_webhooks();
    
    // Get webhook
    std::optional<Webhook> get_webhook(uint64_t id);
    
    // Create webhook
    Webhook create_webhook(const std::string& name,
                          const std::string& url,
                          const std::vector<std::string>& events,
                          AdminID created_by);
    
    // Update webhook
    bool update_webhook(uint64_t id,
                       const std::optional<std::string>& name,
                       const std::optional<std::string>& url,
                       const std::optional<std::vector<std::string>>& events);
    
    // Status
    bool activate_webhook(uint64_t id);
    bool deactivate_webhook(uint64_t id);
    
    // Test
    bool test_webhook(uint64_t id);
    
    // Delete
    bool delete_webhook(uint64_t id);
    
    // Trigger
    void trigger_event(const std::string& event, const JSON& data);
    
private:
    WebhookService() = default;
    
    void send_webhook(const Webhook& webhook, const std::string& event,
                     const JSON& data);
};

// ============================================================================
// Ticket Service
// ============================================================================

class TicketService {
public:
    static TicketService& instance();
    
    void initialize();
    
    // List tickets
    struct TicketListParams {
        std::optional<TicketStatus> status;
        std::optional<TicketPriority> priority;
        std::optional<AdminID> assigned_to;
        std::optional<UserID> created_by;
        std::string search;
        int page = 1;
        int page_size = 20;
    };
    
    struct TicketListResult {
        std::vector<Ticket> tickets;
        int64_t total;
        int page;
        int page_size;
    };
    
    TicketListResult list_tickets(const TicketListParams& params);
    
    // Get ticket
    std::optional<Ticket> get_ticket(TicketID id);
    std::vector<TicketMessage> get_ticket_messages(TicketID ticket_id);
    
    // Create ticket
    Ticket create_ticket(const std::string& title,
                        const std::string& description,
                        const std::string& ticket_type,
                        TicketPriority priority,
                        UserID created_by);
    
    // Update ticket
    bool update_ticket_status(TicketID id, TicketStatus status);
    bool assign_ticket(TicketID id, AdminID assigned_to);
    
    // Add message
    TicketMessage add_message(TicketID id, const std::string& message,
                             bool is_internal, UserID user_id);
    
    // Close
    bool close_ticket(TicketID id);
    
    // Stats
    struct TicketStats {
        int64_t total;
        int64_t open;
        int64_t in_progress;
        int64_t resolved;
        int64_t closed;
    };
    
    TicketStats get_ticket_stats();
    
private:
    TicketService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_BLOCKCHAIN_HPP
