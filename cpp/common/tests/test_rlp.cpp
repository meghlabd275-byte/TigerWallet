// RLP test vectors, taken from the Ethereum RLP specification.

#include "rlp.hpp"

#include <cstdio>
#include <string>
using namespace tigerwallet::rlp;
int main() {
    int failures = 0;
    auto check = [&](const char* name, const Bytes& got, const char* want) {
        std::string g = Encoder::toHex(got);
        bool ok = g == want;
        if (!ok) ++failures;
        printf("%-28s %s got=%s want=%s\n", name, ok?"PASS":"FAIL", g.c_str(), want);
    };
    // From the Ethereum RLP specification test vectors.
    check("empty string", Encoder::encodeString(""), "80");
    check("dog", Encoder::encodeString("dog"), "83646f67");
    check("single 0x00 byte", Encoder::encodeBytes(Bytes{0x00}), "00");
    check("single 0x0f byte", Encoder::encodeBytes(Bytes{0x0f}), "0f");
    check("single 0x7f byte", Encoder::encodeBytes(Bytes{0x7f}), "7f");
    check("single 0x80 byte", Encoder::encodeBytes(Bytes{0x80}), "8180");
    check("int 0", Encoder::encodeUint(0), "80");
    check("int 15", Encoder::encodeUint(15), "0f");
    check("int 1024", Encoder::encodeUint(1024), "820400");
    // "Lorem ipsum dolor sit amet, consectetur adipisicing elit" is 56 bytes,
    // so it needs the long form.
    check("56-byte string",
          Encoder::encodeString("Lorem ipsum dolor sit amet, consectetur adipisicing elit"),
          "b8384c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e7365637465747572206164697069736963696e6720656c6974");
    check("empty list", Encoder::encodeList({}), "c0");
    check("list of cat dog",
          Encoder::encodeList({Encoder::encodeString("cat"), Encoder::encodeString("dog")}),
          "c88363617483646f67");
    check("nested [ [], [[]], [[], [[]]] ]",
          Encoder::encodeList({
              Encoder::encodeList({}),
              Encoder::encodeList({Encoder::encodeList({})}),
              Encoder::encodeList({Encoder::encodeList({}), Encoder::encodeList({Encoder::encodeList({})})})
          }),
          "c7c0c1c0c3c0c1c0");
    // A 256-bit value: 2^64, held as words[1] = 1.
    uint64_t v[4] = {0, 1, 0, 0};
    check("uint256 2^64", Encoder::encodeUint256(v), "89010000000000000000");
    uint64_t zero[4] = {0,0,0,0};
    check("uint256 0", Encoder::encodeUint256(zero), "80");
    printf("%s\n", failures==0?"ALL RLP VECTORS PASS":"FAILURES");
    return failures;
}
