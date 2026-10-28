// Tests for the transaction simulator.
//
// The simulator previously produced its transaction hash by concatenating
// eight hex characters of the sender address with eight of the recipient, and
// accepted any 64-byte string as a valid signature. Both are covered here.

#include "transaction_simulator.h"

#include <cassert>
#include <cstdint>
#include <cstdio>
#include <string>

using namespace TigerWallet;

namespace {

Transaction MakeTransaction() {
    Transaction tx;
    tx.from = Address::fromString("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E");
    tx.to = Address::fromString("0xdAC17F958D2ee523a2206206994597C13D831ec7");
    tx.value = {1000000000000000000ULL, 0, 0, 0};
    tx.gasLimit = 21000;
    tx.gasPrice = 20000000000ULL;
    tx.nonce = 0;
    tx.chainId = 1;
    return tx;
}

// The secp256k1 group order.
const uint8_t kOrder[32] = {
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
    0xba, 0xae, 0xdc, 0xe6, 0xaf, 0x48, 0xa0, 0x3b,
    0xbf, 0xd2, 0x5e, 0x8c, 0xd0, 0x36, 0x41, 0x41
};

}  // namespace

int main() {
    TransactionSimulator simulator(1);

    // --- Transaction hashing -------------------------------------------------

    const Transaction tx = MakeTransaction();
    const std::string hash = simulator.computeTransactionHash(tx);

    // A transaction hash is 32 bytes, so 66 characters including the prefix.
    assert(hash.size() == 66);
    assert(hash.rfind("0x", 0) == 0);

    for (size_t i = 2; i < hash.size(); ++i) {
        const char c = hash[i];
        const bool is_hex = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
        assert(is_hex);
    }

    // The old implementation produced from.substr(2,8) + to.substr(2,8).
    const std::string old_style =
        "0x" + tx.from.toString().substr(2, 8) + tx.to.toString().substr(2, 8);
    assert(hash != old_style);
    assert(hash.size() > old_style.size());

    // A different nonce is a different transaction, so it hashes differently.
    // Under the old implementation every transaction between this pair hashed
    // to the same value regardless of any field.
    Transaction tx_nonce = tx;
    tx_nonce.nonce = 1;
    assert(simulator.computeTransactionHash(tx_nonce) != hash);

    Transaction tx_value = tx;
    tx_value.value = {2000000000000000000ULL, 0, 0, 0};
    assert(simulator.computeTransactionHash(tx_value) != hash);

    Transaction tx_gas = tx;
    tx_gas.gasPrice = 30000000000ULL;
    assert(simulator.computeTransactionHash(tx_gas) != hash);

    Transaction tx_data = tx;
    tx_data.data = {0xa9, 0x05, 0x9c, 0xbb};
    assert(simulator.computeTransactionHash(tx_data) != hash);

    // Identical transactions do hash the same, which is correct.
    assert(simulator.computeTransactionHash(MakeTransaction()) == hash);

    // For an unsigned transaction the signing hash and the identity hash
    // coincide, because the signature is the only difference between them.
    const std::string signing = simulator.signingHash(tx);
    assert(signing.size() == 66);
    assert(signing == hash);

    // Once a signature is present the two differ, because the identity hash
    // covers r, s and v while the signing hash does not.
    Transaction signed_tx = tx;
    signed_tx.signature.resize(65, 0x00);
    signed_tx.signature[31] = 0x01;
    signed_tx.signature[63] = 0x01;
    signed_tx.signature[64] = 27;
    assert(simulator.signingHash(signed_tx) == signing);
    assert(simulator.computeTransactionHash(signed_tx) != hash);

    // --- Signature validation ------------------------------------------------

    // The old check was `signature.size() >= 64`, which accepted 64 zero bytes.
    // checkSignature is private, so this goes through the public path: a
    // transaction carrying an invalid signature must not be validated.
    Transaction short_sig = MakeTransaction();
    short_sig.signature.assign(64, 0x00);
    assert(!simulator.simulateTransaction(short_sig).success);

    // A 65-byte all-zero signature has zero r and s, which is not a signature.
    Transaction zero_sig = MakeTransaction();
    zero_sig.signature.assign(65, 0x00);
    assert(!simulator.simulateTransaction(zero_sig).success);

    // r and s equal to the group order are out of range.
    Transaction order_sig = MakeTransaction();
    order_sig.signature.resize(65, 0x00);
    for (int i = 0; i < 32; ++i) {
        order_sig.signature[i] = kOrder[i];       // r = n
        order_sig.signature[32 + i] = kOrder[i];  // s = n
    }
    order_sig.signature[64] = 27;
    assert(!simulator.simulateTransaction(order_sig).success);

    // A v below 27 is not a recovery id.
    Transaction bad_v = MakeTransaction();
    bad_v.signature.resize(65, 0x00);
    bad_v.signature[31] = 0x01;
    bad_v.signature[63] = 0x01;
    bad_v.signature[64] = 5;
    assert(!simulator.simulateTransaction(bad_v).success);

    // A well-formed signature is accepted.
    Transaction good = MakeTransaction();
    good.signature.resize(65, 0x00);
    good.signature[31] = 0x01;  // r = 1
    good.signature[63] = 0x01;  // s = 1
    good.signature[64] = 27;    // v
    assert(simulator.simulateTransaction(good).success);

    // A transaction with no signature is still simulatable, which is what a
    // user does before signing.
    assert(simulator.simulateTransaction(MakeTransaction()).success);

    // --- Swap quoting --------------------------------------------------------

    // Reserves are not available, so no quote is produced and the caller is
    // told why, rather than receiving a number derived from invented reserves.
    const Address router =
        Address::fromString("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D");
    const uint256_t amount_in = {1000000000000000000ULL, 0, 0, 0};
    const SwapSimulation swap = simulator.simulateSwap(
        router,
        "0x0000000000000000000000000000000000000000",
        "0xdAC17F958D2ee523a2206206994597C13D831ec7",
        amount_in, 0.5);

    assert(!swap.quoteAvailable);
    assert(!swap.quoteUnavailableReason.empty());
    assert(swap.expectedAmountOut[0] == 0);
    assert(swap.minimumAmountOut[0] == 0);
    assert(swap.priceImpact == 0.0);

    std::printf("transaction_simulator: all checks passed\n");
    return 0;
}
