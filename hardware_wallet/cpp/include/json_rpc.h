/**
 * A minimal HTTPS JSON-RPC client.
 *
 * The hardware wallet module reads ERC-20 balances and NFT metadata straight
 * from an EVM node. This header provides the transport for that: a POST of a
 * JSON-RPC request over TLS and a check of the response. It uses OpenSSL only,
 * so the module keeps a single crypto dependency.
 */

#ifndef TIGERWALLET_JSON_RPC_H
#define TIGERWALLET_JSON_RPC_H

#include <cstdint>
#include <string>
#include <vector>

#include "json_value.h"

namespace tigerwallet {
namespace hardware {

/// What went wrong with a call, so the caller can report it rather than
/// returning a fabricated result.
enum class RpcStatus {
    Ok = 0,
    InvalidUrl,
    InvalidParameter,
    ConnectionFailed,
    TlsFailed,
    HttpError,
    MalformedResponse,
    NodeError,
    Timeout,
};

/// The outcome of one JSON-RPC call.
struct RpcResult {
    RpcStatus status = RpcStatus::ConnectionFailed;

    /// The `result` member, valid when status is Ok.
    json::Value result;

    /// A human-readable reason, set whenever status is not Ok. For NodeError
    /// this is the message the node returned.
    std::string error;

    bool ok() const { return status == RpcStatus::Ok; }
};

/// A single JSON-RPC endpoint.
class JsonRpcClient {
public:
    /// `url` must be https. The connection is established per call, which
    /// keeps the class free of shared state and safe to use from any thread.
    explicit JsonRpcClient(std::string url);

    /// Calls a method with no parameters.
    RpcResult call(const std::string& method) const;

    /// Calls a method with one string parameter, such as an address or a
    /// hex-encoded data blob.
    RpcResult call(const std::string& method, const std::string& parameter) const;

    /// Calls `eth_call` against a contract and returns the raw hex result.
    RpcResult eth_call(const std::string& to, const std::string& data) const;

    const std::string& url() const { return url_; }

    /// The number of seconds a single call may take before it is abandoned.
    void set_timeout_seconds(long seconds) { timeout_seconds_ = seconds; }

private:
    /// Sends a request body and returns the response body, or sets `error`.
    bool post(const std::string& body, std::string& response, std::string& error) const;

    /// Builds a JSON-RPC request envelope with a fixed identifier.
    static std::string build_request(const std::string& method,
                                     const std::string& params_json);

    std::string url_;
    long timeout_seconds_ = 15;
};

/// Parses an https URL into its host, port and path. Returns false when the
/// URL is not a usable https URL.
bool parse_https_url(const std::string& url, std::string& host, std::string& port,
                     std::string& path);

/// Decodes a 32-byte ABI word into a decimal string. Returns false when the
/// input is not exactly one hex word.
bool decode_abi_uint256(const std::string& hex_word, std::string& decimal);

/// Decodes a dynamic `string` return value: a 32-byte offset, a 32-byte
/// length, then the UTF-8 bytes. Returns false when the shape is wrong.
bool decode_abi_string(const std::string& hex_data, std::string& text);

}  // namespace hardware
}  // namespace tigerwallet

#endif  // TIGERWALLET_JSON_RPC_H
