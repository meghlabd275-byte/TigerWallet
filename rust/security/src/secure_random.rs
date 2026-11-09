//! Cryptographically secure random helpers.
//!
//! Every value produced here comes from the operating system CSPRNG. Callers
//! must treat a failure as fatal: there is no fallback, because a predictable
//! fallback would silently weaken every key derived from it.

use getrandom::getrandom;

/// Error returned when the OS CSPRNG cannot be reached.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RandomError(pub String);

impl std::fmt::Display for RandomError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "secure random unavailable: {}", self.0)
    }
}

impl std::error::Error for RandomError {}

/// Fill `out` with random bytes from the OS CSPRNG.
pub fn fill_random(out: &mut [u8]) -> Result<(), RandomError> {
    getrandom(out).map_err(|e| RandomError(e.to_string()))
}

/// `len` random bytes from the OS CSPRNG.
pub fn generate_random_bytes(len: usize) -> Result<Vec<u8>, RandomError> {
    let mut out = vec![0u8; len];
    fill_random(&mut out)?;
    Ok(out)
}

/// `bytes` random bytes rendered as lowercase hex.
pub fn generate_random_hex(bytes: usize) -> String {
    // A failure here cannot be reported through the callers' infallible
    // signature, and continuing with zeroed bytes would be a security hole, so
    // the process aborts instead of producing predictable output.
    match generate_random_bytes(bytes) {
        Ok(buf) => hex::encode(buf),
        Err(e) => panic!("secure random unavailable, refusing to continue: {e}"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn random_bytes_are_filled_and_vary() {
        let a = generate_random_bytes(32).unwrap();
        let b = generate_random_bytes(32).unwrap();
        assert_eq!(a.len(), 32);
        assert_ne!(a, b, "two draws must not be identical");
        assert_ne!(a, vec![0u8; 32], "must not return zeroed output");
    }

    #[test]
    fn random_hex_is_hex() {
        let value = generate_random_hex(16);
        assert_eq!(value.len(), 32);
        assert!(value.chars().all(|c| c.is_ascii_hexdigit()));
    }
}
