// Tests for the hardware-wallet primitives.
//
// The expected values come from published Bitcoin vectors and from the hash
// specification, not from this implementation, so a change that breaks the
// encoding or the path handling is caught rather than baked in.

#include "../include/hardware_wallet.h"
#include "../include/json_rpc.h"

#include <cstdio>
#include <sstream>
#include <string>
#include <vector>

namespace {

int failures = 0;
int checks = 0;

void check(bool condition, const std::string& label) {
    checks++;
    if (!condition) {
        failures++;
        std::printf("FAIL  %s\n", label.c_str());
    }
}

void check_equal(const std::string& actual, const std::string& expected, const std::string& label) {
    checks++;
    if (actual != expected) {
        failures++;
        std::printf("FAIL  %s\n      expected %s\n      actual   %s\n",
                    label.c_str(), expected.c_str(), actual.c_str());
    }
}

using namespace tigerwallet::hardware;
using tigerwallet::json::Value;

/// SHA-256 against the specification.
void test_sha256() {
    auto empty_digest = CryptoUtils::sha256(nullptr, 0);
    check_equal(CryptoUtils::hex_encode(empty_digest.data(), empty_digest.size()),
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
                "sha256 of an empty input");

    const std::vector<uint8_t> abc = {'a', 'b', 'c'};
    auto abc_digest = CryptoUtils::sha256(abc);
    check_equal(CryptoUtils::hex_encode(abc_digest.data(), abc_digest.size()),
                "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
                "sha256 of \"abc\"");
}

/// Base58 round-trips, and leading zero bytes survive.
void test_base58() {
    const std::vector<uint8_t> data = {0x00, 0x00, 0x01, 0x02, 0x03};
    const std::string encoded = CryptoUtils::base58_encode(data);

    check_equal(encoded, "11Ldp", "base58 of a value with two leading zeros");
    check(CryptoUtils::base58_decode(encoded) == data, "base58 round-trip");

    // Each leading zero byte becomes one leading '1'.
    check_equal(CryptoUtils::base58_encode({0x00, 0x00, 0x00, 0x01}), "1112",
                "base58 of three leading zeros");

    // A character outside the alphabet must be rejected, not skipped. Skipping
    // would make two different strings decode to the same bytes.
    bool rejected = false;
    try {
        CryptoUtils::base58_decode("0OIl");
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "base58 rejects characters outside the alphabet");
}

/// Base64 round-trips, and bad input is refused.
void test_base64() {
    const std::vector<uint8_t> hello = {'h', 'e', 'l', 'l', 'o'};

    check_equal(CryptoUtils::base64_encode(hello), "aGVsbG8=", "base64 of \"hello\"");
    check(CryptoUtils::base64_decode("aGVsbG8=") == hello, "base64 round-trip");

    // The URL-safe alphabet decodes to the same bytes.
    const std::vector<uint8_t> high = {0xFB, 0xFF};
    check(CryptoUtils::base64_decode("-_8=") == high, "base64 URL-safe alphabet");

    bool rejected = false;
    try {
        CryptoUtils::base64_decode("a*b!");
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "base64 rejects characters outside the alphabet");
}

/// The Bitcoin P2PKH address matches the published vector.
void test_bitcoin_address() {
    // The compressed genesis-block public key.
    const std::string hex = "03678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb6";

    auto raw = CryptoUtils::hex_decode(hex);
    check(raw.size() == 33, "the compressed key decodes to 33 bytes");

    std::array<uint8_t, 33> pubkey{};
    std::copy(raw.begin(), raw.end(), pubkey.begin());

    check_equal(CryptoUtils::derive_bitcoin_address(pubkey),
                "1FYPDCP1uVnPgEE3gaDMbAApdv9XYX7Si5",
                "P2PKH address of the compressed genesis key");

    check(CryptoUtils::derive_bitcoin_address(pubkey, true) !=
              CryptoUtils::derive_bitcoin_address(pubkey),
          "testnet and mainnet addresses differ");
}

/// BIP32 master key and child derivation match the published test vector 1.
void test_bip32_vectors() {
    // Vector 1 seed: 000102030405060708090a0b0c0d0e0f.
    auto seed = CryptoUtils::hex_decode("000102030405060708090a0b0c0d0e0f");
    auto [master_key, chaincode] = BIP32::master_key_from_seed(seed);

    check_equal(CryptoUtils::hex_encode(master_key.data(), 32),
                "e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b35",
                "BIP32 vector 1 master key");
    check_equal(CryptoUtils::hex_encode(chaincode.data(), 32),
                "873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d508",
                "BIP32 vector 1 master chain code");

    // m/0' is hardened, so the private key goes into the HMAC.
    auto [first_key, first_chain] = BIP32::derive_child_key(master_key, chaincode, 0x80000000u);
    check_equal(CryptoUtils::hex_encode(first_key.data(), 32),
                "edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afea",
                "BIP32 vector 1 m/0' key");

    // m/0'/1 is not hardened, so the public key goes into the HMAC. A wrong
    // choice here still produces a key, just not the right one.
    auto [second_key, second_chain] = BIP32::derive_child_key(first_key, first_chain, 1);
    check_equal(CryptoUtils::hex_encode(second_key.data(), 32),
                "3c6cb8d0f6a264c91ea8b5030fadaa8e538b020f0a387421a12de9319dc93368",
                "BIP32 vector 1 m/0'/1 key");

    std::array<uint8_t, 65> public_key{};
    check(CryptoUtils::public_key_from_private(second_key, public_key),
          "the derived key produces a public key");

    auto compressed = CryptoUtils::compress_pubkey(public_key);
    check_equal(CryptoUtils::hex_encode(compressed.data(), 33),
                "03501e454bf00751f24b1b489aa925215d66af2234e3891c3b21a52bedb3cd711c",
                "BIP32 vector 1 m/0'/1 public key");

    // A seed outside the allowed range is refused.
    bool rejected = false;
    try {
        BIP32::master_key_from_seed({1, 2, 3});
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "a seed shorter than 16 bytes is rejected");
}

/// BIP39 seed derivation and checksum validation match the published vectors.
void test_bip39() {
    // The published vector: the all-"abandon" phrase with a final "about".
    const std::string mnemonic =
        "abandon abandon abandon abandon abandon abandon "
        "abandon abandon abandon abandon abandon about";

    check(BIP39::validate_mnemonic(mnemonic), "the published test phrase validates");

    // A phrase with the same words in a different order is a different wallet,
    // so the checksum must reject it.
    const std::string swapped =
        "abandon abandon abandon abandon abandon abandon "
        "abandon abandon abandon abandon about abandon";
    check(!BIP39::validate_mnemonic(swapped),
          "a phrase with the words out of order is rejected");

    // A word outside the list is rejected.
    const std::string unknown =
        "notaword abandon abandon abandon abandon abandon "
        "abandon abandon abandon abandon abandon about";
    check(!BIP39::validate_mnemonic(unknown), "a word outside the list is rejected");

    // The wrong number of words is rejected.
    check(!BIP39::validate_mnemonic("abandon abandon abandon"),
          "a phrase with too few words is rejected");

    auto seed = BIP39::mnemonic_to_seed(mnemonic);
    check_equal(CryptoUtils::hex_encode(seed.data(), seed.size()),
                "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc1"
                "9a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4",
                "the BIP39 seed for the published phrase");

    // A passphrase changes the seed.
    auto with_passphrase = BIP39::mnemonic_to_seed(mnemonic, "TREZOR");
    check(with_passphrase != seed, "a passphrase changes the seed");

    // Generated phrases must validate, and the length follows the strength.
    for (uint32_t strength : {128u, 160u, 192u, 224u, 256u}) {
        auto generated = BIP39::generate_mnemonic(strength);
        const std::string phrase(generated.begin(), generated.end());

        check(BIP39::validate_mnemonic(phrase),
              "a generated " + std::to_string(strength) + "-bit phrase validates");

        const size_t expected_words = (strength + strength / 32) / 11;
        std::istringstream iss(phrase);
        std::string word;
        size_t count = 0;
        while (iss >> word) count++;
        check(count == expected_words,
              "a generated " + std::to_string(strength) + "-bit phrase has the right word count");
    }

    // A strength outside the specification is refused.
    bool rejected = false;
    try {
        BIP39::generate_mnemonic(100);
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "a strength BIP39 does not define is rejected");
}

/// EVM addresses use Keccak-256 and carry the EIP-55 checksum.
void test_evm_address() {
    // The private key 0x...01 has this well-known address.
    std::array<uint8_t, 32> private_key{};
    private_key[31] = 1;

    std::array<uint8_t, 65> public_key{};
    check(CryptoUtils::public_key_from_private(private_key, public_key),
          "private key 1 produces a public key");

    std::array<uint8_t, 33> compressed = CryptoUtils::compress_pubkey(public_key);
    check_equal(CryptoUtils::derive_evm_address(compressed),
                "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf",
                "EIP-55 address for private key 1");

    // The checksum is case-sensitive, so a lower-case spelling must differ.
    const std::string address = CryptoUtils::derive_evm_address(compressed);
    check(address != "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf",
          "the address carries an EIP-55 checksum");
    check(address.substr(0, 2) == "0x", "the address is 0x-prefixed");
    check(address.size() == 42, "the address is 42 characters");
}

/// Keccak-256 is used, not SHA3-256.
void test_keccak() {
    // Keccak-256 of the empty input.
    auto digest = CryptoUtils::keccak256({});
    check_equal(CryptoUtils::hex_encode(digest.data(), 32),
                "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
                "keccak256 of an empty input");

    // SHA3-256 of the same input differs, which is what makes the distinction
    // matter for Ethereum.
    auto sha3 = CryptoUtils::sha256({});
    check(CryptoUtils::hex_encode(sha3.data(), 32) !=
              CryptoUtils::hex_encode(digest.data(), 32),
          "keccak256 is not sha256");
}

/// ECDSA verification accepts a real signature and rejects a tampered one.
void test_ecdsa_verify() {
    auto pubkey = CryptoUtils::hex_decode(
        "02bb50e2d89a4ed70663d080659fe0ad4b9bc3e06c17a227433966cb59ceee020d");
    auto message = CryptoUtils::hex_decode(
        "dc9cba0a7e84acadd5aed5e5553b9de94b3f024bca9c66559fe85e7eddbfc6a0");

    std::vector<uint8_t> signature = CryptoUtils::hex_decode(
        "7ea3304dcdbe64f420fa120f6ac9b428e09fe9a8bcfce340c4569a994aa70043"
        "e3901a45e4506dba08eeeb90ab23b1a5426cfaef26d9edeb22f62a8e376efd32");

    std::array<uint8_t, SECP256K1_PUBKEY_SIZE> key{};
    std::copy(pubkey.begin(), pubkey.end(), key.begin());

    std::array<uint8_t, 32> digest{};
    std::copy(message.begin(), message.end(), digest.begin());

    std::array<uint8_t, SECP256K1_SIG_SIZE> sig{};
    std::copy(signature.begin(), signature.end(), sig.begin());

    check(CryptoUtils::ecdsa_verify(digest, sig, key),
          "a real signature verifies");

    // Flipping one bit in r must make verification fail.
    sig[0] ^= 0x01;
    check(!CryptoUtils::ecdsa_verify(digest, sig, key),
          "a tampered signature is rejected");
    sig[0] ^= 0x01;

    // A different message must not verify against this signature.
    digest[0] ^= 0x01;
    check(!CryptoUtils::ecdsa_verify(digest, sig, key),
          "a signature for another message is rejected");
    digest[0] ^= 0x01;

    // A key that is not a point on the curve is rejected rather than trusted.
    std::array<uint8_t, SECP256K1_PUBKEY_SIZE> bad_key{};
    bad_key[0] = 0x02;
    check(!CryptoUtils::ecdsa_verify(digest, sig, bad_key),
          "a key that is not on the curve is rejected");
}

/// The JSON reader accepts real documents and rejects malformed ones.
void test_json_parser() {
    auto document = Value::parse(
        R"({"jsonrpc":"2.0","id":1,"result":"0x1b4","ok":true,"none":null,"list":[1,2,3]})");

    check(document.is_object(), "a JSON object parses");
    check_equal(document.string_at("jsonrpc"), "2.0", "a string member");
    check(document.at("id").as_number() == 1.0, "a number member");
    check(document.at("ok").as_boolean(), "a boolean member");
    check(document.at("none").is_null(), "a null member");
    check(document.at("list").size() == 3, "an array member");

    // Escapes and surrogate pairs.
    auto escaped = Value::parse(R"({"s":"a\"b\\c\n\u0041\ud83d\ude00"})");
    const std::string expected = std::string("a\"b\\c\n") + "A" + "\xf0\x9f\x98\x80";
    check_equal(escaped.string_at("s"), expected, "escapes and a surrogate pair");

    // A missing member is an error rather than an empty value.
    bool threw = false;
    try {
        document.at("absent");
    } catch (const tigerwallet::json::ParseError&) {
        threw = true;
    }
    check(threw, "a missing member throws");

    // A wrong type is an error rather than a silent conversion.
    threw = false;
    try {
        document.at("id").as_string();
    } catch (const tigerwallet::json::ParseError&) {
        threw = true;
    }
    check(threw, "reading a number as a string throws");

    // Malformed documents are refused.
    const char* bad[] = {
        "",
        "{",
        "{\"a\":}",
        "{\"a\":1,}",
        "[1,2",
        "{\"a\":1}{\"b\":2}",
        "{\"a\":01}",
        "{\"a\":1e}",
        "nul",
        "{\"a\":\"unterminated}",
        "{\"a\":1,\"a\":2}",
        "\"raw\x01control\"",
    };

    for (const char* text : bad) {
        bool rejected = false;
        try {
            Value::parse(text);
        } catch (const tigerwallet::json::ParseError&) {
            rejected = true;
        }
        check(rejected, std::string("a malformed document is rejected: ") + text);
    }
}

/// ABI decoding reads the shapes the EVM returns.
void test_abi_decoding() {
    // balanceOf returns one 32-byte word.
    std::string decimal;
    check(decode_abi_uint256("0x0000000000000000000000000000000000000000000000000000000000ff0000",
                             decimal),
          "a 32-byte word decodes");
    check_equal(decimal, "16711680", "the word decodes to the right decimal");

    // A value above 64 bits is still exact.
    check(decode_abi_uint256("0x0000000000000000000000000000000000000000000000000000000000000001",
                             decimal),
          "one decodes");
    check_equal(decimal, "1", "one is one");

    check(decode_abi_uint256(
              "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", decimal),
          "the maximum word decodes");
    check_equal(decimal,
                "115792089237316195423570985008687907853269984665640564039457584007913129639935",
                "the maximum word decodes exactly");

    // A word that is not 32 bytes is refused.
    check(!decode_abi_uint256("0x00", decimal), "a short word is refused");
    check(!decode_abi_uint256("not hex", decimal), "a non-hex word is refused");

    // tokenURI returns offset, length, then the bytes.
    // offset 32, length 4, "ipfs"
    const std::string encoded =
        "0x0000000000000000000000000000000000000000000000000000000000000020"
        "0000000000000000000000000000000000000000000000000000000000000004"
        "6970667300000000000000000000000000000000000000000000000000000000";

    std::string text;
    check(decode_abi_string(encoded, text), "a dynamic string decodes");
    check_equal(text, "ipfs", "the string content is right");

    // An empty string is valid.
    const std::string empty =
        "0x0000000000000000000000000000000000000000000000000000000000000020"
        "0000000000000000000000000000000000000000000000000000000000000000";
    check(decode_abi_string(empty, text) && text.empty(), "an empty string decodes");

    // A length that runs past the data is refused rather than read.
    const std::string truncated =
        "0x0000000000000000000000000000000000000000000000000000000000000020"
        "00000000000000000000000000000000000000000000000000000000000000ff"
        "69706673";
    check(!decode_abi_string(truncated, text), "a truncated string is refused");
}

/// The HTTPS URL splitter accepts only usable endpoints.
void test_url_parsing() {
    std::string host;
    std::string port;
    std::string path;

    check(parse_https_url("https://rpc.example.org", host, port, path), "a bare host parses");
    check_equal(host, "rpc.example.org", "the host is right");
    check_equal(port, "443", "the default port is 443");
    check_equal(path, "/", "the default path is /");

    check(parse_https_url("https://node.example.org:8545/rpc", host, port, path),
          "a host with a port and path parses");
    check_equal(port, "8545", "the explicit port is right");
    check_equal(path, "/rpc", "the path is right");

    // Anything that is not https is refused, as are embedded credentials.
    check(!parse_https_url("http://rpc.example.org", host, port, path),
          "a plain http URL is refused");
    check(!parse_https_url("https://user:pass@rpc.example.org", host, port, path),
          "credentials in the URL are refused");
    check(!parse_https_url("https://rpc.example.org:abc", host, port, path),
          "a non-numeric port is refused");
    check(!parse_https_url("https://", host, port, path), "an empty host is refused");
}

/// A derivation path round-trips through parse and build.
void test_bip44_path() {
    auto components = BIP44::parse_path("m/44'/60'/0'/0/5");

    check(components.purpose == (44 | 0x80000000), "purpose carries the hardened bit");
    check(components.coin_type == (60 | 0x80000000), "coin type carries the hardened bit");
    check(components.account == (0 | 0x80000000), "account carries the hardened bit");
    check(components.change == 0, "change is not hardened");
    check(components.address_index == 5, "address index is read");

    check_equal(BIP44::build_path(components), "m/44'/60'/0'/0/5", "path round-trip");

    // A malformed path is an error. A silent default would sign with a key the
    // caller did not ask for.
    bool rejected = false;
    try {
        BIP44::parse_path("m/44'/abc/0");
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "a non-numeric path level is rejected");

    rejected = false;
    try {
        BIP44::parse_path("");
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "an empty path is rejected");

    rejected = false;
    try {
        BIP44::parse_path("m/44'/60'/0'/0/4294967296");
    } catch (const std::invalid_argument&) {
        rejected = true;
    }
    check(rejected, "a level above 2^31-1 is rejected");
}

/// Device enumeration only reports what is present.
void test_device_enumeration() {
    auto devices = HIDManager::enumerate_devices();

    // Every entry must map to a model the driver can talk to.
    bool all_known = true;
    for (const auto& device : devices) {
        if (device.model == DeviceModel::UNKNOWN) {
            all_known = false;
        }
    }

    check(all_known, "every enumerated device maps to a known model");

    // The manager never claims a connection it did not make.
    auto& manager = HardwareWalletManager::get_instance();
    check(!manager.get_connected_device(DeviceModel::LEDGER_NANO_X).has_value(),
          "no Ledger is reported as connected before one is opened");
}

/// A model's USB identity is either real or absent, never invented.
void test_usb_identity() {
    auto nano_x = HardwareWalletManager::usb_identity(DeviceModel::LEDGER_NANO_X);
    check(nano_x.has_value(), "the Ledger Nano X has a USB identity");
    check(nano_x->vendor_id == 0x2c97, "the Ledger vendor id is correct");

    auto trezor = HardwareWalletManager::usb_identity(DeviceModel::TREZOR_T);
    check(trezor.has_value(), "the Trezor Model T has a USB identity");
    check(trezor->vendor_id == 0x1209, "the Trezor vendor id is correct");

    // Ellipal and SafePal have no USB identity in this build, so the accessor
    // says so rather than returning a made-up one.
    check(!HardwareWalletManager::usb_identity(DeviceModel::ELLIPAL_TITAN).has_value(),
          "Ellipal has no USB identity");
    check(!HardwareWalletManager::usb_identity(DeviceModel::UNKNOWN).has_value(),
          "an unknown model has no USB identity");
}

/// Signing without a device fails rather than returning a placeholder.
void test_signing_fails_closed() {
    LedgerWallet ledger(DeviceModel::LEDGER_NANO_X);

    PublicKeyInfo info{};
    ErrorCode result = ledger.get_public_key("m/44'/60'/0'/0/0", info);
    check(result != ErrorCode::SUCCESS, "reading a key without a device fails");

    SignatureInfo signature{};
    result = ledger.sign_transaction({1, 2, 3}, "m/44'/60'/0'/0/0", signature);
    check(result != ErrorCode::SUCCESS, "signing without a device fails");
    check(signature.signature.empty(), "no signature is produced without a device");

    TrezorWallet trezor(DeviceModel::TREZOR_T);
    result = trezor.factory_reset();
    check(result != ErrorCode::SUCCESS, "an unimplemented operation reports failure");

    std::string version;
    result = trezor.get_firmware_version(version);
    check(result != ErrorCode::SUCCESS && version.empty(), "the firmware version is not invented");
}

}  // namespace

int main() {
    test_sha256();
    test_base58();
    test_base64();
    test_bitcoin_address();
    test_bip39();
    test_bip32_vectors();
    test_evm_address();
    test_keccak();
    test_json_parser();
    test_abi_decoding();
    test_url_parsing();
    test_ecdsa_verify();
    test_bip44_path();
    test_device_enumeration();
    test_usb_identity();
    test_signing_fails_closed();

    std::printf("\n%d checks, %d failures\n", checks, failures);
    return failures == 0 ? 0 : 1;
}
