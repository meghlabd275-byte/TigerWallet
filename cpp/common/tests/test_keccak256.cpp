// Keccak-256 test vectors.
//
// The vectors are the published Keccak-256 digests: the empty string, "abc"
// and "testing". The selector check is the ERC-20 transfer selector, which is
// what a wallet must produce to build a token transfer.

#include "keccak256.hpp"

#include <cstdio>
#include <string>
using namespace tigerwallet::crypto;
int main() {
    struct { const char* in; const char* out; } vectors[] = {
        {"", "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"},
        {"abc", "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"},
        {"testing", "5f16f4c7f149ac4f9510d9cf8cf384038ad348b3bcdc01915f95de12df9d1b02"},
    };
    int failures = 0;
    for (auto& v : vectors) {
        auto got = Keccak256::toHex(Keccak256::hash(std::string(v.in)));
        bool ok = got == v.out;
        if (!ok) ++failures;
        printf("%-10s %s\n  got %s\n  want %s\n", ok ? "PASS" : "FAIL", v.in, got.c_str(), v.out);
    }
    // transfer(address,uint256) selector is 0xa9059cbb
    uint32_t sel = Keccak256::selector("transfer(address,uint256)");
    printf("%s selector = 0x%08x (want 0xa9059cbb)\n", sel == 0xa9059cbb ? "PASS" : "FAIL", sel);
    if (sel != 0xa9059cbb) ++failures;
    printf("%s\n", failures == 0 ? "ALL KECCAK VECTORS PASS" : "FAILURES");
    return failures;
}
