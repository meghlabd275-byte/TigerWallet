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

// ============================================================================
// Real backend helpers - synchronous access to the tiger::wallet wallet_api
// backend (default http://localhost:8443). Honest results only: never
// fabricate quotes, prices, tx hashes, addresses or signatures.
// ============================================================================

std::shared_ptr<APIClient> backendClient() {
    auto client = APIClient::getInstance();
    if (!client->isInitialized()) {
        client->initialize("http://localhost:8443");
    }
    return client;
}

std::string backendGet(const std::string& endpoint,
                       const std::optional<std::map<std::string, std::string>>& params) {
    auto client = backendClient();
    std::string url = client->buildUrl(endpoint, params);
    return client->executeRequest(HTTPMethod::GET, url, std::nullopt);
}

std::string backendPost(const std::string& endpoint, const std::string& body) {
    auto client = backendClient();
    std::string url = client->buildUrl(endpoint, std::nullopt);
    return client->executeRequest(HTTPMethod::POST, url, body);
}

// Minimal JSON field extractors (no third-party dependency). These only handle
// the flat JSON objects returned by the wallet_api backend used by the desktop
// wallet services. Returns std::nullopt on any parse failure rather than a
// fabricated value.

static bool jsonFindValue(const std::string& json, const std::string& key, size_t& outStart, size_t& outLen) {
    // Build the key pattern: "key"
    std::string needle = "\"" + key + "\"";
    size_t pos = 0;
    while ((pos = json.find(needle, pos)) != std::string::npos) {
        size_t valuePos = pos + needle.size();
        // skip whitespace and a single ':'
        while (valuePos < json.size() && (json[valuePos] == ' ' || json[valuePos] == '\t' ||
               json[valuePos] == '\n' || json[valuePos] == '\r')) {
            ++valuePos;
        }
        if (valuePos >= json.size() || json[valuePos] != ':') {
            pos += needle.size();
            continue;
        }
        ++valuePos;
        while (valuePos < json.size() && (json[valuePos] == ' ' || json[valuePos] == '\t' ||
               json[valuePos] == '\n' || json[valuePos] == '\r')) {
            ++valuePos;
        }
        if (valuePos >= json.size()) return false;

        outStart = valuePos;
        if (json[valuePos] == '"') {
            // string value
            size_t s = valuePos + 1;
            size_t e = s;
            while (e < json.size() && json[e] != '"') {
                if (json[e] == '\\' && e + 1 < json.size()) ++e;
                ++e;
            }
            if (e >= json.size()) return false;
            outStart = s;
            outLen = e - s;
            return true;
        } else {
            // number / literal value runs to next , } ]
            size_t e = valuePos;
            while (e < json.size() && json[e] != ',' && json[e] != '}' && json[e] != ']' &&
                   json[e] != ' ' && json[e] != '\t' && json[e] != '\n' && json[e] != '\r') {
                ++e;
            }
            outStart = valuePos;
            outLen = e - valuePos;
            return outLen > 0;
        }
    }
    return false;
}

std::optional<std::string> jsonStringField(const std::string& json, const std::string& key) {
    size_t start = 0, len = 0;
    if (!jsonFindValue(json, key, start, len)) return std::nullopt;
    return json.substr(start, len);
}

std::optional<double> jsonNumberField(const std::string& json, const std::string& key) {
    size_t start = 0, len = 0;
    if (!jsonFindValue(json, key, start, len)) return std::nullopt;
    try {
        return std::stod(json.substr(start, len));
    } catch (...) {
        return std::nullopt;
    }
}

std::optional<std::string> jsonFirstStringArrayElement(const std::string& json, const std::string& key) {
    std::string needle = "\"" + key + "\"";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) return std::nullopt;
    size_t arrStart = json.find('[', pos);
    if (arrStart == std::string::npos) return std::nullopt;
    size_t strStart = json.find('"', arrStart);
    if (strStart == std::string::npos) return std::nullopt;
    size_t strEnd = strStart + 1;
    while (strEnd < json.size() && json[strEnd] != '"') {
        if (json[strEnd] == '\\' && strEnd + 1 < json.size()) ++strEnd;
        ++strEnd;
    }
    if (strEnd >= json.size()) return std::nullopt;
    return json.substr(strStart + 1, strEnd - strStart - 1);
}

} // namespace wallet
} // namespace tiger
