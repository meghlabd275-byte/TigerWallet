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

pub struct IPFSStorage {
    gateway_url: String,
    auth_token: Option<String>,
}

impl IPFSStorage {
    pub fn new(gateway_url: String, auth_token: Option<String>) -> Self {
        IPFSStorage {
            gateway_url,
            auth_token,
        }
    }
    
    pub fn upload(&self, data: &[u8]) -> Result<String, String> {
        // In production, upload to IPFS and return CID
        // For now, return mock CID
        Ok("QmXyZ...".to_string())
    }
    
    pub fn download(&self, cid: &str) -> Result<Vec<u8>, String> {
        // In production, download from IPFS
        Ok(Vec::new())
    }
    
    pub fn pin(&self, cid: &str) -> Result<(), String> {
        Ok(())
    }
    
    pub fn unpin(&self, cid: &str) -> Result<(), String> {
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
// Zero-Knowledge Proof (zkSNARK)
// ============================================================================

pub mod zk {
    use crate::crypto::{sha256, sha3_256};
    
    // Simplified zkSNARK implementation for demonstration
    // Production would use bellman or arkworks
    
    pub struct ZKProof {
        pub a: [u8; 32],
        pub b: [u8; 64],
        pub c: [u8; 32],
    }
    
    pub struct ZKVerifier;
    
    impl ZKVerifier {
        pub fn verify(proof: &ZKProof, public_inputs: &[u8]) -> bool {
            // Simplified verification
            // Real implementation would verify the cryptographic proof
            !proof.a.iter().all(|&b| b == 0)
        }
    }
    
    // Create a proof that the prover knows a secret
    pub fn create_proof(secret: &[u8], public_inputs: &[u8]) -> ZKProof {
        // Simplified: hash-based proof
        // Real implementation would use Groth16 or PLONK
        
        let mut input = Vec::new();
        input.extend_from_slice(public_inputs);
        input.extend_from_slice(secret);
        
        let hash = sha3_256(&input);
        
        ZKProof {
            a: hash,
            b: [0u8; 64],
            c: sha256(secret),
        }
    }
    
    // Verify ownership without revealing address
    pub fn create_ownership_proof(private_key: &[u8; 32], message: &[u8]) -> ZKProof {
        create_proof(private_key, message)
    }
}

// ============================================================================
// Secure Enclave Simulation
// ============================================================================

pub struct SecureEnclave {
    key_id: String,
    sealed: bool,
}

impl SecureEnclave {
    pub fn new(key_id: String) -> Self {
        SecureEnclave {
            key_id,
            sealed: false,
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
    
    pub fn sign(&self, data: &[u8]) -> Result<Vec<u8>, String> {
        if self.sealed {
            return Err("Enclave is sealed".to_string());
        }
        
        // Simulated signing
        Ok(data.to_vec())
    }
}

// ============================================================================
// Merkle Tree for Proof of Reserves
// ============================================================================

pub struct MerkleTree {
    leaves: Vec<[u8; 32]>,
    nodes: Vec<[u8; 32]>,
}

impl MerkleTree {
    pub fn new(leaves: Vec<[u8; 32]>) -> Self {
        let mut tree = MerkleTree {
            leaves: leaves.clone(),
            nodes: Vec::new(),
        };
        
        tree.build();
        tree
    }
    
    fn build(&mut self) {
        let mut current_level = self.leaves.clone();
        
        while current_level.len() > 1 {
            let mut next_level = Vec::new();
            
            for chunk in current_level.chunks(2) {
                let left = &chunk[0];
                let right = if chunk.len() > 1 { &chunk[1] } else { left };
                
                // Hash pair
                let mut combined = Vec::with_capacity(64);
                combined.extend_from_slice(left);
                combined.extend_from_slice(right);
                
                let hash = sha3_256(&combined);
                next_level.push(hash);
            }
            
            self.nodes.extend(current_level.clone());
            current_level = next_level;
        }
        
        self.nodes.extend(current_level);
    }
    
    pub fn root(&self) -> Option<[u8; 32]> {
        self.nodes.last().cloned()
    }
    
    pub fn prove(&self, index: usize) -> Option<Vec<[u8; 32]>> {
        if index >= self.leaves.len() {
            return None;
        }
        
        let mut proof = Vec::new();
        let mut idx = index;
        
        for level in 0.. {
            let level_size = self.leaves.len() / (2u32.pow(level) as usize);
            if level_size == 0 {
                break;
            }
            
            let sibling = if idx % 2 == 0 { idx + 1 } else { idx - 1 };
            
            if sibling < level_size {
                proof.push(self.leaves[sibling]);
            }
            
            idx /= 2;
        }
        
        Some(proof)
    }
    
    pub fn verify(proof: &[u8; 32], root: &[u8; 32], leaf: &[u8; 32]) -> bool {
        let mut current = sha3_256(leaf);
        
        for p in proof {
            let mut combined = Vec::with_capacity(64);
            combined.extend_from_slice(&current);
            combined.extend_from_slice(p);
            current = sha3_256(&combined);
        }
        
        current == *root
    }
}

// ============================================================================
// Multi-Party Computation (MPC) Wallet
// ============================================================================

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
    
    // Simplified Shamir's Secret Sharing
    pub fn generate_shares(&mut self, secret: &[u8]) -> Result<(), String> {
        if self.participants.len() < self.threshold as usize {
            return Err("Not enough participants".to_string());
        }
        
        // Simplified: just distribute secret
        // Real implementation would use polynomial secret sharing
        self.shares = self.participants.iter()
            .map(|_| secret.to_vec())
            .collect();
        
        Ok(())
    }
    
    pub fn reconstruct(&self, shares: &[Vec<u8>]) -> Result<Vec<u8>, String> {
        if shares.len() < self.threshold as usize {
            return Err("Not enough shares".to_string());
        }
        
        // Simplified: just return first share
        // Real implementation would use Lagrange interpolation
        Ok(shares[0].clone())
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
    fn test_merkle_tree() {
        let leaves: Vec<[u8; 32]> = (0..4)
            .map(|i| sha3_256(&[i]))
            .collect();
        
        let tree = MerkleTree::new(leaves);
        assert!(tree.root().is_some());
    }
    
    #[test]
    fn test_mpc_wallet() {
        let participants = vec!["0x1".to_string(), "0x2".to_string(), "0x3".to_string()];
        let mut mpc = MPCWallet::new(2, participants);
        
        mpc.generate_shares(b"secret").unwrap();
        
        let reconstructed = mpc.reconstruct(&[b"secret".to_vec(), b"secret".to_vec()]).unwrap();
        assert_eq!(reconstructed, b"secret");
    }
}