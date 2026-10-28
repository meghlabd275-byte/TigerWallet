// Tests for the account abstraction layer.
//
// These exercise the real code paths: key derivation and address computation,
// the CREATE2 account address, paymaster accounting, gas-token settlement,
// EIP-7702 authorization hashing, and the failure modes the earlier stubbed
// implementations could not reach (a node that rejects a call, a tampered
// ciphertext, an underfunded paymaster).

#include "account_abstraction.hpp"

#include <cstdio>
#include <string>

using namespace tiger;

static int failures = 0;

#define CHECK(condition, label)                                            \
    do {                                                                   \
        if (!(condition)) {                                                \
            std::printf("FAIL: %s\n", label);                              \
            ++failures;                                                    \
        } else {                                                           \
            std::printf("ok:   %s\n", label);                              \
        }                                                                  \
    } while (0)

namespace {

// A node stub that records what it was asked and returns a scripted answer.
struct FakeNode {
    bool next_result = true;
    Bytes next_output;
    int calls = 0;
    bool last_is_write = false;
    Address last_target;
    Bytes last_data;

    bool operator()(const Address& target, const Bytes& data, bool is_write, Bytes& out) {
        ++calls;
        last_target = target;
        last_data = data;
        last_is_write = is_write;
        out = next_output;
        return next_result;
    }
};

void test_address_derivation() {
    // Private key 1 has the generator point as its public key, so its address
    // is a published constant.
    const std::string one =
        "0000000000000000000000000000000000000000000000000000000000000001";
    Bytes secret;
    for (size_t i = 0; i < one.size(); i += 2) {
        secret.push_back(static_cast<uint8_t>(std::stoi(one.substr(i, 2), nullptr, 16)));
    }

    const auto point = tigerwallet::secp256k1::public_point(
        tigerwallet::secp256k1::from_bytes(secret.data(), secret.size()).get());
    const auto address = tigerwallet::secp256k1::ethereum_address(point.get());
    const auto checksum = tigerwallet::secp256k1::to_checksum_address(address);

    CHECK(checksum == "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf",
          "private key 1 yields the published address");
    CHECK(checksum.size() == 42 && checksum[0] == '0' && checksum[1] == 'x',
          "the address is 0x-prefixed and 20 bytes long");
}

void test_address_validation() {
    CHECK(CryptoUtils::decode_address("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf").has_value(),
          "a well-formed address decodes");
    CHECK(!CryptoUtils::decode_address("").has_value(), "the empty string is rejected");
    CHECK(!CryptoUtils::decode_address("0x1234").has_value(), "a short address is rejected");
    CHECK(!CryptoUtils::decode_address(
              "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdff").has_value(),
          "an over-long address is rejected");
    CHECK(!CryptoUtils::decode_address(
              "0x7E5F4552091A69125d5DfCb7b8C2659029395BdZ").has_value(),
          "a non-hex character is rejected");
    CHECK(!CryptoUtils::decode_address(
              "7E5F4552091A69125d5DfCb7b8C2659029395Bdf").has_value(),
          "a missing 0x prefix is rejected");
}

void test_aes_gcm() {
    Bytes key(32, 0x42);
    Bytes iv;
    Bytes ciphertext;

    const std::string message = "tigerwallet-account-abstraction";
    Bytes plaintext(message.begin(), message.end());

    CHECK(CryptoUtils::encryptAES(plaintext, key, ciphertext, iv), "encryption succeeds");
    CHECK(iv.size() == 12, "the IV is 12 bytes");
    CHECK(ciphertext.size() == plaintext.size() + 16,
          "the ciphertext carries a 16-byte tag");

    Bytes recovered;
    CHECK(CryptoUtils::decryptAES(ciphertext, key, iv, recovered), "decryption succeeds");
    CHECK(recovered == plaintext, "the plaintext round-trips");

    Bytes tampered = ciphertext;
    tampered[0] ^= 0x01;
    Bytes ignored;
    CHECK(!CryptoUtils::decryptAES(tampered, key, iv, ignored), "a tampered body is rejected");

    Bytes bad_tag = ciphertext;
    bad_tag[bad_tag.size() - 1] ^= 0x01;
    CHECK(!CryptoUtils::decryptAES(bad_tag, key, iv, ignored), "a tampered tag is rejected");

    Bytes wrong_key(32, 0x43);
    CHECK(!CryptoUtils::decryptAES(ciphertext, wrong_key, iv, ignored),
          "a wrong key is rejected");

    CHECK(!CryptoUtils::decryptAES(Bytes(8, 0), key, iv, ignored),
          "a truncated ciphertext is rejected");

    Bytes empty_out;
    Bytes empty_iv;
    Bytes empty_cipher;
    CHECK(CryptoUtils::encryptAES(Bytes{}, key, empty_cipher, empty_iv),
          "an empty plaintext encrypts");
    CHECK(CryptoUtils::decryptAES(empty_cipher, key, empty_iv, empty_out),
          "an empty plaintext decrypts");
    CHECK(empty_out.empty(), "an empty plaintext round-trips");
}

void test_eip712() {
    const auto domain = CryptoUtils::domainSeparator(
        "TigerWallet", "1", 1, "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf");
    const bytes32 struct_hash{};

    const auto digest = CryptoUtils::eip712Hash(domain, struct_hash);
    CHECK(digest.size() == 32, "the EIP-712 digest is 32 bytes");

    const auto other_domain = CryptoUtils::domainSeparator(
        "TigerWallet", "1", 137, "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf");
    CHECK(domain != other_domain, "the domain separator depends on the chain id");

    const auto other_digest = CryptoUtils::eip712Hash(other_domain, struct_hash);
    CHECK(digest != other_digest, "the digest depends on the domain separator");

    bytes32 other_struct{};
    other_struct[0] = 1;
    CHECK(CryptoUtils::eip712Hash(domain, other_struct) != digest,
          "the digest depends on the struct hash");

    bool threw = false;
    try {
        CryptoUtils::domainSeparator("TigerWallet", "1", 1, "0xnotanaddress");
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a malformed verifying contract is refused");
}

void test_sign_verify() {
    Bytes private_key, public_key;
    CHECK(CryptoUtils::generateKeyPair(private_key, public_key), "a key pair is generated");
    CHECK(private_key.size() == 32, "the private key is 32 bytes");
    CHECK(public_key.size() == 33, "the public key is compressed");

    const auto digest = tigerwallet::secp256k1::keccak256(
        reinterpret_cast<const uint8_t*>("hello"), 5);
    const Bytes message(digest.begin(), digest.end());

    Bytes signature;
    CHECK(CryptoUtils::sign(message, private_key, signature), "signing succeeds");
    CHECK(signature.size() == 64, "the signature is 64 bytes");
    CHECK(CryptoUtils::verify(message, signature, public_key), "the signature verifies");

    const auto other = tigerwallet::secp256k1::keccak256(
        reinterpret_cast<const uint8_t*>("hello!"), 6);
    CHECK(!CryptoUtils::verify(Bytes(other.begin(), other.end()), signature, public_key),
          "a different message does not verify");

    Bytes tampered = signature;
    tampered[10] ^= 0x01;
    CHECK(!CryptoUtils::verify(message, tampered, public_key),
          "a tampered signature does not verify");

    CHECK(!CryptoUtils::sign(Bytes(31, 0), private_key, signature), "a short digest is refused");
}

void test_create2_address() {
    AccountFactory factory;
    const Address factory_address = "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf";
    const Address implementation = "0x2B5AD5c4795c026514f8317c7a215E218DcCD6cF";
    const Address owner = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    factory.setFactoryAddress(factory_address);
    factory.setImplementationAddress(implementation);

    Bytes salt(32, 0);
    salt[31] = 1;

    const Address first = factory.getAccountAddress(owner, salt);
    const Address second = factory.getAccountAddress(owner, salt);
    CHECK(first == second, "the CREATE2 address is deterministic");
    CHECK(first.size() == 42, "the CREATE2 address is 20 bytes");

    Bytes other_salt(32, 0);
    other_salt[31] = 2;
    CHECK(factory.getAccountAddress(owner, other_salt) != first,
          "a different salt gives a different address");

    CHECK(factory.getAccountAddress(implementation, salt) != first,
          "a different owner gives a different address");

    bool threw = false;
    try {
        factory.getAccountAddress(owner, Bytes(31, 0));
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a salt of the wrong length is refused");

    threw = false;
    try {
        factory.getAccountAddress("0xbad", salt);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a malformed owner is refused");
}

void test_entry_point() {
    EntryPointContract entry("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789", 1);

    UserOperation op;
    op.sender = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    op.callGasLimit = uint256_t::fromUint64(100000);
    op.verificationGasLimit = uint256_t::fromUint64(100000);
    op.maxFeePerGas = uint256_t::fromUint64(2000000000);
    op.maxPriorityFeePerGas = uint256_t::fromUint64(1000000000);

    bool threw = false;
    try {
        entry.simulateHandleOp(op, op.sender, Bytes{});
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "simulation without a node is refused");

    threw = false;
    try {
        entry.executeUserOp(op);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "execution without a node is refused");

    FakeNode node;
    node.next_result = false;
    entry.setNodeCall([&node](const Address& target, const Bytes& data, bool is_write,
                              Bytes& out) { return node(target, data, is_write, out); });

    CHECK(!entry.simulateHandleOp(op, op.sender, Bytes{}),
          "a rejected simulation reports failure");
    CHECK(!node.last_is_write, "simulation is a read-only call");

    node.next_result = true;
    CHECK(entry.simulateHandleOp(op, op.sender, Bytes{}),
          "an accepted simulation reports success");
    CHECK(node.last_target == op.sender, "simulation targets the requested address");

    CHECK(entry.executeUserOp(op), "execution reports the node's answer");
    CHECK(node.last_is_write, "execution is a write call");

    UserOperation bad = op;
    bad.sender = "0xbad";
    CHECK(!entry.simulateHandleOp(bad, op.sender, Bytes{}),
          "an invalid sender fails validation");

    UserOperation bad_fee = op;
    bad_fee.maxFeePerGas = uint256_t::fromUint64(1);
    CHECK(!entry.simulateHandleOp(bad_fee, op.sender, Bytes{}),
          "a fee cap below the priority fee fails validation");
}

void test_deposit_info() {
    EntryPointContract entry("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789", 1);

    FakeNode node;
    node.next_output = Bytes(32, 0);
    node.next_output[31] = 0x01;
    entry.setNodeCall([&node](const Address& target, const Bytes& data, bool is_write,
                              Bytes& out) { return node(target, data, is_write, out); });

    const auto deposit = entry.getDepositInfo("0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69");
    CHECK(deposit == uint256_t::fromUint64(1), "the deposit is read from the node output");
    CHECK(node.last_data.size() == 36, "the call carries the selector and the address");
    CHECK(node.last_data[0] == 0x70 && node.last_data[1] == 0xa0 &&
          node.last_data[2] == 0x82 && node.last_data[3] == 0x31,
          "the selector is balances(address)");

    node.next_result = false;
    bool threw = false;
    try {
        entry.getDepositInfo("0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69");
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a failed deposit read throws");
}

void test_paymaster() {
    PaymasterManager paymasters;
    const Address paymaster = "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf";
    const Address token = "0xdAC17F958D2ee523a2206206994597C13D831ec7";
    paymasters.configurePaymaster(paymaster, token, uint256_t::fromUint64(100),
                                 uint256_t::fromUint64(0));

    UserOperation op;
    op.sender = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    op.paymaster = paymaster;
    op.callGasLimit = uint256_t::fromUint64(100000);
    op.maxFeePerGas = uint256_t::fromUint64(2000000000);

    CHECK(!paymasters.validatePaymasterUserOp(op, uint256_t::fromUint64(1000)),
          "an underfunded paymaster is rejected");

    paymasters.fundPaymasterDeposit(paymaster, uint256_t::fromUint64(10000));
    CHECK(paymasters.paymasterDeposit(paymaster) == uint256_t::fromUint64(10000),
          "the deposit reflects the funding");

    CHECK(paymasters.validatePaymasterUserOp(op, uint256_t::fromUint64(1000)),
          "a funded paymaster is accepted");

    CHECK(!paymasters.validatePaymasterUserOp(op, uint256_t::fromUint64(100000)),
          "a prefund beyond the deposit is rejected");

    bytes32 success{};
    success[31] = 1;
    const auto result = paymasters.postOp(0, op, success, uint256_t::fromUint64(500), Bytes{});
    CHECK(result.size() == 32, "postOp echoes the success flag");
    CHECK(paymasters.paymasterDeposit(paymaster) == uint256_t::fromUint64(9500),
          "postOp charges the paymaster");

    const auto records = paymasters.postOpRecords();
    CHECK(records.size() == 1, "the settlement is recorded");
    CHECK(records[0].actualGasCost == uint256_t::fromUint64(500),
          "the recorded cost is the actual one");

    bool threw = false;
    try {
        paymasters.postOp(0, op, success, uint256_t::fromUint64(100000), Bytes{});
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a cost beyond the deposit throws");
    CHECK(paymasters.paymasterDeposit(paymaster) == uint256_t::fromUint64(9500),
          "the deposit is unchanged after a refused settlement");

    UserOperation no_pm = op;
    no_pm.paymaster.clear();
    CHECK(!paymasters.validatePaymasterUserOp(no_pm, uint256_t::fromUint64(1)),
          "an operation with no paymaster is rejected");
}

void test_gas_abstraction() {
    GasAbstraction gas;
    const Address usdt = "0xdAC17F958D2ee523a2206206994597C13D831ec7";
    gas.addGasToken(usdt, true);

    bool threw = false;
    try {
        gas.calculateGasCost(usdt, uint256_t::fromUint64(21000),
                             uint256_t::fromUint64(2000000000));
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "valuation without an oracle is refused");

    gas.setPriceOracle([](const Address&) {
        return tigerwallet::math::U256::fromDecimal("1000000000000000000");
    });

    // 21000 gas at 2 gwei is 4.2e13 wei, so at one dollar per token the amount in
    // 1e18 scale is 4.2e13.
    const auto cost = gas.calculateGasCost(usdt, uint256_t::fromUint64(21000),
                                           uint256_t::fromUint64(2000000000));
    CHECK(cost == tigerwallet::math::U256::fromDecimal("42000000000000"),
          "the gas cost is computed exactly");

    UserOperation op;
    op.sender = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    op.callGasLimit = uint256_t::fromUint64(21000);
    op.maxFeePerGas = uint256_t::fromUint64(2000000000);

    CHECK(!gas.payWithGasToken(op, usdt, tigerwallet::math::U256::fromDecimal("41999999999999")),
          "an amount below the cost is refused");

    CHECK(gas.payWithGasToken(op, usdt, tigerwallet::math::U256::fromDecimal("42000000000000")),
          "the exact amount is accepted");
    CHECK(gas.settledAmount(op.sender, usdt) ==
              tigerwallet::math::U256::fromDecimal("42000000000000"),
          "the settlement is recorded");

    CHECK(!gas.payWithGasToken(op, "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69",
                               tigerwallet::math::U256::fromDecimal("1000000000000000000")),
          "an unregistered token is refused");

    threw = false;
    try {
        gas.addGasToken("0xbad", true);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a malformed gas token is refused");
}

void test_eip7702() {
    EIP7702Manager manager(1);
    const Address contract = "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf";

    const auto auth = manager.createAuthorization(contract, 7);
    CHECK(auth.chainId == 1, "the authorization carries the chain id");
    CHECK(auth.nonce == 7, "the authorization carries the account nonce");
    CHECK(auth.address == contract, "the authorization carries the contract");

    const auto hash = manager.authorizationHash(auth);
    CHECK(hash.size() == 32, "the authorization hash is 32 bytes");

    auto other = auth;
    other.nonce = 8;
    CHECK(manager.authorizationHash(other) != hash, "the hash depends on the nonce");

    other = auth;
    other.chainId = 137;
    CHECK(manager.authorizationHash(other) != hash, "the hash depends on the chain id");

    other = auth;
    other.address = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    CHECK(manager.authorizationHash(other) != hash, "the hash depends on the address");

    Bytes private_key, public_key;
    CryptoUtils::generateKeyPair(private_key, public_key);
    Bytes signature;
    CHECK(manager.signAuthorization(auth, private_key, signature), "signing succeeds");
    CHECK(manager.verifyAuthorization(auth, signature, public_key), "verification succeeds");

    auto tampered = auth;
    tampered.nonce = 99;
    CHECK(!manager.verifyAuthorization(tampered, signature, public_key),
          "a tampered authorization does not verify");

    bool threw = false;
    try {
        Bytes tx_hash;
        manager.executeAuthorization(auth, signature, Bytes(10, 1), tx_hash);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "submission without a node is refused");

    threw = false;
    try {
        manager.createAuthorization("0x0000000000000000000000000000000000000000", 1);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a zero contract address is refused");

    threw = false;
    try {
        manager.createAuthorization("0xbad", 1);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a malformed contract address is refused");
}

void test_batching() {
    BatchedTransactionManager batches;

    UserOperation first;
    first.sender = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    first.callGasLimit = uint256_t::fromUint64(21000);
    first.callValue = uint256_t::fromUint64(1000);

    UserOperation second = first;
    second.callValue = uint256_t::fromUint64(500);

    const auto id = batches.createBatch(first.sender, {first, second}, 60);
    CHECK(!id.empty() && id.rfind("0x", 0) == 0, "the batch id is 0x-prefixed");

    const auto batch = batches.getBatch(id);
    CHECK(batch.has_value(), "the batch is retrievable");
    CHECK(batch->totalGasLimit == uint256_t::fromUint64(42000), "the batch totals the gas");
    CHECK(batch->totalValue == uint256_t::fromUint64(1500),
          "the batch totals the value it carries");

    const auto other_id = batches.createBatch(first.sender, {first}, 60);
    CHECK(other_id != id, "batch ids are distinct");

    bool threw = false;
    try {
        batches.executeBatch(id);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "execution without a node is refused");

    FakeNode node;
    batches.setNodeExecute([&node](const Address& target, const Bytes& data, Bytes& out) {
        return node(target, data, true, out);
    });
    CHECK(batches.executeBatch(id), "the batch executes");
    CHECK(!batches.getBatch(id).has_value(), "an executed batch is removed");
    CHECK(node.calls == 2, "every operation in the batch is sent");

    const auto failing_id = batches.createBatch(first.sender, {first, second}, 60);
    node.next_result = false;
    CHECK(!batches.executeBatch(failing_id), "a rejected operation fails the batch");
    CHECK(!batches.getBatch(failing_id).has_value(), "a failed batch is removed");

    threw = false;
    try {
        batches.createBatch(first.sender, {}, 60);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "an empty batch is refused");

    threw = false;
    try {
        batches.createBatch("0xbad", {first}, 60);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a malformed sender is refused");
}

void test_manager_configuration() {
    AccountAbstractionManager manager(1);

    bool threw = false;
    try {
        manager.sendUserOperation(UserOperation{}, "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69");
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "sending before initialize() is refused");

    AccountAbstractionManager::Config config;
    config.entryPoint = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789";
    config.accountImplementation = "0x2B5AD5c4795c026514f8317c7a215E218DcCD6cF";
    config.factory = "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf";

    CHECK(manager.initialize(config), "initialize() succeeds with a real configuration");

    const Address owner = "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69";
    const auto address = manager.factory().getAccountAddress(owner, Bytes(32, 1));
    CHECK(address.size() == 42, "the factory derives an account address");

    AccountAbstractionManager other(1);
    AccountAbstractionManager::Config bad = config;
    bad.factory = "0x0000000000000000000000000000000000000000";
    threw = false;
    try {
        other.initialize(bad);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a zero factory address is refused");
}

}  // namespace

int main() {
    test_address_derivation();
    test_address_validation();
    test_aes_gcm();
    test_eip712();
    test_sign_verify();
    test_create2_address();
    test_entry_point();
    test_deposit_info();
    test_paymaster();
    test_gas_abstraction();
    test_eip7702();
    test_batching();
    test_manager_configuration();

    if (failures == 0) {
        std::printf("\nAll account abstraction tests passed.\n");
        return 0;
    }
    std::printf("\n%d account abstraction test(s) failed.\n", failures);
    return 1;
}
