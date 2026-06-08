//! TigerWallet Core - Rust Implementation
//! High-performance wallet core functions

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;
use std::collections::HashMap;

/// Wallet type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum WalletType {
    Mnemonic,
    PrivateKey,
    Hardware,
    MPC,
    SocialRecovery,
}

/// Blockchain
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Blockchain {
    Ethereum,
    Bitcoin,
    Solana,
    BSC,
    Polygon,
    Avalanche,
    Arbitrum,
    Optimism,
    Aptos,
    Sui,
    TON,
}

/// Token info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub address: String,
    pub symbol: String,
    pub name: String,
    pub decimals: u8,
    pub chain: Blockchain,
    pub logo_url: String,
}

/// Wallet account
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Account {
    pub id: String,
    pub address: String,
    pub wallet_type: WalletType,
    pub chain: Blockchain,
    pub name: String,
    pub created_at: i64,
    pub last_used: i64,
}

/// Token balance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenBalance {
    pub token: Token,
    pub balance: String,
    pub balance_usd: f64,
    pub balance_raw: String,
}

/// Wallet core engine
pub struct WalletCore {
    accounts: RwLock<HashMap<String, Account>>,
    balances: RwLock<HashMap<String, Vec<TokenBalance>>>,
}

impl WalletCore {
    pub fn new() -> Self {
        Self {
            accounts: RwLock::new(HashMap::new()),
            balances: RwLock::new(HashMap::new()),
        }
    }

    /// Create new wallet
    pub fn create_wallet(&self, wallet_type: WalletType, chain: Blockchain) -> Account {
        let account = Account {
            id: uuid::Uuid::new_v4().to_string(),
            address: self.generate_address(chain),
            wallet_type,
            chain,
            name: "TigerWallet".to_string(),
            created_at: chrono::Utc::now().timestamp(),
            last_used: chrono::Utc::now().timestamp(),
        };
        
        self.accounts.write().insert(account.id.clone(), account.clone());
        account
    }

    /// Generate address for chain
    fn generate_address(&self, chain: Blockchain) -> String {
        match chain {
            Blockchain::Ethereum | Blockchain::BSC | Blockchain::Polygon 
            | Blockchain::Arbitrum | Blockchain::Optimism | Blockchain::Avalanche => {
                format!("0x{}", hex::encode(&[0u8; 20]))
            }
            Blockchain::Bitcoin => "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh".to_string(),
            Blockchain::Solana => format!("{}", base58::encode(&[0u8; 32])),
            Blockchain::Aptos => format!("0x{}", hex::encode(&[0u8; 32])),
            Blockchain::Sui => format!("0x{}", hex::encode(&[0u8; 32])),
            Blockchain::TON => "UQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA".to_string(),
        }
    }

    /// Get account
    pub fn get_account(&self, id: &str) -> Option<Account> {
        self.accounts.read().get(id).cloned()
    }

    /// Get balance
    pub fn get_balance(&self, account_id: &str) -> Vec<TokenBalance> {
        self.balances.read().get(account_id).cloned().unwrap_or_default()
    }

    /// Update balance
    pub fn update_balance(&self, account_id: &str, token_balances: Vec<TokenBalance>) {
        self.balances.write().insert(account_id.to_string(), token_balances);
    }
}

impl Default for WalletCore {
    fn default() -> Self {
        Self::new()
    }
}