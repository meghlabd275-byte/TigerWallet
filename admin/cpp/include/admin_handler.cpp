/**
 * TigerAdmin C++ Core - Handler Implementation
 */

#include "admin_handler.hpp"
#include "admin_logger.hpp"

#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>
#include <cerrno>
#include <cstring>
#include <cstdlib>
#include <cctype>

namespace tiger {
namespace admin {

AdminHandler& AdminHandler::instance() {
    static AdminHandler handler;
    return handler;
}

void AdminHandler::initialize() {
    LOG_INFO("Admin handler initialized");
    register_all_routes();
}

void AdminHandler::set_server(AdminServer* server) {
    server_ = server;
}

void AdminHandler::register_all_routes() {
    if (!server_) return;
    
    auto& router = server_->router();
    
    // Auth routes
    router.post("/api/v1/auth/login", [this](const HttpRequest& req) { return handle_login(req); });
    router.post("/api/v1/auth/logout", [this](const HttpRequest& req) { return handle_logout(req); });
    router.post("/api/v1/auth/refresh", [this](const HttpRequest& req) { return handle_refresh_token(req); });
    
    // Admin routes
    router.get("/api/v1/admins", [this](const HttpRequest& req) { return handle_list_admins(req); });
    router.get("/api/v1/admins/{id}", [this](const HttpRequest& req) { return handle_get_admin(req); });
    router.post("/api/v1/admins", [this](const HttpRequest& req) { return handle_create_admin(req); });
    router.put("/api/v1/admins/{id}", [this](const HttpRequest& req) { return handle_update_admin(req); });
    router.delete("/api/v1/admins/{id}", [this](const HttpRequest& req) { return handle_delete_admin(req); });
    
    // User routes
    router.get("/api/v1/users", [this](const HttpRequest& req) { return handle_list_users(req); });
    router.get("/api/v1/users/{id}", [this](const HttpRequest& req) { return handle_get_user(req); });
    router.put("/api/v1/users/{id}", [this](const HttpRequest& req) { return handle_update_user(req); });
    router.post("/api/v1/users/{id}/ban", [this](const HttpRequest& req) { return handle_ban_user(req); });
    router.post("/api/v1/users/{id}/suspend", [this](const HttpRequest& req) { return handle_suspend_user(req); });
    
    // KYC routes
    router.get("/api/v1/kyc", [this](const HttpRequest& req) { return handle_list_kyc(req); });
    router.post("/api/v1/kyc/{id}/approve", [this](const HttpRequest& req) { return handle_approve_kyc(req); });
    router.post("/api/v1/kyc/{id}/reject", [this](const HttpRequest& req) { return handle_reject_kyc(req); });
    
    // Transaction routes
    router.get("/api/v1/transactions", [this](const HttpRequest& req) { return handle_list_transactions(req); });
    router.post("/api/v1/transactions/{id}/flag", [this](const HttpRequest& req) { return handle_flag_transaction(req); });
    
    // Withdrawal routes
    router.get("/api/v1/withdrawals", [this](const HttpRequest& req) { return handle_list_withdrawals(req); });
    router.post("/api/v1/withdrawals/{id}/approve", [this](const HttpRequest& req) { return handle_approve_withdrawal(req); });
    router.post("/api/v1/withdrawals/{id}/reject", [this](const HttpRequest& req) { return handle_reject_withdrawal(req); });
    
    // Token routes
    router.get("/api/v1/tokens", [this](const HttpRequest& req) { return handle_list_tokens(req); });
    router.post("/api/v1/tokens", [this](const HttpRequest& req) { return handle_create_token(req); });
    
    // Pair routes
    router.get("/api/v1/pairs", [this](const HttpRequest& req) { return handle_list_pairs(req); });
    router.post("/api/v1/pairs", [this](const HttpRequest& req) { return handle_create_pair(req); });
    router.post("/api/v1/pairs/{id}/halt", [this](const HttpRequest& req) { return handle_halt_pair(req); });
    
    // Blockchain routes
    router.get("/api/v1/blockchains", [this](const HttpRequest& req) { return handle_list_blockchains(req); });
    router.post("/api/v1/blockchains", [this](const HttpRequest& req) { return handle_create_blockchain(req); });
    
    // White label routes
    router.get("/api/v1/whitelabels", [this](const HttpRequest& req) { return handle_list_whitelabels(req); });
    router.post("/api/v1/whitelabels", [this](const HttpRequest& req) { return handle_create_whitelabel(req); });
    
    // Analytics routes
    router.get("/api/v1/analytics/dashboard", [this](const HttpRequest& req) { return handle_dashboard_stats(req); });
    
    // Health check
    router.get("/health", [this](const HttpRequest& req) { return handle_health(req); });
    
    LOG_INFO("All routes registered");
}

// Auth handlers — proxied to the canonical admin/go backend. The C++ surface
// does not mint its own tokens; it forwards credentials to admin/go and
// returns the real JWT (or 401).
HttpResponse AdminHandler::handle_login(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_logout(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_refresh_token(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_2fa_setup(const HttpRequest& req) {
    return proxy_to_admin_go(req, "/api/v1/auth/2fa/setup", HttpMethod::POST);
}

HttpResponse AdminHandler::handle_2fa_verify(const HttpResponse& req) {
    // NOTE: signature kept for ABI compat with the header; 2FA verify is a
    // POST and is forwarded by handle_2fa_setup's callers via admin/go.
    return HttpResponse::error(HttpStatus::BAD_REQUEST,
                               "2FA verify must be called via POST /api/v1/auth/2fa/verify");
}

HttpResponse AdminHandler::handle_change_password(const HttpRequest& req) {
    return proxy_to_admin_go(req, "/api/v1/auth/change-password", HttpMethod::PUT);
}

// Admin handlers
HttpResponse AdminHandler::handle_list_admins(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_admin(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_create_admin(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_admin(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_delete_admin(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_suspend_admin(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_activate_admin(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// User handlers
HttpResponse AdminHandler::handle_list_users(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_user(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_user(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_suspend_user(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_ban_user(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_unban_user(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// KYC handlers
HttpResponse AdminHandler::handle_list_kyc(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_kyc(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_approve_kyc(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_reject_kyc(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Transaction handlers
HttpResponse AdminHandler::handle_list_transactions(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_transaction(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_flag_transaction(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_unflag_transaction(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Withdrawal handlers
HttpResponse AdminHandler::handle_list_withdrawals(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_withdrawal(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_approve_withdrawal(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_reject_withdrawal(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_process_withdrawal(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Token handlers
HttpResponse AdminHandler::handle_list_tokens(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_token(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_create_token(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_token(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_delete_token(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_verify_token(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Pair handlers
HttpResponse AdminHandler::handle_list_pairs(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_pair(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_create_pair(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_pair(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_delete_pair(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_halt_pair(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Blockchain handlers
HttpResponse AdminHandler::handle_list_blockchains(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_blockchain(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_create_blockchain(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_blockchain(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_delete_blockchain(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Fee handlers
HttpResponse AdminHandler::handle_list_fees(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_create_fee(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_fee(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_delete_fee(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// White label handlers
HttpResponse AdminHandler::handle_list_whitelabels(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_get_whitelabel(const HttpRequest& req) {
    return proxy_to_admin_go(req, "", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_create_whitelabel(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_update_whitelabel(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_delete_whitelabel(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_activate_whitelabel(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

HttpResponse AdminHandler::handle_suspend_whitelabel(const HttpRequest& req) {
    return proxy_to_admin_go(req);
}

// Webhook handlers
HttpResponse AdminHandler::handle_list_webhooks(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_create_webhook(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_update_webhook(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_delete_webhook(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_test_webhook(const HttpRequest& req) { return proxy_to_admin_go(req); }

// Ticket handlers
HttpResponse AdminHandler::handle_list_tickets(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_get_ticket(const HttpRequest& req) { return proxy_to_admin_go(req, "", HttpMethod::GET); }
HttpResponse AdminHandler::handle_create_ticket(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_update_ticket(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_assign_ticket(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_add_ticket_message(const HttpRequest& req) { return proxy_to_admin_go(req); }

// Analytics handlers
HttpResponse AdminHandler::handle_dashboard_stats(const HttpRequest& req) {
    return proxy_to_admin_go(req, "/api/v1/dashboard", HttpMethod::GET);
}

HttpResponse AdminHandler::handle_user_analytics(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/analytics/users", HttpMethod::GET); }
HttpResponse AdminHandler::handle_transaction_analytics(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/analytics/transactions", HttpMethod::GET); }
HttpResponse AdminHandler::handle_revenue_analytics(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/analytics/revenue", HttpMethod::GET); }
HttpResponse AdminHandler::handle_chart_data(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/analytics/custom", HttpMethod::GET); }

// Audit handlers
HttpResponse AdminHandler::handle_audit_logs(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/audit-logs", HttpMethod::GET); }
HttpResponse AdminHandler::handle_export_audit_logs(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/audit-logs/export", HttpMethod::POST); }

// Feature flag handlers
HttpResponse AdminHandler::handle_list_feature_flags(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/feature-flags", HttpMethod::GET); }
HttpResponse AdminHandler::handle_set_feature_flag(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/feature-flags", HttpMethod::POST); }

// Notification handlers
HttpResponse AdminHandler::handle_list_notifications(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/notifications", HttpMethod::GET); }
HttpResponse AdminHandler::handle_mark_notification_read(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_send_notification(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/notifications", HttpMethod::POST); }
HttpResponse AdminHandler::handle_broadcast_notification(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/notifications/broadcast", HttpMethod::POST); }

// IP whitelist handlers
HttpResponse AdminHandler::handle_list_ip_whitelist(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/ip-whitelist", HttpMethod::GET); }
HttpResponse AdminHandler::handle_add_ip_whitelist(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/ip-whitelist", HttpMethod::POST); }
HttpResponse AdminHandler::handle_remove_ip_whitelist(const HttpRequest& req) { return proxy_to_admin_go(req); }

// Backup handlers
HttpResponse AdminHandler::handle_list_backups(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/backups", HttpMethod::GET); }
HttpResponse AdminHandler::handle_create_backup(const HttpRequest& req) { return proxy_to_admin_go(req, "/api/v1/backups", HttpMethod::POST); }
HttpResponse AdminHandler::handle_restore_backup(const HttpRequest& req) { return proxy_to_admin_go(req); }
HttpResponse AdminHandler::handle_delete_backup(const HttpRequest& req) { return proxy_to_admin_go(req); }

// Health check
HttpResponse AdminHandler::handle_health(const HttpRequest& req) {
    return HttpResponse::json(HttpStatus::OK, "{\"status\":\"healthy\",\"service\":\"tiger-admin-cpp\"}");
}

// ============================================================================
// Upstream proxy to canonical admin/go backend
// ============================================================================
//
// The admin/go backend (:9093) is the source of truth for every admin
// resource. proxy_to_admin_go() performs a blocking HTTP/1.1 call over a TCP
// socket, forwarding the inbound method/path/query/body + Bearer JWT, and
// returns the real upstream status + body. No data is fabricated; a dead
// upstream surfaces as 503.

namespace {

std::string upstream_go_host() {
    const char* h = std::getenv("TIGERADMIN_UPSTREAM_HOST");
    return h && *h ? std::string(h) : std::string("localhost");
}

int upstream_go_port() {
    const char* p = std::getenv("TIGERADMIN_UPSTREAM_PORT");
    if (p && *p) {
        try { return std::stoi(p); } catch (...) {}
    }
    return 9093;
}

const char* method_str(HttpMethod m) {
    switch (m) {
        case HttpMethod::GET:    return "GET";
        case HttpMethod::POST:   return "POST";
        case HttpMethod::PUT:    return "PUT";
        case HttpMethod::DELETE: return "DELETE";
        case HttpMethod::PATCH:  return "PATCH";
    }
    return "GET";
}

HttpStatus status_from_code(int code) {
    switch (code) {
        case 200: return HttpStatus::OK;
        case 201: return HttpStatus::CREATED;
        case 204: return HttpStatus::NO_CONTENT;
        case 400: return HttpStatus::BAD_REQUEST;
        case 401: return HttpStatus::UNAUTHORIZED;
        case 403: return HttpStatus::FORBIDDEN;
        case 404: return HttpStatus::NOT_FOUND;
        case 409: return HttpStatus::CONFLICT;
        case 422: return HttpStatus::UNPROCESSABLE;
        default:
            if (code >= 500) return HttpStatus::INTERNAL_ERROR;
            return HttpStatus::OK;
    }
}

std::size_t find_subseq(const std::string& h, const std::string& n) {
    return h.find(n);
}

// Decodes an HTTP chunked-transfer body into its concatenation.
std::string dechunk(const std::string& chunked) {
    std::string out;
    std::size_t pos = 0;
    while (pos < chunked.size()) {
        std::size_t eol = chunked.find("\r\n", pos);
        if (eol == std::string::npos) break;
        std::string len_str = chunked.substr(pos, eol - pos);
        std::size_t chunk_len = 0;
        try { chunk_len = std::stoul(len_str, nullptr, 16); } catch (...) { break; }
        if (chunk_len == 0) break;
        std::size_t data_start = eol + 2;
        if (data_start + chunk_len > chunked.size()) break;
        out.append(chunked, data_start, chunk_len);
        pos = data_start + chunk_len + 2;
    }
    return out;
}

} // namespace

HttpResponse AdminHandler::proxy_to_admin_go(const HttpRequest& req,
                                             const std::string& path_override,
                                             HttpMethod method_override) {
    const std::string host = upstream_go_host();
    const int port = upstream_go_port();
    const HttpMethod method = method_override != HttpMethod::GET || !path_override.empty()
        ? method_override : req.method;

    // Build path: explicit override wins; otherwise use the inbound path +
    // re-append its query string.
    std::string path = !path_override.empty() ? path_override : req.path;
    if (path_override.empty()) {
        std::string qs;
        bool first = true;
        for (const auto& kv : req.query_params) {
            qs += (first ? "?" : "&");
            qs += kv.first + "=" + kv.second;
            first = false;
        }
        path += qs;
    }

    // Resolve Bearer token from the inbound Authorization header.
    std::string bearer;
    auto auth = req.get_header("Authorization");
    if (auth) {
        const std::string& v = auth.value();
        if (v.rfind("Bearer ", 0) == 0) bearer = v.substr(7);
        else if (v.rfind("bearer ", 0) == 0) bearer = v.substr(7);
        else bearer = v;
    }
    if (bearer.empty()) {
        return HttpResponse::error(HttpStatus::UNAUTHORIZED, "bearer token required");
    }

    // TCP connect to the admin/go backend.
    struct addrinfo hints{}, *res = nullptr;
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    std::string port_str = std::to_string(port);
    if (getaddrinfo(host.c_str(), port_str.c_str(), &hints, &res) != 0 || !res) {
        return HttpResponse::error(HttpStatus::SERVICE_UNAVAILABLE,
                                   "upstream admin/go unavailable (dns)");
    }
    int fd = ::socket(res->ai_family, res->ai_socktype, res->ai_protocol);
    if (fd < 0) {
        freeaddrinfo(res);
        return HttpResponse::error(HttpStatus::SERVICE_UNAVAILABLE,
                                   "upstream admin/go unavailable (socket)");
    }
    // 5s connect/send/recv timeout so a hung backend surfaces as an error.
    struct timeval tv;
    tv.tv_sec = 5;
    tv.tv_usec = 0;
    ::setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    ::setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
    if (::connect(fd, res->ai_addr, res->ai_addrlen) < 0) {
        ::close(fd);
        freeaddrinfo(res);
        return HttpResponse::error(HttpStatus::SERVICE_UNAVAILABLE,
                                   "upstream admin/go unavailable (connect)");
    }
    freeaddrinfo(res);

    // Build the HTTP/1.1 request.
    std::string body = req.body;
    std::string request;
    request += std::string(method_str(method)) + " " + path + " HTTP/1.1\r\n";
    request += "Host: " + host + ":" + port_str + "\r\n";
    request += "Connection: close\r\n";
    request += "Authorization: Bearer " + bearer + "\r\n";
    request += "Content-Type: application/json\r\n";
    request += "Content-Length: " + std::to_string(body.size()) + "\r\n";
    request += "\r\n";
    request += body;

    // Send.
    std::size_t sent = 0;
    while (sent < request.size()) {
        ssize_t n = ::send(fd, request.data() + sent, request.size() - sent, 0);
        if (n <= 0) {
            ::close(fd);
            return HttpResponse::error(HttpStatus::SERVICE_UNAVAILABLE,
                                       "upstream admin/go write failed");
        }
        sent += static_cast<std::size_t>(n);
    }

    // Receive full response (Connection: close => server closes after body).
    std::string raw;
    char buf[8192];
    for (;;) {
        ssize_t n = ::recv(fd, buf, sizeof(buf), 0);
        if (n <= 0) break;
        raw.append(buf, static_cast<std::size_t>(n));
    }
    ::close(fd);

    if (raw.empty()) {
        return HttpResponse::error(HttpStatus::SERVICE_UNAVAILABLE,
                                   "empty upstream response");
    }

    // Split headers / body at the blank line.
    std::size_t sep = find_subseq(raw, "\r\n\r\n");
    std::string header_block = (sep == std::string::npos) ? raw : raw.substr(0, sep);
    std::string body_bytes = (sep == std::string::npos) ? std::string() : raw.substr(sep + 4);

    // Status code from the first line: "HTTP/1.1 200 OK".
    std::size_t first_eol = header_block.find("\r\n");
    std::string first_line = (first_eol == std::string::npos)
        ? header_block : header_block.substr(0, first_eol);
    int status_code = 502;
    {
        std::istringstream iss(first_line);
        std::string ver; int code; std::string reason;
        iss >> ver >> code >> std::ws;
        std::getline(iss, reason);
        if (iss) status_code = code;
    }

    // De-chunk if Transfer-Encoding: chunked.
    std::string lower = header_block;
    std::transform(lower.begin(), lower.end(), lower.begin(),
                   [](unsigned char c) { return std::tolower(c); });
    if (lower.find("transfer-encoding: chunked") != std::string::npos) {
        body_bytes = dechunk(body_bytes);
    }

    return HttpResponse::json(status_from_code(status_code), body_bytes);
}

// Helper methods
std::optional<Admin> AdminHandler::get_current_admin(const HttpRequest& req) {
    auto auth_header = req.get_header("Authorization");
    if (!auth_header) return std::nullopt;
    
    std::string token = auth_header.value();
    if (token.rfind("Bearer ", 0) == 0) {
        token = token.substr(7);
    }
    
    return AuthService::instance().validate_token(token);
}

bool AdminHandler::has_permission(const Admin& admin, const std::string& permission) {
    return std::find(admin.permissions.begin(), admin.permissions.end(), permission) != admin.permissions.end();
}

HttpResponse AdminHandler::unauthorized(const std::string& message) {
    return HttpResponse::error(HttpStatus::UNAUTHORIZED, message);
}

HttpResponse AdminHandler::forbidden(const std::string& message) {
    return HttpResponse::error(HttpStatus::FORBIDDEN, message);
}

HttpResponse AdminHandler::not_found(const std::string& resource) {
    return HttpResponse::error(HttpStatus::NOT_FOUND, resource + " not found");
}

HttpResponse AdminHandler::bad_request(const std::string& message) {
    return HttpResponse::error(HttpStatus::BAD_REQUEST, message);
}

HttpResponse AdminHandler::server_error(const std::string& message) {
    return HttpResponse::error(HttpStatus::INTERNAL_SERVER_ERROR, message);
}

std::string AdminHandler::to_json(const std::map<std::string, std::string>& data) {
    std::string json = "{";
    bool first = true;
    for (const auto& pair : data) {
        if (!first) json += ",";
        json += "\"" + pair.first + "\":\"" + pair.second + "\"";
        first = false;
    }
    json += "}";
    return json;
}

std::string AdminHandler::to_json(const std::vector<std::string>& data) {
    std::string json = "[";
    bool first = true;
    for (const auto& item : data) {
        if (!first) json += ",";
        json += "\"" + item + "\"";
        first = false;
    }
    json += "]";
    return json;
}

} // namespace admin
} // namespace tiger
