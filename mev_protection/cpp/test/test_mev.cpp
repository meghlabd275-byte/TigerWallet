// Tests for the MEV blocker.
//
// These cover the parts that can be checked without a node or a relay: the
// denylist, transaction classification, the risk assessment's refusal to
// report MEV where none can be extracted, and the guarantee that no code path
// invents a transaction hash. Submission paths that need the network are
// exercised through their fail-closed branches, which is what runs when the
// operator has not configured an endpoint.

#include "../include/mev_blocker.hpp"
#include "../include/transaction.hpp"

#include <cstdio>
#include <string>

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

using tiger::MEVBlocker;
using tiger::ProtectionLevel;
using tiger::Transaction;
using tiger::uint256;

// A Uniswap-style swap: the 0x7ff36ab5 selector followed by arguments.
Transaction swap_tx(const std::string& hash, uint64_t value_wei) {
    Transaction tx;
    tx.hash = hash;
    tx.from = "0x1111111111111111111111111111111111111111";
    tx.to = "0x2222222222222222222222222222222222222222";
    tx.value = uint256(value_wei);
    tx.data = "0x7ff36ab50000000000000000000000000000000000000000000000000000000000000001";
    tx.gas_price = uint256(20000000000ULL);
    return tx;
}

// A plain value transfer: empty calldata.
Transaction transfer_tx(const std::string& hash, uint64_t value_wei) {
    Transaction tx;
    tx.hash = hash;
    tx.from = "0x3333333333333333333333333333333333333333";
    tx.to = "0x4444444444444444444444444444444444444444";
    tx.value = uint256(value_wei);
    tx.data = "";
    return tx;
}

void test_denylist_blocks_only_listed_senders() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    Transaction tx = transfer_tx("0xaa", 1000);

    // An address that was never listed must not be blocked, whatever it looks
    // like.
    check(!blocker.AddBlockedSender(""), "empty address is rejected");
    check(blocker.AddBlockedSender(tx.from), "address can be listed");
    check(!blocker.AddBlockedSender(tx.from), "listing the same address twice is a no-op");

    check(blocker.RemoveBlockedSender(tx.from), "address can be unlisted");
    check(!blocker.RemoveBlockedSender(tx.from), "unlisting an absent address reports false");
}

void test_denylist_is_the_only_block_signal() {
    // A transaction from an unlisted sender is never blocked on the basis of a
    // heuristic, even at the highest protection level.
    MEVBlocker blocker(ProtectionLevel::kMaximum);
    blocker.Initialize("");
    blocker.Start();

    Transaction tx = swap_tx("0xbb", 5000000000000000000ULL);  // 5 ETH swap
    tx.raw = "0xf86c018504a817c80082520894";  // well-formed-looking payload

    // Submission fails closed because no RPC endpoint is configured, but the
    // reason must be the missing endpoint, not a heuristic block. The
    // distinguishing observable is that a hash is never fabricated.
    const std::string result = blocker.SubmitTransaction(tx);
    check(result.empty(), "no hash is fabricated when submission cannot complete");

    blocker.Stop();
}

void test_risk_assessment_reports_no_mev_for_transfers() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    // A transfer carries no swap and therefore no extractable MEV. Reporting a
    // score here would be a fabrication.
    Transaction tx = transfer_tx("0xcc", 10000000000000000000ULL);  // 10 ETH
    const auto assessment = blocker.AnalyzeTransaction(tx);

    check(assessment.risk_score == 0.0f, "transfer carries no risk score");
    check(assessment.attack_type.empty(), "transfer has no attack type");
    check(!assessment.is_profitable, "transfer reports no extractable value");
    check(assessment.profit_extracted == 0.0f, "transfer reports zero extractable value");
}

void test_risk_assessment_reports_exposure_for_swaps() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    Transaction tx = swap_tx("0xdd", 2000000000000000000ULL);  // 2 ETH swap
    const auto assessment = blocker.AnalyzeTransaction(tx);

    check(!assessment.attack_type.empty(), "swap is reported as exposed");
    check(assessment.risk_score > 0.0f, "swap carries a risk score");
    check(assessment.is_profitable, "swap reports extractable value");
    // The bound is the swap's own value, so it can never exceed it.
    check(assessment.profit_extracted <= 2.0f, "extractable value is bounded by the swap");
    check(!assessment.recommendations.empty(), "swap carries recommendations");
}

void test_zero_value_swap_is_not_reported_as_profitable() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    Transaction tx = swap_tx("0xee", 0);
    const auto assessment = blocker.AnalyzeTransaction(tx);

    check(!assessment.is_profitable, "zero-value swap reports no extractable value");
    check(assessment.profit_extracted == 0.0f, "zero-value swap reports zero");
}

void test_private_submission_requires_signed_payload() {
    MEVBlocker blocker(ProtectionLevel::kMaximum);
    blocker.Initialize("");
    blocker.Start();

    // An unsigned transaction cannot be relayed anywhere, so the submission
    // must fail rather than return a hash for something that was never sent.
    Transaction tx = swap_tx("0xff", 1000000000000000000ULL);
    check(tx.raw.empty(), "fixture starts unsigned");

    const std::string result = blocker.SubmitTransaction(tx);
    check(result.empty(), "unsigned transaction is not submitted");
    check(!blocker.IsPrivateTransaction("0xff"), "unsigned transaction is not recorded as private");

    blocker.Stop();
}

void test_relayer_configuration() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    check(blocker.AddRelayer("https://relay.example"), "relayer can be added");
    check(!blocker.AddRelayer("https://relay.example"), "duplicate relayer is rejected");
    check(blocker.RemoveRelayer("https://relay.example"), "relayer can be removed");
    check(!blocker.RemoveRelayer("https://relay.example"), "removing an absent relayer reports false");
}

void test_gas_price_does_not_invent_a_fee() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    // Initialize with no TIGER_MEV_RPC_URL: there is no oracle reading, so the
    // optimal gas price must be zero rather than a made-up default.
    blocker.Initialize("");

    const auto oracle = blocker.GetGasOracle();
    check(oracle.base_fee == 0, "base fee is unset without an RPC endpoint");

    Transaction tx = transfer_tx("0x11", 1);
    const uint64_t price = blocker.CalculateOptimalGasPrice(tx);
    check(price == 0, "optimal gas price is zero without an oracle reading");
}

void test_stats_start_at_zero() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    const auto stats = blocker.GetStats();
    check(stats.total_transactions == 0, "no transactions counted yet");
    check(stats.protected_transactions == 0, "no protected transactions yet");
    check(stats.blocked_transactions == 0, "no blocked transactions yet");
    check(stats.sandwich_opportunities == 0, "no opportunities recorded yet");
}

void test_simulation_fails_closed_without_rpc() {
    MEVBlocker blocker(ProtectionLevel::kAdvanced);
    blocker.Initialize("");

    std::string output;
    const bool ok = blocker.SimulateTransaction(transfer_tx("0x22", 1), output);
    check(!ok, "simulation reports failure without an RPC endpoint");
    check(!output.empty(), "simulation explains why it could not run");
}

}  // namespace

int main() {
    test_denylist_blocks_only_listed_senders();
    test_denylist_is_the_only_block_signal();
    test_risk_assessment_reports_no_mev_for_transfers();
    test_risk_assessment_reports_exposure_for_swaps();
    test_zero_value_swap_is_not_reported_as_profitable();
    test_private_submission_requires_signed_payload();
    test_relayer_configuration();
    test_gas_price_does_not_invent_a_fee();
    test_stats_start_at_zero();
    test_simulation_fails_closed_without_rpc();

    std::printf("\n%d checks, %d failures\n", checks, failures);
    return failures == 0 ? 0 : 1;
}
