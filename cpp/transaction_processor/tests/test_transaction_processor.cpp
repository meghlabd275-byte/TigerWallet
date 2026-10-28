// Tests for address validation and the transaction processor.
//
// The address vectors are real. The Base58Check and Bech32 values were produced
// by reference implementations of those algorithms, so the tests check against
// the specifications rather than against this code's own output.
//
// The processor tests exercise the two bugs that made it unusable: the queue
// never drained, and the validation accepted malformed addresses.

#include "address.hpp"
#include "transaction_processor.hpp"

#include <cassert>
#include <cstdio>
#include <string>
#include <thread>
#include <vector>

using namespace tigerwallet;

namespace {

// Addresses generated from the reference algorithms.
const char* kBitcoinP2pkh = "1111111111111111111114oLvT2";
const char* kBitcoinP2sh = "31h1vYVSYuKP6AhS86fbRdMw9XHieotbST";
const char* kBitcoinSegwitV0 = "bc1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq9e75rs";
const char* kBitcoinSegwitV1 = "bc1pqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqqenm";
const char* kLitecoin = "LKDxGDJq5fF4FohAB8zJH24mDDNHDNtqsE";
const char* kDogecoin = "D596YFweJQuHY1BbjazZYmAbt8jJPbKehC";
const char* kTron = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb";
const char* kSolanaZero = "11111111111111111111111111111111";
const char* kCosmosZero = "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a";
const char* kOsmoZero = "osmo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqmcn030";

void test_base58() {
    // Round trip over byte sequences that include leading zeros.
    const std::vector<std::vector<uint8_t>> cases = {
        {0x00},
        {0x00, 0x00, 0x01},
        {0xff, 0xfe, 0xfd},
        {0x00, 0x01, 0x02, 0x03, 0xff},
        std::vector<uint8_t>(32, 0x00),
        std::vector<uint8_t>(32, 0xff),
    };
    for (const auto& input : cases) {
        const std::string encoded = address::base58_encode(input);
        const auto decoded = address::base58_decode(encoded);
        assert(decoded.has_value());
        assert(*decoded == input);
    }

    // 32 zero bytes encode to 32 '1' characters.
    assert(address::base58_encode(std::vector<uint8_t>(32, 0)) ==
           "11111111111111111111111111111111");

    // Characters outside the alphabet are rejected. 0, O, I and l are not in it.
    assert(!address::base58_decode("0OIl").has_value());
    assert(!address::base58_decode("").has_value());
    assert(!address::base58_decode("l1").has_value());
}

void test_base58check() {
    // A real P2PKH address decodes to version 0x00 and 20 bytes.
    const auto p2pkh = address::base58check_decode(kBitcoinP2pkh);
    assert(p2pkh.has_value());
    assert(p2pkh->size() == 21);
    assert((*p2pkh)[0] == 0x00);

    const auto p2sh = address::base58check_decode(kBitcoinP2sh);
    assert(p2sh.has_value());
    assert((*p2sh)[0] == 0x05);

    // Changing one character breaks the checksum, which is the whole point of
    // Base58Check. A validation that only checks the length would accept this.
    std::string corrupted = kBitcoinP2pkh;
    corrupted[10] = (corrupted[10] == 'a') ? 'b' : 'a';
    assert(!address::base58check_decode(corrupted).has_value());

    // A truncated address fails.
    assert(!address::base58check_decode("1111").has_value());
}

void test_evm() {
    // All lower case carries no checksum, which is how clients treat it.
    assert(address::is_valid_evm_address("0x0000000000000000000000000000000000000000"));
    assert(address::is_valid_evm_address("0xde709f2102306220921060314715629080e2fb77"));

    // A real EIP-55 checksummed address.
    assert(address::is_valid_evm_address("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"));
    assert(address::is_valid_evm_address("0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359"));
    assert(address::is_valid_evm_address("0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB"));
    assert(address::is_valid_evm_address("0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb"));

    // The same address with the case broken fails the checksum.
    assert(!address::is_valid_evm_address("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAeD"));
    assert(!address::is_valid_evm_address("0x5AAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"));

    // The old check was a length test, so these all passed it.
    assert(!address::is_valid_evm_address("0xZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"));
    assert(!address::is_valid_evm_address("0xgggggggggggggggggggggggggggggggggggggggg"));
    assert(!address::is_valid_evm_address("0x00000000000000000000000000000000000000  "));
    assert(!address::is_valid_evm_address("0000000000000000000000000000000000000000"));
    assert(!address::is_valid_evm_address("0x000000000000000000000000000000000000000"));
    assert(!address::is_valid_evm_address(""));
}

void test_bitcoin() {
    assert(address::is_valid_bitcoin_address(kBitcoinP2pkh));
    assert(address::is_valid_bitcoin_address(kBitcoinP2sh));
    assert(address::is_valid_bitcoin_address(kBitcoinSegwitV0));
    assert(address::is_valid_bitcoin_address(kBitcoinSegwitV1));

    // An Ethereum address is not a Bitcoin address, though it is within the
    // 26-to-62 character window the old check used.
    assert(!address::is_valid_bitcoin_address("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"));
    // A Litecoin address is not a Bitcoin address.
    assert(!address::is_valid_bitcoin_address(kLitecoin));
    // A corrupted checksum fails.
    assert(!address::is_valid_bitcoin_address("1111111111111111111114oLvT3"));
    // A Bech32 address with the wrong prefix fails.
    assert(!address::is_valid_bitcoin_address(kCosmosZero));
}

void test_other_chains() {
    assert(address::is_valid_litecoin_address(kLitecoin));
    assert(address::is_valid_litecoin_address("ltc1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp9ysmq"));
    assert(!address::is_valid_litecoin_address(kBitcoinP2pkh));

    assert(address::is_valid_dogecoin_address(kDogecoin));
    assert(!address::is_valid_dogecoin_address(kBitcoinP2pkh));

    assert(address::is_valid_tron_address(kTron));
    assert(!address::is_valid_tron_address(kBitcoinP2pkh));

    assert(address::is_valid_solana_address(kSolanaZero));
    // A real 32-byte key.
    assert(address::is_valid_solana_address("11111111111111111111111111111112"));
    // 44 characters that decode to more than 32 bytes fail.
    assert(!address::is_valid_solana_address("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"));
    // The old check accepted any 32-to-44 character string.
    assert(!address::is_valid_solana_address("not_a_valid_solana_address_at_all_here"));
    assert(!address::is_valid_solana_address(""));

    assert(address::is_valid_cosmos_address(kCosmosZero, "cosmos"));
    assert(address::is_valid_cosmos_address(kOsmoZero, "osmo"));
    // The prefix must match the chain.
    assert(!address::is_valid_cosmos_address(kOsmoZero, "cosmos"));
    assert(!address::is_valid_cosmos_address(kBitcoinSegwitV0, "cosmos"));
}

void test_processor() {
    TransactionProcessorConfig config;
    config.worker_threads = 4;
    config.enable_gas_optimization = false;  // No gas price is set.

    TransactionProcessor processor(config);
    processor.start();

    // A valid EVM transaction reaches the validated pool. This is the path that
    // never ran before: submit pushed to the pending pool, the worker waited on
    // an empty priority queue, and nothing was processed.
    auto tx = std::make_shared<Transaction>();
    tx->type = TransactionType::EVM;
    tx->from = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed";
    tx->to = "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359";
    tx->value = "1000000000000000000";
    tx->gas_price = "20000000000";
    tx->gas_limit = "21000";
    tx->nonce = "1";
    tx->chain_id = 1;
    tx->raw_transaction = "0x02f86c0180843b9aca00";

    const auto submitted = processor.submit_transaction(tx);
    assert(submitted.has_value());

    // The hash is derived from the signed bytes, not left as the caller set it.
    assert(submitted->to_hex().size() == 64);
    assert(submitted->to_hex() != std::string(64, '0'));

    // Wait for a worker to pick it up.
    for (int i = 0; i < 200; ++i) {
        if (processor.get_stats().total_validated >= 1) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    const auto stats = processor.get_stats();
    assert(stats.total_validated == 1);
    assert(stats.total_failed == 0);
    assert(stats.indexed_count == 1);

    // Retrieval by hash works, which the old implementation could not do.
    const auto found = processor.get_transaction(*submitted);
    assert(found != nullptr);
    assert(found->status == TransactionStatus::VALIDATED);

    // Confirmation updates the counter that previously always read zero.
    assert(processor.mark_confirmed(*submitted));
    assert(processor.get_stats().total_confirmed == 1);

    // The same raw transaction submitted twice is a duplicate.
    auto duplicate = std::make_shared<Transaction>();
    duplicate->type = TransactionType::EVM;
    duplicate->from = tx->from;
    duplicate->to = tx->to;
    duplicate->value = tx->value;
    duplicate->gas_price = tx->gas_price;
    duplicate->gas_limit = tx->gas_limit;
    duplicate->nonce = tx->nonce;
    duplicate->chain_id = 1;
    duplicate->raw_transaction = tx->raw_transaction;
    assert(!processor.submit_transaction(duplicate).has_value());
    assert(processor.get_stats().total_duplicates == 1);

    // A malformed EVM address is rejected. The old check was a length test, so
    // this passed it.
    auto bad = std::make_shared<Transaction>();
    bad->type = TransactionType::EVM;
    bad->from = "0xZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ";
    bad->to = tx->to;
    bad->value = "1";
    bad->gas_price = "1";
    bad->gas_limit = "21000";
    bad->nonce = "2";
    bad->chain_id = 1;
    bad->raw_transaction = "0x02f86c0180843b9aca01";
    const auto bad_hash = processor.submit_transaction(bad);
    assert(bad_hash.has_value());
    for (int i = 0; i < 200; ++i) {
        if (processor.get_stats().total_failed >= 1) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    assert(processor.get_stats().total_failed == 1);
    const auto rejected = processor.get_transaction(*bad_hash);
    assert(rejected != nullptr);
    assert(rejected->status == TransactionStatus::FAILED);
    assert(rejected->error_message.find("address") != std::string::npos);

    // A Solana address that is the right length but not Base58 is rejected.
    auto bad_solana = std::make_shared<Transaction>();
    bad_solana->type = TransactionType::SOLANA;
    bad_solana->from = "not_a_valid_solana_address_at_all_here";
    bad_solana->to = kSolanaZero;
    bad_solana->value = "1";
    bad_solana->gas_price = "1";
    bad_solana->gas_limit = "21000";
    bad_solana->nonce = "1";
    bad_solana->chain_id = 1;
    bad_solana->raw_transaction = "0x02f86c0180843b9aca02";
    const auto solana_hash = processor.submit_transaction(bad_solana);
    assert(solana_hash.has_value());
    for (int i = 0; i < 200; ++i) {
        if (processor.get_stats().total_failed >= 2) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    assert(processor.get_stats().total_failed == 2);

    // A real Solana transaction is accepted.
    auto good_solana = std::make_shared<Transaction>();
    good_solana->type = TransactionType::SOLANA;
    good_solana->from = kSolanaZero;
    good_solana->to = "11111111111111111111111111111112";
    good_solana->value = "1";
    good_solana->gas_price = "5000";
    good_solana->gas_limit = "21000";
    good_solana->nonce = "1";
    good_solana->chain_id = 1;
    good_solana->raw_transaction = "0x02f86c0180843b9aca03";
    assert(processor.submit_transaction(good_solana).has_value());
    for (int i = 0; i < 200; ++i) {
        if (processor.get_stats().total_validated >= 2) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    assert(processor.get_stats().total_validated >= 2);

    // A Cosmos transaction whose chain has no prefix is rejected rather than
    // checked against a guess.
    auto bad_cosmos = std::make_shared<Transaction>();
    bad_cosmos->type = TransactionType::COSMOS;
    bad_cosmos->from = kCosmosZero;
    bad_cosmos->to = kCosmosZero;
    bad_cosmos->value = "1";
    bad_cosmos->gas_price = "1";
    bad_cosmos->gas_limit = "21000";
    bad_cosmos->nonce = "1";
    bad_cosmos->chain_id = 99999;  // Not in the prefix table.
    bad_cosmos->raw_transaction = "0x02f86c0180843b9aca04";
    assert(processor.submit_transaction(bad_cosmos).has_value());
    for (int i = 0; i < 200; ++i) {
        if (processor.get_stats().total_failed >= 3) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    assert(processor.get_stats().total_failed == 3);

    // A gas price is only reported when one was set.
    assert(processor.optimal_gas_price(1).empty());
    processor.set_gas_price(1, "30000000000");
    assert(processor.optimal_gas_price(1) == "30000000000");
    // A chain with nothing set still reports nothing.
    assert(processor.optimal_gas_price(137).empty());

    processor.stop();
}

void test_replacement() {
    TransactionProcessorConfig config;
    config.worker_threads = 1;
    config.enable_gas_optimization = false;

    TransactionProcessor processor(config);

    auto original = std::make_shared<Transaction>();
    original->type = TransactionType::EVM;
    original->from = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed";
    original->to = "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359";
    original->value = "1";
    original->gas_price = "20000000000";
    original->gas_limit = "21000";
    original->nonce = "5";
    original->chain_id = 1;
    original->raw_transaction = "0x02f86c0180843b9aca10";
    const auto first = processor.submit_transaction(original);
    assert(first.has_value());

    // A replacement must pay more.
    auto cheaper = std::make_shared<Transaction>();
    cheaper->type = TransactionType::EVM;
    cheaper->from = original->from;
    cheaper->to = original->to;
    cheaper->value = "1";
    cheaper->gas_price = "10000000000";
    cheaper->gas_limit = "21000";
    cheaper->nonce = "5";
    cheaper->chain_id = 1;
    cheaper->raw_transaction = "0x02f86c0180843b9aca11";
    assert(!processor.replace_transaction(*first, cheaper));

    // The same nonce at a higher price is accepted.
    auto dearer = std::make_shared<Transaction>();
    dearer->type = TransactionType::EVM;
    dearer->from = original->from;
    dearer->to = original->to;
    dearer->value = "1";
    dearer->gas_price = "40000000000";
    dearer->gas_limit = "21000";
    dearer->nonce = "5";
    dearer->chain_id = 1;
    dearer->raw_transaction = "0x02f86c0180843b9aca12";
    assert(processor.replace_transaction(*first, dearer));

    // A different nonce is a second transaction, not a replacement.
    auto other_nonce = std::make_shared<Transaction>();
    other_nonce->type = TransactionType::EVM;
    other_nonce->from = original->from;
    other_nonce->to = original->to;
    other_nonce->value = "1";
    other_nonce->gas_price = "90000000000";
    other_nonce->gas_limit = "21000";
    other_nonce->nonce = "6";
    other_nonce->chain_id = 1;
    other_nonce->raw_transaction = "0x02f86c0180843b9aca13";
    assert(!processor.replace_transaction(*first, other_nonce));

    // A transaction with no hash material is rejected.
    auto empty = std::make_shared<Transaction>();
    empty->type = TransactionType::EVM;
    empty->from = original->from;
    empty->to = original->to;
    assert(!processor.submit_transaction(empty).has_value());
    assert(!processor.submit_transaction(nullptr).has_value());
}

}  // namespace

int main() {
    test_base58();
    test_base58check();
    test_evm();
    test_bitcoin();
    test_other_chains();
    test_processor();
    test_replacement();

    std::printf("transaction_processor: all checks passed\n");
    return 0;
}
