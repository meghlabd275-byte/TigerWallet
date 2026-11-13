//! Digest helpers used by the security modules.
//!
//! `Sha256` is re-exported so callers can keep using `Sha256::new()` /
//! `hasher.update(..)` / `hasher.finalize()` directly.

pub use sha2::{Digest, Sha256};

/// SHA-256 of `data`.
pub fn sha256(data: &[u8]) -> [u8; 32] {
    let mut hasher = Sha256::new();
    hasher.update(data);
    let out = hasher.finalize();
    let mut result = [0u8; 32];
    result.copy_from_slice(&out);
    result
}

/// Keccak-256 of `data`, the hash used by EVM address derivation and EIP-191.
pub fn keccak256(data: &[u8]) -> [u8; 32] {
    use sha3::{Digest as _, Keccak256};
    let mut hasher = Keccak256::new();
    hasher.update(data);
    let out = hasher.finalize();
    let mut result = [0u8; 32];
    result.copy_from_slice(&out);
    result
}

/// RIPEMD-160 of `data`, used by Bitcoin/Stacks address derivation.
pub fn ripemd160(data: &[u8]) -> [u8; 20] {
    use ripemd::Ripemd160;
    let mut hasher = Ripemd160::new();
    hasher.update(data);
    let out = hasher.finalize();
    let mut result = [0u8; 20];
    result.copy_from_slice(&out);
    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sha256_matches_the_known_vector() {
        // SHA-256("abc")
        assert_eq!(
            hex::encode(sha256(b"abc")),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
    }

    #[test]
    fn keccak256_matches_the_known_vector() {
        // Keccak-256("") - note this differs from the SHA-3-256 of the same input.
        assert_eq!(
            hex::encode(keccak256(b"")),
            "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
        );
    }

    #[test]
    fn ripemd160_matches_the_known_vector() {
        // RIPEMD-160("abc")
        assert_eq!(
            hex::encode(ripemd160(b"abc")),
            "8eb208f7e05d987a9b044a8e98c6b087f15a0bfc"
        );
    }
}
