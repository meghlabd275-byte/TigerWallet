// Tests for the privacy module.
//
// The behaviours under test here were previously broken in ways that mattered:
// commitment salts and pool secrets came from rand(), the proof verifier
// returned true for any input, and the file did not compile against the
// Crypto++ 8.x headers its own includes named.

#include "privacy_features.hpp"

#include <cassert>
#include <cstdio>
#include <set>
#include <string>

using namespace tiger;

namespace {

const Address kAlice = "0x1111111111111111111111111111111111111111";
const Address kBob = "0x2222222222222222222222222222222222222222";
const Address kCarol = "0x3333333333333333333333333333333333333333";

}  // namespace

int main() {
    // --- Randomness ----------------------------------------------------------

    // A 32-byte draw must not repeat. rand() has a 32-bit state, so draws from
    // it collide once the sequence wraps.
    {
        std::set<std::string> draws;
        for (int i = 0; i < 2000; ++i) {
            const Bytes b = secureRandomBytes(32);
            assert(b.size() == 32);
            assert(draws.insert(std::string(b.begin(), b.end())).second);
        }

        // No byte position may be constant across draws.
        std::vector<std::set<uint8_t>> seen(32);
        for (int i = 0; i < 200; ++i) {
            const Bytes b = secureRandomBytes(32);
            for (size_t p = 0; p < 32; ++p) seen[p].insert(b[p]);
        }
        for (size_t p = 0; p < 32; ++p) {
            assert(seen[p].size() > 100);
        }
    }

    // --- Commitments ---------------------------------------------------------

    // The same secret with two different salts must give two different
    // commitments, which is the property a predictable salt destroys.
    {
        ZKProver prover;
        const Bytes secret = secureRandomBytes(32);
        const Commitment a = prover.generateCommitment(secret, secureRandomBytes(32));
        const Commitment b = prover.generateCommitment(secret, secureRandomBytes(32));
        assert(a.commitment != b.commitment);

        // The same inputs must still give the same commitment.
        const Bytes salt = secureRandomBytes(32);
        const Commitment c = prover.generateCommitment(secret, salt);
        const Commitment d = prover.generateCommitment(secret, salt);
        assert(c.commitment == d.commitment);
    }

    // --- Proofs --------------------------------------------------------------

    {
        ZKProver prover;
        const Bytes secret = secureRandomBytes(32);
        const Bytes salt = secureRandomBytes(32);
        const Commitment commitment = prover.generateCommitment(secret, salt);
        const Nullifier nullifier = prover.generateNullifier(secret, kAlice);

        MerkleNode node;
        node.hash = secureRandomBytes(32);

        const ZKProof proof =
            prover.prove(commitment, nullifier, node, Bytes(kBob.begin(), kBob.end()), secret);
        assert(!proof.pi_a.empty());
        assert(!proof.signature.empty());
        assert(proof.signer.size() == 33);

        // The author's own proof verifies against the recorded signer.
        assert(prover.verify(proof, node.hash, {nullifier.hash}, proof.signer));

        // A different root is a different transcript, so the proof no longer
        // verifies. This is what stops a proof being replayed elsewhere.
        assert(!prover.verify(proof, secureRandomBytes(32), {nullifier.hash}, proof.signer));

        // The signature is bound to the author. A verifier expecting another
        // key rejects it.
        const Bytes other = ZKProver::publicKeyFor(
            ZKProver::signerKeyFromSecret(secureRandomBytes(32)));
        assert(!other.empty());
        assert(other != proof.signer);
        assert(!prover.verify(proof, node.hash, {nullifier.hash}, other));

        // An unsigned proof is rejected. The old verifier accepted it.
        ZKProof unsigned_proof = proof;
        unsigned_proof.signature.clear();
        assert(!prover.verify(unsigned_proof, node.hash, {nullifier.hash}, proof.signer));

        ZKProof empty_proof;
        assert(!prover.verify(empty_proof, node.hash, {}, {}));

        // A tampered public signal invalidates the signature.
        ZKProof tampered = proof;
        tampered.publicSignals[0] ^= 0xFF;
        assert(!prover.verify(tampered, node.hash, {nullifier.hash}, proof.signer));

        // A signature lifted from another proof does not verify here.
        const Nullifier other_nullifier = prover.generateNullifier(secret, kCarol);
        const ZKProof other_proof =
            prover.prove(commitment, other_nullifier, node, Bytes(kCarol.begin(), kCarol.end()), secret);
        ZKProof swapped = proof;
        swapped.signature = other_proof.signature;
        assert(!prover.verify(swapped, node.hash, {nullifier.hash}, proof.signer));
    }

    // --- Serialization -------------------------------------------------------

    {
        ZKProver prover;
        const Bytes secret = secureRandomBytes(32);
        const Commitment commitment =
            prover.generateCommitment(secret, secureRandomBytes(32));
        const Nullifier nullifier = prover.generateNullifier(secret, kAlice);
        MerkleNode node;
        node.hash = secureRandomBytes(32);

        const ZKProof proof =
            prover.prove(commitment, nullifier, node, Bytes(kBob.begin(), kBob.end()), secret);
        const Bytes encoded = PrivacyPool::serializeProof(proof);

        const auto decoded = PrivacyPool::deserializeProof(encoded);
        assert(decoded.has_value());
        assert(decoded->pi_a == proof.pi_a);
        assert(decoded->signature == proof.signature);
        assert(decoded->signer == proof.signer);

        // Truncated and over-long input is refused rather than guessed at.
        assert(!PrivacyPool::deserializeProof(
                    Bytes(encoded.begin(), encoded.end() - 1)).has_value());
        Bytes extended = encoded;
        extended.push_back(0x00);
        assert(!PrivacyPool::deserializeProof(extended).has_value());
        assert(!PrivacyPool::deserializeProof({}).has_value());
    }

    // --- Pool ----------------------------------------------------------------

    {
        PrivacyPool pool;
        const Bytes secret = secureRandomBytes(32);

        const std::string deposit =
            pool.deposit(kAlice, uint256_t::fromUint64(1000), secret);
        assert(!deposit.empty());

        const auto record = pool.getDeposit(deposit);
        assert(record.has_value());
        assert(!record->proofSigner.empty());
        assert(record->nullifier.size() == 32);

        // A withdrawal with no proof is refused. The old code accepted one.
        assert(!pool.withdraw(deposit, kBob, {}, pool.getMerkleRoot()));

        // A withdrawal with bytes that are not a proof is refused.
        assert(!pool.withdraw(deposit, kBob, Bytes(200, 0xAB), pool.getMerkleRoot()));

        // The proof the deposit owner holds, reconstructed from the secret
        // the deposit was made with, authorizes the withdrawal.
        ZKProver prover;
        MerkleNode deposit_node;
        deposit_node.hash = pool.getMerkleRoot();
        const Nullifier nullifier = prover.generateNullifier(secret, kAlice);
        const ZKProof owned = prover.prove(
            Commitment{record->commitment, secret, 0}, nullifier, deposit_node,
            Bytes(kBob.begin(), kBob.end()), secret);
        assert(prover.verify(owned, pool.getMerkleRoot(), {record->nullifier},
                             record->proofSigner));
        assert(pool.withdraw(deposit, kBob, PrivacyPool::serializeProof(owned),
                             pool.getMerkleRoot()));

        // A second withdrawal of the same deposit is refused.
        assert(!pool.withdraw(deposit, kCarol, PrivacyPool::serializeProof(owned),
                              pool.getMerkleRoot()));
    }

    // --- Deposit identifiers -------------------------------------------------

    {
        PrivacyPool pool;
        std::set<std::string> hashes;
        for (int i = 0; i < 500; ++i) {
            const std::string h = pool.deposit(
                kAlice, uint256_t::fromUint64(static_cast<uint64_t>(i) + 1),
                secureRandomBytes(32));
            assert(!h.empty());
            assert(hashes.insert(h).second);
        }
    }

    // --- Transfer ------------------------------------------------------------

    {
        PrivacyPool pool;
        const Bytes secret = secureRandomBytes(32);
        const std::string deposit =
            pool.deposit(kAlice, uint256_t::fromUint64(1000), secret);

        ZKProver prover;
        MerkleNode node;
        node.hash = pool.getMerkleRoot();
        const Nullifier nullifier = prover.generateNullifier(secret, kAlice);
        const ZKProof owned = prover.prove(
            Commitment{{}, secret, 0}, nullifier, node,
            Bytes(kBob.begin(), kBob.end()), secret);

        // A transfer with no proof is refused. The old code moved the deposit
        // on the strength of the secret argument alone.
        assert(!pool.transfer(deposit, kBob, uint256_t::fromUint64(10), {}, secret));

        // A transfer proved with a different secret is refused.
        const Bytes other_secret = secureRandomBytes(32);
        assert(!pool.transfer(deposit, kBob, uint256_t::fromUint64(10),
                              PrivacyPool::serializeProof(owned), other_secret));

        // A proof the deposit owner holds, with the matching secret, moves it.
        const auto record = pool.getDeposit(deposit);
        const ZKProof correct = prover.prove(
            Commitment{record->commitment, secret, 0}, nullifier, node,
            Bytes(kBob.begin(), kBob.end()), secret);
        assert(pool.transfer(deposit, kBob, uint256_t::fromUint64(10),
                             PrivacyPool::serializeProof(correct), secret));

        // The source deposit is now spent, so it cannot move again.
        assert(!pool.transfer(deposit, kCarol, uint256_t::fromUint64(10),
                              PrivacyPool::serializeProof(correct), secret));
    }

    // --- Address recovery ----------------------------------------------------

    {
        AddressRotation rotation;
        const Bytes secret = secureRandomBytes(32);
        const Bytes recovery_key = secureRandomBytes(32);

        const Address rotated = rotation.rotateWithHistory(kAlice, secret);

        // The original is found from the rotated address.
        assert(rotation.recoverAddress(rotated, recovery_key) == kAlice);

        // An address that was never rotated here has no original. The old
        // implementation returned the input, presenting it as the original.
        const Address stranger = "0x9999999999999999999999999999999999999999";
        assert(rotation.recoverAddress(stranger, recovery_key).empty());
        assert(rotation.recoverAddress("", recovery_key).empty());

        // A chain of rotations walks back to the start.
        const Address second = rotation.rotateWithHistory(rotated, secret);
        assert(rotation.recoverAddress(second, recovery_key) == kAlice);
    }

    // --- Merkle tree ---------------------------------------------------------

    {
        MerkleTree tree(8);
        const Bytes leaf = CryptoHash::keccak256(secureRandomBytes(32));
        assert(tree.insert(leaf));
        assert(tree.getSize() == 1);
        assert(tree.getRoot() != Bytes(32, 0));

        // A proof for the leaf verifies against the current root.
        const auto proof = tree.generateProof(0);
        assert(!proof.empty());
        assert(tree.verifyProof(leaf, proof, tree.getRoot()));

        // A different leaf does not.
        assert(!tree.verifyProof(CryptoHash::keccak256(secureRandomBytes(32)), proof,
                                 tree.getRoot()));
    }

    // --- Address rotation ----------------------------------------------------

    {
        AddressRotation rotation;
        const Bytes secret = secureRandomBytes(32);
        const Address a = rotation.rotateAddress(kAlice, secret, 0);
        const Address b = rotation.rotateAddress(kAlice, secret, 1);

        // Rotation is deterministic for the same inputs and the addresses are
        // well-formed 20-byte hex.
        assert(a == rotation.rotateAddress(kAlice, secret, 0));
        assert(a != b);
        for (const Address& addr : {a, b}) {
            assert(addr.size() == 42);
            assert(addr.rfind("0x", 0) == 0);
            for (size_t i = 2; i < addr.size(); ++i) {
                const char c = addr[i];
                assert((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'));
            }
        }

        // A different secret gives a different address.
        assert(a != rotation.rotateAddress(kAlice, secureRandomBytes(32), 0));
    }

    // --- CoinJoin ------------------------------------------------------------

    {
        CoinJoinMixer mixer;
        const std::string round =
            mixer.createRound({kAlice, kBob, kCarol}, 1000000, 10);
        assert(!round.empty());
        assert(round.rfind("0x", 0) == 0);

        const auto fetched = mixer.getRound(round);
        assert(fetched.has_value());
        assert(fetched->participants.size() == 3);
        assert(!mixer.getRound("0xdeadbeef").has_value());

        // Round ids must not repeat.
        std::set<std::string> ids;
        for (int i = 0; i < 100; ++i) {
            const std::string id = mixer.createRound({kAlice, kBob}, 1000, 5);
            assert(ids.insert(id).second);
        }
    }

    std::printf("privacy_features: all checks passed\n");
    return 0;
}
