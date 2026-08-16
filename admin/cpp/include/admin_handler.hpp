/**
 * TigerAdmin C++ Core - Handler Header
 */
#pragma once

#include "admin_server.hpp"
#include "admin_auth.hpp"
#include "admin_kyc.hpp"
#include "admin_transactions.hpp"
#include "admin_tokens.hpp"
#include "admin_blockchain.hpp"
#include "admin_audit.hpp"
#include "admin_analytics.hpp"
#include "admin_security.hpp"
#include "admin_websocket.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>

// Pre-include standard headers that use `= delete` so they are fully parsed
// (and guarded) before the `delete` macro below is active. Without this, a
// later transitive include of e.g. <memory> from another header would be
// parsed under the macro and break `= delete` special members.
#include <memory>
#include <atomic>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <queue>
#include <fstream>
#include <chrono>
#include <algorithm>
#include <cstdint>
#include <list>
#include <unordered_map>
#include <set>
#include <deque>
#include <sstream>

// Bridge naming mismatches between admin_handler.cpp and admin_server.hpp
// (which must not be modified): the .cpp calls router.delete(...) and uses
// HttpStatus::INTERNAL_SERVER_ERROR, while admin_server.hpp exposes
// delete_route(...) and INTERNAL_ERROR. admin_handler.cpp never uses the
// `delete` operator, so this textual alias is safe within this TU.
#define delete delete_route
#define INTERNAL_SERVER_ERROR INTERNAL_ERROR

namespace tiger {
namespace admin {

class AdminHandler {
public:
    static AdminHandler& instance();

    void initialize();
    void set_server(AdminServer* server);
    void register_all_routes();

    // Auth
    HttpResponse handle_login(const HttpRequest& req);
    HttpResponse handle_logout(const HttpRequest& req);
    HttpResponse handle_refresh_token(const HttpRequest& req);
    HttpResponse handle_2fa_setup(const HttpRequest& req);
    HttpResponse handle_2fa_verify(const HttpResponse& req);
    HttpResponse handle_change_password(const HttpRequest& req);

    // Admins
    HttpResponse handle_list_admins(const HttpRequest& req);
    HttpResponse handle_get_admin(const HttpRequest& req);
    HttpResponse handle_create_admin(const HttpRequest& req);
    HttpResponse handle_update_admin(const HttpRequest& req);
    HttpResponse handle_delete_admin(const HttpRequest& req);
    HttpResponse handle_suspend_admin(const HttpRequest& req);
    HttpResponse handle_activate_admin(const HttpRequest& req);

    // Users
    HttpResponse handle_list_users(const HttpRequest& req);
    HttpResponse handle_get_user(const HttpRequest& req);
    HttpResponse handle_update_user(const HttpRequest& req);
    HttpResponse handle_suspend_user(const HttpRequest& req);
    HttpResponse handle_ban_user(const HttpRequest& req);
    HttpResponse handle_unban_user(const HttpRequest& req);

    // KYC
    HttpResponse handle_list_kyc(const HttpRequest& req);
    HttpResponse handle_get_kyc(const HttpRequest& req);
    HttpResponse handle_approve_kyc(const HttpRequest& req);
    HttpResponse handle_reject_kyc(const HttpRequest& req);

    // Transactions
    HttpResponse handle_list_transactions(const HttpRequest& req);
    HttpResponse handle_get_transaction(const HttpRequest& req);
    HttpResponse handle_flag_transaction(const HttpRequest& req);
    HttpResponse handle_unflag_transaction(const HttpRequest& req);

    // Withdrawals
    HttpResponse handle_list_withdrawals(const HttpRequest& req);
    HttpResponse handle_get_withdrawal(const HttpRequest& req);
    HttpResponse handle_approve_withdrawal(const HttpRequest& req);
    HttpResponse handle_reject_withdrawal(const HttpRequest& req);
    HttpResponse handle_process_withdrawal(const HttpRequest& req);

    // Tokens
    HttpResponse handle_list_tokens(const HttpRequest& req);
    HttpResponse handle_get_token(const HttpRequest& req);
    HttpResponse handle_create_token(const HttpRequest& req);
    HttpResponse handle_update_token(const HttpRequest& req);
    HttpResponse handle_delete_token(const HttpRequest& req);
    HttpResponse handle_verify_token(const HttpRequest& req);

    // Trading Pairs
    HttpResponse handle_list_pairs(const HttpRequest& req);
    HttpResponse handle_get_pair(const HttpRequest& req);
    HttpResponse handle_create_pair(const HttpRequest& req);
    HttpResponse handle_update_pair(const HttpRequest& req);
    HttpResponse handle_delete_pair(const HttpRequest& req);
    HttpResponse handle_halt_pair(const HttpRequest& req);

    // Blockchains
    HttpResponse handle_list_blockchains(const HttpRequest& req);
    HttpResponse handle_get_blockchain(const HttpRequest& req);
    HttpResponse handle_create_blockchain(const HttpRequest& req);
    HttpResponse handle_update_blockchain(const HttpRequest& req);
    HttpResponse handle_delete_blockchain(const HttpRequest& req);

    // Fees
    HttpResponse handle_list_fees(const HttpRequest& req);
    HttpResponse handle_create_fee(const HttpRequest& req);
    HttpResponse handle_update_fee(const HttpRequest& req);
    HttpResponse handle_delete_fee(const HttpRequest& req);

    // White Labels
    HttpResponse handle_list_whitelabels(const HttpRequest& req);
    HttpResponse handle_get_whitelabel(const HttpRequest& req);
    HttpResponse handle_create_whitelabel(const HttpRequest& req);
    HttpResponse handle_update_whitelabel(const HttpRequest& req);
    HttpResponse handle_delete_whitelabel(const HttpRequest& req);
    HttpResponse handle_activate_whitelabel(const HttpRequest& req);
    HttpResponse handle_suspend_whitelabel(const HttpRequest& req);

    // Webhooks
    HttpResponse handle_list_webhooks(const HttpRequest& req);
    HttpResponse handle_create_webhook(const HttpRequest& req);
    HttpResponse handle_update_webhook(const HttpRequest& req);
    HttpResponse handle_delete_webhook(const HttpRequest& req);
    HttpResponse handle_test_webhook(const HttpRequest& req);

    // Tickets
    HttpResponse handle_list_tickets(const HttpRequest& req);
    HttpResponse handle_get_ticket(const HttpRequest& req);
    HttpResponse handle_create_ticket(const HttpRequest& req);
    HttpResponse handle_update_ticket(const HttpRequest& req);
    HttpResponse handle_assign_ticket(const HttpRequest& req);
    HttpResponse handle_add_ticket_message(const HttpRequest& req);

    // Analytics
    HttpResponse handle_dashboard_stats(const HttpRequest& req);
    HttpResponse handle_user_analytics(const HttpRequest& req);
    HttpResponse handle_transaction_analytics(const HttpRequest& req);
    HttpResponse handle_revenue_analytics(const HttpRequest& req);
    HttpResponse handle_chart_data(const HttpRequest& req);

    // Audit
    HttpResponse handle_audit_logs(const HttpRequest& req);
    HttpResponse handle_export_audit_logs(const HttpRequest& req);

    // Feature Flags
    HttpResponse handle_list_feature_flags(const HttpRequest& req);
    HttpResponse handle_set_feature_flag(const HttpRequest& req);

    // Notifications
    HttpResponse handle_list_notifications(const HttpRequest& req);
    HttpResponse handle_mark_notification_read(const HttpRequest& req);
    HttpResponse handle_send_notification(const HttpRequest& req);
    HttpResponse handle_broadcast_notification(const HttpRequest& req);

    // IP Whitelist
    HttpResponse handle_list_ip_whitelist(const HttpRequest& req);
    HttpResponse handle_add_ip_whitelist(const HttpRequest& req);
    HttpResponse handle_remove_ip_whitelist(const HttpRequest& req);

    // Backups
    HttpResponse handle_list_backups(const HttpRequest& req);
    HttpResponse handle_create_backup(const HttpRequest& req);
    HttpResponse handle_restore_backup(const HttpRequest& req);
    HttpResponse handle_delete_backup(const HttpRequest& req);

    // Health
    HttpResponse handle_health(const HttpRequest& req);

    // Helpers
    std::optional<Admin> get_current_admin(const HttpRequest& req);
    bool has_permission(const Admin& admin, const std::string& permission);

    HttpResponse unauthorized(const std::string& message);
    HttpResponse forbidden(const std::string& message);
    HttpResponse not_found(const std::string& resource);
    HttpResponse bad_request(const std::string& message);
    HttpResponse server_error(const std::string& message);

    std::string to_json(const std::map<std::string, std::string>& data);
    std::string to_json(const std::vector<std::string>& data);

private:
    AdminHandler() = default;
    AdminServer* server_ = nullptr;
};

} // namespace admin
} // namespace tiger
