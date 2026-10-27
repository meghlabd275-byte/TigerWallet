// Tests for BIP39 and BIP32.
//
// The expectations are the published test vectors, not values produced by this
// code: the BIP39 vectors are the reference set published alongside the BIP,
// and the BIP32 vectors are the ones printed in BIP-0032 itself. Matching the
// xprv and xpub strings means the whole chain is right at once, since those
// strings cover the version bytes, depth, parent fingerprint, child number,
// chain code and key material.

#include <array>
#include <cstdint>
#include <iostream>
#include <string>
#include <vector>

#include "crypto.hpp"
#include "hdwallet.hpp"

#include "bip32_vectors.hpp"
#include "bip39_vectors.hpp"

using namespace tigerwallet;
using namespace tigerwallet::hd;

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

void TestBase58() {
    std::cout << "Base58\n";
    // Round trips over a range of byte patterns, including leading zeros.
    const std::vector<std::vector<uint8_t>> inputs = {
        {},
        {0x00},
        {0x00, 0x00},
        {0x00, 0x01},
        {0x01, 0x00},
        {0xFF},
        {0xFF, 0xFF, 0xFF},
        std::vector<uint8_t>(32, 0xAB),
        std::vector<uint8_t>(78, 0x00),
    };
    bool all_round_trip = true;
    for (const auto& input : inputs) {
        const auto encoded = Base58Encode(input);
        std::vector<uint8_t> decoded;
        if (!Base58Decode(encoded, decoded) || decoded != input) all_round_trip = false;
    }
    Check("encode/decode round trips, including leading zeros", all_round_trip);

    // Known encodings. Base58 has no 0, O, I or l.
    CheckEqual("0x00 encodes as '1'", Base58Encode({0x00}), "1");
    CheckEqual("0x0001 encodes as '12'", Base58Encode({0x00, 0x01}), "12");
    Check("the alphabet excludes lookalike characters",
          Base58Encode({0xFF, 0xFF}).find_first_of("0OIl") == std::string::npos);

    std::vector<uint8_t> out;
    Check("rejects the character 0", !Base58Decode("0", out));
    Check("rejects the character O", !Base58Decode("O", out));
    Check("rejects the character I", !Base58Decode("I", out));
    Check("rejects the character l", !Base58Decode("l", out));
    Check("rejects an embedded space", !Base58Decode("1 2", out));

    // Base58Check detects a single-character corruption.
    const std::vector<uint8_t> payload = {0x00, 0x11, 0x22, 0x33, 0x44, 0x55};
    const auto checked = Base58CheckEncode(payload);
    std::vector<uint8_t> recovered;
    Check("Base58Check round trips", Base58CheckDecode(checked, recovered) && recovered == payload);

    std::string corrupted = checked;
    corrupted[corrupted.size() - 1] =
        corrupted[corrupted.size() - 1] == '1' ? '2' : '1';
    Check("Base58Check rejects a corrupted checksum",
          !Base58CheckDecode(corrupted, recovered));
}

void TestRipemd160() {
    std::cout << "RIPEMD-160\n";
    // The published vectors from the RIPEMD-160 specification.
    struct Case { const char* input; const char* want; };
    const Case cases[] = {
        {"", "9c1185a5c5e9fc54612808977ee8f548b2258d31"},
        {"a", "0bdc9d2d256b3ee9daae347be6f4dc835a467ffe"},
        {"abc", "8eb208f7e05d987a9b044a8e98c6b087f15a0bfc"},
        {"message digest", "5d0689ef49d2fae572b881b123a85ffa21595f36"},
        {"abcdefghijklmnopqrstuvwxyz", "f71c27109c692c1b56bbdceb5b9d2865b3708dbc"},
        {"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq",
         "12a053384a9c0c88e405a06c27dcf49ada62eb2b"},
        {"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
         "b0e20b6e3116640286ed3a87a5713079b21f5189"},
        {"12345678901234567890123456789012345678901234567890123456789012345678901234567890",
         "9b752e45573d4b39f4dbd3323cab82bf63326bfb"},
    };
    for (const auto& c : cases) {
        const std::string input(c.input);
        const auto digest = Ripemd160(reinterpret_cast<const uint8_t*>(input.data()),
                                      input.size());
        const std::string got = crypto::ToHex(digest.data(), digest.size());
        CheckEqual(std::string("RIPEMD-160 of \"") + input.substr(0, 12) + "\"", got, c.want);
    }
}

void TestBip39Vectors() {
    std::cout << "BIP39 vectors\n";
    int checked = 0;
    for (size_t i = 0; i < test_vectors::kBip39VectorCount; ++i) {
        const auto& vector = test_vectors::kBip39Vectors[i];
        const std::string mnemonic = vector.mnemonic;
        const std::string label = "vector " + std::to_string(i) + " (" +
                                  std::to_string(mnemonic.size()) + " chars)";

        // Entropy to mnemonic.
        const auto entropy = crypto::FromHex(vector.entropy_hex);
        const auto generated = EntropyToMnemonic(entropy);
        if (!generated || *generated != mnemonic) {
            Check("entropy -> mnemonic " + label, false);
            if (generated) {
                std::cout << "        got  " << generated->substr(0, 60) << "\n"
                          << "        want " << mnemonic.substr(0, 60) << "\n";
            }
            continue;
        }

        // Mnemonic to entropy, and validation.
        std::vector<uint8_t> round_trip;
        const bool ok = MnemonicToEntropy(mnemonic, round_trip) && round_trip == entropy &&
                        ValidateMnemonic(mnemonic);
        if (!ok) {
            Check("mnemonic -> entropy " + label, false);
            continue;
        }

        // The seed, with the passphrase "TREZOR" that the vectors use.
        const auto seed = MnemonicToSeed(mnemonic, "TREZOR");
        const std::string seed_hex = crypto::ToHex(seed.data(), seed.size());
        if (seed_hex != vector.seed_hex) {
            Check("seed " + label, false);
            std::cout << "        got  " << seed_hex.substr(0, 40) << "\n"
                      << "        want " << std::string(vector.seed_hex).substr(0, 40) << "\n";
            continue;
        }

        // The seed must produce the documented BIP32 root key.
        const auto master = MasterKeyFromSeed(seed);
        const auto xprv = master ? SerializeExtendedPrivateKey(*master) : std::nullopt;
        if (!xprv || *xprv != vector.root_xprv) {
            Check("root xprv " + label, false);
            continue;
        }

        ++checked;
    }
    std::cout << "        " << checked << " of " << test_vectors::kBip39VectorCount
              << " vectors passed in full\n";
    Check("every BIP39 vector matched on entropy, mnemonic, seed and root key",
          checked == static_cast<int>(test_vectors::kBip39VectorCount));
}

void TestBip39Rejections() {
    std::cout << "BIP39 rejections\n";
    // The last word of the 12-word all-zeros vector is "about"; changing it to
    // another valid word breaks the checksum.
    const std::string good =
        "abandon abandon abandon abandon abandon abandon abandon abandon abandon "
        "abandon abandon about";
    Check("a valid mnemonic is accepted", ValidateMnemonic(good));

    const std::string bad_checksum =
        "abandon abandon abandon abandon abandon abandon abandon abandon abandon "
        "abandon abandon abandon";
    Check("a wrong checksum is rejected", !ValidateMnemonic(bad_checksum));

    Check("an unknown word is rejected",
          !ValidateMnemonic("abandon abandon abandon abandon abandon abandon abandon "
                            "abandon abandon abandon abandon zzzz"));
    Check("an 11-word mnemonic is rejected",
          !ValidateMnemonic("abandon abandon abandon abandon abandon abandon abandon "
                            "abandon abandon abandon abandon"));
    Check("an empty mnemonic is rejected", !ValidateMnemonic(""));
    Check("a doubled space is rejected",
          !ValidateMnemonic("abandon  abandon abandon abandon abandon abandon abandon "
                            "abandon abandon abandon abandon about"));
    Check("a trailing space is rejected",
          !ValidateMnemonic(good + " "));
    Check("a leading space is rejected",
          !ValidateMnemonic(" " + good));
    Check("uppercase words are rejected",
          !ValidateMnemonic("Abandon abandon abandon abandon abandon abandon abandon "
                            "abandon abandon abandon abandon about"));

    std::vector<uint8_t> entropy;
    Check("a rejected mnemonic yields no entropy",
          !MnemonicToEntropy(bad_checksum, entropy));
}

void TestMnemonicGeneration() {
    std::cout << "mnemonic generation\n";
    for (size_t words : {12u, 15u, 18u, 21u, 24u}) {
        const auto mnemonic = GenerateMnemonic(words);
        Check(std::to_string(words) + "-word mnemonic is valid",
              mnemonic.has_value() && ValidateMnemonic(*mnemonic));
        if (mnemonic) {
            size_t count = 1;
            for (char c : *mnemonic) {
                if (c == ' ') ++count;
            }
            Check(std::to_string(words) + "-word mnemonic has the right length",
                  count == words);
        }
    }
    Check("13 words is not a valid length", !GenerateMnemonic(13).has_value());
    Check("0 words is not a valid length", !GenerateMnemonic(0).has_value());

    // Two generations must differ: a fixed or low-entropy source would show up
    // here.
    const auto first = GenerateMnemonic(12);
    const auto second = GenerateMnemonic(12);
    Check("two generated mnemonics differ", first && second && *first != *second);

    // Entropy of an unsupported size is refused.
    Check("15 bytes of entropy is refused",
          !EntropyToMnemonic(std::vector<uint8_t>(15, 0x00)).has_value());
    Check("32 bytes of entropy is accepted",
          EntropyToMnemonic(std::vector<uint8_t>(32, 0x00)).has_value());
}

void TestBip32Vectors() {
    std::cout << "BIP32 vectors\n";
    int chains_checked = 0;
    int chains_total = 0;

    for (size_t v = 0; v < test_vectors::kBip32VectorCount; ++v) {
        const auto& vector = test_vectors::kBip32Vectors[v];
        // BIP32 seeds are 16 to 64 bytes; the vectors use several lengths.
        const auto seed = crypto::FromHex(vector.seed_hex);
        if (seed.size() < 16 || seed.size() > 64) {
            Check("vector " + std::to_string(v) + " seed length is in range", false);
            continue;
        }

        const auto master = MasterKeyFromSeed(seed);
        if (!master) {
            Check("vector " + std::to_string(v) + " master key", false);
            continue;
        }

        for (size_t c = 0; c < vector.chain_count; ++c) {
            ++chains_total;
            const auto& chain = vector.chains[c];
            const std::string label =
                "vector " + std::to_string(v) + " " + chain.path;

            std::vector<PathElement> path;
            if (!ParsePath(chain.path, path)) {
                Check("parse " + label, false);
                continue;
            }

            const auto node = DerivePath(*master, path);
            if (!node) {
                Check("derive " + label, false);
                continue;
            }

            const auto xprv = SerializeExtendedPrivateKey(*node);
            const auto xpub = SerializeExtendedPublicKey(*node);
            if (!xprv || *xprv != chain.ext_prv) {
                Check("xprv " + label, false);
                if (xprv) std::cout << "        got  " << *xprv << "\n";
                std::cout << "        want " << chain.ext_prv << "\n";
                continue;
            }
            if (!xpub || *xpub != chain.ext_pub) {
                Check("xpub " + label, false);
                if (xpub) std::cout << "        got  " << *xpub << "\n";
                std::cout << "        want " << chain.ext_pub << "\n";
                continue;
            }
            ++chains_checked;
        }
    }
    std::cout << "        " << chains_checked << " of " << chains_total
              << " chain derivations matched\n";
    Check("every BIP32 vector matched on both xprv and xpub",
          chains_checked == chains_total && chains_total > 0);
}

void TestPublicDerivation() {
    std::cout << "public-only derivation\n";
    // BIP32 test vector 1: deriving m/0'/1 non-hardened from the parent's
    // public key must give the same public key as deriving it privately.
    const auto seed = crypto::FromHex("000102030405060708090a0b0c0d0e0f");
    const auto master = MasterKeyFromSeed(seed);
    Check("master key is available", master.has_value());
    if (!master) return;

    const auto parent = DerivePath(*master, "m/0'");
    Check("hardened child m/0' is available", parent.has_value());
    if (!parent) return;

    // Build a public-only copy of the parent.
    ExtendedKey public_parent;
    public_parent.public_key = parent->public_key;
    public_parent.chain_code = parent->chain_code;
    public_parent.depth = parent->depth;
    public_parent.child_number = parent->child_number;
    public_parent.parent_fingerprint = parent->parent_fingerprint;

    const auto from_private = DeriveChild(*parent, 1, false);
    const auto from_public = DeriveChild(public_parent, 1, false);
    Check("private and public derivation agree",
          from_private && from_public && from_private->public_key == from_public->public_key);

    // A hardened step cannot be derived from a public key.
    Check("hardened derivation from a public key is refused",
          !DeriveChild(public_parent, 1, true).has_value());
}

void TestPathParsing() {
    std::cout << "path parsing\n";
    std::vector<PathElement> path;

    Check("m is the master key", ParsePath("m", path) && path.empty());
    Check("a leading m is optional", ParsePath("0/1", path) && path.size() == 2);

    Check("standard BIP44 path", ParsePath("m/44'/60'/0'/0/0", path) && path.size() == 5);
    if (path.size() == 5) {
        Check("the first step is hardened index 44",
              path[0].hardened && path[0].index == (44u | 0x80000000u));
        Check("the last step is not hardened", !path[4].hardened && path[4].index == 0);
    }

    Check("apostrophe, h and H all mark hardened",
          ParsePath("m/1'/1h/1H", path) && path.size() == 3 && path[0].hardened &&
              path[1].hardened && path[2].hardened);

    Check("empty path is refused", !ParsePath("", path));
    Check("a trailing slash is refused", !ParsePath("m/", path));
    Check("a doubled slash is refused", !ParsePath("m//0", path));
    Check("a missing slash is refused", !ParsePath("m0", path));
    Check("a negative step is refused", !ParsePath("m/-1", path));
    Check("a non-numeric step is refused", !ParsePath("m/x", path));
    Check("a non-hardened index above 2^31-1 is refused",
          !ParsePath("m/2147483648", path));
    Check("a hardened index above 2^31-1 is refused",
          !ParsePath("m/2147483648'", path));
    Check("the largest valid non-hardened index is accepted",
          ParsePath("m/2147483647", path));
    Check("the largest valid hardened index is accepted",
          ParsePath("m/2147483647'", path));

    CheckEqual("Bip44Path builds the expected string",
               Bip44Path(44, 60, 0, 0, 5), "m/44'/60'/0'/0/5");
}

void TestDerivedAddresses() {
    std::cout << "derived addresses\n";
    // A known mnemonic and its first Ethereum address. The mnemonic is the
    // standard test phrase; the address is the widely published one for
    // m/44'/60'/0'/0/0 on it.
    const std::string mnemonic =
        "abandon abandon abandon abandon abandon abandon abandon abandon abandon "
        "abandon abandon about";
    Check("the test mnemonic is valid", ValidateMnemonic(mnemonic));

    const auto seed = MnemonicToSeed(mnemonic, "");
    const auto master = MasterKeyFromSeed(seed);
    Check("master key is available", master.has_value());
    if (!master) return;

    const auto node = DerivePath(*master, Bip44Path(44, 60, 0, 0, 0));
    Check("m/44'/60'/0'/0/0 is derivable", node.has_value());
    if (node) {
        const auto address = EthereumAddress(*node);
        Check("an Ethereum address is produced", address.has_value());
        if (address) {
            CheckEqual("first address for the test mnemonic",
                       crypto::AddressToString(*address),
                       "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
        }
    }

    // Distinct indices must give distinct addresses.
    const auto second = DerivePath(*master, Bip44Path(44, 60, 0, 0, 1));
    Check("index 1 differs from index 0",
          second && node && second->public_key != node->public_key);

    // A passphrase changes the wallet entirely, which is the point of it.
    const auto other_seed = MnemonicToSeed(mnemonic, "different passphrase");
    const auto other_master = MasterKeyFromSeed(other_seed);
    Check("a passphrase produces a different master key",
          other_master && other_master->private_key != master->private_key);

    // Bitcoin hash160 must differ from the Ethereum address derivation.
    const auto btc_node = DerivePath(*master, Bip44Path(44, 0, 0, 0, 0));
    Check("m/44'/0'/0'/0/0 is derivable", btc_node.has_value());
    if (btc_node) {
        Check("a Bitcoin hash160 is produced", BitcoinHash160(*btc_node).has_value());
    }
}

void TestCoinTypes() {
    std::cout << "BIP44 coin types\n";
    Check("Ethereum is 60", CoinTypeForChain("ETH") == 60);
    Check("Bitcoin is 0", CoinTypeForChain("BTC") == 0);
    Check("Solana is 501", CoinTypeForChain("SOL") == 501);
    Check("an unregistered symbol has no coin type",
          !CoinTypeForChain("NOTACOIN").has_value());
    Check("the lookup is case sensitive, as SLIP-44 is",
          !CoinTypeForChain("eth").has_value());
}

}  // namespace

int main() {
    TestBase58();
    TestRipemd160();
    TestBip39Vectors();
    TestBip39Rejections();
    TestMnemonicGeneration();
    TestBip32Vectors();
    TestPublicDerivation();
    TestPathParsing();
    TestDerivedAddresses();
    TestCoinTypes();

    std::cout << (failures == 0 ? "hdwallet: all tests passed\n"
                                : "hdwallet: " + std::to_string(failures) + " test(s) failed\n");
    return failures == 0 ? 0 : 1;
}
