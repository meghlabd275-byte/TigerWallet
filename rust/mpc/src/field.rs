//! Internal secp256k1 scalar-field arithmetic (256-bit) for Shamir/Feldman VSS.
//!
//! Uses crypto_bigint's Montgomery-form modular arithmetic over the
//! secp256k1 curve order n, so full 256-bit secrets round-trip correctly
//! through polynomial evaluation and Lagrange interpolation.

use crypto_bigint::{Encoding, U256};
use crypto_bigint::modular::runtime_mod::{DynResidue, DynResidueParams};
use k256::elliptic_curve::Curve;
use k256::Secp256k1;

type Params = DynResidueParams<{ U256::LIMBS }>;

fn params() -> Params {
    DynResidueParams::new(&Secp256k1::ORDER)
}

/// Decode up to 32 little-endian bytes into a reduced secp256k1 scalar.
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

/// Encode a reduced scalar as 32 big-endian bytes (secp256k1 key material format).
pub fn scalar_to_be_bytes(s: U256) -> [u8; 32] {
    s.to_be_bytes()
}

/// Evaluate the polynomial coeffs[0] + coeffs[1]*x + ... at the integer x,
/// reduced modulo the secp256k1 curve order.
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

/// Lagrange interpolation at x = 0 over the secp256k1 scalar field.
///
/// xs are the distinct x-coordinates and ys the corresponding (reduced)
/// y-values. Returns the interpolated secret (constant term of the polynomial).
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
            // L_i(0) = prod_{j != i} (0 - x_j) / (x_i - x_j) = prod (-x_j) / (x_i - x_j)
            num = num.mul(&x_j.neg());
            den = den.mul(&x_i.sub(&x_j));
        }

        let (den_inv, _is_some) = den.invert();
        let basis = num.mul(&den_inv);
        secret = secret.add(&y_i.mul(&basis));
    }

    secret.retrieve()
}

