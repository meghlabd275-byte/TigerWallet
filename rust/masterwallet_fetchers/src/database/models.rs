//! Models module for database models
//! Provides type-safe model definitions

use serde::{Deserialize, Serialize};

/// Master wallet model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MasterWalletModel {
    pub id: String,
    pub name: String,
    pub wallet_type: String,
    pub address: String,
    pub public_key: Option<String>,
    pub chain_id: i64,
    pub encrypted_private_key: Option<String>,
    pub is_active: bool,
    pub settings: Option<serde_json::Value>,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Sub wallet model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubWalletModel {
    pub id: String,
    pub master_wallet_id: String,
    pub name: String,
    pub address: String,
    pub address_type: String,
    pub public_key: Option<String>,
    pub encrypted_private_key: Option<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

/// User model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserModel {
    pub id: String,
    pub wallet_id: String,
    pub user_id: String,
    pub email: String,
    pub name: Option<String>,
    pub role: String,
    pub permissions: Vec<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Auto sign rule model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoSignRuleModel {
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

/// Transaction approval model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionApprovalModel {
    pub id: String,
    pub master_wallet_id: String,
    pub sub_wallet_id: String,
    pub tx_hash: String,
    pub from_address: String,
    pub to_address: String,
    pub amount: String,
    pub token_id: Option<String>,
    pub chain_id: i64,
    pub status: String,
    pub approved_by: Option<String>,
    pub rejected_by: Option<String>,
    pub reject_reason: Option<String>,
    pub gas_used: Option<String>,
    pub block_number: Option<i64>,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Role permission model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RolePermissionModel {
    pub id: i64,
    pub master_wallet_id: Option<String>,
    pub role_name: String,
    pub permissions: Vec<String>,
    pub description: Option<String>,
    pub created_at: i64,
}

/// Whitelist model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhitelistModel {
    pub id: String,
    pub master_wallet_id: String,
    pub address: String,
    pub address_type: String,
    pub label: Option<String>,
    pub is_verified: bool,
    pub added_by: String,
    pub created_at: i64,
}

/// Wallet token model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletTokenModel {
    pub id: String,
    pub wallet_id: String,
    pub token_id: String,
    pub chain_id: i64,
    pub balance: String,
    pub reserved_balance: String,
    pub updated_at: i64,
}

/// Fee config model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeConfigModel {
    pub id: String,
    pub master_wallet_id: String,
    pub fee_type: String,
    pub percentage: f64,
    pub flat_fee: String,
    pub min_amount: Option<String>,
    pub max_amount: Option<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Audit log model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLogModel {
    pub id: String,
    pub master_wallet_id: Option<String>,
    pub user_id: Option<String>,
    pub action: String,
    pub entity_type: Option<String>,
    pub entity_id: Option<String>,
    pub details: Option<serde_json::Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: i64,
}
