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
    /// Check a proposed new owner against the address denylist before a
    /// recovery may start.
    pub enable_address_denylist: bool,
    /// Addresses that may never become the owner of an account, because they
    /// are known to be controlled by an attacker. Compared case-insensitively.
    pub blocked_addresses: Vec<String>,
    /// Minimum guardian delay, in seconds.
    pub min_guardian_delay: u64,
}

impl Default for AntiScamConfig {
    fn default() -> Self {
        Self {
            enable_address_denylist: true,
            blocked_addresses: vec![],
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

    /// Set the minimum delay a recovery must wait before it can complete.
    ///
    /// The delay is what gives an owner time to notice and cancel a recovery
    /// they did not request, so lowering it weakens that protection.
    pub fn set_min_guardian_delay(&mut self, seconds: u64) {
        self.anti_scam.min_guardian_delay = seconds;
    }

    /// The configured minimum guardian delay, in seconds.
    pub fn min_guardian_delay(&self) -> u64 {
        self.anti_scam.min_guardian_delay
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

    /// Start a recovery request.
    ///
    /// The request is recorded so later confirmations accumulate against it,
    /// and the guardian starting it counts as the first confirmation.
    pub async fn start_recovery(
        &mut self,
        account: &str,
        new_owner: &str,
        guardian: &str,
        chain_id: u64,
    ) -> Result<RecoveryRequest, AAError> {
        if new_owner.is_empty() {
            return Err(AAError::RecoveryError("A new owner is required".to_string()));
        }

        // A request already in flight must be completed or cancelled first,
        // otherwise a second start would silently discard the first.
        if let Some(existing) = self.requests.get(account) {
            if !existing.completed && !existing.cancelled {
                return Err(AAError::RecoveryError(
                    "A recovery request is already in progress for this account".to_string(),
                ));
            }
        }

        let guardians = self
            .guardians
            .get(account)
            .ok_or_else(|| AAError::RecoveryError("No guardians configured".to_string()))?;

        if !guardians.iter().any(|g| g.address == guardian && g.active) {
            return Err(AAError::RecoveryError(
                "The account holder is not an active guardian".to_string(),
            ));
        }

        let threshold = *self
            .thresholds
            .get(account)
            .ok_or_else(|| AAError::RecoveryError("No threshold".to_string()))?;

        let delay = *self
            .delays
            .get(account)
            .ok_or_else(|| AAError::RecoveryError("No delay".to_string()))?;

        if self.anti_scam.enable_address_denylist {
            self.check_denylist(account, new_owner)?;
        }

        let request_id = generate_request_id(account, new_owner, chain_id);
        let now = Utc::now();
        let unlock_time = now + Duration::seconds(delay as i64);

        // The guardian who starts the recovery counts as its first confirmation.
        let request = RecoveryRequestInternal {
            id: request_id.clone(),
            account: account.to_string(),
            new_owner: new_owner.to_string(),
            guardians: vec![guardian.to_string()],
            confirmations: vec![guardian.to_string()],
            threshold,
            started_at: now,
            unlock_time,
            completed: false,
            cancelled: false,
        };

        self.requests.insert(account.to_string(), request);

        Ok(RecoveryRequest {
            id: request_id,
            account: account.to_string(),
            new_owner: new_owner.to_string(),
            guardians: vec![guardian.to_string()],
            confirmations: 1,
            threshold,
            started_at: now,
            unlock_time,
            completed: false,
        })
    }

    /// Record a guardian's confirmation.
    ///
    /// Returns true once the request has reached its threshold. Each guardian
    /// counts once, and only guardians configured for the account are accepted.
    pub async fn confirm_recovery(
        &mut self,
        account: &str,
        new_owner: &str,
        guardian: &str,
    ) -> Result<bool, AAError> {
        let guardians = self
            .guardians
            .get(account)
            .ok_or_else(|| AAError::RecoveryError("No guardians".to_string()))?;

        if !guardians.iter().any(|g| g.address == guardian && g.active) {
            return Err(AAError::RecoveryError(
                "The account holder is not an active guardian".to_string(),
            ));
        }

        let request = self
            .requests
            .get_mut(account)
            .ok_or_else(|| AAError::RecoveryError("No recovery request".to_string()))?;

        if request.cancelled {
            return Err(AAError::RecoveryError(
                "The recovery request was cancelled".to_string(),
            ));
        }
        if request.completed {
            return Err(AAError::RecoveryError(
                "The recovery request is already complete".to_string(),
            ));
        }
        if request.new_owner != new_owner {
            return Err(AAError::RecoveryError(
                "This confirmation names a different new owner".to_string(),
            ));
        }
        if request.confirmations.iter().any(|g| g == guardian) {
            return Err(AAError::RecoveryError(
                "This guardian has already confirmed".to_string(),
            ));
        }

        request.confirmations.push(guardian.to_string());

        Ok(request.confirmations.len() as u32 >= request.threshold)
    }

    /// Complete a recovery once the threshold is met and the time lock expired.
    ///
    /// Returns the new owner, and only then records the change in history. A
    /// request that has not reached its threshold cannot complete, so a single
    /// guardian cannot seize an account.
    pub async fn complete_recovery(&mut self, account: &str) -> Result<String, AAError> {
        let request = self
            .requests
            .get(account)
            .ok_or_else(|| AAError::RecoveryError("No request".to_string()))?;

        if request.cancelled {
            return Err(AAError::RecoveryError(
                "The recovery request was cancelled".to_string(),
            ));
        }
        if request.completed {
            return Err(AAError::RecoveryError(
                "The recovery request is already complete".to_string(),
            ));
        }
        if (request.confirmations.len() as u32) < request.threshold {
            return Err(AAError::RecoveryError(format!(
                "Recovery has {} of {} required confirmations",
                request.confirmations.len(),
                request.threshold
            )));
        }
        if Utc::now() < request.unlock_time {
            return Err(AAError::RecoveryError(
                "The recovery time lock has not expired".to_string(),
            ));
        }

        let new_owner = request.new_owner.clone();
        let guardians = request.guardians.clone();

        let request = self
            .requests
            .get_mut(account)
            .expect("request was present a moment ago");
        request.completed = true;

        self.history.push(RecoveryHistory {
            account: account.to_string(),
            old_owner: String::new(),
            new_owner: new_owner.clone(),
            timestamp: Utc::now(),
            guardians,
            method: RecoveryMethod::Social,
        });

        Ok(new_owner)
    }

    /// Cancel recovery
    pub async fn cancel_recovery(&mut self, account: &str) -> Result<(), AAError> {
        let request = self
            .requests
            .get_mut(account)
            .ok_or_else(|| AAError::RecoveryError("No recovery request".to_string()))?;

        if request.completed {
            return Err(AAError::RecoveryError(
                "A completed recovery cannot be cancelled".to_string(),
            ));
        }

        request.cancelled = true;
        Ok(())
    }

    /// Reject a recovery whose new owner is on the address denylist.
    ///
    /// Addresses are compared case-insensitively, because checksum casing is a
    /// presentation detail rather than part of the address.
    fn check_denylist(&self, account: &str, new_owner: &str) -> Result<(), AAError> {
        if self
            .anti_scam
            .blocked_addresses
            .iter()
            .any(|blocked| blocked.eq_ignore_ascii_case(new_owner))
        {
            return Err(AAError::RecoveryError(format!(
                "The proposed new owner {new_owner} is on the denylist"
            )));
        }

        if self
            .anti_scam
            .blocked_addresses
            .iter()
            .any(|blocked| blocked.eq_ignore_ascii_case(account))
        {
            return Err(AAError::RecoveryError(format!(
                "The account {account} is on the denylist"
            )));
        }

        Ok(())
    }

    /// Add an address to the recovery denylist.
    ///
    /// The list is normalised to lowercase so lookups do not depend on how the
    /// caller spelled the address.
    pub fn block_address(&mut self, address: &str) {
        let normalized = address.to_lowercase();
        if !self.anti_scam.blocked_addresses.contains(&normalized) {
            self.anti_scam.blocked_addresses.push(normalized);
        }
    }

    /// Remove an address from the recovery denylist.
    pub fn unblock_address(&mut self, address: &str) {
        let normalized = address.to_lowercase();
        self.anti_scam
            .blocked_addresses
            .retain(|blocked| *blocked != normalized);
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