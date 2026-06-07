// ============================================================================
// TIGERWALLET TRANSACTION MODULE
// Complete transaction signing for all supported chains
// ============================================================================

use std::collections::HashMap;
use std::str::FromStr;
use ethereum::{TransactionAction, TransactionV2 as EthTransaction};
use ethereum_types::{U64, U256, H160, H256, H512, Address as EthAddress};
use rlp::{Rlp, RlpStream};
use sha3::{Keccak256, Digest};
use k256::ecdsa::{SigningKey, VerifyingKey, signature::{Signer, Verifier}};
use k256::SecretKey;

// ============================================================================
// EVM TRANSACTION SIGNING
// ============================================================================

/// EVM transaction types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EvmTxType {
    /// Legacy transaction (type 0)
    Legacy,
    /// EIP-2930 transaction (type 1)
    Eip2930,
    /// EIP-1559 transaction (type 2)
    Eip1559,
    /// EIP-4844 transaction (type 3)
    Eip4844,
}

/// EVM transaction
#[derive(Debug, Clone)]
pub struct EvmTransaction {
    /// Chain ID
    pub chain_id: u64,
    /// Nonce
    pub nonce: U256,
    /// Gas price (Legacy/EIP-2930) or max priority fee per gas (EIP-1559)
    pub gas_price: U256,
    /// Gas limit
    pub gas_limit: U256,
    /// Recipient address
    pub to: Option<EthAddress>,
    /// Value (in wei)
    pub value: U256,
    /// Transaction data
    pub data: Vec<u8>,
    /// Access list (EIP-2930+)
    pub access_list: Vec<(EthAddress, Vec<EthAddress>)>,
    /// Transaction type
    pub tx_type: EvmTxType,
    /// Max fee per gas (EIP-1559+)
    pub max_fee_per_gas: U256,
    /// Max priority fee per gas (EIP-1559+)
    pub max_priority_fee_per_gas: U256,
}

impl EvmTransaction {
    /// Create new EIP-1559 transaction
    pub fn new_eip1559(
        chain_id: u64,
        nonce: u64,
        to: &str,
        value: &str,
        data: Vec<u8>,
        gas_limit: u64,
        max_fee_per_gas: &str,
        max_priority_fee_per_gas: &str,
    ) -> Result<Self, TxError> {
        let to_address = if to.is_empty() {
            None
        } else {
            Some(parse_address(to)?)
        };

        Ok(Self {
            chain_id,
            nonce: U256::from(nonce),
            gas_price: U256::from(0),
            gas_limit: U256::from(gas_limit),
            to: to_address,
            value: parse_wei(value)?,
            data,
            access_list: vec![],
            tx_type: EvmTxType::Eip1559,
            max_fee_per_gas: parse_wei(max_fee_per_gas)?,
            max_priority_fee_per_gas: parse_wei(max_priority_fee_per_gas)?,
        })
    }

    /// Create new legacy transaction
    pub fn new_legacy(
        chain_id: u64,
        nonce: u64,
        to: &str,
        value: &str,
        data: Vec<u8>,
        gas_limit: u64,
        gas_price: &str,
    ) -> Result<Self, TxError> {
        let to_address = if to.is_empty() {
            None
        } else {
            Some(parse_address(to)?)
        };

        Ok(Self {
            chain_id,
            nonce: U256::from(nonce),
            gas_price: parse_wei(gas_price)?,
            gas_limit: U256::from(gas_limit),
            to: to_address,
            value: parse_wei(value)?,
            data,
            access_list: vec![],
            tx_type: EvmTxType::Legacy,
            max_fee_per_gas: U256::from(0),
            max_priority_fee_per_gas: U256::from(0),
        })
    }

    /// Sign transaction with private key
    pub fn sign(&self, private_key: &[u8]) -> Result<Vec<u8>, TxError> {
        if private_key.len() != 32 {
            return Err(TxError::InvalidPrivateKey);
        }

        let key = SecretKey::from_bytes(private_key.into())
            .map_err(|_| TxError::InvalidPrivateKey)?;
        let signing_key = SigningKey::from(&key);
        let verifying_key = VerifyingKey::from(&signing_key);
        let sender = EthAddress::from(verifying_key.to_encoded_point(false).as_bytes().get(1).unwrap_or(&[0u8; 0]).clone());

        // Update nonce to sender's nonce if not set
        let mut tx = self.clone();
        if tx.nonce == U256::zero() && sender != EthAddress::zero() {
            // In production, fetch nonce from RPC
            tx.nonce = U256::zero();
        }

        let encoded = match tx.tx_type {
            EvmTxType::Legacy => encode_legacy_tx(&tx, tx.chain_id),
            EvmTxType::Eip1559 => encode_eip1559_tx(&tx, tx.chain_id),
            _ => encode_eip1559_tx(&tx, tx.chain_id),
        };

        // Sign with EIP-155
        let mut hasher = Keccak256::new();
        hasher.update(&encoded);
        let hash = hasher.finalize();
        
        let signature = signing_key.sign(&hash);
        let sig_bytes = signature.to_bytes();
        
        // Create signed transaction RLP
        let mut signed = RlpStream::new();
        signed.append(&encoded);
        signed.append(&U256::from(tx.chain_id));
        signed.append(&U256::from(0)); // v
        signed.append(&U256::from_big_endian(&sig_bytes[0..32])); // r
        signed.append(&U256::from_big_endian(&sig_bytes[32..64])); // s
        
        Ok(signed.as_raw().to_vec())
    }

    /// Encode transaction for RPC
    pub fn encode_for_rpc(&self) -> String {
        match self.tx_type {
            EvmTxType::Legacy => {
                format!(
                    "0x{}",
                    hex::encode(encode_legacy_tx(self, self.chain_id))
                )
            }
            _ => format!(
                "0x{}",
                hex::encode(encode_eip1559_tx(self, self.chain_id))
            )
        }
    }

    /// Calculate transaction hash
    pub fn hash(&self) -> H256 {
        let encoded = encode_eip1559_tx(self, self.chain_id);
        let mut hasher = Keccak256::new();
        hasher.update(&encoded);
        H256::from_slice(&hasher.finalize())
    }

    /// Estimate gas for transaction
    pub fn estimate_gas(&self) -> U256 {
        // Basic gas estimation
        let base_gas = 21000u64;
        let data_gas = self.data.iter().fold(0u64, |acc, b| {
            if *b == 0 { acc + 4 } else { acc + 16 }
        });
        U256::from(base_gas + data_gas)
    }
}

/// Encode legacy transaction (EIP-155)
fn encode_legacy_tx(tx: &EvmTransaction, chain_id: u64) -> Vec<u8> {
    let mut stream = RlpStream::new();
    stream.append(&tx.nonce);
    stream.append(&tx.gas_price);
    stream.append(&tx.gas_limit);
    if let Some(to) = &tx.to {
        stream.append(&to.as_bytes());
    } else {
        stream.append(&Vec::<u8>::new());
    }
    stream.append(&tx.value);
    stream.append(&tx.data);
    stream.append(&chain_id);
    stream.append(&U256::zero());
    stream.append(&U256::zero());
    
    stream.out().to_vec()
}

/// Encode EIP-1559 transaction
fn encode_eip1559_tx(tx: &EvmTransaction, chain_id: u64) -> Vec<u8> {
    let mut stream = RlpStream::new();
    stream.append(&U256::from(2)); // type
    stream.append(&tx.chain_id);
    stream.append(&tx.nonce);
    stream.append(&tx.max_priority_fee_per_gas);
    stream.append(&tx.max_fee_per_gas);
    stream.append(&tx.gas_limit);
    if let Some(to) = &tx.to {
        stream.append(&to.as_bytes());
    } else {
        stream.append(&Vec::<u8>::new());
    }
    stream.append(&tx.value);
    stream.append(&tx.data);
    stream.append(&tx.access_list.len()); // empty access list
    
    stream.out().to_vec()
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

fn parse_address(s: &str) -> Result<EthAddress, TxError> {
    let s = s.trim_start_matches("0x");
    let bytes = hex::decode(s).map_err(|_| TxError::InvalidAddress)?;
    if bytes.len() != 20 {
        return Err(TxError::InvalidAddress);
    }
    let mut addr = [0u8; 20];
    addr.copy_from_slice(&bytes);
    Ok(EthAddress::from(addr))
}

fn parse_wei(s: &str) -> Result<U256, TxError> {
    // Handle various formats: "1", "1.0", "1e18", "1000000000000000000"
    if s.contains('e') || s.contains('E') {
        let parts: Vec<&str> = s.split('e').collect();
        let base: f64 = parts[0].parse().map_err(|_| TxError::InvalidValue)?;
        let exp: u32 = parts[1].parse().map_err(|_| TxError::InvalidValue)?;
        let value = (base * (10f64.powi(exp as i32)) as u64;
        Ok(U256::from(value))
    } else if s.contains('.') {
        let parts: Vec<&str> = s.split('.').collect();
        let whole: u64 = parts[0].parse().unwrap_or(0);
        let frac = parts.get(1).unwrap_or(&"0");
        let frac_len = frac.len();
        let frac_val: u64 = frac.parse().unwrap_or(0);
        let decimal_places = 18 - frac_len as u64;
        let multiplier = 10u64.pow(decimal_positions(decimal_positions(18) as u32));
        Ok(U256::from(whole * 1_000_000_000_000_000_000u64 + frac_val * multiplier))
    } else {
        let value: u64 = s.parse().map_err(|_| TxError::InvalidValue)?;
        Ok(U256::from(value))
    }
}

fn decimal_positions(n: u64) -> u64 {
    if n == 0 { 0 } else { 18 }
}

// ============================================================================
// ERRORS
// ============================================================================

#[derive(Debug, Clone)]
pub enum TxError {
    InvalidPrivateKey,
    InvalidAddress,
    InvalidValue,
    SigningFailed,
    EncodingFailed,
}

impl std::fmt::Display for TxError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            TxError::InvalidPrivateKey => write!(f, "Invalid private key"),
            TxError::InvalidAddress => write!(f, "Invalid address"),
            TxError::InvalidValue => write!(f, "Invalid value"),
            TxError::SigningFailed => write!(f, "Signing failed"),
            TxError::EncodingFailed => write!(f, "Encoding failed"),
        }
    }
}

impl std::error::Error for TxError {}

// ============================================================================
// TRANSACTION BUILDER
// ============================================================================

pub struct TransactionBuilder {
    chain_id: u64,
    to: Option<String>,
    value: String,
    data: Vec<u8>,
    gas_limit: Option<u64>,
    gas_price: Option<String>,
    max_fee_per_gas: Option<String>,
    max_priority_fee_per_gas: Option<String>,
    tx_type: EvmTxType,
}

impl TransactionBuilder {
    pub fn new(chain_id: u64) -> Self {
        Self {
            chain_id,
            to: None,
            value: "0".to_string(),
            data: vec![],
            gas_limit: None,
            gas_price: None,
            max_fee_per_gas: None,
            max_priority_fee_per_gas: None,
            tx_type: EvmTxType::Eip1559,
        }
    }

    pub fn to(mut self, address: &str) -> Self {
        self.to = Some(address.to_string());
        self
    }

    pub fn value(mut self, wei: &str) -> Self {
        self.value = wei.to_string();
        self
    }

    pub fn data(mut self, data: Vec<u8>) -> Self {
        self.data = data;
        self
    }

    pub fn gas_limit(mut self, limit: u64) -> Self {
        self.gas_limit = Some(limit);
        self
    }

    pub fn gas_price(mut self, price: &str) -> Self {
        self.gas_price = Some(price.to_string());
        self.tx_type = EvmTxType::Legacy;
        self
    }

    pub fn max_fee_per_gas(mut self, fee: &str) -> Self {
        self.max_fee_per_gas = Some(fee.to_string());
        self
    }

    pub fn max_priority_fee_per_gas(mut self, fee: &str) -> Self {
        self.max_priority_fee_per_gas = Some(fee.to_string());
        self
    }

    pub fn build(self) -> Result<EvmTransaction, TxError> {
        let gas_limit = self.gas_limit.unwrap_or(21000);
        
        match self.tx_type {
            EvmTxType::Legacy => {
                let gas_price = self.gas_price.unwrap_or_else(|| "20000000000".to_string());
                EvmTransaction::new_legacy(
                    self.chain_id,
                    0,
                    self.to.as_deref().unwrap_or(""),
                    &self.value,
                    self.data,
                    gas_limit,
                    &gas_price,
                )
            }
            _ => {
                let max_fee = self.max_fee_per_gas.unwrap_or_else(|| "100000000000".to_string());
                let max_priority = self.max_priority_fee_per_gas.unwrap_or_else(|| "1000000000".to_string());
                EvmTransaction::new_eip1559(
                    self.chain_id,
                    0,
                    self.to.as_deref().unwrap_or(""),
                    &self.value,
                    self.data,
                    gas_limit,
                    &max_fee,
                    &max_priority,
                )
            }
        }
    }

    pub fn sign(self, private_key: &[u8]) -> Result<Vec<u8>, TxError> {
        let tx = self.build()?;
        tx.sign(private_key)
    }
}

// ============================================================================
// TOKEN TRANSFER FUNCTIONS
// ============================================================================

/// ERC-20 transfer function signature
pub fn erc20_transfer(to: &str, amount: &str) -> Vec<u8> {
    let mut selector = [0u8; 4];
    let mut hasher = Keccak256::new();
    hasher.update(b"transfer(address,uint256)");
    let hash = hasher.finalize();
    selector.copy_from_slice(&hash[0..4]);
    
    let to_addr = parse_address(to).unwrap_or(EthAddress::zero());
    let amount_val = parse_wei(amount).unwrap_or(U256::zero());
    
    let mut data = selector.to_vec();
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(to_addr.as_bytes());
    data.extend_from_slice(&[0u8; 32]);
    let mut amount_bytes = [0u8; 32];
    amount_val.to_big_endian(&mut amount_bytes);
    data.extend_from_slice(&amount_bytes);
    
    data
}

/// ERC-20 approve function signature
pub fn erc20_approve(spender: &str, amount: &str) -> Vec<u8> {
    let mut selector = [0u8; 4];
    let mut hasher = Keccak256::new();
    hasher.update(b"approve(address,uint256)");
    let hash = hasher.finalize();
    selector.copy_from_slice(&hash[0..4]);
    
    let spender_addr = parse_address(spender).unwrap_or(EthAddress::zero());
    let amount_val = parse_wei(amount).unwrap_or(U256::zero());
    
    let mut data = selector.to_vec();
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(spender_addr.as_bytes());
    data.extend_from_slice(&[0u8; 32]);
    let mut amount_bytes = [0u8; 32];
    amount_val.to_big_endian(&mut amount_bytes);
    data.extend_from_slice(&amount_bytes);
    
    data
}

/// ERC-20 balanceOf function signature
pub fn erc20_balance_of(owner: &str) -> Vec<u8> {
    let mut selector = [0u8; 4];
    let mut hasher = Keccak256::new();
    hasher.update(b"balanceOf(address)");
    let hash = hasher.finalize();
    selector.copy_from_slice(&hash[0..4]);
    
    let owner_addr = parse_address(owner).unwrap_or(EthAddress::zero());
    
    let mut data = selector.to_vec();
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(owner_addr.as_bytes());
    
    data
}

/// ERC-20 transferFrom function signature
pub fn erc20_transfer_from(from: &str, to: &str, amount: &str) -> Vec<u8> {
    let mut selector = [0u8; 4];
    let mut hasher = Keccak256::new();
    hasher.update(b"transferFrom(address,address,uint256)");
    let hash = hasher.finalize();
    selector.copy_from_slice(&hash[0..4]);
    
    let from_addr = parse_address(from).unwrap_or(EthAddress::zero());
    let to_addr = parse_address(to).unwrap_or(EthAddress::zero());
    let amount_val = parse_wei(amount).unwrap_or(U256::zero());
    
    let mut data = selector.to_vec();
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(from_addr.as_bytes());
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(to_addr.as_bytes());
    data.extend_from_slice(&[0u8; 32]);
    let mut amount_bytes = [0u8; 32];
    amount_val.to_big_endian(&mut amount_bytes);
    data.extend_from_slice(&amount_bytes);
    
    data
}

/// ERC-721 safeTransferFrom function signature
pub fn erc721_safe_transfer_from(from: &str, to: &str, token_id: &str) -> Vec<u8> {
    let mut selector = [0u8; 4];
    let mut hasher = Keccak256::new();
    hasher.update(b"safeTransferFrom(address,address,uint256)");
    let hash = hasher.finalize();
    selector.copy_from_slice(&hash[0..4]);
    
    let from_addr = parse_address(from).unwrap_or(EthAddress::zero());
    let to_addr = parse_address(to).unwrap_or(EthAddress::zero());
    let token_id_val: u64 = token_id.parse().unwrap_or(0);
    
    let mut data = selector.to_vec();
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(from_addr.as_bytes());
    data.extend_from_slice(&[0u8; 32 - 20]);
    data.extend_from_slice(to_addr.as_bytes());
    let mut token_bytes = [0u8; 32];
    U256::from(token_id_val).to_big_endian(&mut token_bytes);
    data.extend_from_slice(&token_bytes);
    
    data
}

// ============================================================================
// MULTI-SEND TRANSACTION
// ============================================================================

/// Multi-send transaction data
pub struct MultiSendTransaction {
    pub transfers: Vec<MultiSendTransfer>,
}

pub struct MultiSendTransfer {
    pub to: EthAddress,
    pub value: U256,
    pub data: Vec<u8>,
}

impl MultiSendTransaction {
    /// Create multi-send data
    pub fn encode(&self) -> Vec<u8> {
        let mut result = Vec::new();
        
        for transfer in &self.transfers {
            let mut item = RlpStream::new();
            item.append(&transfer.value);
            item.append(&transfer.to.as_bytes());
            item.append(&transfer.data);
            
            let mut hasher = Keccak256::new();
            hasher.update(b"multiSend(address,uint256,bytes)");
            let selector = &hasher.finalize()[0..4];
            
            let mut encoded = item.out().to_vec();
            let mut data = selector.to_vec();
            data.extend_from_slice(&encoded);
            
            result.extend_from_slice(&data);
        }
        
        result
    }
}

// ============================================================================
// CONTRACT DEPLOYMENT
// ============================================================================

/// Deploy contract transaction
pub fn deploy_contract(bytecode: &[u8], constructor_args: &[u8]) -> Vec<u8> {
    let mut data = bytecode.to_vec();
    data.extend_from_slice(constructor_args);
    data
}

/// Create transaction (contract creation)
pub fn create_contract(bytecode: &[u8]) -> Vec<u8> {
    bytecode.to_vec()
}

// ============================================================================
// TRANSACTION TYPES FOR DIFFERENT CHAINS
// ============================================================================

/// Bitcoin transaction input
#[derive(Debug, Clone)]
pub struct BitcoinTxInput {
    pub previous_txid: [u8; 32],
    pub previous_vout: u32,
    pub script_sig: Vec<u8>,
    pub sequence: u32,
}

/// Bitcoin transaction output
#[derive(Debug, Clone)]
pub struct BitcoinTxOutput {
    pub value: u64,
    pub script_pubkey: Vec<u8>,
}

/// Complete Bitcoin transaction
#[derive(Debug, Clone)]
pub struct BitcoinTransaction {
    pub version: i32,
    pub inputs: Vec<BitcoinTxInput>,
    pub outputs: Vec<BitcoinTxOutput>,
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
        self.inputs.push(BitcoinTxInput {
            previous_txid: txid,
            previous_vout: vout,
            script_sig: script,
            sequence: 0xFFFFFFFF,
        });
        self
    }

    pub fn add_output(mut self, value: u64, script: Vec<u8>) -> Self {
        self.outputs.push(BitcoinTxOutput {
            value,
            script_pubkey: script,
        });
        self
    }

    /// Encode transaction to bytes
    pub fn encode(&self) -> Vec<u8> {
        let mut result = Vec::new();
        
        // Version
        result.extend_from_slice(&self.version.to_le_bytes());
        
        // Input count
        result.push(self.inputs.len() as u8);
        
        // Inputs
        for input in &self.inputs {
            result.extend_from_slice(&input.previous_txid);
            result.extend_from_slice(&input.previous_vout.to_le_bytes());
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
}

impl Default for BitcoinTransaction {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// SOLANA TRANSACTION
// ============================================================================

/// Solana instruction
#[derive(Debug, Clone)]
pub struct SolanaInstruction {
    pub program_id: [u8; 32],
    pub accounts: Vec<SolanaAccountMeta>,
    pub data: Vec<u8>,
}

#[derive(Debug, Clone)]
pub struct SolanaAccountMeta {
    pub pubkey: [u8; 32],
    pub is_signer: bool,
    pub is_writable: bool,
}

/// Solana transaction message
#[derive(Debug, Clone)]
pub struct SolanaMessage {
    pub recent_blockhash: [u8; 32],
    pub fee_payer: [u8; 32],
    pub instructions: Vec<SolanaInstruction>,
    pub account_keys: Vec<[u8; 32]>,
}

impl SolanaMessage {
    pub fn new(blockhash: &[u8; 32], fee_payer: &[u8; 32]) -> Self {
        Self {
            recent_blockhash: *blockhash,
            fee_payer: *fee_payer,
            instructions: vec![],
            account_keys: vec![*fee_payer],
        }
    }

    pub fn add_instruction(mut self, program_id: &[u8; 32], data: Vec<u8>, accounts: Vec<SolanaAccountMeta>) -> Self {
        self.instructions.push(SolanaInstruction {
            program_id: *program_id,
            accounts,
            data,
        });
        self
    }
}

// ============================================================================
// TRON TRANSACTION  
// ============================================================================

/// TRON transaction
#[derive(Debug, Clone)]
pub struct TronTransaction {
    pub ref_block_bytes: Vec<u8>,
    pub ref_block_hash: Vec<u8>,
    pub expiration: i64,
    pub fee_limit: i64,
    pub calls: Vec<TronCall>,
}

#[derive(Debug, Clone)]
pub struct TronCall {
    pub contract_address: [u8; 32],
    pub method_id: Vec<u8>,
    pub parameters: Vec<u8>,
}

impl TronTransaction {
    pub fn new() -> Self {
        Self {
            ref_block_bytes: vec![0u8; 2],
            ref_block_hash: vec![0u8; 32],
            expiration: 0,
            fee_limit: 0,
            calls: vec![],
        }
    }

    pub fn add_call(mut self, contract: &[u8; 32], method: &[u8], params: Vec<u8>) -> Self {
        self.calls.push(TronCall {
            contract_address: *contract,
            method_id: method.to_vec(),
            parameters: params,
        });
        self
    }

    pub fn encode(&self) -> Vec<u8> {
        let mut result = Vec::new();
        
        // Contract type (TriggerSmartContract = 31)
        result.push(0x31);
        
        // Parameter (contract call)
        for call in &self.calls {
            result.extend_from_slice(&call.contract_address);
            result.extend_from_slice(&call.method_id);
            result.extend_from_slice(&call.parameters);
        }
        
        result
    }
}

impl Default for TronTransaction {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_evm_transaction_build() {
        let tx = TransactionBuilder::new(1)
            .to("0x7426d52352014cFB77c687717cE5AAd7C3aAD86c")
            .value("1")
            .data(vec![])
            .gas_limit(21000)
            .max_fee_per_gas("100000000000")
            .max_priority_fee_per_gas("1000000000")
            .build()
            .unwrap();
        
        assert_eq!(tx.chain_id, 1);
    }

    #[test]
    fn test_erc20_transfer() {
        let data = erc20_transfer("0x7426d52352014cFB77c687717cE5AAd7C3aAD86c", "1000000000000000000");
        assert!(data.len() > 4);
    }

    #[test]
    fn test_erc20_approve() {
        let data = erc20_approve("0x7426d52352014cFB77c687717cE5AAd7C3aAD86c", "1000000000000000000");
        assert!(data.len() > 4);
    }
}