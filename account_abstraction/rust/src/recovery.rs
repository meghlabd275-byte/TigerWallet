//! Social Recovery module for Account Abstraction

use crate::{AAError, RecoveryRequest};
use chrono::{DateTime, Utc, Duration};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Guardian information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GuardianInfo {
    pub address: String,
    pub delay: u64,
    pub weight: u32,
    pub active: bool,
}

/// Recovery request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryRequestInternal {
    pub id: String,
    pub account: String,
    pub new_owner: String,
    pub guardians: Vec<String>,
    pub confirmations: Vec<String>,
    pub threshold: u32,
    pub started_at: DateTime<Utc>,
    pub unlock_time: DateTime<Utc>,
    pub completed: bool,
    pub cancelled: bool,
}

/// Anti-scam configuration
#[derive(Debug, Clone)]
pub struct AntiScamConfig {
    /// Enable phishing detection
    pub enable_phishing_detection: bool,
    /// Enable domain verification
    pub enable_domain_verification: bool,
    /// Enable transaction simulation
    pub enable_transaction_simulation: bool,
    /// Known malicious domains
    pub malicious_domains: Vec<String>,
    /// Minimum guardian delay
    pub min_guardian_delay: u64,
}

impl Default for AntiScamConfig {
    fn default() -> Self {
        Self {
            enable_phishing_detection: true,
            enable_domain_verification: true,
            enable_transaction_simulation: true,
            malicious_domains: vec![],
            min_guardian_delay: 24 * 3600, // 24 hours
        }
    }
}

/// Recovery service
pub struct RecoveryService {
    /// Recovery requests
    requests: HashMap<String, RecoveryRequestInternal>,
    /// Guardian configurations
    guardians: HashMap<String, Vec<GuardianInfo>>,
    /// Recovery thresholds
    thresholds: HashMap<String, u32>,
    /// Recovery delays
    delays: HashMap<String, u64>,
    /// Anti-scam configuration
    anti_scam: AntiScamConfig,
    /// Recovery history
    history: Vec<RecoveryHistory>,
}

/// Recovery history
#[derive(Debug, Clone)]
pub struct RecoveryHistory {
    pub account: String,
    pub old_owner: String,
    pub new_owner: String,
    pub timestamp: DateTime<Utc>,
    pub guardians: Vec<String>,
    pub method: RecoveryMethod,
}

/// Recovery method
#[derive(Debug, Clone)]
pub enum RecoveryMethod {
    Social,
    TimeLock,
    Emergency,
    Guardian,
}

impl RecoveryService {
    pub fn new() -> Self {
        Self {
            requests: HashMap::new(),
            guardians: HashMap::new(),
            thresholds: HashMap::new(),
            delays: HashMap::new(),
            anti_scam: AntiScamConfig::default(),
            history: Vec::new(),
        }
    }

    /// Configure guardians for account
    pub fn configure_guardians(
        &mut self,
        account: &str,
        guardian_list: Vec<GuardianInfo>,
        threshold: u32,
        delay: u64,
    ) -> Result<(), AAError> {
        if guardian_list.is_empty() {
            return Err(AAError::RecoveryError("No guardians".to_string()));
        }
        
        if threshold == 0 || threshold > guardian_list.len() as u32 {
            return Err(AAError::RecoveryError("Invalid threshold".to_string()));
        }
        
        if delay < self.anti_scam.min_guardian_delay {
            return Err(AAError::RecoveryError("Delay too short".to_string()));
        }
        
        self.guardians.insert(account.to_string(), guardian_list);
        self.thresholds.insert(account.to_string(), threshold);
        self.delays.insert(account.to_string(), delay);
        
        Ok(())
    }

    /// Start recovery
    pub async fn start_recovery(
        &self,
        account: &str,
        new_owner: &str,
        guardian: &str,
        chain_id: u64,
    ) -> Result<RecoveryRequest, AAError> {
        // Validate guardian
        let guardians = self.guardians.get(account)
            .ok_or(AAError::RecoveryError("No guardians configured".to_string()))?;
        
        let guardian_info = guardians.iter()
            .find(|g| g.address == guardian && g.active)
            .ok_or(AAError::RecoveryError("Invalid guardian".to_string()))?;
        
        let threshold = self.thresholds.get(account)
            .ok_or(AAError::RecoveryError("No threshold".to_string()))?;
        
        let delay = self.delays.get(account)
            .ok_or(AAError::RecoveryError("No delay".to_string()))?;
        
        // Check anti-scam
        if self.anti_scam.enable_phishing_detection {
            self.check_phishing(account, new_owner)?;
        }
        
        // Create recovery request
        let request_id = generate_request_id(account, new_owner, chain_id);
        let now = Utc::now();
        
        let request = RecoveryRequestInternal {
            id: request_id.clone(),
            account: account.to_string(),
            new_owner: new_owner.to_string(),
            guardians: vec![guardian.to_string()],
            confirmations: vec![guardian.to_string()],
            threshold: *threshold,
            started_at: now,
            unlock_time: now + Duration::seconds(*delay as i64),
            completed: false,
            cancelled: false,
        };
        
        Ok(RecoveryRequest {
            id: request_id,
            account: account.to_string(),
            new_owner: new_owner.to_string(),
            guardians: vec![guardian.to_string()],
            confirmations: *threshold,
            threshold: *threshold,
            started_at: now,
            unlock_time: now + Duration::seconds(*delay as i64),
            completed: false,
        })
    }

    /// Confirm recovery
    pub async fn confirm_recovery(
        &self,
        account: &str,
        new_owner: &str,
        guardian: &str,
    ) -> Result<bool, AAError> {
        // Find existing request
        let request = self.requests.get(account)
            .ok_or(AAError::RecoveryError("No recovery request".to_string()))?;
        
        if request.new_owner != new_owner {
            return Err(AAError::RecoveryError("Different new owner".to_string()));
        }
        
        // Check guardian
        let guardians = self.guardians.get(account)
            .ok_or(AAError::RecoveryError("No guardians".to_string()))?;
        
        if !guardians.iter().any(|g| g.address == guardian) {
            return Err(AAError::RecoveryError("Not guardian".to_string()));
        }
        
        // Add confirmation
        let threshold = self.thresholds.get(account)
            .ok_or(AAError::RecoveryError("No threshold".to_string()))?;
        
        let confirmations = request.confirmations.len() as u32 + 1;
        
        Ok(confirmations >= *threshold)
    }

    /// Complete recovery
    pub async fn complete_recovery(
        &self,
        account: &str,
    ) -> Result<String, AAError> {
        let request = self.requests.get(account)
            .ok_or(AAError::RecoveryError("No request".to_string()))?;
        
        // Check time lock
        if Utc::now() < request.unlock_time {
            return Err(AAError::RecoveryError("Time lock active".to_string()));
        }
        
        // Record history
        let history = RecoveryHistory {
            account: account.to_string(),
            old_owner: "".to_string(),
            new_owner: request.new_owner.clone(),
            timestamp: Utc::now(),
            guardians: request.guardians.clone(),
            method: RecoveryMethod::Social,
        };
        
        Ok(request.new_owner.clone())
    }

    /// Cancel recovery
    pub async fn cancel_recovery(&self, account: &str) -> Result<(), AAError> {
        if let Some(request) = self.requests.get_mut(account) {
            request.cancelled = true;
        }
        Ok(())
    }

    /// Check phishing
    fn check_phishing(&self, _account: &str, new_owner: &str) -> Result<(), AAError> {
        // Check if new owner is a known malicious address
        if self.anti_scam.malicious_domains.contains(&new_owner.to_string()) {
            return Err(AAError::RecoveryError("Suspicious address".to_string()));
        }
        
        // In production, would check:
        // - Transaction simulation
        // - Domain verification
        // - Phishing database
        
        Ok(())
    }

    /// Get pending recoveries
    pub fn get_pending_recoveries(&self, account: &str) -> Vec<RecoveryRequest> {
        self.requests.values()
            .filter(|r| r.account == account && !r.completed && !r.cancelled)
            .map(|r| RecoveryRequest {
                id: r.id.clone(),
                account: r.account.clone(),
                new_owner: r.new_owner.clone(),
                guardians: r.guardians.clone(),
                confirmations: r.confirmations.len() as u32,
                threshold: r.threshold,
                started_at: r.started_at,
                unlock_time: r.unlock_time,
                completed: r.completed,
            })
            .collect()
    }

    /// Get recovery history
    pub fn get_history(&self, account: &str) -> Vec<RecoveryHistory> {
        self.history.iter()
            .filter(|h| h.account == account)
            .cloned()
            .collect()
    }
}

impl Default for RecoveryService {
    fn default() -> Self {
        Self::new()
    }
}

/// Generate request ID
fn generate_request_id(account: &str, new_owner: &str, chain_id: u64) -> String {
    use ring::digest::digest;
    
    let data = format!("{}:{}:{}", account, new_owner, chain_id);
    let hash = digest(&ring::digest::SHA256, data.as_bytes());
    
    hex::encode(hash.as_ref())
}