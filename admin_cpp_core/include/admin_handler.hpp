/**
 * TigerAdmin C++ Core - Main Handler
 * Routes requests to appropriate services
 */

#ifndef TIGER_ADMIN_HANDLER_HPP
#define TIGER_ADMIN_HANDLER_HPP

#include "admin_server.hpp"
#include "admin_auth.hpp"
#include "admin_kyc.hpp"
#include "admin_transactions.hpp"
#include "admin_tokens.hpp"
#include "admin_blockchain.hpp"
#include "admin_analytics.hpp"
#include "admin_websocket.hpp"
#include "admin_cache.hpp"
#include "admin_security.hpp"
#include "admin_audit.hpp"
#include "admin_rate_limiter.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Request Handler
// ============================================================================

class AdminHandler {
public:
    static AdminHandler& instance();
    
    void initialize();
    
    // Set server reference
    void set_server(AdminServer* server);
    
    // Auth handlers
    HttpResponse handle_login(const HttpRequest& req);
    HttpResponse handle_logout(const HttpRequest& req);
    HttpResponse handle_refresh_token(const HttpRequest& req);
    HttpResponse handle_2fa_setup(const HttpRequest& req);
    HttpResponse handle_2fa_verify(const HttpRequest& req);
    HttpResponse handle_change_password(const HttpRequest& req);
    
    // Admin handlers
    HttpResponse handle_list_admins(const HttpRequest& req);
    HttpResponse handle_get_admin(const HttpRequest& req);
    HttpResponse handle_create_admin(const HttpRequest& req);
    HttpResponse handle_update_admin(const HttpRequest& req);
    HttpResponse handle_delete_admin(const HttpRequest& req);
    HttpResponse handle_suspend_admin(const HttpRequest& req);
    HttpResponse handle_activate_admin(const HttpRequest& req);
    
    // User handlers
    HttpResponse handle_list_users(const HttpRequest& req);
    HttpResponse handle_get_user(const HttpRequest& req);
    HttpResponse handle_update_user(const HttpRequest& req);
    HttpResponse handle_suspend_user(const HttpRequest& req);
    HttpResponse handle_ban_user(const HttpRequest& req);
    HttpResponse handle_unban_user(const HttpRequest& req);
    
    // KYC handlers
    HttpResponse handle_list_kyc(const HttpRequest& req);
    HttpResponse handle_get_kyc(const HttpRequest& req);
    HttpResponse handle_approve_kyc(const HttpRequest& req);
    HttpResponse handle_reject_kyc(const HttpRequest& req);
    
    // Transaction handlers
    HttpResponse handle_list_transactions(const HttpRequest& req);
    HttpResponse handle_get_transaction(const HttpRequest& req);
    HttpResponse handle_flag_transaction(const HttpRequest& req);
    HttpResponse handle_unflag_transaction(const HttpRequest& req);
    
    // Withdrawal handlers
    HttpResponse handle_list_withdrawals(const HttpRequest& req);
    HttpResponse handle_get_withdrawal(const HttpRequest& req);
    HttpResponse handle_approve_withdrawal(const HttpRequest& req);
    HttpResponse handle_reject_withdrawal(const HttpRequest& req);
    HttpResponse handle_process_withdrawal(const HttpRequest& req);
    
    // Token handlers
    HttpResponse handle_list_tokens(const HttpRequest& req);
    HttpResponse handle_get_token(const HttpRequest& req);
    HttpResponse handle_create_token(const HttpRequest& req);
    HttpResponse handle_update_token(const HttpRequest& req);
    HttpResponse handle_delete_token(const HttpRequest& req);
    HttpResponse handle_verify_token(const HttpRequest& req);
    
    // Pair handlers
    HttpResponse handle_list_pairs(const HttpRequest& req);
    HttpResponse handle_get_pair(const HttpRequest& req);
    HttpResponse handle_create_pair(const HttpRequest& req);
    HttpResponse handle_update_pair(const HttpRequest& req);
    HttpResponse handle_delete_pair(const HttpRequest& req);
    HttpResponse handle_halt_pair(const HttpRequest& req);
    
    // Blockchain handlers
    HttpResponse handle_list_blockchains(const HttpRequest& req);
    HttpResponse handle_get_blockchain(const HttpRequest& req);
    HttpResponse handle_create_blockchain(const HttpRequest& req);
    HttpResponse handle_update_blockchain(const HttpRequest& req);
    HttpResponse handle_delete_blockchain(const HttpRequest& req);
    
    // Fee handlers
    HttpResponse handle_list_fees(const HttpRequest& req);
    HttpResponse handle_create_fee(const HttpRequest& req);
    HttpResponse handle_update_fee(const HttpRequest& req);
    HttpResponse handle_delete_fee(const HttpRequest& req);
    
    // White label handlers
    HttpResponse handle_list_whitelabels(const HttpRequest& req);
    HttpResponse handle_get_whitelabel(const HttpRequest& req);
    HttpResponse handle_create_whitelabel(const HttpRequest& req);
    HttpResponse handle_update_whitelabel(const HttpRequest& req);
    HttpResponse handle_delete_whitelabel(const HttpRequest& req);
    HttpResponse handle_activate_whitelabel(const HttpRequest& req);
    HttpResponse handle_suspend_whitelabel(const HttpRequest& req);
    
    // Webhook handlers
    HttpResponse handle_list_webhooks(const HttpRequest& req);
    HttpResponse handle_create_webhook(const HttpRequest& req);
    HttpResponse handle_update_webhook(const HttpRequest& req);
    HttpResponse handle_delete_webhook(const HttpRequest& req);
    HttpResponse handle_test_webhook(const HttpRequest& req);
    
    // Ticket handlers
    HttpResponse handle_list_tickets(const HttpRequest& req);
    HttpResponse handle_get_ticket(const HttpRequest& req);
    HttpResponse handle_create_ticket(const HttpRequest& req);
    HttpResponse handle_update_ticket(const HttpRequest& req);
    HttpResponse handle_assign_ticket(const HttpRequest& req);
    HttpResponse handle_add_ticket_message(const HttpRequest& req);
    
    // Analytics handlers
    HttpResponse handle_dashboard_stats(const HttpRequest& req);
    HttpResponse handle_user_analytics(const HttpRequest& req);
    HttpResponse handle_transaction_analytics(const HttpRequest& req);
    HttpResponse handle_revenue_analytics(const HttpRequest& req);
    HttpResponse handle_chart_data(const HttpRequest& req);
    
    // Audit handlers
    HttpResponse handle_audit_logs(const HttpRequest& req);
    HttpResponse handle_export_audit_logs(const HttpRequest& req);
    
    // Feature flag handlers
    HttpResponse handle_list_feature_flags(const HttpRequest& req);
    HttpResponse handle_set_feature_flag(const HttpRequest& req);
    
    // Notification handlers
    HttpResponse handle_list_notifications(const HttpRequest& req);
    HttpResponse handle_mark_notification_read(const HttpRequest& req);
    HttpResponse handle_send_notification(const HttpRequest& req);
    HttpResponse handle_broadcast_notification(const HttpRequest& req);
    
    // IP whitelist handlers
    HttpResponse handle_list_ip_whitelist(const HttpRequest& req);
    HttpResponse handle_add_ip_whitelist(const HttpRequest& req);
    HttpResponse handle_remove_ip_whitelist(const HttpRequest& req);
    
    // Backup handlers
    HttpResponse handle_list_backups(const HttpRequest& req);
    HttpResponse handle_create_backup(const HttpRequest& req);
    HttpResponse handle_restore_backup(const HttpRequest& req);
    HttpResponse handle_delete_backup(const HttpRequest& req);
    
    // Health check
    HttpResponse handle_health(const HttpRequest& req);
    
private:
    AdminHandler() = default;
    AdminServer* server_ = nullptr;
    
    // Helper methods
    std::optional<Admin> get_current_admin(const HttpRequest& req);
    bool has_permission(const Admin& admin, const std::string& permission);
    HttpResponse unauthorized(const std::string& message);
    HttpResponse forbidden(const std::string& message);
    HttpResponse not_found(const std::string& resource);
    HttpResponse bad_request(const std::string& message);
    HttpResponse server_error(const std::string& message);
    
    // JSON helpers
    std::string to_json(const std::map<std::string, std::string>& data);
    std::string to_json(const std::vector<std::string>& data);
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_HANDLER_HPP
