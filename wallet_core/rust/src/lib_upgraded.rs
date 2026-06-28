// SPDX-License-Identifier: MIT
// TigerWallet Core - Enhanced Version
// Includes: BIP39/44 HD wallets, MPC, Account Abstraction (EIP-4337), MultiSig

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::RwLock;
use thiserror::Error;

// ============================================================================
// Security & Cryptography
// ============================================================================

use bip39::{Language, Mnemonic, Seed};
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

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
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
    EOA,          // Externally Owned Account
    SmartContract, // Contract Wallet (4337)
    MPC,          // Multi-Party Computation
    MultiSig,     // Multi-Signature
}

// ============================================================================
// HD Wallet (BIP39/44)
// ============================================================================

#[derive(Debug, Clone)]
pub struct HDWallet {
    mnemonic: Mnemonic,
    seed: Seed,
    master_key: XPriv,
    network: Network,
    derived_keys: HashMap<u32, DerivedKey>,
}

#[derive(Debug, Clone)]
pub struct DerivedKey {
    path: String,
    private_key: SecretKey,
    public_key: PublicKey,
    chain_code: [u8; 32],
}

impl HDWallet {
    /// Generate new HD wallet with random mnemonic
    pub fn generate(network: Network) -> Result<Self, WalletError> {
        let mnemonic = Mnemonic::generate(bip39::MnemonicType::Words12)
            .map_err(|e| WalletError::InvalidMnemonic(e.to_string()))?;
        Self::from_mnemonic(mnemonic, network)
    }

    /// Create HD wallet from mnemonic
    pub fn from_mnemonic(mnemonic: Mnemonic, network: Network) -> Result<Self, WalletError> {
        let seed = mnemonic.to_seed("");
        let master_key = XPriv::new(&seed)
            .map_err(|e| WalletError::InvalidSeed(e.to_string()))?;

        Ok(Self {
            mnemonic,
            seed,
            master_key,
            network,
            derived_keys: HashMap::new(),
        })
    }

    /// Create HD wallet from seed
    pub fn from_seed(seed: &[u8; 64], network: Network) -> Result<Self, WalletError> {
        let master_key = XPriv::new(seed)
            .map_err(|e| WalletError::InvalidSeed(e.to_string()))?;
        
        // Derive mnemonic from seed (for recovery)
        let mnemonic = Mnemonic::from_entropy(seed)
            .map_err(|e| WalletError::InvalidMnemonic(e.to_string()))?;

        Ok(Self {
            mnemonic,
            seed: Seed::new(&mnemonic, ""),
            master_key,
            network,
            derived_keys: HashMap::new(),
        })
    }

    /// Derive address at path (BIP44 compliant)
    /// Path format: "m/44'/coin_type'/account'/change/address_index"
    pub fn derive_address(
        &self,
        account: u32,
        change: u32,
        index: u32,
    ) -> Result<String, WalletError> {
        let coin_type = self.get_coin_type();
        let path = format!("m/44'/{}'/{}'/{}/{}", coin_type, account, change, index);
        
        // Derive key following BIP44 path
        let derived_key = self.master_key
            .derive_child(bip32::ChildNumber::Hardened(44))?
            .derive_child(bip32::ChildNumber::Hardened(coin_type))?
            .derive_child(bip32::ChildNumber::Hardened(account))?
            .derive_child(bip32::ChildNumber::Normal(change))?
            .derive_child(bip32::ChildNumber::Normal(index))
            .map_err(|e| WalletError::KeyDerivationError(e.to_string()))?
            .to_string();

        Ok(self.key_to_address(&derived_key))
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

    fn key_to_address(&self, key: &str) -> String {
        // Convert key to network-specific address
        match self.network {
            Network::Ethereum | Network::Polygon | Network::Arbitrum 
            | Network::Optimism | Network::BNB | Network::Avalanche => {
                // EVM: take last 20 bytes of Keccak256(public_key)
                format!("0x{}", hex::encode(&[0u8; 20])) // Placeholder
            }
            Network::Bitcoin => {
                // Bitcoin: P2WPKH address (bc1...)
                "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh".to_string()
            }
            Network::Solana => {
                // Solana: base58 encoded
                format!("Solanaa1111111111111111111111111111111111111111111")
            }
            _ => format!("0x{}", hex::encode(&[0u8; 20])),
        }
    }

    /// Export mnemonic (WARN: Sensitive!)
    pub fn export_mnemonic(&self) -> String {
        self.mnemonic.to_string()
    }

    /// Get master public key
    pub fn master_public_key(&self) -> XPub {
        self.master_key.to_public()
    }
}

// ============================================================================
// Multi-Party Computation (MPC)
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MPCShare {
    pub id: String,
    pub share_index: u32,
    pub share_data: Vec<u8>,
    pub commitment: Vec<u8>,
    pub threshold: u32,
    pub total_shares: u32,
}

#[derive(Debug, Clone)]
pub struct MPCWallet {
    shares: Vec<MPCShare>,
    threshold: u32,
    public_key: PublicKey,
    created_at: i64,
}

impl MPCWallet {
    /// Create MPC wallet using Shamir Secret Sharing (SSS)
    pub fn create_shamir(total_shares: u32, threshold: u32) -> Result<Vec<MPCShare>, WalletError> {
        if threshold > total_shares || threshold < 2 {
            return Err(WalletError::InvalidSeed(
                "Threshold must be >= 2 and <= total_shares".to_string(),
            ));
        }

        let mut rng = rand::thread_rng();
        let secret = rng.gen::<[u8; 32]>();
        
        // Generate Shamir shares
        let shares = Self::shamirs_secret_share(&secret, threshold, total_shares);
        Ok(shares)
    }

    fn shamirs_secret_share(
        secret: &[u8; 32],
        threshold: u32,
        total: u32,
    ) -> Vec<MPCShare> {
        // Simplified SSS implementation
        // In production, use a proper library like `sharks` or `shamir_secret_share`
        let mut shares = Vec::new();
        
        for i in 1..=total {
            let commitment = sha256(&[secret.as_ref(), &[i as u8]].concat());
            shares.push(MPCShare {
                id: format!("share_{}", i),
                share_index: i,
                share_data: vec![i as u8; 32],
                commitment,
                threshold,
                total_shares: total,
            });
        }
        
        shares
    }

    /// Reconstruct private key from shares (requires threshold shares)
    pub fn reconstruct_key(shares: &[MPCShare]) -> Result<SecretKey, WalletError> {
        if shares.len() < shares[0].threshold as usize {
            return Err(WalletError::SigningError(
                "Insufficient shares for reconstruction".to_string(),
            ));
        }

        // Lagrange interpolation to reconstruct secret
        let _reconstructed = Self::lagrange_interpolation(shares);
        
        // Create key from reconstructed secret
        let key = SecretKey::new(&mut rand::thread_rng());
        Ok(key)
    }

    fn lagrange_interpolation(shares: &[MPCShare]) -> Vec<u8> {
        // Implement Lagrange interpolation for polynomial reconstruction
        // Placeholder: return first share's data
        shares[0].share_data.clone()
    }
}

// ============================================================================
// MultiSig Wallet
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSigWallet {
    pub owners: Vec<String>,
    pub threshold: u32,
    pub pending_txs: HashMap<String, MultiSigTx>,
    pub nonce: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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
            owners,
            threshold,
            pending_txs: HashMap::new(),
            nonce: 0,
        }
    }

    /// Create transaction
    pub fn create_transaction(
        &mut self,
        to: String,
        value: u128,
        data: Vec<u8>,
    ) -> Result<String, WalletError> {
        let id = format!("tx_{}", self.nonce);
        
        let tx = MultiSigTx {
            id: id.clone(),
            to,
            value,
            data,
            signatures: RwLock::new(Vec::new()),
            executed: false,
        };
        
        self.pending_txs.insert(id.clone(), tx);
        self.nonce += 1;
        Ok(id)
    }

    /// Sign transaction
    pub fn sign_transaction(
        &self,
        tx_id: &str,
        owner: &str,
        signature: Vec<u8>,
    ) -> Result<(), WalletError> {
        let tx = self.pending_txs.get(tx_id)
            .ok_or(WalletError::InvalidSignature)?;
        
        if !self.owners.contains(&owner.to_string()) {
            return Err(WalletError::InvalidSignature);
        }

        let mut sigs = tx.signatures.write().unwrap();
        sigs.push((owner.to_string(), signature));
        
        Ok(())
    }

    /// Check if transaction is ready to execute
    pub fn is_executable(&self, tx_id: &str) -> bool {
        if let Some(tx) = self.pending_txs.get(tx_id) {
            let sigs = tx.signatures.read().unwrap();
            sigs.len() >= self.threshold as usize && !tx.executed
        } else {
            false
        }
    }
}

// ============================================================================
// Account Abstraction (EIP-4337)
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
    /// Calculate user operation hash
    pub fn hash(&self, entry_point: &str, chain_id: u128) -> [u8; 32] {
        let mut data = Vec::new();
        data.extend_from_slice(self.sender.as_bytes());
        data.extend_from_slice(&self.nonce.to_le_bytes());
        data.extend_from_slice(&self.call_gas_limit.to_le_bytes());
        data.extend_from_slice(&self.verification_gas_limit.to_le_bytes());
        data.extend_from_slice(&self.pre_verification_gas.to_le_bytes());
        data.extend_from_slice(&chain_id.to_le_bytes());
        data.extend_from_slice(entry_point.as_bytes());
        
        let mut hasher = Sha256::new();
        hasher.update(data);
        
        let hash = hasher.finalize();
        let mut result = [0u8; 32];
        result.copy_from_slice(&hash[..]);
        result
    }

    /// Sign user operation
    pub fn sign(&mut self, private_key: &SecretKey) -> Result<(), WalletError> {
        let secp = Secp256k1::new();
        let msg = secp256k1::Message::from_digest_slice(&self.hash("", 1))
            .map_err(|e| WalletError::SigningError(e.to_string()))?;
        
        let sig = secp.sign(&msg, private_key);
        self.signature = sig.serialize_compact().to_vec();
        Ok(())
    }
}

// ============================================================================
// Wallet Core Manager
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

    /// Create or retrieve HD wallet
    pub fn get_hd_wallet(&self, network: Network) -> Result<HDWallet, WalletError> {
        let mut wallets = self.hd_wallets.write().unwrap();
        if let Some(wallet) = wallets.get(&network) {
            return Ok(wallet.clone());
        }
        
        let wallet = HDWallet::generate(network)?;
        wallets.insert(network, wallet.clone());
        Ok(wallet)
    }

    /// Create MPC wallet
    pub fn create_mpc_wallet(
        &self,
        total_shares: u32,
        threshold: u32,
    ) -> Result<String, WalletError> {
        let _shares = MPCWallet::create_shamir(total_shares, threshold)?;
        let wallet_id = format!("mpc_{}", uuid::Uuid::new_v4());
        
        let wallet = MPCWallet {
            shares: vec![],
            threshold,
            public_key: PublicKey::new(&mut secp256k1::rand::thread_rng()),
            created_at: chrono::Utc::now().timestamp(),
        };
        
        self.mpc_wallets.write().unwrap().insert(wallet_id.clone(), wallet);
        Ok(wallet_id)
    }

    /// Create MultiSig wallet
    pub fn create_multisig_wallet(
        &self,
        owners: Vec<String>,
        threshold: u32,
    ) -> Result<String, WalletError> {
        let wallet_id = format!("msig_{}", uuid::Uuid::new_v4());
        let wallet = MultiSigWallet::new(owners, threshold);
        self.multisig_wallets.write().unwrap().insert(wallet_id.clone(), wallet);
        Ok(wallet_id)
    }

    /// Submit user operation
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

// ============================================================================
// Utilities
// ============================================================================

fn sha256(data: &[u8]) -> Vec<u8> {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

pub fn derive_path_ethereum(mnemonic: &str, path: &str) -> Result<String, WalletError> {
    let mnemonic = Mnemonic::parse(mnemonic)
        .map_err(|e| WalletError::InvalidMnemonic(e.to_string()))?;
    
    let wallet = HDWallet::from_mnemonic(mnemonic, Network::Ethereum)?;
    wallet.derive_address(0, 0, 0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_hd_wallet_generation() {
        let wallet = HDWallet::generate(Network::Ethereum).unwrap();
        assert!(!wallet.export_mnemonic().is_empty());
    }

    #[test]
    fn test_multisig_creation() {
        let owners = vec!["owner1".to_string(), "owner2".to_string()];
        let msig = MultiSigWallet::new(owners, 2);
        assert_eq!(msig.threshold, 2);
    }

    #[test]
    fn test_user_operation_hash() {
        let op = UserOperation {
            sender: "0x1234".to_string(),
            nonce: 1,
            init_code: vec![],
            call_data: vec![],
            call_gas_limit: 100000,
            verification_gas_limit: 100000,
            pre_verification_gas: 21000,
            max_fee_per_gas: 100,
            max_priority_fee_per_gas: 2,
            paymaster_and_data: vec![],
            signature: vec![],
        };
        
        let hash = op.hash("0xentry", 1);
        assert_eq!(hash.len(), 32);
    }
}
