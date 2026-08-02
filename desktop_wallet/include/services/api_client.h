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
    APIClient();
    ~APIClient();
    APIClient(const APIClient&) = delete;
    APIClient& operator=(const APIClient&) = delete;

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

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_API_CLIENT_H
