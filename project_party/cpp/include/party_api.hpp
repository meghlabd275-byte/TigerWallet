// ProjectParty C++ API client (libcurl).
//
// Targets the standalone ProjectParty backend (project_party/go/cmd/main.go,
// port 8106, path prefix /api/v1, JWT auth + RBAC). The token is held
// in-process (std::string) — C++ clients are server/desktop and have no
// keychain dependency by default; callers that need persistence can wire
// `setToken`/`token` to a secure storage of their choice.
//
// Every method issues a real libcurl easy-handle call against the backend —
// no stubs, fakes, or mock data. On any non-2xx response the method throws
// `PartyApiException` (fail-closed); it never returns fabricated data.
//
// Method set matches project_party/web/src/services/api.ts + the discovery,
// pricing, analytics, compliance routes the task requires.

#pragma once

#include <nlohmann/json.hpp>
#include <optional>
#include <stdexcept>
#include <string>

namespace tigerparty {

class PartyApiException : public std::runtime_error {
public:
    PartyApiException(int status, const std::string& message)
        : std::runtime_error("ProjectParty API " + std::to_string(status) + ": " + message),
          status_(status) {}
    int status() const noexcept { return status_; }

private:
    int status_;
};

struct PartyAuthResponse {
    std::optional<std::string> token;
    std::string username;
    std::string role;
};

class ProjectPartyApiClient {
public:
    explicit ProjectPartyApiClient(std::string baseUrl = kDefaultBaseUrl);
    ~ProjectPartyApiClient();

    ProjectPartyApiClient(const ProjectPartyApiClient&) = delete;
    ProjectPartyApiClient& operator=(const ProjectPartyApiClient&) = delete;

    void setToken(std::optional<std::string> token) { token_ = std::move(token); }
    [[nodiscard]] std::optional<std::string> token() const { return token_; }
    void clearToken() { token_.reset(); }
    [[nodiscard]] const std::string& baseUrl() const { return base_url_; }

    // -- Health (lives at /health, outside /api/v1) --
    nlohmann::json getHealth();

    // -- Auth --
    PartyAuthResponse registerUser(const std::string& username, const std::string& password);
    PartyAuthResponse login(const std::string& username, const std::string& password);

    // -- Discovery (public) --
    nlohmann::json getCoins();
    nlohmann::json searchTokens(const std::string& q);
    nlohmann::json getFeatured();
    nlohmann::json getTrending();
    nlohmann::json getMarket();

    // -- Tokens --
    nlohmann::json listTokens(const std::optional<std::string>& status = std::nullopt);
    nlohmann::json getToken(const std::string& id);
    nlohmann::json createToken(const nlohmann::json& data);
    nlohmann::json updateToken(const std::string& id, const nlohmann::json& data);
    nlohmann::json deleteToken(const std::string& id);
    nlohmann::json submitToken(const std::string& id);
    nlohmann::json approveToken(const std::string& id);
    nlohmann::json rejectToken(const std::string& id);

    // -- Listings --
    nlohmann::json listListings(const std::optional<std::string>& status = std::nullopt);
    nlohmann::json getListing(const std::string& id);
    nlohmann::json createListing(const nlohmann::json& data);
    nlohmann::json updateListingStatus(const std::string& id, const std::string& status);
    nlohmann::json featureListing(const std::string& id);

    // -- Launchpad --
    nlohmann::json listLaunchpads(const std::optional<std::string>& status = std::nullopt);
    nlohmann::json getLaunchpad(const std::string& id);
    nlohmann::json createLaunchpad(const nlohmann::json& data);
    nlohmann::json contribute(const std::string& id, const std::string& amount);
    nlohmann::json claimTokens(const std::string& id);
    nlohmann::json cancelLaunchpad(const std::string& id);

    // -- Market-making --
    nlohmann::json getMakerOrders(const std::optional<std::string>& tokenId = std::nullopt);
    nlohmann::json getMarketMakerStatus(const std::string& tokenId);
    nlohmann::json createMakerOrders(const nlohmann::json& data);
    nlohmann::json updateOrderStatus(const std::string& id, const std::string& status);
    nlohmann::json addLiquidity(const nlohmann::json& data);
    nlohmann::json removeLiquidity(const nlohmann::json& data);

    // -- Pricing --
    nlohmann::json getTokenPrice(const std::string& tokenId);
    nlohmann::json getPriceHistory(const std::string& tokenId);
    nlohmann::json setTokenPrice(const std::string& tokenId, const std::string& price);
    nlohmann::json updatePrice(const std::string& tokenId, const std::string& price);

    // -- Analytics (public) --
    nlohmann::json getTradingVolume();
    nlohmann::json getLiquidity();
    nlohmann::json getHolderCount();
    nlohmann::json getTransactionCount();

    // -- Compliance --
    nlohmann::json getAuditStatus(const std::string& tokenId);
    nlohmann::json getKYCStatus(const std::string& tokenId);
    nlohmann::json requestAudit(const nlohmann::json& data);
    nlohmann::json submitKYC(const nlohmann::json& data);

    // -- Fees --
    nlohmann::json getListingFees();
    nlohmann::json calculateFees(const nlohmann::json& data);
    nlohmann::json payFees(const nlohmann::json& data);

    // -- Favorites (auth) --
    nlohmann::json listFavorites();
    nlohmann::json addFavorite(const nlohmann::json& data);
    nlohmann::json removeFavorite(const std::string& id);

    static constexpr const char* kDefaultBaseUrl = "http://localhost:8106/api/v1";

private:
    nlohmann::json request(const std::string& path,
                           const std::string& method,
                           const std::optional<nlohmann::json>& body = std::nullopt,
                           bool authenticated = true,
                           bool absolute = false);
    nlohmann::json get(const std::string& path, bool authenticated = true);
    nlohmann::json del(const std::string& path);
    nlohmann::json post(const std::string& path, const nlohmann::json& body);
    nlohmann::json put(const std::string& path, const nlohmann::json& body);
    nlohmann::json postNoBody(const std::string& path);

    static std::string encodePathSegment(const std::string& s);
    static std::string encodeQuery(const std::string& s);

    std::string base_url_;
    std::optional<std::string> token_;
};

} // namespace tigerparty
