//! Tests for the gasless transaction service.

use super::*;

const SPONSOR: &str = "0x1111111111111111111111111111111111111111";
const USER: &str = "0x2222222222222222222222222222222222222222";

fn signing_key() -> k256::ecdsa::SigningKey {
    k256::ecdsa::SigningKey::from_slice(&[0x42u8; 32]).unwrap()
}

fn address_of(key: &k256::ecdsa::SigningKey) -> String {
    let point = key.verifying_key().to_encoded_point(false);
    let digest = eip191_digest(&point.as_bytes()[1..]);
    // Use the real keccak of the uncompressed key, not the digest above.
    let _ = digest;
    k256_address(key)
}

/// Derives the Ethereum address for a secp256k1 key.
fn k256_address(key: &k256::ecdsa::SigningKey) -> String {
    // k256 does not expose keccak; the address is the last 20 bytes of the
    // keccak-256 of the uncompressed public key. Derive it here with the
    // dependency already in the tree.
    let point = key.verifying_key().to_encoded_point(false);
    let hash = keccak256(&point.as_bytes()[1..]);
    format!("0x{}", hex::encode(&hash[12..]))
}

fn keccak256(data: &[u8]) -> [u8; 32] {
    use sha3::{Digest, Keccak256};
    Keccak256::digest(data).into()
}

fn tx(from: &str, nonce: u64) -> MetaTransaction {
    MetaTransaction {
        from: from.to_string(),
        to: "0x9999999999999999999999999999999999999999".to_string(),
        value: 0,
        data: Vec::new(),
        gas_limit: 100_000,
        relayer: String::new(),
        nonce,
        signature: Vec::new(),
    }
}

// ============================================================================
// Signing
// ============================================================================

#[test]
fn sign_message_produces_a_recoverable_signature() {
    let key = signing_key();
    let message = b"meta transaction payload";

    let signature = sign_message(message, &key).unwrap();
    assert_eq!(signature.len(), 65);

    // The signature recovers to the signer's own address. A hash of the
    // message, which is what this used to return, recovers to nothing.
    let recovered = recover_signer(message, &signature).unwrap();
    assert_eq!(format!("0x{}", hex::encode(recovered)), k256_address(&key));
}

#[test]
fn recover_signer_rejects_malformed_signatures() {
    let key = signing_key();
    let message = b"payload";
    let signature = sign_message(message, &key).unwrap();

    // Wrong length.
    assert!(recover_signer(message, &signature[..64]).is_none());
    assert!(recover_signer(message, &[]).is_none());

    // A different message does not recover to the same signer.
    let other = recover_signer(b"different", &signature);
    assert_ne!(
        other.map(|a| format!("0x{}", hex::encode(a))),
        Some(k256_address(&key))
    );

    // An invalid v.
    let mut bad = signature.clone();
    bad[64] = 7;
    assert!(recover_signer(message, &bad).is_none());
}

#[test]
fn signature_covers_the_message() {
    let key = signing_key();
    let a = sign_message(b"one", &key).unwrap();
    let b = sign_message(b"two", &key).unwrap();
    assert_ne!(a, b);
}

// ============================================================================
// Address parsing
// ============================================================================

#[test]
fn parse_address_accepts_only_20_byte_hex() {
    assert_eq!(parse_address(SPONSOR).unwrap()[0], 0x11);
    assert!(parse_address("0x1234").is_none());
    assert!(parse_address(SPONSOR.trim_start_matches("0x")).is_none());
    assert!(parse_address("0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz").is_none());
}

// ============================================================================
// Relayer
// ============================================================================

#[tokio::test]
async fn relayer_queues_and_consumes_the_nonce() {
    let mut relayer = RelayerService::new();

    let mut t = tx(USER, 0);
    t.signature = vec![1u8; 65];

    let id = relayer.submit_transaction(t.clone()).await.unwrap();
    assert!(relayer.get_pending(&id).is_some());
    assert_eq!(relayer.get_nonce(USER), 1, "the nonce must advance");

    // Replaying the same signed transaction is rejected. The previous
    // implementation generated a fresh id each time and never advanced the
    // nonce, so the same signed transaction could be submitted forever.
    assert!(relayer.submit_transaction(t).await.is_err());
}

#[tokio::test]
async fn relayer_rejects_a_stale_nonce() {
    let mut relayer = RelayerService::new();

    let mut t = tx(USER, 5);
    t.signature = vec![1u8; 65];
    assert!(relayer.submit_transaction(t).await.is_err());
}

#[tokio::test]
async fn relayer_rejects_an_unsigned_transaction() {
    let mut relayer = RelayerService::new();
    assert!(relayer
        .submit_transaction(tx(USER, 0))
        .await
        .is_err());
}

#[tokio::test]
async fn relayer_rejects_a_missing_from_or_to() {
    let mut relayer = RelayerService::new();

    let mut t = tx("", 0);
    t.signature = vec![1u8; 65];
    assert!(relayer.submit_transaction(t).await.is_err());

    let mut t = tx(USER, 0);
    t.to = String::new();
    t.signature = vec![1u8; 65];
    assert!(relayer.submit_transaction(t).await.is_err());
}

#[tokio::test]
async fn relayer_status_counts_failures() {
    let mut relayer = RelayerService::new();

    let mut a = tx(USER, 0);
    a.signature = vec![1u8; 65];
    let a = relayer.submit_transaction(a).await.unwrap();

    let mut b = tx(USER, 1);
    b.signature = vec![1u8; 65];
    let b = relayer.submit_transaction(b).await.unwrap();

    relayer.confirm_transaction(&a, true).await.unwrap();
    relayer.confirm_transaction(&b, false).await.unwrap();

    let status = relayer.get_status().await;
    // The old status reported executed == history.len() and failed == 0.
    assert_eq!(status.executed_transactions, 1);
    assert_eq!(status.failed_transactions, 1);
    assert_eq!(status.pending_transactions, 0);
    assert_eq!(relayer.failed_count(), 1);
}

#[tokio::test]
async fn relayer_cannot_confirm_an_unknown_transaction() {
    let mut relayer = RelayerService::new();
    assert!(relayer.confirm_transaction("nope", true).await.is_err());
}

#[tokio::test]
async fn relayer_registration() {
    let mut relayer = RelayerService::new();

    relayer.register("relayer-1").await.unwrap();
    assert_eq!(relayer.get_status().await.active_relayers, 1);

    relayer.deregister("relayer-1").await.unwrap();
    assert_eq!(relayer.get_status().await.active_relayers, 0);

    assert!(relayer.deregister("unknown").await.is_err());
}

// ============================================================================
// Sponsor
// ============================================================================

/// 1 gwei.
const GAS_PRICE: u64 = 1_000_000_000;
/// The cost of `sponsored_tx` at `GAS_PRICE`.
const COST: u64 = 100_000 * GAS_PRICE;

fn sponsored_tx() -> MetaTransaction {
    let mut t = tx(USER, 0);
    t.gas_limit = 100_000;
    t
}

#[tokio::test]
async fn sponsor_requires_a_configured_policy() {
    let mut s = SponsorService::new();
    s.add_sponsor(SPONSOR, 1_000_000_000_000).await.unwrap();

    // No policy has been set. The old implementation used a hardcoded 30 gwei
    // and sponsored regardless.
    assert!(s
        .sponsor(SPONSOR, USER, &sponsored_tx())
        .await
        .is_err());
}

#[tokio::test]
async fn sponsor_requires_an_allowance() {
    let mut s = SponsorService::new();
    s.set_policy(GAS_PRICE, 500_000).await.unwrap();
    s.add_sponsor(SPONSOR, 1_000_000_000_000_000).await.unwrap();

    // The sponsor is funded but has granted this user nothing.
    assert!(s.sponsor(SPONSOR, USER, &sponsored_tx()).await.is_err());

    // An allowance below the gas cost is still refused.
    s.set_allowance(SPONSOR, USER, COST - 1).await.unwrap();
    assert!(s.sponsor(SPONSOR, USER, &sponsored_tx()).await.is_err());

    // Exactly the cost is enough.
    s.set_allowance(SPONSOR, USER, COST).await.unwrap();
    assert!(s.sponsor(SPONSOR, USER, &sponsored_tx()).await.is_ok());
}

#[tokio::test]
async fn sponsor_rejects_an_unknown_sponsor() {
    let mut s = SponsorService::new();
    s.set_policy(GAS_PRICE, 500_000).await.unwrap();
    assert!(s.sponsor(SPONSOR, USER, &sponsored_tx()).await.is_err());
}

#[tokio::test]
async fn sponsor_deducts_from_both_budgets() {
    // A budget too small for even one sponsorship is refused.
    let mut s = SponsorService::new();
    s.set_policy(GAS_PRICE, 500_000).await.unwrap();
    s.add_sponsor(SPONSOR, COST - 1).await.unwrap();
    s.set_allowance(SPONSOR, USER, COST).await.unwrap();
    assert!(s.sponsor(SPONSOR, USER, &sponsored_tx()).await.is_err());

    // With a budget of five sponsorships, each one draws down the allowance.
    let mut s = SponsorService::new();
    s.set_policy(GAS_PRICE, 500_000).await.unwrap();
    s.add_sponsor(SPONSOR, COST * 5).await.unwrap();
    s.set_allowance(SPONSOR, USER, COST * 5).await.unwrap();

    for expected in (0..5).rev() {
        s.sponsor(SPONSOR, USER, &sponsored_tx()).await.unwrap();
        assert_eq!(s.remaining_allowance(SPONSOR, USER).await, COST * expected);
    }

    // The sixth has nothing left to spend.
    assert!(s.sponsor(SPONSOR, USER, &sponsored_tx()).await.is_err());
}

#[tokio::test]
async fn sponsor_rejects_a_transaction_over_the_gas_cap() {
    let mut s = SponsorService::new();
    s.set_policy(GAS_PRICE, 50_000).await.unwrap();
    s.add_sponsor(SPONSOR, COST * 10).await.unwrap();
    s.set_allowance(SPONSOR, USER, COST * 10).await.unwrap();

    let mut t = sponsored_tx();
    t.gas_limit = 100_000; // above the 50_000 cap
    assert!(matches!(
        s.sponsor(SPONSOR, USER, &t).await.unwrap_err(),
        GaslessError::GasLimitExceeded
    ));
}

#[tokio::test]
async fn sponsor_policy_rejects_zero_values() {
    let mut s = SponsorService::new();
    assert!(s.set_policy(0, 500_000).await.is_err());
    assert!(s.set_policy(1_000_000_000, 0).await.is_err());
    assert!(s.set_policy(1_000_000_000, 500_000).await.is_ok());
}

#[tokio::test]
async fn removing_a_sponsor_clears_its_allowances() {
    let mut s = SponsorService::new();
    s.add_sponsor(SPONSOR, 1_000_000).await.unwrap();
    s.set_allowance(SPONSOR, USER, 500_000).await.unwrap();
    assert_eq!(s.remaining_allowance(SPONSOR, USER).await, 500_000);

    s.remove_sponsor(SPONSOR).await.unwrap();
    assert_eq!(s.remaining_allowance(SPONSOR, USER).await, 0);
    assert!(s.remove_sponsor(SPONSOR).await.is_err());
}

// ============================================================================
// Forwarder
// ============================================================================

#[tokio::test]
async fn forwarder_trust_list() {
    let mut f = ForwarderService::new();

    assert!(!f.is_trusted("0xabc").await);
    f.add("0xabc").await.unwrap();
    assert!(f.is_trusted("0xabc").await);
    f.remove("0xabc").await.unwrap();
    assert!(!f.is_trusted("0xabc").await);
}

// ============================================================================
// End to end
// ============================================================================

#[tokio::test]
async fn service_signs_and_submits_a_meta_transaction() {
    let mut service = GaslessService::new(1);
    service.initialize("0x0000000000000000000000000000000000000abc".to_string()).await;
    let key = signing_key();

    let mut t = MetaTransaction {
        from: k256_address(&key),
        to: "0x9999999999999999999999999999999999999999".to_string(),
        value: 0,
        data: Vec::new(),
        gas_limit: 100_000,
        relayer: String::new(),
        nonce: 0,
        signature: Vec::new(),
    };

    service.sign_meta_tx(&mut t, &key).await.unwrap();
    assert_eq!(t.signature.len(), 65);

    let id = service.submit_meta_tx(t).await.unwrap();
    assert!(id.contains(&k256_address(&key)));
}

#[tokio::test]
async fn service_rejects_a_transaction_signed_by_the_wrong_key() {
    let mut service = GaslessService::new(1);
    service.initialize("0x0000000000000000000000000000000000000abc".to_string()).await;
    let key = signing_key();
    let other = k256::ecdsa::SigningKey::from_slice(&[0x43u8; 32]).unwrap();

    // The transaction claims to be from `key` but is signed by `other`.
    let mut t = MetaTransaction {
        from: k256_address(&key),
        to: "0x9999999999999999999999999999999999999999".to_string(),
        value: 0,
        data: Vec::new(),
        gas_limit: 100_000,
        relayer: String::new(),
        nonce: 0,
        signature: Vec::new(),
    };

    let err = service.sign_meta_tx(&mut t, &other).await;
    assert!(err.is_err(), "a mismatched signer must be rejected");
}
