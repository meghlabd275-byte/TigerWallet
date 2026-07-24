/**
 * TigerWallet Signature Verifier
 * Ultra-Low Latency C++ Implementation
 */

#ifndef TIGER_SIG_H
#define TIGER_SIG_H

#include <string>
#include <vector>

namespace tiger {
namespace crypto {

bool verify_signature(const std::string& message, const std::string& signature, const std::string& public_key);
std::string sign_message(const std::string& message, const std::string& private_key);
bool verify_ecdsa(const std::string& message, const std::vector<uint8_t>& signature, const std::vector<uint8_t>& pubkey);
bool verify_ed25519(const std::string& message, const std::vector<uint8_t>& signature, const std::vector<uint8_t>& pubkey);

}
}

#endif
