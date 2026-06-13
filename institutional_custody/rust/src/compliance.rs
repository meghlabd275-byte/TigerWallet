//! Compliance module for Institutional Custody

use crate::{CustodyError, ComplianceReport};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use chrono::{DateTime, Utc};

/// Transaction record
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionRecord {
    pub id: String,
    pub wallet: String,
    pub transaction_type: String,
    pub amount: u64,
    pub counterparty: Option<String>,
    pub status: String,
    pub timestamp: DateTime<Utc>,
    pub risk_score: u32,
    pub flags: Vec<String>,
}

/// Audit entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditEntry {
    pub id: String,
    pub wallet: String,
    pub action: String,
    pub actor: String,
    pub details: String,
    pub timestamp: DateTime<Utc>,
    pub ip_address: Option<String>,
}

/// Compliance rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceRule {
    pub id: String,
    pub name: String,
    pub rule_type: String,
    pub conditions: String,
    pub action: String,
    pub enabled: bool,
}

/// Restricted asset
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RestrictedAsset {
    pub asset: String,
    pub reason: String,
    pub restricted_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
}

/// Compliance service
pub struct ComplianceService {
    /// Transactions
    transactions: HashMap<String, Vec<TransactionRecord>>,
    /// Audit log
    audit_log: HashMap<String, Vec<AuditEntry>>,
    /// Compliance rules
    rules: Vec<ComplianceRule>,
    /// Restricted assets
    restricted_assets: HashMap<String, RestrictedAsset>,
    /// Wallet stats
    stats: HashMap<String, WalletStats>,
}

/// Wallet stats
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct WalletStats {
    pub transaction_count: u64,
    pub total_volume: u64,
    pub daily_limit: u64,
    pub daily_spent: u64,
    pub daily_reset: DateTime<Utc>,
    pub restricted_assets: Vec<String>,
}

impl ComplianceService {
    pub fn new() -> Self {
        Self {
            transactions: HashMap::new(),
            audit_log: HashMap::new(),
            rules: vec![],
            restricted_assets: HashMap::new(),
            stats: HashMap::new(),
        }
    }

    /// Initialize wallet
    pub async fn initialize_wallet(&self, wallet: &str) -> Result<(), CustodyError> {
        self.stats.insert(wallet.to_string(), WalletStats::default());
        Ok(())
    }

    /// Record transaction
    pub async fn record_transaction(
        &self,
        wallet: &str,
        tx_type: &str,
        amount: u64,
        counterparty: Option<&str>,
    ) -> Result<String, CustodyError> {
        let tx_id = uuid::Uuid::new_v4().to_string();
        
        let record = TransactionRecord {
            id: tx_id.clone(),
            wallet: wallet.to_string(),
            transaction_type: tx_type.to_string(),
            amount,
            counterparty: counterparty.map(|s| s.to_string()),
            status: "completed".to_string(),
            timestamp: Utc::now(),
            risk_score: self.calculate_risk_score(wallet, amount),
            flags: vec![],
        };
        
        let txs = self.transactions
            .entry(wallet.to_string())
            .or_insert_with(Vec::new);
        
        txs.push(record);
        
        // Update stats
        if let Some(stats) = self.stats.get_mut(wallet) {
            stats.transaction_count += 1;
            stats.total_volume += amount;
            stats.daily_spent += amount;
        }
        
        Ok(tx_id)
    }

    /// Calculate risk score
    fn calculate_risk_score(&self, wallet: &str, amount: u64) -> u32 {
        let mut score = 0;
        
        // Amount-based risk
        if amount > 10000 {
            score += 20;
        }
        
        // Would check other risk factors
        score
    }

    /// Record audit entry
    pub async fn record_audit(
        &self,
        wallet: &str,
        action: &str,
        actor: &str,
        details: &str,
    ) -> Result<String, CustodyError> {
        let entry_id = uuid::Uuid::new_v4().to_string();
        
        let entry = AuditEntry {
            id: entry_id.clone(),
            wallet: wallet.to_string(),
            action: action.to_string(),
            actor: actor.to_string(),
            details: details.to_string(),
            timestamp: Utc::now(),
            ip_address: None,
        };
        
        let log = self.audit_log
            .entry(wallet.to_string())
            .or_insert_with(Vec::new);
        
        log.push(entry);
        
        Ok(entry_id)
    }

    /// Add compliance rule
    pub async fn add_rule(&mut self, rule: ComplianceRule) -> Result<(), CustodyError> {
        self.rules.push(rule);
        Ok(())
    }

    /// Enable rule
    pub async fn enable_rule(&mut self, rule_id: &str, enabled: bool) -> Result<(), CustodyError> {
        for rule in &mut self.rules {
            if rule.id == rule_id {
                rule.enabled = enabled;
                return Ok(());
            }
        }
        Err(CustodyError::ComplianceError("Rule not found".to_string()))
    }

    /// Add restricted asset
    pub async fn add_restricted_asset(
        &mut self,
        asset: &str,
        reason: &str,
    ) -> Result<(), CustodyError> {
        self.restricted_assets.insert(asset.to_string(), RestrictedAsset {
            asset: asset.to_string(),
            reason: reason.to_string(),
            restricted_at: Utc::now(),
            expires_at: None,
        });
        Ok(())
    }

    /// Check asset restricted
    pub async fn is_asset_restricted(&self, asset: &str) -> bool {
        self.restricted_assets.contains_key(asset)
    }

    /// Generate compliance report
    pub async fn generate_report(&self, wallet: &str) -> Result<ComplianceReport, CustodyError> {
        let stats = self.stats.get(wallet)
            .ok_or(CustodyError::ComplianceError("Wallet not found".to_string()))?;
        
        let report_hash = format!(
            "{:x}:{}:{}",
            uuid::Uuid::new_v4(),
            stats.transaction_count,
            stats.total_volume
        );
        
        Ok(ComplianceReport {
            wallet: wallet.to_string(),
            transaction_count: stats.transaction_count,
            total_volume: stats.total_volume,
            daily_limit: stats.daily_limit,
            restricted_assets: stats.restricted_assets.clone(),
            generated_at: Utc::now(),
            report_hash,
        })
    }

    /// Get transaction history
    pub async fn get_transactions(
        &self,
        wallet: &str,
        limit: usize,
    ) -> Result<Vec<TransactionRecord>, CustodyError> {
        let txs = self.transactions.get(wallet)
            .ok_or(CustodyError::ComplianceError("Wallet not found".to_string()))?;
        
        let start = if txs.len() > limit {
            txs.len() - limit
        } else {
            0
        };
        
        Ok(txs[start..].to_vec())
    }

    /// Get audit log
    pub async fn get_audit_log(
        &self,
        wallet: &str,
        limit: usize,
    ) -> Result<Vec<AuditEntry>, CustodyError> {
        let log = self.audit_log.get(wallet)
            .ok_or(CustodyError::ComplianceError("Wallet not found".to_string()))?;
        
        let start = if log.len() > limit {
            log.len() - limit
        } else {
            0
        };
        
        Ok(log[start..].to_vec())
    }
}

impl Default for ComplianceService {
    fn default() -> Self {
        Self::new()
    }
}