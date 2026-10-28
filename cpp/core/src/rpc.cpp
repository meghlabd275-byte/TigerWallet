// JSON-RPC client for Ethereum-compatible nodes. See include/rpc.hpp.

#include "rpc.hpp"

#include "crypto.hpp"

#include <cstdint>
#include <cstdlib>
#include <mutex>
#include <sstream>
#include <string>

#include <curl/curl.h>

namespace tigerwallet {
namespace {

std::once_flag g_curl_init;

void EnsureCurlInitialised() {
    std::call_once(g_curl_init, []() { curl_global_init(CURL_GLOBAL_DEFAULT); });
}

size_t WriteCallback(char* ptr, size_t size, size_t nmemb, void* userdata) {
    const size_t total = size * nmemb;
    static_cast<std::string*>(userdata)->append(ptr, total);
    return total;
}

// Escape a string for inclusion in a JSON document. Ethereum addresses, hashes
// and hex data are all plain ASCII, but this is applied regardless so that a
// caller cannot inject JSON through an argument.
std::string JsonEscape(const std::string& in) {
    std::string out;
    out.reserve(in.size() + 8);
    for (char ch : in) {
        const unsigned char c = static_cast<unsigned char>(ch);
        switch (c) {
            case '"': out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\b': out += "\\b"; break;
            case '\f': out += "\\f"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            default:
                if (c < 0x20) {
                    char buf[7];
                    std::snprintf(buf, sizeof(buf), "\\u%04x", c);
                    out += buf;
                } else {
                    out.push_back(static_cast<char>(c));
                }
        }
    }
    return out;
}

// Extract the string value of a top-level field. This is a narrow extractor for
// well-formed node responses, not a general JSON parser; it returns false when
// the field is absent, in which case the caller treats the response as an error.
bool ExtractJsonString(const std::string& body, const std::string& field, std::string& out) {
    const std::string key = "\"" + field + "\"";
    size_t pos = body.find(key);
    if (pos == std::string::npos) return false;
    pos = body.find(':', pos + key.size());
    if (pos == std::string::npos) return false;

    size_t value_start = pos + 1;
    while (value_start < body.size() &&
           (body[value_start] == ' ' || body[value_start] == '\t')) {
        ++value_start;
    }
    if (value_start >= body.size() || body[value_start] != '"') return false;

    // Walk the string, honouring backslash escapes, so an embedded quote does
    // not terminate the scan early.
    std::string value;
    for (size_t i = value_start + 1; i < body.size(); ++i) {
        const char c = body[i];
        if (c == '\\' && i + 1 < body.size()) {
            value.push_back(body[i + 1]);
            ++i;
            continue;
        }
        if (c == '"') {
            out = value;
            return true;
        }
        value.push_back(c);
    }
    return false;
}

// Parse a 0x-prefixed hexadecimal quantity into an unsigned integer.
bool ParseHexUint(const std::string& text, uint64_t& out) {
    if (text.size() < 3 || text[0] != '0' || (text[1] != 'x' && text[1] != 'X')) return false;
    uint64_t value = 0;
    for (size_t i = 2; i < text.size(); ++i) {
        const char c = text[i];
        uint64_t digit;
        if (c >= '0' && c <= '9') digit = static_cast<uint64_t>(c - '0');
        else if (c >= 'a' && c <= 'f') digit = static_cast<uint64_t>(c - 'a' + 10);
        else if (c >= 'A' && c <= 'F') digit = static_cast<uint64_t>(c - 'A' + 10);
        else return false;
        if (value > (UINT64_MAX - digit) / 16) return false;
        value = value * 16 + digit;
    }
    out = value;
    return true;
}

// Pull the JSON-RPC error message out of a response body, if present.
std::string ExtractErrorMessage(const std::string& body) {
    std::string message;
    if (ExtractJsonString(body, "message", message)) return message;
    return "node returned an error";
}

}  // namespace

RpcClient::RpcClient(std::string url, long timeout_seconds)
    : url_(std::move(url)), timeout_seconds_(timeout_seconds) {
    EnsureCurlInitialised();
}

RpcResult<std::string> RpcClient::Post(const std::string& method,
                                        const std::string& params_json) const {
    if (url_.empty()) {
        return RpcResult<std::string>::Fail(-1, "no RPC endpoint configured");
    }

    const std::string request = "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"" +
                                JsonEscape(method) + "\",\"params\":" + params_json + "}";

    CURL* curl = curl_easy_init();
    if (curl == nullptr) {
        return RpcResult<std::string>::Fail(-1, "curl_easy_init failed");
    }

    std::string response;
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");

    curl_easy_setopt(curl, CURLOPT_URL, url_.c_str());
    curl_easy_setopt(curl, CURLOPT_POST, 1L);
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, request.c_str());
    curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, static_cast<long>(request.size()));
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, timeout_seconds_);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, timeout_seconds_);
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);

    const CURLcode code = curl_easy_perform(curl);
    long http_status = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_status);

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (code != CURLE_OK) {
        return RpcResult<std::string>::Fail(-1, std::string("transport error: ") +
                                                    curl_easy_strerror(code));
    }
    if (http_status != 200) {
        std::ostringstream msg;
        msg << "HTTP " << http_status << " from " << url_;
        return RpcResult<std::string>::Fail(static_cast<int>(http_status), msg.str());
    }
    if (response.find("\"error\"") != std::string::npos) {
        return RpcResult<std::string>::Fail(-1, ExtractErrorMessage(response));
    }

    return RpcResult<std::string>::Ok(std::move(response));
}

RpcResult<std::string> RpcClient::Call(const std::string& method,
                                       const std::string& params_json) const {
    const auto response = Post(method, params_json);
    if (!response.ok) {
        return response;
    }

    std::string result;
    if (!ExtractJsonString(response.value, "result", result)) {
        return RpcResult<std::string>::Fail(-1, "response had no result field");
    }
    return RpcResult<std::string>::Ok(std::move(result));
}

RpcResult<std::string> RpcClient::CallRaw(const std::string& method,
                                          const std::string& params_json) const {
    return Post(method, params_json);
}

RpcResult<uint64_t> RpcClient::ChainId() const {
    const auto r = Call("eth_chainId", "[]");
    if (!r.ok) return RpcResult<uint64_t>::Fail(r.error.code, r.error.message);
    uint64_t value = 0;
    if (!ParseHexUint(r.value, value)) {
        return RpcResult<uint64_t>::Fail(-1, "eth_chainId returned a malformed quantity");
    }
    return RpcResult<uint64_t>::Ok(value);
}

RpcResult<std::string> RpcClient::GetBalance(const std::string& address,
                                             const std::string& block) const {
    const std::string params = "[\"" + JsonEscape(address) + "\",\"" + JsonEscape(block) + "\"]";
    const auto r = Call("eth_getBalance", params);
    if (!r.ok) return r;
    // Balances are 256-bit. Validating with the 64-bit parser rejected any
    // account holding more than about 18.4 ETH, which is most of them.
    if (!crypto::IsValidHexQuantity(r.value, 32)) {
        return RpcResult<std::string>::Fail(-1, "eth_getBalance returned a malformed quantity");
    }
    return r;
}

RpcResult<uint64_t> RpcClient::GetTransactionCount(const std::string& address,
                                                   const std::string& block) const {
    const std::string params = "[\"" + JsonEscape(address) + "\",\"" + JsonEscape(block) + "\"]";
    const auto r = Call("eth_getTransactionCount", params);
    if (!r.ok) return RpcResult<uint64_t>::Fail(r.error.code, r.error.message);
    uint64_t value = 0;
    if (!ParseHexUint(r.value, value)) {
        return RpcResult<uint64_t>::Fail(-1, "eth_getTransactionCount returned a malformed quantity");
    }
    return RpcResult<uint64_t>::Ok(value);
}

RpcResult<std::string> RpcClient::GetGasPrice() const {
    const auto r = Call("eth_gasPrice", "[]");
    if (!r.ok) return r;
    if (!crypto::IsValidHexQuantity(r.value, 32)) {
        return RpcResult<std::string>::Fail(-1, "eth_gasPrice returned a malformed quantity");
    }
    return r;
}

RpcResult<uint64_t> RpcClient::EstimateGas(const std::string& tx_object_json) const {
    const std::string params = "[" + tx_object_json + ",\"latest\"]";
    const auto r = Call("eth_estimateGas", params);
    if (!r.ok) return RpcResult<uint64_t>::Fail(r.error.code, r.error.message);
    uint64_t value = 0;
    if (!ParseHexUint(r.value, value)) {
        return RpcResult<uint64_t>::Fail(-1, "eth_estimateGas returned a malformed quantity");
    }
    return RpcResult<uint64_t>::Ok(value);
}

RpcResult<std::string> RpcClient::SendRawTransaction(const std::string& raw_hex) const {
    const std::string params = "[\"" + JsonEscape(raw_hex) + "\"]";
    return Call("eth_sendRawTransaction", params);
}

RpcResult<std::string> RpcClient::GetTransactionReceipt(const std::string& tx_hash) const {
    const std::string params = "[\"" + JsonEscape(tx_hash) + "\"]";
    return Call("eth_getTransactionReceipt", params);
}

RpcResult<std::string> RpcClient::EthCall(const std::string& to, const std::string& data) const {
    const std::string params =
        "[{\"to\":\"" + JsonEscape(to) + "\",\"data\":\"" + JsonEscape(data) + "\"},\"latest\"]";
    return Call("eth_call", params);
}

RpcResult<uint64_t> RpcClient::BlockNumber() const {
    const auto r = Call("eth_blockNumber", "[]");
    if (!r.ok) return RpcResult<uint64_t>::Fail(r.error.code, r.error.message);
    uint64_t value = 0;
    if (!ParseHexUint(r.value, value)) {
        return RpcResult<uint64_t>::Fail(-1, "eth_blockNumber returned a malformed quantity");
    }
    return RpcResult<uint64_t>::Ok(value);
}

}  // namespace tigerwallet
