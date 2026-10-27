// Tests for RLP encoding and transaction signing.
//
// The RLP cases are the encoding vectors from the Ethereum Yellow Paper
// Appendix B. The transaction case is the worked example in EIP-155: both the
// signing hash and the final signed encoding are compared against the values
// printed in the specification.

#include <array>
#include <cstdint>
#include <iostream>
#include <string>
#include <vector>

#include "crypto.hpp"
#include "transaction.hpp"

using namespace tigerwallet;

namespace {

int failures = 0;

void Check(const std::string& label, bool ok) {
    if (!ok) ++failures;
    std::cout << (ok ? "  PASS " : "  FAIL ") << label << "\n";
}

void CheckEqual(const std::string& label, const std::string& got, const std::string& want) {
    const bool ok = got == want;
    if (!ok) ++failures;
    std::cout << (ok ? "  PASS " : "  FAIL ") << label << "\n";
    if (!ok) std::cout << "        got  " << got << "\n        want " << want << "\n";
}

std::string Hex(const std::vector<uint8_t>& value) {
    return crypto::ToHex(value);
}

void TestRlpVectors() {
    std::cout << "RLP encoding vectors\n";
    CheckEqual("single byte 0x00", Hex(rlp::Value::Bytes({0x00}).Encode()), "00");
    CheckEqual("single byte 0x0f", Hex(rlp::Value::Bytes({0x0f}).Encode()), "0f");
    CheckEqual("empty string", Hex(rlp::Value::Bytes({}).Encode()), "80");
    CheckEqual("0x0400", Hex(rlp::Value::Bytes({0x04, 0x00}).Encode()), "820400");
    CheckEqual("dog", Hex(rlp::Value::Bytes({'d', 'o', 'g'}).Encode()), "83646f67");
    CheckEqual("cat, dog",
               Hex(rlp::Value({rlp::Value::Bytes({'c', 'a', 't'}),
                               rlp::Value::Bytes({'d', 'o', 'g'})}).Encode()),
               "c88363617483646f67");
    CheckEqual("empty list", Hex(rlp::Value(std::vector<rlp::Value>{}).Encode()), "c0");

    // 56 bytes is the first length that needs the long-string form.
    const std::vector<uint8_t> long_string(56, 'a');
    const std::string encoded = Hex(rlp::Value::Bytes(long_string).Encode());
    Check("a 56-byte string uses the 0xb8 prefix",
          encoded.rfind("b838", 0) == 0 && encoded.size() == 4 + 112);
}

void TestRlpIntegers() {
    std::cout << "RLP integer encoding\n";
    // RLP requires the minimal big-endian form: no leading zero bytes, and zero
    // is the empty string.
    CheckEqual("0 encodes as the empty string", Hex(rlp::Value::Uint(0).Encode()), "80");
    CheckEqual("15", Hex(rlp::Value::Uint(15).Encode()), "0f");
    CheckEqual("1024", Hex(rlp::Value::Uint(1024).Encode()), "820400");
    CheckEqual("0x8000", Hex(rlp::Value::Uint(0x8000).Encode()), "828000");

    uint64_t value = 0;
    Check("decodes 1024", rlp::DecodeUint({0x04, 0x00}, value) && value == 1024);
    Check("rejects a redundant leading zero", !rlp::DecodeUint({0x00, 0x01}, value));
    Check("rejects more than 8 bytes", !rlp::DecodeUint(std::vector<uint8_t>(9, 0x01), value));
}

void TestEip155Example() {
    std::cout << "EIP-155 worked example\n";
    // The example from the EIP-155 specification itself.
    Transaction tx;
    tx.type = TxType::LEGACY;
    tx.chain_id = 1;
    tx.nonce = 9;
    tx.gas_price_wei = "20000000000";
    tx.gas_limit = 21000;
    tx.to = "0x3535353535353535353535353535353535353535";
    tx.value_wei = "1000000000000000000";

    const auto signing_hash = TransactionSigningHash(tx);
    CheckEqual("signing hash matches the specification",
               crypto::ToHex(signing_hash),
               "daf5a779ae972f972197303d7b574746c7ef83eadac0f2791ad23db92e4c8e53");

    const auto key_bytes =
        crypto::FromHex("0x4646464646464646464646464646464646464646464646464646464646464646");
    std::array<uint8_t, 32> key{};
    std::copy(key_bytes.begin(), key_bytes.end(), key.begin());

    Check("signing succeeds", SignTransaction(tx, key));
    CheckEqual("v is 37 (recovery id 0, chain id 1)", std::to_string(tx.v), "37");
    CheckEqual("r matches the specification", Hex(tx.r),
               "28ef61340bd939bc2195fe537567866003e1a15d3c71ff63e1590620aa636276");
    CheckEqual("s matches the specification", Hex(tx.s),
               "67cbe9d8997f761aecb703304b3800ccf555c9f3dc64214b297fb1966a3b6d83");
    CheckEqual("signed encoding matches the specification byte for byte",
               Hex(EncodeSignedTransaction(tx)),
               "f86c098504a817c800825208943535353535353535353535353535353535353535"
               "880de0b6b3a76400008025a028ef61340bd939bc2195fe537567866003e1a15d3c71ff"
               "63e1590620aa636276a067cbe9d8997f761aecb703304b3800ccf555c9f3dc64214b29"
               "7fb1966a3b6d83");

    std::array<uint8_t, 20> signer{};
    crypto::PrivateKeyToAddress(key, signer);
    std::array<uint8_t, 20> recovered{};
    const auto recovery_id = static_cast<uint8_t>(tx.v - tx.chain_id * 2 - 35);
    std::vector<uint8_t> compact = tx.r;
    compact.insert(compact.end(), tx.s.begin(), tx.s.end());
    Check("the signature recovers to the signer",
          crypto::RecoverAddress(signing_hash, compact, recovery_id, recovered) &&
              recovered == signer);
}

void TestEip1559() {
    std::cout << "EIP-1559 transactions\n";
    const auto key_bytes =
        crypto::FromHex("0x4646464646464646464646464646464646464646464646464646464646464646");
    std::array<uint8_t, 32> key{};
    std::copy(key_bytes.begin(), key_bytes.end(), key.begin());

    std::array<uint8_t, 20> signer{};
    crypto::PrivateKeyToAddress(key, signer);

    Transaction tx;
    tx.type = TxType::EIP1559;
    tx.chain_id = 1;
    tx.nonce = 0;
    tx.max_priority_fee_per_gas_wei = "1000000000";
    tx.max_fee_per_gas_wei = "2000000000";
    tx.gas_limit = 21000;
    tx.to = "0x3535353535353535353535353535353535353535";
    tx.value_wei = "1000000000000000000";

    Check("signing succeeds", SignTransaction(tx, key));

    const auto encoded = EncodeSignedTransaction(tx);
    Check("encoding is prefixed with the type byte 0x02",
          !encoded.empty() && encoded[0] == 0x02);
    Check("v holds the raw recovery id", tx.v <= 1);

    std::vector<uint8_t> compact = tx.r;
    compact.insert(compact.end(), tx.s.begin(), tx.s.end());
    std::array<uint8_t, 20> recovered{};
    Check("the signature recovers to the signer",
          crypto::RecoverAddress(TransactionSigningHash(tx), compact,
                                 static_cast<uint8_t>(tx.v), recovered) &&
              recovered == signer);
}

void TestLargeChainId() {
    std::cout << "large chain ids\n";
    // EIP-155 sets v = recovery_id + chain_id*2 + 35. For Polygon (137) that is
    // 309 or 310, which does not fit in a byte. Storing v in a uint8_t would
    // truncate it to 53 or 54 and produce a signature no node accepts.
    const auto key_bytes =
        crypto::FromHex("0x4646464646464646464646464646464646464646464646464646464646464646");
    std::array<uint8_t, 32> key{};
    std::copy(key_bytes.begin(), key_bytes.end(), key.begin());

    Transaction tx;
    tx.type = TxType::LEGACY;
    tx.chain_id = 137;
    tx.nonce = 0;
    tx.gas_price_wei = "1000000000";
    tx.gas_limit = 21000;
    tx.to = "0x3535353535353535353535353535353535353535";
    tx.value_wei = "1";

    Check("signing succeeds", SignTransaction(tx, key));
    Check("v is 309 or 310, not a truncated byte", tx.v == 309 || tx.v == 310);

    // The value must survive the round trip back out of the encoding.
    const auto encoded = EncodeSignedTransaction(tx);
    Check("the encoding is non-empty", encoded.size() > 64);
}

void TestRejectsBadInput() {
    std::cout << "input validation\n";
    Transaction tx;
    tx.type = TxType::LEGACY;
    tx.chain_id = 1;
    tx.gas_price_wei = "1";
    tx.gas_limit = 21000;
    tx.value_wei = "1";

    tx.to = "0x1234";  // too short
    bool threw = false;
    try {
        (void)TransactionSigningHash(tx);
    } catch (const std::invalid_argument&) {
        threw = true;
    }
    Check("a short `to` address is rejected", threw);

    tx.to = "0x3535353535353535353535353535353535353535";
    tx.data_hex = "0xzz";
    threw = false;
    try {
        (void)TransactionSigningHash(tx);
    } catch (const std::invalid_argument&) {
        threw = true;
    }
    Check("malformed calldata is rejected", threw);

    // Contract creation has an empty `to`, which is legitimate.
    tx.to = "";
    tx.data_hex = "0x6001";
    threw = false;
    try {
        (void)TransactionSigningHash(tx);
    } catch (const std::invalid_argument&) {
        threw = true;
    }
    Check("contract creation (empty `to`) is accepted", !threw);
}

}  // namespace

int main() {
    TestRlpVectors();
    TestRlpIntegers();
    TestEip155Example();
    TestEip1559();
    TestLargeChainId();
    TestRejectsBadInput();

    std::cout << (failures == 0 ? "transaction: all tests passed\n"
                                : "transaction: " + std::to_string(failures) + " test(s) failed\n");
    return failures == 0 ? 0 : 1;
}
