// TigerBots C++ API client implementation (libcurl).
//
// See bot_api.hpp for the contract. All methods issue real HTTP calls via a
// libcurl easy handle. Fail-closed: any non-2xx response throws
// `BotApiException` carrying the backend's `error`/`message` text.

#include "bot_api.hpp"

#include <curl/curl.h>

#include <algorithm>
#include <cctype>
#include <cstddef>
#include <iomanip>
#include <sstream>

namespace tigerbots {

namespace {

class CurlGlobal {
public:
    CurlGlobal() { curl_global_init(CURL_GLOBAL_DEFAULT); }
    ~CurlGlobal() { curl_global_cleanup(); }
};

// RAII guard for libcurl's process-global init/cleanup. Created once on first
// use so the client can be compiled into libraries that don't control main().
CurlGlobal& curlGlobal() {
    static CurlGlobal g;
    return g;
}

struct WriteCtx {
    std::string buf;
};

size_t writeCb(char* ptr, size_t size, size_t nmemb, void* userdata) {
    auto* ctx = static_cast<WriteCtx*>(userdata);
    const size_t total = size * nmemb;
    ctx->buf.append(ptr, total);
    return total;
}

size_t headerCb(char* /*ptr*/, size_t size, size_t nmemb, void* /*userdata*/) {
    return size * nmemb;
}

} // namespace

BotApiClient::BotApiClient(std::string baseUrl)
    : base_url_(std::move(baseUrl)) {
    curlGlobal();
}

BotApiClient::~BotApiClient() = default;

std::string BotApiClient::encodePathSegment(const std::string& s) {
    std::ostringstream out;
    out << std::hex << std::uppercase << std::setfill('0');
    for (unsigned char c : s) {
        if (std::isalnum(c) || c == '-' || c == '_' || c == '.' || c == '~') {
            out << static_cast<char>(c);
        } else {
            out << '%' << std::setw(2) << static_cast<int>(c);
        }
    }
    return out.str();
}

nlohmann::json BotApiClient::request(const std::string& path,
                                     const std::string& method,
                                     const std::optional<nlohmann::json>& body,
                                     bool authenticated) {
    CURL* easy = curl_easy_init();
    if (!easy) {
        throw BotApiException(0, "curl_easy_init failed");
    }
    WriteCtx ctx;
    struct curl_slist* headers = nullptr;
    std::string bodyStr;
    try {
        headers = curl_slist_append(headers, "Accept: application/json");
        if (body.has_value()) {
            bodyStr = body->dump();
            headers = curl_slist_append(headers, "Content-Type: application/json; charset=utf-8");
        }
        if (authenticated) {
            if (!token_.has_value()) {
                curl_slist_free_all(headers);
                curl_easy_cleanup(easy);
                throw BotApiException(401, "not authenticated: no JWT token set");
            }
            const std::string auth = "Authorization: Bearer " + *token_;
            headers = curl_slist_append(headers, auth.c_str());
        }
        const std::string url = base_url_ + path;
        curl_easy_setopt(easy, CURLOPT_URL, url.c_str());
        curl_easy_setopt(easy, CURLOPT_CUSTOMREQUEST, method.c_str());
        curl_easy_setopt(easy, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(easy, CURLOPT_TIMEOUT, 30L);
        curl_easy_setopt(easy, CURLOPT_CONNECTTIMEOUT, 15L);
        curl_easy_setopt(easy, CURLOPT_FOLLOWLOCATION, 0L);
        curl_easy_setopt(easy, CURLOPT_WRITEFUNCTION, writeCb);
        curl_easy_setopt(easy, CURLOPT_WRITEDATA, &ctx);
        curl_easy_setopt(easy, CURLOPT_HEADERFUNCTION, headerCb);
        curl_easy_setopt(easy, CURLOPT_NOSIGNAL, 1L);
        if (body.has_value()) {
            curl_easy_setopt(easy, CURLOPT_POSTFIELDS, bodyStr.c_str());
            curl_easy_setopt(easy, CURLOPT_POSTFIELDSIZE,
                             static_cast<long>(bodyStr.size()));
        }
        CURLcode rc = curl_easy_perform(easy);
        if (rc != CURLE_OK) {
            curl_slist_free_all(headers);
            curl_easy_cleanup(easy);
            throw BotApiException(0, std::string("curl: ") + curl_easy_strerror(rc));
        }
        long httpCode = 0;
        curl_easy_getinfo(easy, CURLINFO_RESPONSE_CODE, &httpCode);
        curl_slist_free_all(headers);
        curl_easy_cleanup(easy);
        if (httpCode < 200 || httpCode > 299) {
            std::string msg;
            try {
                auto j = nlohmann::json::parse(ctx.buf.empty() ? "{}" : ctx.buf);
                if (j.contains("error") && j["error"].is_string()) {
                    msg = j["error"].get<std::string>();
                } else if (j.contains("message") && j["message"].is_string()) {
                    msg = j["message"].get<std::string>();
                }
            } catch (...) {
                msg = ctx.buf;
            }
            if (msg.empty()) msg = "HTTP " + std::to_string(httpCode);
            throw BotApiException(static_cast<int>(httpCode), msg);
        }
        if (ctx.buf.empty()) return nlohmann::json{};
        return nlohmann::json::parse(ctx.buf);
    } catch (...) {
        if (headers) curl_slist_free_all(headers);
        curl_easy_cleanup(easy);
        throw;
    }
}

nlohmann::json BotApiClient::get(const std::string& path, bool authenticated) {
    return request(path, "GET", std::nullopt, authenticated);
}

nlohmann::json BotApiClient::del(const std::string& path) {
    return request(path, "DELETE", std::nullopt, true);
}

nlohmann::json BotApiClient::post(const std::string& path, const nlohmann::json& body) {
    return request(path, "POST", body, true);
}

nlohmann::json BotApiClient::put(const std::string& path, const nlohmann::json& body) {
    return request(path, "PUT", body, true);
}

nlohmann::json BotApiClient::postNoBody(const std::string& path) {
    return request(path, "POST", std::nullopt, true);
}

// -- Auth --

BotAuthResponse BotApiClient::registerUser(const std::string& username,
                                            const std::string& password,
                                            const std::optional<std::string>& email,
                                            const std::optional<std::string>& walletAddress,
                                            const std::optional<std::string>& role) {
    nlohmann::json body = {
        {"username", username},
        {"password", password},
    };
    if (email) body["email"] = *email;
    if (walletAddress) body["wallet_address"] = *walletAddress;
    if (role) body["role"] = *role;
    auto res = request("/auth/register", "POST", body, false);
    BotAuthResponse out{};
    if (res.contains("token") && res["token"].is_string()) {
        out.token = res["token"].get<std::string>();
        token_ = out.token;
    }
    out.userId = res.value("user_id", "");
    out.role = res.value("role", "");
    return out;
}

BotAuthResponse BotApiClient::login(const std::string& username, const std::string& password) {
    nlohmann::json body = {{"username", username}, {"password", password}};
    auto res = request("/auth/login", "POST", body, false);
    BotAuthResponse out{};
    if (res.contains("token") && res["token"].is_string()) {
        out.token = res["token"].get<std::string>();
        token_ = out.token;
    }
    out.userId = res.value("user_id", "");
    out.role = res.value("role", "");
    return out;
}

void BotApiClient::logout() {
    try {
        postNoBody("/auth/logout");
    } catch (...) {
        token_.reset();
        throw;
    }
    token_.reset();
}

// -- Health + public tiers --

nlohmann::json BotApiClient::health() { return get("/health", false); }
nlohmann::json BotApiClient::publicTiers() { return get("/public/tiers", false); }

// -- Bots CRUD + lifecycle --

nlohmann::json BotApiClient::listBots() { return get("/bots"); }
nlohmann::json BotApiClient::getBot(const std::string& id) {
    return get("/bots/" + encodePathSegment(id));
}
nlohmann::json BotApiClient::createBot(const std::string& name,
                                       const std::string& botType,
                                       const nlohmann::json& config,
                                       const std::optional<std::string>& exchange,
                                       const std::optional<std::string>& pair) {
    nlohmann::json body = {{"name", name}, {"bot_type", botType}, {"config", config}};
    if (exchange) body["exchange"] = *exchange;
    if (pair) body["pair"] = *pair;
    return post("/bots", body);
}
nlohmann::json BotApiClient::deleteBot(const std::string& id) {
    return del("/bots/" + encodePathSegment(id));
}
nlohmann::json BotApiClient::startBot(const std::string& id) {
    return postNoBody("/bots/" + encodePathSegment(id) + "/start");
}
nlohmann::json BotApiClient::stopBot(const std::string& id) {
    return postNoBody("/bots/" + encodePathSegment(id) + "/stop");
}
nlohmann::json BotApiClient::pauseBot(const std::string& id) {
    return postNoBody("/bots/" + encodePathSegment(id) + "/pause");
}
nlohmann::json BotApiClient::listBotExecutions(const std::string& id) {
    return get("/bots/" + encodePathSegment(id) + "/executions");
}
nlohmann::json BotApiClient::listBotLogs(const std::string& id) {
    return get("/bots/" + encodePathSegment(id) + "/logs");
}
nlohmann::json BotApiClient::listBotInstances() { return get("/bots/instances"); }
nlohmann::json BotApiClient::currentBotUser() { return get("/bots/me"); }

// -- Bot users --

nlohmann::json BotApiClient::listBotUsers() { return get("/bots/users"); }
nlohmann::json BotApiClient::createBotUser(const std::string& username,
                                           const std::string& password,
                                           const std::optional<std::string>& email,
                                           const std::optional<std::string>& walletAddress,
                                           const std::optional<std::string>& role) {
    nlohmann::json body = {{"username", username}, {"password", password}};
    if (email) body["email"] = *email;
    if (walletAddress) body["wallet_address"] = *walletAddress;
    if (role) body["role"] = *role;
    return post("/bots/users", body);
}
nlohmann::json BotApiClient::deleteBotUser(const std::string& id) {
    return del("/bots/users/" + encodePathSegment(id));
}
nlohmann::json BotApiClient::listBotTransactions() { return get("/bots/transactions"); }

// -- Subscriptions --

nlohmann::json BotApiClient::getSubscription() { return get("/subscription"); }
nlohmann::json BotApiClient::createSubscription(const std::string& tier,
                                                const std::optional<std::string>& expiresIn) {
    nlohmann::json body = {{"tier", tier}};
    if (expiresIn) body["expires_in"] = *expiresIn;
    return post("/subscription", body);
}

// -- Fees --

nlohmann::json BotApiClient::getFeeConfigs() { return get("/fees"); }
nlohmann::json BotApiClient::updateFeeConfig(const std::string& id,
                                             const std::optional<std::string>& name,
                                             const std::optional<std::string>& percentage,
                                             const std::optional<bool>& enabled) {
    nlohmann::json body = {{"id", id}};
    if (name) body["name"] = *name;
    if (percentage) body["percentage"] = *percentage;
    if (enabled) body["enabled"] = *enabled;
    return put("/fees", body);
}

// -- CEX connectors --

nlohmann::json BotApiClient::listCEX() { return get("/cex"); }
nlohmann::json BotApiClient::addCEX(const std::string& name, const nlohmann::json& config) {
    return post("/cex", {{"name", name}, {"config", config}});
}
nlohmann::json BotApiClient::removeCEX(const std::string& id) {
    return del("/cex/" + encodePathSegment(id));
}

// -- DEX connectors --

nlohmann::json BotApiClient::listDEX() { return get("/dex"); }
nlohmann::json BotApiClient::addDEX(const std::string& name, const nlohmann::json& config) {
    return post("/dex", {{"name", name}, {"config", config}});
}
nlohmann::json BotApiClient::removeDEX(const std::string& id) {
    return del("/dex/" + encodePathSegment(id));
}

// -- API keys --

nlohmann::json BotApiClient::listAPIKeys() { return get("/keys"); }
nlohmann::json BotApiClient::createAPIKey(const std::string& exchange, const std::string& apiKey) {
    return post("/keys", {{"exchange", exchange}, {"api_key", apiKey}});
}
nlohmann::json BotApiClient::deleteAPIKey(const std::string& id) {
    return del("/keys/" + encodePathSegment(id));
}

// -- Admin --

nlohmann::json BotApiClient::adminListUsers() { return get("/admin/users"); }
nlohmann::json BotApiClient::adminUserStatus(const std::string& id, bool active) {
    nlohmann::json body = {{"id", id}, {"is_active", active}};
    return put("/admin/users/" + encodePathSegment(id) + "/status", body);
}
nlohmann::json BotApiClient::adminStats() { return get("/admin/stats"); }
nlohmann::json BotApiClient::adminGetFeeAddresses() { return get("/admin/fee-addresses"); }
nlohmann::json BotApiClient::adminSetFeeAddress(const std::string& label,
                                                long long chainId,
                                                const std::string& address) {
    return post("/admin/fee-addresses",
                {{"label", label}, {"chain_id", chainId}, {"address", address}});
}
nlohmann::json BotApiClient::adminDeleteFeeAddress(const std::string& id) {
    return del("/admin/fee-addresses/" + encodePathSegment(id));
}
nlohmann::json BotApiClient::adminBotStatus(const std::string& id, const std::string& status) {
    return post("/admin/bots/" + encodePathSegment(id) + "/status",
                {{"id", id}, {"status", status}});
}

} // namespace tigerbots
