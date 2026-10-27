// Tests for the privacy module.
//
// The critical cases are the ones that were wrong before: a forged proof must
// be rejected, a SNARK range proof must not be reported as valid, and the
// shielded balance must reflect what the wallet actually holds rather than a
// hardcoded zero.

#include <iostream>
#include <string>
#include <vector>

#include "privacy.hpp"

using namespace tigerwallet::privacy;

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

void TestCommitmentHash() {
    std::cout << "CommitmentHash\n";
    const auto a = CommitmentHash::hash_single("abc");
    Check("hash_single is 64 hex characters", a.size() == 64);
    Check("hash_single is deterministic", a == CommitmentHash::hash_single("abc"));
    Check("different inputs differ", a != CommitmentHash::hash_single("abd"));

    // hash_single length-prefixes its input with a 4-byte little-endian
    // length before hashing, so this is Keccak-256 of "\x03\x00\x00\x00abc".
    // Verified against `openssl dgst -keccak-256` on that exact byte string.
    CheckEqual("hash_single matches Keccak-256 of the length-prefixed input", a,
               "e3b9babb640b590b338a90440465c015e8c007497ccf5f457682009a4d60ad67");

    // Without the prefix the digest must differ, which is the point of the
    // prefix: hash_single("abc") must not equal keccak("abc").
    Check("the length prefix changes the digest",
          a != std::string("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"));

    // Domain separation: the length prefix must stop these colliding.
    const auto pair = CommitmentHash::hash_pair("ab", "c");
    const auto other = CommitmentHash::hash_pair("a", "bc");
    Check("hash_pair separates different splits of the same bytes", pair != other);

    const auto single = CommitmentHash::hash_single("abc");
    Check("hash_pair differs from hash_single of the concatenation", pair != single);
}

void TestHashCommitment() {
    std::cout << "HashCommitment\n";
    const auto c = HashCommitment::commit("100", "randomness");
    Check("commit is deterministic", c == HashCommitment::commit("100", "randomness"));
    Check("a different value gives a different commitment",
          c != HashCommitment::commit("101", "randomness"));
    Check("different randomness gives a different commitment",
          c != HashCommitment::commit("100", "other"));
    Check("verify accepts the right opening", HashCommitment::verify(c, "100", "randomness"));
    Check("verify rejects a wrong value", !HashCommitment::verify(c, "101", "randomness"));
    Check("verify rejects a wrong randomness",
          !HashCommitment::verify(c, "100", "other"));

    const auto with_asset = HashCommitment::commit_with_asset("100", "USDT", "r");
    Check("commit_with_asset is deterministic",
          with_asset == HashCommitment::commit_with_asset("100", "USDT", "r"));
    Check("the asset is bound into the commitment",
          with_asset != HashCommitment::commit_with_asset("100", "USDC", "r"));
}

void TestMerkleTree() {
    std::cout << "MerkleTree\n";
    MerkleTree tree(32);
    Check("an empty tree has no root", tree.get_root().empty());
    Check("an empty tree has size 0", tree.size() == 0);

    const auto first = tree.insert("leaf-1");
    Check("inserting returns a non-empty root", !first.empty());
    Check("size is 1", tree.size() == 1);

    const auto root_one = tree.get_root();
    const auto second = tree.insert("leaf-2");
    Check("adding a leaf changes the root", second != root_one);
    Check("size is 2", tree.size() == 2);

    const auto path = tree.get_path(0);
    Check("a path is produced for a leaf", !path.empty());
    Check("the path verifies against the root", tree.verify_path(path, 0, tree.get_root()));
    Check("the path does not verify against a different root",
          !tree.verify_path(path, 0, root_one));

    tree.reset();
    Check("reset clears the tree", tree.size() == 0 && tree.get_root().empty());
}

void TestRangeProofIsNotFaked() {
    std::cout << "range proofs\n";
    ZKProofGenerator generator("bn254", "groth16");

    const auto proof = generator.generate_range_proof("500", "commitment-abc");
    // There is no on-device prover, so this must not look like a valid proof.
    Check("a range proof is not reported as valid", !proof.is_valid);
    Check("a range proof carries no proof data", proof.proof_data.empty());
    Check("a range proof carries no verification key", proof.verification_key.empty());
    // The amount must not be published: the proof exists to hide it.
    Check("the amount is not in the public inputs",
          proof.public_inputs.find("500") == std::string::npos);

    Check("verifying an empty proof fails", !generator.verify_proof(proof));
}

void TestForgedProofIsRejected() {
    std::cout << "forged proofs\n";
    ZKProofGenerator generator("bn254", "groth16");

    // This is the case the previous implementation accepted. It returned true
    // whenever proof_data, is_valid and verification_key were all non-empty,
    // and all three are fields the caller controls.
    ZKProof forged;
    forged.proof_data = "i-am-not-a-proof";
    forged.public_inputs = "whatever";
    forged.verification_key = "i-am-not-a-key";
    forged.protocol = "groth16";
    forged.is_valid = true;
    Check("a proof with self-declared validity is rejected", !generator.verify_proof(forged));

    ZKProof forged_no_key = forged;
    forged_no_key.verification_key.clear();
    Check("a keyless hand-built proof is rejected", !generator.verify_proof(forged_no_key));

    ZKProof forged_merkle = forged;
    forged_merkle.proof_data = "2:aaaa:bbbb";
    forged_merkle.verification_key.clear();
    Check("a fabricated Merkle path is rejected", !generator.verify_proof(forged_merkle));

    ZKProof empty;
    Check("a default-constructed proof is rejected", !generator.verify_proof(empty));
}

void TestMerkleProofGeneration() {
    std::cout << "Merkle proof generation\n";
    ZKProofGenerator generator("bn254", "merkle");

    // A path that genuinely reaches the root must be recognised as valid, and
    // one that does not must not.
    std::string computed = "leaf";
    std::vector<std::string> path;
    for (int i = 0; i < 3; ++i) {
        const std::string sibling = "sibling-" + std::to_string(i);
        path.push_back(sibling);
        computed = CommitmentHash::hash_pair(computed, sibling);
    }

    const auto good = generator.generate_merkle_proof("leaf", path, computed);
    Check("a matching path is marked valid", good.is_valid);
    Check("a matching path carries proof data", !good.proof_data.empty());

    const auto bad = generator.generate_merkle_proof("leaf", path, "not-the-root");
    Check("a non-matching path is marked invalid", !bad.is_valid);
    Check("a non-matching path carries no proof data", bad.proof_data.empty());
}

void TestShieldedBalance() {
    std::cout << "shielded balance\n";
    PrivacyWallet wallet("test-seed-for-balance", PrivacyConfig());

    // The balance started as a hardcoded "0" regardless of deposits.
    CheckEqual("a new wallet has a zero balance", wallet.get_shielded_balance(), "0");

    const auto id = wallet.shield_funds("0xfrom", "100", "shielded");
    Check("shielding funds returns a transaction id", !id.empty());
    CheckEqual("the balance reflects the deposit", wallet.get_shielded_balance(), "100");

    const auto second = wallet.shield_funds("0xfrom", "250", "shielded");
    Check("a second deposit succeeds", !second.empty());
    CheckEqual("deposits accumulate", wallet.get_shielded_balance(), "350");

    const auto balances = wallet.get_all_balances();
    Check("the balances map is not empty", !balances.empty());
    Check("the balances map does not claim a fabricated transparent balance",
          balances.find("transparent") == balances.end());

    Check("shielding without a source address is refused",
          wallet.shield_funds("", "10", "shielded").empty());
    Check("shielding a zero amount is refused",
          wallet.shield_funds("0xfrom", "", "shielded").empty());
}

void TestUnshield() {
    std::cout << "unshield\n";
    PrivacyWallet wallet("test-seed-for-unshield", PrivacyConfig());
    wallet.shield_funds("0xfrom", "500", "shielded");

    // Withdrawing more than the balance must fail: the wallet cannot create a
    // claim on funds it does not hold.
    Check("withdrawing more than the balance is refused",
          wallet.unshield_funds("0xto", "600", "shielded").empty());
    CheckEqual("a refused withdrawal leaves the balance alone",
               wallet.get_shielded_balance(), "500");

    const auto id = wallet.unshield_funds("0xto", "200", "shielded");
    Check("a withdrawal within the balance succeeds", !id.empty());
    CheckEqual("the balance is debited", wallet.get_shielded_balance(), "300");

    Check("withdrawing exactly the remainder succeeds",
          !wallet.unshield_funds("0xto", "300", "shielded").empty());
    CheckEqual("the balance is now zero", wallet.get_shielded_balance(), "0");

    Check("withdrawing with no destination is refused",
          wallet.unshield_funds("", "1", "shielded").empty());
}

void TestNotes() {
    std::cout << "notes\n";
    PrivacyWallet wallet("test-seed-for-notes", PrivacyConfig());
    wallet.shield_funds("0xfrom", "100", "shielded");
    wallet.shield_funds("0xfrom", "250", "shielded");

    const auto notes = wallet.scan_for_notes("0", "1000");
    Check("both notes are returned", notes.size() == 2);
    if (notes.size() == 2) {
        Check("the first note has a commitment", !notes[0].commitment.empty());
        Check("the first note has a nullifier", !notes[0].nullifier.empty());
        Check("the notes are unspent", !notes[0].spent && !notes[1].spent);
        Check("the two notes have different commitments",
              notes[0].commitment != notes[1].commitment);

        // Spending a note must reduce the balance by that note's value.
        wallet.mark_note_spent(notes[0].nullifier);
        CheckEqual("spending a note debits its value", wallet.get_shielded_balance(), "250");

        // Spending the same note twice must not debit twice.
        wallet.mark_note_spent(notes[0].nullifier);
        CheckEqual("spending the same note twice does not double-debit",
                   wallet.get_shielded_balance(), "250");

        wallet.mark_note_spent("0xnot-a-nullifier");
        CheckEqual("an unknown nullifier changes nothing",
                   wallet.get_shielded_balance(), "250");
    }
}

void TestShieldedKeys() {
    std::cout << "shielded keys\n";
    PrivacyWallet first("seed-one", PrivacyConfig());
    PrivacyWallet second("seed-two", PrivacyConfig());

    Check("a viewing key is produced", !first.get_viewing_key().empty());
    Check("a spending key is produced", !first.get_spending_key().empty());
    Check("the viewing and spending keys differ",
          first.get_viewing_key() != first.get_spending_key());
    Check("different seeds give different viewing keys",
          first.get_viewing_key() != second.get_viewing_key());
    Check("different seeds give different spending keys",
          first.get_spending_key() != second.get_spending_key());
    Check("a shielded address is produced", !first.get_shielded_address().empty());
    Check("the same seed is reproducible",
          PrivacyWallet("seed-one", PrivacyConfig()).get_viewing_key() ==
              first.get_viewing_key());
}

void TestTransfers() {
    std::cout << "shielded transfers\n";
    PrivacyWallet wallet("test-seed-for-transfers", PrivacyConfig());
    wallet.shield_funds("0xfrom", "400", "shielded");

    const auto id = wallet.transfer_between_shielded("shield_other", "150", "memo");
    Check("a transfer within the balance succeeds", !id.empty());
    CheckEqual("the balance is debited", wallet.get_shielded_balance(), "250");

    Check("a transfer beyond the balance is refused",
          wallet.transfer_between_shielded("shield_other", "1000", "").empty());
    CheckEqual("a refused transfer leaves the balance alone",
               wallet.get_shielded_balance(), "250");

    Check("a transfer with no destination is refused",
          wallet.transfer_between_shielded("", "10", "").empty());
}

void TestCoinJoin() {
    std::cout << "CoinJoin\n";

    // An anonymity set of 3, so a round is only mixable with three participants.
    CoinJoinMixer mixer(5, 3);

    // A round with no participants must not be created.
    Check("a round with no inputs is refused", mixer.create_round({}).empty());

    const auto round_id = mixer.create_round({"input-a", "input-b"});
    Check("a round with participants is created", !round_id.empty());

    auto round = mixer.get_round_status(round_id);
    Check("the round records its participants", round.current_participants == 2);
    Check("the round is not yet collectable", round.status == "pending");

    // Joining needs a proof of ownership of the input.
    Check("joining without a proof is refused",
          mixer.join_round(round_id, "input-c", "").empty());
    Check("joining without an input is refused",
          mixer.join_round(round_id, "", "proof").empty());

    // The same input must not be counted twice.
    Check("a duplicate input is refused",
          mixer.join_round(round_id, "input-a", "proof").empty());

    Check("a valid join is accepted",
          !mixer.join_round(round_id, "input-c", "proof-c").empty());
    round = mixer.get_round_status(round_id);
    Check("the round is now collectable", round.status == "collecting");

    // A round without a denomination has no agreed output size, so it must not
    // produce outputs. The previous version emitted one output per participant
    // with the amount "0".
    const auto no_denom = mixer.process_round(round_id);
    Check("a round with no denomination produces no outputs", no_denom.empty());
    Check("a round with no denomination is marked failed",
          mixer.get_round_status(round_id).status == "failed");

    // An unknown round must not be mixable.
    Check("processing an unknown round produces no outputs",
          mixer.process_round("ROUND_does_not_exist").empty());

    // An empty round cannot be looked up either.
    const auto missing = mixer.get_round_status("ROUND_does_not_exist");
    Check("an unknown round returns an empty status", missing.id.empty());

    // A round that is not full must not be processed.
    CoinJoinMixer second(5, 3);
    const auto small = second.create_round({"only-one"});
    Check("a round below the anonymity set produces no outputs",
          second.process_round(small).empty());
    Check("a round below the anonymity set stays collecting",
          second.get_round_status(small).status == "collecting");
}

}  // namespace

int main() {
    TestCommitmentHash();
    TestHashCommitment();
    TestMerkleTree();
    TestRangeProofIsNotFaked();
    TestForgedProofIsRejected();
    TestMerkleProofGeneration();
    TestShieldedBalance();
    TestUnshield();
    TestNotes();
    TestShieldedKeys();
    TestTransfers();
    TestCoinJoin();

    std::cout << (failures == 0 ? "privacy: all tests passed\n"
                                : "privacy: " + std::to_string(failures) + " test(s) failed\n");
    return failures == 0 ? 0 : 1;
}
