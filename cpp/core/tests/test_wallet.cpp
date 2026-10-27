// Tests for wallet key handling, the transaction pool and the chain registry.
//
// The emphasis is on the paths that must fail closed: an invalid private key, a
// wrong password, a tampered keystore, a keystore whose key does not match the
// account address, and a broadcast with no reachable node.

#include <array>
#include <cstdint>
#include <iostream>
#include <string>
#include <vector>

#include "chain.hpp"
#include "crypto.hpp"
#include "mempool.hpp"
#include "wallet.hpp"

using namespace tigerwallet;

namespace {

int failures = 0;

void Check(const std::string& label, bool ok) {
    if (!ok) ++failures;
    std::cout << (ok ? "  PASS " : "  FAIL ") << label << "\n";
}

std::array<uint8_t, 32> KeyWithLowByte(uint8_t value) {
    std::array<uint8_t, 32> key{};
    key[31] = value;
    return key;
}

Transaction SampleTransaction() {
    Transaction tx;
    tx.type = TxType::LEGACY;
    tx.chain_id = 1;
    tx.nonce = 0;
    tx.gas_price_wei = "1000000000";
    tx.gas_limit = 21000;
    tx.to = "0x3535353535353535353535353535353535353535";
    tx.value_wei = "1000000000000000000";
    return tx;
}

void TestWalletCreation() {
    std::cout << "wallet creation\n";
    const auto account = Wallet::FromPrivateKey(KeyWithLowByte(1), "pw123", "main");
    Check("an account is produced", account.has_value());
    Check("the address is the known value for key 0x01",
          account && account->address == "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf");
    Check("the public key is 64 bytes", account && account->public_key.size() == 64);
    Check("the label is kept", account && account->label == "main");

    // The stored secret must be ciphertext, not the raw key.
    const auto key = KeyWithLowByte(1);
    const bool stores_raw_key =
        account && account->encrypted_secret.size() >= 32 &&
        std::vector<uint8_t>(account->encrypted_secret.begin(),
                             account->encrypted_secret.begin() + 32) ==
            std::vector<uint8_t>(key.begin(), key.end());
    Check("the private key is not stored in the clear", account && !stores_raw_key);

    std::string address;
    Check("Address() returns the account address",
          account && Wallet::Address(*account, address) && address == account->address);
}

void TestInvalidKeys() {
    std::cout << "invalid private keys\n";
    const std::array<uint8_t, 32> zero{};
    Check("all-zero key is rejected", !Wallet::FromPrivateKey(zero, "pw").has_value());

    // The secp256k1 group order is not a valid private key.
    std::array<uint8_t, 32> order{};
    const auto order_bytes =
        crypto::FromHex("0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141");
    std::copy(order_bytes.begin(), order_bytes.end(), order.begin());
    Check("a key equal to the curve order is rejected",
          !Wallet::FromPrivateKey(order, "pw").has_value());
}

void TestSigning() {
    std::cout << "signing through the wallet\n";
    const auto account = Wallet::FromPrivateKey(KeyWithLowByte(1), "pw123");
    Check("an account is available for signing", account.has_value());
    if (!account) return;

    auto tx = SampleTransaction();
    Check("signs with the correct password", Wallet::SignTransaction(*account, "pw123", tx).has_value());

    auto wrong_tx = SampleTransaction();
    Check("refuses to sign with the wrong password",
          !Wallet::SignTransaction(*account, "not-the-password", wrong_tx).has_value());

    auto tampered = *account;
    tampered.encrypted_secret[20] ^= 0x01;
    auto tampered_tx = SampleTransaction();
    Check("refuses to sign from a tampered keystore",
          !Wallet::SignTransaction(tampered, "pw123", tampered_tx).has_value());

    // A keystore holding one key but labelled with another account's address
    // must not sign: otherwise the wrong key would be used silently.
    const auto other = Wallet::FromPrivateKey(KeyWithLowByte(2), "pw123");
    Check("a second account is available", other.has_value());
    if (other) {
        WalletAccount mismatched = *other;
        mismatched.address = account->address;
        auto mismatched_tx = SampleTransaction();
        Check("refuses to sign when the address and key disagree",
              !Wallet::SignTransaction(mismatched, "pw123", mismatched_tx).has_value());
    }
}

void TestBroadcastFailsClosed() {
    std::cout << "broadcast without a node\n";
    const auto account = Wallet::FromPrivateKey(KeyWithLowByte(1), "pw123");
    if (!account) {
        Check("an account is available", false);
        return;
    }

    auto tx = SampleTransaction();
    // Port 1 is not a JSON-RPC endpoint, so the transport must fail.
    const auto result = Wallet::SignAndSend(*account, "pw123", tx, "http://127.0.0.1:1");
    Check("the broadcast reports failure", !result.ok);
    Check("an error message is provided", !result.error.empty());
    Check("no transaction hash is invented", result.tx_hash.empty());
    Check("the signed bytes are still produced for inspection",
          result.raw_signed_hex.rfind("0x", 0) == 0 && result.raw_signed_hex.size() > 2);

    Transaction empty;
    Check("PopulateTransactionFromNode fails against a dead endpoint",
          !PopulateTransactionFromNode("http://127.0.0.1:1", account->address, empty));
    Check("a failed populate leaves the transaction untouched", empty.chain_id == 1);
}

void TestMempool() {
    std::cout << "transaction pool\n";
    TransactionPool pool;

    PooledTransaction first;
    first.hash = "0xaa";
    first.from = "0x1111111111111111111111111111111111111111";
    first.nonce = 1;
    first.gas_price_wei = "100";

    Check("a new transaction is accepted", pool.Add(first) == PoolResult::ACCEPTED);
    Check("the same hash is rejected", pool.Add(first) == PoolResult::REJECTED_DUPLICATE);

    PooledTransaction same_price = first;
    same_price.hash = "0xbb";
    Check("an equal price at the same nonce does not replace",
          pool.Add(same_price) == PoolResult::REJECTED_UNDERPRICED);

    PooledTransaction lower_price = first;
    lower_price.hash = "0xcc";
    lower_price.gas_price_wei = "99";
    Check("a lower price at the same nonce does not replace",
          pool.Add(lower_price) == PoolResult::REJECTED_UNDERPRICED);

    PooledTransaction higher_price = first;
    higher_price.hash = "0xdd";
    higher_price.gas_price_wei = "101";
    Check("a higher price at the same nonce replaces",
          pool.Add(higher_price) == PoolResult::REPLACED);
    Check("the replaced transaction is gone",
          !pool.Get("0xaa").has_value() && pool.Get("0xdd").has_value());
    Check("only one entry remains for the sender", pool.ForSender(first.from).size() == 1);

    Check("confirming removes the transaction", pool.Confirm("0xdd") && pool.Size() == 0);
    Check("confirming an unknown hash returns false", !pool.Confirm("0xdead"));

    PooledTransaction stale = first;
    stale.hash = "0xee";
    stale.nonce = 5;
    stale.added_at = 100;
    pool.Add(stale);
    Check("stale entries are pruned", pool.PruneOlderThan(60, 1000) == 1 && pool.Size() == 0);

    PooledTransaction fresh = first;
    fresh.hash = "0xff";
    fresh.nonce = 7;
    fresh.added_at = 990;
    pool.Add(fresh);
    Check("fresh entries survive pruning", pool.PruneOlderThan(60, 1000) == 0 && pool.Size() == 1);

    Check("decimal comparison handles differing lengths",
          DecimalGreater("1000000000000000000", "999999999999999999"));
    Check("decimal comparison handles leading zeros",
          DecimalGreater("100", "099") && !DecimalGreater("099", "100"));
    Check("decimal comparison of equal magnitudes",
          !DecimalGreater("1000", "1000"));
}

void TestChainRegistry() {
    std::cout << "chain registry\n";
    ChainRegistry registry;

    const auto ethereum = registry.Get(1);
    Check("Ethereum is present", ethereum.has_value());
    Check("Ethereum has the right symbol and id",
          ethereum && ethereum->native_symbol == "ETH" && ethereum->chain_id == 1);
    Check("no RPC endpoint is bundled", ethereum && ethereum->rpc_url.empty());
    Check("a chain with no endpoint is not reported as configured",
          registry.Configured().empty());
    Check("an unknown chain returns nothing", !registry.Get(999999).has_value());

    Check("an endpoint can be set", registry.SetRpcUrl(1, "https://node.example"));
    Check("setting an endpoint on an unknown chain fails",
          !registry.SetRpcUrl(999999, "https://node.example"));
    Check("the configured list now has one entry", registry.Configured().size() == 1);

    Check("explorer URL for a transaction",
          registry.ExplorerTxUrl(1, "0xabc") == "https://etherscan.io/tx/0xabc");
    Check("explorer URL for an address",
          registry.ExplorerAddressUrl(1, "0xabc") == "https://etherscan.io/address/0xabc");
    Check("no explorer URL for an unknown chain",
          registry.ExplorerTxUrl(999999, "0xabc").empty());

    Check("a chain can be disabled", registry.SetEnabled(1, false) && !registry.Get(1)->enabled);
    Check("disabling an unknown chain fails", !registry.SetEnabled(999999, false));

    ChainConfig custom;
    custom.chain_id = 31337;
    custom.name = "Local";
    custom.native_symbol = "ETH";
    custom.explorer_url = "";
    custom.rpc_url = "http://127.0.0.1:8545";
    registry.Set(custom);
    Check("a custom chain can be added", registry.IsSupported(31337));

    const auto builtin = BuiltinChains();
    Check("the built-in list is not empty", builtin.size() >= 30);
    bool all_have_ids = true;
    for (const auto& chain : builtin) {
        if (chain.chain_id == 0 || chain.native_symbol.empty()) all_have_ids = false;
    }
    Check("every built-in chain has an id and a symbol", all_have_ids);

    // No built-in chain may ship an endpoint: the operator supplies those.
    bool none_have_endpoints = true;
    for (const auto& chain : builtin) {
        if (!chain.rpc_url.empty()) none_have_endpoints = false;
    }
    Check("no built-in chain bundles an RPC endpoint", none_have_endpoints);
}

void TestSecureZero() {
    std::cout << "secure wipe\n";
    uint8_t buffer[32];
    for (size_t i = 0; i < sizeof(buffer); ++i) buffer[i] = 0xAB;
    SecureZero(buffer, sizeof(buffer));
    bool all_zero = true;
    for (uint8_t byte : buffer) {
        if (byte != 0) all_zero = false;
    }
    Check("SecureZero clears the buffer", all_zero);
    SecureZero(nullptr, 0);  // must not crash
    Check("SecureZero tolerates a null pointer", true);
}

}  // namespace

int main() {
    TestWalletCreation();
    TestInvalidKeys();
    TestSigning();
    TestBroadcastFailsClosed();
    TestMempool();
    TestChainRegistry();
    TestSecureZero();

    std::cout << (failures == 0 ? "wallet: all tests passed\n"
                                : "wallet: " + std::to_string(failures) + " test(s) failed\n");
    return failures == 0 ? 0 : 1;
}
