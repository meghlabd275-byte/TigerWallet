//! Tests for the institutional custody service.

use super::*;

fn owners() -> Vec<String> {
    vec!["owner-1".to_string(), "owner-2".to_string()]
}

fn signers() -> Vec<String> {
    vec![
        "signer-1".to_string(),
        "signer-2".to_string(),
        "signer-3".to_string(),
    ]
}

async fn service() -> (InstitutionalCustodyService, String) {
    let service = InstitutionalCustodyService::new(1);
    let wallet = service
        .create_wallet(
            "Treasury".to_string(),
            WalletType::Institutional,
            owners(),
            2,
            signers(),
        )
        .await
        .unwrap();
    (service, wallet.address)
}

// ============================================================================
// Key material
// ============================================================================

#[tokio::test]
async fn created_wallet_has_key_material() {
    let (service, address) = service().await;

    // The key generated at creation must actually be retrievable. Previously
    // generate_key_share returned a share and stored nothing, so every
    // signing call failed with "No key pair".
    let mpc = service.mpc.read().await;
    assert!(
        mpc.public_key_for(&address).is_some(),
        "no key material for {}",
        address
    );
}

#[tokio::test]
async fn wallet_address_is_derived_from_the_key() {
    let (service, address) = service().await;

    let mpc = service.mpc.read().await;
    // The address must equal the address of the key that will sign for it.
    // It used to be a SHA-256 hash of the owner list and the current time,
    // which no key could ever produce.
    assert_eq!(mpc.address_for(&address).as_deref(), Some(address.as_str()));

    assert!(address.starts_with("0x"));
    assert_eq!(address.len(), 42);
}

#[tokio::test]
async fn two_wallets_get_different_keys_and_addresses() {
    let (service, first) = service().await;
    let second = service
        .create_wallet(
            "Second".to_string(),
            WalletType::Institutional,
            owners(),
            2,
            signers(),
        )
        .await
        .unwrap()
        .address;

    assert_ne!(first, second);
}

// ============================================================================
// Signing and verification
// ============================================================================

#[tokio::test]
async fn a_signature_verifies_against_the_wallet_key() {
    let (service, address) = service().await;
    let message = b"release 100 USDC to vendor";

    let signature = service
        .generate_mpc_signature(&address, message)
        .await
        .unwrap();

    // A real ECDSA P-256 signature is 64 bytes; the old code returned a
    // SHA-256 hash of the message, which is not a signature.
    assert_eq!(signature.len(), 64);

    assert!(service
        .verify_mpc_signature(&address, message, &signature)
        .await
        .unwrap());
}

#[tokio::test]
async fn verification_rejects_a_wrong_message() {
    let (service, address) = service().await;

    let signature = service
        .generate_mpc_signature(&address, b"release 100 USDC")
        .await
        .unwrap();

    // The old verify() returned Ok(true) for any non-empty byte string, so a
    // signature over a different message was accepted.
    assert!(!service
        .verify_mpc_signature(&address, b"release 1000000 USDC", &signature)
        .await
        .unwrap());
}

#[tokio::test]
async fn verification_rejects_a_forged_signature() {
    let (service, address) = service().await;

    let signature = service
        .generate_mpc_signature(&address, b"message")
        .await
        .unwrap();

    // Flip a byte in the signature.
    let mut forged = signature.clone();
    forged[0] ^= 0xff;
    assert!(!service
        .verify_mpc_signature(&address, b"message", &forged)
        .await
        .unwrap());

    // An empty signature is not valid either.
    assert!(!service
        .verify_mpc_signature(&address, b"message", &[])
        .await
        .unwrap());
}

#[tokio::test]
async fn verification_rejects_an_unknown_wallet() {
    let (service, _) = service().await;
    let err = service
        .verify_mpc_signature(
            "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
            b"message",
            &[0u8; 64],
        )
        .await;
    assert!(err.is_err(), "an unknown wallet must not verify");
}

#[tokio::test]
async fn a_signature_does_not_verify_under_another_wallet() {
    let (service, first) = service().await;
    let second = service
        .create_wallet(
            "Second".to_string(),
            WalletType::Institutional,
            owners(),
            2,
            signers(),
        )
        .await
        .unwrap()
        .address;
    let message = b"transfer";

    let signature = service
        .generate_mpc_signature(&first, message)
        .await
        .unwrap();

    assert!(!service
        .verify_mpc_signature(&second, message, &signature)
        .await
        .unwrap());
}

// ============================================================================
// Share aggregation
// ============================================================================

#[tokio::test]
async fn combine_shares_fails_closed_on_short_shares() {
    let (service, address) = service().await;

    // A raw key share is not a partial signature. The old code returned the
    // first share as the combined signature and reported success.
    let mut mpc = service.mpc.write().await;
    let share = mpc
        .generate_key_share(&address)
        .await
        .unwrap();
    drop(mpc);

    let mut mpc = service.mpc.write().await;
    mpc.add_signature_share(&address, "req-1", "party-1", share.share.clone())
        .await
        .unwrap();
    let result = mpc.combine_shares(&address, "req-1", b"message").await;
    assert!(
        result.is_err(),
        "a key share must not be accepted as a combined signature"
    );
}

// ============================================================================
// Wallet validation
// ============================================================================

#[tokio::test]
async fn wallet_creation_validates_its_arguments() {
    let service = InstitutionalCustodyService::new(1);

    assert!(service
        .create_wallet(
            "No owners".to_string(),
            WalletType::Institutional,
            Vec::new(),
            1,
            signers(),
        )
        .await
        .is_err());

    assert!(service
        .create_wallet(
            "Zero threshold".to_string(),
            WalletType::Institutional,
            owners(),
            0,
            signers(),
        )
        .await
        .is_err());

    assert!(service
        .create_wallet(
            "Threshold above signers".to_string(),
            WalletType::Institutional,
            owners(),
            4,
            signers(),
        )
        .await
        .is_err());
}

// ============================================================================
// RBAC
// ============================================================================

#[tokio::test]
async fn owner_and_signer_roles_are_assigned_at_creation() {
    let (service, address) = service().await;

    assert!(service
        .has_role(&address, "owner-1", Role::Owner)
        .await);
    assert!(service
        .has_role(&address, "signer-1", Role::Signer)
        .await);
    assert!(!service
        .has_role(&address, "stranger", Role::Owner)
        .await);
}

#[tokio::test]
async fn roles_can_be_granted_and_revoked() {
    let (service, address) = service().await;

    service
        .grant_role(&address, "new-signer", Role::Signer)
        .await
        .unwrap();
    assert!(service
        .has_role(&address, "new-signer", Role::Signer)
        .await);

    service
        .revoke_role(&address, "new-signer")
        .await
        .unwrap();
    assert!(!service
        .has_role(&address, "new-signer", Role::Signer)
        .await);
}
