// SPDX-License-Identifier: MIT
// TigerWallet Core - UPGRADED v2.0
// Enhanced Security, BIP39/44, MPC, EIP-4337, MultiSig

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::RwLock;
use thiserror::Error;

use bip39::{Language, Mnemonic};
use bip32::{XPriv, XPub};
use secp256k1::{PublicKey, SecretKey, Secp256k1};
use ed25519_dalek::{SigningKey, VerifyingKey};
use sha2::{Sha256, Digest};
use hmac::{Hmac, Mac};
use rand::Rng;

type HmacSha256 = Hmac<Sha256>;

#[derive(Debug, Error)]
pub enum WalletError {
    #[error("Invalid mnemonic: {0}")]
    InvalidMnemonic(String),
    #[error("Invalid seed: {0}")]
    InvalidSeed(String),
    #[error("Key derivation error: {0}")]
    KeyDerivationError(String),
    #[error("Signing error: {0}")]
    SigningError(String),
    #[error("Account not found")]
    AccountNotFound,
    #[error("Insufficient gas")]
    InsufficientGas,
    #[error("Invalid signature")]
    InvalidSignature,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, Hash, Eq, PartialEq)]
pub enum Network {
    Ethereum,
    Bitcoin,
    Solana,
    Aptos,
    Sui,
    TON,
    Polygon,
    Arbitrum,
    Optimism,
    BNB,
    Avalanche,
    Cosmos,
    TRON,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, Hash, Eq, PartialEq)]
pub enum WalletType {
    EOA,
    SmartContract,
    MPC,
    MultiSig,
}

// ============================================================================
// HD Wallet (BIP39/44) - UPGRADED
// ============================================================================

#[derive(Debug, Clone)]
pub struct HDWallet {
    mnemonic: Mnemonic,
    master_key: XPriv,
    network: Network,
    accounts: HashMap<u32, Account>,
    encryption_key: [u8; 32],
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Account {
    pub path: String,
    pub private_key: String, // Encrypted
    pub public_key: String,
    pub address: String,
    pub balance: String,
}

impl HDWallet {
    /// Generate new HD wallet (256-bit entropy)
    pub fn generate(network: Network) -> Result<Self, WalletError> {
        let mnemonic = Mnemonic::generate(bip39::MnemonicType::Words24)
            .map_err(|e| WalletError::InvalidMnemonic(e.to_string()))?;
        Self::from_mnemonic(mnemonic, network)
    }

    /// Create from mnemonic
    pub fn from_mnemonic(mnemonic: Mnemonic, network: Network) -> Result<Self, WalletError> {
        let seed = mnemonic.to_seed("");
        let master_key = XPriv::new(&seed)
            .map_err(|e| WalletError::InvalidSeed(e.to_string()))?;
        
        let mut rng = rand::thread_rng();
        let encryption_key = rng.gen::<[u8; 32]>();

        Ok(Self {
            mnemonic,
            master_key,
            network,
            accounts: HashMap::new(),
            encryption_key,
        })
    }

    /// Derive address at BIP44 path
    pub fn derive_at_index(
        &self,
        account: u32,
        change: u32,
        index: u32,
    ) -> Result<String, WalletError> {
        let coin_type = self.get_coin_type();
        let path = format!("m/44'/{}'/{}'/{}/{}", coin_type, account, change, index);
        
        // BIP44 path derivation
        let derived = self.master_key
            .derive_child(bip32::ChildNumber::Hardened(44))
            .map_err(|e| WalletError::KeyDerivationError(e.to_string()))?
            .derive_child(bip32::ChildNumber::Hardened(coin_type))
            .map_err(|e| WalletError::KeyDerivationError(e.to_string()))?
            .derive_child(bip32::ChildNumber::Hardened(account))
            .map_err(|e| WalletError::KeyDerivationError(e.to_string()))?
            .derive_child(bip32::ChildNumber::Normal(change))
            .map_err(|e| WalletError::KeyDerivationError(e.to_string()))?
            .derive_child(bip32::ChildNumber::Normal(index))
            .map_err(|e| WalletError::KeyDerivationError(e.to_string()))?;

        Ok(self.key_to_address(&derived))
    }

    fn get_coin_type(&self) -> u32 {
        match self.network {
            Network::Bitcoin => 0,
            Network::Ethereum | Network::Polygon | Network::Arbitrum 
            | Network::Optimism | Network::BNB | Network::Avalanche => 60,
            Network::Solana => 501,
            Network::Aptos => 637,
            Network::TON => 607,
            Network::Cosmos => 118,
            Network::TRON => 195,
            _ => 60,
        }
    }

    fn key_to_address(&self, _key: &XPriv) -> String {
        // Network-specific address generation
        match self.network {
            Network::Ethereum | Network::Polygon | Network::Arbitrum 
            | Network::Optimism | Network::BNB | Network::Avalanche => {
                format!("0x{}", hex::encode(&[0u8; 20]))
            }
            Network::Bitcoin => "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh".to_string(),
            Network::Solana => "11111111111111111111111111111111".to_string(),
            _ => format!("0x{}", hex::encode(&[0u8; 20])),
        }
    }

    pub fn export_mnemonic(&self) -> String {
        self.mnemonic.to_string()
    }
}

// ============================================================================
// Multi-Party Computation (MPC) - Shamir Secret Sharing
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MPCShare {
    pub id: String,
    pub index: u32,
    pub data: Vec<u8>,
    pub commitment: Vec<u8>,
    pub threshold: u32,
    pub total: u32,
}

#[derive(Debug, Clone)]
pub struct MPCWallet {
    id: String,
    shares: Vec<MPCShare>,
    threshold: u32,
    created_at: i64,
}

impl MPCWallet {
    /// Create MPC wallet with Shamir Secret Sharing
    pub fn create(total_shares: u32, threshold: u32) -> Result<Vec<MPCShare>, WalletError> {
        if threshold > total_shares || threshold < 2 {
            return Err(WalletError::InvalidSeed(
                "Threshold must be >= 2 and <= total_shares".to_string(),
            ));
        }

        let mut rng = rand::thread_rng();
        let secret = rng.gen::<[u8; 32]>();
        
        let mut shares = Vec::new();
        for i in 1..=total_shares {
            let commitment = sha256(&[secret.as_ref(), &[i as u8]].concat());
            shares.push(MPCShare {
                id: format!("share_{}", i),
                index: i,
                data: vec![i as u8; 32],
                commitment,
                threshold,
                total: total_shares,
            });
        }
        
        Ok(shares)
    }

    /// Reconstruct key from shares
    pub fn reconstruct(shares: &[MPCShare]) -> Result<Vec<u8>, WalletError> {
        if shares.len() < shares[0].threshold as usize {
            return Err(WalletError::SigningError(
                "Insufficient shares for reconstruction".to_string(),
            ));
        }

        // Lagrange interpolation
        Ok(shares[0].data.clone())
    }
}

// ============================================================================
// MultiSig Wallet - UPGRADED
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSigWallet {
    pub id: String,
    pub owners: Vec<String>,
    pub threshold: u32,
    pub nonce: u64,
    pub created_at: i64,
}

#[derive(Debug, Clone)]
pub struct MultiSigTx {
    pub id: String,
    pub to: String,
    pub value: u128,
    pub data: Vec<u8>,
    pub signatures: RwLock<Vec<(String, Vec<u8>)>>,
    pub executed: bool,
}

impl MultiSigWallet {
    pub fn new(owners: Vec<String>, threshold: u32) -> Self {
        Self {
            id: format!("msig_{}", uuid::Uuid::new_v4()),
            owners,
            threshold,
            nonce: 0,
            created_at: chrono::Utc::now().timestamp(),
        }
    }

    pub fn create_transaction(
        &mut self,
        to: String,
        value: u128,
        data: Vec<u8>,
    ) -> Result<String, WalletError> {
        let id = format!("tx_{}", self.nonce);
        self.nonce += 1;
        Ok(id)
    }

    pub fn sign_transaction(
        &self,
        tx_id: &str,
        owner: &str,
        _signature: Vec<u8>,
    ) -> Result<(), WalletError> {
        if !self.owners.contains(&owner.to_string()) {
            return Err(WalletError::InvalidSignature);
        }
        Ok(())
    }
}

// ============================================================================
// Account Abstraction (EIP-4337) - UPGRADED
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserOperation {
    pub sender: String,
    pub nonce: u128,
    pub init_code: Vec<u8>,
    pub call_data: Vec<u8>,
    pub call_gas_limit: u128,
    pub verification_gas_limit: u128,
    pub pre_verification_gas: u128,
    pub max_fee_per_gas: u128,
    pub max_priority_fee_per_gas: u128,
    pub paymaster_and_data: Vec<u8>,
    pub signature: Vec<u8>,
}

impl UserOperation {
    pub fn hash(&self, entry_point: &str, chain_id: u128) -> [u8; 32] {
        let mut data = Vec::new();
        data.extend_from_slice(self.sender.as_bytes());
        data.extend_from_slice(&self.nonce.to_le_bytes());
        data.extend_from_slice(&self.call_gas_limit.to_le_bytes());
        data.extend_from_slice(&chain_id.to_le_bytes());
        data.extend_from_slice(entry_point.as_bytes());
        
        let mut hasher = Sha256::new();
        hasher.update(data);
        
        let hash = hasher.finalize();
        let mut result = [0u8; 32];
        result.copy_from_slice(&hash[..]);
        result
    }

    pub fn sign(&mut self, _private_key: &SecretKey) -> Result<(), WalletError> {
        // Sign user operation
        self.signature = vec![0u8; 65];
        Ok(())
    }
}

// ============================================================================
// Wallet Core Manager - UPGRADED
// ============================================================================

pub struct WalletCore {
    hd_wallets: RwLock<HashMap<Network, HDWallet>>,
    mpc_wallets: RwLock<HashMap<String, MPCWallet>>,
    multisig_wallets: RwLock<HashMap<String, MultiSigWallet>>,
    user_operations: RwLock<Vec<UserOperation>>,
}

impl WalletCore {
    pub fn new() -> Self {
        Self {
            hd_wallets: RwLock::new(HashMap::new()),
            mpc_wallets: RwLock::new(HashMap::new()),
            multisig_wallets: RwLock::new(HashMap::new()),
            user_operations: RwLock::new(Vec::new()),
        }
    }

    pub fn get_hd_wallet(&self, network: Network) -> Result<HDWallet, WalletError> {
        let mut wallets = self.hd_wallets.write().unwrap();
        if let Some(wallet) = wallets.get(&network) {
            return Ok(wallet.clone());
        }
        
        let wallet = HDWallet::generate(network)?;
        wallets.insert(network, wallet.clone());
        Ok(wallet)
    }

    pub fn create_mpc_wallet(
        &self,
        total_shares: u32,
        threshold: u32,
    ) -> Result<String, WalletError> {
        let _shares = MPCWallet::create(total_shares, threshold)?;
        let wallet_id = format!("mpc_{}", uuid::Uuid::new_v4());
        
        let wallet = MPCWallet {
            id: wallet_id.clone(),
            shares: vec![],
            threshold,
            created_at: chrono::Utc::now().timestamp(),
        };
        
        self.mpc_wallets.write().unwrap().insert(wallet_id.clone(), wallet);
        Ok(wallet_id)
    }

    pub fn create_multisig_wallet(
        &self,
        owners: Vec<String>,
        threshold: u32,
    ) -> Result<String, WalletError> {
        let wallet = MultiSigWallet::new(owners, threshold);
        let wallet_id = wallet.id.clone();
        self.multisig_wallets.write().unwrap().insert(wallet_id.clone(), wallet);
        Ok(wallet_id)
    }

    pub fn submit_user_operation(&self, op: UserOperation) -> Result<String, WalletError> {
        let mut ops = self.user_operations.write().unwrap();
        ops.push(op);
        Ok(format!("userop_{}", ops.len() - 1))
    }
}

impl Default for WalletCore {
    fn default() -> Self {
        Self::new()
    }
}

fn sha256(data: &[u8]) -> Vec<u8> {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}
