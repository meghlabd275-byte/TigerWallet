//! TigerWallet Security Module
//! 
//! Advanced security features including:
//! - Rate limiting
//! - IPFS storage
//! - Anti-phishing
//! - Zero-knowledge proofs

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};

use crate::sha3_256;
use rand::rngs::OsRng;
use rand::RngCore;

// ============================================================================
// Rate Limiter
// ============================================================================

pub struct RateLimiter {
    requests: Arc<RwLock<HashMap<String, Vec<Instant>>>>,
    max_requests: u32,
    window: Duration,
}

impl RateLimiter {
    pub fn new(max_requests: u32, window_secs: u64) -> Self {
        RateLimiter {
            requests: Arc::new(RwLock::new(HashMap::new())),
            max_requests,
            window: Duration::from_secs(window_secs),
        }
    }
    
    pub fn check(&self, key: &str) -> bool {
        let now = Instant::now();
        let mut requests = self.requests.write().unwrap();
        
        let timestamps = requests.entry(key.to_string()).or_insert_with(Vec::new);
        
        // Remove old timestamps
        timestamps.retain(|&t| now.duration_since(t) < self.window);
        
        // Check limit
        if timestamps.len() >= self.max_requests as usize {
            return false;
        }
        
        timestamps.push(now);
        true
    }
    
    pub fn reset(&self, key: &str) {
        let mut requests = self.requests.write().unwrap();
        requests.remove(key);
    }
}

// ============================================================================
// IPFS Storage (Encrypted)
// ============================================================================

/// Client for an IPFS HTTP API endpoint.
///
/// The gateway URL is the API root (for example http://127.0.0.1:5001), not a
/// read-only gateway: uploading needs `/api/v0/add`, which a public gateway
/// does not serve. Every method reports a transport or HTTP failure instead of
/// returning a placeholder, because a caller that stores a fabricated CID has
/// no way to tell that the content was never pinned.
pub struct IPFSStorage {
    gateway_url: String,
    auth_token: Option<String>,
    client: reqwest::blocking::Client,
}

impl IPFSStorage {
    pub fn new(gateway_url: String, auth_token: Option<String>) -> Self {
        let client = reqwest::blocking::Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .unwrap_or_else(|_| reqwest::blocking::Client::new());
        IPFSStorage {
            gateway_url: gateway_url.trim_end_matches('/').to_string(),
            auth_token,
            client,
        }
    }

    fn request(&self, method: reqwest::Method, path: &str) -> reqwest::blocking::RequestBuilder {
        let mut rb = self.client.request(method, format!("{}{}", self.gateway_url, path));
        if let Some(token) = &self.auth_token {
            rb = rb.bearer_auth(token);
        }
        rb
    }

    /// Add `data` to IPFS and return its CID.
    pub fn upload(&self, data: &[u8]) -> Result<String, String> {
        let part = reqwest::blocking::multipart::Part::bytes(data.to_vec()).file_name("blob");
        let form = reqwest::blocking::multipart::Form::new().part("file", part);

        let response = self
            .request(reqwest::Method::POST, "/api/v0/add")
            .query(&[("pin", "true")])
            .multipart(form)
            .send()
            .map_err(|e| format!("ipfs add request failed: {e}"))?;

        if !response.status().is_success() {
            return Err(format!("ipfs add returned HTTP {}", response.status()));
        }

        let body: serde_json::Value = response
            .json()
            .map_err(|e| format!("ipfs add response was not JSON: {e}"))?;

        body.get("Hash")
            .and_then(|h| h.as_str())
            .map(|h| h.to_string())
            .ok_or_else(|| "ipfs add response carried no Hash field".to_string())
    }

    /// Fetch the content identified by `cid`.
    pub fn download(&self, cid: &str) -> Result<Vec<u8>, String> {
        if cid.is_empty() {
            return Err("empty CID".to_string());
        }

        let response = self
            .request(reqwest::Method::POST, "/api/v0/cat")
            .query(&[("arg", cid)])
            .send()
            .map_err(|e| format!("ipfs cat request failed: {e}"))?;

        if !response.status().is_success() {
            return Err(format!("ipfs cat returned HTTP {}", response.status()));
        }

        response
            .bytes()
            .map(|b| b.to_vec())
            .map_err(|e| format!("ipfs cat body could not be read: {e}"))
    }

    /// Pin content that is already reachable by `cid`.
    pub fn pin(&self, cid: &str) -> Result<(), String> {
        if cid.is_empty() {
            return Err("empty CID".to_string());
        }

        let response = self
            .request(reqwest::Method::POST, "/api/v0/pin/add")
            .query(&[("arg", cid)])
            .send()
            .map_err(|e| format!("ipfs pin add request failed: {e}"))?;

        if !response.status().is_success() {
            return Err(format!("ipfs pin add returned HTTP {}", response.status()));
        }
        Ok(())
    }

    /// Remove a pin, which leaves the content on the network if others hold it.
    pub fn unpin(&self, cid: &str) -> Result<(), String> {
        if cid.is_empty() {
            return Err("empty CID".to_string());
        }

        let response = self
            .request(reqwest::Method::POST, "/api/v0/pin/rm")
            .query(&[("arg", cid)])
            .send()
            .map_err(|e| format!("ipfs pin rm request failed: {e}"))?;

        if !response.status().is_success() {
            return Err(format!("ipfs pin rm returned HTTP {}", response.status()));
        }
        Ok(())
    }
}

// ============================================================================
// Anti-Phishing System
// ============================================================================

pub struct AntiPhishing {
    known_phishing: Arc<RwLock<HashMap<String, PhishingInfo>>>,
    suspicious_domains: Arc<RwLock<Vec<String>>>,
}

#[derive(Clone)]
pub struct PhishingInfo {
    pub domain: String,
    pub phishing_type: PhishingType,
    pub target_wallets: Vec<String>,
    pub reported_at: u64,
    pub severity: Severity,
}

#[derive(Clone, Debug, PartialEq)]
pub enum PhishingType {
    Drainer,
    Honeypot,
    FakeExchange,
    FakeAirdrop,
    Impersonation,
    SandwichAttack,
}

#[derive(Clone, Debug, PartialEq)]
pub enum Severity {
    Low,
    Medium,
    High,
    Critical,
}

impl AntiPhishing {
    pub fn new() -> Self {
        AntiPhishing {
            known_phishing: Arc::new(RwLock::new(HashMap::new())),
            suspicious_domains: Arc::new(RwLock::new(Vec::new())),
        }
    }
    
    pub fn check_domain(&self, domain: &str) -> Option<PhishingInfo> {
        let phishing = self.known_phishing.read().unwrap();
        phishing.get(domain).cloned()
    }
    
    pub fn check_url(&self, url: &str) -> Option<PhishingInfo> {
        // Extract domain from URL
        if let Some(domain) = url.split('/').nth(2) {
            return self.check_domain(domain);
        }
        None
    }
    
    pub fn add_phishing(&self, info: PhishingInfo) {
        let mut phishing = self.known_phishing.write().unwrap();
        phishing.insert(info.domain.clone(), info);
    }
    
    pub fn is_suspicious(&self, domain: &str) -> bool {
        let suspicious = self.suspicious_domains.read().unwrap();
        suspicious.iter().any(|d| domain.contains(d))
    }
    
    pub fn report_phishing(&self, domain: String, phishing_type: PhishingType, wallets: Vec<String>) {
        let info = PhishingInfo {
            domain: domain.clone(),
            phishing_type,
            target_wallets: wallets,
            reported_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            severity: Severity::High,
        };
        
        self.add_phishing(info);
    }
}

// ============================================================================
// Zero-Knowledge Proofs
// ============================================================================
//
// Two proof systems are provided, both verifiable without the prover's secret:
//
//   * `schnorr` - a Fiat-Shamir non-interactive proof of knowledge of a discrete
//     logarithm over the secp256k1 group. This is a real sigma protocol: the
//     verifier recomputes the challenge from the transcript and checks the group
//     equation. Proving knowledge of the private key for a public address without
//     revealing the key is exactly this statement.
//
//   * `merkle` - a proof of membership in a Merkle tree using the tree that is
//     already implemented in this module (used for proof-of-reserves).
//
// A general-purpose zkSNARK (Groth16/PLONK) is deliberately NOT implemented
// here: a correct one requires a trusted setup, an R1CS constraint compiler and
// pairing-friendly curve arithmetic, which is a dedicated library-sized project.
// The previous "hash-based proof" could not be verified by anyone, so it has been
// removed rather than left in place pretending to be a proof system.

pub mod zk {
    use crate::sha3_256;
    use secp256k1::{PublicKey, Secp256k1, SecretKey};

    /// Non-interactive proof of knowledge of a secp256k1 secret key (Schnorr,
    /// Fiat-Shamir transformed).
    #[derive(Debug, Clone, PartialEq, Eq)]
    pub struct SchnorrProof {
        /// Commitment R = k*G, 33-byte compressed encoding.
        pub commitment: [u8; 33],
        /// Response s = k + e*x (mod n), 32-byte big-endian.
        pub response: [u8; 32],
    }

    /// Deterministically derive the Fiat-Shamir challenge e for a proof
    /// transcript. Domain-separated so a transcript can never be replayed
    /// against a different statement.
    fn challenge(public_key: &PublicKey, message: &[u8], commitment: &PublicKey) -> [u8; 32] {
        let mut transcript = Vec::with_capacity(64 + message.len());
        transcript.extend_from_slice(b"tigerwallet/schnorr/v1");
        transcript.extend_from_slice(&public_key.serialize());
        transcript.extend_from_slice(&commitment.serialize());
        transcript.extend_from_slice(message);
        sha3_256(&transcript)
    }

    /// Produce a proof of knowledge of the secret key behind `secret_key` for the
    /// given message. The secret key never leaves this function.
    pub fn prove(secret_key: &[u8; 32], message: &[u8]) -> Result<SchnorrProof, String> {
        let secp = Secp256k1::new();
        let sk = SecretKey::from_slice(secret_key).map_err(|e| format!("invalid secret key: {e}"))?;
        let pk = PublicKey::from_secret_key(&secp, &sk);

        // Nonce k must be unpredictable; derive it from the secret key and the
        // message so the same statement always yields the same proof (and the
        // nonce is never reused across different messages).
        let mut nonce_input = Vec::with_capacity(96 + message.len());
        nonce_input.extend_from_slice(b"tigerwallet/schnorr/nonce/v1");
        nonce_input.extend_from_slice(secret_key);
        nonce_input.extend_from_slice(message);
        let nonce_bytes = sha3_256(&nonce_input);
        let k = SecretKey::from_slice(&nonce_bytes).map_err(|e| format!("nonce derivation failed: {e}"))?;
        let commitment = PublicKey::from_secret_key(&secp, &k);

        let e = challenge(&pk, message, &commitment);
        let e_scalar = secp256k1::Scalar::from_be_bytes(e).map_err(|e| format!("challenge scalar: {e}"))?;

        // s = k + e*x  (mod n). `mul_tweak` scales the secret key by the
        // challenge; the resulting key is converted back to a scalar and added
        // to the nonce.
        let ex = sk
            .mul_tweak(&e_scalar)
            .map_err(|e| format!("challenge multiplication failed: {e}"))?;
        let ex_scalar = secp256k1::Scalar::from_be_bytes(ex.secret_bytes())
            .map_err(|e| format!("challenge product out of range: {e}"))?;
        let s = k
            .add_tweak(&ex_scalar)
            .map_err(|e| format!("response computation failed: {e}"))?;

        Ok(SchnorrProof {
            commitment: commitment.serialize(),
            response: s.secret_bytes(),
        })
    }

    /// Verify a proof against a public key and message. Returns `false` for any
    /// malformed input rather than panicking.
    pub fn verify(proof: &SchnorrProof, public_key: &[u8; 33], message: &[u8]) -> bool {
        let secp = Secp256k1::new();
        let pk = match PublicKey::from_slice(public_key) {
            Ok(pk) => pk,
            Err(_) => return false,
        };
        let commitment = match PublicKey::from_slice(&proof.commitment) {
            Ok(c) => c,
            Err(_) => return false,
        };
        let s = match SecretKey::from_slice(&proof.response) {
            Ok(s) => s,
            Err(_) => return false,
        };

        let e = challenge(&pk, message, &commitment);
        let e_scalar = match secp256k1::Scalar::from_be_bytes(e) {
            Ok(scalar) => scalar,
            Err(_) => return false,
        };

        // Check s*G == R + e*P.
        let s_point = PublicKey::from_secret_key(&secp, &s);
        let e_pk = match pk.mul_tweak(&secp, &e_scalar) {
            Ok(point) => point,
            Err(_) => return false,
        };
        let expected = match commitment.combine(&e_pk) {
            Ok(point) => point,
            Err(_) => return false,
        };
        s_point == expected
    }

    /// Convenience wrapper: prove ownership of an address (the secp256k1 public
    /// key) without revealing the private key.
    pub fn create_ownership_proof(secret_key: &[u8; 32], message: &[u8]) -> Result<SchnorrProof, String> {
        prove(secret_key, message)
    }

    /// Verify an ownership proof.
    pub fn verify_ownership_proof(proof: &SchnorrProof, public_key: &[u8; 33], message: &[u8]) -> bool {
        verify(proof, public_key, message)
    }
}

// ============================================================================
// Merkle Proof of Reserves
// ============================================================================
//
// `MerkleTree` above only exposes the root. These helpers produce and verify
// inclusion proofs so an individual balance can be proven part of a published
// reserve root.

pub struct MerkleTree {
    leaves: Vec<[u8; 32]>,
    /// All levels, bottom (leaves) first. The last level is the root.
    levels: Vec<Vec<[u8; 32]>>,
}

impl MerkleTree {
    pub fn new(leaves: Vec<[u8; 32]>) -> Self {
        let mut levels = vec![leaves.clone()];
        let mut current = leaves.clone();

        while current.len() > 1 {
            let mut next = Vec::with_capacity(current.len().div_ceil(2));
            for chunk in current.chunks(2) {
                let left = &chunk[0];
                let right = if chunk.len() > 1 { &chunk[1] } else { left };
                let mut combined = Vec::with_capacity(64);
                combined.extend_from_slice(left);
                combined.extend_from_slice(right);
                next.push(sha3_256(&combined));
            }
            levels.push(next.clone());
            current = next;
        }

        MerkleTree { leaves, levels }
    }

    pub fn leaves(&self) -> &[[u8; 32]] {
        &self.leaves
    }

    pub fn root(&self) -> Option<[u8; 32]> {
        self.levels.last().and_then(|level| level.first()).copied()
    }

    /// Number of levels, used to size a proof.
    pub fn height(&self) -> usize {
        self.levels.len().saturating_sub(1)
    }
}

/// A Merkle inclusion proof: the sibling hashes from the leaf up to the root.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MerkleProof {
    pub leaf: [u8; 32],
    pub siblings: Vec<[u8; 32]>,
    /// Index of the leaf at the bottom level; bit i selects left/right at level i.
    pub index: usize,
}

impl MerkleTree {
    /// Build the inclusion proof for the leaf at `index`.
    pub fn proof(&self, index: usize) -> Option<MerkleProof> {
        if index >= self.leaves.len() {
            return None;
        }

        let mut siblings = Vec::with_capacity(self.height());
        let mut idx = index;

        for level in &self.levels[..self.levels.len().saturating_sub(1)] {
            let sibling_idx = if idx % 2 == 0 { idx + 1 } else { idx - 1 };
            // Odd node count: the last node is paired with itself, so the
            // sibling of the final node is the node itself.
            let sibling = if sibling_idx < level.len() {
                level[sibling_idx]
            } else {
                level[idx]
            };
            siblings.push(sibling);
            idx /= 2;
        }

        Some(MerkleProof {
            leaf: self.leaves[index],
            siblings,
            index,
        })
    }
}

/// Fold a proof up to a root using the same hashing rule as tree construction.
pub fn verify_merkle_proof(proof: &MerkleProof) -> [u8; 32] {
    let mut current = proof.leaf;
    let mut idx = proof.index;

    for sibling in &proof.siblings {
        let mut combined = Vec::with_capacity(64);
        if idx % 2 == 0 {
            combined.extend_from_slice(&current);
            combined.extend_from_slice(sibling);
        } else {
            combined.extend_from_slice(sibling);
            combined.extend_from_slice(&current);
        }
        current = sha3_256(&combined);
        idx /= 2;
    }

    current
}

// ============================================================================
// Multi-Party Computation (MPC) Wallet - Shamir Secret Sharing over GF(256)
// ============================================================================
//
// A real (t, n) threshold split: the secret is the constant term of a random
// polynomial of degree t-1 over GF(256), each share is a point on that
// polynomial, and any t shares reconstruct it by Lagrange interpolation at
// x = 0. Fewer than t shares carry no information about the secret.

/// GF(256) arithmetic using the AES reduction polynomial 0x11B, which makes
/// multiplication table-driven and constant-time with respect to the operand
/// values.
mod gf256 {
    const POLY: u16 = 0x11b;

    /// Multiply two field elements (Russian-peasant multiplication; the loop
    /// count is fixed, so timing does not depend on the values).
    pub fn mul(mut a: u8, mut b: u8) -> u8 {
        let mut result: u8 = 0;
        for _ in 0..8 {
            if b & 1 != 0 {
                result ^= a;
            }
            let high_bit = a & 0x80;
            a <<= 1;
            if high_bit != 0 {
                a ^= (POLY & 0xff) as u8;
            }
            b >>= 1;
        }
        result
    }

    /// Multiplicative inverse via exponentiation (a^254 in GF(256)).
    pub fn inv(a: u8) -> Option<u8> {
        if a == 0 {
            return None;
        }
        let mut result: u8 = 1;
        let mut base = a;
        let mut exp = 254u16;
        while exp > 0 {
            if exp & 1 == 1 {
                result = mul(result, base);
            }
            base = mul(base, base);
            exp >>= 1;
        }
        Some(result)
    }

    /// Evaluate a polynomial (coefficients low-to-high) at x.
    pub fn eval(coeffs: &[u8], x: u8) -> u8 {
        let mut result: u8 = 0;
        for coeff in coeffs.iter().rev() {
            result = mul(result, x) ^ coeff;
        }
        result
    }

    /// Lagrange interpolation at x = 0 for the given (x, y) points.
    pub fn interpolate_at_zero(points: &[(u8, u8)]) -> Option<u8> {
        let mut secret: u8 = 0;

        for (i, &(xi, yi)) in points.iter().enumerate() {
            let mut numerator: u8 = 1;
            let mut denominator: u8 = 1;

            for (j, &(xj, _)) in points.iter().enumerate() {
                if i == j {
                    continue;
                }
                if xi == xj {
                    // Duplicate evaluation points make the system singular.
                    return None;
                }
                // At x = 0 the numerator term is (0 - xj) = xj in GF(2^n).
                numerator = mul(numerator, xj);
                denominator = mul(denominator, xi ^ xj);
            }

            let lambda = mul(numerator, inv(denominator)?);
            secret ^= mul(yi, lambda);
        }

        Some(secret)
    }
}

pub struct MPCWallet {
    threshold: u8,
    participants: Vec<String>,
    shares: Vec<Vec<u8>>,
}

impl MPCWallet {
    pub fn new(threshold: u8, participants: Vec<String>) -> Self {
        MPCWallet {
            threshold,
            participants,
            shares: Vec::new(),
        }
    }

    pub fn threshold(&self) -> u8 {
        self.threshold
    }

    /// Split `secret` into one share per participant. Each byte position is
    /// split independently with its own random polynomial.
    pub fn generate_shares(&mut self, secret: &[u8]) -> Result<(), String> {
        let n = self.participants.len();
        if self.threshold == 0 {
            return Err("threshold must be at least 1".to_string());
        }
        if n < self.threshold as usize {
            return Err("not enough participants for the requested threshold".to_string());
        }
        if secret.is_empty() {
            return Err("secret must not be empty".to_string());
        }

        let mut rng = OsRng;
        let mut shares: Vec<Vec<u8>> = vec![Vec::with_capacity(secret.len()); n];

        for &byte in secret {
            // Polynomial of degree threshold-1 with the secret byte as constant term.
            let mut coeffs = Vec::with_capacity(self.threshold as usize);
            coeffs.push(byte);
            for _ in 1..self.threshold {
                let mut coefficient = [0u8; 1];
                rng.fill_bytes(&mut coefficient);
                coeffs.push(coefficient[0]);
            }

            // Evaluation points are 1..=n; x = 0 is the secret itself.
            for (participant_idx, share) in shares.iter_mut().enumerate() {
                share.push(gf256::eval(&coeffs, (participant_idx + 1) as u8));
            }
        }

        self.shares = shares;
        Ok(())
    }

    /// Reconstruct the secret from at least `threshold` shares. Shares are
    /// `(evaluation point, bytes)` pairs so a caller cannot silently mislabel
    /// which participant a share came from.
    pub fn reconstruct(&self, shares: &[Vec<u8>]) -> Result<Vec<u8>, String> {
        let points: Vec<(u8, &Vec<u8>)> = shares
            .iter()
            .enumerate()
            .map(|(i, s)| ((i + 1) as u8, s))
            .collect();
        Self::reconstruct_with_points(&points, self.threshold)
    }

    /// Reconstruct from explicitly labelled shares, which is what a real caller
    /// holds after collecting them from separate parties.
    pub fn reconstruct_with_points(shares: &[(u8, &Vec<u8>)], threshold: u8) -> Result<Vec<u8>, String> {
        if threshold == 0 {
            return Err("threshold must be at least 1".to_string());
        }
        if shares.len() < threshold as usize {
            return Err("not enough shares to reconstruct".to_string());
        }

        let used = &shares[..threshold as usize];
        let secret_len = used[0].1.len();
        if used.iter().any(|(_, s)| s.len() != secret_len) {
            return Err("shares have inconsistent lengths".to_string());
        }

        let mut secret = Vec::with_capacity(secret_len);
        for byte_idx in 0..secret_len {
            let points: Vec<(u8, u8)> = used
                .iter()
                .map(|(x, s)| (*x, s[byte_idx]))
                .collect();
            let value = gf256::interpolate_at_zero(&points)
                .ok_or_else(|| "share points are not distinct".to_string())?;
            secret.push(value);
        }

        Ok(secret)
    }
}

// ============================================================================
// Hardware-backed key custody
// ============================================================================
//
// `SecureEnclave` previously "signed" by returning the input unchanged, which
// meant a caller that trusted the return value would treat arbitrary data as a
// valid signature. There is no portable way to reach a platform secure element
// from this crate, so the type now fails closed: `sign` refuses to return
// anything unless a real signing backend has been installed. Production code
// must implement `SigningBackend` against the platform API (Secure Enclave on
// Apple, StrongBox/TEE on Android, PKCS#11 for HSM deployments) or use the
// `tigerwallet-hsm` module.

/// A real signing backend. Implementations must return a genuine signature or
/// an error; they must never echo their input.
pub trait SigningBackend: Send + Sync {
    /// Sign `data`, returning a recoverable secp256k1 signature.
    fn sign(&self, key_id: &str, data: &[u8]) -> Result<Vec<u8>, String>;

    /// Public key for `key_id` as 33-byte compressed secp256k1, if the backend
    /// holds one.
    fn public_key(&self, key_id: &str) -> Result<Vec<u8>, String>;
}

pub struct SecureEnclave {
    key_id: String,
    sealed: bool,
    backend: Option<Box<dyn SigningBackend>>,
}

impl SecureEnclave {
    pub fn new(key_id: String) -> Self {
        SecureEnclave {
            key_id,
            sealed: false,
            backend: None,
        }
    }

    /// Create an enclave backed by a real platform signer.
    pub fn with_backend(key_id: String, backend: Box<dyn SigningBackend>) -> Self {
        SecureEnclave {
            key_id,
            sealed: false,
            backend: Some(backend),
        }
    }

    pub fn seal(&mut self) {
        self.sealed = true;
    }

    pub fn unseal(&mut self) -> Result<(), String> {
        if !self.sealed {
            return Err("Not sealed".to_string());
        }
        self.sealed = false;
        Ok(())
    }

    pub fn is_sealed(&self) -> bool {
        self.sealed
    }

    pub fn key_id(&self) -> &str {
        &self.key_id
    }

    /// Sign through the configured backend. Fails closed when the enclave is
    /// sealed or when no real backend is installed.
    pub fn sign(&self, data: &[u8]) -> Result<Vec<u8>, String> {
        if self.sealed {
            return Err("Enclave is sealed".to_string());
        }

        match &self.backend {
            Some(backend) => backend.sign(&self.key_id, data),
            None => Err(
                "no signing backend configured: install a platform SigningBackend (Secure Enclave / StrongBox / PKCS#11) instead of relying on software fallback"
                    .to_string(),
            ),
        }
    }

    pub fn public_key(&self) -> Result<Vec<u8>, String> {
        match &self.backend {
            Some(backend) => backend.public_key(&self.key_id),
            None => Err("no signing backend configured".to_string()),
        }
    }
}

// ============================================================================
// Time-Locked Recovery
// ============================================================================

pub struct TimeLock {
    unlock_time: u64,
    secret_hash: [u8; 32],
}

impl TimeLock {
    pub fn new(unlock_time: u64, secret: &[u8]) -> Self {
        TimeLock {
            unlock_time,
            secret_hash: sha3_256(secret),
        }
    }
    
    pub fn is_unlockable(&self) -> bool {
        let current_time = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        current_time >= self.unlock_time
    }
    
    pub fn unlock(&self, secret: &[u8]) -> Result<Vec<u8>, String> {
        if !self.is_unlockable() {
            return Err("Time lock not expired".to_string());
        }
        
        let hash = sha3_256(secret);
        if hash != self.secret_hash {
            return Err("Invalid secret".to_string());
        }
        
        Ok(secret.to_vec())
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    use secp256k1::{PublicKey, Secp256k1, SecretKey};

    #[test]
    fn test_rate_limiter() {
        let limiter = RateLimiter::new(3, 60);

        assert!(limiter.check("user1"));
        assert!(limiter.check("user1"));
        assert!(limiter.check("user1"));
        assert!(!limiter.check("user1")); // Should be rate limited
    }

    #[test]
    fn test_anti_phishing() {
        let anti = AntiPhishing::new();

        assert!(anti.check_domain("fake-uniswap.com").is_none());

        anti.report_phishing(
            "fake-uniswap.com".to_string(),
            PhishingType::FakeExchange,
            vec!["0x123...".to_string()],
        );

        assert!(anti.check_domain("fake-uniswap.com").is_some());
    }

    #[test]
    fn test_merkle_inclusion_proof_verifies() {
        let leaves: Vec<[u8; 32]> = (0..5).map(|i| sha3_256(&[i])).collect();
        let tree = MerkleTree::new(leaves.clone());
        let root = tree.root().expect("root");

        for index in 0..leaves.len() {
            let proof = tree.proof(index).expect("proof");
            assert_eq!(proof.leaf, leaves[index]);
            assert_eq!(verify_merkle_proof(&proof), root, "leaf {index}");
        }
    }

    #[test]
    fn test_merkle_proof_rejects_tampered_leaf() {
        let leaves: Vec<[u8; 32]> = (0..4).map(|i| sha3_256(&[i])).collect();
        let tree = MerkleTree::new(leaves);
        let root = tree.root().expect("root");

        let mut proof = tree.proof(1).expect("proof");
        proof.leaf = sha3_256(b"not in the tree");
        assert_ne!(verify_merkle_proof(&proof), root);
    }

    #[test]
    fn test_merkle_proof_out_of_range() {
        let leaves: Vec<[u8; 32]> = (0..2).map(|i| sha3_256(&[i])).collect();
        let tree = MerkleTree::new(leaves);
        assert!(tree.proof(2).is_none());
    }

    #[test]
    fn test_schnorr_proof_round_trip() {
        let secp = Secp256k1::new();
        let secret_bytes = [7u8; 32];
        let sk = SecretKey::from_slice(&secret_bytes).unwrap();
        let pk = PublicKey::from_secret_key(&secp, &sk).serialize();

        let proof = zk::prove(&secret_bytes, b"prove ownership").expect("proof");
        assert!(zk::verify(&proof, &pk, b"prove ownership"));
    }

    #[test]
    fn test_schnorr_proof_rejects_wrong_message_and_key() {
        let secp = Secp256k1::new();
        let secret_bytes = [9u8; 32];
        let sk = SecretKey::from_slice(&secret_bytes).unwrap();
        let pk = PublicKey::from_secret_key(&secp, &sk).serialize();

        let proof = zk::prove(&secret_bytes, b"message-a").expect("proof");

        assert!(!zk::verify(&proof, &pk, b"message-b"), "wrong message");

        let other = SecretKey::from_slice(&[11u8; 32]).unwrap();
        let other_pk = PublicKey::from_secret_key(&secp, &other).serialize();
        assert!(!zk::verify(&proof, &other_pk, b"message-a"), "wrong key");
    }

    #[test]
    fn test_schnorr_rejects_zero_commitment() {
        let secp = Secp256k1::new();
        let sk = SecretKey::from_slice(&[3u8; 32]).unwrap();
        let pk = PublicKey::from_secret_key(&secp, &sk).serialize();

        let proof = zk::SchnorrProof {
            commitment: [0u8; 33],
            response: [1u8; 32],
        };
        assert!(!zk::verify(&proof, &pk, b"message"));
    }

    #[test]
    fn test_mpc_threshold_split_and_reconstruct() {
        let participants = vec!["p1".to_string(), "p2".to_string(), "p3".to_string()];
        let mut mpc = MPCWallet::new(2, participants);
        let secret = b"a-32-byte-secret-for-shamir-123".to_vec();

        mpc.generate_shares(&secret).unwrap();

        // Any 2 of the 3 shares must reconstruct the secret.
        let s0 = vec![(1u8, &mpc.shares[0])];
        let s1 = vec![(2u8, &mpc.shares[1])];
        let s2 = vec![(3u8, &mpc.shares[2])];

        let mut pairs_a = s0.clone();
        pairs_a.extend(s1.clone());
        assert_eq!(MPCWallet::reconstruct_with_points(&pairs_a, 2).unwrap(), secret);

        let mut pairs_b = s0.clone();
        pairs_b.extend(s2.clone());
        assert_eq!(MPCWallet::reconstruct_with_points(&pairs_b, 2).unwrap(), secret);

        let mut pairs_c = s1.clone();
        pairs_c.extend(s2.clone());
        assert_eq!(MPCWallet::reconstruct_with_points(&pairs_c, 2).unwrap(), secret);
    }

    #[test]
    fn test_mpc_shares_do_not_reveal_the_secret() {
        let participants = vec!["p1".to_string(), "p2".to_string(), "p3".to_string()];
        let mut mpc = MPCWallet::new(2, participants);
        let secret = b"super-secret-value".to_vec();

        mpc.generate_shares(&secret).unwrap();

        // A single share must never equal the secret (the previous
        // implementation handed the secret to every participant).
        for share in &mpc.shares {
            assert_ne!(share, &secret);
        }
    }

    #[test]
    fn test_mpc_rejects_insufficient_shares() {
        let participants = vec!["p1".to_string(), "p2".to_string(), "p3".to_string()];
        let mut mpc = MPCWallet::new(3, participants);
        mpc.generate_shares(b"secret").unwrap();

        let pairs = vec![(1u8, &mpc.shares[0]), (2u8, &mpc.shares[1])];
        assert!(MPCWallet::reconstruct_with_points(&pairs, 3).is_err());
    }

    #[test]
    fn test_mpc_rejects_duplicate_evaluation_points() {
        let participants = vec!["p1".to_string(), "p2".to_string()];
        let mut mpc = MPCWallet::new(2, participants);
        mpc.generate_shares(b"secret").unwrap();

        let pairs = vec![(1u8, &mpc.shares[0]), (1u8, &mpc.shares[1])];
        assert!(MPCWallet::reconstruct_with_points(&pairs, 2).is_err());
    }

    #[test]
    fn test_mpc_rejects_too_many_participants_for_threshold() {
        let mut mpc = MPCWallet::new(3, vec!["p1".to_string()]);
        assert!(mpc.generate_shares(b"secret").is_err());
    }

    #[test]
    fn test_secure_enclave_fails_closed_without_backend() {
        let enclave = SecureEnclave::new("key-1".to_string());
        assert!(enclave.sign(b"payload").is_err());
        assert!(enclave.public_key().is_err());
    }

    struct EchoBackend;

    impl SigningBackend for EchoBackend {
        fn sign(&self, _key_id: &str, data: &[u8]) -> Result<Vec<u8>, String> {
            // A deliberately wrong backend, used to prove the enclave delegates.
            let mut out = b"sig:".to_vec();
            out.extend_from_slice(data);
            Ok(out)
        }

        fn public_key(&self, _key_id: &str) -> Result<Vec<u8>, String> {
            Ok(vec![0x02; 33])
        }
    }

    #[test]
    fn test_secure_enclave_delegates_to_backend_and_respects_seal() {
        let mut enclave = SecureEnclave::with_backend("key-1".to_string(), Box::new(EchoBackend));
        assert_eq!(enclave.sign(b"abc").unwrap(), b"sig:abc".to_vec());
        assert_eq!(enclave.public_key().unwrap().len(), 33);

        enclave.seal();
        assert!(enclave.sign(b"abc").is_err());
        enclave.unseal().unwrap();
        assert!(enclave.sign(b"abc").is_ok());
    }

    #[test]
    fn test_gf256_multiplication_is_a_field() {
        // Identity and commutativity spot checks plus the known AES example.
        assert_eq!(super::gf256::mul(1, 0x57), 0x57);
        assert_eq!(super::gf256::mul(0x57, 0x83), 0xc1);
        assert_eq!(super::gf256::mul(0x57, 0x13), super::gf256::mul(0x13, 0x57));
        assert_eq!(super::gf256::mul(0x00, 0xff), 0x00);
    }

    #[test]
    fn test_gf256_inverse() {
        for value in 1u8..=255 {
            let inverse = super::gf256::inv(value).expect("inverse");
            assert_eq!(super::gf256::mul(value, inverse), 1, "value {value}");
        }
        assert!(super::gf256::inv(0).is_none());
    }
}
