/**
 * MasterWallet Desktop - API Client
 * Real HTTP client (libcurl) for the canonical MasterWallet backend on :8450.
 * All protected routes carry `Authorization: Bearer <JWT>`.
 * Honest results only: never fabricates balances, transactions, hashes,
 * addresses or signatures. Parse failures return std::nullopt / empty /
 * throw rather than inventing data.
 */

#ifndef MASTER_WALLET_API_CLIENT_HPP
#define MASTER_WALLET_API_CLIENT_HPP

#include <memory>
#include <string>
#include <map>
#include <vector>
#include <optional>
#include <stdexcept>
#include <cstdint>

namespace tiger {
namespace master {
namespace api {

enum class HTTPMethod {
    GET,
    POST,
    PUT,
    PATCH,
    DELETE
};

class APIException : public std::runtime_error {
public:
    enum class ErrorCode {
        NetworkError,
        Unauthorized,
        Forbidden,
        NotFound,
        RateLimited,
        ServerError,
        HTTPError,
        InvalidResponse
    };

    APIException(ErrorCode code, int statusCode, const std::string& message)
        : std::runtime_error(message), code_(code), statusCode_(statusCode) {}

    ErrorCode code() const { return code_; }
    int statusCode() const { return statusCode_; }

private:
    ErrorCode code_;
    int statusCode_;
};

class APIClient : public std::enable_shared_from_this<APIClient> {
public:
    static std::shared_ptr<APIClient> instance();

    APIClient();
    ~APIClient();

    void initialize(const std::string& baseUrl = "http://localhost:8450");
    void shutdown();

    void setBaseUrl(const std::string& baseUrl);
    std::string baseUrl() const { return baseUrl_; }

    void setAuthToken(const std::string& token);
    void clearAuthToken();
    std::string authToken() const { return authToken_; }

    void setTimeout(int seconds);
    bool isInitialized() const { return initialized_; }

    // Raw synchronous request. Returns response body; throws APIException on
    // network/HTTP errors.
    std::string request(HTTPMethod method,
                        const std::string& endpoint,
                        const std::optional<std::string>& body = std::nullopt,
                        const std::optional<std::map<std::string, std::string>>& params = std::nullopt);

    // Convenience helpers.
    std::string get(const std::string& endpoint,
                    const std::optional<std::map<std::string, std::string>>& params = std::nullopt);
    std::string post(const std::string& endpoint, const std::string& body);
    std::string put(const std::string& endpoint, const std::string& body);
    std::string del(const std::string& endpoint);

    // URL encoding for query values.
    static std::string urlEncode(const std::string& value);

private:
    APIClient(const APIClient&) = delete;
    APIClient& operator=(const APIClient&) = delete;

    std::string buildUrl(const std::string& endpoint,
                         const std::optional<std::map<std::string, std::string>>& params);

    std::string baseUrl_;
    std::string authToken_;
    bool initialized_ = false;
    int timeout_ = 30;
};

// ---- Backend convenience namespace -----------------------------------------
// Initializes a shared APIClient to http://localhost:8450 (overridable via the
// MASTER_WALLET_API_URL environment variable) and returns it.

std::shared_ptr<APIClient> backend();

std::string backendGet(const std::string& endpoint,
                       const std::optional<std::map<std::string, std::string>>& params = std::nullopt);
std::string backendPost(const std::string& endpoint, const std::string& body);
std::string backendPut(const std::string& endpoint, const std::string& body);
std::string backendDelete(const std::string& endpoint);

// ---- Minimal JSON helpers (no third-party dependency) ----------------------
// These handle flat JSON objects/arrays returned by the backend. They return
// std::nullopt on any parse failure rather than a fabricated value.

std::optional<std::string> jsonStringField(const std::string& json, const std::string& key);
std::optional<double> jsonNumberField(const std::string& json, const std::string& key);
std::optional<bool> jsonBoolField(const std::string& json, const std::string& key);

// Parse a JSON array-of-objects value (the value following `"key":`) into the
// individual object substrings (balanced text between top-level '{' and '}').
std::vector<std::string> jsonArrayOfObjects(const std::string& json, const std::string& key);

// First string element of an array value following `"key":`.
std::optional<std::string> jsonFirstStringArrayElement(const std::string& json, const std::string& key);

// Build a JSON object string from key/value pairs (string values only).
std::string buildJsonObject(const std::vector<std::pair<std::string, std::string>>& fields);
std::string jsonEscape(const std::string& value);

} // namespace api
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_API_CLIENT_HPP
