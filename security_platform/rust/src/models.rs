//! Data models for Security Platform

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Audit Log
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    pub id: String,
    pub user_id: String,
    pub action: String,
    pub resource: String,
    pub result: AuditResult,
    pub timestamp: DateTime<Utc>,
}

/// Audit Result
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum AuditResult {
    Success,
    Failed,
    Blocked,
}

/// Audit Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditRequest {
    pub user_id: String,
    pub action: String,
    pub resource: String,
}

/// Security Check
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityCheck {
    pub id: String,
    pub check_type: String,
    pub passed: bool,
    pub details: String,
    pub timestamp: DateTime<Utc>,
}