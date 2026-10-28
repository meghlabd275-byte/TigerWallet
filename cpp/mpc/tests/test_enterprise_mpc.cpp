// Tests for the MPC engine's threshold scheme and signature verification.
//
// These target the three fabricated operations that were here before:
//
//   1. verifySignature returned true for any non-empty byte string, so every
//      forgery was accepted. Every forgery case below passed under that check.
//   2. generatePublicShare returned SHA-256 of a share, and generateKeyShares
//      set the public key to the polynomial's constant term, which is the
//      secret itself.
//   3. combineSignatureShares XORed shares rather than interpolating them.
//
// The tests assert real properties: the public shares are points on the curve,
// the public key is the secret times the generator, any t of n shares
// interpolate to the same key, fewer than t do not, and a signature verifies
// only against the message it was made for.

#include "enterprise_mpc.hpp"

#include <cassert>
#include <cstdio>
#include <string>
#include <vector>

namespace {

std::string to_hex(const std::vector<uint8_t>& data) {
    static const char* kHex = "0123456789abcdef";
    std::string out;
    for (uint8_t byte : data) {
        out.push_back(kHex[byte >> 4]);
        out.push_back(kHex[byte & 0x0f]);
    }
    return out;
}

void test_key_generation_is_a_real_sharing() {
    tiger::TSSEngine engine(2, 3);
    const auto shares = engine.generateKeyShares();

    assert(shares.size() == 3);
    for (const auto& share : shares) {
        assert(share.share.size() == 32);
        // The public share is a compressed curve point, not a hash. A SHA-256
        // digest is also 32 bytes, so the length is not the check; a compressed
        // point is 33 bytes and starts with 0x02 or 0x03.
        assert(share.publicShare.size() == 33);
        assert(share.publicShare[0] == 0x02 || share.publicShare[0] == 0x03);
        assert(share.isActive);
        // It must decode as a point on the curve.
        assert(tigerwallet::secp256k1::decode_point(share.publicShare).has_value());
    }

    // The public key is the secret times the generator, so it is a valid point.
    const auto public_key = engine.getPublicKey();
    assert(public_key.size() == 33);
    assert(tigerwallet::secp256k1::decode_point(public_key).has_value());

    // It must not equal any share: publishing a share is publishing a piece of
    // the secret. The previous code set the public key to the polynomial's
    // constant term, which is the secret itself.
    for (const auto& share : shares) {
        assert(share.share != public_key);
        const auto digest = tigerwallet::secp256k1::keccak256(share.share);
        assert(std::vector<uint8_t>(digest.begin(), digest.end()) != public_key);
    }

    const std::string address = engine.getAddress();
    assert(address.size() == 42);
    assert(tigerwallet::address::is_valid_evm_address(address));
}

void test_any_threshold_subset_reconstructs_the_key() {
    // Every 2-of-3 subset must reconstruct the same secret. Under the old XOR
    // combination this would fail, because XOR depends on which shares happen
    // to be present rather than on interpolating a polynomial through them.
    tiger::TSSEngine engine(2, 3);
    const auto shares = engine.generateKeyShares();

    const std::vector<std::vector<size_t>> subsets = {{0, 1}, {0, 2}, {1, 2}};
    std::string reference;
    for (const auto& subset : subsets) {
        std::vector<tiger::KeyShare> selected;
        for (size_t index : subset) selected.push_back(shares[index]);

        const auto secret = engine.reconstructSecret(selected);
        assert(secret.size() == 32);
        const std::string hex = to_hex(secret);
        if (reference.empty()) {
            reference = hex;
        } else {
            assert(hex == reference);
        }
    }

    // The reconstructed key must correspond to the published public key. This
    // is the property that makes the sharing a sharing: the shares interpolate
    // to the key behind the public point.
    const auto public_key = engine.getPublicKey();
    std::vector<tiger::KeyShare> selected{shares[0], shares[1]};
    const auto secret = engine.reconstructSecret(selected);
    const auto secret_bn = tigerwallet::secp256k1::from_bytes(secret.data(), secret.size());
    const auto derived = tigerwallet::secp256k1::compress(
        tigerwallet::secp256k1::public_point(secret_bn.get()).get());
    assert(derived == public_key);

    // All three shares together must also give the same key.
    const auto all = engine.reconstructSecret(shares);
    assert(to_hex(all) == reference);
}

void test_below_threshold_is_refused() {
    tiger::TSSEngine engine(2, 3);
    const auto shares = engine.generateKeyShares();

    bool threw = false;
    try {
        std::vector<tiger::KeyShare> one{shares[0]};
        engine.reconstructSecret(one);
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);

    // A single share must not equal the secret, and must not equal any other
    // share.
    assert(to_hex(shares[0].share) != to_hex(shares[1].share));
    assert(to_hex(shares[1].share) != to_hex(shares[2].share));
}

void test_combine_requires_threshold() {
    tiger::TSSEngine engine(3, 5);
    const auto shares = engine.generateKeyShares();

    std::vector<tiger::SignatureShare> two;
    for (size_t i = 0; i < 2; ++i) {
        tiger::SignatureShare share;
        share.partyId = shares[i].partyId;
        share.share = shares[i].share;
        share.sessionId = {'a'};
        two.push_back(share);
    }

    const std::vector<uint8_t> hash(32, 0x11);
    bool threw = false;
    try {
        engine.combineShares(two, engine.getAddress(), hash);
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);  // Two shares of a 3-of-5 scheme must not sign.

    // Shares from different sessions are refused, because combining them would
    // mix nonces from separate signing rounds.
    std::vector<tiger::SignatureShare> mixed;
    for (size_t i = 0; i < 3; ++i) {
        tiger::SignatureShare share;
        share.partyId = shares[i].partyId;
        share.share = shares[i].share;
        share.sessionId = {static_cast<uint8_t>('a' + i)};
        mixed.push_back(share);
    }
    threw = false;
    try {
        engine.combineShares(mixed, engine.getAddress(), hash);
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);

    // A message hash that is not 32 bytes is refused.
    threw = false;
    try {
        std::vector<tiger::SignatureShare> three(mixed);
        for (auto& share : three) share.sessionId = {'a'};
        engine.combineShares(three, engine.getAddress(), std::vector<uint8_t>(31, 0));
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);

    // Duplicate party indices are refused rather than silently interpolated.
    threw = false;
    try {
        std::vector<tiger::SignatureShare> dup;
        for (size_t i = 0; i < 3; ++i) {
            tiger::SignatureShare share;
            share.partyId = 1;  // The same party three times.
            share.share = shares[i].share;
            share.sessionId = {'a'};
            dup.push_back(share);
        }
        engine.combineShares(dup, engine.getAddress(), hash);
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);
}

void test_verify_rejects_forgeries() {
    // The old verifySignature returned true whenever the signature bytes were
    // non-empty, so every case below would have been accepted.
    tiger::TSSEngine engine(2, 3);
    engine.generateKeyShares();

    const std::vector<uint8_t> hash(32, 0x22);

    tiger::MPCSignature zeros;
    zeros.signature = std::vector<uint8_t>(64, 0);
    assert(!engine.verifySignature(zeros, hash));

    tiger::MPCSignature garbage;
    garbage.signature = std::vector<uint8_t>(64, 0xab);
    assert(!engine.verifySignature(garbage, hash));

    // A 64-byte value where r is in range but s is not a valid scalar.
    tiger::MPCSignature out_of_range;
    out_of_range.signature = std::vector<uint8_t>(64, 0xff);
    assert(!engine.verifySignature(out_of_range, hash));

    // Wrong lengths are rejected.
    tiger::MPCSignature short_signature;
    short_signature.signature = std::vector<uint8_t>(63, 0xff);
    assert(!engine.verifySignature(short_signature, hash));
    assert(!engine.verifySignature(short_signature, std::vector<uint8_t>(31, 0)));

    // An empty signature. The old check returned false here and true for
    // everything else, which is the wrong way round.
    tiger::MPCSignature empty;
    assert(!engine.verifySignature(empty, hash));
}

void test_verification_accepts_a_real_signature() {
    // A signature made with the reconstructed key must verify against the
    // public key the engine publishes. This ties the sharing, the public key
    // derivation and verification together.
    tiger::TSSEngine engine(2, 3);
    const auto shares = engine.generateKeyShares();

    std::vector<tiger::KeyShare> selected{shares[0], shares[2]};
    const auto secret = engine.reconstructSecret(selected);
    const auto secret_bn = tigerwallet::secp256k1::from_bytes(secret.data(), secret.size());

    const auto hash = tigerwallet::secp256k1::keccak256(
        reinterpret_cast<const uint8_t*>("transfer 1 ETH"), 14);

    const auto signature = tigerwallet::secp256k1::sign(secret_bn.get(), hash);

    tiger::MPCSignature mpc_signature;
    mpc_signature.signature = signature.to_bytes();
    mpc_signature.messageHash = std::vector<uint8_t>(hash.begin(), hash.end());

    assert(engine.verifySignature(mpc_signature, mpc_signature.messageHash));

    // The same signature against a different message must fail.
    const auto other_hash = tigerwallet::secp256k1::keccak256(
        reinterpret_cast<const uint8_t*>("transfer 2 ETH"), 14);
    assert(!engine.verifySignature(mpc_signature,
                                   std::vector<uint8_t>(other_hash.begin(), other_hash.end())));

    // And a signature from a different wallet must fail.
    tiger::TSSEngine other_engine(2, 3);
    const auto other_shares = other_engine.generateKeyShares();
    const auto other_secret = other_engine.reconstructSecret({other_shares[0], other_shares[1]});
    const auto other_bn =
        tigerwallet::secp256k1::from_bytes(other_secret.data(), other_secret.size());
    const auto other_signature = tigerwallet::secp256k1::sign(other_bn.get(), hash);
    tiger::MPCSignature foreign;
    foreign.signature = other_signature.to_bytes();
    assert(!engine.verifySignature(foreign, mpc_signature.messageHash));
}

void test_transaction_hash_commits_to_every_field() {
    // The old serializeTransaction copied only the two addresses, so a
    // signature did not cover the amount, the chain, the gas or the nonce and
    // any of them could be changed after signing.
    tiger::MPCWallet wallet(2, 3);
    wallet.initialize("test");

    tiger::TransactionRequest base;
    base.from = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed";
    base.to = "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359";
    base.value = tigerwallet::math::U256::fromUint64(1000);
    base.gasLimit = tigerwallet::math::U256::fromUint64(21000);
    base.gasPrice = tigerwallet::math::U256::fromUint64(20000000000ULL);
    base.nonce = 1;
    base.chainId = 1;

    const auto base_hash = wallet.hashTransaction(wallet.serializeTransaction(base));

    tiger::TransactionRequest changed = base;
    changed.value = tigerwallet::math::U256::fromUint64(1001);
    assert(wallet.hashTransaction(wallet.serializeTransaction(changed)) != base_hash);

    changed = base;
    changed.chainId = 56;
    assert(wallet.hashTransaction(wallet.serializeTransaction(changed)) != base_hash);

    changed = base;
    changed.nonce = 2;
    assert(wallet.hashTransaction(wallet.serializeTransaction(changed)) != base_hash);

    changed = base;
    changed.gasPrice = tigerwallet::math::U256::fromUint64(30000000000ULL);
    assert(wallet.hashTransaction(wallet.serializeTransaction(changed)) != base_hash);

    changed = base;
    changed.to = "0x0000000000000000000000000000000000000001";
    assert(wallet.hashTransaction(wallet.serializeTransaction(changed)) != base_hash);

    // The hash is deterministic for identical requests.
    assert(wallet.hashTransaction(wallet.serializeTransaction(base)) == base_hash);

    const std::string address = wallet.getAddress();
    assert(address.size() == 42);
    assert(tigerwallet::address::is_valid_evm_address(address));
}

void test_policy_rules_are_evaluated() {
    // daily_limit and whitelist returned true unconditionally before, so a
    // policy that capped daily spend or restricted destinations approved
    // everything.
    tiger::PolicyEngine engine;

    tiger::TransactionRequest tx;
    tx.from = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed";
    tx.to = "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359";
    tx.value = tigerwallet::math::U256::fromUint64(500);

    // A whitelist admits only its own destination.
    tiger::PolicyRule whitelist;
    whitelist.type = "whitelist";
    whitelist.value = tx.to;
    assert(engine.evaluateRule(whitelist, tx));
    whitelist.value = "0x0000000000000000000000000000000000000001";
    assert(!engine.evaluateRule(whitelist, tx));

    // A blacklist blocks its own destination.
    tiger::PolicyRule blacklist;
    blacklist.type = "blacklist";
    blacklist.value = tx.to;
    assert(!engine.evaluateRule(blacklist, tx));

    // A transaction limit compares the amount.
    tiger::PolicyRule limit;
    limit.type = "tx_limit";
    limit.operator_ = "lte";
    limit.value = "1000";
    assert(engine.evaluateRule(limit, tx));
    limit.value = "100";
    assert(!engine.evaluateRule(limit, tx));

    // A daily limit accumulates across calls, so a second transaction that
    // would exceed the day's total is refused.
    tiger::PolicyEngine daily;
    tiger::PolicyRule daily_rule;
    daily_rule.type = "daily_limit";
    daily_rule.value = "800";
    assert(daily.evaluateRule(daily_rule, tx));   // 500 of 800
    assert(!daily.evaluateRule(daily_rule, tx));  // 1000 would exceed it

    // An unknown rule type is refused, so a typo does not silently disable a
    // policy.
    tiger::PolicyRule unknown;
    unknown.type = "not_a_real_rule";
    assert(!engine.evaluateRule(unknown, tx));
}

void test_audit_serialization_is_real() {
    tiger::AuditLogger logger;
    logger.log("0xabc", "sign", "a \"quoted\" detail", "0xdef", "127.0.0.1");

    const auto json = logger.exportAuditLog("0xabc", "json");
    const std::string json_text(json.begin(), json.end());
    // The previous implementation returned a zero-filled buffer, so none of
    // this content was present.
    assert(json_text.find("\"action\":\"sign\"") != std::string::npos);
    assert(json_text.find("\"wallet\":\"0xabc\"") != std::string::npos);
    assert(json_text.find("\"actor\":\"0xdef\"") != std::string::npos);
    // The embedded quote must be escaped, or the JSON is malformed and a field
    // can break out of its string.
    assert(json_text.find("\\\"quoted\\\"") != std::string::npos);

    const auto csv = logger.exportAuditLog("0xabc", "csv");
    const std::string csv_text(csv.begin(), csv.end());
    assert(csv_text.find("id,wallet,action") == 0);
    assert(csv_text.find("0xabc") != std::string::npos);
}

void test_party_transport() {
    // sendToParty claimed to have delivered a message while only recording it
    // locally. It now reports whether delivery actually happened.
    tiger::PartyCommunication comm;
    tiger::Party party;
    party.id = 1;
    party.name = "party-1";
    party.isOnline = true;
    comm.addParty(party);

    // With no transport installed, nothing is delivered.
    assert(!comm.sendToParty(1, {'h', 'i'}));

    // With one installed, the message reaches it.
    std::vector<tiger::Bytes> delivered;
    comm.setTransport([&delivered](tiger::PartyId, const tiger::Bytes& message) {
        delivered.push_back(message);
        return true;
    });
    assert(comm.sendToParty(1, {'h', 'i'}));
    assert(delivered.size() == 1);
    assert(delivered[0] == tiger::Bytes({'h', 'i'}));

    // An offline party is not delivered to.
    comm.updatePartyStatus(1, false);
    assert(!comm.sendToParty(1, {'b', 'y', 'e'}));
    assert(delivered.size() == 1);

    // An unknown party is not delivered to.
    assert(!comm.sendToParty(99, {'x'}));
}

}  // namespace

int main() {
    test_key_generation_is_a_real_sharing();
    test_any_threshold_subset_reconstructs_the_key();
    test_below_threshold_is_refused();
    test_combine_requires_threshold();
    test_verify_rejects_forgeries();
    test_verification_accepts_a_real_signature();
    test_transaction_hash_commits_to_every_field();
    test_policy_rules_are_evaluated();
    test_audit_serialization_is_real();
    test_party_transport();

    std::printf("enterprise_mpc: all checks passed\n");
    return 0;
}
