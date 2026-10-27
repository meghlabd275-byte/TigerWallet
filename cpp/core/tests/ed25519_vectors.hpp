// Ed25519 test vectors from RFC 8032, section 7.1.
//
// Source: https://www.rfc-editor.org/rfc/rfc8032.txt
// The secret key is the 32-byte seed. The public key and the signature are
// the values the RFC prints for that seed and message.
//
// The RFC elides the body of very long messages with "...", so TEST 1024
// (a 1023-byte message) cannot be reproduced from the document and is not
// included. The remaining vectors publish their message in full.

#ifndef TIGERWALLET_TEST_ED25519_VECTORS_HPP
#define TIGERWALLET_TEST_ED25519_VECTORS_HPP

#include <cstddef>

namespace tigerwallet {
namespace test_vectors {

struct Ed25519Vector {
    const char* name;
    const char* secret_key_hex;
    const char* public_key_hex;
    const char* message_hex;
    const char* signature_hex;
};

inline constexpr Ed25519Vector kEd25519Vectors[] = {
    {"RFC 8032 TEST 1 (message 0 bytes)",
     "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60",
     "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a",
     "",
     "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b"},
    {"RFC 8032 TEST 2 (message 1 bytes)",
     "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb",
     "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
     "72",
     "92a009a9f0d4cab8720e820b5f642540a2b27b5416503f8fb3762223ebdb69da085ac1e43e15996e458f3613d0f11d8c387b2eaeb4302aeeb00d291612bb0c00"},
    {"RFC 8032 TEST 3 (message 2 bytes)",
     "c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7",
     "fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025",
     "af82",
     "6291d657deec24024827e69c3abe01a30ce548a284743a445e3680d7db5ac3ac18ff9b538d16f290ae67f760984dc6594a7c15e9716ed28dc027beceea1ec40a"},
};

inline constexpr size_t kEd25519VectorCount =
    sizeof(kEd25519Vectors) / sizeof(kEd25519Vectors[0]);

}  // namespace test_vectors
}  // namespace tigerwallet

#endif  // TIGERWALLET_TEST_ED25519_VECTORS_HPP
