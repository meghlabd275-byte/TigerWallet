// 256-bit arithmetic tests.
//
// The case that matters is the one the 64-bit code got wrong: a product larger
// than 64 bits. The test prints what the old path produced so the difference
// is visible.

#include "uint256.hpp"

#include <cstdio>
#include <string>
using namespace tigerwallet::math;
static int failures = 0;
static void check(const char* name, const std::string& got, const std::string& want) {
    bool ok = got == want;
    if (!ok) ++failures;
    printf("%-34s %s got=%s want=%s\n", name, ok?"PASS":"FAIL", got.c_str(), want.c_str());
}
int main() {
    // 2^128 * 2^128 = 2^256 -> overflows 256 bits, so the low 256 bits are 0.
    U256 a = U256::fromWords(0,0,1,0);   // 2^128
    U256 b = U256::fromWords(0,0,1,0);   // 2^128
    check("(2^128 * 2^128) low 256", (a*b).toDecimal(), "0");

    // mulDiv must not overflow: 2^128 * 2^128 / 2^128 = 2^128.
    check("mulDiv 2^128*2^128/2^128", mulDiv(a,b,a).toDecimal(),
          U256::fromWords(0,0,1,0).toDecimal());

    // A real AMM case: 1000 tokens in, reserves 1e18 and 5e17.
    U256 amountIn(1000ULL);
    U256 reserveIn(1000000000000000000ULL);
    U256 reserveOut(500000000000000000ULL);
    // amountOut = 997*1000*5e17 / (1e18*1000 + 997*1000) = 498499999999999502
    U256 withFee = mulDiv(amountIn, U256(997), U256(1000));
    U256 numerator = mulDiv(withFee, reserveOut, U256(1));
    U256 denominator = reserveIn + withFee;
    U256 out = divideFull(multiplyFull(numerator, U256(1)), denominator);
    printf("    amm out = %s\n", out.toDecimal().c_str());
    check("amm out < reserveOut", out < reserveOut ? "yes":"no", "yes");
    check("amm out > 0", out.isZero() ? "no":"yes", "yes");

    // The old 64-bit path: 1e18 * 5e17 overflows uint64 and wraps.
    uint64_t oldStyle = (amountIn.toUint64() * 997 / 1000) * reserveOut.toUint64();
    printf("    old 64-bit product wrapped to %llu\n", (unsigned long long)oldStyle);

    // Division by zero returns zero rather than trapping.
    check("mulDiv by zero", mulDiv(U256(5), U256(7), U256(0)).toDecimal(), "0");

    // Decimal round trip.
    check("fromDecimal 2^64", U256::fromDecimal("18446744073709551616").toDecimal(),
          "18446744073709551616");
    check("add carry", (U256::fromWords(~0ULL,0,0,0) + U256(1)).toDecimal(),
          "18446744073709551616");
    check("sub borrow", (U256::fromWords(0,1,0,0) - U256(1)).toDecimal(),
          "18446744073709551615");
    check("compare", (U256(5) > U256(3)) ? "gt":"le", "gt");
    printf("%s\n", failures==0?"ALL U256 CHECKS PASS":"FAILURES");
    return failures;
}
