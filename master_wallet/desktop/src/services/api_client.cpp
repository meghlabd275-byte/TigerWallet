/**
 * MasterWallet Desktop - API Client Implementation (libcurl)
 */

#include "api_client.hpp"

#include <curl/curl.h>
#include <cstdlib>
#include <cstring>
#include <iostream>
#include <mutex>
#include <sstream>

namespace tiger {
namespace master {
namespace api {

namespace {
struct GlobalCurlInit {
    GlobalCurlInit() { curl_global_init(CURL_GLOBAL_DEFAULT); }
    ~GlobalCurlInit() { curl_global_cleanup(); }
};
GlobalCurlInit g_curlInit;
} // namespace

std::shared_ptr<APIClient> APIClient::instance() {
    static std::shared_ptr<APIClient> inst = std::shared_ptr<APIClient>(new APIClient());
    return inst;
}

APIClient::APIClient()
    : initialized_(false), timeout_(30) {
    baseUrl_ = "http://localhost:8450";
    const char* env = std::getenv("MASTER_WALLET_API_URL");
    if (env && env[0]) baseUrl_ = env;
}

APIClient::~APIClient() {
    shutdown();
}

void APIClient::initialize(const std::string& baseUrl) {
    if (!baseUrl.empty()) baseUrl_ = baseUrl;
    initialized_ = true;
}

void APIClient::shutdown() {
    initialized_ = false;
}

void APIClient::setBaseUrl(const std::string& baseUrl) { baseUrl_ = baseUrl; }

void APIClient::setAuthToken(const std::string& token) { authToken_ = token; }
void APIClient::clearAuthToken() { authToken_.clear(); }
void APIClient::setTimeout(int seconds) { timeout_ = seconds; }

std::string APIClient::urlEncode(const std::string& value) {
    CURL* curl = curl_easy_init();
    char* enc = curl ? curl_easy_escape(curl, value.c_str(), static_cast<int>(value.size())) : nullptr;
    std::string out;
    if (enc) { out = enc; curl_free(enc); }
    else { out = value; }
    if (curl) curl_easy_cleanup(curl);
    return out;
}

std::string APIClient::buildUrl(const std::string& endpoint,
                                const std::optional<std::map<std::string, std::string>>& params) {
    std::string url = baseUrl_;
    if (!endpoint.empty() && endpoint[0] != '/' && baseUrl_.back() != '/') url += "/";
    url += endpoint;
    if (params && !params->empty()) {
        url += "?";
        bool first = true;
        for (const auto& kv : *params) {
            if (!first) url += "&";
            first = false;
            url += urlEncode(kv.first) + "=" + urlEncode(kv.second);
        }
    }
    return url;
}

namespace {
size_t writeCb(char* ptr, size_t size, size_t nmemb, void* userdata) {
    static_cast<std::string*>(userdata)->append(ptr, size * nmemb);
    return size * nmemb;
}
} // namespace

std::string APIClient::request(HTTPMethod method,
                                const std::string& endpoint,
                                const std::optional<std::string>& body,
                                const std::optional<std::map<std::string, std::string>>& params) {
    if (!initialized_) initialize(baseUrl_);

    std::string url = buildUrl(endpoint, params);
    std::string response;

    CURL* curl = curl_easy_init();
    if (!curl) {
        throw APIException(APIException::ErrorCode::NetworkError, 0, "curl_easy_init failed");
    }

    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");
    headers = curl_slist_append(headers, "Accept: application/json");
    if (!authToken_.empty()) {
        headers = curl_slist_append(headers, ("Authorization: Bearer " + authToken_).c_str());
    }

    switch (method) {
        case HTTPMethod::GET:    curl_easy_setopt(curl, CURLOPT_HTTPGET, 1L); break;
        case HTTPMethod::POST:   curl_easy_setopt(curl, CURLOPT_POST, 1L); break;
        case HTTPMethod::PUT:    curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT"); break;
        case HTTPMethod::PATCH:  curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PATCH"); break;
        case HTTPMethod::DELETE: curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "DELETE"); break;
    }

    if (body) {
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body->c_str());
        curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, static_cast<long>(body->size()));
    }

    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeCb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, static_cast<long>(timeout_));
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, static_cast<long>(timeout_));
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);

    CURLcode res = curl_easy_perform(curl);

    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK) {
        throw APIException(APIException::ErrorCode::NetworkError, 0,
                           std::string("Request failed: ") + curl_easy_strerror(res));
    }

    if (http_code == 401) throw APIException(APIException::ErrorCode::Unauthorized, http_code, "Unauthorized");
    if (http_code == 403) throw APIException(APIException::ErrorCode::Forbidden, http_code, "Forbidden");
    if (http_code == 404) throw APIException(APIException::ErrorCode::NotFound, http_code, "Not found");
    if (http_code == 429) throw APIException(APIException::ErrorCode::RateLimited, http_code, "Rate limited");
    if (http_code >= 500) throw APIException(APIException::ErrorCode::ServerError, http_code, "Server error");
    if (http_code >= 400) throw APIException(APIException::ErrorCode::HTTPError, http_code, "HTTP error");

    return response;
}

std::string APIClient::get(const std::string& endpoint,
                            const std::optional<std::map<std::string, std::string>>& params) {
    return request(HTTPMethod::GET, endpoint, std::nullopt, params);
}

std::string APIClient::post(const std::string& endpoint, const std::string& body) {
    return request(HTTPMethod::POST, endpoint, body);
}

std::string APIClient::put(const std::string& endpoint, const std::string& body) {
    return request(HTTPMethod::PUT, endpoint, body);
}

std::string APIClient::del(const std::string& endpoint) {
    return request(HTTPMethod::DELETE, endpoint);
}

// ---- Backend helpers -------------------------------------------------------

std::shared_ptr<APIClient> backend() {
    auto c = APIClient::instance();
    if (!c->isInitialized()) c->initialize();
    return c;
}

std::string backendGet(const std::string& endpoint,
                       const std::optional<std::map<std::string, std::string>>& params) {
    return backend()->get(endpoint, params);
}

std::string backendPost(const std::string& endpoint, const std::string& body) {
    return backend()->post(endpoint, body);
}

std::string backendPut(const std::string& endpoint, const std::string& body) {
    return backend()->put(endpoint, body);
}

std::string backendDelete(const std::string& endpoint) {
    return backend()->del(endpoint);
}

// ---- JSON helpers ----------------------------------------------------------

namespace {
bool jsonFindValue(const std::string& json, const std::string& key, size_t& outStart, size_t& outLen) {
    std::string needle = "\"" + key + "\"";
    size_t pos = 0;
    while ((pos = json.find(needle, pos)) != std::string::npos) {
        size_t valuePos = pos + needle.size();
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
} // namespace

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

std::optional<bool> jsonBoolField(const std::string& json, const std::string& key) {
    size_t start = 0, len = 0;
    if (!jsonFindValue(json, key, start, len)) return std::nullopt;
    std::string v = json.substr(start, len);
    if (v == "true") return true;
    if (v == "false") return false;
    return std::nullopt;
}

std::vector<std::string> jsonArrayOfObjects(const std::string& json, const std::string& key) {
    std::vector<std::string> out;
    std::string needle = "\"" + key + "\"";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) return out;
    size_t arr = json.find('[', pos);
    if (arr == std::string::npos) return out;

    size_t i = arr + 1;
    while (i < json.size()) {
        while (i < json.size() && (json[i] == ' ' || json[i] == '\t' || json[i] == '\n' ||
               json[i] == '\r' || json[i] == ',')) {
            ++i;
        }
        if (i >= json.size() || json[i] == ']') break;
        if (json[i] != '{') break;

        size_t start = i;
        int depth = 0;
        bool inString = false;
        bool escape = false;
        for (; i < json.size(); ++i) {
            char c = json[i];
            if (inString) {
                if (escape) { escape = false; }
                else if (c == '\\') { escape = true; }
                else if (c == '"') { inString = false; }
            } else {
                if (c == '"') { inString = true; }
                else if (c == '{') { ++depth; }
                else if (c == '}') {
                    --depth;
                    if (depth == 0) { ++i; break; }
                }
            }
        }
        if (depth != 0) break;
        out.push_back(json.substr(start, i - start));
    }
    return out;
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

std::string jsonEscape(const std::string& value) {
    std::string out;
    out.reserve(value.size() + 2);
    for (char c : value) {
        switch (c) {
            case '"':  out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            default:   out += c;
        }
    }
    return out;
}

std::string buildJsonObject(const std::vector<std::pair<std::string, std::string>>& fields) {
    std::ostringstream oss;
    oss << "{";
    for (size_t i = 0; i < fields.size(); ++i) {
        if (i) oss << ",";
        oss << "\"" << jsonEscape(fields[i].first) << "\":";
        const std::string& v = fields[i].second;
        // Numbers / booleans / null pass through unquoted.
        if (!v.empty() && (v == "true" || v == "false" || v == "null" ||
            (v[0] == '-' || (v[0] >= '0' && v[0] <= '9')))) {
            // Only treat as a number if it parses as a number.
            bool isNum = true;
            size_t j = (v[0] == '-') ? 1 : 0;
            if (j == v.size()) isNum = false;
            for (; j < v.size(); ++j) {
                char c = v[j];
                if (!((c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-')) {
                    isNum = false; break;
                }
            }
            if (isNum) { oss << v; }
            else { oss << "\"" << jsonEscape(v) << "\""; }
        } else {
            oss << "\"" << jsonEscape(v) << "\"";
        }
    }
    oss << "}";
    return oss.str();
}

} // namespace api
} // namespace master
} // namespace tiger
