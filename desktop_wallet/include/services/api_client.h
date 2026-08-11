/**
 * TigerWallet Desktop - API Client
 * HTTP client for backend communication
 */

#ifndef TIGER_WALLET_API_CLIENT_H
#define TIGER_WALLET_API_CLIENT_H

#include <memory>
#include <string>
#include <map>
#include <vector>
#include <future>
#include <optional>
#include <functional>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

// ============================================================================
// HTTP Methods
// ============================================================================

enum class HTTPMethod {
    GET,
    POST,
    PUT,
    PATCH,
    DELETE
};

// ============================================================================
// API Client
// ============================================================================

class APIClient {
public:
    static std::shared_ptr<APIClient> getInstance();

    // Initialization
    void initialize(const std::string& baseUrl);
    void shutdown();

    // Configuration
    void setBaseUrl(const std::string& baseUrl);
    void setAuthToken(const std::string& token);
    void clearAuthToken();
    void setTimeout(int seconds);
    bool isInitialized() const { return initialized_; }

    // Requests
    template<typename T>
    std::future<T> get(const std::string& endpoint, const std::optional<std::map<std::string, std::string>>& params = std::nullopt);
    
    template<typename T, typename B>
    std::future<T> post(const std::string& endpoint, const B& body);
    
    template<typename T, typename B>
    std::future<T> put(const std::string& endpoint, const B& body);
    
    template<typename T, typename B>
    std::future<T> patch(const std::string& endpoint, const B& body);
    
    template<typename T>
    std::future<T> deleteRequest(const std::string& endpoint);

    // Error Handling
    class APIException : public std::runtime_error {
    public:
        enum class ErrorCode {
            InvalidURL,
            NetworkError,
            InvalidResponse,
            HTTPError,
            Unauthorized,
            Forbidden,
            NotFound,
            RateLimited,
            ServerError,
            Unknown
        };

        APIException(ErrorCode code, int statusCode, const std::string& message);
        ErrorCode getErrorCode() const;
        int getStatusCode() const;

    private:
        ErrorCode code_;
        int statusCode_;
    };

private:
    APIClient(const APIClient&) = delete;
    APIClient& operator=(const APIClient&) = delete;

public:
    APIClient();
    ~APIClient();

    // Request Building
    std::string buildUrl(const std::string& endpoint, const std::optional<std::map<std::string, std::string>>& params);
    std::string buildQueryString(const std::map<std::string, std::string>& params);

    // Response Handling
    template<typename T>
    T parseResponse(const std::string& response);

    // HTTP Execution
    std::string executeRequest(HTTPMethod method, const std::string& url, const std::optional<std::string>& body);

    // Members
    static std::shared_ptr<APIClient> instance_;
    CURL* curl_;
    bool initialized_;
    std::string baseUrl_;
    std::string authToken_;
    int timeout_;
};

// ============================================================================
// Template Implementations
// ============================================================================

template<typename T>
std::future<T> APIClient::get(const std::string& endpoint, const std::optional<std::map<std::string, std::string>>& params) {
    return std::async(std::launch::async, [this, endpoint, params]() -> T {
        std::string url = buildUrl(endpoint, params);
        std::string response = executeRequest(HTTPMethod::GET, url, std::nullopt);
        return parseResponse<T>(response);
    });
}

template<typename T, typename B>
std::future<T> APIClient::post(const std::string& endpoint, const B& body) {
    return std::async(std::launch::async, [this, endpoint, body]() -> T {
        std::string url = buildUrl(endpoint, std::nullopt);
        // Serialize body to JSON (simplified)
        std::string bodyStr = "{}"; 
        std::string response = executeRequest(HTTPMethod::POST, url, bodyStr);
        return parseResponse<T>(response);
    });
}

template<typename T, typename B>
std::future<T> APIClient::put(const std::string& endpoint, const B& body) {
    return std::async(std::launch::async, [this, endpoint, body]() -> T {
        std::string url = buildUrl(endpoint, std::nullopt);
        std::string bodyStr = "{}";
        std::string response = executeRequest(HTTPMethod::PUT, url, bodyStr);
        return parseResponse<T>(response);
    });
}

template<typename T, typename B>
std::future<T> APIClient::patch(const std::string& endpoint, const B& body) {
    return std::async(std::launch::async, [this, endpoint, body]() -> T {
        std::string url = buildUrl(endpoint, std::nullopt);
        std::string bodyStr = "{}";
        std::string response = executeRequest(HTTPMethod::PATCH, url, bodyStr);
        return parseResponse<T>(response);
    });
}

template<typename T>
std::future<T> APIClient::deleteRequest(const std::string& endpoint) {
    return std::async(std::launch::async, [this, endpoint]() -> T {
        std::string url = buildUrl(endpoint, std::nullopt);
        std::string response = executeRequest(HTTPMethod::DELETE, url, std::nullopt);
        return parseResponse<T>(response);
    });
}

template<typename T>
T APIClient::parseResponse(const std::string& response) {
    // Simplified - in production use proper JSON parsing
    T result;
    return result;
}

// ===========================================================================
// Real backend helpers (tiger::wallet wallet_api, default http://localhost:8443)
//
// These perform synchronous HTTP against the real backend and return honest
// results: the parsed value on success, std::nullopt / empty on failure.
// They NEVER fabricate quotes, prices, tx hashes, addresses or signatures.
// ===========================================================================

// Get the singleton APIClient, initializing it to the default backend URL if
// it has not been initialized yet.
std::shared_ptr<APIClient> backendClient();

// Synchronous GET/POST against the configured backend. Returns the raw
// response body on success and throws APIException on network/HTTP errors.
// Body for post() is sent as-is (JSON string).
std::string backendGet(const std::string& endpoint,
                       const std::optional<std::map<std::string, std::string>>& params = std::nullopt);
std::string backendPost(const std::string& endpoint, const std::string& body = "{}");

// Minimal, allocation-free JSON value extractors. They find the value
// associated with a top-level (or first-occurrence) key in a JSON object and
// return std::nullopt if absent/unparseable. Intended only for the small,
// well-known backend responses used by the wallet services.
std::optional<std::string> jsonStringField(const std::string& json, const std::string& key);
std::optional<double> jsonNumberField(const std::string& json, const std::string& key);
std::optional<std::string> jsonFirstStringArrayElement(const std::string& json, const std::string& key);

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_API_CLIENT_H
