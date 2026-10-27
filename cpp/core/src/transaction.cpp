// Transaction construction, RLP encoding and signing. See include/transaction.hpp.

#include "transaction.hpp"

#include "crypto.hpp"

#include <algorithm>
#include <stdexcept>

namespace tigerwallet {
namespace rlp {
namespace {

// Big-endian bytes with no leading zeros; zero becomes the empty string.
std::vector<uint8_t> ToBigEndian(uint64_t value) {
    std::vector<uint8_t> out;
    while (value > 0) {
        out.push_back(static_cast<uint8_t>(value & 0xFF));
        value >>= 8;
    }
    std::reverse(out.begin(), out.end());
    return out;
}

std::vector<uint8_t> EncodeLength(size_t length, uint8_t offset) {
    if (length < 56) {
        return {static_cast<uint8_t>(offset + length)};
    }
    const auto len_bytes = ToBigEndian(static_cast<uint64_t>(length));
    std::vector<uint8_t> out;
    out.push_back(static_cast<uint8_t>(offset + 55 + len_bytes.size()));
    out.insert(out.end(), len_bytes.begin(), len_bytes.end());
    return out;
}

void Append(std::vector<uint8_t>& out, const std::vector<uint8_t>& part) {
    out.insert(out.end(), part.begin(), part.end());
}

}  // namespace

std::vector<uint8_t> EncodeUint(uint64_t value) {
    return ToBigEndian(value);
}

bool DecodeUint(const std::vector<uint8_t>& bytes, uint64_t& out) {
    if (bytes.size() > 8) return false;
    if (!bytes.empty() && bytes[0] == 0) return false;  // non-minimal encoding
    uint64_t value = 0;
    for (uint8_t b : bytes) {
        value = (value << 8) | b;
    }
    out = value;
    return true;
}

Value Value::Uint(uint64_t v) {
    return Value(EncodeUint(v));
}

std::vector<uint8_t> Value::Encode() const {
    std::vector<uint8_t> payload;
    if (is_list_) {
        for (const auto& item : items_) {
            Append(payload, item.Encode());
        }
        std::vector<uint8_t> out = EncodeLength(payload.size(), 0xC0);
        Append(out, payload);
        return out;
    }

    // Single byte below 0x80 encodes as itself.
    if (bytes_.size() == 1 && bytes_[0] < 0x80) {
        return {bytes_[0]};
    }
    std::vector<uint8_t> out = EncodeLength(bytes_.size(), 0x80);
    Append(out, bytes_);
    return out;
}

}  // namespace rlp

namespace {

// Decimal string to 32-byte big-endian. Values wider than 256 bits are
// truncated to zero rather than silently wrapping.
std::array<uint8_t, 32> DecimalToU256(const std::string& decimal) {
    std::array<uint8_t, 32> out{};
    for (char c : decimal) {
        if (c < '0' || c > '9') continue;
        uint32_t carry = static_cast<uint32_t>(c - '0');
        for (size_t i = 32; i-- > 0;) {
            const uint32_t cur = static_cast<uint32_t>(out[i]) * 10 + carry;
            out[i] = static_cast<uint8_t>(cur & 0xFF);
            carry = cur >> 8;
        }
    }
    return out;
}

// Strip leading zero bytes; zero becomes empty, as RLP requires for integers.
std::vector<uint8_t> MinimalBytes(const std::array<uint8_t, 32>& value) {
    size_t start = 0;
    while (start < 32 && value[start] == 0) ++start;
    return std::vector<uint8_t>(value.begin() + start, value.end());
}

std::vector<uint8_t> MinimalBytesFromDecimal(const std::string& decimal) {
    return MinimalBytes(DecimalToU256(decimal));
}

std::vector<uint8_t> AddressBytes(const std::string& address) {
    if (address.empty()) return {};  // contract creation
    const auto bytes = crypto::FromHex(address);
    if (bytes.size() != 20) {
        throw std::invalid_argument("transaction: `to` must be a 20-byte address");
    }
    return bytes;
}

std::vector<uint8_t> DataBytes(const std::string& data_hex) {
    if (data_hex.empty()) return {};
    std::vector<uint8_t> out;
    if (!crypto::FromHex(data_hex, out)) {
        throw std::invalid_argument("transaction: `data` is not valid hex");
    }
    return out;
}

rlp::Value LegacyFields(const Transaction& tx, bool for_signing) {
    std::vector<rlp::Value> fields = {
        rlp::Value::Uint(tx.nonce),
        rlp::Value::Bytes(MinimalBytesFromDecimal(tx.gas_price_wei)),
        rlp::Value::Uint(tx.gas_limit),
        rlp::Value::Bytes(AddressBytes(tx.to)),
        rlp::Value::Bytes(MinimalBytesFromDecimal(tx.value_wei)),
        rlp::Value::Bytes(DataBytes(tx.data_hex)),
    };

    if (for_signing) {
        // EIP-155: include chain id, then two empty fields in place of r and s.
        if (tx.chain_id > 0) {
            fields.push_back(rlp::Value::Uint(tx.chain_id));
            fields.push_back(rlp::Value::Bytes({}));
            fields.push_back(rlp::Value::Bytes({}));
        }
    } else {
        fields.push_back(rlp::Value::Uint(tx.v));
        fields.push_back(rlp::Value::Bytes(tx.r));
        fields.push_back(rlp::Value::Bytes(tx.s));
    }
    return rlp::Value(fields);
}

rlp::Value Eip1559Fields(const Transaction& tx, bool for_signing) {
    std::vector<rlp::Value> fields = {
        rlp::Value::Uint(tx.chain_id),
        rlp::Value::Uint(tx.nonce),
        rlp::Value::Bytes(MinimalBytesFromDecimal(tx.max_priority_fee_per_gas_wei)),
        rlp::Value::Bytes(MinimalBytesFromDecimal(tx.max_fee_per_gas_wei)),
        rlp::Value::Uint(tx.gas_limit),
        rlp::Value::Bytes(AddressBytes(tx.to)),
        rlp::Value::Bytes(MinimalBytesFromDecimal(tx.value_wei)),
        rlp::Value::Bytes(DataBytes(tx.data_hex)),
    };
    if (for_signing) {
        fields.push_back(rlp::Value::Bytes({}));  // access list
    } else {
        fields.push_back(rlp::Value::Bytes({}));  // access list
        fields.push_back(rlp::Value::Uint(tx.v));
        fields.push_back(rlp::Value::Bytes(tx.r));
        fields.push_back(rlp::Value::Bytes(tx.s));
    }
    return rlp::Value(fields);
}

}  // namespace

std::array<uint8_t, 32> TransactionSigningHash(const Transaction& tx) {
    if (tx.type == TxType::EIP1559) {
        const auto payload = Eip1559Fields(tx, true).Encode();
        // EIP-2718: the signing preimage is the type byte followed by the
        // RLP payload, and the whole thing is hashed.
        std::vector<uint8_t> preimage;
        preimage.push_back(static_cast<uint8_t>(TxType::EIP1559));
        preimage.insert(preimage.end(), payload.begin(), payload.end());
        return crypto::Keccak256(preimage);
    }
    return crypto::Keccak256(LegacyFields(tx, true).Encode());
}

std::vector<uint8_t> EncodeSignedTransaction(const Transaction& tx) {
    if (tx.type == TxType::EIP1559) {
        const auto payload = Eip1559Fields(tx, false).Encode();
        std::vector<uint8_t> out;
        out.push_back(static_cast<uint8_t>(TxType::EIP1559));
        out.insert(out.end(), payload.begin(), payload.end());
        return out;
    }
    return LegacyFields(tx, false).Encode();
}

std::string ComputeTransactionHash(const Transaction& tx) {
    const auto encoded = EncodeSignedTransaction(tx);
    return "0x" + crypto::ToHex(crypto::Keccak256(encoded));
}

bool SignTransaction(Transaction& tx, const std::array<uint8_t, 32>& private_key) {
    const auto digest = TransactionSigningHash(tx);

    uint8_t recovery_id = 0;
    const auto signature = crypto::SignHash(digest, private_key, recovery_id);
    if (signature.size() != 64) return false;

    tx.r.assign(signature.begin(), signature.begin() + 32);
    tx.s.assign(signature.begin() + 32, signature.end());

    if (tx.type == TxType::EIP1559) {
        tx.v = recovery_id;  // 0 or 1; the type byte disambiguates
    } else if (tx.chain_id > 0) {
        // EIP-155: v = recovery_id + chain_id * 2 + 35
        tx.v = recovery_id + tx.chain_id * 2 + 35;
    } else {
        tx.v = recovery_id + 27;
    }
    return true;
}

}  // namespace tigerwallet
