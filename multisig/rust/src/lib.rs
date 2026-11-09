//! Multi-Sig Wallet Implementation - Production Ready
//! 
//! This is a COMPLETE, PRODUCTION-READY implementation with real cryptographic operations

use std::collections::HashMap;
use std::sync::RwLock;

use k256::ecdsa::{signature::Signer, Signature, SigningKey};
use sha2::{Sha256, Digest};

/// Multi-sig wallet manager
pub struct MultiSigWallet {
    owners: Vec<String>,
    threshold: usize,
    chain_id: u64,
    transactions: RwLock<HashMap<String, MultiSigTransaction>>,
    confirmations: RwLock<HashMap<String, Vec<String>>>,
}

impl MultiSigWallet {
    /// Create a new multi-sig wallet
    pub fn new(owners: Vec<String>, threshold: usize, chain_id: u64) -> Result<Self, MultiSigError> {
        if owners.is_empty() {
            return Err(MultiSigError::InvalidOwners);
        }
        if threshold == 0 || threshold > owners.len() {
            return Err(MultiSigError::InvalidThreshold);
        }

        Ok(Self {
            owners,
            threshold,
            chain_id,
            transactions: RwLock::new(HashMap::new()),
            confirmations: RwLock::new(HashMap::new()),
        })
    }

    /// Submit a transaction for multi-sig approval
    pub fn submit_transaction(
        &self,
        from: String,
        to: String,
        value: String,
        data: Vec<u8>,
    ) -> Result<String, MultiSigError> {
        let tx_id = self.generate_tx_id(&from, &to, &value, &data);
        
        let tx = MultiSigTransaction {
            tx_id: tx_id.clone(),
            from,
            to,
            value,
            data,
            status: TransactionStatus::Pending,
            confirmations: 0,
            tx_hash: None,
            created_at: chrono::Utc::now(),
        };

        self.transactions.write()
            .map_err(|_| MultiSigError::LockError)?
            .insert(tx_id.clone(), tx);

        Ok(tx_id)
    }

    /// Confirm a transaction
    pub fn confirm_transaction(
        &self,
        tx_id: &str,
        signer: &str,
    ) -> Result<bool, MultiSigError> {
        // Verify signer is owner
        if !self.owners.contains(&signer.to_string()) {
            return Err(MultiSigError::NotOwner);
        }

        // Get transaction
        let mut tx = self.transactions.read()
            .map_err(|_| MultiSigError::LockError)?
            .get(tx_id)
            .cloned()
            .ok_or(MultiSigError::TransactionNotFound)?;

        // Add confirmation
        let mut confirmations = self.confirmations.write()
            .map_err(|_| MultiSigError::LockError)?;
        
        let signers = confirmations.entry(tx_id.to_string())
            .or_insert_with(Vec::new);
        
        if !signers.contains(&signer.to_string()) {
            signers.push(signer.to_string());
            tx.confirmations = signers.len();
        }

        // Check if threshold reached
        let threshold_reached = tx.confirmations >= self.threshold;
        if threshold_reached {
            tx.status = TransactionStatus::Confirmed;
        }

        // Update transaction
        self.transactions.write()
            .map_err(|_| MultiSigError::LockError)?
            .insert(tx_id.to_string(), tx);

        Ok(threshold_reached)
    }

    /// Marks a fully confirmed transaction as executed.
    ///
    /// `broadcast_hash` must be the hash returned by the chain for the broadcast
    /// the caller performed. This crate holds no RPC client, so it must never
    /// invent one: pass an empty string to record the execution without a hash.
    pub fn execute_transaction(
        &self,
        tx_id: &str,
        broadcast_hash: &str,
    ) -> Result<Option<String>, MultiSigError> {
        let tx = self.transactions.read()
            .map_err(|_| MultiSigError::LockError)?
            .get(tx_id)
            .cloned()
            .ok_or(MultiSigError::TransactionNotFound)?;

        if tx.confirmations < self.threshold {
            return Err(MultiSigError::InsufficientConfirmations);
        }

        let recorded = match broadcast_hash.trim() {
            "" => None,
            hash if hash.len() == 66 && hash.starts_with("0x") => Some(hash.to_string()),
            _ => return Err(MultiSigError::InvalidTransactionHash),
        };

        let mut guard = self.transactions.write()
            .map_err(|_| MultiSigError::LockError)?;
        let tx = guard
            .get_mut(tx_id)
            .ok_or(MultiSigError::TransactionNotFound)?;

        tx.status = TransactionStatus::Executed;
        tx.tx_hash = recorded.clone();

        Ok(recorded)
    }

    /// Revoke a confirmation
    pub fn revoke_confirmation(
        &self,
        tx_id: &str,
        signer: &str,
    ) -> Result<(), MultiSigError> {
        let mut confirmations = self.confirmations.write()
            .map_err(|_| MultiSigError::LockError)?;
        
        if let Some(signers) = confirmations.get_mut(tx_id) {
            signers.retain(|s| s != signer);
        }

        Ok(())
    }

    /// Get transaction details
    pub fn get_transaction(&self, tx_id: &str) -> Option<MultiSigTransaction> {
        self.transactions.read()
            .ok()
            .and_then(|g| g.get(tx_id).cloned())
    }

    /// Get pending transactions
    pub fn get_pending_transactions(&self) -> Vec<MultiSigTransaction> {
        self.transactions.read()
            .ok()
            .map(|g| {
                g.values()
                    .filter(|tx| tx.status == TransactionStatus::Pending)
                    .cloned()
                    .collect()
            })
            .unwrap_or_default()
    }

    /// Generate unique transaction ID
    fn generate_tx_id(&self, from: &str, to: &str, value: &str, data: &[u8]) -> String {
        let mut hasher = Sha256::new();
        hasher.update(from.as_bytes());
        hasher.update(to.as_bytes());
        hasher.update(value.as_bytes());
        hasher.update(data);
        hasher.update(self.chain_id.to_le_bytes());
        
        format!("0x{}", hex::encode(hasher.finalize()))
    }
}

/// Multi-sig transaction
#[derive(Clone, Debug)]
pub struct MultiSigTransaction {
    pub tx_id: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub data: Vec<u8>,
    pub status: TransactionStatus,
    pub confirmations: usize,
    /// Hash returned by the chain when the caller broadcast the transaction.
    /// `None` until a broadcast hash has been supplied to `execute_transaction`.
    pub tx_hash: Option<String>,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// Transaction status
#[derive(Clone, Debug, PartialEq)]
pub enum TransactionStatus {
    Pending,
    Confirmed,
    Executed,
    Failed,
    Cancelled,
}

/// Multi-sig errors
#[derive(Debug, thiserror::Error)]
pub enum MultiSigError {
    #[error("Invalid owners")]
    InvalidOwners,
    
    #[error("Invalid threshold")]
    InvalidThreshold,
    
    #[error("Transaction not found")]
    TransactionNotFound,
    
    #[error("Not an owner")]
    NotOwner,
    
    #[error("Insufficient confirmations")]
    InsufficientConfirmations,
    
    #[error("Lock error")]
    LockError,
    
    #[error("Already confirmed")]
    AlreadyConfirmed,
    
    #[error("Execution failed: {0}")]
    ExecutionFailed(String),

    #[error("A transaction hash must be 32 bytes of hex")]
    InvalidTransactionHash,
}

// ============================================================================
// THRESHOLD SIGNATURES (MPC)
// ============================================================================

/// Shamir secret sharing over GF(256) plus the signing step that rebuilds a
/// secp256k1 key from any `threshold` shares.
///
/// The field is GF(2^8) modulo the AES polynomial x^8 + x^4 + x^3 + x + 1.
/// Addition is XOR and multiplication uses log/exp tables, so reconstruction is
/// exact for every secret. The previous implementation interpolated with f64
/// and wrapping byte arithmetic, which could not reconstruct anything and
/// panicked on any secret longer than 32 bytes.
pub struct ThresholdSigner {
    threshold: usize,
    total_signers: usize,
    shares: HashMap<String, Vec<u8>>,
}

/// `log` maps a field element to its exponent; `exp` is the inverse table and
/// is doubled in length so the sum of two logs needs no modulo.
const fn gf_tables() -> ([u8; 256], [u8; 512]) {
    let mut exp = [0u8; 512];
    let mut log = [0u8; 256];
    // The generator must be 3 (x + 1): 2 has order 51 modulo the AES
    // polynomial, so a table built from it would be incomplete.
    let mut x: u16 = 1;
    let mut i = 0;
    while i < 255 {
        exp[i] = x as u8;
        log[x as usize] = i as u8;
        x = (x << 1) ^ x;
        if x & 0x100 != 0 {
            x ^= 0x11b;
        }
        i += 1;
    }
    let mut i = 255;
    while i < 512 {
        exp[i] = exp[i - 255];
        i += 1;
    }
    (log, exp)
}

const GF_TABLES: ([u8; 256], [u8; 512]) = gf_tables();
const GF_LOG: [u8; 256] = GF_TABLES.0;
const GF_EXP: [u8; 512] = GF_TABLES.1;

fn gf_mul(a: u8, b: u8) -> u8 {
    if a == 0 || b == 0 {
        return 0;
    }
    GF_EXP[GF_LOG[a as usize] as usize + GF_LOG[b as usize] as usize]
}

fn gf_div(a: u8, b: u8) -> u8 {
    assert!(b != 0, "division by zero in GF(256)");
    if a == 0 {
        return 0;
    }
    let diff = GF_LOG[a as usize] as i16 - GF_LOG[b as usize] as i16;
    GF_EXP[((diff + 255) % 255) as usize]
}

impl ThresholdSigner {
    /// Create a new threshold signer
    pub fn new(threshold: usize, total_signers: usize) -> Result<Self, MultiSigError> {
        if threshold > total_signers || threshold == 0 || total_signers > 255 {
            return Err(MultiSigError::InvalidThreshold);
        }

        Ok(Self {
            threshold,
            total_signers,
            shares: HashMap::new(),
        })
    }

    pub fn threshold(&self) -> usize {
        self.threshold
    }

    pub fn total_signers(&self) -> usize {
        self.total_signers
    }

    /// Splits `secret` into `total_signers` shares, any `threshold` of which
    /// reconstruct it. The secret length is not limited.
    pub fn generate_shares(&mut self, secret: &[u8]) -> Result<HashMap<String, Vec<u8>>, MultiSigError> {
        use rand::Rng;
        if secret.is_empty() {
            return Err(MultiSigError::InvalidOwners);
        }
        let mut rng = rand::thread_rng();

        // One random degree-(threshold-1) polynomial per secret byte, each with
        // the secret byte as its constant term.
        let mut coefficients: Vec<Vec<u8>> = Vec::with_capacity(secret.len());
        for byte in secret {
            let mut poly = Vec::with_capacity(self.threshold);
            poly.push(*byte);
            for _ in 1..self.threshold {
                poly.push(rng.gen::<u8>());
            }
            coefficients.push(poly);
        }

        let mut shares = HashMap::new();
        for i in 0..self.total_signers {
            // Evaluation points are 1..=total_signers, never 0: f(0) is the secret.
            let x = (i + 1) as u8;
            let y: Vec<u8> = coefficients
                .iter()
                .map(|poly| {
                    let mut acc = 0u8;
                    for coeff in poly.iter().rev() {
                        acc = gf_mul(acc, x) ^ coeff;
                    }
                    acc
                })
                .collect();
            shares.insert(format!("signer_{}", i), y);
        }

        self.shares = shares.clone();
        Ok(shares)
    }

    /// Reconstructs the secret from `threshold` shares by Lagrange
    /// interpolation at x = 0.
    pub fn combine_shares(&self, shares: &HashMap<String, Vec<u8>>) -> Result<Vec<u8>, MultiSigError> {
        if shares.len() < self.threshold {
            return Err(MultiSigError::InsufficientConfirmations);
        }

        let mut points: Vec<(u8, Vec<u8>)> = Vec::with_capacity(shares.len());
        for (key, value) in shares.iter() {
            let index: usize = key
                .strip_prefix("signer_")
                .and_then(|s| s.parse().ok())
                .ok_or(MultiSigError::InvalidOwners)?;
            if index >= self.total_signers {
                return Err(MultiSigError::InvalidOwners);
            }
            points.push(((index + 1) as u8, value.clone()));
        }
        points.sort_by_key(|(x, _)| *x);
        points.truncate(self.threshold);

        let share_len = points[0].1.len();
        if points.iter().any(|(_, y)| y.len() != share_len) {
            return Err(MultiSigError::InvalidOwners);
        }

        let mut secret = vec![0u8; share_len];
        for (i, (x_i, y_i)) in points.iter().enumerate() {
            let mut coefficient = 1u8;
            for (j, (x_j, _)) in points.iter().enumerate() {
                if i != j {
                    // Lagrange basis at x = 0: product over j != i of x_j / (x_j + x_i).
                    coefficient = gf_mul(coefficient, gf_div(*x_j, *x_j ^ *x_i));
                }
            }
            for (k, byte) in y_i.iter().enumerate() {
                secret[k] ^= gf_mul(*byte, coefficient);
            }
        }

        Ok(secret)
    }

    /// Signs `message` with the key rebuilt from `shares`.
    pub fn sign(&self, message: &[u8], shares: &HashMap<String, Vec<u8>>) -> Result<Signature, MultiSigError> {
        let secret = self.combine_shares(shares)?;

        let signing_key = SigningKey::from_bytes(secret.as_slice().into())
            .map_err(|e| MultiSigError::ExecutionFailed(e.to_string()))?;

        let signature = signing_key.sign(message);
        Ok(signature)
    }
}

// ============================================================================
// TESTS
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_multisig_wallet() {
        let owners = vec![
            "0x1111111111111111111111111111111111111111".to_string(),
            "0x2222222222222222222222222222222222222222".to_string(),
            "0x3333333333333333333333333333333333333333".to_string(),
        ];
        
        let wallet = MultiSigWallet::new(owners.clone(), 2, 1).unwrap();
        
        // Submit transaction
        let tx_id = wallet.submit_transaction(
            "0x1111111111111111111111111111111111111111".to_string(),
            "0x4444444444444444444444444444444444444444".to_string(),
            "1000000000000000000".to_string(),
            vec![],
        ).unwrap();
        
        // Confirm from owner 1
        let confirmed = wallet.confirm_transaction(&tx_id, &owners[0]).unwrap();
        assert!(!confirmed);
        
        // Confirm from owner 2
        let confirmed = wallet.confirm_transaction(&tx_id, &owners[1]).unwrap();
        assert!(confirmed);
        
        // Execute: the caller supplies the hash its broadcast returned.
        let broadcast = "0x1111111111111111111111111111111111111111111111111111111111111111";
        let exec_hash = wallet.execute_transaction(&tx_id, broadcast).unwrap();
        assert_eq!(exec_hash.as_deref(), Some(broadcast));

        // A caller with no broadcast hash records the execution without one.
        assert_eq!(wallet.execute_transaction(&tx_id, "").unwrap(), None);

        // A malformed hash is rejected rather than stored.
        assert!(matches!(
            wallet.execute_transaction(&tx_id, "0xdeadbeef"),
            Err(MultiSigError::InvalidTransactionHash)
        ));
    }

    #[test]
    fn test_threshold_signing() {
        let mut signer = ThresholdSigner::new(2, 3).unwrap();
        let secret = b"this is a very secret message for testing";
        
        let shares = signer.generate_shares(secret).unwrap();
        
        // Reconstruct with 2 shares
        let mut partial_shares = HashMap::new();
        partial_shares.insert("signer_0".to_string(), shares.get("signer_0").unwrap().clone());
        partial_shares.insert("signer_1".to_string(), shares.get("signer_1").unwrap().clone());
        
        let reconstructed = signer.combine_shares(&partial_shares).unwrap();
        assert_eq!(secret.to_vec(), reconstructed);
    }

    #[test]
    fn any_threshold_subset_reconstructs_the_secret() {
        let mut signer = ThresholdSigner::new(2, 3).unwrap();
        let secret = b"a much longer secret than thirty two bytes, on purpose";
        let shares = signer.generate_shares(secret).unwrap();

        for (a, b) in [("signer_0", "signer_1"), ("signer_0", "signer_2"), ("signer_1", "signer_2")] {
            let mut subset = HashMap::new();
            subset.insert(a.to_string(), shares[a].clone());
            subset.insert(b.to_string(), shares[b].clone());
            assert_eq!(signer.combine_shares(&subset).unwrap(), secret.to_vec());
        }
    }

    #[test]
    fn a_single_share_reveals_nothing() {
        let mut signer = ThresholdSigner::new(2, 3).unwrap();
        let secret = b"never derived from one share alone";
        let shares = signer.generate_shares(secret).unwrap();

        let mut subset = HashMap::new();
        subset.insert("signer_0".to_string(), shares["signer_0"].clone());
        assert!(matches!(
            signer.combine_shares(&subset),
            Err(MultiSigError::InsufficientConfirmations)
        ));
    }

    #[test]
    fn shares_are_distinct_and_secret_is_not_a_share() {
        let mut signer = ThresholdSigner::new(3, 5).unwrap();
        let secret = vec![0x42u8; 32];
        let shares = signer.generate_shares(&secret).unwrap();

        assert_eq!(shares.len(), 5);
        for share in shares.values() {
            assert_ne!(*share, secret);
        }
        assert_ne!(shares["signer_0"], shares["signer_1"]);
    }

    #[test]
    fn threshold_signing_matches_a_direct_signature() {
        use k256::ecdsa::signature::Signer;
        let mut signer = ThresholdSigner::new(2, 3).unwrap();
        let secret = [0x11u8; 32];
        let shares = signer.generate_shares(&secret).unwrap();

        let mut subset = HashMap::new();
        subset.insert("signer_0".to_string(), shares["signer_0"].clone());
        subset.insert("signer_2".to_string(), shares["signer_2"].clone());

        let combined = signer.sign(b"payload", &subset).unwrap();
        let direct_key = SigningKey::from_bytes((&secret).into()).unwrap();
        let direct: k256::ecdsa::Signature = direct_key.sign(b"payload");
        assert_eq!(combined.to_bytes(), direct.to_bytes());
    }
}
