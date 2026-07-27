//! Starknet Transaction Types
//! 
//! Implementation of Starknet transaction types and building.

use serde::{Deserialize, Serialize};
use crate::address::StarknetAddress;
use crate::crypto::{KeyPair, Signature};

/// Invoke transaction type
#[derive(Debug, Clone)]
pub enum InvokeType {
    /// Direct call to a function
    InvokeFunction(InvokeFunction),
    /// Contract deployment
    Deploy(DeployTransaction),
}

/// Invoke function transaction
#[derive(Debug, Clone)]
pub struct InvokeFunction {
    /// Contract address
    pub contract_address: StarknetAddress,
    /// Entry point selector (function name hash)
    pub entry_point_selector: [u8; 32],
    /// Calldata (encoded parameters)
    pub calldata: Vec<[u8; 32]>,
    /// Maximum fee
    pub max_fee: [u8; 32],
    /// Version
    pub version: [u8; 32],
    /// Nonce
    pub nonce: [u8; 32],
    /// Signature
    pub signature: Option<Signature>,
}

impl InvokeFunction {
    /// Create new invoke
    pub fn new(
        contract_address: StarknetAddress,
        entry_point_selector: [u8; 32],
        calldata: Vec<[u8; 32]>,
    ) -> Self {
        Self {
            contract_address,
            entry_point_selector,
            calldata,
            max_fee: [0u8; 32],
            version: [0u8; 32], // v0
            nonce: [0u8; 32],
            signature: None,
        }
    }
    
    /// Set max fee
    pub fn with_max_fee(mut self, max_fee: [u8; 32]) -> Self {
        self.max_fee = max_fee;
        self
    }
    
    /// Set nonce
    pub fn with_nonce(mut self, nonce: [u8; 32]) -> Self {
        self.nonce = nonce;
        self
    }
    
    /// Sign transaction
    pub fn sign(&mut self, key_pair: &KeyPair) {
        // Build transaction hash
        let tx_hash = self.compute_hash();
        
        // Sign the hash
        let sig = key_pair.sign(&tx_hash).unwrap();
        self.signature = Some(sig);
    }
    
    /// Compute transaction hash
    fn compute_hash(&self) -> [u8; 32] {
        use sha3::{Keccak256, Digest};
        
        let mut hasher = Keccak256::new();
        
        hasher.update(b"invoke");
        hasher.update(&self.version);
        hasher.update(&self.contract_address.to_felt252());
        hasher.update(&self.entry_point_selector);
        
        // Calldata
        hasher.update(&(self.calldata.len() as u32).to_be_bytes());
        for data in &self.calldata {
            hasher.update(data);
        }
        
        hasher.update(&self.max_fee);
        hasher.update(&self.nonce);
        
        let result = hasher.finalize();
        
        let mut hash = [0u8; 32];
        hash.copy_from_slice(&result);
        
        hash
    }
    
    /// Encode to JSON for RPC
    pub fn to_json(&self) -> String {
        serde_json::to_string(self).unwrap_or_default()
    }
}

/// Deploy account transaction
#[derive(Debug, Clone)]
pub struct DeployAccountTransaction {
    /// Class hash
    pub class_hash: [u8; 32],
    /// Salt for address derivation
    pub salt: [u8; 32],
    /// Constructor calldata
    pub constructor_calldata: Vec<[u8; 32]>,
    /// Version
    pub version: [u8; 32],
    /// Max fee
    pub max_fee: [u8; 32],
    /// Nonce
    pub nonce: [u8; 32],
    /// Signature
    pub signature: Option<Signature>,
}

impl DeployAccountTransaction {
    /// Create new deploy account transaction
    pub fn new(
        class_hash: [u8; 32],
        constructor_calldata: Vec<[u8; 32]>,
    ) -> Self {
        let salt = rand::random();
        
        Self {
            class_hash,
            salt,
            constructor_calldata,
            version: [0u8; 32],
            max_fee: [0u8; 32],
            nonce: [0u8; 32],
            signature: None,
        }
    }
    
    /// Sign transaction
    pub fn sign(&mut self, key_pair: &KeyPair) {
        let tx_hash = self.compute_hash();
        let sig = key_pair.sign(&tx_hash).unwrap();
        self.signature = Some(sig);
    }
    
    /// Compute transaction hash
    fn compute_hash(&self) -> [u8; 32] {
        use sha3::{Keccak256, Digest};
        
        let mut hasher = Keccak256::new();
        
        hasher.update(b"deploy_account");
        hasher.update(&self.version);
        hasher.update(&self.class_hash);
        hasher.update(&self.salt);
        
        hasher.update(&(self.constructor_calldata.len() as u32).to_be_bytes());
        for data in &self.constructor_calldata {
            hasher.update(data);
        }
        
        hasher.update(&self.max_fee);
        hasher.update(&self.nonce);
        
        let result = hasher.finalize();
        
        let mut hash = [0u8; 32];
        hash.copy_from_slice(&result);
        
        hash
    }
}

/// Declare transaction (for Cairo 1 contracts)
#[derive(Debug, Clone)]
pub struct DeclareTransaction {
    /// Class hash to declare
    pub class_hash: [u8; 32],
    /// Sender address (account)
    pub sender_address: StarknetAddress,
    /// Version
    pub version: [u8; 32],
    /// Max fee
    pub max_fee: [u8; 32],
    /// Nonce
    pub nonce: [u8; 32],
    /// Signature
    pub signature: Option<Signature>,
}

impl DeclareTransaction {
    /// Create new declare
    pub fn new(class_hash: [u8; 32], sender_address: StarknetAddress) -> Self {
        Self {
            class_hash,
            sender_address,
            version: [0u8; 32],
            max_fee: [0u8; 32],
            nonce: [0u8; 32],
            signature: None,
        }
    }
    
    /// Sign transaction
    pub fn sign(&mut self, key_pair: &KeyPair) {
        let tx_hash = self.compute_hash();
        let sig = key_pair.sign(&tx_hash).unwrap();
        self.signature = Some(sig);
    }
    
    /// Compute transaction hash
    fn compute_hash(&self) -> [u8; 32] {
        use sha3::{Keccak256, Digest};
        
        let mut hasher = Keccak256::new();
        
        hasher.update(b"declare");
        hasher.update(&self.version);
        hasher.update(&self.sender_address.to_felt252());
        hasher.update(&self.class_hash);
        hasher.update(&self.max_fee);
        hasher.update(&self.nonce);
        
        let result = hasher.finalize();
        
        let mut hash = [0u8; 32];
        hash.copy_from_slice(&result);
        
        hash
    }
}

/// Transaction receipt
#[derive(Debug, Clone, Default)]
pub struct TransactionReceipt {
    /// Transaction hash
    pub transaction_hash: [u8; 32],
    /// Block hash
    pub block_hash: Option<[u8; 32]>,
    /// Block number
    pub block_number: Option<u64>,
    /// Status
    pub status: TransactionStatus,
    /// Actual fee paid
    pub actual_fee: Option<[u8; 32]>,
    /// Events
    pub events: Vec<Event>,
    /// Messages sent to L1
    pub l1_messages: Vec<L1Message>,
}

/// Transaction status
#[derive(Debug, Clone, Default)]
pub enum TransactionStatus {
    #[default]
    Unknown,
    Received,
    Pending,
    AcceptedOnL2,
    AcceptedOnL1,
    Rejected,
}

/// Event from transaction
#[derive(Debug, Clone)]
pub struct Event {
    /// From address
    pub from_address: StarknetAddress,
    /// Keys (topics)
    pub keys: Vec<[u8; 32]>,
    /// Data
    pub data: Vec<[u8; 32]>,
}

/// L1 message
#[derive(Debug, Clone)]
pub struct L1Message {
    /// From address
    pub from_address: [u8; 32],
    /// Payload
    pub payload: Vec<[u8; 32]>,
}

/// Transaction type enum for RPC
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum Transaction {
    #[serde(rename = "INVOKE_FUNCTION")]
    Invoke(InvokeTransaction),
    #[serde(rename = "DECLARE")]
    Declare(DeclareTransaction),
    #[serde(rename = "DEPLOY_ACCOUNT")]
    DeployAccount(DeployAccountTransaction),
    #[serde(rename = "L1_HANDLER")]
    L1Handler(L1HandlerTransaction),
}

/// Invoke transaction for RPC
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InvokeTransaction {
    #[serde(rename = "contract_address")]
    pub contract_address: String,
    #[serde(rename = "entry_point_selector")]
    pub entry_point_selector: String,
    #[serde(rename = "calldata")]
    pub calldata: Vec<String>,
    #[serde(rename = "max_fee")]
    pub max_fee: String,
    pub version: String,
    pub nonce: String,
    pub signature: Option<Vec<String>>,
}

/// L1 handler transaction
#[derive(Debug, Clone)]
pub struct L1HandlerTransaction {
    /// Contract address
    pub contract_address: StarknetAddress,
    /// Entry point selector
    pub entry_point_selector: [u8; 32],
    /// Calldata
    pub calldata: Vec<[u8; 32]>,
    /// Nonce
    pub nonce: [u8; 32],
}
