//! Manager Module - Wallet management

use std::collections::HashMap;
use tokio::sync::RwLock;

use crate::{Chain, HDWallet, WalletError};

/// Wallet record stored in the manager.
#[derive(Debug, Clone)]
pub struct WalletRecord {
    pub id: String,
    pub name: String,
    pub address: String,
    pub chain: Chain,
    pub created_at: i64,
    pub is_primary: bool,
}

/// Wallet info for API responses
#[derive(Debug, Clone)]
pub struct WalletInfo {
    pub wallet_id: String,
    pub name: String,
    pub address: String,
    pub chain: Chain,
    pub mnemonic: String,
}

/// Balance information
#[derive(Debug, Clone)]
pub struct Balance {
    pub address: String,
    pub chain: Chain,
    pub balances: Vec<TokenBalance>,
}

/// Token balance
#[derive(Debug, Clone)]
pub struct TokenBalance {
    pub symbol: String,
    pub balance: String,
    pub balance_raw: String,
}

/// Wallet Manager
pub struct WalletManager {
    wallets: RwLock<HashMap<String, WalletRecord>>,
    keys: RwLock<HashMap<String, HDWallet>>,
}

impl WalletManager {
    pub fn new() -> Self {
        Self {
            wallets: RwLock::new(HashMap::new()),
            keys: RwLock::new(HashMap::new()),
        }
    }

    /// Create new wallet
    pub async fn create_wallet(
        &self,
        name: String,
        chain: Chain,
    ) -> Result<WalletInfo, WalletError> {
        let wallet = HDWallet::generate(chain);
        let address = wallet.default_address()?;
        let mnemonic = wallet.mnemonic_phrase();
        let wallet_id = uuid::Uuid::new_v4().to_string();

        let record = WalletRecord {
            id: wallet_id.clone(),
            name: name.clone(),
            address: address.clone(),
            chain,
            created_at: chrono::Utc::now().timestamp(),
            is_primary: false,
        };

        self.wallets.write().await.insert(wallet_id.clone(), record);
        self.keys.write().await.insert(wallet_id.clone(), wallet);

        Ok(WalletInfo {
            wallet_id,
            name,
            address,
            chain,
            mnemonic,
        })
    }

    /// Import wallet from mnemonic
    pub async fn import_wallet(
        &self,
        name: String,
        mnemonic: &str,
        chain: Chain,
    ) -> Result<WalletInfo, WalletError> {
        let parsed = crate::Mnemonic::from_phrase(mnemonic)?;
        let wallet = HDWallet::from_mnemonic(parsed, chain);
        let address = wallet.default_address()?;
        let wallet_id = uuid::Uuid::new_v4().to_string();

        let record = WalletRecord {
            id: wallet_id.clone(),
            name: name.clone(),
            address: address.clone(),
            chain,
            created_at: chrono::Utc::now().timestamp(),
            is_primary: false,
        };

        self.wallets.write().await.insert(wallet_id.clone(), record);
        self.keys.write().await.insert(wallet_id.clone(), wallet);

        Ok(WalletInfo {
            wallet_id,
            name,
            address,
            chain,
            mnemonic: mnemonic.to_string(),
        })
    }

    /// Get wallet
    pub async fn get_wallet(&self, wallet_id: &str) -> Option<WalletInfo> {
        let wallets = self.wallets.read().await;
        wallets.get(wallet_id).map(|w| WalletInfo {
            wallet_id: w.id.clone(),
            name: w.name.clone(),
            address: w.address.clone(),
            chain: w.chain,
            mnemonic: String::new(),
        })
    }

    /// List wallets
    pub async fn list_wallets(&self) -> Vec<WalletInfo> {
        let wallets = self.wallets.read().await;
        wallets
            .values()
            .map(|w| WalletInfo {
                wallet_id: w.id.clone(),
                name: w.name.clone(),
                address: w.address.clone(),
                chain: w.chain,
                mnemonic: String::new(),
            })
            .collect()
    }

    /// Sign a transaction payload with the wallet's own derived key.
    pub async fn sign_transaction(
        &self,
        wallet_id: &str,
        message: &[u8],
    ) -> Result<String, WalletError> {
        let keys = self.keys.read().await;
        let wallet = keys.get(wallet_id).ok_or(WalletError::WalletNotFound)?;
        let signature = wallet.sign(message)?;
        Ok(hex::encode(signature))
    }

    /// Sign an arbitrary 32-byte digest with the wallet's own derived key.
    pub async fn sign_digest(
        &self,
        wallet_id: &str,
        digest: &[u8; 32],
    ) -> Result<String, WalletError> {
        let keys = self.keys.read().await;
        let wallet = keys.get(wallet_id).ok_or(WalletError::WalletNotFound)?;
        Ok(hex::encode(wallet.sign_digest(digest)?))
    }

    /// Return the wallet's public identity and its (currently unknown) holdings.
    ///
    /// On-chain balances are read by the Go/Rust fetcher services, which own the
    /// RPC connections. This method reports the address and an empty holding set
    /// rather than inventing a balance.
    pub async fn get_balance(&self, wallet_id: &str) -> Result<Balance, WalletError> {
        let wallets = self.wallets.read().await;
        let record = wallets.get(wallet_id).ok_or(WalletError::WalletNotFound)?;

        Ok(Balance {
            address: record.address.clone(),
            chain: record.chain,
            balances: Vec::new(),
        })
    }

    /// Delete wallet
    pub async fn delete_wallet(&self, wallet_id: &str) -> Result<(), WalletError> {
        let mut wallets = self.wallets.write().await;
        wallets.remove(wallet_id).ok_or(WalletError::WalletNotFound)?;
        self.keys.write().await.remove(wallet_id);
        Ok(())
    }
}

impl Default for WalletManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_wallet() {
        let manager = WalletManager::new();
        let wallet = manager
            .create_wallet("Test".to_string(), Chain::Ethereum)
            .await
            .unwrap();
        assert!(!wallet.wallet_id.is_empty());
        assert!(wallet.address.starts_with("0x"));
        assert_eq!(wallet.address.len(), 42);
    }

    #[tokio::test]
    async fn test_import_wallet_is_deterministic() {
        let manager = WalletManager::new();
        let phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        let wallet = manager
            .import_wallet("Imported".to_string(), phrase, Chain::Ethereum)
            .await
            .unwrap();
        assert_eq!(wallet.address, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
    }

    #[tokio::test]
    async fn test_sign_transaction_uses_wallet_key() {
        let manager = WalletManager::new();
        let phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        let wallet = manager
            .import_wallet("Signer".to_string(), phrase, Chain::Ethereum)
            .await
            .unwrap();

        let sig_a = manager
            .sign_transaction(&wallet.wallet_id, b"test message")
            .await
            .unwrap();
        let sig_b = manager
            .sign_transaction(&wallet.wallet_id, b"test message")
            .await
            .unwrap();

        // Signing must be deterministic for the same key and payload; the old
        // implementation generated a random wallet per call and never matched.
        assert_eq!(sig_a, sig_b);
        assert_eq!(sig_a.len(), 130);
    }

    #[tokio::test]
    async fn test_balance_reports_address_without_fabricating() {
        let manager = WalletManager::new();
        let wallet = manager
            .create_wallet("Test".to_string(), Chain::Ethereum)
            .await
            .unwrap();
        let balance = manager.get_balance(&wallet.wallet_id).await.unwrap();
        assert_eq!(balance.address, wallet.address);
        assert!(balance.balances.is_empty());
    }

    #[tokio::test]
    async fn test_delete_wallet() {
        let manager = WalletManager::new();
        let wallet = manager
            .create_wallet("Test".to_string(), Chain::Ethereum)
            .await
            .unwrap();
        manager.delete_wallet(&wallet.wallet_id).await.unwrap();
        assert!(manager.get_wallet(&wallet.wallet_id).await.is_none());
        assert!(manager
            .sign_transaction(&wallet.wallet_id, b"x")
            .await
            .is_err());
    }
}
