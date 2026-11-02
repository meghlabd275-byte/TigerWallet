//! Scalar-field arithmetic modulo the Ed25519 group order
//! `L = 2^252 + 27742317777372353535851937790883648493`.
//!
//! Shamir secret sharing over Ed25519 must be done modulo the group order, not
//! modulo an arbitrary 256-bit modulus. `ed25519-dalek` derives its scalar as
//! `clamp(SHA512(seed)[0..32])`, which is **not** uniformly distributed over
//! `[0, L)`: its top three bits are forced to zero and bit 254 is forced to
//! one, so it always lies in `[2^254, 2^255)`. Reducing such a value mod L
//! changes it, and any share arithmetic done mod 2^256 produces shares whose
//! Lagrange combination is not the key. Sharing the *seed* modulo L keeps the
//! reconstruction exact, because the seed is the actual secret being split.
//!
//! `curve25519-dalek` exposes the field arithmetic but not the scalar type, so
//! the scalar type is implemented here as four 64-bit limbs in Montgomery form
//! with the standard CIOS multiplication and a constant-time conditional
//! subtraction. `L` and `2^256 mod L` are the published constants for the
//! Ed25519 group order.

use curve25519_dalek::constants::ED25519_BASEPOINT_TABLE;
use curve25519_dalek::scalar::Scalar;
use curve25519_dalek::traits::Identity;
use rand::rngs::OsRng;
use rand::RngCore;

/// L = 2^252 + 27742317777372353535851937790883648493, little-endian limbs.
const L: [u64; 4] = [
    0x5812631a5cf5d3ed,
    0x14def9dea2f79cd6,
    0x0000000000000000,
    0x1000000000000000,
];

/// R = 2^256 mod L, the Montgomery radix in limb form.
const R: [u64; 4] = [
    0xd6ec31748d98951d,
    0xc6ef5bf4737dcf70,
    0xfffffffffffffffe,
    0x0fffffffffffffff,
];

/// R^2 mod L, used to convert into Montgomery form.
const R2: [u64; 4] = [
    0xa40611e3449c0f01,
    0xd00e1ba768859347,
    0xceec73d217f5be65,
    0x399411b7c309a3d,
];

/// floor((2^256 - 1) / L). Reducing a full-width value needs at most this many
/// conditional subtractions.
const REDUCE_STEPS: usize = 15;

/// A scalar in Montgomery form.
#[derive(Clone, Copy, Debug)]
pub struct EdScalar {
    n: [u64; 4],
}

/// Add two 4-limb numbers, returning the sum and the carry out.
#[inline]
fn adc(a: u64, b: u64, carry: u64) -> (u64, u64) {
    let sum = (a as u128) + (b as u128) + (carry as u128);
    (sum as u64, (sum >> 64) as u64)
}

/// Subtract with borrow; returns the difference and the borrow out.
#[inline]
fn sbb(a: u64, b: u64, borrow: u64) -> (u64, u64) {
    let diff = (a as u128).wrapping_sub(b as u128).wrapping_sub(borrow as u128);
    (diff as u64, ((diff >> 64) as u64) & 1)
}

/// Multiply a 4-limb number by a 64-bit word, adding a carry word.
#[inline]
fn mac(acc: u64, a: u64, b: u64, carry: u64) -> (u64, u64) {
    let product = (a as u128) * (b as u128) + (acc as u128) + (carry as u128);
    (product as u64, (product >> 64) as u64)
}

impl EdScalar {
    pub const ZERO: EdScalar = EdScalar { n: [0, 0, 0, 0] };

    /// Build from raw little-endian limbs. Input must already be reduced mod L.
    pub fn from_limbs(n: [u64; 4]) -> Self {
        let mut out = EdScalar { n };
        out.reduce();
        out
    }

    /// Build from 32 little-endian bytes, reducing mod L.
    pub fn from_le_bytes(bytes: &[u8; 32]) -> Self {
        let mut n = [0u64; 4];
        for (i, limb) in n.iter_mut().enumerate() {
            let mut buf = [0u8; 8];
            buf.copy_from_slice(&bytes[i * 8..i * 8 + 8]);
            *limb = u64::from_le_bytes(buf);
        }
        Self::from_limbs(n)
    }

    /// Encode as 32 little-endian bytes of the canonical representative.
    pub fn to_le_bytes(&self) -> [u8; 32] {
        let mut out = [0u8; 32];
        for (i, limb) in self.n.iter().enumerate() {
            out[i * 8..i * 8 + 8].copy_from_slice(&limb.to_le_bytes());
        }
        out
    }

    /// Reject a share that is not already a canonical field element.
    pub fn is_canonical(bytes: &[u8; 32]) -> bool {
        let s = Self::from_le_bytes(bytes);
        s.to_le_bytes() == *bytes
    }

    /// Conditional subtraction of L. `sub` must be 0 or 1.
    #[inline]
    fn conditional_sub_l(&mut self, mut sub: u64) {
        let mut borrow = 0u64;
        let mut t = [0u64; 4];
        for i in 0..4 {
            let (d, b) = sbb(self.n[i], L[i], borrow);
            t[i] = d;
            borrow = b;
        }
        // borrow == 0 means self >= L, so keep the subtracted value.
        sub &= 1 - borrow;
        let mask = 0u64.wrapping_sub(sub);
        for i in 0..4 {
            self.n[i] = (self.n[i] & !mask) | (t[i] & mask);
        }
    }

    /// Full reduction mod L. A 256-bit input can exceed L by up to
    /// floor((2^256 - 1) / L) = 15 multiples, so that many conditional
    /// subtractions are required; fewer would leave an unreduced value.
    pub fn reduce(&mut self) {
        for _ in 0..REDUCE_STEPS {
            self.conditional_sub_l(1);
        }
    }

    /// Return self >= L as 0/1 without branching.
    fn ge_l(&self) -> u64 {
        let mut borrow = 0u64;
        for i in 0..4 {
            let (_, b) = sbb(self.n[i], L[i], borrow);
            borrow = b;
        }
        1 - borrow
    }

    /// Montgomery multiplication (CIOS).
    fn mont_mul(&self, other: &EdScalar) -> EdScalar {
        const INV: u64 = 0xd2b51da312547e1b; // -L^{-1} mod 2^64

        let mut t = [0u64; 6];

        for i in 0..4 {
            let mut carry = 0u64;

            // t += a[i] * b
            for j in 0..4 {
                let (lo, hi) = mac(t[j], self.n[i], other.n[j], carry);
                t[j] = lo;
                carry = hi;
            }
            let (t4, c4) = adc(t[4], carry, 0);
            t[4] = t4;
            t[5] = t[5].wrapping_add(c4);

            // m = t[0] * INV mod 2^64; t = (t + m * L) / 2^64
            let m = t[0].wrapping_mul(INV);
            let (_, mut carry) = mac(t[0], m, L[0], 0);
            for j in 1..4 {
                let (lo, hi) = mac(t[j], m, L[j], carry);
                t[j - 1] = lo;
                carry = hi;
            }
            let (t4, c4) = adc(t[4], carry, 0);
            t[3] = t4;
            t[4] = t[5].wrapping_add(c4);
            t[5] = 0;
        }

        let mut out = EdScalar {
            n: [t[0], t[1], t[2], t[3]],
        };
        // The result may exceed L by at most one multiple; conditionally subtract.
        out.conditional_sub_l(out.ge_l());
        out
    }

    /// Convert into Montgomery form.
    fn to_mont(&self) -> EdScalar {
        let r2 = EdScalar { n: R2 };
        self.mont_mul(&r2)
    }

    /// Convert out of Montgomery form.
    fn from_mont(&self) -> EdScalar {
        let one = EdScalar {
            n: [1, 0, 0, 0],
        };
        self.mont_mul(&one)
    }

    pub fn add(&self, other: &EdScalar) -> EdScalar {
        let mut carry = 0u64;
        let mut n = [0u64; 4];
        for i in 0..4 {
            let (s, c) = adc(self.n[i], other.n[i], carry);
            n[i] = s;
            carry = c;
        }
        // Both operands are canonical (< L), so the sum is < 2L < 2^256 and the
        // carry out is always zero; one conditional subtraction reduces it.
        debug_assert_eq!(carry, 0, "canonical operands must not overflow 256 bits");
        let mut out = EdScalar { n };
        out.conditional_sub_l(out.ge_l());
        out
    }

    pub fn sub(&self, other: &EdScalar) -> EdScalar {
        let mut borrow = 0u64;
        let mut n = [0u64; 4];
        for i in 0..4 {
            let (d, b) = sbb(self.n[i], other.n[i], borrow);
            n[i] = d;
            borrow = b;
        }
        // If we borrowed, add L back.
        let mask = 0u64.wrapping_sub(borrow);
        let mut carry = 0u64;
        for i in 0..4 {
            let (s, c) = adc(n[i], L[i] & mask, carry);
            n[i] = s;
            carry = c;
        }
        EdScalar { n }
    }

    pub fn neg(&self) -> EdScalar {
        EdScalar::ZERO.sub(self)
    }

    pub fn mul(&self, other: &EdScalar) -> EdScalar {
        self.to_mont().mont_mul(&other.to_mont()).from_mont()
    }

    /// Modular exponentiation by a 256-bit little-endian exponent.
    pub fn pow_le(&self, exp: &[u8; 32]) -> EdScalar {
        let mut result = EdScalar { n: [1, 0, 0, 0] }.to_mont();
        let base = self.to_mont();
        for byte in exp.iter().rev() {
            for bit in (0..8).rev() {
                result = result.mont_mul(&result);
                if (byte >> bit) & 1 == 1 {
                    result = result.mont_mul(&base);
                }
            }
        }
        result.from_mont()
    }

    /// Multiplicative inverse via Fermat: a^(L-2).
    pub fn invert(&self) -> EdScalar {
        // L - 2
        let mut exp = L;
        exp[0] = exp[0].wrapping_sub(2);
        let mut bytes = [0u8; 32];
        for (i, limb) in exp.iter().enumerate() {
            bytes[i * 8..i * 8 + 8].copy_from_slice(&limb.to_le_bytes());
        }
        self.pow_le(&bytes)
    }

    /// Uniformly random non-zero scalar from the operating system CSPRNG.
    pub fn random() -> EdScalar {
        loop {
            let mut bytes = [0u8; 32];
            OsRng.fill_bytes(&mut bytes);
            let s = EdScalar::from_le_bytes(&bytes);
            if s.to_le_bytes() != [0u8; 32] {
                return s;
            }
        }
    }

    pub fn is_zero(&self) -> bool {
        self.n == [0, 0, 0, 0]
    }

    /// Wipe the limbs. Called by `Zeroize` on drop.
    fn wipe(&mut self) {
        self.n = [0u64; 4];
    }

    /// Evaluate `coeffs[0] + coeffs[1]*x + ...` at the integer x.
    pub fn eval_polynomial(coeffs: &[EdScalar], x: u32) -> EdScalar {
        let x_s = EdScalar::from_limbs([x as u64, 0, 0, 0]);
        let mut result = EdScalar::ZERO;
        let mut x_pow = EdScalar { n: [1, 0, 0, 0] };
        for coeff in coeffs {
            result = result.add(&coeff.mul(&x_pow));
            x_pow = x_pow.mul(&x_s);
        }
        result
    }

    /// Lagrange interpolation at x = 0 from explicit (x, y) pairs.
    pub fn lagrange_at_zero(xs: &[u32], ys: &[EdScalar]) -> Result<EdScalar, &'static str> {
        if xs.len() != ys.len() {
            return Err("x and y coordinate counts differ");
        }
        let mut secret = EdScalar::ZERO;
        for i in 0..xs.len() {
            let x_i = EdScalar::from_limbs([xs[i] as u64, 0, 0, 0]);
            let mut num = EdScalar { n: [1, 0, 0, 0] };
            let mut den = EdScalar { n: [1, 0, 0, 0] };
            for j in 0..xs.len() {
                if i == j {
                    continue;
                }
                if xs[i] == xs[j] {
                    return Err("duplicate share index");
                }
                let x_j = EdScalar::from_limbs([xs[j] as u64, 0, 0, 0]);
                // L_i(0) = prod (-x_j) / (x_i - x_j)
                num = num.mul(&x_j.neg());
                den = den.mul(&x_i.sub(&x_j));
            }
            if den.is_zero() {
                return Err("non-invertible Lagrange denominator");
            }
            secret = secret.add(&ys[i].mul(&num.mul(&den.invert())));
        }
        Ok(secret)
    }
}

impl zeroize::Zeroize for EdScalar {
    fn zeroize(&mut self) {
        self.wipe();
    }
}

/// Convert an Ed25519 scalar to the `curve25519_dalek` scalar type by encoding
/// the canonical little-endian bytes. The bytes are already reduced mod L, so
/// `from_bytes` cannot fail its reduction.
pub fn to_dalek_scalar(s: &EdScalar) -> Scalar {
    Scalar::from_bytes_mod_order(s.to_le_bytes())
}

/// Derive the Ed25519 public key for a scalar: `A = s * B`.
pub fn public_key_from_scalar(s: &EdScalar) -> [u8; 32] {
    let point = &to_dalek_scalar(s) * ED25519_BASEPOINT_TABLE;
    point.compress().to_bytes()
}

/// The Ed25519 identity point, used to verify a reconstructed key.
pub fn is_identity(bytes: &[u8; 32]) -> bool {
    use curve25519_dalek::edwards::CompressedEdwardsY;
    CompressedEdwardsY(*bytes)
        .decompress()
        .map(|p| p == curve25519_dalek::edwards::EdwardsPoint::identity())
        .unwrap_or(false)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn zero_and_one_round_trip() {
        assert_eq!(EdScalar::ZERO.to_le_bytes(), [0u8; 32]);
        let one = EdScalar::from_limbs([1, 0, 0, 0]);
        assert_eq!(one.to_le_bytes()[0], 1);
    }

    #[test]
    fn addition_wraps_mod_l() {
        let l_minus_1 = EdScalar::from_limbs([L[0] - 1, L[1], L[2], L[3]]);
        let one = EdScalar::from_limbs([1, 0, 0, 0]);
        assert!(l_minus_1.add(&one).is_zero(), "(L-1) + 1 must be 0 mod L");
    }

    #[test]
    fn subtraction_is_inverse_of_addition() {
        let a = EdScalar::random();
        let b = EdScalar::random();
        assert_eq!(a.add(&b).sub(&b).to_le_bytes(), a.to_le_bytes());
    }

    #[test]
    fn multiplication_and_inverse_agree() {
        let a = EdScalar::random();
        let inv = a.invert();
        let product = a.mul(&inv);
        let mut expected = [0u8; 32];
        expected[0] = 1;
        assert_eq!(product.to_le_bytes(), expected, "a * a^-1 must be 1");
    }

    #[test]
    fn multiplication_is_commutative() {
        let a = EdScalar::random();
        let b = EdScalar::random();
        assert_eq!(a.mul(&b).to_le_bytes(), b.mul(&a).to_le_bytes());
    }

    #[test]
    fn multiplication_is_distributive() {
        let a = EdScalar::random();
        let b = EdScalar::random();
        let c = EdScalar::random();
        let lhs = a.mul(&b.add(&c));
        let rhs = a.mul(&b).add(&a.mul(&c));
        assert_eq!(lhs.to_le_bytes(), rhs.to_le_bytes());
    }

    #[test]
    fn polynomial_matches_manual_evaluation() {
        let c0 = EdScalar::random();
        let c1 = EdScalar::random();
        let c2 = EdScalar::random();
        let coeffs = [c0, c1, c2];
        let x = EdScalar::from_limbs([7, 0, 0, 0]);
        let manual = c0.add(&c1.mul(&x)).add(&c2.mul(&x.mul(&x)));
        assert_eq!(
            EdScalar::eval_polynomial(&coeffs, 7).to_le_bytes(),
            manual.to_le_bytes()
        );
    }

    #[test]
    fn lagrange_recovers_the_constant_term_from_any_subset() {
        let secret = EdScalar::random();
        let coeffs = [secret, EdScalar::random(), EdScalar::random()];
        let points: Vec<(u32, EdScalar)> = (1..=5)
            .map(|i| (i, EdScalar::eval_polynomial(&coeffs, i)))
            .collect();

        // Any 3 of the 5 points must recover the secret.
        for subset in [[0, 1, 2], [0, 2, 4], [1, 3, 4], [2, 3, 4]] {
            let xs: Vec<u32> = subset.iter().map(|&i| points[i].0).collect();
            let ys: Vec<EdScalar> = subset.iter().map(|&i| points[i].1).collect();
            let got = EdScalar::lagrange_at_zero(&xs, &ys).unwrap();
            assert_eq!(got.to_le_bytes(), secret.to_le_bytes(), "subset {:?}", subset);
        }
    }

    #[test]
    fn duplicate_indices_are_rejected() {
        let ys = [EdScalar::random(), EdScalar::random()];
        assert!(EdScalar::lagrange_at_zero(&[3, 3], &ys).is_err());
    }

    #[test]
    fn public_key_derivation_is_deterministic() {
        let s = EdScalar::random();
        assert_eq!(public_key_from_scalar(&s), public_key_from_scalar(&s));
    }

    #[test]
    fn distinct_scalars_give_distinct_public_keys() {
        let a = EdScalar::random();
        let b = EdScalar::random();
        assert_ne!(public_key_from_scalar(&a), public_key_from_scalar(&b));
    }

    #[test]
    fn canonical_check_rejects_out_of_range() {
        let mut bad = [0xffu8; 32];
        bad[31] = 0xff;
        assert!(!EdScalar::is_canonical(&bad), "2^256-1 is not a canonical scalar");
    }
}
