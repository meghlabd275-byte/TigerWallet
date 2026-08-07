//! Threshold multisignature primitives for wallet authorization.
use secp256k1::{ecdsa::Signature, Message, PublicKey, Secp256k1, SecretKey};
use sha3::{Digest, Keccak256};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum MultisigError {
    #[error("threshold must be between one and signer count")]
    InvalidThreshold,
    #[error("duplicate signer public key")]
    DuplicateSigner,
    #[error("invalid digest length")]
    InvalidDigest,
    #[error("invalid secret key")]
    InvalidSecretKey,
    #[error("invalid signature")]
    InvalidSignature,
    #[error("signature signer is not in policy")]
    UnauthorizedSigner,
    #[error("quorum not reached")]
    QuorumNotReached,
}

#[derive(Debug, Clone)]
pub struct MultisigPolicy {
    pub signers: Vec<PublicKey>,
    pub threshold: usize,
}

#[derive(Debug, Clone)]
pub struct MultisigSignature {
    pub signer: PublicKey,
    pub signature: Signature,
}

impl MultisigPolicy {
    pub fn new(signers: Vec<PublicKey>, threshold: usize) -> Result<Self, MultisigError> {
        if threshold == 0 || threshold > signers.len() {
            return Err(MultisigError::InvalidThreshold);
        }
        for (index, signer) in signers.iter().enumerate() {
            if signers[..index].contains(signer) {
                return Err(MultisigError::DuplicateSigner);
            }
        }
        Ok(Self { signers, threshold })
    }

    pub fn contains(&self, signer: &PublicKey) -> bool {
        self.signers.contains(signer)
    }

    pub fn verify_quorum(&self, digest: &[u8; 32], signatures: &[MultisigSignature]) -> Result<(), MultisigError> {
        let secp = Secp256k1::verification_only();
        let message = Message::from_slice(digest).map_err(|_| MultisigError::InvalidDigest)?;
        let mut valid = 0usize;
        let mut seen = Vec::with_capacity(signatures.len());
        for item in signatures {
            if !self.contains(&item.signer) || seen.contains(&item.signer) {
                return Err(MultisigError::UnauthorizedSigner);
            }
            secp.verify_ecdsa(&message, &item.signature, &item.signer)
                .map_err(|_| MultisigError::InvalidSignature)?;
            seen.push(item.signer);
            valid += 1;
        }
        if valid < self.threshold {
            return Err(MultisigError::QuorumNotReached);
        }
        Ok(())
    }
}

pub fn sign_digest(secret_key: &[u8; 32], digest: &[u8; 32]) -> Result<MultisigSignature, MultisigError> {
    let secret = SecretKey::from_slice(secret_key).map_err(|_| MultisigError::InvalidSecretKey)?;
    let secp = Secp256k1::signing_only();
    let message = Message::from_slice(digest).map_err(|_| MultisigError::InvalidDigest)?;
    let signature = secp.sign_ecdsa(&message, &secret);
    Ok(MultisigSignature { signer: PublicKey::from_secret_key(&secp, &secret), signature })
}

pub fn policy_digest(policy: &MultisigPolicy) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update((policy.threshold as u64).to_be_bytes());
    for signer in &policy.signers {
        hasher.update(signer.serialize());
    }
    hasher.finalize().into()
}

pub fn digest_message(payload: &[u8]) -> [u8; 32] {
    Keccak256::digest(payload).into()
}
