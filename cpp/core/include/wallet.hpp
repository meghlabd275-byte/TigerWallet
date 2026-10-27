// Wallet: key storage, signing and the node calls needed to send a transaction.
//
// Private keys are held only inside an encrypted blob. The plaintext key exists
// on the stack for the duration of a signing call and is wiped afterwards. It is
// never written to disk unencrypted and never returned to a caller.

#ifndef TIGERWALLET_WALLET_HPP
#define TIGERWALLET_WALLET_HPP

#include <array>
#include <cstdint>
#include <optional>
#include <string>
#include <vector>

#include "transaction.hpp"

namespace tigerwallet {

struct WalletAccount {
    std::string address;      // EIP-55 checksummed
    std::vector<uint8_t> public_key;       // 64-byte uncompressed, no prefix
    std::vector<uint8_t> encrypted_secret; // AES-256-GCM blob
    std::string label;
};

// Result of a broadcast attempt.
struct BroadcastResult {
    bool ok = false;
    std::string tx_hash;
    std::string raw_signed_hex;
    std::string error;
};

class Wallet {
public:
    // Creates an account from a 32-byte private key and encrypts it under
    // `password`. Returns std::nullopt if the key is not a valid secp256k1
    // scalar (zero, or >= the group order).
    static std::optional<WalletAccount> FromPrivateKey(const std::array<uint8_t, 32>& private_key,
                                                       const std::string& password,
                                                       const std::string& label = "");

    // Recovers the address from an encrypted account without decrypting it.
    static bool Address(const WalletAccount& account, std::string& out);

    // Signs a transaction and returns the raw signed bytes. The account must
    // already be unlocked. `chain_id` and `nonce` should be filled in from the
    // node before calling.
    static std::optional<std::vector<uint8_t>> SignTransaction(const WalletAccount& account,
                                                               const std::string& password,
                                                               Transaction& tx);

    // Signs a transaction, encodes it, and broadcasts it through `rpc_url`.
    // On success `tx` is left populated with v, r and s.
    static BroadcastResult SignAndSend(const WalletAccount& account,
                                       const std::string& password,
                                       Transaction& tx,
                                       const std::string& rpc_url);
};

// Fills nonce, gas price and chain id from the node. For EIP-1559
// transactions, `max_fee_per_gas_wei` and `max_priority_fee_per_gas_wei` are
// used as supplied (the caller is expected to have derived them from
// eth_feeHistory or eth_gasPrice). Returns false on any RPC failure, leaving
// the transaction untouched.
bool PopulateTransactionFromNode(const std::string& rpc_url,
                                 const std::string& from_address,
                                 Transaction& tx);

// Overwrites a buffer with zeros in a way the compiler is not permitted to
// remove. Used for private key material.
void SecureZero(void* data, size_t length);

}  // namespace tigerwallet

#endif  // TIGERWALLET_WALLET_HPP
