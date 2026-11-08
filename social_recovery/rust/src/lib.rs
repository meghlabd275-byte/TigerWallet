/**
 * TigerWallet Social Recovery System - Production-Ready Rust Implementation
 * Guardian-based wallet recovery with threshold signatures
 * Ultra-low latency design with concurrent processing
 */

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use sha2::{Sha256, Digest};
use ring::signature::{Ed25519KeyPair, KeyPair, UnparsedPublicKey, ED25519};
use base64::{Engine as _, engine::general_purpose};
use chrono::{DateTime, Utc};

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecoveryError {
    InvalidSignature,
    InvalidGuardian,
    GuardianNotFound,
    InsufficientGuardians,
    RecoveryInProgress,
    RecoveryAlreadyCompleted,
    RecoveryExpired,
    InvalidThreshold,
    InvalidUser,
    InvalidAddress,
    DatabaseError(String),
    NetworkError(String),
}

impl std::fmt::Display for RecoveryError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            RecoveryError::InvalidSignature => write!(f, "Invalid signature"),
            RecoveryError::InvalidGuardian => write!(f, "Invalid guardian address"),
            RecoveryError::GuardianNotFound => write!(f, "Guardian not found"),
            RecoveryError::InsufficientGuardians => write!(f, "Insufficient guardians for recovery"),
            RecoveryError::RecoveryInProgress => write!(f, "Recovery already in progress"),
            RecoveryError::RecoveryAlreadyCompleted => write!(f, "Recovery already completed"),
            RecoveryError::RecoveryExpired => write!(f, "Recovery request expired"),
            RecoveryError::InvalidThreshold => write!(f, "Invalid threshold configuration"),
            RecoveryError::InvalidUser => write!(f, "Invalid user address"),
            RecoveryError::InvalidAddress => write!(f, "Invalid address"),
            RecoveryError::DatabaseError(msg) => write!(f, "Database error: {}", msg),
            RecoveryError::NetworkError(msg) => write!(f, "Network error: {}", msg),
        }
    }
}

impl std::error::Error for RecoveryError {}

// ============================================================================
// Data Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Guardian {
    pub address: String,
    pub name: String,
    pub public_key: Option<String>,
    pub verified: bool,
    pub added_at: u64,
    pub last_activity: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GuardianSet {
    pub user_address: String,
    pub guardians: Vec<Guardian>,
    pub threshold: u8,
    pub locked: bool,
    pub created_at: u64,
    pub updated_at: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryRequest {
    pub id: String,
    pub user_address: String,
    pub new_address: String,
    pub initiated_at: u64,
    pub expires_at: u64,
    pub completed_at: Option<u64>,
    pub signatures: Vec<GuardianSignature>,
    pub status: RecoveryStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GuardianSignature {
    pub guardian_address: String,
    pub signature: String,
    pub signed_at: u64,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum RecoveryStatus {
    Pending,
    Confirming,
    Completed,
    Expired,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryConfig {
    pub min_guardians: u8,
    pub max_guardians: u8,
    pub default_threshold: u8,
    pub recovery_window_hours: u32,
    pub guardian_cooldown_seconds: u64,
    pub max_recovery_attempts: u32,
}

impl Default for RecoveryConfig {
    fn default() -> Self {
        Self {
            min_guardians: 3,
            max_guardians: 10,
            default_threshold: 3,
            recovery_window_hours: 24,
            guardian_cooldown_seconds: 300,
            max_recovery_attempts: 3,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoverySession {
    pub session_id: String,
    pub user_address: String,
    pub new_wallet_address: String,
    pub guardians_approved: Vec<String>,
    pub threshold: u8,
    pub started_at: u64,
    pub expires_at: u64,
    pub status: SessionStatus,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum SessionStatus {
    Initiated,
    AwaitingConfirmations,
    Completed,
    Expired,
    Cancelled,
}

// ============================================================================
// Social Recovery Service
// ============================================================================

pub struct SocialRecoveryService {
    // Storage
    guardian_sets: Arc<RwLock<HashMap<String, GuardianSet>>>,
    recovery_requests: Arc<RwLock<HashMap<String, RecoveryRequest>>>,
    sessions: Arc<RwLock<HashMap<String, RecoverySession>>>,
    
    // Configuration
    config: RecoveryConfig,
    
    // Metrics
    total_recoveries: Arc<RwLock<u64>>,
    successful_recoveries: Arc<RwLock<u64>>,
    failed_recoveries: Arc<RwLock<u64>>,
}

impl SocialRecoveryService {
    pub fn new(config: RecoveryConfig) -> Self {
        Self {
            guardian_sets: Arc::new(RwLock::new(HashMap::new())),
            recovery_requests: Arc::new(RwLock::new(HashMap::new())),
            sessions: Arc::new(RwLock::new(HashMap::new())),
            config,
            total_recoveries: Arc::new(RwLock::new(0)),
            successful_recoveries: Arc::new(RwLock::new(0)),
            failed_recoveries: Arc::new(RwLock::new(0)),
        }
    }

    /// Initialize guardian set for a user
    pub async fn initialize_guardian_set(
        &self,
        user_address: &str,
        guardians: Vec<Guardian>,
        threshold: u8,
    ) -> Result<GuardianSet, RecoveryError> {
        // Validate inputs
        if guardians.len() < self.config.min_guardians as usize {
            return Err(RecoveryError::InsufficientGuardians);
        }
        if guardians.len() > self.config.max_guardians as usize {
            return Err(RecoveryError::InvalidThreshold);
        }
        if threshold < 2 || threshold > guardians.len() as u8 {
            return Err(RecoveryError::InvalidThreshold);
        }

        // Validate guardian addresses
        for guardian in &guardians {
            if !self.is_valid_address(&guardian.address) {
                return Err(RecoveryError::InvalidGuardian);
            }
        }

        let now = Utc::now().timestamp() as u64;
        let guardian_set = GuardianSet {
            user_address: user_address.to_string(),
            guardians,
            threshold,
            locked: false,
            created_at: now,
            updated_at: now,
        };

        // Store guardian set
        let mut sets = self.guardian_sets.write().await;
        sets.insert(user_address.to_string(), guardian_set.clone());

        Ok(guardian_set)
    }

    /// Add a new guardian to the set
    pub async fn add_guardian(
        &self,
        user_address: &str,
        guardian: Guardian,
    ) -> Result<GuardianSet, RecoveryError> {
        let mut sets = self.guardian_sets.write().await;
        
        let guardian_set = sets
            .get_mut(user_address)
            .ok_or(RecoveryError::InvalidUser)?;

        if guardian_set.locked {
            return Err(RecoveryError::RecoveryInProgress);
        }

        if guardian_set.guardians.len() >= self.config.max_guardians as usize {
            return Err(RecoveryError::InsufficientGuardians);
        }

        // Check for duplicate
        for existing in &guardian_set.guardians {
            if existing.address == guardian.address {
                return Err(RecoveryError::InvalidGuardian);
            }
        }

        guardian_set.guardians.push(guardian);
        guardian_set.updated_at = Utc::now().timestamp() as u64;

        Ok(guardian_set.clone())
    }

    /// Remove a guardian from the set
    pub async fn remove_guardian(
        &self,
        user_address: &str,
        guardian_address: &str,
    ) -> Result<GuardianSet, RecoveryError> {
        let mut sets = self.guardian_sets.write().await;
        
        let guardian_set = sets
            .get_mut(user_address)
            .ok_or(RecoveryError::InvalidUser)?;

        if guardian_set.locked {
            return Err(RecoveryError::RecoveryInProgress);
        }

        let initial_len = guardian_set.guardians.len();
        guardian_set.guardians.retain(|g| g.address != guardian_address);

        if guardian_set.guardians.len() == initial_len {
            return Err(RecoveryError::GuardianNotFound);
        }

        // Ensure we still have minimum guardians
        if guardian_set.guardians.len() < self.config.min_guardians as usize {
            return Err(RecoveryError::InsufficientGuardians);
        }

        // Update threshold if needed
        if guardian_set.threshold > guardian_set.guardians.len() as u8 {
            guardian_set.threshold = guardian_set.guardians.len() as u8;
        }

        guardian_set.updated_at = Utc::now().timestamp() as u64;

        Ok(guardian_set.clone())
    }

    /// Update guardian set (add/remove multiple guardians)
    pub async fn update_guardian_set(
        &self,
        user_address: &str,
        add_guardians: Vec<Guardian>,
        remove_guardians: Vec<String>,
        new_threshold: Option<u8>,
    ) -> Result<GuardianSet, RecoveryError> {
        let mut sets = self.guardian_sets.write().await;
        
        let guardian_set = sets
            .get_mut(user_address)
            .ok_or(RecoveryError::InvalidUser)?;

        if guardian_set.locked {
            return Err(RecoveryError::RecoveryInProgress);
        }

        // Remove guardians first
        for addr in &remove_guardians {
            guardian_set.guardians.retain(|g| &g.address != addr);
        }

        // Add new guardians
        for guardian in add_guardians {
            // Check for duplicates
            let exists = guardian_set.guardians.iter().any(|g| g.address == guardian.address);
            if !exists && guardian_set.guardians.len() < self.config.max_guardians as usize {
                guardian_set.guardians.push(guardian);
            }
        }

        // Validate minimum
        if guardian_set.guardians.len() < self.config.min_guardians as usize {
            return Err(RecoveryError::InsufficientGuardians);
        }

        // Update threshold
        if let Some(threshold) = new_threshold {
            if threshold < 2 || threshold > guardian_set.guardians.len() as u8 {
                return Err(RecoveryError::InvalidThreshold);
            }
            guardian_set.threshold = threshold;
        }

        guardian_set.updated_at = Utc::now().timestamp() as u64;

        Ok(guardian_set.clone())
    }

    /// Initiate recovery process
    pub async fn initiate_recovery(
        &self,
        user_address: &str,
        new_wallet_address: &str,
    ) -> Result<RecoveryRequest, RecoveryError> {
        // Validate addresses
        if !self.is_valid_address(user_address) || !self.is_valid_address(new_wallet_address) {
            return Err(RecoveryError::InvalidAddress);
        }

        // Get guardian set
        let guardian_set = {
            let sets = self.guardian_sets.read().await;
            sets
                .get(user_address)
                .cloned()
                .ok_or(RecoveryError::InvalidUser)?
        };

        if guardian_set.locked {
            return Err(RecoveryError::RecoveryInProgress);
        }

        let now = Utc::now().timestamp() as u64;
        let expires_at = now + (self.config.recovery_window_hours as u64 * 3600);

        // Create recovery request
        let request = RecoveryRequest {
            id: Self::generate_id(),
            user_address: user_address.to_string(),
            new_address: new_wallet_address.to_string(),
            initiated_at: now,
            expires_at,
            completed_at: None,
            signatures: Vec::new(),
            status: RecoveryStatus::Pending,
        };

        // Store request
        {
            let mut requests = self.recovery_requests.write().await;
            requests.insert(request.id.clone(), request.clone());
        }

        // Lock guardian set
        {
            let mut sets = self.guardian_sets.write().await;
            if let Some(set) = sets.get_mut(user_address) {
                set.locked = true;
            }
        }

        // Create session
        let session = RecoverySession {
            session_id: request.id.clone(),
            user_address: user_address.to_string(),
            new_wallet_address: new_wallet_address.to_string(),
            guardians_approved: Vec::new(),
            threshold: guardian_set.threshold,
            started_at: now,
            expires_at,
            status: SessionStatus::Initiated,
        };

        {
            let mut sessions = self.sessions.write().await;
            sessions.insert(session.session_id.clone(), session);
        }

        // Update metrics
        {
            let mut total = self.total_recoveries.write().await;
            *total += 1;
        }

        Ok(request)
    }

    /// Submit guardian signature for recovery
    pub async fn submit_guardian_signature(
        &self,
        request_id: &str,
        guardian_address: &str,
        signature: &str,
    ) -> Result<RecoveryRequest, RecoveryError> {
        // Validate guardian
        if !self.is_valid_address(guardian_address) {
            return Err(RecoveryError::InvalidGuardian);
        }

        // Get request
        let (user_address, threshold, expires_at) = {
            let requests = self.recovery_requests.read().await;
            let request = requests
                .get(request_id)
                .ok_or(RecoveryError::InvalidSignature)?;

            if request.status == RecoveryStatus::Completed {
                return Err(RecoveryError::RecoveryAlreadyCompleted);
            }

            let now = Utc::now().timestamp() as u64;
            if now > request.expires_at {
                return Err(RecoveryError::RecoveryExpired);
            }

            (request.user_address.clone(), 0, request.expires_at)
        };

        // Verify guardian is in user's guardian set
        let is_valid_guardian = {
            let sets = self.guardian_sets.read().await;
            if let Some(set) = sets.get(&user_address) {
                set.guardians.iter().any(|g| g.address == guardian_address)
            } else {
                false
            }
        };

        if !is_valid_guardian {
            return Err(RecoveryError::InvalidGuardian);
        }

        // Verify signature
        let message = format!("{}:{}", request_id, user_address);
        if !self.verify_signature(guardian_address, &message, signature).await {
            return Err(RecoveryError::InvalidSignature);
        }

        // Update request
        let mut requests = self.recovery_requests.write().await;
        let request = requests
            .get_mut(request_id)
            .ok_or(RecoveryError::InvalidSignature)?;

        // Check for duplicate signature
        if request.signatures.iter().any(|s| s.guardian_address == guardian_address) {
            return Err(RecoveryError::InvalidSignature);
        }

        // Add signature
        request.signatures.push(GuardianSignature {
            guardian_address: guardian_address.to_string(),
            signature: signature.to_string(),
            signed_at: Utc::now().timestamp() as u64,
        });

        // Check if threshold met
        let guardian_set = {
            let sets = self.guardian_sets.read().await;
            sets.get(&user_address).cloned()
        };

        let threshold_met = match &guardian_set {
            Some(set) => request.signatures.len() >= set.threshold as usize,
            None => false,
        };

        if threshold_met {
            request.status = RecoveryStatus::Completed;
            request.completed_at = Some(Utc::now().timestamp() as u64);
        } else {
            request.status = RecoveryStatus::Confirming;
        }

        let result_request = request.clone();
        drop(requests);
        drop(guardian_set);

        if threshold_met {
            // Unlock guardian set
            {
                let mut sets = self.guardian_sets.write().await;
                if let Some(gs) = sets.get_mut(&user_address) {
                    gs.locked = false;
                }
            }

            // Update session
            {
                let mut sessions = self.sessions.write().await;
                if let Some(session) = sessions.get_mut(request_id) {
                    session.status = SessionStatus::Completed;
                }
            }

            // Update metrics
            {
                let mut successful = self.successful_recoveries.write().await;
                *successful += 1;
            }
        }

        Ok(result_request)
    }

    /// Cancel recovery process (only by original user)
    pub async fn cancel_recovery(
        &self,
        request_id: &str,
        user_signature: &str,
    ) -> Result<RecoveryRequest, RecoveryError> {
        let requests = self.recovery_requests.read().await;
        let request = requests
            .get(request_id)
            .ok_or(RecoveryError::InvalidSignature)?;

        if request.status == RecoveryStatus::Completed {
            return Err(RecoveryError::RecoveryAlreadyCompleted);
        }

        // Verify user signature
        let message = format!("cancel:{}", request_id);
        if !self.verify_signature(&request.user_address, &message, user_signature).await {
            return Err(RecoveryError::InvalidSignature);
        }

        drop(requests);

        // Update request
        let mut requests = self.recovery_requests.write().await;
        let request = requests
            .get_mut(request_id)
            .ok_or(RecoveryError::InvalidSignature)?;

        request.status = RecoveryStatus::Cancelled;

        // Unlock guardian set
        let user_address = request.user_address.clone();
        let result_request = request.clone();
        drop(requests);

        {
            let mut sets = self.guardian_sets.write().await;
            if let Some(set) = sets.get_mut(&user_address) {
                set.locked = false;
            }
        }

        // Update session
        {
            let mut sessions = self.sessions.write().await;
            if let Some(session) = sessions.get_mut(request_id) {
                session.status = SessionStatus::Cancelled;
            }
        }

        // Update metrics
        {
            let mut failed = self.failed_recoveries.write().await;
            *failed += 1;
        }

        Ok(result_request)
    }

    /// Get guardian set for user
    pub async fn get_guardian_set(&self, user_address: &str) -> Option<GuardianSet> {
        let sets = self.guardian_sets.read().await;
        sets.get(user_address).cloned()
    }

    /// Get recovery request status
    pub async fn get_recovery_request(&self, request_id: &str) -> Option<RecoveryRequest> {
        let requests = self.recovery_requests.read().await;
        requests.get(request_id).cloned()
    }

    /// Get all guardian sets (for admin)
    pub async fn get_all_guardian_sets(&self) -> Vec<GuardianSet> {
        let sets = self.guardian_sets.read().await;
        sets.values().cloned().collect()
    }

    /// Get recovery statistics
    pub async fn get_statistics(&self) -> RecoveryStatistics {
        let total = *self.total_recoveries.read().await;
        let successful = *self.successful_recoveries.read().await;
        let failed = *self.failed_recoveries.read().await;

        RecoveryStatistics {
            total_recoveries: total,
            successful_recoveries: successful,
            failed_recoveries: failed,
            success_rate: if total > 0 {
                (successful as f64 / total as f64) * 100.0
            } else {
                0.0
            },
        }
    }

    // Helper functions

    fn is_valid_address(&self, address: &str) -> bool {
        address.starts_with("0x") && address.len() == 42
    }

    fn generate_id() -> String {
        use std::time::{SystemTime, UNIX_EPOCH};
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        format!("{:x}", timestamp)
    }

    async fn verify_signature(&self, address: &str, message: &str, signature: &str) -> bool {
        // Look up the guardian's public key across all guardian sets.
        let public_key_hex = {
            let sets = self.guardian_sets.read().await;
            sets.values()
                .flat_map(|gs| gs.guardians.iter())
                .find(|g| g.address == address || g.public_key.as_deref() == Some(address))
                .and_then(|g| g.public_key.clone())
        };

        let pk_hex = match public_key_hex {
            Some(pk) => pk,
            None => return false,
        };

        let pk_bytes = match hex::decode(&pk_hex) {
            Ok(b) => b,
            Err(_) => return false,
        };
        let sig_bytes = match hex::decode(signature) {
            Ok(b) => b,
            Err(_) => return false,
        };

        let peer_pk = UnparsedPublicKey::new(&ED25519, pk_bytes);
        peer_pk
            .verify(message.as_bytes(), &sig_bytes)
            .is_ok()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryStatistics {
    pub total_recoveries: u64,
    pub successful_recoveries: u64,
    pub failed_recoveries: u64,
    pub success_rate: f64,
}

// ============================================================================
// Threshold Signature Scheme (TSS) - Advanced Recovery
// ============================================================================

/// Arithmetic in GF(2^8) with the AES reduction polynomial x^8+x^4+x^3+x+1
/// (0x11b), matching the field used elsewhere in the wallet stack.
///
/// Shamir's scheme requires a finite field. Doing the polynomial arithmetic on
/// plain bytes with wrapping arithmetic is not a field: the Lagrange
/// coefficients below are ratios like -x_j/(x_i-x_j) whose denominators are
/// not invertible modulo 256, so integer division both loses the value and
/// makes the result depend on which subset of shares was supplied.
mod gf256 {
    /// Reduction polynomial, low byte only: x^8+x^4+x^3+x+1 = 0x1b once the
    /// implicit x^8 term is dropped.
    const REDUCTION: u8 = 0x1b;

    /// Multiplicative inverse by exponentiation: a^254 = a^-1 in GF(2^8),
    /// since the multiplicative group has order 255.
    pub fn inv(a: u8) -> Option<u8> {
        if a == 0 {
            return None;
        }
        Some(pow(a, 254))
    }

    pub fn mul(mut a: u8, mut b: u8) -> u8 {
        let mut result: u8 = 0;
        while b != 0 {
            if b & 1 != 0 {
                result ^= a;
            }
            let high = a & 0x80;
            a <<= 1;
            if high != 0 {
                a ^= REDUCTION;
            }
            b >>= 1;
        }
        result
    }

    pub fn pow(mut base: u8, mut exp: u32) -> u8 {
        let mut result: u8 = 1;
        while exp > 0 {
            if exp & 1 == 1 {
                result = mul(result, base);
            }
            base = mul(base, base);
            exp >>= 1;
        }
        result
    }

    /// Evaluate the polynomial whose coefficients are `coeffs` (constant term
    /// first) at `x`.
    pub fn eval(coeffs: &[u8], x: u8) -> u8 {
        // Horner's method, evaluated from the highest degree down.
        let mut acc: u8 = 0;
        for &c in coeffs.iter().rev() {
            acc = mul(acc, x) ^ c;
        }
        acc
    }

    /// Lagrange interpolation at x=0, recovering the constant term.
    ///
    /// Returns None when the points are degenerate (a repeated x, or a pair
    /// whose difference has no inverse), which is what a caller must know
    /// rather than receiving a silently wrong secret.
    pub fn interpolate_at_zero(points: &[(u8, u8)]) -> Option<u8> {
        let mut secret: u8 = 0;
        for (i, &(xi, yi)) in points.iter().enumerate() {
            let mut basis: u8 = 1;
            for (j, &(xj, _)) in points.iter().enumerate() {
                if i == j {
                    continue;
                }
                if xi == xj {
                    return None;
                }
                // basis *= xj / (xj - xi), noting that subtraction is XOR.
                let denom = xj ^ xi;
                basis = mul(basis, mul(xj, inv(denom)?));
            }
            secret ^= mul(yi, basis);
        }
        Some(secret)
    }
}

pub struct ThresholdSignatureScheme {
    threshold: u8,
    total_shares: u8,
}

impl ThresholdSignatureScheme {
    pub fn new(threshold: u8, total_shares: u8) -> Self {
        Self { threshold, total_shares }
    }

    /// Generate key shares using Shamir's Secret Sharing
    /// Split `secret` into `num_shares` shares such that any `threshold` of
    /// them reconstruct it, and any `threshold - 1` reveal nothing about it.
    ///
    /// Each byte of the secret is shared independently as the constant term of
    /// a random polynomial of degree `threshold - 1` over GF(2^8), which is the
    /// standard Shamir construction. Share `i` is stored as a leading x
    /// coordinate followed by one y byte per secret byte, so a share is
    /// `secret.len() + 1` bytes.
    pub fn generate_shares(&self, secret: &[u8], num_shares: u8) -> Result<Vec<Vec<u8>>, RecoveryError> {
        if self.threshold == 0 || self.threshold > 255 {
            return Err(RecoveryError::InvalidThreshold);
        }
        if num_shares < self.threshold {
            return Err(RecoveryError::InvalidThreshold);
        }
        if secret.is_empty() {
            return Err(RecoveryError::InvalidThreshold);
        }

        let mut rng = ring::rand::SystemRandom::new();

        // One coefficient vector per secret byte, all sharing the same x
        // coordinates so a single interpolation recovers every byte.
        let mut coeffs: Vec<Vec<u8>> = Vec::with_capacity(secret.len());
        for &byte in secret {
            let mut c = vec![byte];
            for _ in 1..self.threshold {
                let mut r = [0u8; 1];
                ring::rand::SecureRandom::fill(&mut rng, &mut r)
                    .map_err(|_| RecoveryError::InvalidSignature)?;
                c.push(r[0]);
            }
            coeffs.push(c);
        }

        let mut shares = Vec::with_capacity(num_shares as usize);
        for i in 1..=num_shares {
            let mut share = Vec::with_capacity(secret.len() + 1);
            share.push(i);
            for c in &coeffs {
                share.push(gf256::eval(c, i));
            }
            shares.push(share);
        }

        Ok(shares)
    }

    /// Reconstruct the secret from at least `threshold` shares.
    ///
    /// Returns an error when too few shares are supplied, when a share is
    /// malformed, or when the x coordinates are not distinct.
    pub fn reconstruct_secret(&self, shares: &[Vec<u8>]) -> Result<Vec<u8>, RecoveryError> {
        if shares.len() < self.threshold as usize {
            return Err(RecoveryError::InsufficientGuardians);
        }

        // Every share must carry an x coordinate plus at least one y byte, and
        // they must all describe the same secret length.
        let secret_len = shares
            .first()
            .map(|s| s.len())
            .filter(|&n| n > 1)
            .ok_or(RecoveryError::InvalidGuardian)?
            - 1;
        if shares.iter().any(|s| s.len() != secret_len + 1) {
            return Err(RecoveryError::InvalidGuardian);
        }

        let points: Vec<(u8, Vec<u8>)> = shares
            .iter()
            .map(|s| (s[0], s[1..].to_vec()))
            .collect();

        let mut seen = std::collections::HashSet::new();
        for (x, _) in &points {
            if *x == 0 || !seen.insert(*x) {
                return Err(RecoveryError::InvalidGuardian);
            }
        }

        let mut secret = vec![0u8; secret_len];
        for (byte_index, out) in secret.iter_mut().enumerate() {
            let pairs: Vec<(u8, u8)> = points
                .iter()
                .map(|(x, ys)| (*x, ys[byte_index]))
                .collect();
            *out = gf256::interpolate_at_zero(&pairs).ok_or(RecoveryError::InvalidGuardian)?;
        }

        Ok(secret)
    }
}

// ============================================================================
// Social Graph Integration (for guardian discovery)
// ============================================================================

pub struct GuardianDiscovery {
    trusted_oracles: Vec<String>,
}

impl GuardianDiscovery {
    pub fn new() -> Self {
        Self {
            trusted_oracles: Vec::new(),
        }
    }

    /// Find potential guardians based on social graph
    pub async fn discover_guardians(&self, user_address: &str) -> Vec<PotentialGuardian> {
        // In production, this would query social graph oracles
        // For now, return empty list
        Vec::new()
    }

    /// Verify guardian identity through social proof
    pub async fn verify_guardian(&self, guardian_address: &str) -> GuardianVerification {
        GuardianVerification {
            address: guardian_address.to_string(),
            verified: false,
            verification_type: None,
            proof: None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PotentialGuardian {
    pub address: String,
    pub trust_score: f64,
    pub connection_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GuardianVerification {
    pub address: String,
    pub verified: bool,
    pub verification_type: Option<String>,
    pub proof: Option<String>,
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_guardian_set_creation() {
        let service = SocialRecoveryService::new(RecoveryConfig::default());
        
        let guardians = vec![
            Guardian {
                address: "0x1111111111111111111111111111111111111111".to_string(),
                name: "Guardian 1".to_string(),
                public_key: None,
                verified: true,
                added_at: 0,
                last_activity: 0,
            },
            Guardian {
                address: "0x2222222222222222222222222222222222222222".to_string(),
                name: "Guardian 2".to_string(),
                public_key: None,
                verified: true,
                added_at: 0,
                last_activity: 0,
            },
            Guardian {
                address: "0x3333333333333333333333333333333333333333".to_string(),
                name: "Guardian 3".to_string(),
                public_key: None,
                verified: true,
                added_at: 0,
                last_activity: 0,
            },
        ];

        let result = service
            .initialize_guardian_set("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E", guardians, 2)
            .await;

        assert!(result.is_ok());
    }

    #[test]
    fn test_threshold_scheme() {
        let tss = ThresholdSignatureScheme::new(2, 3);
        
        let secret = b"my secret key for wallet recovery";
        let shares = tss.generate_shares(secret, 3).unwrap();
        
        assert_eq!(shares.len(), 3);
        
        // Reconstruct with 2 shares. The recovered value must be the secret
        // itself, not merely the right length.
        let recovered = tss.reconstruct_secret(&shares[..2]).unwrap();
        assert_eq!(recovered, secret);
    }
}

#[cfg(test)]
mod gf256_tests {
    use super::gf256;

    // FIPS-197 worked example: 0x57 * 0x83 = 0xc1 in GF(2^8) mod 0x11b.
    #[test]
    fn multiplication_matches_fips_197() {
        assert_eq!(gf256::mul(0x57, 0x83), 0xc1);
        assert_eq!(gf256::mul(0x57, 0x13), 0xfe);
        assert_eq!(gf256::mul(0x00, 0x83), 0x00);
        assert_eq!(gf256::mul(0x01, 0x83), 0x83);
        // Commutative.
        assert_eq!(gf256::mul(0x83, 0x57), 0xc1);
    }

    // The AES S-box is built from the multiplicative inverse, so a known
    // inverse pins the field down. inv(0x53) = 0xca.
    #[test]
    fn inverse_matches_aes_field() {
        assert_eq!(gf256::inv(0x53), Some(0xca));
        assert_eq!(gf256::mul(0x53, 0xca), 0x01);
        assert_eq!(gf256::inv(0x00), None);
        assert_eq!(gf256::inv(0x01), Some(0x01));
        for a in 1u8..=255 {
            let i = gf256::inv(a).expect("nonzero has an inverse");
            assert_eq!(gf256::mul(a, i), 0x01, "a * a^-1 must be 1 for {a}");
        }
    }

    #[test]
    fn eval_is_the_polynomial_value() {
        // p(x) = 5 + 3x + 7x^2 evaluated at x = 2.
        let coeffs = [0x05u8, 0x03, 0x07];
        let expected = 0x05 ^ gf256::mul(0x03, 0x02) ^ gf256::mul(0x07, gf256::mul(0x02, 0x02));
        assert_eq!(gf256::eval(&coeffs, 0x02), expected);
        // At x = 0 the value is the constant term.
        assert_eq!(gf256::eval(&coeffs, 0x00), 0x05);
    }

    #[test]
    fn interpolation_recovers_the_constant_term() {
        // p(x) = 0x42 + 0x11 x + 0x99 x^2 over GF(2^8).
        let coeffs = [0x42u8, 0x11, 0x99];
        let points: Vec<(u8, u8)> = (1u8..=3).map(|x| (x, gf256::eval(&coeffs, x))).collect();
        assert_eq!(gf256::interpolate_at_zero(&points), Some(0x42));
    }

    #[test]
    fn interpolation_rejects_degenerate_points() {
        assert_eq!(gf256::interpolate_at_zero(&[(1, 2), (1, 3)]), None);
    }
}
