//! Data models for Fiat Ramp service

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

// KYC Level constants
pub const KYC_LEVEL_NONE: u8 = 0;
pub const KYC_LEVEL_BASIC: u8 = 1;
pub const KYC_LEVEL_MEDIUM: u8 = 2;
pub const KYC_LEVEL_HIGH: u8 = 3;
pub const KYC_LEVEL_MAX: u8 = 4;

/// KYC Level
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum KycLevel {
    None = 0,
    Basic = 1,
    Medium = 2,
    High = 3,
    Max = 4,
}

impl Default for KycLevel {
    fn default() -> Self {
        KycLevel::None
    }
}

/// KYC Status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum KycStatus {
    Pending,
    InProgress,
    Verified,
    Rejected,
    Expired,
}

/// KYC Application
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KycApplication {
    pub id: String,
    pub user_id: String,
    pub email: String,
    pub phone: Option<String>,
    pub country: String,
    pub level: KycLevel,
    pub status: KycStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Start KYC Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartKycRequest {
    pub user_id: String,
    pub email: String,
    pub phone: Option<String>,
    pub country: String,
    pub level: u8,
}

/// Verify KYC Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyKycRequest {
    pub code: String,
}

/// Upload Document Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadDocumentRequest {
    pub doc_type: String,
    pub data: String,
}

/// Quote Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuoteRequest {
    pub user_id: String,
    pub fiat_amt: f64,
    pub crypto_amt: f64,
    pub fiat: String,
    pub crypto: String,
    pub method: String,
}

/// Quote Response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Quote {
    pub id: String,
    pub fiat_amt: f64,
    pub crypto_amt: f64,
    pub rate: f64,
    pub expires_at: DateTime<Utc>,
}

/// Execute Buy Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecuteBuyRequest {
    pub user_id: String,
    pub quote_id: String,
    pub fiat_amt: f64,
    pub crypto: String,
    pub method: String,
    pub bank_account: String,
}

/// Execute Sell Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecuteSellRequest {
    pub user_id: String,
    pub quote_id: String,
    pub crypto_amt: f64,
    pub crypto: String,
    pub method: String,
    pub bank_account: String,
}

/// Payment Method
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaymentMethod {
    pub id: String,
    pub name: String,
    pub method_type: String,
    pub supported_fiats: Vec<String>,
    pub supported_cryptos: Vec<String>,
    pub fees: f64,
}

/// Create Payment Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePaymentRequest {
    pub user_id: String,
    pub order_id: String,
    pub amount: f64,
    pub currency: String,
    pub method: String,
}

/// Payment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Payment {
    pub id: String,
    pub user_id: String,
    pub order_id: String,
    pub amount: f64,
    pub currency: String,
    pub method: String,
    pub status: PaymentStatus,
    pub created_at: DateTime<Utc>,
}

/// Payment Status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum PaymentStatus {
    Pending,
    Processing,
    Completed,
    Failed,
    Refunded,
}

/// Limits
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Limits {
    pub level: u8,
    pub daily_limit: f64,
    pub monthly_limit: f64,
    pub yearly_limit: f64,
}

/// Update Limits Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateLimitsRequest {
    pub level: u8,
    pub daily_limit: f64,
    pub monthly_limit: f64,
    pub yearly_limit: f64,
}

/// Bank Account
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BankAccount {
    pub id: String,
    pub user_id: String,
    pub bank_name: String,
    pub account_num: String,
    pub routing_num: String,
    pub country: String,
    pub currency: String,
    pub created_at: DateTime<Utc>,
}

/// Add Bank Account Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddBankAccountRequest {
    pub user_id: String,
    pub bank_name: String,
    pub account_num: String,
    pub routing_num: String,
    pub country: String,
    pub currency: String,
}

/// Withdrawal Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WithdrawalRequest {
    pub user_id: String,
    pub amount: f64,
    pub currency: String,
    pub bank_account: String,
}

/// Withdrawal
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Withdrawal {
    pub id: String,
    pub user_id: String,
    pub amount: f64,
    pub currency: String,
    pub bank_account: String,
    pub status: WithdrawalStatus,
    pub created_at: DateTime<Utc>,
}

/// Withdrawal Status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum WithdrawalStatus {
    Pending,
    Processing,
    Completed,
    Failed,
}

/// Screen Request (Compliance)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScreenRequest {
    pub user_id: String,
    pub amount: f64,
    pub tx_type: String,
    pub country: String,
}

/// Screen Result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScreenResult {
    pub approved: bool,
    pub risk_score: f64,
    pub flags: Vec<String>,
}

/// Report Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReportRequest {
    pub user_id: String,
    pub tx_type: String,
    pub details: String,
}

/// Config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub port: String,
    pub database_url: String,
    pub redis_url: String,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            port: "8083".to_string(),
            database_url: "postgres://localhost:5432/tigerwallet".to_string(),
            redis_url: "localhost:6379".to_string(),
        }
    }
}

/// Health Response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: i64,
}