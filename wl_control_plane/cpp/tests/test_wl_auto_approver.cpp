// Tests for the WlAutoApprover transaction classifier. Compiles + runs with
// g++ -std=c++20. Asserts the two-mode security boundary:
//   - User transfers / swaps / stakes / signs => AUTO mode (license-alive => approved)
//   - Revenue / treasury / fee withdrawals => MANUAL mode (never auto-approved)
//   - Recipient on the treasury-address set => MANUAL mode
//   - License dead => AUTO mode + DENIED
//   - Blocking auto-sign rule => AUTO mode + DENIED
#include "wl_auto_approver.hpp"

#include <cassert>
#include <cstdio>
#include <string>

using namespace tigerwallet::wl;

static void reset_approver() {
    WlAutoApprover::instance().set_alive(false, "reset");
    WlAutoApprover::instance().set_treasury_addresses({});
    WlAutoApprover::instance().set_rules({});
}

static void test_user_transfer_auto_approved() {
    reset_approver();
    WlAutoApprover::instance().set_alive(true);
    auto d = WlAutoApprover::instance().classify("transfer", "0x742d35Cc", "", "1.5");
    assert(d.mode == ApprovalMode::Auto);
    assert(d.approved);
    printf("ok: user transfer => Auto + approved (fast path <1ms)\n");
}

static void test_swap_stake_sign_auto_approved() {
    reset_approver();
    WlAutoApprover::instance().set_alive(true);
    assert(WlAutoApprover::instance().classify("swap", "0xUniswapRouter", "", "100").mode == ApprovalMode::Auto);
    assert(WlAutoApprover::instance().classify("stake", "0xLido", "", "32").mode == ApprovalMode::Auto);
    assert(WlAutoApprover::instance().classify("personal_sign", "", "", "").mode == ApprovalMode::Auto);
    printf("ok: swap / stake / personal_sign => Auto\n");
}

static void test_revenue_treasury_fee_always_manual() {
    reset_approver();
    WlAutoApprover::instance().set_alive(true);
    // Even with the license alive, these MUST be manual.
    assert(WlAutoApprover::instance().classify("revenue_payout", "0xColdWallet", "", "50000").mode == ApprovalMode::Manual);
    assert(WlAutoApprover::instance().classify("treasury_transfer", "0xSafe", "", "10000").mode == ApprovalMode::Manual);
    assert(WlAutoApprover::instance().classify("treasury_sweep", "0xSafe", "", "0").mode == ApprovalMode::Manual);
    assert(WlAutoApprover::instance().classify("fee_withdrawal", "0xFeeAcct", "", "200").mode == ApprovalMode::Manual);
    printf("ok: revenue/treasury/fee withdrawals => Manual (never auto-approved)\n");
}

static void test_treasury_recipient_forces_manual() {
    reset_approver();
    WlAutoApprover::instance().set_alive(true);
    WlAutoApprover::instance().add_treasury_address("0x1234567890abcdef1234567890abcdef12345678");
    // A "transfer" to a treasury address => Manual (security boundary)
    auto d = WlAutoApprover::instance().classify("transfer", "0x1234567890abcdef1234567890abcdef12345678", "", "5");
    assert(d.mode == ApprovalMode::Manual);
    printf("ok: transfer to treasury address => Manual (can't route fees through fast path)\n");
}

static void test_license_dead_denies_auto() {
    reset_approver();
    WlAutoApprover::instance().set_alive(false, "halted by SuperAdmin");
    auto d = WlAutoApprover::instance().classify("transfer", "0xabc", "", "1.0");
    assert(d.mode == ApprovalMode::Auto);
    assert(!d.approved);
    assert(d.reason.find("not authorized") != std::string::npos);
    printf("ok: license dead => Auto mode + DENIED (fail-closed)\n");
}

static void test_blocking_rule_denies_auto() {
    reset_approver();
    WlAutoApprover::instance().set_alive(true);
    WlAutoApprover::instance().set_rules({{"r1", "user_wallet", "*", "transfer", "*", "0", true}});
    auto d = WlAutoApprover::instance().classify("transfer", "0xabc", "", "1.0");
    assert(d.mode == ApprovalMode::Auto);
    assert(!d.approved);
    assert(d.rule_id == "r1");
    printf("ok: blocking auto-sign rule => Auto mode + DENIED\n");
}

static void test_c_abi() {
    reset_approver();
    wl_auto_approver_set_alive(1, nullptr);
    const char* reason = nullptr;
    const char* rule_id = nullptr;
    // user transfer => 0 (auto-approved)
    int rc = wl_auto_approve_classify("transfer", "0xUser", "", "2.0", &reason, &rule_id);
    assert(rc == 0);

    // revenue payout => 1 (manual)
    rc = wl_auto_approve_classify("revenue_payout", "0xColdWallet", "", "9999", &reason, &rule_id);
    assert(rc == 1);

    // license dead => 2 (denied)
    wl_auto_approver_set_alive(0, "halted");
    rc = wl_auto_approve_classify("transfer", "0xUser", "", "2.0", &reason, &rule_id);
    assert(rc == 2);
    assert(reason != nullptr && reason[0] != '\0');

    // treasury address json snapshot
    wl_auto_approver_set_alive(1, nullptr);
    wl_auto_approver_set_treasury_addresses_json("[\"0xabc123\",\"0xdef456\"]");
    rc = wl_auto_approve_classify("transfer", "0xabc123", "", "1.0", &reason, &rule_id);
    assert(rc == 1); // manual

    // rules json snapshot
    wl_auto_approver_set_rules_json("[{\"rule_id\":\"r99\",\"tx_type\":\"swap\",\"token\":\"*\",\"max_amount\":\"0\",\"block\":true}]");
    rc = wl_auto_approve_classify("swap", "0xRouter", "", "10", &reason, &rule_id);
    assert(rc == 2); // denied by rule
    assert(rule_id != nullptr && std::string(rule_id) == "r99");

    printf("ok: C ABI roundtrip (classify + json snapshots)\n");
}

int main() {
    test_user_transfer_auto_approved();
    test_swap_stake_sign_auto_approved();
    test_revenue_treasury_fee_always_manual();
    test_treasury_recipient_forces_manual();
    test_license_dead_denies_auto();
    test_blocking_rule_denies_auto();
    test_c_abi();
    printf("ALL AUTO-APPROVER TESTS PASSED\n");
    return 0;
}
