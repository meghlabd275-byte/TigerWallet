//! Types module for MasterWallet fetchers
//! Defines all data structures used across the system

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// MasterWallet - The main wallet that controls all user wallets
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MasterWallet {
    pub id: String,
    pub name: String,
    pub wallet_type: WalletType,
    pub address: String,
    pub public_key: String,
    pub chain_id: i64,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Wallet type enum
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum WalletType {
    Hot,
    Cold,
    Operations,
    Treasury,
}

/// SubWallet - User wallet under master wallet
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubWallet {
    pub id: String,
    pub master_wallet_id: String,
    pub name: String,
    pub address: String,
    pub address_type: AddressType,
    pub public_key: Option<String>,
    pub encrypted_private_key: Option<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Address type enum
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AddressType {
    EVM,
    Solana,
    Bitcoin,
    Cosmos,
    StarkNet,
    Aptos,
    Sui,
}

/// Wallet user - User associated with a sub-wallet
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletUser {
    pub id: String,
    pub wallet_id: String,
    pub user_id: String,
    pub email: String,
    pub name: Option<String>,
    pub role: UserRole,
    pub permissions: Vec<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

/// User role enum
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum UserRole {
    Admin,
    Manager,
    User,
    Viewer,
}

/// Auto-sign rule for automatic transaction signing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoSignRule {
    pub id: String,
    pub master_wallet_id: String,
    pub name: String,
    pub max_amount: String,
    pub chain_ids: Vec<String>,
    pub token_ids: Vec<String>,
    pub enabled: bool,
    pub conditions: Vec<String>,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Transaction approval request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionApproval {
    pub id: String,
    pub master_wallet_id: String,
    pub sub_wallet_id: String,
    pub tx_hash: String,
    pub from_address: String,
    pub to_address: String,
    pub amount: String,
    pub token_id: Option<String>,
    pub chain_id: i64,
    pub status: TransactionStatus,
    pub approved_by: Option<String>,
    pub rejected_by: Option<String>,
    pub reject_reason: Option<String>,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Transaction status enum
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransactionStatus {
    Pending,
    Approved,
    Rejected,
    Broadcasted,
    Confirmed,
    Failed,
}

/// Volume statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VolumeStats {
    pub total_volume: String,
    pub daily_volume: String,
    pub monthly_volume: String,
    pub daily_breakdown: Vec<DailyVolume>,
    pub transaction_count: i64,
    pub period: String,
    pub timestamp: i64,
}

/// Daily volume data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DailyVolume {
    pub date: String,
    pub volume: f64,
}

/// Analytics data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Analytics {
    pub wallet_count: i64,
    pub user_count: i64,
    pub active_wallets: i64,
    pub new_wallets_this_week: i64,
    pub volume: VolumeData,
    pub transactions: TransactionData,
    pub growth: String,
    pub timestamp: i64,
}

/// Volume data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VolumeData {
    pub daily: String,
    pub weekly: String,
    pub total: String,
}

/// Transaction data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionData {
    pub total: i64,
}

/// Permission data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Permission {
    pub id: String,
    pub user_id: String,
    pub email: String,
    pub role: String,
    pub permissions: Vec<String>,
}

/// Role definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoleDefinition {
    pub role_name: String,
    pub permissions: Vec<String>,
    pub description: Option<String>,
}

/// Whitelist entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhitelistEntry {
    pub id: String,
    pub address: String,
    pub address_type: String,
    pub label: Option<String>,
    pub is_verified: bool,
    pub added_by: String,
    pub created_at: i64,
}

/// Fetcher result wrapper
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherResult {
    pub success: bool,
    pub data: Option<serde_json::Value>,
    pub error: Option<String>,
    pub timestamp: i64,
}

impl FetcherResult {
    pub fn success(data: serde_json::Value) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
            timestamp: chrono::Utc::now().timestamp(),
        }
    }
    
    pub fn error(message: &str) -> Self {
        Self {
            success: false,
            data: None,
            error: Some(message.to_string()),
            timestamp: chrono::Utc::now().timestamp(),
        }
    }
}
