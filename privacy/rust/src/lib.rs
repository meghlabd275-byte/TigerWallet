//! Privacy features: ERC-5564 stealth addresses and a commitment-note mixer.
//!
//! # Stealth addresses (ERC-5564 / ERC-6538)
//!
//! A recipient publishes a spending public key `P_spend` and a viewing public
//! key `P_view`. A sender picks an ephemeral key `r`, computes the shared
//! secret `s = keccak256(r * P_view)` and derives the recipient's one-time
//! address from `P_spend + s*G`. The sender publishes `R = r*G`; only the
//! recipient (holding the viewing key) can recompute `s` and spend the funds.
//!
//! # Mixer notes
//!
//! The mixer works on commitments rather than amounts. A deposit records
//! `commitment = keccak256(secret || denomination || leaf_index)`, and a
//! withdrawal reveals the nullifier so the same note cannot be spent twice.
//!
//! Everything here fails closed. When a value cannot be computed honestly the
//! function returns an error rather than a placeholder.

use secp256k1::rand::thread_rng;
use secp256k1::{PublicKey, Scalar, SecretKey, SECP256K1};
use sha3::{Digest, Keccak256};
use thiserror::Error;

/// Errors returned by the privacy service.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum PrivacyError {
    #[error("invalid public key: {0}")]
    InvalidPublicKey(String),
    #[error("invalid secret key: {0}")]
    InvalidSecretKey(String),
    #[error("invalid hex: {0}")]
    InvalidHex(String),
    #[error("insufficient entropy: {0}")]
    InsufficientEntropy(String),
    #[error("note is not in the tree")]
    UnknownNote,
    #[error("note has already been spent")]
    NoteAlreadySpent,
    #[error("deposit does not match the withdrawal")]
    DepositMismatch,
}

// ============================================================================
// Hex helpers
// ============================================================================

fn hex_decode(s: &str) -> Result<Vec<u8>, PrivacyError> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    if s.len() % 2 != 0 {
        return Err(PrivacyError::InvalidHex(format!(
            "odd-length hex string ({})",
            s.len()
        )));
    }
    hex::decode(s).map_err(|e| PrivacyError::InvalidHex(e.to_string()))
}

fn hex_encode(bytes: &[u8]) -> String {
    format!("0x{}", hex::encode(bytes))
}

/// Parse a 33-byte compressed or 65-byte uncompressed secp256k1 public key.
fn parse_public_key(hex_str: &str) -> Result<PublicKey, PrivacyError> {
    let bytes = hex_decode(hex_str)?;
    PublicKey::from_slice(&bytes).map_err(|e| PrivacyError::InvalidPublicKey(e.to_string()))
}

/// Parse a 32-byte secp256k1 secret key.
fn parse_secret_key(hex_str: &str) -> Result<SecretKey, PrivacyError> {
    let bytes = hex_decode(hex_str)?;
    if bytes.len() != 32 {
        return Err(PrivacyError::InvalidSecretKey(format!(
            "expected 32 bytes, got {}",
            bytes.len()
        )));
    }
    SecretKey::from_slice(&bytes).map_err(|e| PrivacyError::InvalidSecretKey(e.to_string()))
}

/// Derive the Ethereum address for a public key.
///
/// The address is the last 20 bytes of the Keccak-256 hash of the uncompressed
/// public key with its `0x04` prefix removed.
fn public_key_to_address(pubkey: &PublicKey) -> String {
    let uncompressed = pubkey.serialize_uncompressed();
    let hash = Keccak256::digest(&uncompressed[1..]);
    hex_encode(&hash[12..])
}

/// Reduce a 32-byte value to a valid secp256k1 scalar.
///
/// The raw Keccak output is interpreted as a big-endian integer and reduced
/// modulo the curve order. `Scalar::from_be_bytes` performs that reduction and
/// rejects only the (astronomically unlikely) zero value.
fn hash_to_scalar(shared_secret: &[u8]) -> Result<Scalar, PrivacyError> {
    let digest = Keccak256::digest(shared_secret);
    Scalar::from_be_bytes(digest.into())
        .map_err(|_| PrivacyError::InvalidSecretKey("shared secret reduced to zero".into()))
}

// ============================================================================
// Stealth addresses (ERC-5564)
// ============================================================================

/// A recipient's published stealth-address keys.
///
/// The viewing key pair may be shared with a third party (for example an
/// auditor) without granting spending authority.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StealthKeys {
    pub spending_public_key: String,
    pub viewing_public_key: String,
}

/// The recipient's private keys. These must never leave the device.
pub struct StealthKeyPair {
    spending_private_key: SecretKey,
    viewing_private_key: SecretKey,
    pub public: StealthKeys,
}

impl StealthKeyPair {
    /// Generate a fresh key pair from the operating system entropy source.
    pub fn generate() -> Result<Self, PrivacyError> {
        let spending_private_key = SecretKey::new(&mut thread_rng());
        let viewing_private_key = SecretKey::new(&mut thread_rng());
        Self::from_secrets(spending_private_key, viewing_private_key)
    }

    /// Rebuild a key pair from stored private keys.
    pub fn from_private_keys(
        spending_private_key: &str,
        viewing_private_key: &str,
    ) -> Result<Self, PrivacyError> {
        Self::from_secrets(
            parse_secret_key(spending_private_key)?,
            parse_secret_key(viewing_private_key)?,
        )
    }

    fn from_secrets(
        spending_private_key: SecretKey,
        viewing_private_key: SecretKey,
    ) -> Result<Self, PrivacyError> {
        if spending_private_key == viewing_private_key {
            return Err(PrivacyError::InvalidSecretKey(
                "spending and viewing keys must differ".into(),
            ));
        }
        let public = StealthKeys {
            spending_public_key: hex_encode(
                &PublicKey::from_secret_key(SECP256K1, &spending_private_key).serialize(),
            ),
            viewing_public_key: hex_encode(
                &PublicKey::from_secret_key(SECP256K1, &viewing_private_key).serialize(),
            ),
        };
        Ok(Self {
            spending_private_key,
            viewing_private_key,
            public,
        })
    }

    /// Export the private keys for encrypted storage.
    ///
    /// Callers are responsible for encrypting the result before it is
    /// persisted; this method exists so the keys can round-trip.
    pub fn export_private_keys(&self) -> (String, String) {
        (
            hex_encode(&self.spending_private_key.secret_bytes()),
            hex_encode(&self.viewing_private_key.secret_bytes()),
        )
    }
}

/// The data a sender must publish so the recipient can find the payment.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StealthPayment {
    /// The one-time address the funds were sent to.
    pub stealth_address: String,
    /// The ephemeral public key `R = r*G`, published alongside the transfer.
    pub ephemeral_public_key: String,
    /// First byte of `keccak256(s)`, the ERC-5564 `viewTag`.
    ///
    /// It lets a scanning recipient skip the expensive key derivation for
    /// announcements that cannot belong to them.
    pub view_tag: u8,
}

/// Derive a one-time stealth address for a recipient (sender side).
///
/// Returns the address to pay, the ephemeral public key to publish, and the
/// view tag. A fresh ephemeral key is generated for every payment, so two
/// payments to the same recipient produce unrelated addresses.
pub fn generate_stealth_address(keys: &StealthKeys) -> Result<StealthPayment, PrivacyError> {
    let spending_pub = parse_public_key(&keys.spending_public_key)?;
    let viewing_pub = parse_public_key(&keys.viewing_public_key)?;

    let ephemeral_secret = SecretKey::new(&mut thread_rng());
    let ephemeral_public = PublicKey::from_secret_key(SECP256K1, &ephemeral_secret);

    derive_payment(
        &spending_pub,
        &viewing_pub,
        &ephemeral_secret,
        &ephemeral_public,
    )
}

/// Derive a payment for an already-known ephemeral key.
///
/// This is the shared core of sender-side derivation and recipient-side
/// scanning.
fn derive_payment(
    spending_pub: &PublicKey,
    viewing_pub: &PublicKey,
    ephemeral_secret: &SecretKey,
    ephemeral_public: &PublicKey,
) -> Result<StealthPayment, PrivacyError> {
    let shared_point = viewing_pub
        .mul_tweak(SECP256K1, &Scalar::from(*ephemeral_secret))
        .map_err(|e| PrivacyError::InvalidPublicKey(e.to_string()))?;
    let shared_secret = shared_point.serialize();
    let scalar = hash_to_scalar(&shared_secret)?;

    let stealth_pub = spending_pub
        .add_exp_tweak(SECP256K1, &scalar)
        .map_err(|e| PrivacyError::InvalidPublicKey(e.to_string()))?;

    let tag_digest = Keccak256::digest(shared_secret);

    Ok(StealthPayment {
        stealth_address: public_key_to_address(&stealth_pub),
        ephemeral_public_key: hex_encode(&ephemeral_public.serialize()),
        view_tag: tag_digest[0],
    })
}

/// Recompute the stealth address for an announcement (recipient side).
///
/// The recipient uses their viewing private key and the published ephemeral
/// key. This is the operation a wallet performs while scanning for payments,
/// so it is also the way to confirm that an announced address really belongs
/// to the recipient before spending.
pub fn scan_payment(
    viewing_private_key: &str,
    spending_public_key: &str,
    ephemeral_public_key: &str,
    view_tag: u8,
) -> Result<String, PrivacyError> {
    let viewing_secret = parse_secret_key(viewing_private_key)?;
    let spending_pub = parse_public_key(spending_public_key)?;
    let ephemeral_pub = parse_public_key(ephemeral_public_key)?;

    let shared_point = ephemeral_pub
        .mul_tweak(SECP256K1, &Scalar::from(viewing_secret))
        .map_err(|e| PrivacyError::InvalidPublicKey(e.to_string()))?;
    let shared_secret = shared_point.serialize();

    let tag_digest = Keccak256::digest(shared_secret);
    if tag_digest[0] != view_tag {
        // The announcement is not addressed to this viewing key. Reporting a
        // derived address here would produce a false match.
        return Err(PrivacyError::DepositMismatch);
    }

    let scalar = hash_to_scalar(&shared_secret)?;
    let stealth_pub = spending_pub
        .add_exp_tweak(SECP256K1, &scalar)
        .map_err(|e| PrivacyError::InvalidPublicKey(e.to_string()))?;

    Ok(public_key_to_address(&stealth_pub))
}

/// Derive the private key that controls a stealth address (recipient side).
///
/// `spending_private + keccak256(s)` yields the key for the one-time address.
/// This is what allows the recipient to spend the funds.
pub fn derive_stealth_private_key(
    spending_private_key: &str,
    viewing_private_key: &str,
    ephemeral_public_key: &str,
) -> Result<String, PrivacyError> {
    let spending_secret = parse_secret_key(spending_private_key)?;
    let viewing_secret = parse_secret_key(viewing_private_key)?;
    let ephemeral_pub = parse_public_key(ephemeral_public_key)?;

    let shared_point = ephemeral_pub
        .mul_tweak(SECP256K1, &Scalar::from(viewing_secret))
        .map_err(|e| PrivacyError::InvalidPublicKey(e.to_string()))?;
    let scalar = hash_to_scalar(&shared_point.serialize())?;

    let stealth_secret = spending_secret
        .add_tweak(&scalar)
        .map_err(|e| PrivacyError::InvalidSecretKey(e.to_string()))?;

    Ok(hex_encode(&stealth_secret.secret_bytes()))
}

// ============================================================================
// Mixer
// ============================================================================

/// A fixed-denomination mixer note.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Note {
    /// Public commitment recorded on deposit.
    pub commitment: String,
    /// Secret revealed on withdrawal. Derived from the note secret so it
    /// cannot be guessed from the commitment.
    pub nullifier: String,
}

impl Note {
    /// Create a note from caller-supplied secret material.
    ///
    /// The secret must come from a cryptographically secure source; the
    /// caller passes it in so notes can be re-derived from a wallet seed.
    pub fn from_secret(
        secret: &[u8],
        denomination: u64,
        leaf_index: u64,
    ) -> Result<Self, PrivacyError> {
        if secret.len() < 32 {
            return Err(PrivacyError::InsufficientEntropy(format!(
                "note secret must be at least 32 bytes, got {}",
                secret.len()
            )));
        }
        if denomination == 0 {
            return Err(PrivacyError::DepositMismatch);
        }

        let nullifier = {
            let mut hasher = Keccak256::new();
            hasher.update(b"tigerwallet:nullifier:v1");
            hasher.update(secret);
            hasher.update(denomination.to_be_bytes());
            hex_encode(&hasher.finalize())
        };

        let commitment = {
            let mut hasher = Keccak256::new();
            hasher.update(b"tigerwallet:commitment:v1");
            hasher.update(secret);
            hasher.update(denomination.to_be_bytes());
            hasher.update(leaf_index.to_be_bytes());
            hex_encode(&hasher.finalize())
        };

        Ok(Self {
            commitment,
            nullifier,
        })
    }
}

/// Tracks deposited notes so a nullifier cannot be spent twice.
#[derive(Debug, Default)]
pub struct MixerState {
    deposits: std::collections::HashMap<String, u64>,
    spent_nullifiers: std::collections::HashSet<String>,
    next_leaf: u64,
}

impl MixerState {
    pub fn new() -> Self {
        Self::default()
    }

    /// Record a deposit and return the assigned leaf index.
    ///
    /// Re-depositing the same commitment is rejected: a commitment that
    /// appears twice would let one deposit be withdrawn twice.
    pub fn deposit(&mut self, commitment: &str, denomination: u64) -> Result<u64, PrivacyError> {
        if denomination == 0 {
            return Err(PrivacyError::DepositMismatch);
        }
        if self.deposits.contains_key(commitment) {
            return Err(PrivacyError::DepositMismatch);
        }
        let leaf = self.next_leaf;
        self.deposits.insert(commitment.to_string(), denomination);
        self.next_leaf += 1;
        Ok(leaf)
    }

    /// Mark a note as withdrawn.
    ///
    /// Fails if the commitment was never deposited or if the nullifier has
    /// already been used.
    pub fn withdraw(&mut self, note: &Note) -> Result<u64, PrivacyError> {
        let denomination = *self
            .deposits
            .get(&note.commitment)
            .ok_or(PrivacyError::UnknownNote)?;

        if self.spent_nullifiers.contains(&note.nullifier) {
            return Err(PrivacyError::NoteAlreadySpent);
        }

        self.spent_nullifiers.insert(note.nullifier.clone());
        Ok(denomination)
    }

    pub fn deposit_count(&self) -> usize {
        self.deposits.len()
    }

    pub fn is_spent(&self, nullifier: &str) -> bool {
        self.spent_nullifiers.contains(nullifier)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stealth_keys_generate_distinct_pairs() {
        let a = StealthKeyPair::generate().unwrap();
        let b = StealthKeyPair::generate().unwrap();

        assert_ne!(a.public.spending_public_key, b.public.spending_public_key);
        assert_ne!(a.public.viewing_public_key, b.public.viewing_public_key);
        // Spending and viewing keys must not coincide within one pair.
        assert_ne!(a.public.spending_public_key, a.public.viewing_public_key);
    }

    #[test]
    fn stealth_address_is_deterministic_for_a_given_ephemeral_key() {
        let keys = StealthKeyPair::generate().unwrap().public;

        let spending_pub = parse_public_key(&keys.spending_public_key).unwrap();
        let viewing_pub = parse_public_key(&keys.viewing_public_key).unwrap();
        let ephemeral_secret = SecretKey::new(&mut thread_rng());
        let ephemeral_public = PublicKey::from_secret_key(SECP256K1, &ephemeral_secret);

        let first =
            derive_payment(&spending_pub, &viewing_pub, &ephemeral_secret, &ephemeral_public)
                .unwrap();
        let second =
            derive_payment(&spending_pub, &viewing_pub, &ephemeral_secret, &ephemeral_public)
                .unwrap();

        assert_eq!(first, second);
        assert!(first.stealth_address.starts_with("0x"));
        assert_eq!(first.stealth_address.len(), 42);
    }

    #[test]
    fn two_payments_to_the_same_recipient_use_different_addresses() {
        let keys = StealthKeyPair::generate().unwrap().public;

        let first = generate_stealth_address(&keys).unwrap();
        let second = generate_stealth_address(&keys).unwrap();

        assert_ne!(first.stealth_address, second.stealth_address);
        assert_ne!(first.ephemeral_public_key, second.ephemeral_public_key);
    }

    #[test]
    fn recipient_recovers_the_address_the_sender_derived() {
        let pair = StealthKeyPair::generate().unwrap();
        let payment = generate_stealth_address(&pair.public).unwrap();

        let (_, viewing_private) = pair.export_private_keys();
        let recovered = scan_payment(
            &viewing_private,
            &pair.public.spending_public_key,
            &payment.ephemeral_public_key,
            payment.view_tag,
        )
        .unwrap();

        assert_eq!(recovered, payment.stealth_address);
    }

    #[test]
    fn recipient_derives_a_private_key_controlling_the_stealth_address() {
        let pair = StealthKeyPair::generate().unwrap();
        let payment = generate_stealth_address(&pair.public).unwrap();

        let (spending_private, viewing_private) = pair.export_private_keys();
        let stealth_private = derive_stealth_private_key(
            &spending_private,
            &viewing_private,
            &payment.ephemeral_public_key,
        )
        .unwrap();

        // The derived private key must produce the address the sender paid.
        let secret = parse_secret_key(&stealth_private).unwrap();
        let pubkey = PublicKey::from_secret_key(SECP256K1, &secret);
        assert_eq!(public_key_to_address(&pubkey), payment.stealth_address);
    }

    #[test]
    fn scan_rejects_an_announcement_for_a_different_recipient() {
        let intended = StealthKeyPair::generate().unwrap();
        let other = StealthKeyPair::generate().unwrap();

        let payment = generate_stealth_address(&intended.public).unwrap();
        let (_, other_viewing_private) = other.export_private_keys();

        // A wrong viewing key yields a different shared secret, so the view
        // tag will not match and no address is reported.
        let result = scan_payment(
            &other_viewing_private,
            &intended.public.spending_public_key,
            &payment.ephemeral_public_key,
            payment.view_tag,
        );

        assert!(matches!(result, Err(PrivacyError::DepositMismatch)));
    }

    #[test]
    fn scan_rejects_a_tampered_view_tag() {
        let pair = StealthKeyPair::generate().unwrap();
        let payment = generate_stealth_address(&pair.public).unwrap();
        let (_, viewing_private) = pair.export_private_keys();

        let tampered = payment.view_tag.wrapping_add(1);
        let result = scan_payment(
            &viewing_private,
            &pair.public.spending_public_key,
            &payment.ephemeral_public_key,
            tampered,
        );

        assert!(matches!(result, Err(PrivacyError::DepositMismatch)));
    }

    #[test]
    fn invalid_keys_are_rejected() {
        assert!(parse_public_key("0x1234").is_err());
        assert!(parse_public_key("not-hex").is_err());
        assert!(parse_secret_key("0x00").is_err());
        // Zero is not a valid secp256k1 scalar.
        assert!(parse_secret_key(&"0".repeat(64)).is_err());
    }

    #[test]
    fn identical_spending_and_viewing_keys_are_rejected() {
        let secret = hex_encode(
            &SecretKey::new(&mut thread_rng()).secret_bytes(),
        );
        assert!(StealthKeyPair::from_private_keys(&secret, &secret).is_err());
    }

    #[test]
    fn note_commitment_and_nullifier_differ() {
        let secret = [7u8; 32];
        let note = Note::from_secret(&secret, 1_000_000, 0).unwrap();

        assert_ne!(note.commitment, note.nullifier);
        assert!(note.commitment.starts_with("0x"));
        assert_eq!(note.commitment.len(), 66);
    }

    #[test]
    fn note_derivation_is_deterministic_and_index_sensitive() {
        let secret = [9u8; 32];

        let a = Note::from_secret(&secret, 100, 0).unwrap();
        let b = Note::from_secret(&secret, 100, 0).unwrap();
        assert_eq!(a, b);

        // A different leaf index is a different deposit.
        let c = Note::from_secret(&secret, 100, 1).unwrap();
        assert_ne!(a.commitment, c.commitment);

        // The nullifier is bound to the denomination.
        let d = Note::from_secret(&secret, 200, 0).unwrap();
        assert_ne!(a.nullifier, d.nullifier);
    }

    #[test]
    fn note_requires_sufficient_entropy() {
        assert!(Note::from_secret(&[1u8; 16], 100, 0).is_err());
        assert!(Note::from_secret(&[1u8; 32], 0, 0).is_err());
    }

    #[test]
    fn mixer_accepts_a_deposit_and_a_withdrawal() {
        let mut state = MixerState::new();
        let note = Note::from_secret(&[3u8; 32], 500, 0).unwrap();

        let leaf = state.deposit(&note.commitment, 500).unwrap();
        assert_eq!(leaf, 0);
        assert_eq!(state.deposit_count(), 1);

        let denomination = state.withdraw(&note).unwrap();
        assert_eq!(denomination, 500);
        assert!(state.is_spent(&note.nullifier));
    }

    #[test]
    fn mixer_rejects_a_double_spend() {
        let mut state = MixerState::new();
        let note = Note::from_secret(&[4u8; 32], 500, 0).unwrap();
        state.deposit(&note.commitment, 500).unwrap();
        state.withdraw(&note).unwrap();

        assert!(matches!(
            state.withdraw(&note),
            Err(PrivacyError::NoteAlreadySpent)
        ));
    }

    #[test]
    fn mixer_rejects_a_withdrawal_for_an_unknown_note() {
        let mut state = MixerState::new();
        let note = Note::from_secret(&[5u8; 32], 500, 0).unwrap();

        assert!(matches!(
            state.withdraw(&note),
            Err(PrivacyError::UnknownNote)
        ));
    }

    #[test]
    fn mixer_rejects_a_duplicate_commitment() {
        let mut state = MixerState::new();
        let note = Note::from_secret(&[6u8; 32], 500, 0).unwrap();

        state.deposit(&note.commitment, 500).unwrap();
        // The same commitment appearing twice would allow a double withdrawal.
        assert!(matches!(
            state.deposit(&note.commitment, 500),
            Err(PrivacyError::DepositMismatch)
        ));
        assert_eq!(state.deposit_count(), 1);
    }

    #[test]
    fn mixer_rejects_a_zero_denomination() {
        let mut state = MixerState::new();
        assert!(matches!(
            state.deposit("0xdeadbeef", 0),
            Err(PrivacyError::DepositMismatch)
        ));
    }
}
