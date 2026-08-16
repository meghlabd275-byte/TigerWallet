/**
 * TigerAdmin C++ Core - Blockchain Header
 */
#pragma once

#include "admin_security.hpp"
#include "admin_kyc.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

using BlockchainID = uint64_t;
using WhiteLabelID = uint64_t;
using TicketID = uint64_t;

enum class BlockchainStatus {
    ACTIVE = 0,
    INACTIVE = 1,
    MAINTENANCE = 2,
    DEPRECATED = 3
};

enum class WhiteLabelStatus {
    ACTIVE = 0,
    SUSPENDED = 1,
    PENDING = 2,
    TERMINATED = 3
};

enum class TicketPriority { LOW = 0, MEDIUM = 1, HIGH = 2, URGENT = 3 };
enum class TicketStatus {
    OPEN = 0,
    IN_PROGRESS = 1,
    WAITING_CUSTOMER = 2,
    RESOLVED = 3,
    CLOSED = 4
};

struct Blockchain {
    BlockchainID id = 0;
    std::string name;
    std::string symbol;
    int chain_id = 0;
    bool is_evm = false;
    std::string rpc_url;
    std::string explorer_url;
    std::string native_token;
    int decimals = 0;
    bool is_active = true;
    BlockchainStatus status = BlockchainStatus::ACTIVE;
    int64_t created_at = 0;
};

struct WhiteLabel {
    WhiteLabelID id = 0;
    std::string company_name;
    std::string domain;
    AdminID admin_user_id = 0;
    std::string contact_email;
    std::string logo_url;
    std::string primary_color;
    std::string secondary_color;
    std::string theme_mode;
    JSON features;
    int max_users = 0;
    double max_daily_volume = 0.0;
    double platform_fee_percent = 0.0;
    double custom_fee_percent = 0.0;
    std::string client_id;
    std::string client_secret;
    WhiteLabelStatus status = WhiteLabelStatus::PENDING;
    bool domain_verified = false;
    int64_t created_at = 0;
};

struct Webhook {
    uint64_t id = 0;
    std::string name;
    std::string url;
    std::string secret;
    std::vector<std::string> events;
    AdminID created_by = 0;
    bool is_active = true;
    int64_t created_at = 0;
};

struct Ticket {
    TicketID id = 0;
    std::string title;
    std::string description;
    std::string ticket_type;
    TicketPriority priority = TicketPriority::MEDIUM;
    TicketStatus status = TicketStatus::OPEN;
    AdminID assigned_to = 0;
    UserID created_by = 0;
    int64_t created_at = 0;
    int64_t updated_at = 0;
};

struct TicketMessage {
    uint64_t id = 0;
    TicketID ticket_id = 0;
    std::string message;
    bool is_internal = false;
    UserID created_by = 0;
    int64_t created_at = 0;
};

class BlockchainService {
public:
    struct BlockchainStats {
        int64_t total = 0;
        int64_t active = 0;
        int64_t inactive = 0;
        int64_t in_maintenance = 0;
    };

    static BlockchainService& instance();

    void initialize();

    std::vector<Blockchain> list_blockchains(bool active_only);
    std::optional<Blockchain> get_blockchain(BlockchainID id);
    std::optional<Blockchain> get_blockchain_by_chain_id(int chain_id);

    Blockchain create_blockchain(const std::string& name, const std::string& symbol,
                                 int chain_id, bool is_evm,
                                 const std::string& rpc_url,
                                 const std::string& explorer_url,
                                 const std::string& native_token, int decimals);
    bool update_blockchain(BlockchainID id,
                           const std::optional<std::string>& rpc_url,
                           const std::optional<std::string>& explorer_url,
                           const std::optional<std::string>& avg_gas_price_gwei);
    bool activate_blockchain(BlockchainID id);
    bool deactivate_blockchain(BlockchainID id);
    bool set_maintenance(BlockchainID id, bool maintenance);
    bool check_rpc_health(BlockchainID id);
    double get_gas_price(BlockchainID id);
    BlockchainStats get_blockchain_stats();
};

class WhiteLabelService {
public:
    struct WhiteLabelListParams {
        int page = 1;
        int page_size = 20;
        std::optional<WhiteLabelStatus> status;
        std::optional<std::string> search;
    };

    struct WhiteLabelListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<WhiteLabel> whitelabels;
    };

    struct WhiteLabelStats {
        int64_t total = 0;
        int64_t active = 0;
        int64_t suspended = 0;
        int64_t pending = 0;
    };

    static WhiteLabelService& instance();

    void initialize();

    WhiteLabelListResult list_white_labels(const WhiteLabelListParams& params);
    std::optional<WhiteLabel> get_white_label(WhiteLabelID id);
    std::optional<WhiteLabel> get_white_label_by_domain(const std::string& domain);
    std::optional<WhiteLabel> get_white_label_by_client_id(const std::string& client_id);

    WhiteLabel create_white_label(const std::string& company_name,
                                  const std::string& domain,
                                  AdminID admin_user_id,
                                  const std::string& contact_email);
    bool update_white_label(WhiteLabelID id,
                            const std::optional<std::string>& company_name,
                            const std::optional<std::string>& logo_url,
                            const std::optional<std::string>& primary_color,
                            const std::optional<std::string>& secondary_color,
                            const std::optional<std::string>& theme_mode,
                            const std::optional<JSON>& features);
    bool update_limits(WhiteLabelID id, int max_users, double max_daily_volume);
    bool update_fees(WhiteLabelID id, double platform_fee_percent,
                     double custom_fee_percent);
    bool activate_white_label(WhiteLabelID id);
    bool suspend_white_label(WhiteLabelID id);
    bool terminate_white_label(WhiteLabelID id);
    bool verify_domain(WhiteLabelID id);
    WhiteLabelStats get_white_label_stats();
};

class WebhookService {
public:
    static WebhookService& instance();

    void initialize();

    std::vector<Webhook> list_webhooks();
    std::optional<Webhook> get_webhook(uint64_t id);

    Webhook create_webhook(const std::string& name, const std::string& url,
                           const std::vector<std::string>& events,
                           AdminID created_by);
    bool update_webhook(uint64_t id, const std::optional<std::string>& name,
                        const std::optional<std::string>& url,
                        const std::optional<std::vector<std::string>>& events);
    bool activate_webhook(uint64_t id);
    bool deactivate_webhook(uint64_t id);
    bool test_webhook(uint64_t id);
    bool delete_webhook(uint64_t id);
    void trigger_event(const std::string& event, const JSON& data);
    void send_webhook(const Webhook& webhook, const std::string& event,
                      const JSON& data);
};

class TicketService {
public:
    struct TicketListParams {
        int page = 1;
        int page_size = 20;
        std::optional<TicketStatus> status;
        std::optional<TicketPriority> priority;
        std::optional<AdminID> assigned_to;
        std::optional<std::string> search;
    };

    struct TicketListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<Ticket> tickets;
    };

    struct TicketStats {
        int64_t total = 0;
        int64_t open = 0;
        int64_t in_progress = 0;
        int64_t resolved = 0;
        int64_t urgent = 0;
    };

    static TicketService& instance();

    void initialize();

    TicketListResult list_tickets(const TicketListParams& params);
    std::optional<Ticket> get_ticket(TicketID id);
    std::vector<TicketMessage> get_ticket_messages(TicketID ticket_id);

    Ticket create_ticket(const std::string& title, const std::string& description,
                         const std::string& ticket_type, TicketPriority priority,
                         UserID created_by);
    bool update_ticket_status(TicketID id, TicketStatus status);
    bool assign_ticket(TicketID id, AdminID assigned_to);
    TicketMessage add_message(TicketID id, const std::string& message,
                              bool is_internal, UserID user_id);
    bool close_ticket(TicketID id);
    TicketStats get_ticket_stats();
};

} // namespace admin
} // namespace tiger
