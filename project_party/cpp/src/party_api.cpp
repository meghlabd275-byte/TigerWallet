// ProjectParty C++ API client implementation (libcurl).
//
// See party_api.hpp for the contract. All methods issue real HTTP calls via a
// libcurl easy handle. Fail-closed: any non-2xx response throws
// `PartyApiException` carrying the backend's `error`/`message` text.

#include "party_api.hpp"

#include <curl/curl.h>

#include <cctype>
#include <iomanip>
#include <sstream>

namespace tigerparty {

namespace {

class CurlGlobal {
public:
    CurlGlobal() { curl_global_init(CURL_GLOBAL_DEFAULT); }
    ~CurlGlobal() { curl_global_cleanup(); }
};

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

ProjectPartyApiClient::ProjectPartyApiClient(std::string baseUrl)
    : base_url_(std::move(baseUrl)) {
    curlGlobal();
}

ProjectPartyApiClient::~ProjectPartyApiClient() = default;

std::string ProjectPartyApiClient::encodePathSegment(const std::string& s) {
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

std::string ProjectPartyApiClient::encodeQuery(const std::string& s) {
    // Query values use the same percent-encoding as path segments.
    return encodePathSegment(s);
}

nlohmann::json ProjectPartyApiClient::request(const std::string& path,
                                              const std::string& method,
                                              const std::optional<nlohmann::json>& body,
                                              bool authenticated,
                                              bool absolute) {
    CURL* easy = curl_easy_init();
    if (!easy) {
        throw PartyApiException(0, "curl_easy_init failed");
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
                throw PartyApiException(401, "not authenticated: no JWT token set");
            }
            const std::string auth = "Authorization: Bearer " + *token_;
            headers = curl_slist_append(headers, auth.c_str());
        }
        std::string url;
        if (absolute) {
            std::string base = base_url_;
            const std::string suffix = "/api/v1";
            if (base.size() >= suffix.size() &&
                base.compare(base.size() - suffix.size(), suffix.size(), suffix) == 0) {
                base.erase(base.size() - suffix.size());
            }
            url = base + path;
        } else {
            url = base_url_ + path;
        }
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
            throw PartyApiException(0, std::string("curl: ") + curl_easy_strerror(rc));
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
            throw PartyApiException(static_cast<int>(httpCode), msg);
        }
        if (ctx.buf.empty()) return nlohmann::json{};
        return nlohmann::json::parse(ctx.buf);
    } catch (...) {
        if (headers) curl_slist_free_all(headers);
        curl_easy_cleanup(easy);
        throw;
    }
}

nlohmann::json ProjectPartyApiClient::get(const std::string& path, bool authenticated) {
    return request(path, "GET", std::nullopt, authenticated, false);
}

nlohmann::json ProjectPartyApiClient::del(const std::string& path) {
    return request(path, "DELETE", std::nullopt, true, false);
}

nlohmann::json ProjectPartyApiClient::post(const std::string& path, const nlohmann::json& body) {
    return request(path, "POST", body, true, false);
}

nlohmann::json ProjectPartyApiClient::put(const std::string& path, const nlohmann::json& body) {
    return request(path, "PUT", body, true, false);
}

nlohmann::json ProjectPartyApiClient::postNoBody(const std::string& path) {
    return request(path, "POST", std::nullopt, true, false);
}

// -- Health --

nlohmann::json ProjectPartyApiClient::getHealth() {
    return request("/health", "GET", std::nullopt, false, true);
}

// -- Auth --

PartyAuthResponse ProjectPartyApiClient::registerUser(const std::string& username,
                                                       const std::string& password) {
    auto res = request("/auth/register", "POST",
                       nlohmann::json{{"username", username}, {"password", password}},
                       false, false);
    PartyAuthResponse out{};
    if (res.contains("token") && res["token"].is_string()) {
        out.token = res["token"].get<std::string>();
        token_ = out.token;
    }
    out.username = res.value("username", "");
    out.role = res.value("role", "");
    return out;
}

PartyAuthResponse ProjectPartyApiClient::login(const std::string& username,
                                               const std::string& password) {
    auto res = request("/auth/login", "POST",
                       nlohmann::json{{"username", username}, {"password", password}},
                       false, false);
    PartyAuthResponse out{};
    if (res.contains("token") && res["token"].is_string()) {
        out.token = res["token"].get<std::string>();
        token_ = out.token;
    }
    out.username = res.value("username", "");
    out.role = res.value("role", "");
    return out;
}

// -- Discovery (public) --

nlohmann::json ProjectPartyApiClient::getCoins() { return get("/coins", false); }
nlohmann::json ProjectPartyApiClient::searchTokens(const std::string& q) {
    return get("/search?q=" + encodeQuery(q), false);
}
nlohmann::json ProjectPartyApiClient::getFeatured() { return get("/featured", false); }
nlohmann::json ProjectPartyApiClient::getTrending() { return get("/trending", false); }
nlohmann::json ProjectPartyApiClient::getMarket() { return get("/market", false); }

// -- Tokens --

nlohmann::json ProjectPartyApiClient::listTokens(const std::optional<std::string>& status) {
    return get(status ? "/tokens?status=" + encodeQuery(*status) : "/tokens");
}
nlohmann::json ProjectPartyApiClient::getToken(const std::string& id) {
    return get("/tokens/" + encodePathSegment(id));
}
nlohmann::json ProjectPartyApiClient::createToken(const nlohmann::json& data) {
    return post("/tokens", data);
}
nlohmann::json ProjectPartyApiClient::updateToken(const std::string& id, const nlohmann::json& data) {
    return put("/tokens/" + encodePathSegment(id), data);
}
nlohmann::json ProjectPartyApiClient::deleteToken(const std::string& id) {
    return del("/tokens/" + encodePathSegment(id));
}
nlohmann::json ProjectPartyApiClient::submitToken(const std::string& id) {
    return postNoBody("/tokens/" + encodePathSegment(id) + "/submit");
}
nlohmann::json ProjectPartyApiClient::approveToken(const std::string& id) {
    return postNoBody("/tokens/" + encodePathSegment(id) + "/approve");
}
nlohmann::json ProjectPartyApiClient::rejectToken(const std::string& id) {
    return postNoBody("/tokens/" + encodePathSegment(id) + "/reject");
}

// -- Listings --

nlohmann::json ProjectPartyApiClient::listListings(const std::optional<std::string>& status) {
    return get(status ? "/listings?status=" + encodeQuery(*status) : "/listings");
}
nlohmann::json ProjectPartyApiClient::getListing(const std::string& id) {
    return get("/listings/" + encodePathSegment(id));
}
nlohmann::json ProjectPartyApiClient::createListing(const nlohmann::json& data) {
    return post("/listings", data);
}
nlohmann::json ProjectPartyApiClient::updateListingStatus(const std::string& id,
                                                          const std::string& status) {
    return put("/listings/" + encodePathSegment(id) + "/status",
               nlohmann::json{{"status", status}});
}
nlohmann::json ProjectPartyApiClient::featureListing(const std::string& id) {
    return postNoBody("/listings/" + encodePathSegment(id) + "/featured");
}

// -- Launchpad --

nlohmann::json ProjectPartyApiClient::listLaunchpads(const std::optional<std::string>& status) {
    return get(status ? "/launchpad?status=" + encodeQuery(*status) : "/launchpad");
}
nlohmann::json ProjectPartyApiClient::getLaunchpad(const std::string& id) {
    return get("/launchpad/" + encodePathSegment(id));
}
nlohmann::json ProjectPartyApiClient::createLaunchpad(const nlohmann::json& data) {
    return post("/launchpad/create", data);
}
nlohmann::json ProjectPartyApiClient::contribute(const std::string& id, const std::string& amount) {
    return post("/launchpad/" + encodePathSegment(id) + "/contribute",
                nlohmann::json{{"amount", amount}});
}
nlohmann::json ProjectPartyApiClient::claimTokens(const std::string& id) {
    return postNoBody("/launchpad/" + encodePathSegment(id) + "/claim");
}
nlohmann::json ProjectPartyApiClient::cancelLaunchpad(const std::string& id) {
    return postNoBody("/launchpad/" + encodePathSegment(id) + "/cancel");
}

// -- Market-making --

nlohmann::json ProjectPartyApiClient::getMakerOrders(const std::optional<std::string>& tokenId) {
    return get(tokenId ? "/market-making/orders?token_id=" + encodeQuery(*tokenId)
                       : "/market-making/orders");
}
nlohmann::json ProjectPartyApiClient::getMarketMakerStatus(const std::string& tokenId) {
    return get("/market-making/status/" + encodePathSegment(tokenId));
}
nlohmann::json ProjectPartyApiClient::createMakerOrders(const nlohmann::json& data) {
    return post("/market-making/orders", data);
}
nlohmann::json ProjectPartyApiClient::updateOrderStatus(const std::string& id,
                                                        const std::string& status) {
    return put("/market-making/orders/" + encodePathSegment(id) + "/status",
               nlohmann::json{{"status", status}});
}
nlohmann::json ProjectPartyApiClient::addLiquidity(const nlohmann::json& data) {
    return post("/market-making/liquidity/add", data);
}
nlohmann::json ProjectPartyApiClient::removeLiquidity(const nlohmann::json& data) {
    return post("/market-making/liquidity/remove", data);
}

// -- Pricing --

nlohmann::json ProjectPartyApiClient::getTokenPrice(const std::string& tokenId) {
    return get("/pricing/" + encodePathSegment(tokenId));
}
nlohmann::json ProjectPartyApiClient::getPriceHistory(const std::string& tokenId) {
    return get("/pricing/history/" + encodePathSegment(tokenId));
}
nlohmann::json ProjectPartyApiClient::setTokenPrice(const std::string& tokenId,
                                                    const std::string& price) {
    return post("/pricing/set", nlohmann::json{{"token_id", tokenId}, {"price", price}});
}
nlohmann::json ProjectPartyApiClient::updatePrice(const std::string& tokenId,
                                                  const std::string& price) {
    return post("/pricing/update", nlohmann::json{{"token_id", tokenId}, {"price", price}});
}

// -- Analytics (public) --

nlohmann::json ProjectPartyApiClient::getTradingVolume() { return get("/analytics/volume", false); }
nlohmann::json ProjectPartyApiClient::getLiquidity() { return get("/analytics/liquidity", false); }
nlohmann::json ProjectPartyApiClient::getHolderCount() { return get("/analytics/holders", false); }
nlohmann::json ProjectPartyApiClient::getTransactionCount() { return get("/analytics/transactions", false); }

// -- Compliance --

nlohmann::json ProjectPartyApiClient::getAuditStatus(const std::string& tokenId) {
    return get("/compliance/audit/" + encodePathSegment(tokenId));
}
nlohmann::json ProjectPartyApiClient::getKYCStatus(const std::string& tokenId) {
    return get("/compliance/kyc/" + encodePathSegment(tokenId));
}
nlohmann::json ProjectPartyApiClient::requestAudit(const nlohmann::json& data) {
    return post("/compliance/audit", data);
}
nlohmann::json ProjectPartyApiClient::submitKYC(const nlohmann::json& data) {
    return post("/compliance/kyc/submit", data);
}

// -- Fees --

nlohmann::json ProjectPartyApiClient::getListingFees() { return get("/fees", false); }
nlohmann::json ProjectPartyApiClient::calculateFees(const nlohmann::json& data) {
    return post("/fees/calculate", data);
}
nlohmann::json ProjectPartyApiClient::payFees(const nlohmann::json& data) {
    return post("/fees/pay", data);
}

// -- Favorites (auth) --

nlohmann::json ProjectPartyApiClient::listFavorites() { return get("/favorites"); }
nlohmann::json ProjectPartyApiClient::addFavorite(const nlohmann::json& data) {
    return post("/favorites", data);
}
nlohmann::json ProjectPartyApiClient::removeFavorite(const std::string& id) {
    return del("/favorites/" + encodePathSegment(id));
}

} // namespace tigerparty
