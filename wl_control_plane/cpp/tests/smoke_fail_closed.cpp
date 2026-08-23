// Real fail-closed smoke test for the WL control-plane C++ components.
//
// Simulates the exact production failure scenario: the process booted, but
// license validation NEVER succeeded (e.g. obviously-invalid license — bad
// Ed25519 signature / expired / revoked). The Rust LicenseClient therefore
// never calls wl_gate_set_alive(1, ...). Asserts every hot-path check denies.
//
// Ed25519 signature verification is intentionally NOT exercised here: it is
// delegated to the Rust SDK (wl_control_plane/rust/src/license.rs, tested
// separately via cargo). This test covers the C++-only paths: gate logic,
// flag cache, auto-approver classifier, and the C ABI FFI surface.
#include "wl_gate.hpp"
#include "wl_auto_approver.hpp"
#include "wl_gate_abi.h"

#include <cassert>
#include <cstdio>
#include <cstring>

using namespace tigerwallet::wl;

int main() {
    // --- 1. Gate with an invalid license => alive flag never set => deny all.
    WlGate& gate = WlGate::instance();
    gate.set_alive(false, "license invalid: Ed25519 signature verification failed");
    gate.set_flags({}); // no flags pushed either (validation never succeeded)

    assert(!gate.is_alive());
    assert(gate.reason() == "license invalid: Ed25519 signature verification failed");
    // Fail-closed: every fetcher of every product must be denied.
    assert(!gate.fetcher_enabled("user_wallet", "balances"));
    assert(!gate.fetcher_enabled("user_wallet", "swap_quote"));
    assert(!gate.fetcher_enabled("master_wallet", "transfer"));
    assert(!gate.request_allowed("user_wallet", "transactions"));

    // --- 2. C ABI view of the same invalid-license state.
    assert(wl_gate_is_alive() == 0);
    assert(std::strlen(wl_gate_reason()) > 0);
    assert(wl_gate_fetcher_enabled("user_wallet", "balances") == 0);
    assert(wl_gate_fetcher_enabled(nullptr, "balances") == 0); // null-safe deny

    // --- 3. Auto-approver with invalid license => Auto mode but DENIED (rc 2).
    WlAutoApprover& appr = WlAutoApprover::instance();
    appr.set_alive(false, "license invalid");
    appr.set_treasury_addresses({});
    appr.set_rules({});
    auto d = appr.classify("transfer", "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", "", "1.5");
    assert(d.mode == ApprovalMode::Auto);
    assert(!d.approved); // license dead => never auto-approved
    const char* reason = nullptr; const char* rule_id = nullptr;
    int rc = wl_auto_approve_classify("transfer", "0x742d35Cc", "", "1.5", &reason, &rule_id);
    assert(rc == 2); // Auto + DENIED
    assert(reason != nullptr && std::strlen(reason) > 0);

    // --- 4. Treasury/fee withdrawals are MANUAL even with a VALID license.
    appr.set_alive(true);
    appr.add_treasury_address("0xFEE000000000000000000000000000000000C0DE");
    auto m1 = appr.classify("revenue_payout", "0xAnywhere", "", "10");
    assert(m1.mode == ApprovalMode::Manual && !m1.approved);
    auto m2 = appr.classify("transfer", "0xfee000000000000000000000000000000000c0de", "", "0.1");
    assert(m2.mode == ApprovalMode::Manual && !m2.approved); // case-insensitive match
    rc = wl_auto_approve_classify("treasury_sweep", "0xTreasury", "", "5", &reason, &rule_id);
    assert(rc == 1); // Manual => caller must go through TwoPartyGate

    // --- 5. Valid license + user-initiated tx => AUTO approved (rc 0).
    gate.set_alive(true);
    assert(wl_gate_is_alive() == 1);
    assert(wl_gate_fetcher_enabled("user_wallet", "balances") == 1);
    rc = wl_auto_approve_classify("swap", "0xUniswapRouter", "", "100", &reason, &rule_id);
    assert(rc == 0);

    // --- 6. Revocation is immediate: flip back to dead => deny again.
    wl_gate_set_alive(0, "license revoked by SuperAdmin");
    wl_auto_approver_set_alive(0, "license revoked by SuperAdmin");
    assert(wl_gate_fetcher_enabled("user_wallet", "balances") == 0);
    rc = wl_auto_approve_classify("swap", "0xUniswapRouter", "", "100", &reason, &rule_id);
    assert(rc == 2);

    printf("SMOKE TEST PASSED: fail-closed on invalid license, manual-mode boundary, C ABI OK\n");
    return 0;
}
