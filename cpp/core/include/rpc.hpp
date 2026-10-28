// JSON-RPC client for Ethereum-compatible nodes.
//
// Every call goes to the operator's own endpoint over HTTP. Nothing here
// fabricates a result: a transport failure, an HTTP error or a JSON-RPC error
// all surface as a failure to the caller.

#ifndef TIGERWALLET_RPC_HPP
#define TIGERWALLET_RPC_HPP

#include <cstdint>
#include <optional>
#include <string>
#include <vector>

namespace tigerwallet {

struct RpcError {
    int code = 0;
    std::string message;
};

template <typename T>
struct RpcResult {
    bool ok = false;
    T value{};
    RpcError error;

    static RpcResult<T> Ok(T v) {
        RpcResult<T> r;
        r.ok = true;
        r.value = std::move(v);
        return r;
    }
    static RpcResult<T> Fail(int code, const std::string& message) {
        RpcResult<T> r;
        r.ok = false;
        r.error = RpcError{code, message};
        return r;
    }
};

class RpcClient {
public:
    // `timeout_seconds` applies to the whole request, including connect.
    explicit RpcClient(std::string url, long timeout_seconds = 10);

    const std::string& url() const { return url_; }

    // Raw JSON-RPC. `params_json` is the params array, e.g. "[]" or
    // "[\"0xabc\",\"latest\"]". Returns the string the "result" field holds.
    // A method whose result is an object or an array must use CallRaw.
    RpcResult<std::string> Call(const std::string& method, const std::string& params_json) const;

    // Raw JSON-RPC for a result that is not a string. Returns the complete
    // response body, so the caller reads the object or array it asked for.
    RpcResult<std::string> CallRaw(const std::string& method,
                                   const std::string& params_json) const;

    // eth_chainId
    RpcResult<uint64_t> ChainId() const;

    // eth_getBalance, in wei.
    RpcResult<std::string> GetBalance(const std::string& address, const std::string& block = "latest") const;

    // eth_getTransactionCount (nonce).
    RpcResult<uint64_t> GetTransactionCount(const std::string& address,
                                            const std::string& block = "latest") const;

    // eth_gasPrice, in wei.
    RpcResult<std::string> GetGasPrice() const;

    // eth_estimateGas, in gas units. `tx_object_json` is the transaction object.
    RpcResult<uint64_t> EstimateGas(const std::string& tx_object_json) const;

    // eth_sendRawTransaction. Returns the transaction hash.
    RpcResult<std::string> SendRawTransaction(const std::string& raw_hex) const;

    // eth_getTransactionReceipt. Returns the raw receipt JSON.
    RpcResult<std::string> GetTransactionReceipt(const std::string& tx_hash) const;

    // eth_call against the latest block.
    RpcResult<std::string> EthCall(const std::string& to, const std::string& data) const;

    // eth_blockNumber
    RpcResult<uint64_t> BlockNumber() const;

private:
    // Performs the request and returns the response body. A JSON-RPC error is
    // turned into a failure here, so the callers only deal with results.
    RpcResult<std::string> Post(const std::string& method,
                                const std::string& params_json) const;

    std::string url_;
    long timeout_seconds_;
};

}  // namespace tigerwallet

#endif  // TIGERWALLET_RPC_HPP
