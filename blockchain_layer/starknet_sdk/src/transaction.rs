//! Starknet Transaction Types
//! 
//! Implementation of Starknet transaction types and building.

use serde::{Deserialize, Serialize};
use starknet_crypto::{pedersen_hash, FieldElement};
use std::fmt;
use crate::address::StarknetAddress;
use crate::crypto::{KeyPair, Signature};

/// The felt prefixes Starknet mixes into each transaction hash so that a hash of
/// one transaction kind cannot be replayed as another.
const INVOKE_PREFIX: &str = "0x696e766f6b65";
const DEPLOY_ACCOUNT_PREFIX: &str = "0x6465706c6f795f6163636f756e74";
const DECLARE_PREFIX: &str = "0x6465636c617265";

/// `SN_MAIN`, the mainnet chain id, used when a transaction does not set one.
fn mainnet_chain_id() -> [u8; 32] {
    let mut out = [0u8; 32];
    out[32 - b"SN_MAIN".len()..].copy_from_slice(b"SN_MAIN");
    out
}

/// Failures that prevent a transaction from being hashed or signed.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TransactionError {
    /// A field of the transaction is not a valid felt, so it cannot be hashed.
    InvalidFieldElement(&'static str),
    /// The key pair refused to sign the hash.
    Signing(String),
}

impl fmt::Display for TransactionError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            TransactionError::InvalidFieldElement(field) => {
                write!(f, "{field} is not a valid Starknet field element")
            }
            TransactionError::Signing(reason) => write!(f, "signing failed: {reason}"),
        }
    }
}

impl std::error::Error for TransactionError {}

/// Convert 32 big-endian bytes to a felt. Values at or above the field modulus
/// are not felts and are rejected rather than silently reduced, since a reduced
/// value would hash to a transaction the signer never intended.
fn felt(bytes: &[u8; 32], field: &'static str) -> Result<FieldElement, TransactionError> {
    FieldElement::from_bytes_be(bytes).map_err(|_| TransactionError::InvalidFieldElement(field))
}

fn prefix(hex: &str) -> FieldElement {
    FieldElement::from_hex_be(hex).expect("transaction prefix constants are valid felts")
}

/// Starknet's `compute_hash_on_elements`, used for calldata: a Pedersen chain
/// seeded and terminated with the element count.
fn hash_elements(elements: &[FieldElement]) -> FieldElement {
    let count = FieldElement::from(elements.len() as u64);
    let mut hash = count;
    for element in elements {
        hash = pedersen_hash(&hash, element);
    }
    pedersen_hash(&hash, &count)
}

/// Hash a calldata list, which the transaction hash commits to as a single felt.
fn calldata_hash(calldata: &[[u8; 32]]) -> Result<FieldElement, TransactionError> {
    let mut elements = Vec::with_capacity(calldata.len());
    for item in calldata {
        elements.push(felt(item, "calldata")?);
    }
    Ok(hash_elements(&elements))
}

/// Invoke transaction type
#[derive(Debug, Clone)]
pub enum InvokeType {
    /// Direct call to a function
    InvokeFunction(InvokeFunction),
    /// Contract deployment
    Deploy(DeployTransaction),
}

/// Deploy transaction (legacy CONTRACT_DEPLOY variant)
#[derive(Debug, Clone)]
pub struct DeployTransaction {
    /// Class hash being deployed
    pub class_hash: [u8; 32],
    /// Constructor calldata
    pub constructor_calldata: Vec<[u8; 32]>,
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
    /// Chain id, which the hash commits to so a signature cannot be replayed
    /// on another network.
    pub chain_id: [u8; 32],
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
            chain_id: mainnet_chain_id(),
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
    pub fn sign(&mut self, key_pair: &KeyPair) -> Result<(), TransactionError> {
        let tx_hash = self.compute_hash()?;
        let sig = key_pair
            .sign(&tx_hash)
            .map_err(|e| TransactionError::Signing(e.to_string()))?;
        self.signature = Some(sig);
        Ok(())
    }
    
    /// Compute the transaction hash: `pedersen(prefix, version, address,
    /// entry_point_selector, calldata_hash, max_fee, chain_id, nonce)`. The
    /// chain id is part of the hash, so a signature cannot be replayed on
    /// another network.
    pub fn compute_hash(&self) -> Result<[u8; 32], TransactionError> {
        let hash = pedersen_hash(&prefix(INVOKE_PREFIX), &felt(&self.version, "version")?);
        let hash = pedersen_hash(&hash, &felt(&self.contract_address.to_felt252(), "contract_address")?);
        let hash = pedersen_hash(&hash, &felt(&self.entry_point_selector, "entry_point_selector")?);
        let hash = pedersen_hash(&hash, &calldata_hash(&self.calldata)?);
        let hash = pedersen_hash(&hash, &felt(&self.max_fee, "max_fee")?);
        let hash = pedersen_hash(&hash, &felt(&self.chain_id, "chain_id")?);
        Ok(pedersen_hash(&hash, &felt(&self.nonce, "nonce")?).to_bytes_be())
    }
    
    /// Encode to JSON for RPC
    pub fn to_json(&self) -> String {
        serde_json::json!({
            "contract_address": self.contract_address.to_hex(),
            "entry_point_selector": hex::encode(self.entry_point_selector),
            "calldata": self.calldata.iter().map(hex::encode).collect::<Vec<_>>(),
            "max_fee": hex::encode(self.max_fee),
            "version": hex::encode(self.version),
            "nonce": hex::encode(self.nonce),
        })
        .to_string()
    }
}

/// Deploy account transaction
#[derive(Debug, Clone)]
pub struct DeployAccountTransaction {
    /// The account address being deployed, which the hash commits to.
    pub address: StarknetAddress,
    /// Class hash
    pub class_hash: [u8; 32],
    /// Salt for address derivation
    pub salt: [u8; 32],
    /// Constructor calldata
    pub constructor_calldata: Vec<[u8; 32]>,
    /// Version
    pub version: [u8; 32],
    /// Chain id, which the hash commits to so a signature cannot be replayed
    /// on another network.
    pub chain_id: [u8; 32],
    /// Max fee
    pub max_fee: [u8; 32],
    /// Nonce
    pub nonce: [u8; 32],
    /// Signature
    pub signature: Option<Signature>,
}

impl DeployAccountTransaction {
    /// Create a new deploy-account transaction. The address must be the one
    /// derived from this class hash, salt and calldata, since the hash commits
    /// to it and the node rejects a mismatch.
    pub fn new(
        address: StarknetAddress,
        class_hash: [u8; 32],
        salt: [u8; 32],
        constructor_calldata: Vec<[u8; 32]>,
    ) -> Self {
        Self {
            address,
            class_hash,
            salt,
            constructor_calldata,
            version: [0u8; 32],
            chain_id: mainnet_chain_id(),
            max_fee: [0u8; 32],
            nonce: [0u8; 32],
            signature: None,
        }
    }
    
    /// Sign transaction
    pub fn sign(&mut self, key_pair: &KeyPair) -> Result<(), TransactionError> {
        let tx_hash = self.compute_hash()?;
        let sig = key_pair
            .sign(&tx_hash)
            .map_err(|e| TransactionError::Signing(e.to_string()))?;
        self.signature = Some(sig);
        Ok(())
    }
    
    /// Compute the transaction hash: `pedersen(prefix, version, address,
    /// class_hash, salt, calldata_hash, max_fee, chain_id, nonce)`. The address
    /// is included because it is derived from the class hash and salt, so
    /// omitting it would let two accounts claim the same signature.
    pub fn compute_hash(&self) -> Result<[u8; 32], TransactionError> {
        let hash = pedersen_hash(&prefix(DEPLOY_ACCOUNT_PREFIX), &felt(&self.version, "version")?);
        let hash = pedersen_hash(&hash, &felt(&self.address.to_felt252(), "address")?);
        let hash = pedersen_hash(&hash, &felt(&self.class_hash, "class_hash")?);
        let hash = pedersen_hash(&hash, &felt(&self.salt, "salt")?);
        let hash = pedersen_hash(&hash, &calldata_hash(&self.constructor_calldata)?);
        let hash = pedersen_hash(&hash, &felt(&self.max_fee, "max_fee")?);
        let hash = pedersen_hash(&hash, &felt(&self.chain_id, "chain_id")?);
        Ok(pedersen_hash(&hash, &felt(&self.nonce, "nonce")?).to_bytes_be())
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
    /// Chain id, which the hash commits to so a signature cannot be replayed
    /// on another network.
    pub chain_id: [u8; 32],
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
            chain_id: mainnet_chain_id(),
            max_fee: [0u8; 32],
            nonce: [0u8; 32],
            signature: None,
        }
    }
    
    /// Sign transaction
    pub fn sign(&mut self, key_pair: &KeyPair) -> Result<(), TransactionError> {
        let tx_hash = self.compute_hash()?;
        let sig = key_pair
            .sign(&tx_hash)
            .map_err(|e| TransactionError::Signing(e.to_string()))?;
        self.signature = Some(sig);
        Ok(())
    }
    
    /// Compute the transaction hash: `pedersen(prefix, version, sender_address,
    /// class_hash, max_fee, chain_id, nonce)`.
    pub fn compute_hash(&self) -> Result<[u8; 32], TransactionError> {
        let hash = pedersen_hash(&prefix(DECLARE_PREFIX), &felt(&self.version, "version")?);
        let hash = pedersen_hash(&hash, &felt(&self.sender_address.to_felt252(), "sender_address")?);
        let hash = pedersen_hash(&hash, &felt(&self.class_hash, "class_hash")?);
        let hash = pedersen_hash(&hash, &felt(&self.max_fee, "max_fee")?);
        let hash = pedersen_hash(&hash, &felt(&self.chain_id, "chain_id")?);
        Ok(pedersen_hash(&hash, &felt(&self.nonce, "nonce")?).to_bytes_be())
    }
}

/// Transaction receipt
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
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
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
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
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    /// From address
    pub from_address: StarknetAddress,
    /// Keys (topics)
    pub keys: Vec<[u8; 32]>,
    /// Data
    pub data: Vec<[u8; 32]>,
}

/// L1 message
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct L1Message {
    /// From address
    pub from_address: [u8; 32],
    /// Payload
    pub payload: Vec<[u8; 32]>,
}

/// Transaction type enum for RPC
#[derive(Debug, Clone)]
pub enum Transaction {
    Invoke(InvokeTransaction),
    Declare(DeclareTransaction),
    DeployAccount(DeployAccountTransaction),
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


#[cfg(test)]
mod tests {
    use super::*;
    use crate::crypto::KeyPair;

    fn addr() -> StarknetAddress {
        StarknetAddress::from_public_key(&[3u8; 32], &[4u8; 32]).unwrap()
    }

    fn invoke() -> InvokeFunction {
        InvokeFunction::new(addr(), [5u8; 32], vec![[6u8; 32]])
    }

    #[test]
    fn invoke_hash_is_deterministic() {
        assert_eq!(invoke().compute_hash().unwrap(), invoke().compute_hash().unwrap());
    }

    /// The chain id is part of the hash, so the same transaction signed for two
    /// networks hashes differently. Omitting it was the replay bug.
    #[test]
    fn invoke_hash_commits_to_chain_id() {
        let mainnet = invoke();
        let mut other = invoke();
        other.chain_id = [1u8; 32];
        assert_ne!(mainnet.compute_hash().unwrap(), other.compute_hash().unwrap());
    }

    #[test]
    fn invoke_hash_commits_to_nonce_and_fee() {
        let base = invoke();
        let mut bumped_nonce = invoke();
        bumped_nonce.nonce = [1u8; 32];
        assert_ne!(base.compute_hash().unwrap(), bumped_nonce.compute_hash().unwrap());

        let mut bumped_fee = invoke();
        bumped_fee.max_fee = [2u8; 32];
        assert_ne!(base.compute_hash().unwrap(), bumped_fee.compute_hash().unwrap());
    }

    /// Calldata is committed to as a whole, so changing one argument changes the
    /// hash. Hashing the concatenated bytes would miss argument boundaries.
    #[test]
    fn invoke_hash_commits_to_calldata() {
        let base = invoke();
        let mut changed = invoke();
        changed.calldata = vec![[7u8; 32]];
        assert_ne!(base.compute_hash().unwrap(), changed.compute_hash().unwrap());
    }

    #[test]
    fn invoke_hash_commits_to_entry_point() {
        let base = invoke();
        let mut changed = invoke();
        changed.entry_point_selector = [1u8; 32];
        assert_ne!(base.compute_hash().unwrap(), changed.compute_hash().unwrap());
    }

    /// A transaction kind prefix keeps an invoke hash from ever equalling a
    /// declare hash built from otherwise identical fields.
    #[test]
    fn transaction_kinds_use_distinct_prefixes() {
        assert_ne!(prefix(INVOKE_PREFIX), prefix(DECLARE_PREFIX));
        assert_ne!(prefix(DECLARE_PREFIX), prefix(DEPLOY_ACCOUNT_PREFIX));
        assert_ne!(prefix(INVOKE_PREFIX), prefix(DEPLOY_ACCOUNT_PREFIX));
    }

    #[test]
    fn signing_sets_a_verifiable_signature() {
        let kp = KeyPair::from_private_key([1u8; 32]).unwrap();
        let mut tx = invoke();
        tx.sign(&kp).unwrap();
        let sig = tx.signature.clone().expect("signature is set");
        assert!(kp.verify(&tx.compute_hash().unwrap(), &sig));
    }

    /// A field value at or above the modulus is not a felt and must be rejected
    /// rather than reduced, which would sign a different transaction.
    #[test]
    fn out_of_field_value_is_rejected() {
        let mut tx = invoke();
        tx.nonce = [0xFFu8; 32];
        assert_eq!(
            tx.compute_hash().unwrap_err(),
            TransactionError::InvalidFieldElement("nonce")
        );
    }

    #[test]
    fn deploy_account_hash_commits_to_address_and_salt() {
        let base = DeployAccountTransaction::new(addr(), [1u8; 32], [0u8; 32], vec![[2u8; 32]]);
        let mut other_salt = DeployAccountTransaction::new(addr(), [1u8; 32], [1u8; 32], vec![[2u8; 32]]);
        other_salt.chain_id = base.chain_id;
        assert_ne!(base.compute_hash().unwrap(), other_salt.compute_hash().unwrap());

        let other_addr = StarknetAddress::from_public_key(&[3u8; 32], &[5u8; 32]).unwrap();
        let mut other = DeployAccountTransaction::new(other_addr, [1u8; 32], [0u8; 32], vec![[2u8; 32]]);
        other.chain_id = base.chain_id;
        assert_ne!(base.compute_hash().unwrap(), other.compute_hash().unwrap());
    }

    #[test]
    fn declare_hash_is_deterministic_and_commits_to_class_hash() {
        let base = DeclareTransaction::new([1u8; 32], addr());
        assert_eq!(base.compute_hash().unwrap(), base.compute_hash().unwrap());
        let other = DeclareTransaction::new([2u8; 32], addr());
        assert_ne!(base.compute_hash().unwrap(), other.compute_hash().unwrap());
    }

    #[test]
    fn mainnet_chain_id_is_sn_main() {
        let id = mainnet_chain_id();
        assert_eq!(&id[25..], b"SN_MAIN");
        assert_eq!(&id[..25], &[0u8; 25]);
    }
}
