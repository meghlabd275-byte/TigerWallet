/**
 * TigerWallet Multi-Sig Wallet Service
 * Threshold Signature & Multi-Sig Transaction Management
 */

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use uuid::Uuid;

// ============================================================================
// Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum MultiSigType {
    MultiSig,       // M-of-N signatures required
    Threshold,       // Threshold signature scheme
    RoleBased,       // Role-based permissions
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TransactionStatus {
    Pending,
    Confirmed,
    Executed,
    Failed,
    Cancelled,
}

// ============================================================================
// Core Structures
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Wallet {
    pub wallet_id: String,
    pub name: String,
    pub owners: Vec<String>,           // Owner addresses
    pub threshold: u8,                // Signatures required
    pub multi_sig_type: MultiSigType,
    pub chain_id: u64,
    pub address: String,
    pub balance: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub tx_id: String,
    pub wallet_id: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub data: String,
    pub gas_limit: u64,
    pub gas_price: String,
    pub nonce: u64,
    pub chain_id: u64,
    pub status: TransactionStatus,
    pub confirmations: Vec<Confirmation>,
    pub required_confirmations: u8,
    pub executed_at: Option<DateTime<Utc>>,
    pub tx_hash: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Confirmation {
    pub owner: String,
    pub signature: String,
    pub signed_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Owner {
    pub owner_id: String,
    pub address: String,
    pub name: String,
    pub is_active: bool,
    pub added_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PendingTransaction {
    pub tx_id: String,
    pub wallet_id: String,
    pub transaction: Transaction,
    pub pending_signatures: Vec<String>,  // Owners who haven't signed yet
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

// ============================================================================
// Multi-Sig Service
// ============================================================================

pub struct MultiSigService {
    wallets: Arc<RwLock<HashMap<String, Wallet>>>,
    transactions: Arc<RwLock<HashMap<String, Transaction>>>,
    pending_txs: Arc<RwLock<HashMap<String, PendingTransaction>>>,
    owners: Arc<RwLock<HashMap<String, Owner>>>,
}

impl MultiSigService {
    pub fn new() -> Self {
        MultiSigService {
            wallets: Arc::new(RwLock::new(HashMap::new())),
            transactions: Arc::new(RwLock::new(HashMap::new())),
            pending_txs: Arc::new(RwLock::new(HashMap::new())),
            owners: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    // ========================================================================
    // Wallet Management
    // ========================================================================

    pub fn create_wallet(
        &self,
        name: String,
        owners: Vec<String>,
        threshold: u8,
        multi_sig_type: MultiSigType,
        chain_id: u64,
    ) -> Result<Wallet, String> {
        // Validate threshold
        if threshold == 0 || threshold as usize > owners.len() {
            return Err("Invalid threshold".to_string());
        }

        let wallet_id = Uuid::new_v4().to_string();
        let address = self.generate_wallet_address(&wallet_id);

        let wallet = Wallet {
            wallet_id: wallet_id.clone(),
            name,
            owners: owners.clone(),
            threshold,
            multi_sig_type,
            chain_id,
            address,
            balance: "0".to_string(),
            is_active: true,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };

        // Register owners
        for owner_addr in &owners {
            let owner = Owner {
                owner_id: Uuid::new_v4().to_string(),
                address: owner_addr.clone(),
                name: "Owner".to_string(),
                is_active: true,
                added_at: Utc::now(),
            };
            self.add_owner(owner)?;
        }

        let mut wallets = self.wallets.write().map_err(|e| e.to_string())?;
        wallets.insert(wallet_id, wallet.clone());

        Ok(wallet)
    }

    pub fn get_wallet(&self, wallet_id: &str) -> Result<Option<Wallet>, String> {
        let wallets = self.wallets.read().map_err(|e| e.to_string())?;
        Ok(wallets.get(wallet_id).cloned())
    }

    pub fn list_wallets(&self) -> Result<Vec<Wallet>, String> {
        let wallets = self.wallets.read().map_err(|e| e.to_string())?;
        Ok(wallets.values().cloned().collect())
    }

    pub fn get_wallets_by_owner(&self, owner_address: &str) -> Result<Vec<Wallet>, String> {
        let wallets = self.wallets.read().map_err(|e| e.to_string())?;
        let result: Vec<Wallet> = wallets
            .values()
            .filter(|w| w.owners.contains(&owner_address.to_string()))
            .cloned()
            .collect();
        Ok(result)
    }

    pub fn add_owner(&self, owner: Owner) -> Result<(), String> {
        let mut owners = self.owners.write().map_err(|e| e.to_string())?;
        owners.insert(owner.address.clone(), owner);
        Ok(())
    }

    pub fn remove_owner(&self, wallet_id: &str, owner_address: &str) -> Result<(), String> {
        let mut wallets = self.wallets.write().map_err(|e| e.to_string())?;

        if let Some(wallet) = wallets.get_mut(wallet_id) {
            wallet.owners.retain(|o| o != owner_address);
            wallet.updated_at = Utc::now();
            Ok(())
        } else {
            Err("Wallet not found".to_string())
        }
    }

    pub fn update_threshold(&self, wallet_id: &str, new_threshold: u8) -> Result<(), String> {
        let mut wallets = self.wallets.write().map_err(|e| e.to_string())?;

        if let Some(wallet) = wallets.get_mut(wallet_id) {
            if new_threshold as usize > wallet.owners.len() {
                return Err("Threshold cannot exceed number of owners".to_string());
            }
            wallet.threshold = new_threshold;
            wallet.updated_at = Utc::now();
            Ok(())
        } else {
            Err("Wallet not found".to_string())
        }
    }

    // ========================================================================
    // Transaction Management
    // ========================================================================

    pub fn create_transaction(
        &self,
        wallet_id: &str,
        to: String,
        value: String,
        data: String,
    ) -> Result<Transaction, String> {
        let wallet = {
            let wallets = self.wallets.read().map_err(|e| e.to_string())?;
            wallets.get(wallet_id).cloned()
        }.ok_or("Wallet not found")?;

        let tx_id = Uuid::new_v4().to_string();

        let transaction = Transaction {
            tx_id: tx_id.clone(),
            wallet_id: wallet_id.to_string(),
            from: wallet.address.clone(),
            to,
            value,
            data,
            gas_limit: 21000,
            gas_price: "1".to_string(),
            nonce: 0,
            chain_id: wallet.chain_id,
            status: TransactionStatus::Pending,
            confirmations: Vec::new(),
            required_confirmations: wallet.threshold,
            executed_at: None,
            tx_hash: None,
            created_at: Utc::now(),
        };

        // Create pending transaction
        let pending_tx = PendingTransaction {
            tx_id: tx_id.clone(),
            wallet_id: wallet_id.to_string(),
            transaction: transaction.clone(),
            pending_signatures: wallet.owners.clone(),
            created_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::days(7),
        };

        let mut pending = self.pending_txs.write().map_err(|e| e.to_string())?;
        pending.insert(tx_id.clone(), pending_tx);

        let mut txs = self.transactions.write().map_err(|e| e.to_string())?;
        txs.insert(tx_id, transaction.clone());

        Ok(transaction)
    }

    pub fn confirm_transaction(
        &self,
        tx_id: &str,
        owner_address: &str,
        signature: String,
    ) -> Result<Transaction, String> {
        let mut pending = self.pending_txs.write().map_err(|e| e.to_string())?;

        if let Some(pending_tx) = pending.get_mut(tx_id) {
            // Verify owner is part of wallet
            if !pending_tx.pending_signatures.contains(&owner_address.to_string()) {
                return Err("Not an owner of this wallet".to_string());
            }

            // Add confirmation
            let confirmation = Confirmation {
                owner: owner_address.to_string(),
                signature,
                signed_at: Utc::now(),
            };

            // Update pending signatures
            pending_tx.pending_signatures.retain(|o| o != owner_address);

            // Update transaction
            let mut txs = self.transactions.write().map_err(|e| e.to_string())?;
            if let Some(tx) = txs.get_mut(tx_id) {
                tx.confirmations.push(confirmation);

                // Check if threshold met
                if tx.confirmations.len() as u8 >= tx.required_confirmations {
                    tx.status = TransactionStatus::Confirmed;
                    pending.remove(tx_id);
                }

                return Ok(tx.clone());
            }
        }

        Err("Transaction not found".to_string())
    }

    pub fn execute_transaction(&self, tx_id: &str) -> Result<Transaction, String> {
        let mut txs = self.transactions.write().map_err(|e| e.to_string())?;

        if let Some(tx) = txs.get_mut(tx_id) {
            if tx.status != TransactionStatus::Confirmed {
                return Err("Transaction not confirmed".to_string());
            }

            // Simulate execution
            tx.status = TransactionStatus::Executed;
            tx.executed_at = Some(Utc::now());
            tx.tx_hash = Some(generate_tx_hash());

            // Update wallet balance
            let mut wallets = self.wallets.write().map_err(|e| e.to_string())?;
            if let Some(wallet) = wallets.get_mut(&tx.wallet_id) {
                let value: f64 = tx.value.parse().unwrap_or(0.0);
                let balance: f64 = wallet.balance.parse().unwrap_or(0.0);
                wallet.balance = (balance - value).to_string();
            }

            Ok(tx.clone())
        } else {
            Err("Transaction not found".to_string())
        }
    }

    pub fn cancel_transaction(&self, tx_id: &str, owner_address: &str) -> Result<(), String> {
        let mut pending = self.pending_txs.write().map_err(|e| e.to_string())?;
        let mut txs = self.transactions.write().map_err(|e| e.to_string())?;

        if pending.contains_key(tx_id) {
            pending.remove(tx_id);
        }

        if let Some(tx) = txs.get_mut(tx_id) {
            tx.status = TransactionStatus::Cancelled;
            Ok(())
        } else {
            Err("Transaction not found".to_string())
        }
    }

    pub fn get_transaction(&self, tx_id: &str) -> Result<Option<Transaction>, String> {
        let txs = self.transactions.read().map_err(|e| e.to_string())?;
        Ok(txs.get(tx_id).cloned())
    }

    pub fn get_wallet_transactions(&self, wallet_id: &str) -> Result<Vec<Transaction>, String> {
        let txs = self.transactions.read().map_err(|e| e.to_string())?;
        let result: Vec<Transaction> = txs
            .values()
            .filter(|t| t.wallet_id == wallet_id)
            .cloned()
            .collect();
        Ok(result)
    }

    pub fn get_pending_transactions(&self, wallet_id: &str) -> Result<Vec<PendingTransaction>, String> {
        let pending = self.pending_txs.read().map_err(|e| e.to_string())?;
        let result: Vec<PendingTransaction> = pending
            .values()
            .filter(|p| p.wallet_id == wallet_id)
            .cloned()
            .collect();
        Ok(result)
    }

    // ========================================================================
    // Helper Functions
    // ========================================================================

    fn generate_wallet_address(&self, wallet_id: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(wallet_id.as_bytes());
        format!("0x{:x}", hasher.finalize())[..42].to_string()
    }
}

// ============================================================================
// Main
// ============================================================================

#[tokio::main]
async fn main() {
    env_logger::init();

    let service = MultiSigService::new();

    // Create multi-sig wallet
    let owners = vec![
        "0x742d35Cc6634C0532925a3b844Bc9e7595f".to_string(),
        "0x1234567890abcdef1234567890abcdef12345678".to_string(),
        "0xabcdef1234567890abcdef1234567890abcdef12".to_string(),
    ];

    let wallet = service.create_wallet(
        "Team Wallet".to_string(),
        owners.clone(),
        2,
        MultiSigType::MultiSig,
        1, // Ethereum
    ).unwrap();

    println!("Created wallet: {} at {}", wallet.wallet_id, wallet.address);

    // Create transaction
    let tx = service.create_transaction(
        &wallet.wallet_id,
        "0xrecipient1234567890abcdef1234567890abcd".to_string(),
        "1.5".to_string(),
        "0x".to_string(),
    ).unwrap();

    println!("Created transaction: {}", tx.tx_id);

    // First confirmation
    let tx = service.confirm_transaction(
        &tx.tx_id,
        &owners[0],
        "signature1".to_string(),
    ).unwrap();

    println!("Confirmations: {}/{}", tx.confirmations.len(), tx.required_confirmations);

    // Second confirmation
    let tx = service.confirm_transaction(
        &tx.tx_id,
        &owners[1],
        "signature2".to_string(),
    ).unwrap();

    println!("Confirmations: {}/{}", tx.confirmations.len(), tx.required_confirmations);
    println!("Status: {:?}", tx.status);

    // Execute
    let tx = service.execute_transaction(&tx.tx_id).unwrap();
    println!("Executed! Tx hash: {}", tx.tx_hash.unwrap());

    // List wallets
    let wallets = service.list_wallets().unwrap();
    println!("\nTotal wallets: {}", wallets.len());
}

fn generate_tx_hash() -> String {
    use uuid::Uuid;
    let mut hasher = Sha256::new();
    hasher.update(Uuid::new_v4().to_string().as_bytes());
    hasher.update(Utc::now().timestamp_nanos_opt().unwrap_or(0).to_string().as_bytes());
    format!("0x{:x}", hasher.finalize())
}
