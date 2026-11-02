//! Feldman verifiable secret sharing (VSS).
//!
//! Each polynomial coefficient is committed as a curve point `C_j = a_j * G`.
//! A participant can then check its own share without learning the secret:
//!
//! ```text
//! share_i * G  ==  sum_j ( i^j * C_j )
//! ```
//!
//! The previous implementation set `verification_key = SHA256(share_data)`,
//! which is just a hash of the share. It proves nothing: a dealer that handed
//! out shares from an unrelated polynomial would pass every check, because the
//! "commitment" is recomputed from the very value being verified. These are real
//! curve commitments, so a bad share is detected before it is ever used to sign.
//!
//! The  calls below come from 's bundled
//! generic-array 0.14; upgrading that would require a wider dependency bump.
#![allow(deprecated)]

use crypto_bigint::{Encoding, U256};
use k256::elliptic_curve::group::GroupEncoding;
use k256::elliptic_curve::PrimeField;
use k256::ProjectivePoint as K256Point;
use p256::ProjectivePoint as P256Point;

use crate::field_ed25519::{public_key_from_scalar, EdScalar};

/// Compressed SEC1 point size for secp256k1 / P-256.
const SEC1_COMPRESSED: usize = 33;

fn k256_scalar_from_u256(v: &U256) -> Option<k256::Scalar> {
    let bytes = v.to_be_bytes();
    Option::<k256::Scalar>::from(k256::Scalar::from_repr(bytes.into()))
}

fn p256_scalar_from_u256(v: &U256) -> Option<p256::Scalar> {
    let bytes = v.to_be_bytes();
    Option::<p256::Scalar>::from(p256::Scalar::from_repr(bytes.into()))
}

/// Commit to secp256k1 polynomial coefficients.
pub fn commit_secp256k1(coeffs: &[U256]) -> Result<Vec<Vec<u8>>, &'static str> {
    let mut out = Vec::with_capacity(coeffs.len());
    for c in coeffs {
        let s = k256_scalar_from_u256(c).ok_or("coefficient is not a valid secp256k1 scalar")?;
        let point = (K256Point::GENERATOR * s).to_affine();
        out.push(point.to_bytes().as_slice().to_vec());
    }
    Ok(out)
}

/// Verify a secp256k1 share against the commitments.
pub fn verify_share_secp256k1(
    index: u32,
    share: &U256,
    commitments: &[Vec<u8>],
) -> Result<bool, &'static str> {
    let share_scalar =
        k256_scalar_from_u256(share).ok_or("share is not a valid secp256k1 scalar")?;
    let lhs = K256Point::GENERATOR * share_scalar;

    let mut rhs = K256Point::IDENTITY;
    let mut x_pow = k256::Scalar::ONE;
    let x = k256::Scalar::from(index as u64);
    for c in commitments {
        if c.len() != SEC1_COMPRESSED {
            return Err("malformed commitment");
        }
        let mut buf = [0u8; SEC1_COMPRESSED];
        buf.copy_from_slice(c);
        let point = Option::<k256::AffinePoint>::from(k256::AffinePoint::from_bytes(&buf.into()))
            .ok_or("commitment is not a valid curve point")?;
        rhs += K256Point::from(point) * x_pow;
        x_pow *= x;
    }
    Ok(lhs == rhs)
}

/// Commit to P-256 polynomial coefficients.
pub fn commit_p256(coeffs: &[U256]) -> Result<Vec<Vec<u8>>, &'static str> {
    let mut out = Vec::with_capacity(coeffs.len());
    for c in coeffs {
        let s = p256_scalar_from_u256(c).ok_or("coefficient is not a valid P-256 scalar")?;
        let point = (P256Point::GENERATOR * s).to_affine();
        out.push(point.to_bytes().as_slice().to_vec());
    }
    Ok(out)
}

/// Verify a P-256 share against the commitments.
pub fn verify_share_p256(
    index: u32,
    share: &U256,
    commitments: &[Vec<u8>],
) -> Result<bool, &'static str> {
    let share_scalar = p256_scalar_from_u256(share).ok_or("share is not a valid P-256 scalar")?;
    let lhs = P256Point::GENERATOR * share_scalar;

    let mut rhs = P256Point::IDENTITY;
    let mut x_pow = p256::Scalar::ONE;
    let x = p256::Scalar::from(index as u64);
    for c in commitments {
        if c.len() != SEC1_COMPRESSED {
            return Err("malformed commitment");
        }
        let mut buf = [0u8; SEC1_COMPRESSED];
        buf.copy_from_slice(c);
        let point = Option::<p256::AffinePoint>::from(p256::AffinePoint::from_bytes(&buf.into()))
            .ok_or("commitment is not a valid curve point")?;
        rhs += P256Point::from(point) * x_pow;
        x_pow *= x;
    }
    Ok(lhs == rhs)
}

/// Commit to Ed25519 polynomial coefficients. Each commitment is the compressed
/// Edwards encoding of `a_j * B`, the same 32-byte form as an Ed25519 public key.
pub fn commit_ed25519(coeffs: &[EdScalar]) -> Vec<Vec<u8>> {
    coeffs
        .iter()
        .map(|c| public_key_from_scalar(c).to_vec())
        .collect()
}

/// Verify an Ed25519 share against the commitments.
pub fn verify_share_ed25519(
    index: u32,
    share: &EdScalar,
    commitments: &[Vec<u8>],
) -> Result<bool, &'static str> {
    use curve25519_dalek::constants::ED25519_BASEPOINT_TABLE;
    use curve25519_dalek::edwards::CompressedEdwardsY;
    use curve25519_dalek::traits::Identity;

    let share_point = &crate::field_ed25519::to_dalek_scalar(share) * ED25519_BASEPOINT_TABLE;

    let mut rhs = curve25519_dalek::edwards::EdwardsPoint::identity();
    let mut x_pow = EdScalar::from_limbs([1, 0, 0, 0]);
    let x = EdScalar::from_limbs([index as u64, 0, 0, 0]);
    for c in commitments {
        if c.len() != 32 {
            return Err("malformed commitment");
        }
        let mut buf = [0u8; 32];
        buf.copy_from_slice(c);
        let point = CompressedEdwardsY(buf)
            .decompress()
            .ok_or("commitment is not a valid Edwards point")?;
        rhs += point * crate::field_ed25519::to_dalek_scalar(&x_pow);
        x_pow = x_pow.mul(&x);
    }
    Ok(share_point == rhs)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::field_ed25519::EdScalar;

    #[test]
    fn secp256k1_share_verification_accepts_genuine_and_rejects_forged() {
        let coeffs = [U256::from(11u64), U256::from(22u64), U256::from(33u64)];
        let commitments = commit_secp256k1(&coeffs).unwrap();

        let share = crate::field::eval_polynomial(&coeffs, 3);
        assert!(verify_share_secp256k1(3, &share, &commitments).unwrap());

        // A share from a different polynomial must be rejected.
        let forged = crate::field::eval_polynomial(&[U256::from(11u64), U256::from(22u64), U256::from(34u64)], 3);
        assert!(!verify_share_secp256k1(3, &forged, &commitments).unwrap());

        // The same share presented at the wrong index must be rejected.
        assert!(!verify_share_secp256k1(4, &share, &commitments).unwrap());
    }

    #[test]
    fn p256_share_verification_accepts_genuine_and_rejects_forged() {
        let coeffs = [U256::from(7u64), U256::from(9u64)];
        let commitments = commit_p256(&coeffs).unwrap();
        let share = crate::field_p256::eval_polynomial(&coeffs, 5);
        assert!(verify_share_p256(5, &share, &commitments).unwrap());

        let forged = crate::field_p256::eval_polynomial(&[U256::from(7u64), U256::from(10u64)], 5);
        assert!(!verify_share_p256(5, &forged, &commitments).unwrap());
    }

    #[test]
    fn ed25519_share_verification_accepts_genuine_and_rejects_forged() {
        let secret = EdScalar::random();
        let coeffs = [secret, EdScalar::random()];
        let commitments = commit_ed25519(&coeffs);

        let share = EdScalar::eval_polynomial(&coeffs, 2);
        assert!(verify_share_ed25519(2, &share, &commitments).unwrap());

        let forged = EdScalar::eval_polynomial(&[secret, EdScalar::random()], 2);
        assert!(!verify_share_ed25519(2, &forged, &commitments).unwrap());
    }

    #[test]
    fn commitment_is_a_real_curve_point_not_a_hash() {
        let coeffs = [U256::from(5u64)];
        let c = commit_secp256k1(&coeffs).unwrap();
        // The commitment for a_0 = 5 must equal the known generator multiple.
        let five = k256::Scalar::from(5u64);
        let expected = (K256Point::GENERATOR * five).to_affine().to_bytes().as_slice().to_vec();
        assert_eq!(c[0], expected);
    }

    #[test]
    fn malformed_commitments_are_rejected() {
        let short = vec![vec![0u8; 5]];
        assert!(verify_share_secp256k1(1, &U256::from(1u64), &short).is_err());
        assert!(verify_share_ed25519(1, &EdScalar::random(), &short).is_err());
    }
}
