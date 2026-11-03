// TigerWallet DEX connector interfaces.
//
// Every value a connector reports is read from the chain it is configured
// against. There is no cached pool table, no fixed price and no simulated fill:
// a connector that cannot reach its node reports the failure instead.

#ifndef TIGERWALLET_DEX_CONNECTOR_HPP
#define TIGERWALLET_DEX_CONNECTOR_HPP

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

namespace tigerwallet {
namespace dex {

using Timestamp = int64_t;

enum class DexProtocol {
    UNISWAP_V2,
    UNISWAP_V3,
    SUSHISWAP,
    PANCAKESWAP,
    QUICKSWAP,
    CURVE,
    BALANCER
};

const char* to_string(DexProtocol protocol);

// ---------------------------------------------------------------------------
// Data structures
// ---------------------------------------------------------------------------

struct Token {
    std::string address;   // Lowercase 0x-prefixed contract address.
    std::string symbol;
    std::string name;
    uint8_t decimals = 18;
};

struct Pool {
    std::string address;       // Pair contract.
    Token token_a;
    Token token_b;
    std::string reserve_a;     // Raw reserve as a decimal string.
    std::string reserve_b;     // Raw reserve as a decimal string.
    uint32_t fee_bps = 30;     // 30 == 0.30%.
};

struct Quote {
    std::string from_token;
    std::string to_token;
    std::string from_amount;   // Raw input amount.
    std::string to_amount;     // Raw output amount from the router.
    std::string price;         // to_amount / from_amount, decimal string.
    std::vector<std::string> path;
};

struct SwapResult {
    bool success = false;
    std::string from_token;
    std::string to_token;
    std::string from_amount;
    std::string to_amount;     // Output measured from the balance delta.
    std::string tx_hash;
    std::string error_message;
};

// ---------------------------------------------------------------------------
// ABI helpers (exposed for testing)
// ---------------------------------------------------------------------------

// 4-byte function selector for `signature`, e.g. "balanceOf(address)".
std::string function_selector(const std::string& signature);

// Left-pads a 20-byte address into a 32-byte word.
std::string encode_address(const std::string& address);

// Encodes a decimal string as a 32-byte big-endian word. Rejects negative
// values and values that do not fit in 256 bits.
bool encode_uint256(const std::string& decimal, std::string& out_word);

// Decodes a 32-byte big-endian word into a decimal string.
std::string decode_uint256(const std::string& word32);

// Decodes a dynamic `bytes` or `string` return value (offset + length + data).
std::string decode_string(const std::string& return_data);

// Decodes a dynamic address[] return value.
std::vector<std::string> decode_address_array(const std::string& return_data);

// Reads the `amounts` array that a V2 router quote returns.
bool decode_amounts_array(const std::string& return_data, std::vector<std::string>& out);

// Applies a V2 fee (in basis points) to an amount.
std::string apply_fee(const std::string& amount, uint32_t fee_bps);

// Decimal string arithmetic, used for quotes and balance deltas.
bool decimal_add(const std::string& a, const std::string& b, std::string& out);
bool decimal_sub(const std::string& a, const std::string& b, std::string& out);
bool decimal_mul_div(const std::string& a, const std::string& b,
                     const std::string& divisor, std::string& out);

// ---------------------------------------------------------------------------
// Connector
// ---------------------------------------------------------------------------

class DexConnector {
public:
    virtual ~DexConnector() = default;

    virtual bool connect() = 0;
    virtual void disconnect() = 0;
    virtual bool is_connected() const = 0;

    // Reads symbol/name/decimals from the token contract.
    virtual bool get_token_info(const std::string& address, Token& out) = 0;

    // Reads reserves from the pair contract for `token_a`/`token_b`.
    virtual bool get_pool(const std::string& token_a, const std::string& token_b,
                          Pool& out) = 0;

    // Router quote. Returns false when no route exists.
    virtual bool get_quote(const std::string& from_token, const std::string& to_token,
                           const std::string& amount, Quote& out) = 0;

    // Sends a swap and reports the amount actually received, measured from the
    // recipient's balance delta.
    virtual SwapResult execute_swap(const std::string& from_token,
                                    const std::string& to_token,
                                    const std::string& amount,
                                    const std::string& min_out) = 0;

    // ERC-20 balance held by `owner`.
    virtual bool get_balance(const std::string& token, const std::string& owner,
                             std::string& out) = 0;

    virtual DexProtocol protocol() const = 0;
    virtual uint64_t chain_id() const = 0;
    virtual const std::string& name() const = 0;
};

// Builds a V2-style connector. `router` and `factory` must be the deployed
// contracts on `chain_id`; a connector never assumes a mainnet address.
std::unique_ptr<DexConnector> make_v2_connector(DexProtocol protocol,
                                                std::string rpc_url,
                                                uint64_t chain_id,
                                                std::string router,
                                                std::string factory,
                                                std::string private_key = "");

}  // namespace dex
}  // namespace tigerwallet

#endif  // TIGERWALLET_DEX_CONNECTOR_HPP
