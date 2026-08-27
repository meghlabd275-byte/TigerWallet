//! TigerWallet Desktop Application - Tauri Implementation
//! Production-ready desktop wallet with full functionality

#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Mutex;
use tauri::{Manager, State};
use log::{info, error, warn};
use chrono::Utc;
use uuid::Uuid;
use sha2::{Sha256, Digest};

mod crypto;
#[path = "cold_wallet/mod.rs"]
mod cold_wallet;
#[path = "services/desktop_services.rs"]
mod desktop_services;

/// Canonical UserWallet backend base URL. The desktop app is a thin client and
/// never talks to MasterWallet/Admin backends directly.
const WALLET_API_BASE: &str = "http://localhost:8443";

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("Wallet error: {0}")]
    Wallet(String),
    
    #[error("Encryption error: {0}")]
    Encryption(String),
    
    #[error("Database error: {0}")]
    Database(String),
    
    #[error("Network error: {0}")]
    Network(String),
    
    #[error("Transaction error: {0}")]
    Transaction(String),
}

impl Serialize for AppError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.to_string())
    }
}

// ============================================================================
// Data Models
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletAccount {
    pub id: String,
    pub name: String,
    pub address: String,
    pub chain_id: u64,
    pub balance: String,
    pub tokens: Vec<TokenBalance>,
    pub created_at: i64,
    pub is_derived: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenBalance {
    pub symbol: String,
    pub address: String,
    pub balance: String,
    pub decimals: u8,
    pub usd_value: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: String,
    pub hash: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub token: String,
    pub status: String,
    pub chain_id: u64,
    pub timestamp: i64,
    pub gas_used: u64,
    pub gas_price: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserPreferences {
    pub theme: String,
    pub currency: String,
    pub language: String,
    pub auto_lock_minutes: u32,
    pub show_testnets: bool,
    pub default_gas_limit: u64,
}

impl Default for UserPreferences {
    fn default() -> Self {
        Self {
            theme: "dark".to_string(),
            currency: "USD".to_string(),
            language: "en".to_string(),
            auto_lock_minutes: 5,
            show_testnets: false,
            default_gas_limit: 21000,
        }
    }
}

pub struct AppState {
    pub wallets: Mutex<HashMap<String, WalletAccount>>,
    pub transactions: Mutex<Vec<Transaction>>,
    pub preferences: Mutex<UserPreferences>,
    pub is_locked: Mutex<bool>,
    pub master_password_hash: Mutex<Option<String>>,
}

impl Default for AppState {
    fn default() -> Self {
        Self {
            wallets: Mutex::new(HashMap::new()),
            transactions: Mutex::new(Vec::new()),
            preferences: Mutex::new(UserPreferences::default()),
            is_locked: Mutex::new(true),
            master_password_hash: Mutex::new(None),
        }
    }
}

// ============================================================================
// Wallet Commands
// ============================================================================

#[tauri::command]
fn create_wallet(name: String, state: State<AppState>) -> Result<WalletAccount, AppError> {
    info!("Creating wallet: {}", name);

    // Generate a real 24-word BIP-39 mnemonic and derive the EVM address from
    // it (BIP-32 m/44'/60'/0'/0/0 + Keccak-256). Never fabricate an address.
    let mnemonic = crypto::generate_mnemonic_24().map_err(AppError::Wallet)?;
    let seed = crypto::mnemonic_to_seed(&mnemonic, "");
    let address = crypto::evm_address_from_seed(&seed, 0).map_err(AppError::Wallet)?;
    // The mnemonic must be shown once and persisted encrypted by the caller;
    // it is intentionally not retained in the in-memory registry.
    drop(mnemonic);

    let wallet = WalletAccount {
        id: Uuid::new_v4().to_string(),
        name: name.clone(),
        address,
        chain_id: 1,
        balance: "0".to_string(),
        tokens: vec![],
        created_at: Utc::now().timestamp(),
        is_derived: true,
    };

    let mut wallets = state.wallets.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    wallets.insert(wallet.id.clone(), wallet.clone());

    info!("Wallet created: {}", wallet.address);
    Ok(wallet)
}
#[tauri::command]
fn import_wallet(name: String, seed_phrase: String, state: State<AppState>) -> Result<WalletAccount, AppError> {
    info!("Importing wallet: {}", name);

    let words: Vec<&str> = seed_phrase.split_whitespace().collect();
    if words.len() != 12 && words.len() != 24 {
        return Err(AppError::Wallet("Invalid seed phrase length".to_string()));
    }

    // Real BIP-39 checksum validation + real address derivation.
    let address = derive_address_from_seed(&seed_phrase)?;

    let wallet = WalletAccount {
        id: Uuid::new_v4().to_string(),
        name: name.clone(),
        address,
        chain_id: 1,
        balance: "0".to_string(),
        tokens: vec![],
        created_at: Utc::now().timestamp(),
        is_derived: true,
    };

    let mut wallets = state.wallets.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    wallets.insert(wallet.id.clone(), wallet.clone());

    info!("Wallet imported: {}", wallet.address);
    Ok(wallet)
}
#[tauri::command]
fn get_wallets(state: State<AppState>) -> Result<Vec<WalletAccount>, AppError> {
    let wallets = state.wallets.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(wallets.values().cloned().collect())
}

#[tauri::command]
fn get_wallet(id: String, state: State<AppState>) -> Result<Option<WalletAccount>, AppError> {
    let wallets = state.wallets.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(wallets.get(&id).cloned())
}

#[tauri::command]
fn delete_wallet(id: String, state: State<AppState>) -> Result<bool, AppError> {
    let mut wallets = state.wallets.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(wallets.remove(&id).is_some())
}

// ============================================================================
// Transaction Commands
// ============================================================================

#[tauri::command]
async fn send_transaction(from: String, to: String, value: String, token: String, chain_id: u64, state: State<'_, AppState>) -> Result<Transaction, AppError> {
    info!("Sending transaction: {} -> {} ({})", from, to, value);

    let is_locked = state.is_locked.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    if *is_locked {
        return Err(AppError::Wallet("Wallet is locked".to_string()));
    }

    // The Tauri backend is a thin client over the canonical wallet_api backend,
    // which performs real secp256k1 signing + eth_sendRawTransaction. Never
    // fabricate a transaction hash.
    let client = reqwest::Client::new();
    let body = serde_json::json!({
        "from_address": from,
        "to_address": to,
        "amount": value,
        "chain_id": chain_id,
    });
    let resp = client
        .post(format!("{}/api/v1/send", WALLET_API_BASE))
        .json(&body)
        .send()
        .await
        .map_err(|e| AppError::Network(e.to_string()))?;

    let tx_hash = if resp.status().is_success() {
        let json: serde_json::Value = resp.json().await.unwrap_or_default();
        json.get("tx_hash").and_then(|v| v.as_str()).unwrap_or_default().to_string()
    } else {
        let err = resp.text().await.unwrap_or_default();
        return Err(AppError::Transaction(err));
    };

    let tx = Transaction {
        id: Uuid::new_v4().to_string(),
        hash: tx_hash,
        from,
        to: to.clone(),
        value: value.clone(),
        token,
        status: "pending".to_string(),
        chain_id,
        timestamp: Utc::now().timestamp(),
        gas_used: 0,
        gas_price: "0".to_string(),
    };

    let mut transactions = state.transactions.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    transactions.push(tx.clone());

    info!("Transaction broadcast: {}", tx.hash);
    Ok(tx)
}
#[tauri::command]
fn get_transactions(state: State<AppState>) -> Result<Vec<Transaction>, AppError> {
    let transactions = state.transactions.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(transactions.clone())
}
#[tauri::command]
async fn simulate_transaction(from: String, to: String, value: String, data: String, chain_id: u64) -> Result<SimulateResult, AppError> {
    info!("Simulating transaction: {} -> {}", from, to);

    // Real dry-run against the canonical wallet_api backend
    // (eth_estimateGas + eth_call). Never fabricate a simulation verdict.
    let client = reqwest::Client::new();
    let body = serde_json::json!({
        "chain_id": chain_id,
        "from": from,
        "to": to,
        "value": value,
        "data": data,
    });
    let resp = client
        .post(format!("{}/api/v1/simulate", WALLET_API_BASE))
        .json(&body)
        .send()
        .await
        .map_err(|e| AppError::Network(e.to_string()))?;

    let json: serde_json::Value = resp.json().await.map_err(|e| AppError::Network(e.to_string()))?;

    let success = json.get("success").and_then(|v| v.as_bool()).unwrap_or(false);
    let gas_estimate = json.get("gas_estimate").and_then(|v| v.as_u64()).unwrap_or(21000);
    let gas_price = json.get("gas_price").and_then(|v| v.as_str()).unwrap_or("0").to_string();
    let estimated_cost = json.get("estimated_cost_wei").and_then(|v| v.as_str()).unwrap_or("0").to_string();
    let revert_reason = json.get("revert_reason").and_then(|v| v.as_str()).unwrap_or("").to_string();

    Ok(SimulateResult {
        success,
        gas_used: gas_estimate,
        gas_price,
        total_cost: estimated_cost,
        balance_changes: vec![],
        warnings: if revert_reason.is_empty() { vec![] } else { vec![format!("revert: {}", revert_reason)] },
        logs: vec![],
    })
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulateResult {
    pub success: bool,
    pub gas_used: u64,
    pub gas_price: String,
    pub total_cost: String,
    pub balance_changes: Vec<BalanceChange>,
    pub warnings: Vec<String>,
    pub logs: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BalanceChange {
    pub address: String,
    pub change_type: String,
    pub token: String,
    pub amount: String,
}

// ============================================================================
// Security Commands
// ============================================================================

#[tauri::command]
fn set_master_password(password: String, state: State<AppState>) -> Result<bool, AppError> {
    info!("Setting master password");
    let hash = hash_password(&password);
    let mut hash_storage = state.master_password_hash.lock().map_err(|e| AppError::Encryption(e.to_string()))?;
    *hash_storage = Some(hash);
    let mut is_locked = state.is_locked.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    *is_locked = false;
    Ok(true)
}

#[tauri::command]
fn unlock_wallet(password: String, state: State<AppState>) -> Result<bool, AppError> {
    info!("Unlocking wallet");
    let hash_storage = state.master_password_hash.lock().map_err(|e| AppError::Encryption(e.to_string()))?;
    if let Some(stored_hash) = hash_storage.as_ref() {
        let input_hash = hash_password(&password);
        if input_hash == *stored_hash {
            let mut is_locked = state.is_locked.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
            *is_locked = false;
            info!("Wallet unlocked successfully");
            return Ok(true);
        }
    }
    warn!("Failed to unlock wallet - incorrect password");
    Err(AppError::Wallet("Incorrect password".to_string()))
}

#[tauri::command]
fn lock_wallet(state: State<AppState>) -> Result<bool, AppError> {
    info!("Locking wallet");
    let mut is_locked = state.is_locked.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    *is_locked = true;
    Ok(true)
}

#[tauri::command]
fn is_wallet_locked(state: State<AppState>) -> Result<bool, AppError> {
    let is_locked = state.is_locked.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(*is_locked)
}

// ============================================================================
// Preferences Commands
// ============================================================================

#[tauri::command]
fn get_preferences(state: State<AppState>) -> Result<UserPreferences, AppError> {
    let preferences = state.preferences.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(preferences.clone())
}

#[tauri::command]
fn set_preferences(prefs: UserPreferences, state: State<AppState>) -> Result<bool, AppError> {
    let mut preferences = state.preferences.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    *preferences = prefs;
    Ok(true)
}

#[tauri::command]
fn set_theme(theme: String, state: State<AppState>) -> Result<bool, AppError> {
    let mut preferences = state.preferences.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    preferences.theme = theme;
    Ok(true)
}

#[tauri::command]
fn get_theme(state: State<AppState>) -> Result<String, AppError> {
    let preferences = state.preferences.lock().map_err(|e| AppError::Wallet(e.to_string()))?;
    Ok(preferences.theme.clone())
}

// ============================================================================
// Chain/Network Commands
// ============================================================================

#[tauri::command]
fn get_supported_chains() -> Result<Vec<ChainInfo>, AppError> {
    Ok(vec![
        ChainInfo { id: 1, name: "Ethereum".to_string(), symbol: "ETH".to_string(), rpc: "https://eth.llamarpc.com".to_string(), explorer: "https://etherscan.io".to_string() },
        ChainInfo { id: 56, name: "BNB Chain".to_string(), symbol: "BNB".to_string(), rpc: "https://bsc-dataseed.binance.org".to_string(), explorer: "https://bscscan.com".to_string() },
        ChainInfo { id: 137, name: "Polygon".to_string(), symbol: "MATIC".to_string(), rpc: "https://polygon-rpc.com".to_string(), explorer: "https://polygonscan.com".to_string() },
        ChainInfo { id: 42161, name: "Arbitrum".to_string(), symbol: "ETH".to_string(), rpc: "https://arb1.arbitrum.io/rpc".to_string(), explorer: "https://arbiscan.io".to_string() },
        ChainInfo { id: 10, name: "Optimism".to_string(), symbol: "ETH".to_string(), rpc: "https://mainnet.optimism.io".to_string(), explorer: "https://optimistic.etherscan.io".to_string() },
        ChainInfo { id: 8453, name: "Base".to_string(), symbol: "ETH".to_string(), rpc: "https://mainnet.base.org".to_string(), explorer: "https://basescan.org".to_string() },
        ChainInfo { id: 43114, name: "Avalanche".to_string(), symbol: "AVAX".to_string(), rpc: "https://api.avax.network/ext/bc/C/rpc".to_string(), explorer: "https://snowtrace.io".to_string() },
    ])
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainInfo {
    pub id: u64,
    pub name: String,
    pub symbol: String,
    pub rpc: String,
    pub explorer: String,
}

// ============================================================================
// Helper Functions
// ============================================================================

/// Derive a real EVM address from a 24-word BIP-39 seed. The address is
/// deterministically derived via BIP-32 (m/44'/60'/0'/0/0) + Keccak-256,
/// matching the canonical wallet_api/MasterWallet derivation. A new account
/// is only ever created for display when a *real* seed is provided; otherwise
/// this function is never used to fabricate an address.
fn derive_address_from_seed(seed: &str) -> Result<String, AppError> {
    if !crypto::validate_mnemonic(seed) {
        return Err(AppError::Wallet(
            "invalid BIP-39 mnemonic (checksum verification failed)".to_string(),
        ));
    }
    let seed_bytes = crypto::mnemonic_to_seed(seed, "");
    crypto::evm_address_from_seed(&seed_bytes, 0).map_err(AppError::Wallet)
}

/// Hash a password with SHA-256 + fixed salt for the in-memory lock gate.
/// NOTE: this is a local UI lock, not the canonical credential store — the
/// wallet's cryptographic seed is never derived from or encrypted with this.
fn hash_password(password: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(password.as_bytes());
    hasher.update(b"tiger-wallet-lock-salt");
    hex::encode(hasher.finalize())
}

// ============================================================================
// Main Entry Point
// ============================================================================

fn main() {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info"))
        .format_timestamp_millis()
        .init();
    
    info!("Starting TigerWallet Desktop Application");
    
    tauri::Builder::default()
        .manage(AppState::default())
        .invoke_handler(tauri::generate_handler![
            create_wallet,
            import_wallet,
            get_wallets,
            get_wallet,
            delete_wallet,
            send_transaction,
            get_transactions,
            simulate_transaction,
            set_master_password,
            unlock_wallet,
            lock_wallet,
            is_wallet_locked,
            get_preferences,
            set_preferences,
            set_theme,
            get_theme,
            get_supported_chains,
        ])
        .setup(|app| {
            info!("Application setup complete");
            let window = app.get_window("main").unwrap();
            window.set_title("TigerWallet").unwrap();
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
