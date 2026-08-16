/**
 * TigerWallet Admin - Super Admin Domain Handlers
 *
 * Governance handlers for the 12 admin domains driven by the real
 * `super_admin/go` backend on port 8082. Each handler performs real HTTP
 * calls (POSIX sockets) to `http://localhost:8082/api/v1/admin/...` with JWT
 * bearer auth. No stubs, fakes, or mocks: network and parse failures surface
 * as an honest error in the returned DomainResult.
 *
 * Governance records only; never moves crypto assets.
 *
 * Compile: g++ -std=c++20 -fsyntax-only -I super_admin/cpp/include <entry>
 */
#ifndef TIGER_ADMIN_SUPER_ADMIN_DOMAINS_HPP
#define TIGER_ADMIN_SUPER_ADMIN_DOMAINS_HPP

#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <map>
#include <memory>
#include <netdb.h>
#include <netinet/in.h>
#include <optional>
#include <sstream>
#include <string>
#include <sys/socket.h>
#include <sys/types.h>
#include <thread>
#include <unistd.h>
#include <utility>
#include <vector>

namespace tiger {
namespace admin {
namespace domains {

/// HTTP method used by the domain handlers.
enum class HttpMethod { GET, POST, PUT, DELETE };

/// A single key/value header.
using Headers = std::vector<std::pair<std::string, std::string>>;

/// Outcome of a domain HTTP call: status code + raw body + optional error.
struct DomainResult {
    int status_code{0};          ///< 0 when a transport error occurred.
    std::string body;            ///< raw upstream response body (JSON).
    std::string error;           ///< non-empty on transport/parse failure.
    bool ok() const { return error.empty() && status_code >= 200 && status_code < 300; }
    bool transport_error() const { return !error.empty(); }
};

/// Minimal blocking HTTP/1.1 client over POSIX sockets targeting the Go
/// super-admin backend (127.0.0.1:8082). Intentionally dependency-free so the
/// header compiles standalone with the project's C++20 toolchain.
class SuperAdminHttpClient {
public:
    static constexpr const char* kHost = "localhost";
    static constexpr const char* kAddr = "127.0.0.1";
    static constexpr uint16_t kPort = 8082;
    static constexpr const char* kBase = "/api/v1/admin";

    explicit SuperAdminHttpClient(std::string jwt = {})
        : jwt_(std::move(jwt)) {}

    /// Issue a request to `kBase/path` with the given method and optional
    /// JSON body. `path` must not begin with a slash.
    DomainResult request(HttpMethod method, const std::string& path,
                         const std::string& body = {}, Headers extra = {}) const {
        const std::string method_str = method_name(method);
        const std::string url_path = std::string(kBase) + "/" + path;
        std::string payload = body;
        Headers hdrs = extra;
        if (!payload.empty()) {
            set_default(hdrs, "Content-Type", "application/json");
        }
        if (!jwt_.empty()) {
            set_default(hdrs, "Authorization", "Bearer " + jwt_);
        }
        set_default(hdrs, "Host", std::string(kHost) + ":" + std::to_string(kPort));
        set_default(hdrs, "Connection", "close");
        set_default(hdrs, "Content-Length", std::to_string(payload.size()));

        std::string req = build_request(method_str, url_path, hdrs, payload);
        return send_and_parse(req);
    }

    DomainResult get(const std::string& path) const { return request(HttpMethod::GET, path); }
    DomainResult post(const std::string& path, const std::string& body) const {
        return request(HttpMethod::POST, path, body);
    }
    DomainResult put(const std::string& path, const std::string& body) const {
        return request(HttpMethod::PUT, path, body);
    }
    DomainResult del(const std::string& path) const { return request(HttpMethod::DELETE, path); }

    void set_jwt(std::string jwt) { jwt_ = std::move(jwt); }
    const std::string& jwt() const { return jwt_; }

private:
    std::string jwt_;

    static const char* method_name(HttpMethod m) {
        switch (m) {
            case HttpMethod::GET: return "GET";
            case HttpMethod::POST: return "POST";
            case HttpMethod::PUT: return "PUT";
            case HttpMethod::DELETE: return "DELETE";
        }
        return "GET";
    }

    static void set_default(Headers& hdrs, const std::string& key, const std::string& val) {
        for (const auto& [k, v] : hdrs) {
            (void)v;
            if (case_insensitive_equal(k, key)) return;
        }
        hdrs.emplace_back(key, val);
    }

    static bool case_insensitive_equal(const std::string& a, const std::string& b) {
        if (a.size() != b.size()) return false;
        return std::equal(a.begin(), a.end(), b.begin(),
                          [](char x, char y) { return std::tolower(static_cast<unsigned char>(x)) ==
                                                      std::tolower(static_cast<unsigned char>(y)); });
    }

    static std::string build_request(const std::string& method, const std::string& path,
                                     const Headers& hdrs, const std::string& body) {
        std::ostringstream out;
        out << method << " " << path << " HTTP/1.1\r\n";
        for (const auto& [k, v] : hdrs) out << k << ": " << v << "\r\n";
        out << "\r\n" << body;
        return out.str();
    }

    DomainResult send_and_parse(const std::string& request_data) const {
        DomainResult result;

        addrinfo hints{};
        hints.ai_family = AF_INET;
        hints.ai_socktype = SOCK_STREAM;
        addrinfo* res = nullptr;
        std::string port_str = std::to_string(kPort);
        int gai = getaddrinfo(kAddr, port_str.c_str(), &hints, &res);
        if (gai != 0 || res == nullptr) {
            result.error = std::string("getaddrinfo failed: ") + gai_strerror(gai);
            return result;
        }

        int fd = ::socket(res->ai_family, res->ai_socktype, res->ai_protocol);
        if (fd < 0) {
            result.error = std::string("socket failed: ") + std::strerror(errno);
            freeaddrinfo(res);
            return result;
        }

        if (::connect(fd, res->ai_addr, res->ai_addrlen) < 0) {
            result.error = std::string("connect failed: ") + std::strerror(errno);
            ::close(fd);
            freeaddrinfo(res);
            return result;
        }
        freeaddrinfo(res);

        ssize_t total = 0;
        while (total < static_cast<ssize_t>(request_data.size())) {
            ssize_t n = ::send(fd, request_data.data() + total, request_data.size() - total, 0);
            if (n <= 0) {
                result.error = std::string("send failed: ") + std::strerror(errno);
                ::close(fd);
                return result;
            }
            total += n;
        }

        std::string raw;
        char buf[8192];
        while (true) {
            ssize_t n = ::recv(fd, buf, sizeof(buf), 0);
            if (n > 0) {
                raw.append(buf, static_cast<size_t>(n));
            } else if (n == 0) {
                break;
            } else {
                if (errno == EINTR) continue;
                break;
            }
        }
        ::close(fd);

        // Split headers/body at the first blank line.
        const std::string sep = "\r\n\r\n";
        size_t header_end = raw.find(sep);
        if (header_end == std::string::npos) {
            result.error = "malformed HTTP response (no header terminator)";
            result.body = std::move(raw);
            return result;
        }
        std::string header_block = raw.substr(0, header_end);
        std::string body = raw.substr(header_end + sep.size());

        // Parse the status line: "HTTP/1.1 200 OK".
        size_t first_sp = header_block.find(' ');
        if (first_sp == std::string::npos) {
            result.error = "malformed HTTP status line";
            result.body = std::move(body);
            return result;
        }
        size_t second_sp = header_block.find(' ', first_sp + 1);
        std::string code_str = header_block.substr(
            first_sp + 1, (second_sp == std::string::npos ? std::string::npos : second_sp - first_sp - 1));
        try {
            result.status_code = std::stoi(code_str);
        } catch (...) {
            result.error = "non-numeric HTTP status code";
        }
        result.body = std::move(body);
        return result;
    }
};

/// Tiny JSON string builder for the create payloads used by domain handlers.
/// Not a general-purpose JSON library: only the primitive values we need.
class JsonValue {
public:
    JsonValue& add(const std::string& key, const std::string& val) {
        fields_.emplace_back(key, quote_json_string(val));
        return *this;
    }
    JsonValue& add(const std::string& key, const char* val) {
        fields_.emplace_back(key, quote_json_string(std::string(val)));
        return *this;
    }
    JsonValue& add(const std::string& key, double val) {
        fields_.emplace_back(key, number_json(val));
        return *this;
    }
    JsonValue& add(const std::string& key, int64_t val) {
        fields_.emplace_back(key, std::to_string(val));
        return *this;
    }
    JsonValue& add_strings(const std::string& key, const std::vector<std::string>& vals) {
        std::ostringstream out;
        out << "[";
        for (size_t i = 0; i < vals.size(); ++i) {
            if (i) out << ",";
            out << quote_json_string(vals[i]);
        }
        out << "]";
        fields_.emplace_back(key, out.str());
        return *this;
    }
    std::string build() const {
        std::ostringstream out;
        out << "{";
        for (size_t i = 0; i < fields_.size(); ++i) {
            if (i) out << ",";
            out << quote_json_string(fields_[i].first) << ":" << fields_[i].second;
        }
        out << "}";
        return out.str();
    }

private:
    std::vector<std::pair<std::string, std::string>> fields_;

    static std::string quote_json_string(const std::string& s) {
        std::string out;
        out.reserve(s.size() + 2);
        out.push_back('"');
        for (char c : s) {
            switch (c) {
                case '"': out += "\\\""; break;
                case '\\': out += "\\\\"; break;
                case '\b': out += "\\b"; break;
                case '\f': out += "\\f"; break;
                case '\n': out += "\\n"; break;
                case '\r': out += "\\r"; break;
                case '\t': out += "\\t"; break;
                default:
                    if (static_cast<unsigned char>(c) < 0x20) {
                        char buf[8];
                        std::snprintf(buf, sizeof(buf), "\\u%04x", c);
                        out += buf;
                    } else {
                        out.push_back(c);
                    }
            }
        }
        out.push_back('"');
        return out;
    }
    static std::string number_json(double v) {
        std::ostringstream out;
        if (v == static_cast<int64_t>(v)) out << static_cast<int64_t>(v);
        else out << v;
        return out.str();
    }
};

/// Generic CRUD + governance actions for one admin domain. The 12 concrete
/// domain handlers below configure an instance of this with their resource
/// path and supported actions.
class DomainHandler {
public:
    enum class Action { STATUS, APPROVE, REJECT };

    DomainHandler(std::string resource, std::vector<Action> actions, SuperAdminHttpClient client)
        : resource_(std::move(resource)), actions_(std::move(actions)), client_(std::move(client)) {}

    // CRUD
    DomainResult list() const { return client_.get(resource_); }
    DomainResult create(const std::string& json_body) const { return client_.post(resource_, json_body); }
    DomainResult get_one(const std::string& id) const { return client_.get(resource_ + "/" + id); }
    DomainResult update(const std::string& id, const std::string& json_body) const {
        return client_.put(resource_ + "/" + id, json_body);
    }
    DomainResult remove(const std::string& id) const { return client_.del(resource_ + "/" + id); }

    // Governance sub-actions
    DomainResult set_status(const std::string& id, const std::string& status) const {
        if (!supports(Action::STATUS)) return unsupported("status");
        JsonValue body;
        body.add("status", status);
        return client_.put(resource_ + "/" + id + "/status", body.build());
    }
    DomainResult approve(const std::string& id) const {
        if (!supports(Action::APPROVE)) return unsupported("approve");
        return client_.post(resource_ + "/" + id + "/approve", "{}");
    }
    DomainResult reject(const std::string& id, const std::string& reason) const {
        if (!supports(Action::REJECT)) return unsupported("reject");
        JsonValue body;
        body.add("reason", reason);
        return client_.post(resource_ + "/" + id + "/reject", body.build());
    }

    const std::string& resource() const { return resource_; }
    bool supports(Action a) const {
        return std::find(actions_.begin(), actions_.end(), a) != actions_.end();
    }

private:
    std::string resource_;
    std::vector<Action> actions_;
    SuperAdminHttpClient client_;

    static DomainResult unsupported(const std::string& action) {
        DomainResult r;
        r.error = "action '" + action + "' not supported by this domain";
        return r;
    }
};

// ---------------------------------------------------------------------------
// 12 domain handlers (real HTTP to :8082)
// ---------------------------------------------------------------------------

/// 1. Futures positions: CRUD + status.
struct FuturesHandler {
    DomainHandler handler;
    explicit FuturesHandler(SuperAdminHttpClient c)
        : handler("futures", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& pair, const std::string& side, double size,
                                   double leverage, double entry_price, double liquidation_price,
                                   double margin, int64_t chain_id) {
        JsonValue v;
        v.add("pair", pair).add("side", side).add("size", size).add("leverage", leverage)
            .add("entry_price", entry_price).add("liquidation_price", liquidation_price)
            .add("margin", margin).add("chain_id", chain_id);
        return v.build();
    }
};

/// 2. Options contracts: CRUD + status.
struct OptionsHandler {
    DomainHandler handler;
    explicit OptionsHandler(SuperAdminHttpClient c)
        : handler("options", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& underlying, const std::string& option_type,
                                   double strike, const std::string& expiry, double premium,
                                   double size, int64_t chain_id) {
        JsonValue v;
        v.add("underlying", underlying).add("option_type", option_type).add("strike", strike)
            .add("expiry", expiry).add("premium", premium).add("size", size).add("chain_id", chain_id);
        return v.build();
    }
};

/// 3. Copy-trading configs: CRUD + status.
struct CopyTradingHandler {
    DomainHandler handler;
    explicit CopyTradingHandler(SuperAdminHttpClient c)
        : handler("copy-trading", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& follower_id, const std::string& leader_id,
                                   double allocation, double max_leverage) {
        JsonValue v;
        v.add("follower_id", follower_id).add("leader_id", leader_id)
            .add("allocation", allocation).add("max_leverage", max_leverage);
        return v.build();
    }
};

/// 4. Convert orders: CRUD + status.
struct ConvertHandler {
    DomainHandler handler;
    explicit ConvertHandler(SuperAdminHttpClient c)
        : handler("convert", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& user_id, const std::string& from_token,
                                   const std::string& to_token, double from_amount, double to_amount,
                                   double rate, int64_t chain_id) {
        JsonValue v;
        v.add("user_id", user_id).add("from_token", from_token).add("to_token", to_token)
            .add("from_amount", from_amount).add("to_amount", to_amount)
            .add("rate", rate).add("chain_id", chain_id);
        return v.build();
    }
};

/// 5. Onramp orders: CRUD + approve + reject(reason).
struct OnrampHandler {
    DomainHandler handler;
    explicit OnrampHandler(SuperAdminHttpClient c)
        : handler("onramp",
                  {DomainHandler::Action::APPROVE, DomainHandler::Action::REJECT},
                  std::move(c)) {}
    static std::string create_body(const std::string& user_id, const std::string& provider,
                                   const std::string& fiat_currency, const std::string& crypto_token,
                                   double fiat_amount, double crypto_amount) {
        JsonValue v;
        v.add("user_id", user_id).add("provider", provider).add("fiat_currency", fiat_currency)
            .add("crypto_token", crypto_token).add("fiat_amount", fiat_amount)
            .add("crypto_amount", crypto_amount);
        return v.build();
    }
};

/// 6. Offramp orders: CRUD + approve + reject(reason).
struct OfframpHandler {
    DomainHandler handler;
    explicit OfframpHandler(SuperAdminHttpClient c)
        : handler("offramp",
                  {DomainHandler::Action::APPROVE, DomainHandler::Action::REJECT},
                  std::move(c)) {}
    static std::string create_body(const std::string& user_id, const std::string& provider,
                                   const std::string& crypto_token, const std::string& fiat_currency,
                                   double crypto_amount, double fiat_amount) {
        JsonValue v;
        v.add("user_id", user_id).add("provider", provider).add("crypto_token", crypto_token)
            .add("fiat_currency", fiat_currency).add("crypto_amount", crypto_amount)
            .add("fiat_amount", fiat_amount);
        return v.build();
    }
};

/// 7. P2P clients: CRUD + status.
struct P2PClientsHandler {
    DomainHandler handler;
    explicit P2PClientsHandler(SuperAdminHttpClient c)
        : handler("p2p-clients", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& user_id, const std::string& username) {
        JsonValue v;
        v.add("user_id", user_id).add("username", username);
        return v.build();
    }
};

/// 8. Partners: CRUD + status + approve + reject.
struct PartnersHandler {
    DomainHandler handler;
    explicit PartnersHandler(SuperAdminHttpClient c)
        : handler("partners",
                  {DomainHandler::Action::STATUS, DomainHandler::Action::APPROVE,
                   DomainHandler::Action::REJECT},
                  std::move(c)) {}
    static std::string create_body(const std::string& name, const std::string& contact_email,
                                   double revenue_share) {
        JsonValue v;
        v.add("name", name).add("contact_email", contact_email).add("revenue_share", revenue_share);
        return v.build();
    }
};

/// 9. Reward campaigns: CRUD + status.
struct RewardsHandler {
    DomainHandler handler;
    explicit RewardsHandler(SuperAdminHttpClient c)
        : handler("rewards", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& name, const std::string& reward_type,
                                   double amount, const std::string& token, const std::string& start_at,
                                   const std::string& end_at) {
        JsonValue v;
        v.add("name", name).add("reward_type", reward_type).add("amount", amount)
            .add("token", token).add("start_at", start_at).add("end_at", end_at);
        return v.build();
    }
};

/// 10. Marketing campaigns: CRUD + status.
struct MarketingHandler {
    DomainHandler handler;
    explicit MarketingHandler(SuperAdminHttpClient c)
        : handler("marketing", {DomainHandler::Action::STATUS}, std::move(c)) {}
    static std::string create_body(const std::string& name, const std::string& channel,
                                   double budget, const std::string& start_at,
                                   const std::string& end_at) {
        JsonValue v;
        v.add("name", name).add("channel", channel).add("budget", budget)
            .add("start_at", start_at).add("end_at", end_at);
        return v.build();
    }
};

/// 11. Admin RBAC: roles + permissions CRUD, role assignment, effective perms.
struct AdminRolesHandler {
    SuperAdminHttpClient client;
    explicit AdminRolesHandler(SuperAdminHttpClient c) : client(std::move(c)) {}

    // Roles CRUD
    DomainResult list_roles() const { return client.get("admin-roles"); }
    DomainResult create_role(const std::string& name, const std::string& description,
                             const std::vector<std::string>& permissions) const {
        JsonValue v;
        v.add("name", name).add("description", description).add_strings("permissions", permissions);
        return client.post("admin-roles", v.build());
    }
    DomainResult update_role(const std::string& id, const std::string& body) const {
        return client.put("admin-roles/" + id, body);
    }
    DomainResult delete_role(const std::string& id) const { return client.del("admin-roles/" + id); }

    // Permissions CRUD
    DomainResult list_permissions() const { return client.get("admin-permissions"); }
    DomainResult create_permission(const std::string& name, const std::string& description,
                                   const std::string& category) const {
        JsonValue v;
        v.add("name", name).add("description", description).add("category", category);
        return client.post("admin-permissions", v.build());
    }
    DomainResult delete_permission(const std::string& id) const {
        return client.del("admin-permissions/" + id);
    }

    // Assign role to admin + effective permissions.
    DomainResult assign_role(const std::string& admin_id, const std::string& role_id) const {
        JsonValue v;
        v.add("role_id", role_id);
        return client.post("admins/" + admin_id + "/roles", v.build());
    }
    DomainResult revoke_role(const std::string& admin_id, const std::string& role_id) const {
        return client.del("admins/" + admin_id + "/roles/" + role_id);
    }
    DomainResult effective_permissions(const std::string& admin_id) const {
        return client.get("admins/" + admin_id + "/permissions");
    }
};

/// 12. White-label control: five sub-resources, each CRUD + status.
struct WLControlHandler {
    SuperAdminHttpClient client;
    explicit WLControlHandler(SuperAdminHttpClient c) : client(std::move(c)) {}

    DomainHandler clients() const {
        return DomainHandler("wl-clients", {DomainHandler::Action::STATUS}, client);
    }
    DomainHandler master_wallets() const {
        return DomainHandler("wl-master-wallets", {DomainHandler::Action::STATUS}, client);
    }
    DomainHandler user_wallets() const {
        return DomainHandler("wl-user-wallets", {DomainHandler::Action::STATUS}, client);
    }
    DomainHandler bots() const {
        return DomainHandler("wl-bots", {DomainHandler::Action::STATUS}, client);
    }
    DomainHandler bots_clients() const {
        return DomainHandler("wl-bots-clients", {DomainHandler::Action::STATUS}, client);
    }

    // Convenience create bodies for each white-label resource.
    static std::string create_wl_client(const std::string& name, const std::string& domain) {
        JsonValue v;
        v.add("name", name).add("domain", domain);
        return v.build();
    }
    static std::string create_master_wallet(const std::string& name, const std::string& address,
                                            int64_t chain_id) {
        JsonValue v;
        v.add("name", name).add("address", address).add("chain_id", chain_id);
        return v.build();
    }
    static std::string create_user_wallet(const std::string& name, const std::string& master_wallet_id,
                                          const std::string& address, int64_t chain_id) {
        JsonValue v;
        v.add("name", name).add("master_wallet_id", master_wallet_id)
            .add("address", address).add("chain_id", chain_id);
        return v.build();
    }
    static std::string create_wl_bot(const std::string& name, const std::string& bot_type) {
        JsonValue v;
        v.add("name", name).add("bot_type", bot_type);
        return v.build();
    }
    static std::string create_bots_client(const std::string& name, const std::string& company,
                                          const std::string& email, const std::string& permission_level) {
        JsonValue v;
        v.add("name", name).add("company", company).add("email", email)
            .add("permission_level", permission_level);
        return v.build();
    }
};

/// Aggregate of all 12 domain handlers, wired to a single HTTP client.
struct SuperAdminDomains {
    SuperAdminHttpClient client;
    FuturesHandler futures;
    OptionsHandler options;
    CopyTradingHandler copy_trading;
    ConvertHandler convert;
    OnrampHandler onramp;
    OfframpHandler offramp;
    P2PClientsHandler p2p_clients;
    PartnersHandler partners;
    RewardsHandler rewards;
    MarketingHandler marketing;
    AdminRolesHandler admin_roles;
    WLControlHandler wl_control;

    explicit SuperAdminDomains(std::string jwt = {})
        : client(std::move(jwt)),
          futures(client),
          options(client),
          copy_trading(client),
          convert(client),
          onramp(client),
          offramp(client),
          p2p_clients(client),
          partners(client),
          rewards(client),
          marketing(client),
          admin_roles(client),
          wl_control(client) {}

    /// Names of the 12 governance domains (for UI wiring / logging).
    static std::vector<std::string> domain_names() {
        return {"futures", "options", "copy-trading", "convert",   "onramp",
                "offramp", "p2p-clients", "partners",     "rewards",   "marketing",
                "admin-roles", "wl-control"};
    }
};

inline std::unique_ptr<SuperAdminDomains> create_super_admin_domains(std::string jwt = {}) {
    return std::make_unique<SuperAdminDomains>(std::move(jwt));
}

}  // namespace domains
}  // namespace admin
}  // namespace tiger

#endif  // TIGER_ADMIN_SUPER_ADMIN_DOMAINS_HPP
