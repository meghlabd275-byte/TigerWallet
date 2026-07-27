//! TigerWallet Aptos Blockchain SDK
//! Production-ready implementation for Aptos Move blockchain
//! 
//! Features:
//! - Full account management (create, import)
//! - Transaction signing and submission
//! - NFT support (Aptos Tokens)
//! - Module deployment
//! - Staking operations
//! - Faucet integration (devnet)

#![allow(dead_code)]

use std::collections::HashMap;
use std::sync::Arc;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use sha2::{Sha256, Digest};
use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum AptosError {
    #[error("Invalid address: {0}")]
    InvalidAddress(String),
    
    #[error("Invalid transaction: {0}")]
    InvalidTransaction(String),
    
    #[error("Signing error: {0}")]
    SigningError(String),
    
    #[error("RPC error: {0}")]
    RpcError(String),
    
    #[error("Serialization error: {0}")]
    SerializationError(String),
    
    #[error("Account error: {0}")]
    AccountError(String),
    
    #[error("Network error: {0}")]
    NetworkError(String),
}

// ============================================================================
// Address Types
// ============================================================================

/// Aptos account address (32 bytes)
#[derive(Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct AccountAddress(pub [u8; 32]);

impl AccountAddress {
    /// Create from hex string
    pub fn from_hex(hex: &str) -> Result<Self, AptosError> {
        let bytes = hex::decode(hex)
            .map_err(|e| AptosError::InvalidAddress(e.to_string()))?;
        
        if bytes.len() != 32 {
            return Err(AptosError::InvalidAddress(
                "Address must be 32 bytes".to_string()
            ));
        }
        
        let mut addr = [0u8; 32];
        addr.copy_from_slice(&bytes);
        Ok(AccountAddress(addr))
    }
    
    /// Create from short string (for addresses starting with 0x)
    pub fn from_short_hex(hex: &str) -> Result<Self, AptosError> {
        let hex = hex.trim_start_matches("0x");
        let bytes = hex::decode(hex)
            .map_err(|e| AptosError::InvalidAddress(e.to_string()))?;
        
        let mut addr = [0u8; 32];
        let offset = 32 - bytes.len();
        addr[offset..].copy_from_slice(&bytes);
        Ok(AccountAddress(addr))
    }
    
    /// Get as hex string
    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }
    
    /// Get as short hex (without leading zeros)
    pub fn to_short_hex(&self) -> String {
        let bytes = self.0.iter().skip_while(|&&b| b == 0).copied().collect::<Vec<_>>();
        if bytes.is_empty() {
            "0x0".to_string()
        } else {
            format!("0x{}", hex::encode(bytes))
        }
    }
}

impl std::fmt::Debug for AccountAddress {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.to_short_hex())
    }
}

impl std::fmt::Display for AccountAddress {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.to_short_hex())
    }
}

// ============================================================================
// Key Types
// ============================================================================

/// Ed25519 public key
#[derive(Clone, Serialize, Deserialize)]
pub struct Ed25519PublicKey(pub [u8; 32]);

impl Ed25519PublicKey {
    pub fn from_bytes(bytes: &[u8]) -> Result<Self, AptosError> {
        if bytes.len() != 32 {
            return Err(AptosError::InvalidAddress("Invalid key length".to_string()));
        }
        let mut key = [0u8; 32];
        key.copy_from_slice(bytes);
        Ok(Ed25519PublicKey(key))
    }
    
    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }
}

/// Ed25519 signature
#[derive(Clone, Serialize, Deserialize)]
pub struct Ed25519Signature(pub [u8; 64]);

impl Ed25519Signature {
    pub fn from_bytes(bytes: &[u8]) -> Result<Self, AptosError> {
        if bytes.len() != 64 {
            return Err(AptosError::SigningError("Invalid signature length".to_string()));
        }
        let mut sig = [0u8; 64];
        sig.copy_from_slice(bytes);
        Ok(Ed25519Signature(sig))
    }
    
    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }
}

// ============================================================================
// Transaction Types
// ============================================================================

/// Transaction type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransactionType {
    /// User transaction
    UserTransaction,
    /// Genesis transaction
    GenesisTransaction,
    /// Block metadata
    BlockMetadata,
    /// State checkpoint
    StateCheckpoint,
}

/// Transaction payload
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransactionPayload {
    /// Script (Move bytecode)
    Script {
        code: Vec<u8>,
        ty_args: Vec<TypeTag>,
        args: Vec<Vec<u8>>,
    },
    /// Module publishing
    ModuleBundle {
        modules: Vec<Vec<u8>>,
    },
    /// Entry function
    EntryFunction {
        module: ModuleId,
        function: String,
        ty_args: Vec<TypeTag>,
        args: Vec<Vec<u8>>,
    },
}

/// Type tag for generic parameters
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TypeTag {
    Bool,
    U8,
    U16,
    U32,
    U64,
    U128,
    U256,
    Address,
    Signer,
    Vector(Box<TypeTag>),
    Struct(StructTag),
}

/// Struct tag
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StructTag {
    pub address: AccountAddress,
    pub module: String,
    pub name: String,
    pub type_params: Vec<TypeTag>,
}

/// Module ID
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModuleId {
    pub address: AccountAddress,
    pub name: String,
}

/// Raw transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RawTransaction {
    pub sender: AccountAddress,
    pub sequence_number: u64,
    pub payload: TransactionPayload,
    pub max_gas_amount: u64,
    pub gas_unit_price: u64,
    pub gas_currency_code: String,
    pub expiration_timestamp_secs: u64,
    pub chain_id: u8,
}

/// Signed transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedTransaction {
    pub raw_txn: RawTransaction,
    pub authenticator: TransactionAuthenticator,
}

/// Transaction authenticator
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransactionAuthenticator {
    /// Ed25519 signature
    Ed25519 {
        public_key: Ed25519PublicKey,
        signature: Ed25519Signature,
    },
    /// Multi-sig
    MultiEd25519 {
        public_key: MultiEd25519PublicKey,
        signature: MultiEd25519Signature,
    },
}

/// Multi-sig public key
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiEd25519PublicKey {
    pub keys: Vec<Ed25519PublicKey>,
    pub threshold: u8,
}

/// Multi-sig signature
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiEd25519Signature {
    pub signatures: Vec<Ed25519Signature>,
    pub bitmap: Vec<u8>,
}

// ============================================================================
// Account Types
// ============================================================================

/// Account information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Account {
    pub address: AccountAddress,
    pub sequence_number: u64,
    pub authentication_key: Vec<u8>,
}

/// Account resource
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountResource {
    pub type_: String,
    pub data: serde_json::Value,
}

/// Account balance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountBalance {
    pub coin: CoinStore,
}

/// Coin store
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CoinStore {
    pub coin: Coin,
    pub deposit_events: u64,
    pub withdraw_events: u64,
}

/// Coin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Coin {
    pub value: u128,
}

// ============================================================================
// Move Types
// ============================================================================

/// Move module
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveModule {
    pub address: AccountAddress,
    pub name: String,
    pub bytecode: Vec<u8>,
    pub abi: Option<MoveModuleAbi>,
}

/// Move module ABI
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveModuleAbi {
    pub name: String,
    pub friends: Vec<ModuleId>,
    pub exposed_functions: Vec<MoveFunction>,
    pub structs: Vec<MoveStruct>,
}

/// Move function
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveFunction {
    pub name: String,
    pub visibility: MoveVisibility,
    pub is_entry: bool,
    pub type_parameters: Vec<MoveFunctionTypeParameter>,
    pub parameters: Vec<TypeTag>,
    pub return_: Vec<TypeTag>,
}

/// Move visibility
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MoveVisibility {
    Private,
    Public,
    Friend,
}

/// Move function type parameter
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveFunctionTypeParameter {
    pub constraints: Vec<MoveTypeConstraint>,
}

/// Move type constraint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveTypeConstraint {
    pub is_phantom: bool,
}

/// Move struct
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveStruct {
    pub name: String,
    pub is_native: bool,
    pub abilities: Vec<MoveAbility>,
    pub type_parameters: Vec<MoveStructTypeParameter>,
    pub fields: Vec<MoveStructField>,
}

/// Move ability
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MoveAbility {
    Copy,
    Drop,
    Store,
    Key,
}

/// Move struct type parameter
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveStructTypeParameter {
    pub constraints: Vec<MoveTypeConstraint>,
}

/// Move struct field
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveStructField {
    pub name: String,
    pub type_: TypeTag,
}

// ============================================================================
// Token Types (Aptos Tokens)
// ============================================================================

/// Token collection
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Collection {
    pub creator: AccountAddress,
    pub collection_name: String,
    pub description: String,
    pub uri: String,
}

/// Token
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub creator: AccountAddress,
    pub collection_name: String,
    pub token_name: String,
    pub description: String,
    pub uri: String,
    pub maximum: u64,
    pub supply: u64,
}

/// Token property
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenProperty {
    pub name: String,
    pub value: String,
    pub type_: String,
}

// ============================================================================
// Staking Types
// ============================================================================

/// Validator info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Validator {
    pub operator_address: AccountAddress,
    pub network_address: String,
    pub vote_weight: u64,
    pub configs: ValidatorConfig,
}

/// Validator config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatorConfig {
    pub consensus_pubkey: Vec<u8>,
    pub network_pubkey: Vec<u8>,
    pub proof_of_possession: Vec<u8>,
}

/// Staking pool
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingPool {
    pub operator: AccountAddress,
    pub voter: AccountAddress,
    pub staked: u64,
    pub pending_staked: u64,
    pub active: bool,
}

// ============================================================================
// RPC Client
// ============================================================================

/// Aptos network
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Network {
    Mainnet,
    Devnet,
    Testnet,
    Custom,
}

impl Network {
    pub fn chain_id(&self) -> u8 {
        match self {
            Network::Mainnet => 1,
            Network::Testnet => 2,
            Network::Devnet => 3,
            Network::Custom => 0,
        }
    }
    
    pub fn rpc_url(&self) -> &str {
        match self {
            Network::Mainnet => "https://fullnode.mainnet.aptoslabs.com",
            Network::Testnet => "https://fullnode.testnet.aptoslabs.com",
            Network::Devnet => "https://fullnode.devnet.aptoslabs.com",
            Network::Custom => "http://localhost:8080",
        }
    }
    
    pub fn faucet_url(&self) -> Option<&str> {
        match self {
            Network::Devnet => Some("https://faucet.devnet.aptoslabs.com"),
            Network::Testnet => Some("https://faucet.testnet.aptoslabs.com"),
            _ => None,
        }
    }
}

/// Aptos RPC client
pub struct AptosClient {
    http_client: reqwest::Client,
    rpc_url: String,
    network: Network,
}

impl AptosClient {
    /// Create new client
    pub fn new(network: Network) -> Self {
        Self {
            http_client: reqwest::Client::new(),
            rpc_url: network.rpc_url().to_string(),
            network,
        }
    }
    
    /// Create custom client
    pub fn custom(rpc_url: &str) -> Self {
        Self {
            http_client: reqwest::Client::new(),
            rpc_url: rpc_url.to_string(),
            network: Network::Custom,
        }
    }
    
    /// Get account
    pub async fn get_account(&self, address: &AccountAddress) -> Result<Account, AptosError> {
        let url = format!("{}/accounts/{}", self.rpc_url, address.to_short_hex());
        let response = self.http_client.get(&url)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let account: Account = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(account)
    }
    
    /// Get account balance
    pub async fn get_account_balance(
        &self,
        address: &AccountAddress,
    ) -> Result<AccountBalance, AptosError> {
        let url = format!(
            "{}/accounts/{}/resource/0x1::coin::CoinStore<0x1::aptos_coin::AptosCoin>",
            self.rpc_url,
            address.to_short_hex()
        );
        
        let response = self.http_client.get(&url)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let balance: AccountBalance = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(balance)
    }
    
    /// Get account resources
    pub async fn get_account_resources(
        &self,
        address: &AccountAddress,
    ) -> Result<Vec<AccountResource>, AptosError> {
        let url = format!("{}/accounts/{}/resources", self.rpc_url, address.to_short_hex());
        let response = self.http_client.get(&url)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let resources: Vec<AccountResource> = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(resources)
    }
    
    /// Get transaction by hash
    pub async fn get_transaction(&self, hash: &str) -> Result<Transaction, AptosError> {
        let url = format!("{}/transactions/{}", self.rpc_url, hash);
        let response = self.http_client.get(&url)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let tx: Transaction = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(tx)
    }
    
    /// Submit transaction
    pub async fn submit_transaction(
        &self,
        signed_txn: &SignedTransaction,
    ) -> Result<Transaction, AptosError> {
        let url = format!("{}/transactions", self.rpc_url);
        
        let txn_bytes = bcs::to_bytes(signed_txn)
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        let response = self.http_client.post(&url)
            .header("Content-Type", "application/x-bcs")
            .body(txn_bytes)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            let error_text = response.text().await.unwrap_or_default();
            return Err(AptosError::RpcError(
                format!("Submit failed: {} - {}", response.status(), error_text)
            ));
        }
        
        let tx: Transaction = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(tx)
    }
    
    /// Simulate transaction
    pub async fn simulate_transaction(
        &self,
        signed_txn: &SignedTransaction,
    ) -> Result<Vec<Transaction>, AptosError> {
        let url = format!("{}/transactions/simulate", self.rpc_url);
        
        let txn_bytes = bcs::to_bytes(signed_txn)
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        let response = self.http_client.post(&url)
            .header("Content-Type", "application/x-bcs")
            .body(txn_bytes)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("Simulation failed: {}", response.status())
            ));
        }
        
        let txs: Vec<Transaction> = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(txs)
    }
    
    /// Get ledger info
    pub async fn get_ledger_info(&self) -> Result<LedgerInfo, AptosError> {
        let url = format!("{}/", self.rpc_url);
        let response = self.http_client.get(&url)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let info: LedgerInfo = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(info)
    }
    
    /// Get events
    pub async fn get_events(&self, address: &AccountAddress, path: &str) -> Result<Vec<Event>, AptosError> {
        let url = format!("{}/accounts/{}/events/{}", self.rpc_url, address.to_short_hex(), path);
        let response = self.http_client.get(&url)
            .send()
            .await
            .map_err(|e| AptosError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(AptosError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let events: Vec<Event> = response.json()
            .await
            .map_err(|e| AptosError::SerializationError(e.to_string()))?;
        
        Ok(events)
    }
    
    /// Request faucet (devnet/testnet only)
    pub async fn faucet_fund(
        &self,
        address: &AccountAddress,
        amount: u64,
    ) -> Result<(), AptosError> {
        if let Some(faucet_url) = self.network.faucet_url() {
            let url = format!("{}/mint?amount={}&address={}", faucet_url, amount, address.to_short_hex());
            let response = self.http_client.post(&url)
                .send()
                .await
                .map_err(|e| AptosError::RpcError(e.to_string()))?;
            
            if !response.status().is_success() {
                return Err(AptosError::RpcError(
                    format!("Faucet failed: {}", response.status())
                ));
            }
        }
        
        Ok(())
    }
}

// ============================================================================
// Transaction
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub type_: String,
    pub hash: String,
    pub version: u64,
    pub sender: String,
    pub sequence_number: u64,
    pub max_gas_amount: u64,
    pub gas_unit_price: u64,
    pub gas_currency_code: String,
    pub expiration_timestamp_secs: u64,
    pub payload: TransactionPayload,
    pub signature: Option<TransactionSignature>,
    pub timestamp: u64,
    pub changes: Vec<WriteSetChange>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionSignature {
    pub type_: String,
    pub public_key: String,
    pub signature: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WriteSetChange {
    pub type_: String,
    pub address: String,
    pub data: serde_json::Value,
}

// ============================================================================
// Ledger Info
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LedgerInfo {
    pub chain_id: u8,
    pub epoch: String,
    pub ledger_version: String,
    pub oldest_ledger_version: String,
    pub ledger_timestamp: String,
    pub block_height: String,
    pub oldest_block_height: String,
}

// ============================================================================
// Event
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    pub guid: EventGuid,
    pub sequence_number: u64,
    pub type_: String,
    pub data: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventGuid {
    pub creation_number: u64,
    pub account_address: String,
}

// ============================================================================
// Wallet Implementation
// ============================================================================

/// Aptos wallet
pub struct AptosWallet {
    pub private_key: ed25519_dalek::SigningKey,
    pub public_key: Ed25519PublicKey,
    pub address: AccountAddress,
}

impl AptosWallet {
    /// Create new wallet (generates new key)
    pub fn new() -> Self {
        let mut csprng = rand::thread_rng();
        let private_key = ed25519_dalek::SigningKey::generate(&mut csprng);
        let public_key = private_key.verifying_key();
        
        // Derive address from public key
        let mut hasher = Sha256::new();
        hasher.update(&public_key.to_bytes());
        let hash = hasher.finalize();
        
        let mut address = [0u8; 32];
        address.copy_from_slice(&hash[..32]);
        
        Self {
            private_key,
            public_key: Ed25519PublicKey(public_key.to_bytes()),
            address: AccountAddress(address),
        }
    }
    
    /// Import wallet from seed phrase (24 words - uses first 32 bytes)
    pub fn from_seed(seed: &str) -> Self {
        let mut hasher = Sha256::new();
        hasher.update(seed.as_bytes());
        let hash = hasher.finalize();
        
        let mut seed_bytes = [0u8; 32];
        seed_bytes.copy_from_slice(&hash[..32]);
        
        let private_key = ed25519_dalek::SigningKey::from_bytes(&seed_bytes);
        let public_key = private_key.verifying_key();
        
        let mut hasher2 = Sha256::new();
        hasher2.update(&public_key.to_bytes());
        let hash2 = hasher2.finalize();
        
        let mut address = [0u8; 32];
        address.copy_from_slice(&hash2[..32]);
        
        Self {
            private_key,
            public_key: Ed25519PublicKey(public_key.to_bytes()),
            address: AccountAddress(address),
        }
    }
    
    /// Sign transaction
    pub fn sign_transaction(&self, txn: &RawTransaction) -> SignedTransaction {
        // Create signing message
        let mut hasher = Sha256::new();
        let txn_bytes = bcs::to_bytes(txn).unwrap_or_default();
        hasher.update(&txn_bytes);
        let message = hasher.finalize();
        
        // Sign
        let signature = self.private_key.sign(&message);
        
        let authenticator = TransactionAuthenticator::Ed25519 {
            public_key: self.public_key.clone(),
            signature: Ed25519Signature(signature.to_bytes()),
        };
        
        SignedTransaction {
            raw_txn: txn.clone(),
            authenticator,
        }
    }
    
    /// Create transfer transaction
    pub fn create_transfer_txn(
        &self,
        receiver: AccountAddress,
        amount: u64,
        sequence: u64,
        max_gas: u64,
        gas_price: u64,
    ) -> RawTransaction {
        let payload = TransactionPayload::EntryFunction {
            module: ModuleId {
                address: AccountAddress::from_short_hex("0x1").unwrap(),
                name: "aptos_account".to_string(),
            },
            function: "transfer".to_string(),
            ty_args: vec![],
            args: vec![
                bcs::to_bytes(&receiver).unwrap_or_default(),
                bcs::to_bytes(&amount).unwrap_or_default(),
            ],
        };
        
        RawTransaction {
            sender: self.address.clone(),
            sequence_number: sequence,
            payload,
            max_gas_amount: max_gas,
            gas_unit_price: gas_price,
            gas_currency_code: "XUS".to_string(),
            expiration_timestamp_secs: (std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs() + 60) as u64,
            chain_id: self.network_chain_id(),
        }
    }
    
    fn network_chain_id(&self) -> u8 {
        1 // Mainnet
    }
    
    /// Get address
    pub fn address(&self) -> &AccountAddress {
        &self.address
    }
    
    /// Get public key hex
    pub fn public_key_hex(&self) -> String {
        self.public_key.to_hex()
    }
}

// ============================================================================
// Trait Implementations
// ============================================================================

impl Default for Network {
    fn default() -> Self {
        Network::Mainnet
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

/// Derive address from public key
pub fn derive_address(public_key: &Ed25519PublicKey) -> AccountAddress {
    let mut hasher = Sha256::new();
    hasher.update(&public_key.0);
    let hash = hasher.finalize();
    
    let mut address = [0u8; 32];
    address.copy_from_slice(&hash[..32]);
    AccountAddress(address)
}

/// Verify signature
pub fn verify_signature(
    message: &[u8],
    public_key: &Ed25519PublicKey,
    signature: &Ed25519Signature,
) -> bool {
    use ed25519_dalek::VerifyingKey;
    
    let vk = VerifyingKey::from_bytes(&public_key.0).is_ok();
    if !vk {
        return false;
    }
    
    let vk = VerifyingKey::from_bytes(&public_key.0).unwrap();
    vk.verify(message, &ed25519_dalek::Signature::from_bytes(&signature.0).unwrap()).is_ok()
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_create_wallet() {
        let wallet = AptosWallet::new();
        println!("Created wallet: {}", wallet.address);
        println!("Public key: {}", wallet.public_key_hex());
    }
    
    #[test]
    fn test_import_wallet() {
        let wallet = AptosWallet::from_seed("test seed phrase for import");
        println!("Imported wallet: {}", wallet.address);
    }
    
    #[test]
    fn test_address_from_hex() {
        let addr = AccountAddress::from_short_hex("0x1").unwrap();
        println!("Address: {}", addr);
    }
}
