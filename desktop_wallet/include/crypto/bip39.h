/**
 * TigerWallet Desktop - BIP-39 Mnemonic Implementation
 *
 * Real BIP-39: entropy → checksum → mnemonic words, and mnemonic → seed
 * via PBKDF2-HMAC-SHA512 (2048 iterations). Uses the standard English
 * wordlist (2048 words). No hardcoded mnemonics.
 */

#ifndef TIGERWALLET_BIP39_H
#define TIGERWALLET_BIP39_H

#include <string>
#include <vector>
#include <cstdint>

namespace tiger {
namespace wallet {

class BIP39 {
public:
    // Generate a mnemonic from random entropy.
    // entropyBits: 128 (12 words), 160 (15), 192 (18), 224 (21), 256 (24).
    static std::string generateMnemonic(int entropyBits = 256);

    // Validate a mnemonic: correct word count, all words in list, valid checksum.
    static bool validateMnemonic(const std::string& mnemonic);

    // Convert mnemonic to seed via PBKDF2-HMAC-SHA512 (2048 iterations).
    // passphrase is optional (BIP-39 standard).
    static std::vector<uint8_t> mnemonicToSeed(const std::string& mnemonic,
                                                const std::string& passphrase = "");

    // Get the wordlist index of a word (-1 if not found).
    static int wordIndex(const std::string& word);

    // Get word at index.
    static std::string wordAt(int index);

    // Number of words in the list.
    static constexpr int WORD_COUNT = 2048;

private:
    static const char* WORDLIST[];
    static std::vector<uint8_t> generateEntropy(int bits);
    static int checksumBits(const std::vector<uint8_t>& entropy);
    static std::vector<uint8_t> sha256(const uint8_t* data, size_t len);
};

} // namespace wallet
} // namespace tiger

#endif // TIGERWALLET_BIP39_H
