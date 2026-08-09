/**
 * TigerWallet Desktop - EVM Transaction Signer
 *
 * Real EVM transaction signing: RLP encoding, ECDSA secp256k1 signing
 * with EIP-155, and eth_sendRawTransaction broadcast. No unsigned
 * eth_sendTransaction, no 0x+64 zeros.
 */

#ifndef TIGERWALLET_EVM_SIGNER_H
#define TIGERWALLET_EVM_SIGNER_H

#include <string>
#include <vector>
#include <cstdint>

namespace tiger {
namespace wallet {

struct EvmTxParams {
    uint64_t nonce;
    uint64_t gasLimit;
    std::string gasPriceWei;     // hex string e.g. "0x77359400"
    std::string maxFeePerGas;    // EIP-1559, hex string (empty for legacy)
    std::string maxPriorityFee;  // EIP-1559, hex string (empty for legacy)
    std::string toAddress;       // hex string "0x..."
    std::string valueWei;        // hex string "0x..."
    std::string data;            // hex string "0x..."
    uint64_t chainId;
};

class EvmSigner {
public:
    // Sign a legacy (type-0) transaction with EIP-155.
    // Returns the hex-encoded signed raw transaction.
    static std::string signLegacy(const EvmTxParams& params,
                                   const std::vector<uint8_t>& privateKey);

    // Sign an EIP-1559 (type-2) transaction.
    static std::string signEIP1559(const EvmTxParams& params,
                                    const std::vector<uint8_t>& privateKey);

    // Personal sign: keccak256("\x19Ethereum Signed Message:\n" + len + msg)
    // Returns 65-byte signature as hex (r||s||v, v=27 or 28).
    static std::string personalSign(const std::string& message,
                                     const std::vector<uint8_t>& privateKey);

    // RLP encode a byte string (with length prefix).
    static std::vector<uint8_t> rlpEncodeBytes(const std::vector<uint8_t>& data);

    // RLP encode a list of already-encoded items.
    static std::vector<uint8_t> rlpEncodeList(const std::vector<std::vector<uint8_t>>& items);

    // RLP encode an unsigned integer.
    static std::vector<uint8_t> rlpEncodeUint(uint64_t value);

    // Convert hex string to bytes.
    static std::vector<uint8_t> hexToBytes(const std::string& hex);

    // Convert bytes to hex string (with 0x prefix).
    static std::string bytesToHex(const std::vector<uint8_t>& bytes);

private:
    // ECDSA sign a 32-byte hash. Returns 65 bytes: r(32) || s(32) || v(1).
    // v is the recovery id (0 or 1) — caller adds 27 for personal_sign, or
    // uses for EIP-155 chain_id calculation.
    static std::vector<uint8_t> ecdsaSign(const std::vector<uint8_t>& hash,
                                           const std::vector<uint8_t>& privateKey);

    // Low-level RLP length encoding.
    static void encodeLength(std::vector<uint8_t>& out, size_t len, uint8_t offset);
};

} // namespace wallet
} // namespace tiger

#endif // TIGERWALLET_EVM_SIGNER_H
