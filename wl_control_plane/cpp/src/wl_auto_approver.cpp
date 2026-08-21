// WlAutoApprover implementation + C ABI. The hot-path classify() is in the
// header (inline, wait-free atomic read + shared_mutex read lock). This TU
// holds the C ABI wrappers + the JSON parsers for the rule/treasury-address
// snapshots (called only on heartbeat, NOT on the hot path).
#include "wl_auto_approver.hpp"

#include <cstdlib>
#include <cstring>
#include <cstdio>
#include <string>
#include <string_view>
#include <vector>

namespace {

// Minimal dependency-free JSON string-array parser for the treasury-address
// snapshot. Accepts: ["0xabc...","0xdef...", ...]
std::vector<std::string> parse_string_array(const char* json) {
    std::vector<std::string> out;
    if (!json) return out;
    const char* p = json;
    while (*p) {
        const char* s = strchr(p, '"');
        if (!s) break;
        const char* e = strchr(s + 1, '"');
        if (!e) break;
        out.emplace_back(s + 1, e);
        p = e + 1;
    }
    return out;
}

// Extract the string value following a key like "rule_id" from a JSON object
// body. Returns "" if not found.
std::string extract_str(const std::string& body, const char* key) {
    const char* k = strstr(body.c_str(), key);
    if (!k) return "";
    const char* c = strchr(k + strlen(key), ':');
    if (!c) return "";
    const char* s = strchr(c, '"');
    if (!s) return "";
    const char* e = strchr(s + 1, '"');
    if (!e) return "";
    return std::string(s + 1, e);
}

} // namespace

extern "C" {

int wl_auto_approve_classify(const char* tx_type,
                             const char* to,
                             const char* token,
                             const char* amount,
                             const char** out_reason,
                             const char** out_rule_id) {
    using namespace tigerwallet::wl;
    std::string_view t  = tx_type  ? std::string_view(tx_type)  : std::string_view();
    std::string_view r  = to      ? std::string_view(to)      : std::string_view();
    std::string_view tk = token   ? std::string_view(token)   : std::string_view();
    std::string_view am = amount  ? std::string_view(amount)  : std::string_view();

    auto d = WlAutoApprover::instance().classify(t, r, tk, am);

    if (out_reason) {
        static thread_local std::string rbuf;
        rbuf = d.reason;
        *out_reason = rbuf.c_str();
    }
    if (out_rule_id) {
        static thread_local std::string ribuf;
        ribuf = d.rule_id;
        *out_rule_id = ribuf.c_str();
    }

    if (d.mode == ApprovalMode::Manual) return 1; // manual two-party required
    return d.approved ? 0 : 2; // 0 = auto-approved, 2 = denied
}

void wl_auto_approver_set_alive(int alive, const char* reason) {
    tigerwallet::wl::WlAutoApprover::instance().set_alive(alive != 0, reason);
}

void wl_auto_approver_add_treasury_address(const char* addr) {
    if (!addr) return;
    tigerwallet::wl::WlAutoApprover::instance().add_treasury_address(std::string(addr));
}

void wl_auto_approver_set_treasury_addresses_json(const char* json_array) {
    auto addrs = parse_string_array(json_array);
    std::unordered_set<std::string> s(addrs.begin(), addrs.end());
    tigerwallet::wl::WlAutoApprover::instance().set_treasury_addresses(s);
}

// Minimal dependency-free JSON object-array parser for the rule snapshot.
// Accepts: [{"rule_id":"...","product":"...","fetcher":"...","tx_type":"...",
// "token":"...","max_amount":"...","block":true}, ...]
void wl_auto_approver_set_rules_json(const char* json_array) {
    using namespace tigerwallet::wl;
    std::vector<AutoSignRule> rules;
    if (!json_array) {
        WlAutoApprover::instance().set_rules(rules);
        return;
    }
    const char* p = json_array;
    while (*p) {
        const char* obj = strchr(p, '{');
        if (!obj) break;
        const char* obj_end = strchr(obj, '}');
        if (!obj_end) break;
        std::string body(obj, obj_end - obj + 1);

        AutoSignRule r;
        r.rule_id   = extract_str(body, "\"rule_id\"");
        r.product   = extract_str(body, "\"product\"");
        r.fetcher   = extract_str(body, "\"fetcher\"");
        r.tx_type   = extract_str(body, "\"tx_type\"");
        r.token     = extract_str(body, "\"token\"");
        r.max_amount= extract_str(body, "\"max_amount\"");
        const char* bk = strstr(body.c_str(), "\"block\"");
        if (bk) {
            const char* c = strchr(bk + 7, ':');
            if (c) {
                while (*c == ':' || *c == ' ' || *c == '\t') c++;
                r.block = (strncmp(c, "true", 4) == 0);
            }
        } else {
            r.block = true;
        }
        if (!r.rule_id.empty()) rules.push_back(std::move(r));
        p = obj_end + 1;
    }
    WlAutoApprover::instance().set_rules(rules);
}

} // extern "C"
