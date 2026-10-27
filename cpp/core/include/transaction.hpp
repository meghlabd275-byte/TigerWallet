// Transaction construction, RLP encoding and signing.
//
// RLP is implemented here rather than pulled from a library because it is small
// and fully specified, and because the encoding is consensus-critical: a single
// wrong length prefix produces a transaction the network rejects or, worse, a
// different signing hash than the one that was signed.

#ifndef TIGERWALLET_TRANSACTION_HPP
#define TIGERWALLET_TRANSACTION_HPP

#include <array>
#include <cstdint>
#include <optional>
#include <string>
#include <vector>

namespace tigerwallet {

// ---------------------------------------------------------------------------
// RLP (Recursive Length Prefix, Ethereum Yellow Paper Appendix B)
// ---------------------------------------------------------------------------

namespace rlp {

// A value is either a byte string or a list of values.
class Value {
public:
    Value() = default;
    explicit Value(std::vector<uint8_t> bytes) : bytes_(std::move(bytes)), is_list_(false) {}
    explicit Value(std::vector<Value> items) : items_(std::move(items)), is_list_(true) {}

    static Value Bytes(const std::vector<uint8_t>& b) { return Value(b); }
    static Value Uint(uint64_t v);

    bool is_list() const { return is_list_; }
    const std::vector<uint8_t>& bytes() const { return bytes_; }
    const std::vector<Value>& items() const { return items_; }

    // Serialize this value, including its own header.
    std::vector<uint8_t> Encode() const;

private:
    std::vector<uint8_t> bytes_;
    std::vector<Value> items_;
    bool is_list_ = false;
};

// Minimal big-endian encoding of an unsigned integer: no leading zero bytes,
// and zero encodes as the empty string.
std::vector<uint8_t> EncodeUint(uint64_t value);

// Decode a big-endian unsigned integer. Returns false if the input is longer
// than 8 bytes or has a redundant leading zero byte.
bool DecodeUint(const std::vector<uint8_t>& bytes, uint64_t& out);

}  // namespace rlp

// ---------------------------------------------------------------------------
// Transactions
// ---------------------------------------------------------------------------

enum class TxType : uint8_t {
    LEGACY = 0,     // pre-EIP-2718
    EIP1559 = 2,    // dynamic fee
};

struct Transaction {
    uint64_t chain_id = 1;
    uint64_t nonce = 0;
    std::string to;              // empty for contract creation
    std::string value_wei;       // decimal string; wei does not fit in 64 bits
    std::string data_hex;        // 0x-prefixed, may be empty
    uint64_t gas_limit = 21000;

    // Legacy pricing. For EIP-1559 these are unused.
    std::string gas_price_wei;

    TxType type = TxType::LEGACY;
    std::string max_fee_per_gas_wei;        // EIP-1559
    std::string max_priority_fee_per_gas_wei;  // EIP-1559

    // Populated after signing. v is 64-bit because EIP-155 sets
    // v = recovery_id + chain_id*2 + 35, which exceeds a byte for any chain id
    // above 109 (Polygon is 137, so v is 309 or 310).
    uint64_t v = 0;
    std::vector<uint8_t> r;
    std::vector<uint8_t> s;
};

// Signing hash: keccak256 over the RLP-encoded transaction with the signature
// fields omitted (and, for legacy transactions, the EIP-155 chain id included).
std::array<uint8_t, 32> TransactionSigningHash(const Transaction& tx);

// Full signed encoding, ready to broadcast via eth_sendRawTransaction.
std::vector<uint8_t> EncodeSignedTransaction(const Transaction& tx);

// Transaction hash (keccak256 of the signed encoding).
std::string ComputeTransactionHash(const Transaction& tx);

// Signs in place: fills v, r and s. Returns false if the private key is invalid.
// The recovery id is stored as v = recovery_id + 27 for legacy transactions and
// v = recovery_id for EIP-1559 (the type byte already disambiguates).
bool SignTransaction(Transaction& tx, const std::array<uint8_t, 32>& private_key);

}  // namespace tigerwallet

#endif  // TIGERWALLET_TRANSACTION_HPP
