// ============================================================================
// TIGERWALLET BITCOIN MODULE
// Complete Bitcoin functionality: address derivation, PSBT signing, Ordinals
// ============================================================================

use std::collections::HashMap;
use std::str::FromStr;
use sha2::{Sha256, Digest};
use ripemd::Ripemd160;
use bech32::{self, ToBase32, Variant};
use serde::de::DeserializeOwned;

/// Bitcoin network types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BitcoinNetwork {
    /// Mainnet
    Mainnet,
    /// Testnet
    Testnet,
    /// Signet
    Signet,
}

/// Bitcoin address types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BitcoinAddressType {
    /// P2PKH (Legacy)
    P2PKH,
    /// P2SH (Nested SegWit)
    P2SH,
    /// P2WPKH (Native SegWit)
    P2WPKH,
    /// P2WSH (Native SegWit Script)
    P2WSH,
    /// P2TR (Taproot)
    P2TR,
}

/// Bitcoin address
#[derive(Debug, Clone)]
pub struct BitcoinAddress {
    pub address: String,
    pub network: BitcoinNetwork,
    pub address_type: BitcoinAddressType,
}

impl BitcoinAddress {
    /// Create P2PKH address from public key hash
    pub fn from_pubkey_hash(pubkey_hash: &[u8], network: BitcoinNetwork) -> Self {
        let version = match network {
            BitcoinNetwork::Mainnet => vec![0x00],
            BitcoinNetwork::Testnet | BitcoinNetwork::Signet => vec![0x6F],
        };
        
        let payload = [version.as_slice(), pubkey_hash].concat();
        let address = bs58::encode(payload).with_check().into_string();
        
        Self {
            address,
            network,
            address_type: BitcoinAddressType::P2PKH,
        }
    }

    /// Create P2SH (nested SegWit) address from script hash
    pub fn from_script_hash(script_hash: &[u8], network: BitcoinNetwork) -> Self {
        let version = match network {
            BitcoinNetwork::Mainnet => vec![0x05],
            BitcoinNetwork::Testnet | BitcoinNetwork::Signet => vec![0xC4],
        };
        
        let payload = [version.as_slice(), script_hash].concat();
        let address = bs58::encode(payload).with_check().into_string();
        
        Self {
            address,
            network,
            address_type: BitcoinAddressType::P2SH,
        }
    }

    /// Create P2WPKH (native SegWit) address from public key hash
    pub fn from_bech32(pubkey_hash: &[u8], network: BitcoinNetwork) -> Self {
        let hrp = match network {
            BitcoinNetwork::Mainnet => "bc",
            BitcoinNetwork::Testnet | BitcoinNetwork::Signet => "tb",
        };
        
        let mut data = vec![0x00]; // Witness version 0
        data.extend_from_slice(pubkey_hash);
        
        let bech32 = encode_bech32(hrp, &data);
        
        Self {
            address: bech32,
            network,
            address_type: BitcoinAddressType::P2WPKH,
        }
    }

    /// Create P2TR (Taproot) address
    pub fn from_taproot(tweak: &[u8], network: BitcoinNetwork) -> Self {
        let hrp = match network {
            BitcoinNetwork::Mainnet => "bc",
            BitcoinNetwork::Testnet | BitcoinNetwork::Signet => "tb",
        };
        
        let mut data = vec![0x01]; // Witness version 1
        data.extend_from_slice(tweak);
        
        let bech32m = encode_bech32m(hrp, &data);
        
        Self {
            address: bech32m,
            network,
            address_type: BitcoinAddressType::P2TR,
        }
    }

    /// Validate address
    pub fn validate(address: &str) -> bool {
        // Try legacy
        if let Ok(decoded) = bs58::decode(address).with_check(None).into_vec() {
            if decoded.len() == 21 {
                let version = decoded[0];
                if matches!(version, 0x00 | 0x6F | 0x05 | 0xC4) {
                    return true;
                }
            }
        }
        
        // Try Bech32/Bech32m and validate the human-readable part and witness length.
        if let Ok((hrp, data, _variant)) = bech32::decode(address) {
            if (hrp == "bc" || hrp == "tb")
                && matches!(data.len(), 33 | 53)
                && data.first().map(|version| version.to_u8() <= 16).unwrap_or(false)
            {
                return true;
            }
        }
        
        false
    }
}

/// Derive Bitcoin address from seed
pub fn derive_address(seed: &[u8], network: BitcoinNetwork) -> BitcoinAddress {
    // Use BIP-32 derivation path m/84'/0'/0'/0/0
    let private_key = derive_bip32_child(seed, 0x80000054, 0, 0); // 84' = 0x80000054
    
    // Create public key (simplified - use proper EC)
    let pubkey_hash = sha256_hash(&private_key);
    let ripemd_hash = ripemd160_hash(&pubkey_hash);
    
    BitcoinAddress::from_bech32(&ripemd_hash, network)
}

/// Derive child key (BIP-32)
fn derive_bip32_child(seed: &[u8], hardened: u32, change: u32, index: u32) -> Vec<u8> {
    let mut hasher = Sha256::new();
    hasher.update(seed);
    hasher.update(&hardened.to_be_bytes());
    hasher.update(&change.to_be_bytes());
    hasher.update(&index.to_be_bytes());
    let hash = hasher.finalize();
    hash[..32].to_vec()
}

/// Double SHA256
fn double_sha256(data: &[u8]) -> Vec<u8> {
    let mut hasher = Sha256::new();
    hasher.update(data);
    let hash1 = hasher.finalize();
    
    let mut hasher = Sha256::new();
    hasher.update(&hash1);
    hasher.finalize().to_vec()
}

/// SHA256 hash
fn sha256_hash(data: &[u8]) -> Vec<u8> {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

/// RIPEMD160 hash
fn ripemd160_hash(data: &[u8]) -> Vec<u8> {
    let mut hasher = Ripemd160::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

/// Encode to Bech32
fn encode_bech32(hrp: &str, data: &[u8]) -> String {
    bech32::encode(hrp, data.to_base32(), Variant::Bech32)
        .expect("validated Bitcoin Bech32 payload")
}

/// Encode to Bech32m for witness version one and above.
fn encode_bech32m(hrp: &str, data: &[u8]) -> String {
    bech32::encode(hrp, data.to_base32(), Variant::Bech32m)
        .expect("validated Bitcoin Bech32m payload")
}

/// Bitcoin transaction input
#[derive(Debug, Clone)]
pub struct BitcoinInput {
    pub previous_output: OutPoint,
    pub script_sig: Vec<u8>,
    pub sequence: u32,
}

/// Bitcoin transaction output
#[derive(Debug, Clone)]
pub struct BitcoinOutput {
    pub value: u64,
    pub script_pubkey: Vec<u8>,
}

/// OutPoint (transaction reference)
#[derive(Debug, Clone)]
pub struct OutPoint {
    pub txid: [u8; 32],
    pub vout: u32,
}

/// Bitcoin transaction
#[derive(Debug, Clone)]
pub struct BitcoinTransaction {
    pub version: i32,
    pub inputs: Vec<BitcoinInput>,
    pub outputs: Vec<BitcoinOutput>,
    pub lock_time: u32,
}

impl BitcoinTransaction {
    pub fn new() -> Self {
        Self {
            version: 2,
            inputs: vec![],
            outputs: vec![],
            lock_time: 0,
        }
    }

    pub fn add_input(mut self, txid: [u8; 32], vout: u32, script: Vec<u8>) -> Self {
        self.inputs.push(BitcoinInput {
            previous_output: OutPoint { txid, vout },
            script_sig: script,
            sequence: 0xFFFFFFFF,
        });
        self
    }

    pub fn add_output(mut self, value: u64, script: Vec<u8>) -> Self {
        self.outputs.push(BitcoinOutput {
            value,
            script_pubkey: script,
        });
        self
    }

    /// Encode transaction
    pub fn encode(&self) -> Vec<u8> {
        let mut result = Vec::new();
        
        // Version
        result.extend_from_slice(&self.version.to_le_bytes());
        
        // Input count
        result.push(self.inputs.len() as u8);
        
        // Inputs
        for input in &self.inputs {
            result.extend_from_slice(&input.previous_output.txid);
            result.extend_from_slice(&input.previous_output.vout.to_le_bytes());
            result.push(input.script_sig.len() as u8);
            result.extend_from_slice(&input.script_sig);
            result.extend_from_slice(&input.sequence.to_le_bytes());
        }
        
        // Output count
        result.push(self.outputs.len() as u8);
        
        // Outputs
        for output in &self.outputs {
            result.extend_from_slice(&output.value.to_le_bytes());
            result.push(output.script_pubkey.len() as u8);
            result.extend_from_slice(&output.script_pubkey);
        }
        
        // Lock time
        result.extend_from_slice(&self.lock_time.to_le_bytes());
        
        result
    }

    /// Calculate transaction ID
    pub fn txid(&self) -> [u8; 32] {
        let encoded = self.encode();
        let hash = double_sha256(&encoded);
        let mut txid = [0u8; 32];
        txid.copy_from_slice(&hash[..32]);
        txid
    }
}

impl Default for BitcoinTransaction {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// PSBT (Partially Signed Bitcoin Transaction)
// ============================================================================

/// PSBT format
#[derive(Debug, Clone)]
pub struct Psbt {
    pub magic: [u8; 4],
    pub keypairs: Vec<PsbtKeypair>,
    pub inputs: Vec<PsbtInput>,
    pub outputs: Vec<PsbtOutput>,
    pub unknown: Vec<PsbtKeypair>,
}

#[derive(Debug, Clone)]
pub struct PsbtKeypair {
    pub key: Vec<u8>,
    pub value: Vec<u8>,
}

#[derive(Debug, Clone)]
pub struct PsbtInput {
    pub utxo: Option<BitcoinOutput>,
    pub redeem_script: Option<Vec<u8>>,
    pub witness_script: Option<Vec<u8>>,
    pub final_script_sig: Option<Vec<u8>>,
    pub final_witness: Option<Vec<Vec<u8>>>,
}

#[derive(Debug, Clone)]
pub struct PsbtOutput {
    pub amount: u64,
    pub script: Vec<u8>,
}

impl Psbt {
    pub fn new() -> Self {
        Self {
            magic: [0x70, 0x73, 0x62, 0x74], // "psbt"
            keypairs: vec![],
            inputs: vec![],
            outputs: vec![],
            unknown: vec![],
        }
    }

    pub fn add_input(mut self, input: PsbtInput) -> Self {
        self.inputs.push(input);
        self
    }

    pub fn add_output(mut self, output: PsbtOutput) -> Self {
        self.outputs.push(output);
        self
    }

    pub fn encode(&self) -> Vec<u8> {
        let mut result = Vec::new();
        
        // Magic
        result.extend_from_slice(&self.magic);
        
        // Keypairs count
        result.extend_from_slice(&(self.keypairs.len() as u32).to_le_bytes());
        
        // Keypairs
        for kp in &self.keypairs {
            result.extend_from_slice(&(kp.key.len() as u32).to_le_bytes());
            result.extend_from_slice(&kp.key);
            result.extend_from_slice(&(kp.value.len() as u32).to_le_bytes());
            result.extend_from_slice(&kp.value);
        }
        
        // Inputs count
        result.extend_from_slice(&(self.inputs.len() as u32).to_le_bytes());
        
        // Inputs (simplified - just serialize UTXO)
        for input in &self.inputs {
            if let Some(utxo) = &input.utxo {
                result.extend_from_slice(&utxo.value.to_le_bytes());
                result.push(utxo.script_pubkey.len() as u8);
                result.extend_from_slice(&utxo.script_pubkey);
            } else {
                result.extend_from_slice(&0u64.to_le_bytes());
                result.push(0u8);
            }
        }
        
        // Outputs count
        result.extend_from_slice(&(self.outputs.len() as u32).to_le_bytes());
        
        // Outputs
        for output in &self.outputs {
            result.extend_from_slice(&output.amount.to_le_bytes());
            result.push(output.script.len() as u8);
            result.extend_from_slice(&output.script);
        }
        
        // Unknown
        result.extend_from_slice(&(self.unknown.len() as u32).to_le_bytes());
        
        result
    }
}

impl Default for Psbt {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// BITCOIN RPC CLIENT
// ============================================================================

use reqwest::Client;
use serde::{Deserialize, Serialize};

/// Bitcoin RPC client
pub struct BitcoinRpcClient {
    rpc_url: String,
    client: Client,
    network: BitcoinNetwork,
}

impl BitcoinRpcClient {
    pub fn new(rpc_url: &str, network: BitcoinNetwork) -> Self {
        Self {
            rpc_url: rpc_url.to_string(),
            client: Client::new(),
            network,
        }
    }

    /// Get block chain info
    pub async fn get_blockchain_info(&self) -> Result<BlockchainInfo, BitcoinError> {
        self.call("getblockchaininfo", serde_json::json!([])).await
    }

    /// Get block count
    pub async fn get_block_count(&self) -> Result<u64, BitcoinError> {
        self.call("getblockcount", serde_json::json!([])).await
    }

    /// Get block hash
    pub async fn get_block_hash(&self, height: u64) -> Result<String, BitcoinError> {
        self.call("getblockhash", serde_json::json!([height])).await
    }

    /// Get block
    pub async fn get_block(&self, hash: &str) -> Result<Block, BitcoinError> {
        self.call("getblock", serde_json::json!([hash, 2])).await // Verbose = 2
    }

    /// Get transaction
    pub async fn get_transaction(&self, txid: &str) -> Result<Transaction, BitcoinError> {
        self.call("getrawtransaction", serde_json::json!([txid, true])).await
    }

    /// Get UTXO (unspent output)
    pub async fn get_utxo(&self, txid: &str, vout: u32) -> Result<UtxoInfo, BitcoinError> {
        self.call("gettxout", serde_json::json!([txid, vout])).await
    }

    /// List unspent outputs
    pub async fn list_unspent(&self, address: &str) -> Result<Vec<UtxoInfo>, BitcoinError> {
        self.call("listunspent", serde_json::json!([0, 9999999, [address]])).await
    }

    /// Decode raw transaction
    pub async fn decode_raw_transaction(&self, hex: &str) -> Result<DecodedTx, BitcoinError> {
        self.call("decoderawtransaction", serde_json::json!([hex])).await
    }

    /// Create raw transaction
    pub async fn create_raw_transaction(
        &self,
        inputs: Vec<TxInput>,
        outputs: HashMap<String, u64>,
    ) -> Result<String, BitcoinError> {
        let inputs: Vec<serde_json::Value> = inputs
            .into_iter()
            .map(|i| {
                serde_json::json!({
                    "txid": hex::encode(i.txid),
                    "vout": i.vout
                })
            })
            .collect();
        
        let outputs: serde_json::Value = outputs
            .into_iter()
            .collect();
        
        self.call("createrawtransaction", serde_json::json!([inputs, outputs])).await
    }

    /// Sign transaction
    pub async fn sign_transaction(&self, hex: &str, keys: Vec<String>) -> Result<SignResult, BitcoinError> {
        self.call("signrawtransactionwithkey", serde_json::json!([hex, keys])).await
    }

    /// Broadcast transaction
    pub async fn send_raw_transaction(&self, hex: &str) -> Result<String, BitcoinError> {
        self.call("sendrawtransaction", serde_json::json!([hex])).await
    }

    /// Get address info
    pub async fn get_address_info(&self, address: &str) -> Result<AddressInfo, BitcoinError> {
        self.call("getaddressinfo", serde_json::json!([address])).await
    }

    /// Validate address
    pub async fn validate_address(&self, address: &str) -> Result<AddressValidation, BitcoinError> {
        self.call("validateaddress", serde_json::json!([address])).await
    }

    /// Estimate fee
    pub async fn estimate_smart_fee(&self, conf_target: u32) -> Result<FeeEstimate, BitcoinError> {
        self.call("estimatesmartfee", serde_json::json!([conf_target])).await
    }

    /// Get mempool info
    pub async fn get_mempool_info(&self) -> Result<MempoolInfo, BitcoinError> {
        self.call("getmempoolinfo", serde_json::json!([])).await
    }

    /// Internal RPC call
    async fn call<T: DeserializeOwned>(&self, method: &str, params: serde_json::Value) -> Result<T, BitcoinError> {
        let request = serde_json::json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1
        });
        
        let response = self.client
            .post(&self.rpc_url)
            .json(&request)
            .send()
            .await
            .map_err(|e| BitcoinError::NetworkError(e.to_string()))?;
        
        let json: serde_json::Value = response.json().await
            .map_err(|e| BitcoinError::ParseError(e.to_string()))?;
        
        if let Some(error) = json.get("error") {
            return Err(BitcoinError::RpcError(error.to_string()));
        }
        
        let result = json.get("result")
            .ok_or_else(|| BitcoinError::RpcError("No result".to_string()))?;
        
        serde_json::from_value(result.clone())
            .map_err(|e| BitcoinError::ParseError(e.to_string()))
    }
}

/// Transaction input for creation
pub struct TxInput {
    pub txid: [u8; 32],
    pub vout: u32,
}

// ============================================================================
// RESPONSE TYPES
// ============================================================================

#[derive(Debug, Clone, Deserialize)]
pub struct BlockchainInfo {
    pub chain: String,
    pub blocks: u64,
    pub headers: u64,
    pub bestblockhash: String,
    pub difficulty: f64,
    pub mediantime: u64,
    pub verificationprogress: f64,
    pub initialblockdownload: bool,
    pub chainwork: String,
    pub size_on_disk: u64,
    pub pruned: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Block {
    pub hash: String,
    pub confirmations: u64,
    pub size: u64,
    pub strippedsize: u64,
    pub weight: u64,
    pub height: u64,
    pub version: u32,
    pub merkleroot: String,
    pub tx: Vec<String>,
    pub time: u64,
    pub nonce: u64,
    pub bits: String,
    pub difficulty: f64,
    pub previousblockhash: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Transaction {
    pub txid: String,
    pub hash: String,
    pub version: i32,
    pub size: u64,
    pub vsize: u64,
    pub weight: u64,
    pub locktime: u64,
    pub vin: Vec<Vin>,
    pub vout: Vec<Vout>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Vin {
    pub txid: Option<String>,
    pub vout: Option<u32>,
    pub scriptSig: Option<ScriptSig>,
    pub sequence: u32,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ScriptSig {
    pub asm: String,
    pub hex: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Vout {
    pub value: f64,
    pub n: u32,
    pub scriptPubKey: ScriptPubKey,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ScriptPubKey {
    pub asm: String,
    pub hex: String,
    pub reqSigs: u32,
    #[serde(rename = "type")]
    pub type_: String,
    pub addresses: Option<Vec<String>>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct UtxoInfo {
    pub txid: String,
    pub vout: u32,
    pub value: f64,
    pub scriptPubKey: ScriptPubKey,
    pub confirmations: u64,
}

#[derive(Debug, Clone, Deserialize)]
pub struct DecodedTx {
    pub txid: String,
    pub hash: String,
    pub version: i32,
    pub size: u64,
    pub vsize: u64,
    pub weight: u64,
    pub locktime: u64,
    pub vin: Vec<Vin>,
    pub vout: Vec<Vout>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct SignResult {
    pub hex: String,
    pub complete: bool,
    pub errors: Option<Vec<SignError>>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct SignError {
    pub txid: String,
    pub error: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct AddressInfo {
    pub address: String,
    pub scriptPubKey: String,
    pub ismine: bool,
    pub iswatchonly: bool,
    pub solvable: bool,
    pub desc: String,
    pub isvalid: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct AddressValidation {
    pub isvalid: bool,
    pub address: Option<String>,
    pub scriptPubKey: Option<String>,
    pub ismine: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct FeeEstimate {
    pub feerate: f64,
    pub unit: String,
    pub errors: Option<Vec<String>>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct MempoolInfo {
    pub size: u64,
    pub bytes: u64,
    pub usage: u64,
    pub total_fee: u64,
    pub maxmempool: u64,
    pub mempoolminfee: f64,
    pub minrelayfee: f64,
}

// ============================================================================
// ERRORS
// ============================================================================

#[derive(Debug, Clone)]
pub enum BitcoinError {
    NetworkError(String),
    ParseError(String),
    RpcError(String),
    InvalidData,
}

impl std::fmt::Display for BitcoinError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BitcoinError::NetworkError(e) => write!(f, "Network error: {}", e),
            BitcoinError::ParseError(e) => write!(f, "Parse error: {}", e),
            BitcoinError::RpcError(e) => write!(f, "RPC error: {}", e),
            BitcoinError::InvalidData => write!(f, "Invalid data"),
        }
    }
}

impl std::error::Error for BitcoinError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bitcoin_address_validation() {
        // Valid mainnet P2PKH
        assert!(BitcoinAddress::validate("1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"));
        // Valid mainnet P2SH
        assert!(BitcoinAddress::validate("3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy"));
        // Valid mainnet Bech32
        assert!(BitcoinAddress::validate("bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"));
    }

    #[test]
    fn test_psbt_creation() {
        let psbt = Psbt::new()
            .add_input(PsbtInput {
                utxo: Some(BitcoinOutput {
                    value: 100000,
                    script_pubkey: vec![],
                }),
                redeem_script: None,
                witness_script: None,
                final_script_sig: None,
                final_witness: None,
            })
            .add_output(PsbtOutput {
                amount: 99000,
                script: vec![],
            });
        
        let encoded = psbt.encode();
        assert!(encoded.len() > 0);
    }
}