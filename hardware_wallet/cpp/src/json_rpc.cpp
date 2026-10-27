#include "json_rpc.h"

#include <openssl/err.h>
#include <openssl/ssl.h>
#include <openssl/x509v3.h>

#include <netdb.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <unistd.h>

#include <algorithm>
#include <cstring>

namespace tigerwallet {
namespace hardware {

namespace {

/// Closes a descriptor when the object goes out of scope.
class ScopedFd {
public:
    explicit ScopedFd(int fd) : fd_(fd) {}
    ~ScopedFd() {
        if (fd_ >= 0) {
            ::close(fd_);
        }
    }

    ScopedFd(const ScopedFd&) = delete;
    ScopedFd& operator=(const ScopedFd&) = delete;

    int get() const { return fd_; }
    bool valid() const { return fd_ >= 0; }

private:
    int fd_;
};

/// Frees an SSL object when it goes out of scope.
class ScopedSsl {
public:
    ScopedSsl(SSL* ssl, SSL_CTX* ctx) : ssl_(ssl), ctx_(ctx) {}
    ~ScopedSsl() {
        if (ssl_ != nullptr) {
            SSL_free(ssl_);
        }
        if (ctx_ != nullptr) {
            SSL_CTX_free(ctx_);
        }
    }

    ScopedSsl(const ScopedSsl&) = delete;
    ScopedSsl& operator=(const ScopedSsl&) = delete;

    SSL* get() const { return ssl_; }

private:
    SSL* ssl_;
    SSL_CTX* ctx_;
};

std::string last_openssl_error() {
    const unsigned long code = ERR_get_error();
    if (code == 0) {
        return "unknown TLS error";
    }

    char buffer[256];
    ERR_error_string_n(code, buffer, sizeof(buffer));
    return buffer;
}

/// Splits a hex string into bytes. Returns false on an odd length or a
/// character that is not a hex digit.
bool hex_to_bytes(const std::string& hex, std::vector<uint8_t>& out) {
    std::string trimmed = hex;
    if (trimmed.size() >= 2 && trimmed[0] == '0' &&
        (trimmed[1] == 'x' || trimmed[1] == 'X')) {
        trimmed = trimmed.substr(2);
    }

    if (trimmed.size() % 2 != 0) {
        return false;
    }

    out.clear();
    out.reserve(trimmed.size() / 2);

    auto nibble = [](char c) -> int {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return c - 'a' + 10;
        if (c >= 'A' && c <= 'F') return c - 'A' + 10;
        return -1;
    };

    for (size_t i = 0; i < trimmed.size(); i += 2) {
        const int high = nibble(trimmed[i]);
        const int low = nibble(trimmed[i + 1]);
        if (high < 0 || low < 0) {
            return false;
        }
        out.push_back(static_cast<uint8_t>((high << 4) | low));
    }

    return true;
}

}  // namespace

bool parse_https_url(const std::string& url, std::string& host, std::string& port,
                     std::string& path) {
    const std::string scheme = "https://";
    if (url.compare(0, scheme.size(), scheme) != 0) {
        return false;
    }

    const std::string rest = url.substr(scheme.size());
    const size_t slash = rest.find('/');

    std::string authority = (slash == std::string::npos) ? rest : rest.substr(0, slash);
    path = (slash == std::string::npos) ? "/" : rest.substr(slash);

    if (authority.empty()) {
        return false;
    }

    // A user-info section is not accepted: credentials must not travel in a URL.
    if (authority.find('@') != std::string::npos) {
        return false;
    }

    const size_t colon = authority.rfind(':');
    if (colon != std::string::npos) {
        host = authority.substr(0, colon);
        port = authority.substr(colon + 1);

        if (port.empty() ||
            !std::all_of(port.begin(), port.end(),
                         [](char c) { return c >= '0' && c <= '9'; })) {
            return false;
        }
    } else {
        host = authority;
        port = "443";
    }

    return !host.empty();
}

bool decode_abi_uint256(const std::string& hex_word, std::string& decimal) {
    std::vector<uint8_t> bytes;
    if (!hex_to_bytes(hex_word, bytes) || bytes.size() != 32) {
        return false;
    }

    // Convert the big-endian word to decimal by repeated division, so a value
    // larger than 64 bits is still exact.
    std::vector<uint8_t> value = bytes;
    std::string digits;

    while (true) {
        unsigned int remainder = 0;
        bool all_zero = true;

        for (auto& byte : value) {
            const unsigned int current = (remainder << 8) | byte;
            byte = static_cast<uint8_t>(current / 10);
            remainder = current % 10;
            if (byte != 0) {
                all_zero = false;
            }
        }

        digits.push_back(static_cast<char>('0' + remainder));

        if (all_zero) {
            break;
        }
    }

    std::reverse(digits.begin(), digits.end());
    decimal = digits.empty() ? "0" : digits;
    return true;
}

bool decode_abi_string(const std::string& hex_data, std::string& text) {
    std::vector<uint8_t> bytes;
    if (!hex_to_bytes(hex_data, bytes) || bytes.size() < 64) {
        return false;
    }

    // The first word is the offset to the string data.
    size_t offset = 0;
    for (size_t i = 0; i < 32; i++) {
        offset = (offset << 8) | bytes[i];
    }

    if (offset + 32 > bytes.size()) {
        return false;
    }

    size_t length = 0;
    for (size_t i = offset; i < offset + 32; i++) {
        length = (length << 8) | bytes[i];
    }

    if (length == 0) {
        text.clear();
        return true;
    }

    if (offset + 32 + length > bytes.size()) {
        return false;
    }

    text.assign(reinterpret_cast<const char*>(bytes.data() + offset + 32), length);
    return true;
}

JsonRpcClient::JsonRpcClient(std::string url) : url_(std::move(url)) {}

std::string JsonRpcClient::build_request(const std::string& method,
                                         const std::string& params_json) {
    // The identifier is fixed: one request is in flight per connection, so
    // there is nothing to correlate.
    return "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"" + method +
           "\",\"params\":" + params_json + "}";
}

bool JsonRpcClient::post(const std::string& body, std::string& response,
                         std::string& error) const {
    std::string host;
    std::string port;
    std::string path;

    if (!parse_https_url(url_, host, port, path)) {
        error = "the endpoint URL is not a usable https URL";
        return false;
    }

    struct addrinfo hints {};
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;

    struct addrinfo* addresses = nullptr;
    if (::getaddrinfo(host.c_str(), port.c_str(), &hints, &addresses) != 0 ||
        addresses == nullptr) {
        error = "the host could not be resolved";
        return false;
    }

    int socket_fd = -1;
    for (struct addrinfo* entry = addresses; entry != nullptr; entry = entry->ai_next) {
        socket_fd = ::socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (socket_fd < 0) {
            continue;
        }

        struct timeval timeout {};
        timeout.tv_sec = timeout_seconds_;
        ::setsockopt(socket_fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
        ::setsockopt(socket_fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));

        if (::connect(socket_fd, entry->ai_addr, entry->ai_addrlen) == 0) {
            break;
        }

        ::close(socket_fd);
        socket_fd = -1;
    }

    ::freeaddrinfo(addresses);

    ScopedFd connection(socket_fd);
    if (!connection.valid()) {
        error = "the connection could not be established";
        return false;
    }

    SSL_CTX* ctx = SSL_CTX_new(TLS_client_method());
    if (ctx == nullptr) {
        error = "the TLS context could not be created";
        return false;
    }

    // Only current protocol versions, and the peer certificate is verified
    // against the system trust store with the host name checked.
    SSL_CTX_set_min_proto_version(ctx, TLS1_2_VERSION);
    SSL_CTX_set_verify(ctx, SSL_VERIFY_PEER, nullptr);
    SSL_CTX_set_default_verify_paths(ctx);

    SSL* ssl = SSL_new(ctx);
    if (ssl == nullptr) {
        error = "the TLS session could not be created";
        SSL_CTX_free(ctx);
        return false;
    }

    SSL_set_tlsext_host_name(ssl, host.c_str());
    SSL_set1_host(ssl, host.c_str());
    SSL_set_fd(ssl, connection.get());

    ScopedSsl session(ssl, ctx);

    if (SSL_connect(ssl) != 1) {
        error = "the TLS handshake failed: " + last_openssl_error();
        return false;
    }

    // The request is built from a fixed template, so the only variable parts
    // are the body length and the host. Both are checked for characters that
    // would break the framing.
    if (host.find_first_of("\r\n") != std::string::npos ||
        path.find_first_of("\r\n") != std::string::npos) {
        error = "the endpoint URL has a line break";
        return false;
    }

    std::string request = "POST " + path + " HTTP/1.1\r\n";
    request += "Host: " + host + "\r\n";
    request += "Content-Type: application/json\r\n";
    request += "Accept: application/json\r\n";
    // Several public nodes reject a request with no User-Agent.
    request += "User-Agent: TigerWallet/1.0\r\n";
    request += "Content-Length: " + std::to_string(body.size()) + "\r\n";
    request += "Connection: close\r\n\r\n";
    request += body;

    size_t written = 0;
    while (written < request.size()) {
        const int n = SSL_write(ssl, request.data() + written,
                                static_cast<int>(request.size() - written));
        if (n <= 0) {
            error = "the request could not be sent: " + last_openssl_error();
            return false;
        }
        written += static_cast<size_t>(n);
    }

    std::string raw;
    char buffer[8192];

    while (true) {
        const int n = SSL_read(ssl, buffer, sizeof(buffer));
        if (n > 0) {
            raw.append(buffer, static_cast<size_t>(n));

            // A response larger than this is not something this client should
            // be asked to carry.
            if (raw.size() > 8u * 1024u * 1024u) {
                error = "the response is larger than 8 MiB";
                return false;
            }
            continue;
        }

        const int ssl_error = SSL_get_error(ssl, n);
        if (ssl_error == SSL_ERROR_ZERO_RETURN || ssl_error == SSL_ERROR_SYSCALL) {
            break;
        }
        if (ssl_error == SSL_ERROR_WANT_READ || ssl_error == SSL_ERROR_WANT_WRITE) {
            continue;
        }

        error = "the response could not be read: " + last_openssl_error();
        return false;
    }

    const size_t header_end = raw.find("\r\n\r\n");
    if (header_end == std::string::npos) {
        error = "the response has no header terminator";
        return false;
    }

    const std::string status_line = raw.substr(0, raw.find("\r\n"));
    const size_t first_space = status_line.find(' ');
    if (first_space == std::string::npos || status_line.size() < first_space + 4) {
        error = "the response has no status line";
        return false;
    }

    const std::string code_text = status_line.substr(first_space + 1, 3);
    const int status_code = std::atoi(code_text.c_str());
    if (status_code < 200 || status_code >= 300) {
        error = "the endpoint answered with HTTP " + code_text;
        return false;
    }

    response = raw.substr(header_end + 4);
    return true;
}

RpcResult JsonRpcClient::call(const std::string& method) const {
    return call(method, "");
}

RpcResult JsonRpcClient::call(const std::string& method,
                              const std::string& parameter) const {
    RpcResult outcome;

    std::string params = "[]";
    if (!parameter.empty()) {
        params = "[\"" + parameter + "\"]";
    }

    std::string body;
    std::string error;

    if (!post(build_request(method, params), body, error)) {
        outcome.status = RpcStatus::ConnectionFailed;
        outcome.error = error;
        return outcome;
    }

    json::Value document;
    try {
        document = json::Value::parse(body);
    } catch (const std::exception& parse_error) {
        outcome.status = RpcStatus::MalformedResponse;
        outcome.error = std::string("the response is not valid JSON: ") + parse_error.what();
        return outcome;
    }

    // A JSON-RPC error member means the node refused the call.
    if (const json::Value* rpc_error = document.find("error")) {
        if (!rpc_error->is_null()) {
            outcome.status = RpcStatus::NodeError;
            if (const json::Value* message = rpc_error->find("message")) {
                outcome.error = message->is_string() ? message->as_string()
                                                     : "the node returned an error";
            } else {
                outcome.error = "the node returned an error";
            }
            return outcome;
        }
    }

    const json::Value* result = document.find("result");
    if (result == nullptr) {
        outcome.status = RpcStatus::MalformedResponse;
        outcome.error = "the response has no result member";
        return outcome;
    }

    outcome.status = RpcStatus::Ok;
    outcome.result = *result;
    return outcome;
}

RpcResult JsonRpcClient::eth_call(const std::string& to,
                                  const std::string& data) const {
    RpcResult outcome;

    // The call object is built from hex strings the caller supplied. They are
    // validated so a quote cannot break out of the JSON string.
    auto is_hex = [](const std::string& value) {
        if (value.size() < 3 || value.compare(0, 2, "0x") != 0) {
            return false;
        }
        return std::all_of(value.begin() + 2, value.end(), [](char c) {
            return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') ||
                   (c >= 'A' && c <= 'F');
        });
    };

    if (!is_hex(to) || !is_hex(data)) {
        outcome.status = RpcStatus::InvalidParameter;
        outcome.error = "the contract address or call data is not hex";
        return outcome;
    }

    const std::string params =
        "[{\"to\":\"" + to + "\",\"data\":\"" + data + "\"},\"latest\"]";

    std::string body;
    std::string error;

    if (!post(build_request("eth_call", params), body, error)) {
        outcome.status = RpcStatus::ConnectionFailed;
        outcome.error = error;
        return outcome;
    }

    json::Value document;
    try {
        document = json::Value::parse(body);
    } catch (const std::exception& parse_error) {
        outcome.status = RpcStatus::MalformedResponse;
        outcome.error = std::string("the response is not valid JSON: ") + parse_error.what();
        return outcome;
    }

    if (const json::Value* rpc_error = document.find("error")) {
        if (!rpc_error->is_null()) {
            outcome.status = RpcStatus::NodeError;
            if (const json::Value* message = rpc_error->find("message")) {
                outcome.error = message->is_string() ? message->as_string()
                                                     : "the node returned an error";
            } else {
                outcome.error = "the node returned an error";
            }
            return outcome;
        }
    }

    const json::Value* result = document.find("result");
    if (result == nullptr || !result->is_string()) {
        outcome.status = RpcStatus::MalformedResponse;
        outcome.error = "the response has no result member";
        return outcome;
    }

    outcome.status = RpcStatus::Ok;
    outcome.result = *result;
    return outcome;
}

}  // namespace hardware
}  // namespace tigerwallet
