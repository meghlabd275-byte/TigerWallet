//! Regression tests for the defects found in the original MPC implementation.
//!
//! Each test fails against the previous code and passes now:
//!  1. Ed25519 shares did not reconstruct the key (XOR "sharing" of a clamped
//!     scalar).
//!  2. secp256k1 signing ignored each share's x-coordinate, so any subset other
//!     than the first `threshold` shares produced an invalid signature.
//!  3. Backups stored the shares in plaintext next to an HMAC of the same bytes.

use tiger_mpc_wallet::key_gen::generate_key_shares_with_passphrase;
use tiger_mpc_wallet::key_sharing::{backup_key, recover_key, restore_from_backup};
use tiger_mpc_wallet::signing::{sign_with_shares, verify, SigningShare};
use tiger_mpc_wallet::{CurveType, MpcConfig};

const PASS: &[u8] = b"regression-test-passphrase";

fn setup(curve: CurveType, threshold: u32, total: u32) -> (tiger_mpc_wallet::KeyGenResult, MpcConfig) {
    let config = MpcConfig {
        threshold,
        total_shares: total,
        curve,
        key_id: String::new(),
    };
    let result = generate_key_shares_with_passphrase(&config, b"entropy", PASS).unwrap();
    (result, config)
}

fn shares_at(result: &tiger_mpc_wallet::KeyGenResult, idx: &[usize]) -> Vec<SigningShare> {
    idx.iter()
        .map(|&i| SigningShare {
            index: result.shares[i].index,
            share_data: result.shares[i].share_data.clone(),
            commitments: result.shares[i].commitments.clone(),
        })
        .collect()
}

/// Bug 1: Ed25519 shares could not reconstruct the key.
#[test]
fn ed25519_shares_reconstruct_the_key() {
    let (result, _) = setup(CurveType::Ed25519, 2, 3);

    let recovered = recover_key(&result.shares[..2], 2, CurveType::Ed25519).unwrap();
    assert_eq!(
        recovered, result.master_seed,
        "the recovered seed must equal the seed the public key was derived from"
    );

    // And a signature built from the shares must verify against the public key.
    let shares = shares_at(&result, &[0, 1]);
    let sig = sign_with_shares(&shares, &result.public_key, b"transfer 1 BTC", &MpcConfig {
        threshold: 2,
        total_shares: 3,
        curve: CurveType::Ed25519,
        key_id: String::new(),
    })
    .unwrap();
    assert!(
        verify(&sig.signature, &result.public_key, b"transfer 1 BTC", CurveType::Ed25519).unwrap(),
        "an Ed25519 threshold signature must verify against the published public key"
    );
}

/// Bug 2: signing ignored share indices, so non-prefix subsets failed.
#[test]
fn non_prefix_subset_signs_for_every_curve() {
    for curve in [CurveType::Secp256k1, CurveType::P256, CurveType::Ed25519] {
        let (result, _) = setup(curve, 2, 3);
        let config = MpcConfig {
            threshold: 2,
            total_shares: 3,
            curve,
            key_id: String::new(),
        };

        // Shares 1 and 3 (indices 0 and 2). The old code interpolated them as if
        // they were indices 1 and 2.
        let shares = shares_at(&result, &[0, 2]);
        let msg = b"send 1 ETH";
        let sig = sign_with_shares(&shares, &result.public_key, msg, &config)
            .unwrap_or_else(|e| panic!("{:?} shares(1,3) failed to sign: {}", curve, e));
        assert!(
            verify(&sig.signature, &result.public_key, msg, curve).unwrap(),
            "{:?}: any t-of-n subset must produce a valid signature",
            curve
        );
    }
}

/// Bug 3: backups were plaintext shares plus an HMAC of those same bytes.
#[test]
fn backups_are_encrypted_not_just_tagged() {
    let (result, _) = setup(CurveType::Secp256k1, 2, 3);

    let backup = backup_key(
        &result.shares,
        &result.key_id,
        2,
        3,
        CurveType::Secp256k1,
        PASS,
    )
    .unwrap();

    for s in &result.shares {
        assert!(
            !backup.windows(s.share_data.len()).any(|w| w == &s.share_data[..]),
            "share bytes must not appear in the backup"
        );
    }

    // The restored shares must still reconstruct the real key.
    let restored = restore_from_backup(&backup, PASS).unwrap();
    let secret = recover_key(&restored, 2, CurveType::Secp256k1).unwrap();
    assert_eq!(secret, result.master_seed);
}

/// A share that is not on the committed polynomial is rejected, not used.
#[test]
fn forged_share_is_rejected() {
    let (result, _) = setup(CurveType::Secp256k1, 2, 3);
    let config = MpcConfig {
        threshold: 2,
        total_shares: 3,
        curve: CurveType::Secp256k1,
        key_id: String::new(),
    };

    let mut forged = shares_at(&result, &[0, 1]);
    forged[1].share_data[0] ^= 0x01;

    assert!(
        sign_with_shares(&forged, &result.public_key, b"msg", &config).is_err(),
        "a share failing VSS verification must never be used for signing"
    );
}

/// The published public key is checked against the reconstructed key.
#[test]
fn signing_with_the_wrong_public_key_fails() {
    let (a, _) = setup(CurveType::Secp256k1, 2, 3);
    let (b, _) = setup(CurveType::Secp256k1, 2, 3);
    let config = MpcConfig {
        threshold: 2,
        total_shares: 3,
        curve: CurveType::Secp256k1,
        key_id: String::new(),
    };

    let shares = shares_at(&a, &[0, 1]);
    assert!(
        sign_with_shares(&shares, &b.public_key, b"msg", &config).is_err(),
        "signing must fail rather than emit a signature no verifier accepts"
    );
}
