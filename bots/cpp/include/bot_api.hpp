// TigerBots C++ API client (libcurl).
//
// Targets the standalone Bots backend (mm_bot_platform/bot_api, port 8471,
// path prefix /api/v1). JWT bearer auth; the token is held in-process
// (std::string) — C++ clients are server/desktop and have no keychain
// dependency by default; callers that need persistence can wire `setToken`/
// `token` to a secure storage of their choice.
//
// Every method issues a real libcurl easy-handle call against the backend —
// no stubs, fakes, or mock data. On any non-2xx response the method throws
// `BotApiException` (fail-closed); it never returns fabricated data.
//
// Method set mirrors bots/web/src/services/api.ts (auth, bots CRUD + start/
// stop/pause, executions, logs, users, transactions, subscriptions, fees,
// cex/dex connectors, api-keys, admin endpoints, public tiers, health).

#pragma once

#include <map>
#include <nlohmann/json.hpp>
#include <optional>
#include <string>
#include <vector>

namespace tigerbots {

class BotApiException : public std::runtime_error {
public:
    BotApiException(int status, const std::string& message)
        : std::runtime_error("Bots API " + std::to_string(status) + ": " + message),
          status_(status) {}
    int status() const noexcept { return status_; }

private:
    int status_;
};

struct BotAuthResponse {
    std::optional<std::string> token;
    std::string userId;
    std::string role;
};

class BotApiClient {
public:
    explicit BotApiClient(std::string baseUrl = kDefaultBaseUrl);
    ~BotApiClient();

    BotApiClient(const BotApiClient&) = delete;
    BotApiClient& operator=(const BotApiClient&) = delete;

    void setToken(std::optional<std::string> token) { token_ = std::move(token); }
    [[nodiscard]] std::optional<std::string> token() const { return token_; }
    void clearToken() { token_.reset(); }
    [[nodiscard]] const std::string& baseUrl() const { return base_url_; }

    // -- Auth --
    BotAuthResponse registerUser(const std::string& username,
                                 const std::string& password,
                                 const std::optional<std::string>& email = std::nullopt,
                                 const std::optional<std::string>& walletAddress = std::nullopt,
                                 const std::optional<std::string>& role = std::nullopt);
    BotAuthResponse login(const std::string& username, const std::string& password);
    void logout();

    // -- Health + public tiers --
    nlohmann::json health();
    nlohmann::json publicTiers();

    // -- Bots CRUD + lifecycle --
    nlohmann::json listBots();
    nlohmann::json getBot(const std::string& id);
    nlohmann::json createBot(const std::string& name,
                             const std::string& botType,
                             const nlohmann::json& config,
                             const std::optional<std::string>& exchange = std::nullopt,
                             const std::optional<std::string>& pair = std::nullopt);
    nlohmann::json deleteBot(const std::string& id);
    nlohmann::json startBot(const std::string& id);
    nlohmann::json stopBot(const std::string& id);
    nlohmann::json pauseBot(const std::string& id);
    nlohmann::json listBotExecutions(const std::string& id);
    nlohmann::json listBotLogs(const std::string& id);
    nlohmann::json listBotInstances();
    nlohmann::json currentBotUser();

    // -- Bot users --
    nlohmann::json listBotUsers();
    nlohmann::json createBotUser(const std::string& username,
                                 const std::string& password,
                                 const std::optional<std::string>& email = std::nullopt,
                                 const std::optional<std::string>& walletAddress = std::nullopt,
                                 const std::optional<std::string>& role = std::nullopt);
    nlohmann::json deleteBotUser(const std::string& id);
    nlohmann::json listBotTransactions();

    // -- Subscriptions --
    nlohmann::json getSubscription();
    nlohmann::json createSubscription(const std::string& tier,
                                      const std::optional<std::string>& expiresIn = std::nullopt);

    // -- Fees --
    nlohmann::json getFeeConfigs();
    nlohmann::json updateFeeConfig(const std::string& id,
                                   const std::optional<std::string>& name = std::nullopt,
                                   const std::optional<std::string>& percentage = std::nullopt,
                                   const std::optional<bool>& enabled = std::nullopt);

    // -- CEX connectors --
    nlohmann::json listCEX();
    nlohmann::json addCEX(const std::string& name, const nlohmann::json& config);
    nlohmann::json removeCEX(const std::string& id);

    // -- DEX connectors --
    nlohmann::json listDEX();
    nlohmann::json addDEX(const std::string& name, const nlohmann::json& config);
    nlohmann::json removeDEX(const std::string& id);

    // -- API keys --
    nlohmann::json listAPIKeys();
    nlohmann::json createAPIKey(const std::string& exchange, const std::string& apiKey);
    nlohmann::json deleteAPIKey(const std::string& id);

    // -- Admin (super-admin / finance-admin only) --
    nlohmann::json adminListUsers();
    nlohmann::json adminUserStatus(const std::string& id, bool active);
    nlohmann::json adminStats();
    nlohmann::json adminGetFeeAddresses();
    nlohmann::json adminSetFeeAddress(const std::string& label,
                                      long long chainId,
                                      const std::string& address);
    nlohmann::json adminDeleteFeeAddress(const std::string& id);
    nlohmann::json adminBotStatus(const std::string& id, const std::string& status);

    static constexpr const char* kDefaultBaseUrl = "http://localhost:8471/api/v1";

private:
    nlohmann::json request(const std::string& path,
                           const std::string& method,
                           const std::optional<nlohmann::json>& body = std::nullopt,
                           bool authenticated = true);
    nlohmann::json get(const std::string& path, bool authenticated = true);
    nlohmann::json del(const std::string& path);
    nlohmann::json post(const std::string& path, const nlohmann::json& body);
    nlohmann::json put(const std::string& path, const nlohmann::json& body);
    nlohmann::json postNoBody(const std::string& path);

    static std::string encodePathSegment(const std::string& s);

    std::string base_url_;
    std::optional<std::string> token_;
};

} // namespace tigerbots
