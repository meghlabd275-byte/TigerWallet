/**
 * TigerWallet Desktop - API Client Implementation
 */

#include "services/api_client.h"
#include <iostream>
#include <sstream>
#include <thread>
#include <chrono>
#include <cmath>

namespace tiger {
namespace wallet {

// ============================================================================
// Static Instance
// ============================================================================

std::shared_ptr<APIClient> APIClient::instance_ = nullptr;

// ============================================================================
// Constructor/Destructor
// ============================================================================

APIClient::APIClient() : curl_(nullptr), initialized_(false), timeout_(30) {}

APIClient::~APIClient() {
    shutdown();
}

// ============================================================================
// Singleton
// ============================================================================

std::shared_ptr<APIClient> APIClient::getInstance() {
    if (!instance_) {
        instance_ = std::make_shared<APIClient>();
    }
    return instance_;
}

// ============================================================================
// Initialization
// ============================================================================

void APIClient::initialize(const std::string& baseUrl) {
    if (initialized_) return;
    
    curl_ = curl_easy_init();
    baseUrl_ = baseUrl;
    initialized_ = true;
    std::cout << "[APIClient] Initialized with base URL: " << baseUrl << std::endl;
}

void APIClient::shutdown() {
    if (curl_) {
        curl_easy_cleanup(curl_);
        curl_ = nullptr;
    }
    initialized_ = false;
}

// ============================================================================
// Configuration
// ============================================================================

void APIClient::setBaseUrl(const std::string& baseUrl) {
    baseUrl_ = baseUrl;
}

void APIClient::setAuthToken(const std::string& token) {
    authToken_ = token;
}

void APIClient::clearAuthToken() {
    authToken_.clear();
}

void APIClient::setTimeout(int seconds) {
    timeout_ = seconds;
}

// ============================================================================
// Request Building
// ============================================================================

std::string APIClient::buildUrl(const std::string& endpoint, const std::optional<std::map<std::string, std::string>>& params) {
    std::string url = baseUrl_ + endpoint;
    
    if (params) {
        url += "?" + buildQueryString(*params);
    }
    
    return url;
}

std::string APIClient::buildQueryString(const std::map<std::string, std::string>& params) {
    std::ostringstream oss;
    
    bool first = true;
    for (const auto& pair : params) {
        if (!first) oss << "&";
        first = false;
        
        // URL encode value
        oss << pair.first << "=" << pair.second;
    }
    
    return oss.str();
}

// ============================================================================
// HTTP Execution
// ============================================================================

std::string APIClient::executeRequest(HTTPMethod method, const std::string& url, const std::optional<std::string>& body) {
    if (!curl_) {
        curl_ = curl_easy_init();
    }
    
    std::string response_string;
    struct curl_slist* headers = nullptr;
    
    // Set headers
    headers = curl_slist_append(headers, "Content-Type: application/json");
    headers = curl_slist_append(headers, "Accept: application/json");
    
    if (!authToken_.empty()) {
        std::string authHeader = "Authorization: Bearer " + authToken_;
        headers = curl_slist_append(headers, authHeader.c_str());
    }
    
    // Set method
    switch (method) {
        case HTTPMethod::GET:
            curl_easy_setopt(curl_, CURLOPT_HTTPGET, 1);
            break;
        case HTTPMethod::POST:
            curl_easy_setopt(curl_, CURLOPT_POST, 1);
            break;
        case HTTPMethod::PUT:
            curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, "PUT");
            break;
        case HTTPMethod::PATCH:
            curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, "PATCH");
            break;
        case HTTPMethod::DELETE:
            curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, "DELETE");
            break;
    }
    
    // Set body if present
    if (body) {
        curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, body->c_str());
    }
    
    // Set URL and headers
    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, +[](char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* str = static_cast<std::string*>(userdata);
        str->append(ptr, size * nmemb);
        return size * nmemb;
    });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response_string);
    curl_easy_setopt(curl_, CURLOPT_TIMEOUT, timeout_);
    curl_easy_setopt(curl_, CURLOPT_FOLLOWLOCATION, 1L);
    
    CURLcode res = curl_easy_perform(curl_);
    
    long http_code = 0;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &http_code);
    
    curl_slist_free_all(headers);
    
    if (res != CURLE_OK) {
        throw APIException(APIException::ErrorCode::NetworkError, 0,
            std::string("Request failed: ") + curl_easy_strerror(res));
    }
    
    // Handle HTTP errors
    switch (http_code) {
        case 401:
            throw APIException(APIException::ErrorCode::Unauthorized, http_code, "Unauthorized");
        case 403:
            throw APIException(APIException::ErrorCode::Forbidden, http_code, "Forbidden");
        case 404:
            throw APIException(APIException::ErrorCode::NotFound, http_code, "Not found");
        case 429:
            throw APIException(APIException::ErrorCode::RateLimited, http_code, "Rate limited");
        case 500 ... 599:
            throw APIException(APIException::ErrorCode::ServerError, http_code, "Server error");
    }
    
    return response_string;
}

// ============================================================================
// API Exception
// ============================================================================

APIClient::APIException::APIException(ErrorCode code, int statusCode, const std::string& message)
    : std::runtime_error(message), code_(code), statusCode_(statusCode) {}

APIClient::APIException::ErrorCode APIClient::APIException::getErrorCode() const {
    return code_;
}

int APIClient::APIException::getStatusCode() const {
    return statusCode_;
}

} // namespace wallet
} // namespace tiger
