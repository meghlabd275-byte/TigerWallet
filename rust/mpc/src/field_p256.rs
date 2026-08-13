//! Internal P-256 (secp256r1) scalar-field arithmetic (256-bit) for Shamir.
//!
//! Mirrors `field.rs` but over the NIST P-256 curve order, so P-256 secrets
//! round-trip correctly through polynomial evaluation and Lagrange
//! interpolation.

use crypto_bigint::{Encoding, U256};
use crypto_bigint::modular::runtime_mod::{DynResidue, DynResidueParams};
use p256::elliptic_curve::Curve;
use p256::NistP256;

type Params = DynResidueParams<{ U256::LIMBS }>;

fn params() -> Params {
    DynResidueParams::new(&NistP256::ORDER)
}

/// Decode up to 32 little-endian bytes into a reduced P-256 scalar.
pub fn bytes_to_scalar(bytes: &[u8]) -> U256 {
    let mut arr = [0u8; 32];
    let n = bytes.len().min(32);
    arr[..n].copy_from_slice(&bytes[..n]);
    U256::from_le_bytes(arr)
}

/// Encode a reduced scalar as 32 little-endian bytes (fixed-width share format).
pub fn scalar_to_le_bytes(s: U256) -> [u8; 32] {
    s.to_le_bytes()
}

/// Encode a reduced scalar as 32 big-endian bytes (P-256 key material format).
pub fn scalar_to_be_bytes(s: U256) -> [u8; 32] {
    s.to_be_bytes()
}

/// Evaluate the polynomial coeffs[0] + coeffs[1]*x + ... at the integer x,
/// reduced modulo the P-256 curve order.
pub fn eval_polynomial(coeffs: &[U256], x: u32) -> U256 {
    let p = params();
    let x_r = DynResidue::new(&U256::from_word(x as u64), p);
    let mut result = DynResidue::zero(p);
    let mut x_pow = DynResidue::one(p);
    for coeff in coeffs {
        let term = DynResidue::new(coeff, p).mul(&x_pow);
        result = result.add(&term);
        x_pow = x_pow.mul(&x_r);
    }
    result.retrieve()
}

/// Lagrange interpolation at x = 0 over the P-256 scalar field.
pub fn lagrange_at_zero(xs: &[u32], ys: &[U256]) -> U256 {
    let p = params();
    let mut secret = DynResidue::zero(p);

    for i in 0..xs.len() {
        let y_i = DynResidue::new(&ys[i], p);
        let x_i = DynResidue::new(&U256::from_word(xs[i] as u64), p);

        let mut num = DynResidue::one(p);
        let mut den = DynResidue::one(p);
        for j in 0..xs.len() {
            if i == j {
                continue;
            }
            let x_j = DynResidue::new(&U256::from_word(xs[j] as u64), p);
            num = num.mul(&x_j.neg());
            den = den.mul(&x_i.sub(&x_j));
        }

        let (den_inv, _is_some) = den.invert();
        let basis = num.mul(&den_inv);
        secret = secret.add(&y_i.mul(&basis));
    }

    secret.retrieve()
}
