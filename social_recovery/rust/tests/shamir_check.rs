// Shamir threshold sharing properties.
//
// The field arithmetic here is the basis of wallet recovery, so these check the
// properties that make the scheme safe rather than only the happy path: every
// threshold-sized subset must agree, below-threshold input must be refused
// outright, and shares must not leak the secret.

use social_recovery::ThresholdSignatureScheme;

fn secret() -> Vec<u8> {
    (0u8..32).collect()
}

#[test]
fn t2_of_3_roundtrip() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(2, 3);
    let shares = scheme.generate_shares(&s, 3).expect("shares");
    assert_eq!(shares.len(), 3);
    let r = scheme.reconstruct_secret(&shares[0..2]).expect("reconstruct");
    assert_eq!(r, s, "2 of 3 must recover");
}

#[test]
fn t3_of_5_roundtrip() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(3, 5);
    let shares = scheme.generate_shares(&s, 5).expect("shares");
    let r = scheme.reconstruct_secret(&shares[0..3]).expect("reconstruct");
    assert_eq!(r, s, "3 of 5 must recover");
}

#[test]
fn every_threshold_subset_agrees() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(3, 5);
    let shares = scheme.generate_shares(&s, 5).expect("shares");

    // All 10 ways of choosing 3 of 5 shares must reconstruct the same secret.
    for a in 0..5 {
        for b in (a + 1)..5 {
            for c in (b + 1)..5 {
                let subset = vec![shares[a].clone(), shares[b].clone(), shares[c].clone()];
                let r = scheme.reconstruct_secret(&subset).expect("reconstruct");
                assert_eq!(r, s, "subset {a},{b},{c} must recover the secret");
            }
        }
    }
}

#[test]
fn below_threshold_is_refused() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(3, 5);
    let shares = scheme.generate_shares(&s, 5).expect("shares");

    // Two shares cannot reconstruct a 3-of-5 secret, so the call must fail
    // rather than return a wrong value the caller might act on.
    assert!(scheme.reconstruct_secret(&shares[0..2]).is_err());
    assert!(scheme.reconstruct_secret(&shares[0..1]).is_err());
    assert!(scheme.reconstruct_secret(&[]).is_err());
}

#[test]
fn duplicate_shares_are_refused() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(2, 3);
    let shares = scheme.generate_shares(&s, 3).expect("shares");

    // The same share twice is not two shares; interpolation would divide by
    // zero. This must be reported, not papered over.
    let dup = vec![shares[0].clone(), shares[0].clone()];
    assert!(scheme.reconstruct_secret(&dup).is_err());
}

#[test]
fn malformed_shares_are_refused() {
    let scheme = ThresholdSignatureScheme::new(2, 3);
    assert!(scheme.reconstruct_secret(&[vec![1], vec![2]]).is_err());
    // Mismatched y lengths.
    assert!(scheme
        .reconstruct_secret(&[vec![1, 9, 9, 9], vec![2, 9]])
        .is_err());
}

#[test]
fn shares_must_not_equal_the_secret() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(2, 3);
    let shares = scheme.generate_shares(&s, 3).expect("shares");
    for sh in &shares {
        assert_ne!(sh[1..], s[..], "a share must not be the secret itself");
    }
}

#[test]
fn shares_must_be_distinct() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(2, 3);
    let shares = scheme.generate_shares(&s, 3).expect("shares");
    assert_ne!(shares[0], shares[1]);
    assert_ne!(shares[1], shares[2]);
    assert_ne!(shares[0], shares[2]);
}

#[test]
fn share_count_is_exact() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(2, 5);
    let shares = scheme.generate_shares(&s, 5).expect("shares");
    assert_eq!(shares.len(), 5);
    // Each share is the x coordinate plus one byte per secret byte.
    for sh in &shares {
        assert_eq!(sh.len(), s.len() + 1);
    }
}

#[test]
fn insufficient_share_count_is_rejected() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(4, 5);
    assert!(scheme.generate_shares(&s, 3).is_err());
}

#[test]
fn empty_secret_is_rejected() {
    let scheme = ThresholdSignatureScheme::new(2, 3);
    assert!(scheme.generate_shares(&[], 3).is_err());
}

#[test]
fn two_generations_produce_different_shares() {
    let s = secret();
    let scheme = ThresholdSignatureScheme::new(2, 3);
    let a = scheme.generate_shares(&s, 3).expect("shares");
    let b = scheme.generate_shares(&s, 3).expect("shares");
    // Fresh random coefficients each time; identical shares would mean the
    // randomness is not being applied.
    assert_ne!(a, b, "shares must be randomised per generation");
}
