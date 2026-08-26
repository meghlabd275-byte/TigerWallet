/**
 * TigerWallet White Label Admin - Domain Handlers (C++)
 *
 * Governance-record handlers for the 11 WL admin domains mirrored across all
 * WL clients (web/android/ios/desktop/extensions/rust). Each handler produces a
 * normalized request descriptor for the WL backend at http://localhost:8456.
 * Governance records ONLY - no fund movement, no balance mutation.
 */
#ifndef TIGER_WL_ADMIN_DOMAINS_HPP
#define TIGER_WL_ADMIN_DOMAINS_HPP

#include <cstdint>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>

namespace tiger {
namespace wl_admin {

enum class HttpMethod : uint8_t { GET, POST, PUT, DELETE };

inline std::string_view to_string_view(HttpMethod m) {
    switch (m) {
        case HttpMethod::GET:    return "GET";
        case HttpMethod::POST:   return "POST";
        case HttpMethod::PUT:    return "PUT";
        case HttpMethod::DELETE: return "DELETE";
    }
    return "GET";
}

// WL backend base. Port 8456 - matches the Go WL admin server.
constexpr std::string_view WL_API_BASE = "http://localhost:8456/api/v1/admin";

struct DomainRequest {
    HttpMethod method;
    std::string path;          // relative to WL_API_BASE
    std::string body_json;     // empty for GET/DELETE without body
    bool governance_action;    // true for status/approve/reject (no fund movement)
};

struct DomainSpec {
    std::string key;
    std::string label;
    std::vector<DomainRequest> endpoints;
    std::vector<DomainRequest> governance_actions;
};

// Build the full URL for a request. Used by the (platform-specific) HTTP layer
// in the desktop/native shells; the C++ side only normalizes the descriptor.
inline std::string url_for(const DomainRequest& req) {
    std::string url(WL_API_BASE);
    url += req.path;
    return url;
}

// Registry of all 11 domains. Mirrors the Go backend routes in
// white_label_admin/go/main.go (RBAC scopes preserved on the server side).
inline const std::vector<DomainSpec>& domain_registry() {
    static const std::vector<DomainSpec> registry = []() {
        std::vector<DomainSpec> v;

        auto add_crud = [](DomainSpec& s, const std::string& base) {
            s.endpoints.push_back({HttpMethod::GET,    base,            "", false});
            s.endpoints.push_back({HttpMethod::POST,   base,            "{}", false});
            s.endpoints.push_back({HttpMethod::PUT,    base + "/:id",   "{}", false});
            s.endpoints.push_back({HttpMethod::DELETE, base + "/:id",   "",   false});
        };

        {
            DomainSpec s{"futures", "Futures", {}, {}};
            add_crud(s, "/futures");
            s.governance_actions.push_back({HttpMethod::PUT, "/futures/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"options", "Options", {}, {}};
            add_crud(s, "/options");
            s.governance_actions.push_back({HttpMethod::PUT, "/options/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"copy-trading", "Copy Trading", {}, {}};
            add_crud(s, "/copy-trading");
            s.governance_actions.push_back({HttpMethod::PUT, "/copy-trading/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"convert", "Convert", {}, {}};
            add_crud(s, "/convert");
            s.governance_actions.push_back({HttpMethod::PUT, "/convert/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"onramp", "On-Ramp", {}, {}};
            add_crud(s, "/onramp");
            s.governance_actions.push_back({HttpMethod::POST, "/onramp/:id/approve", "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/onramp/:id/reject", "{\"reason\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"offramp", "Off-Ramp", {}, {}};
            add_crud(s, "/offramp");
            s.governance_actions.push_back({HttpMethod::POST, "/offramp/:id/approve", "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/offramp/:id/reject", "{\"reason\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"p2p-clients", "P2P Clients", {}, {}};
            add_crud(s, "/p2p-clients");
            s.governance_actions.push_back({HttpMethod::PUT, "/p2p-clients/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"partners", "Partners", {}, {}};
            add_crud(s, "/partners");
            s.governance_actions.push_back({HttpMethod::PUT,  "/partners/:id/status",  "{\"status\":\"\"}", true});
            s.governance_actions.push_back({HttpMethod::POST, "/partners/:id/approve", "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/partners/:id/reject",  "{\"reason\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"rewards", "Rewards", {}, {}};
            add_crud(s, "/rewards");
            s.governance_actions.push_back({HttpMethod::PUT, "/rewards/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            DomainSpec s{"marketing", "Marketing", {}, {}};
            add_crud(s, "/marketing");
            s.governance_actions.push_back({HttpMethod::PUT, "/marketing/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // Structured RBAC over the existing scope system (RequireScope
            // stays enforced server-side). No fund movement.
            DomainSpec s{"rbac", "Admin Roles & Permissions", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,  "/admin-roles",              "",   false});
            s.endpoints.push_back({HttpMethod::POST, "/admin-roles",              "{}", false});
            s.endpoints.push_back({HttpMethod::GET,  "/admin-roles/:id",          "",   false});
            s.endpoints.push_back({HttpMethod::PUT,  "/admin-roles/:id",          "{}", false});
            s.endpoints.push_back({HttpMethod::DELETE, "/admin-roles/:id",        "",   false});
            s.endpoints.push_back({HttpMethod::GET,  "/admin-permissions",        "",   false});
            s.endpoints.push_back({HttpMethod::POST, "/admin-permissions",        "{}", false});
            s.endpoints.push_back({HttpMethod::GET,  "/admins/:id/permissions",   "",   false});
            s.governance_actions.push_back({HttpMethod::POST,   "/admins/:id/role",         "{\"role_id\":\"\"}", true});
            s.governance_actions.push_back({HttpMethod::DELETE, "/admins/:id/role/:roleId", "", true});
            v.push_back(std::move(s));
        }

        // -----------------------------------------------------------------
        // 9 scoped admin domains — real main.go routes on the WL backend
        // (port 8456). Governance/config records + scoped approval actions.
        // -----------------------------------------------------------------
        {
            // LiquidityAdmin: wl-liquidity/sources CRUD + status + allocations + stats.
            // Matches the Go backend routes exactly (wl_admin_handlers.go).
            DomainSpec s{"liquidity", "Liquidity Sources", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,    "/wl-liquidity/sources",      "", false});
            s.endpoints.push_back({HttpMethod::POST,   "/wl-liquidity/sources",      "", true});
            s.endpoints.push_back({HttpMethod::PUT,    "/wl-liquidity/sources/:id",  "", true});
            s.endpoints.push_back({HttpMethod::DELETE, "/wl-liquidity/sources/:id",  "", true});
            s.endpoints.push_back({HttpMethod::GET,    "/wl-liquidity/allocations",   "", false});
            s.endpoints.push_back({HttpMethod::POST,  "/wl-liquidity/allocations",   "", true});
            s.endpoints.push_back({HttpMethod::GET,    "/wl-liquidity/stats",         "", false});
            s.governance_actions.push_back({HttpMethod::PUT,  "/wl-liquidity/sources/:id",  "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // CardAdmin: wl-cards CRUD + status + transactions + stats.
            DomainSpec s{"crypto-card", "Crypto Cards", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,    "/wl-cards",            "", false});
            s.endpoints.push_back({HttpMethod::POST,   "/wl-cards",            "", true});
            s.endpoints.push_back({HttpMethod::GET,    "/wl-cards/transactions","", false});
            s.endpoints.push_back({HttpMethod::GET,    "/wl-cards/stats",      "", false});
            s.governance_actions.push_back({HttpMethod::POST, "/wl-cards/:id/block",    "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/wl-cards/:id/activate", "", true});
            s.governance_actions.push_back({HttpMethod::PUT,  "/wl-cards/:id/limit",    "{\"limit\":\"\"}", true});
            s.governance_actions.push_back({HttpMethod::PUT,  "/wl-cards/:id/status",   "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // BotAdmin: wl-bots/operators CRUD + status + config + stats.
            DomainSpec s{"bots", "Bots", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,    "/wl-bots/operators",   "", false});
            s.endpoints.push_back({HttpMethod::POST,   "/wl-bots/operators",   "", true});
            s.endpoints.push_back({HttpMethod::GET,    "/wl-bots/config",      "", false});
            s.endpoints.push_back({HttpMethod::GET,    "/wl-bots/stats",        "", false});
            s.governance_actions.push_back({HttpMethod::PUT, "/wl-bots/operators/:id/status", "{\"status\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // KYCAdmin: list + approve + reject {reason}
            DomainSpec s{"kyc", "KYC", {}, {}};
            s.endpoints.push_back({HttpMethod::GET, "/kyc", "", false});
            s.governance_actions.push_back({HttpMethod::POST, "/kyc/:id/approve", "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/kyc/:id/reject",  "{\"reason\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // CustomerServiceAdmin: tickets list + create + status + assign
            DomainSpec s{"tickets", "Support Tickets", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,  "/tickets", "{}", false});
            s.endpoints.push_back({HttpMethod::POST, "/tickets", "{}", false});
            s.governance_actions.push_back({HttpMethod::PUT,  "/tickets/:id/status",  "{\"status\":\"\"}", true});
            s.governance_actions.push_back({HttpMethod::POST, "/tickets/:id/assign",  "{\"assignee_id\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // SecurityAdmin (WL client only): ip-whitelist add/remove
            DomainSpec s{"ip-whitelist", "IP Whitelist", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,  "/ip-whitelist",      "{}", false});
            s.endpoints.push_back({HttpMethod::POST, "/ip-whitelist",      "{\"ip\":\"\"}", false});
            s.governance_actions.push_back({HttpMethod::DELETE, "/ip-whitelist/:id", "", true});
            v.push_back(std::move(s));
        }
        {
            // ComplianceAdmin: audit-logs (paginated) + reports
            DomainSpec s{"audit-logs", "Audit Logs & Reports", {}, {}};
            s.endpoints.push_back({HttpMethod::GET, "/audit-logs", "?page=1&page_size=20", false});
            s.endpoints.push_back({HttpMethod::GET, "/reports",    "", false});
            v.push_back(std::move(s));
        }
        {
            // WalletAdmin: wallets list + create + status + approve + reject
            DomainSpec s{"wallet-management", "Wallet Management", {}, {}};
            s.endpoints.push_back({HttpMethod::GET,  "/wallets", "{}", false});
            s.endpoints.push_back({HttpMethod::POST, "/wallets", "{}", false});
            s.governance_actions.push_back({HttpMethod::PUT,  "/wallets/:id/status",  "{\"status\":\"\"}", true});
            s.governance_actions.push_back({HttpMethod::POST, "/wallets/:id/approve", "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/wallets/:id/reject",  "{\"reason\":\"\"}", true});
            v.push_back(std::move(s));
        }
        {
            // Withdrawals: two-party approval (list + approve + reject {reason})
            DomainSpec s{"withdrawals", "Withdrawals", {}, {}};
            s.endpoints.push_back({HttpMethod::GET, "/withdrawals", "", false});
            s.governance_actions.push_back({HttpMethod::POST, "/withdrawals/:id/approve", "", true});
            s.governance_actions.push_back({HttpMethod::POST, "/withdrawals/:id/reject",  "{\"reason\":\"\"}", true});
            v.push_back(std::move(s));
        }
        return v;
    }();
    return registry;
}

inline std::optional<DomainSpec> find_domain(std::string_view key) {
    const auto& r = domain_registry();
    for (const auto& s : r) {
        if (s.key == key) return s;
    }
    return std::nullopt;
}

// All endpoints (CRUD + governance) for a domain, useful for rendering the
// read-only contract in native clients.
inline std::vector<DomainRequest> all_endpoints(const DomainSpec& s) {
    std::vector<DomainRequest> out = s.endpoints;
    out.insert(out.end(), s.governance_actions.begin(), s.governance_actions.end());
    return out;
}

}  // namespace wl_admin
}  // namespace tiger

#endif  // TIGER_WL_ADMIN_DOMAINS_HPP
