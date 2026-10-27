// Wallet implementation. See include/wallet.hpp.

#include "wallet.hpp"

#include "crypto.hpp"
#include "rpc.hpp"

#include <cstring>
#include <string>

#include <openssl/bn.h>
#include <openssl/ec.h>
#include <openssl/obj_mac.h>

namespace tigerwallet {
namespace {

// True when the 32-byte value is a valid secp256k1 private key: non-zero and
// below the group order n. libsecp256k1 rejects an out-of-range key when it is
// used, but checking here lets FromPrivateKey fail cleanly instead of producing
// an account whose key cannot sign.
bool IsValidPrivateKey(const std::array<uint8_t, 32>& key) {
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
    if (group == nullptr) return false;

    BIGNUM* order = BN_new();
    BIGNUM* value = BN_bin2bn(key.data(), static_cast<int>(key.size()), nullptr);
    bool ok = false;
    if (order != nullptr && value != nullptr && EC_GROUP_get_order(group, order, nullptr) == 1) {
        ok = !BN_is_zero(value) && BN_cmp(value, order) < 0;
    }

    BN_free(value);
    BN_free(order);
    EC_GROUP_free(group);
    return ok;
}

}  // namespace

void SecureZero(void* data, size_t length) {
    if (data == nullptr || length == 0) return;
    volatile uint8_t* p = static_cast<volatile uint8_t*>(data);
    while (length-- > 0) *p++ = 0;
}

std::optional<WalletAccount> Wallet::FromPrivateKey(const std::array<uint8_t, 32>& private_key,
                                                    const std::string& password,
                                                    const std::string& label) {
    if (!IsValidPrivateKey(private_key)) return std::nullopt;

    const auto public_key = crypto::PrivateKeyToPublicKey(private_key);
    if (public_key.size() != 64) return std::nullopt;

    std::array<uint8_t, 20> address_bytes{};
    try {
        address_bytes = crypto::PublicKeyToAddress(public_key.data(), public_key.size());
    } catch (const std::exception&) {
        return std::nullopt;
    }

    WalletAccount account;
    account.address = crypto::AddressToString(address_bytes);
    account.public_key = public_key;
    account.label = label;
    try {
        account.encrypted_secret =
            crypto::AesGcmEncrypt(password, private_key.data(), private_key.size());
    } catch (const std::exception&) {
        return std::nullopt;
    }
    return account;
}

bool Wallet::Address(const WalletAccount& account, std::string& out) {
    if (account.address.empty()) return false;
    out = account.address;
    return true;
}

std::optional<std::vector<uint8_t>> Wallet::SignTransaction(const WalletAccount& account,
                                                            const std::string& password,
                                                            Transaction& tx) {
    const auto secret = crypto::AesGcmDecrypt(password, account.encrypted_secret.data(),
                                              account.encrypted_secret.size());
    if (!secret || secret->size() != 32) return std::nullopt;

    std::array<uint8_t, 32> private_key{};
    std::memcpy(private_key.data(), secret->data(), 32);

    // The decrypted key must belong to this account; a mismatched blob would
    // otherwise sign with the wrong key.
    std::array<uint8_t, 20> derived{};
    const bool matches = crypto::PrivateKeyToAddress(private_key, derived) &&
                         crypto::AddressToString(derived) == account.address;

    std::optional<std::vector<uint8_t>> result;
    // Qualified: the unqualified name would resolve to this member function.
    if (matches && tigerwallet::SignTransaction(tx, private_key)) {
        result = EncodeSignedTransaction(tx);
    }

    SecureZero(private_key.data(), private_key.size());
    return result;
}

BroadcastResult Wallet::SignAndSend(const WalletAccount& account,
                                    const std::string& password,
                                    Transaction& tx,
                                    const std::string& rpc_url) {
    BroadcastResult result;

    const auto encoded = SignTransaction(account, password, tx);
    if (!encoded) {
        result.error = "signing failed: wrong password, tampered keystore, or invalid key";
        return result;
    }

    result.raw_signed_hex = "0x" + crypto::ToHex(*encoded);

    const RpcClient client(rpc_url);
    const auto sent = client.SendRawTransaction(result.raw_signed_hex);
    if (!sent.ok) {
        result.error = sent.error.message;
        return result;
    }

    result.ok = true;
    result.tx_hash = sent.value;
    return result;
}

bool PopulateTransactionFromNode(const std::string& rpc_url,
                                 const std::string& from_address,
                                 Transaction& tx) {
    const RpcClient client(rpc_url);

    // Every field is fetched before anything is written, so a partial RPC
    // failure leaves the caller's transaction untouched rather than half-filled.
    const auto chain_id = client.ChainId();
    if (!chain_id.ok) return false;

    const auto nonce = client.GetTransactionCount(from_address);
    if (!nonce.ok) return false;

    std::string gas_price;
    if (tx.type == TxType::LEGACY) {
        const auto price = client.GetGasPrice();
        if (!price.ok) return false;
        gas_price = price.value;
    }

    tx.chain_id = chain_id.value;
    tx.nonce = nonce.value;
    if (tx.type == TxType::LEGACY && tx.gas_price_wei.empty()) {
        tx.gas_price_wei = gas_price;
    }
    return true;
}

}  // namespace tigerwallet
