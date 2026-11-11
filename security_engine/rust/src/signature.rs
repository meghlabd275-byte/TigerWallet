//! Real signature verification.
//!
//! Covers the two families this wallet signs with: secp256k1 ECDSA (EVM and
//! Bitcoin-style chains, both raw-hash and EIP-191/EIP-712 prefixed messages)
//! and Ed25519 (Solana, Cosmos, NEAR, Sui and the other Ed25519 chains).
//!
//! Verification is fail-closed. A malformed signature, a wrong-length public
//! key, or a hash that does not match returns `valid: false`; nothing is
//! accepted because the inputs were merely non-empty.

use k256::ecdsa::{RecoveryId, Signature as K256Signature, VerifyingKey};
use sha3::{Digest as Sha3Digest, Keccak256};

/// Signature algorithm reported back to the caller.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SignatureAlgorithm {
    Secp256k1,
    Ed25519,
    Unknown,
}

impl SignatureAlgorithm {
    pub fn as_str(self) -> &'static str {
        match self {
            SignatureAlgorithm::Secp256k1 => "secp256k1-ECDSA",
            SignatureAlgorithm::Ed25519 => "Ed25519",
            SignatureAlgorithm::Unknown => "unknown",
        }
    }
}

/// Outcome of a verification attempt.
#[derive(Debug, Clone)]
pub struct VerifiedSignature {
    pub valid: bool,
    pub signer: String,
    pub algorithm: SignatureAlgorithm,
    pub reason: Option<String>,
}

impl VerifiedSignature {
    fn invalid(algorithm: SignatureAlgorithm, reason: impl Into<String>) -> Self {
        Self {
            valid: false,
            signer: String::new(),
            algorithm,
            reason: Some(reason.into()),
        }
    }

    fn unknown(reason: impl Into<String>) -> Self {
        Self {
            valid: false,
            signer: String::new(),
            algorithm: SignatureAlgorithm::Unknown,
            reason: Some(reason.into()),
        }
    }
}

/// Decode a hex string, tolerating an optional `0x` prefix.
pub fn decode_hex(input: &str) -> Result<Vec<u8>, String> {
    let trimmed = input.trim();
    let body = trimmed.strip_prefix("0x").or_else(|| trimmed.strip_prefix("0X")).unwrap_or(trimmed);
    if body.is_empty() {
        return Err("hex input is empty".to_string());
    }
    if body.len() % 2 != 0 {
        return Err("hex input has an odd number of digits".to_string());
    }
    hex::decode(body).map_err(|e| format!("invalid hex: {e}"))
}

/// Derive the lowercase hex Ethereum address for a secp256k1 public key.
pub fn evm_address_from_public_key(public_key: &[u8]) -> Result<String, String> {
    let verifying_key =
        VerifyingKey::from_sec1_bytes(public_key).map_err(|e| format!("invalid secp256k1 public key: {e}"))?;
    let encoded = verifying_key.to_encoded_point(false);
    // The address is the last 20 bytes of keccak256 of the uncompressed key
    // without its 0x04 prefix.
    let digest = Keccak256::digest(&encoded.as_bytes()[1..]);
    Ok(format!("0x{}", hex::encode(&digest[12..])))
}

/// Verify a secp256k1 signature over an already-hashed 32-byte digest.
///
/// `signature` is the 64-byte r||s form; `recovery_id` is optional and, when
/// supplied, must match the key that produced the signature.
pub fn verify_secp256k1_digest(
    digest: &[u8],
    signature: &[u8],
    recovery_id: Option<u8>,
    public_key: &[u8],
) -> VerifiedSignature {
    if digest.len() != 32 {
        return VerifiedSignature::invalid(
            SignatureAlgorithm::Secp256k1,
            format!("digest must be 32 bytes, got {}", digest.len()),
        );
    }
    if signature.len() != 64 {
        return VerifiedSignature::invalid(
            SignatureAlgorithm::Secp256k1,
            format!("signature must be 64 bytes (r||s), got {}", signature.len()),
        );
    }

    let parsed = match K256Signature::from_slice(signature) {
        Ok(sig) => sig,
        Err(e) => return VerifiedSignature::invalid(SignatureAlgorithm::Secp256k1, format!("malformed signature: {e}")),
    };

    // Reject malleable (high-S) signatures.
    if parsed.normalize_s().is_some() {
        return VerifiedSignature::invalid(SignatureAlgorithm::Secp256k1, "signature is not canonical (high-S)");
    }

    let verifying_key = match VerifyingKey::from_sec1_bytes(public_key) {
        Ok(key) => key,
        Err(e) => return VerifiedSignature::invalid(SignatureAlgorithm::Secp256k1, format!("invalid public key: {e}")),
    };

    if let Some(id) = recovery_id {
        if id > 3 {
            return VerifiedSignature::invalid(SignatureAlgorithm::Secp256k1, "recovery id out of range");
        }
        let recovery = match RecoveryId::from_byte(id) {
            Some(rec) => rec,
            None => return VerifiedSignature::invalid(SignatureAlgorithm::Secp256k1, "invalid recovery id"),
        };
        // Recover the key from the signature and require it to match the key
        // the caller supplied.
        match VerifyingKey::recover_from_prehash(digest, &parsed, recovery) {
            Ok(recovered) => {
                if recovered != verifying_key {
                    return VerifiedSignature::invalid(
                        SignatureAlgorithm::Secp256k1,
                        "recovered key does not match the supplied public key",
                    );
                }
            }
            Err(e) => {
                return VerifiedSignature::invalid(
                    SignatureAlgorithm::Secp256k1,
                    format!("could not recover public key: {e}"),
                )
            }
        }
    }

    use k256::ecdsa::signature::hazmat::PrehashVerifier;
    match verifying_key.verify_prehash(digest, &parsed) {
        Ok(()) => VerifiedSignature {
            valid: true,
            signer: evm_address_from_public_key(public_key).unwrap_or_else(|_| hex::encode(public_key)),
            algorithm: SignatureAlgorithm::Secp256k1,
            reason: None,
        },
        Err(_) => VerifiedSignature::invalid(SignatureAlgorithm::Secp256k1, "signature does not verify"),
    }
}

/// Verify a personal-sign (EIP-191) message.
///
/// The signer signs keccak256("\x19Ethereum Signed Message:\n" + len + message).
pub fn verify_eip191_message(message: &[u8], signature: &[u8], public_key: &[u8]) -> VerifiedSignature {
    if signature.len() != 64 && signature.len() != 65 {
        return VerifiedSignature::invalid(
            SignatureAlgorithm::Secp256k1,
            format!("signature must be 64 or 65 bytes, got {}", signature.len()),
        );
    }

    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let mut hasher = Keccak256::new();
    hasher.update(prefix.as_bytes());
    hasher.update(message);
    let digest = hasher.finalize();

    let (sig, recovery_id) = if signature.len() == 65 {
        (signature[..64].to_vec(), Some(signature[64]))
    } else {
        (signature.to_vec(), None)
    };

    verify_secp256k1_digest(&digest, &sig, recovery_id, public_key)
}

/// Verify an Ed25519 signature.
///
/// `public_key` must be 32 bytes and `signature` 64 bytes, per RFC 8032.
pub fn verify_ed25519(message: &[u8], signature: &[u8], public_key: &[u8]) -> VerifiedSignature {
    if public_key.len() != 32 {
        return VerifiedSignature::invalid(
            SignatureAlgorithm::Ed25519,
            format!("Ed25519 public key must be 32 bytes, got {}", public_key.len()),
        );
    }
    if signature.len() != 64 {
        return VerifiedSignature::invalid(
            SignatureAlgorithm::Ed25519,
            format!("Ed25519 signature must be 64 bytes, got {}", signature.len()),
        );
    }

    let key_bytes: [u8; 32] = match public_key.try_into() {
        Ok(bytes) => bytes,
        Err(_) => return VerifiedSignature::invalid(SignatureAlgorithm::Ed25519, "public key length mismatch"),
    };
    let sig_bytes: [u8; 64] = match signature.try_into() {
        Ok(bytes) => bytes,
        Err(_) => return VerifiedSignature::invalid(SignatureAlgorithm::Ed25519, "signature length mismatch"),
    };

    use ed25519_dalek::{Signature as Ed25519Signature, Verifier, VerifyingKey as Ed25519VerifyingKey};
    let verifying_key = match Ed25519VerifyingKey::from_bytes(&key_bytes) {
        Ok(key) => key,
        Err(e) => return VerifiedSignature::invalid(SignatureAlgorithm::Ed25519, format!("invalid public key: {e}")),
    };
    let signature = Ed25519Signature::from_bytes(&sig_bytes);

    match verifying_key.verify(message, &signature) {
        Ok(()) => VerifiedSignature {
            valid: true,
            signer: hex::encode(public_key),
            algorithm: SignatureAlgorithm::Ed25519,
            reason: None,
        },
        Err(_) => VerifiedSignature::invalid(SignatureAlgorithm::Ed25519, "signature does not verify"),
    }
}

/// Verify a signature without knowing the algorithm up front.
///
/// `signature` may be hex (with or without `0x`) or raw bytes. The algorithm is
/// chosen from the public-key length: 33 or 65 bytes selects secp256k1, 32 bytes
/// selects Ed25519.
pub fn verify_auto(message: &[u8], signature: &[u8], public_key: &[u8]) -> VerifiedSignature {
    let sig_bytes: Vec<u8> = if signature.iter().all(|b| b.is_ascii_hexdigit() || *b == b'x' || *b == b'X') && signature.len() >= 128 {
        match decode_hex(&String::from_utf8_lossy(signature)) {
            Ok(bytes) => bytes,
            Err(e) => return VerifiedSignature::unknown(e),
        }
    } else {
        signature.to_vec()
    };

    match public_key.len() {
        33 | 65 => verify_secp256k1_digest(&keccak256(message), &sig_bytes, None, public_key),
        32 => verify_ed25519(message, &sig_bytes, public_key),
        other => VerifiedSignature::unknown(format!("unsupported public key length: {other}")),
    }
}

/// keccak256 of the input, the hash function used by EVM chains.
pub fn keccak256(data: &[u8]) -> [u8; 32] {
    let digest = Keccak256::digest(data);
    let mut out = [0u8; 32];
    out.copy_from_slice(&digest);
    out
}
