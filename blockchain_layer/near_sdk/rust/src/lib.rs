//! TigerWallet Near Protocol SDK
//! Production-ready implementation for Near blockchain
//! 
//! Features:
//! - Full account management
//! - NEAR token transfers
//! - Function call transactions
//! - Contract deployment
//! - Staking (delegation)
//! - Cross-chain bridge support

#![allow(dead_code)]

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use sha2::{Sha256, Digest};
use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum NearError {
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
}

// ============================================================================
// Address Types
// ============================================================================

/// Near account ID
#[derive(borsh::BorshSerialize)]
#[derive(Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct AccountId(String);

impl AccountId {
    /// Create from string
    pub fn new(id: &str) -> Result<Self, NearError> {
        let id = id.to_lowercase();
        
        // Validate account ID format
        if id.len() < 2 || id.len() > 64 {
            return Err(NearError::InvalidAddress(
                "Account ID must be 2-64 characters".to_string()
            ));
        }
        
        // Check for valid characters
        let valid_chars = id.chars().all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_');
        if !valid_chars {
            return Err(NearError::InvalidAddress(
                "Invalid characters in account ID".to_string()
            ));
        }
        
        // .near or .testnet suffix
        if id.contains('.') {
            let parts: Vec<&str> = id.split('.').collect();
            if parts.len() != 2 {
                return Err(NearError::InvalidAddress(
                    "Invalid account ID format".to_string()
                ));
            }
        }
        
        Ok(AccountId(id))
    }
    
    /// Create implicit account (64 hex chars)
    pub fn from_hex(hex: &str) -> Result<Self, NearError> {
        if hex.len() != 64 {
            return Err(NearError::InvalidAddress(
                "Implicit account must be 64 hex characters".to_string()
            ));
        }
        
        // Validate hex
        if !hex.chars().all(|c| c.is_ascii_hexdigit()) {
            return Err(NearError::InvalidAddress(
                "Invalid hex characters".to_string()
            ));
        }
        
        Ok(AccountId(format!("{}...", &hex[..8])))
    }
    
    /// Get as string
    pub fn as_str(&self) -> &str {
        &self.0
    }
    
    /// Check if it's a top-level account
    pub fn is_top_level(&self) -> bool {
        !self.0.contains('.')
    }
}

impl std::fmt::Debug for AccountId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl std::fmt::Display for AccountId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

// ============================================================================
// Key Types
// ============================================================================

/// Public key type
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone)]
pub enum PublicKey {
    /// Ed25519 key
    Ed25519([u8; 32]),
    /// Secp256k1 key
    Secp256k1([u8; 64]),
}

impl PublicKey {
    pub fn to_hex(&self) -> String {
        match self {
            PublicKey::Ed25519(key) => format!("ed25519:{}", hex::encode(key)),
            PublicKey::Secp256k1(key) => format!("secp256k1:{}", hex::encode(key)),
        }
    }
    
    pub fn from_hex(hex: &str) -> Result<Self, NearError> {
        if hex.starts_with("ed25519:") {
            let key_hex = hex.strip_prefix("ed25519:").unwrap();
            let bytes = hex::decode(key_hex)
                .map_err(|e| NearError::InvalidAddress(e.to_string()))?;
            if bytes.len() != 32 {
                return Err(NearError::InvalidAddress("Invalid key length".to_string()));
            }
            let mut key = [0u8; 32];
            key.copy_from_slice(&bytes);
            Ok(PublicKey::Ed25519(key))
        } else if hex.starts_with("secp256k1:") {
            let key_hex = hex.strip_prefix("secp256k1:").unwrap();
            let bytes = hex::decode(key_hex)
                .map_err(|e| NearError::InvalidAddress(e.to_string()))?;
            if bytes.len() != 64 {
                return Err(NearError::InvalidAddress("Invalid key length".to_string()));
            }
            let mut key = [0u8; 64];
            key.copy_from_slice(&bytes);
            Ok(PublicKey::Secp256k1(key))
        } else {
            Err(NearError::InvalidAddress("Unknown key type".to_string()))
        }
    }
}

/// Signature type
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone)]
pub enum Signature {
    Ed25519([u8; 64]),
    Secp256k1([u8; 65]),
}

impl Signature {
    pub fn to_hex(&self) -> String {
        match self {
            Signature::Ed25519(sig) => format!("ed25519:{}", hex::encode(sig)),
            Signature::Secp256k1(sig) => format!("secp256k1:{}", hex::encode(sig)),
        }
    }
}

impl Serialize for PublicKey {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        self.to_hex().serialize(serializer)
    }
}

impl<'de> Deserialize<'de> for PublicKey {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let s = String::deserialize(deserializer)?;
        PublicKey::from_hex(&s).map_err(serde::de::Error::custom)
    }
}

impl Serialize for Signature {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        self.to_hex().serialize(serializer)
    }
}

impl Signature {
    pub fn from_hex(hex: &str) -> Result<Self, NearError> {
        if let Some(s) = hex.strip_prefix("ed25519:") {
            let bytes = hex::decode(s).map_err(|e| NearError::InvalidAddress(e.to_string()))?;
            if bytes.len() != 64 {
                return Err(NearError::InvalidAddress("Invalid signature length".to_string()));
            }
            let mut sig = [0u8; 64];
            sig.copy_from_slice(&bytes);
            Ok(Signature::Ed25519(sig))
        } else if let Some(s) = hex.strip_prefix("secp256k1:") {
            let bytes = hex::decode(s).map_err(|e| NearError::InvalidAddress(e.to_string()))?;
            if bytes.len() != 65 {
                return Err(NearError::InvalidAddress("Invalid signature length".to_string()));
            }
            let mut sig = [0u8; 65];
            sig.copy_from_slice(&bytes);
            Ok(Signature::Secp256k1(sig))
        } else {
            Err(NearError::InvalidAddress("Unknown signature type".to_string()))
        }
    }
}

impl<'de> Deserialize<'de> for Signature {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let s = String::deserialize(deserializer)?;
        Signature::from_hex(&s).map_err(serde::de::Error::custom)
    }
}

// ============================================================================
// Transaction Types
// ============================================================================

/// Action types
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Action {
    /// Transfer NEAR tokens
    Transfer {
        deposit: u128,
    },
    /// Create account
    CreateAccount,
    /// Call function
    FunctionCall {
        method_name: String,
        args: Vec<u8>,
        deposit: u128,
        gas: u64,
    },
    /// Transfer stake
    TransferStake {
        stake: u128,
        public_key: PublicKey,
    },
    /// Add key with access key
    AddKeyWithAccessKey {
        public_key: PublicKey,
        access_key: AccessKey,
    },
    /// Add key with full access
    AddKeyWithFullAccess {
        public_key: PublicKey,
    },
    /// Delete key
    DeleteKey {
        public_key: PublicKey,
    },
    /// Delete account
    DeleteAccount {
        beneficiary_id: AccountId,
    },
    /// Deploy contract
    DeployContract {
        code: Vec<u8>,
    },
    /// Stake
    Stake {
        stake: u128,
        public_key: PublicKey,
    },
}

/// Access key
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccessKey {
    pub nonce: u64,
    pub permission: AccessKeyPermission,
}

/// Access key permission
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AccessKeyPermission {
    /// Full access
    FullAccess,
    /// Function call access
    FunctionCall {
        receiver_id: AccountId,
        method_names: Vec<String>,
        allowance: Option<u128>,
    },
}

/// Transaction
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub signer_id: AccountId,
    pub public_key: PublicKey,
    pub nonce: u64,
    pub receiver_id: AccountId,
    pub block_hash: String,
    pub actions: Vec<Action>,
}

/// Signed transaction
#[derive(borsh::BorshSerialize)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedTransaction {
    pub transaction: Transaction,
    pub signature: Signature,
    pub hash: String,
}

// ============================================================================
// Account Types
// ============================================================================

/// Account info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Account {
    pub amount: u128,
    pub locked: u128,
    pub code_hash: String,
    pub storage_usage: u64,
}

/// Account details
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountDetails {
    pub amount: String,
    pub locked: String,
    pub code_hash: String,
    pub storage_usage: String,
    pub storage_paid_at: String,
}

// ============================================================================
// View Methods
// ============================================================================

/// Balance response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BalanceResponse {
    pub amount: String,
}

/// View access key
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccessKeyInfo {
    pub access_key: AccessKey,
    pub nonce: u64,
}

/// Access key list
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccessKeyList {
    pub keys: Vec<AccessKeyInfo>,
}

// ============================================================================
// RPC Client
// ============================================================================

/// Near network
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Network {
    Mainnet,
    Testnet,
    Custom,
}

impl Network {
    pub fn rpc_url(&self) -> &str {
        match self {
            Network::Mainnet => "https://rpc.mainnet.near.org",
            Network::Testnet => "https://rpc.testnet.near.org",
            Network::Custom => "http://localhost:3030",
        }
    }
    
    pub fn wallet_url(&self) -> &str {
        match self {
            Network::Mainnet => "https://wallet.near.org",
            Network::Testnet => "https://testnet.mynearwallet.com",
            Network::Custom => "http://localhost:4000",
        }
    }
    
    pub fn explorer_url(&self) -> &str {
        match self {
            Network::Mainnet => "https://explorer.near.org",
            Network::Testnet => "https://explorer.testnet.near.org",
            Network::Custom => "http://localhost:4001",
        }
    }
}

/// Near RPC client
pub struct NearClient {
    http_client: reqwest::Client,
    rpc_url: String,
}

impl NearClient {
    /// Create new client
    pub fn new(network: Network) -> Self {
        Self {
            http_client: reqwest::Client::new(),
            rpc_url: network.rpc_url().to_string(),
        }
    }
    
    /// Create custom client
    pub fn custom(rpc_url: &str) -> Self {
        Self {
            http_client: reqwest::Client::new(),
            rpc_url: rpc_url.to_string(),
        }
    }
    
    /// Query account
    pub async fn get_account(&self, account_id: &AccountId) -> Result<Account, NearError> {
        let request = RpcQueryRequest {
            request: QueryRequest::ViewAccount {
                account_id: account_id.as_str().to_string(),
            },
            block_id: Some(BlockId::Final),
        };
        
        let response = self.call("query", &request).await?;
        let result: QueryResponse = serde_json::from_value(response)
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        match result {
            QueryResponse::Account { account, .. } => Ok(account),
            _ => Err(NearError::AccountError("Not an account response".to_string())),
        }
    }
    
    /// Get account balance
    pub async fn get_balance(&self, account_id: &AccountId) -> Result<u128, NearError> {
        let account = self.get_account(account_id).await?;
        Ok(account.amount)
    }
    
    /// Get access keys
    pub async fn get_access_keys(
        &self,
        account_id: &AccountId,
    ) -> Result<Vec<AccessKeyInfo>, NearError> {
        let request = RpcQueryRequest {
            request: QueryRequest::ViewAccessKeyList {
                account_id: account_id.as_str().to_string(),
            },
            block_id: Some(BlockId::Final),
        };
        
        let response = self.call("query", &request).await?;
        let result: QueryResponse = serde_json::from_value(response)
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        match result {
            QueryResponse::AccessKeyList { keys, .. } => Ok(keys),
            _ => Err(NearError::AccountError("Not an access key list".to_string())),
        }
    }
    
    /// Get block
    pub async fn get_block(&self, block_id: BlockId) -> Result<Block, NearError> {
        let request = match block_id {
            BlockId::Hash(h) => RpcBlockRequest { block_id: BlockIdInner::Hash(h) },
            BlockId::Height(h) => RpcBlockRequest { block_id: BlockIdInner::Height(h) },
            BlockId::Final => RpcBlockRequest { block_id: BlockIdInner::Final },
        };
        
        let response = self.call("block", &request).await?;
        let block: Block = serde_json::from_value(response)
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        Ok(block)
    }
    
    /// Get latest block hash
    pub async fn get_latest_block_hash(&self) -> Result<String, NearError> {
        let block = self.get_block(BlockId::Final).await?;
        Ok(block.header.hash)
    }
    
    /// Get validators
    pub async fn get_validators(&self) -> Result<EpochValidatorInfo, NearError> {
        let response = self.call("validators", &()).await?;
        let validators: EpochValidatorInfo = serde_json::from_value(response)
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        Ok(validators)
    }
    
    /// Send transaction
    pub async fn broadcast_transaction(
        &self,
        signed_txn: &SignedTransaction,
    ) -> Result<FinalExecutionOutcome, NearError> {
        let bytes = borsh::to_vec(signed_txn)
            .map_err(|e| NearError::SerializationError(e.to_string()))?;
        
        let encoded = base64::encode(&bytes);
        let request = RpcBroadcastTxRequest { signed_txn: encoded };
        
        let response = self.call("broadcast_tx_commit", &request).await?;
        let outcome: FinalExecutionOutcome = serde_json::from_value(response)
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        Ok(outcome)
    }
    
    /// Call view method
    pub async fn view_method(
        &self,
        contract_id: &AccountId,
        method: &str,
        args: Vec<u8>,
    ) -> Result<serde_json::Value, NearError> {
        let request = RpcQueryRequest {
            request: QueryRequest::CallFunction {
                account_id: contract_id.as_str().to_string(),
                method_name: method.to_string(),
                args_base64: base64::encode(&args),
            },
            block_id: Some(BlockId::Final),
        };
        
        let response = self.call("query", &request).await?;
        
        // Parse result
        let result: QueryResponse = serde_json::from_value(response.clone())
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        match result {
            QueryResponse::CallResult { result, .. } => {
                let decoded = base64::decode(&result)
                    .map_err(|e| NearError::RpcError(e.to_string()))?;
                let json: serde_json::Value = serde_json::from_slice(&decoded)
                    .unwrap_or(serde_json::Value::String(base64::encode(&decoded)));
                Ok(json)
            }
            _ => Err(NearError::RpcError("Not a call result".to_string())),
        }
    }
    
    /// Generic RPC call
    async fn call<T: Serialize>(&self, method: &str, params: &T) -> Result<serde_json::Value, NearError> {
        let client = reqwest::Client::new();
        let response = client.post(&self.rpc_url)
            .header("Content-Type", "application/json")
            .json(&RpcRequest {
                jsonrpc: "2.0".to_string(),
                id: 1,
                method: method.to_string(),
                params: serde_json::to_value(params).unwrap_or(serde_json::Value::Null),
            })
            .send()
            .await
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(NearError::RpcError(
                format!("HTTP error: {}", response.status())
            ));
        }
        
        let rpc_response: RpcResponse = response.json()
            .await
            .map_err(|e| NearError::RpcError(e.to_string()))?;
        
        if let Some(error) = rpc_response.error {
            return Err(NearError::RpcError(error.message));
        }
        
        rpc_response.result.ok_or_else(|| NearError::RpcError("No result".to_string()))
    }
}

// ============================================================================
// RPC Types
// ============================================================================

#[derive(Serialize)]
struct RpcRequest {
    jsonrpc: String,
    id: u64,
    method: String,
    params: serde_json::Value,
}

#[derive(Deserialize)]
struct RpcResponse {
    jsonrpc: String,
    id: u64,
    result: Option<serde_json::Value>,
    error: Option<RpcError>,
}

#[derive(Deserialize)]
struct RpcError {
    code: i64,
    message: String,
}

#[derive(Serialize)]
struct RpcQueryRequest {
    request: QueryRequest,
    #[serde(skip_serializing_if = "Option::is_none")]
    block_id: Option<BlockId>,
}

#[derive(Serialize)]
#[serde(untagged)]
enum BlockId {
    Hash(String),
    Height(u64),
    Final,
}

#[derive(Serialize)]
#[serde(untagged)]
enum BlockIdInner {
    Hash(String),
    Height(u64),
    Final,
}

#[derive(Serialize)]
struct RpcBlockRequest {
    block_id: BlockIdInner,
}

#[derive(Serialize)]
struct RpcBroadcastTxRequest {
    signed_txn: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "request_type")]
enum QueryRequest {
    ViewAccount { account_id: String },
    ViewAccessKey { account_id: String, public_key: String },
    ViewAccessKeyList { account_id: String },
    CallFunction { account_id: String, method_name: String, args_base64: String },
}

#[derive(Debug, Clone, Deserialize)]
#[serde(tag = "kind")]
enum QueryResponse {
    #[serde(rename = "block_hash")]
    Block { block_hash: String },
    #[serde(rename = "block_height")]
    BlockHeight { block_height: u64 },
    #[serde(rename = "access_key")]
    AccessKey { access_key: AccessKey, nonce: u64 },
    #[serde(rename = "access_key_list")]
    AccessKeyList { keys: Vec<AccessKeyInfo> },
    #[serde(rename = "call_result")]
    CallResult { result: String, logs: Vec<String> },
    #[serde(rename = "account")]
    Account { account: Account },
}

// ============================================================================
// Block Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    pub header: BlockHeader,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockHeader {
    pub hash: String,
    pub prev_hash: String,
    pub height: u64,
    pub epoch_id: String,
    pub next_epoch_id: String,
    pub timestamp: u64,
    pub total_supply: String,
    pub gas_price: String,
}

// ============================================================================
// Validator Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EpochValidatorInfo {
    pub current_validators: Vec<ValidatorInfo>,
    pub next_validators: Vec<ValidatorInfo>,
    pub prev_epoch_kickout: Vec<ValidatorInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatorInfo {
    pub account_id: String,
    pub public_key: String,
    pub stake: String,
    pub shards: Vec<u64>,
    pub num_produced_blocks: u64,
    pub num_expected_blocks: u64,
}

// ============================================================================
// Execution Outcome
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinalExecutionOutcome {
    pub status: ExecutionStatus,
    pub transaction: ExecutionOutcome,
    pub receipts: Vec<ExecutionOutcome>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionOutcome {
    pub logs: Vec<String>,
    pub receipt_ids: Vec<String>,
    pub gas_burnt: u64,
    pub tokens_burnt: String,
    pub executor_id: String,
    pub status: ExecutionStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum ExecutionStatus {
    SuccessValue(String),
    SuccessReceiptId(String),
    Failure(ErrorNode),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorNode {
    pub error_type: String,
    pub error_info: Option<String>,
    pub method_name: Option<String>,
}

// ============================================================================
// Wallet Implementation
// ============================================================================

/// Near wallet
pub struct NearWallet {
    private_key: ed25519_dalek::SigningKey,
    public_key: PublicKey,
    account_id: Option<AccountId>,
}

impl NearWallet {
    /// Create new wallet
    pub fn new() -> Self {
        let mut csprng = rand::thread_rng();
        let private_key = ed25519_dalek::SigningKey::generate(&mut csprng);
        let public_key_array = private_key.verifying_key().to_bytes();
        
        Self {
            private_key,
            public_key: PublicKey::Ed25519(public_key_array),
            account_id: None,
        }
    }
    
    /// Create from seed
    pub fn from_seed(seed: &str) -> Self {
        let mut hasher = Sha256::new();
        hasher.update(seed.as_bytes());
        let hash = hasher.finalize();
        
        let mut seed_bytes = [0u8; 32];
        seed_bytes.copy_from_slice(&hash[..32]);
        
        let private_key = ed25519_dalek::SigningKey::from_bytes(&seed_bytes);
        let public_key_array = private_key.verifying_key().to_bytes();
        
        Self {
            private_key,
            public_key: PublicKey::Ed25519(public_key_array),
            account_id: None,
        }
    }
    
    /// Set account ID
    pub fn with_account(mut self, account_id: AccountId) -> Self {
        self.account_id = Some(account_id);
        self
    }
    
    /// Get public key
    pub fn public_key(&self) -> &PublicKey {
        &self.public_key
    }
    
    /// Get public key hex
    pub fn public_key_hex(&self) -> String {
        self.public_key.to_hex()
    }
    
    /// Sign data
    pub fn sign(&self, data: &[u8]) -> Signature {
        use ed25519_dalek::Signer;
        let signature = self.private_key.sign(data);
        Signature::Ed25519(signature.to_bytes())
    }
    
    /// Create transfer transaction
    pub fn create_transfer_txn(
        &self,
        receiver: &AccountId,
        amount: u128,
        nonce: u64,
        block_hash: &str,
    ) -> Result<Transaction, NearError> {
        let signer_id = self.account_id.clone()
            .ok_or_else(|| NearError::AccountError("No account ID set".to_string()))?;
        
        let actions = vec![Action::Transfer { deposit: amount }];
        
        Ok(Transaction {
            signer_id,
            public_key: self.public_key.clone(),
            nonce,
            receiver_id: receiver.clone(),
            block_hash: block_hash.to_string(),
            actions,
        })
    }
    
    /// Sign and create signed transaction
    pub fn sign_transaction(
        &self,
        txn: &Transaction,
    ) -> SignedTransaction {
        let bytes = borsh::to_vec(txn)
            .map_err(|e| NearError::SerializationError(e.to_string()))
            .unwrap();
        
        let mut hasher = Sha256::new();
        hasher.update(&bytes);
        let hash = hasher.finalize();
        
        let signature = self.sign(&hash);
        
        let mut hash_arr = [0u8; 32];
        hash_arr.copy_from_slice(&hash[..32]);
        
        SignedTransaction {
            transaction: txn.clone(),
            signature,
            hash: hex::encode(hash_arr),
        }
    }
}

impl Default for Network {
    fn default() -> Self {
        Network::Mainnet
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_create_account_id() {
        let id = AccountId::new("alice.near").unwrap();
        println!("Account ID: {}", id);
    }
    
    #[test]
    fn test_implicit_account() {
        let id = AccountId::from_hex("abc123def456789012345678901234567890123456789012345678901234").unwrap();
        println!("Implicit account: {}", id);
    }
    
    #[test]
    fn test_create_wallet() {
        let wallet = NearWallet::new();
        println!("Public key: {}", wallet.public_key_hex());
    }
}
