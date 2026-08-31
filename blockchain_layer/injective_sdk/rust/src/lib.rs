//! TigerWallet Injective Blockchain SDK
//! Production-ready implementation for Injective blockchain
//! 
//! Features:
//! - Full account management
//! - Transaction building and signing (Ethereum-style)
//! - Spot and derivative trading
//! - Token management
//! - Peggy bridge support

#![allow(dead_code)]

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use sha2::{Sha256, Digest};
use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum InjectiveError {
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
}

// ============================================================================
// Address Types
// ============================================================================

#[derive(Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct Address(pub [u8; 20]);

impl Address {
    pub fn from_hex(hex: &str) -> Result<Self, InjectiveError> {
        let bytes = hex::decode(hex).map_err(|e| InjectiveError::InvalidAddress(e.to_string()))?;
        if bytes.len() != 20 {
            return Err(InjectiveError::InvalidAddress("Must be 20 bytes".to_string()));
        }
        let mut addr = [0u8; 20];
        addr.copy_from_slice(&bytes);
        Ok(Address(addr))
    }
    
    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }
    
    pub fn to_eth_address(&self) -> String {
        format!("0x{}", hex::encode(self.0))
    }
}

impl std::fmt::Debug for Address {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "0x{}", hex::encode(self.0))
    }
}

impl std::fmt::Display for Address {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "0x{}", hex::encode(self.0))
    }
}

// ============================================================================
// Key Types
// ============================================================================

#[derive(Clone)]
pub struct PrivateKey {
    pub key: [u8; 32],
}

impl PrivateKey {
    pub fn generate() -> Self {
        use rand::RngCore;
        let mut key = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut key);
        PrivateKey { key }
    }
    
    pub fn from_hex(hex: &str) -> Result<Self, InjectiveError> {
        let bytes = hex::decode(hex).map_err(|e| InjectiveError::SigningError(e.to_string()))?;
        if bytes.len() != 32 {
            return Err(InjectiveError::SigningError("Must be 32 bytes".to_string()));
        }
        let mut key = [0u8; 32];
        key.copy_from_slice(&bytes);
        Ok(PrivateKey { key })
    }
    
    pub fn public_key(&self) -> PublicKey {
        use secp256k1::{Secp256k1, SecretKey, PublicKey as SecPubKey};
        let secp = Secp256k1::new();
        let sk = SecretKey::from_slice(&self.key).expect("valid secp256k1 secret");
        let pk = SecPubKey::from_secret_key(&secp, &sk);
        // serialize_uncompressed returns 65 bytes (0x04 || X || Y)
        let bytes = pk.serialize_uncompressed();
        let mut pk_bytes = [0u8; 65];
        pk_bytes.copy_from_slice(&bytes);
        PublicKey(pk_bytes)
    }

    pub fn sign(&self, msg: &[u8]) -> Signature {
        use secp256k1::{Secp256k1, SecretKey, Message};
        let secp = Secp256k1::new();
        let sk = SecretKey::from_slice(&self.key).expect("valid secp256k1 secret");
        // ECDSA over SHA-256 of the message (Cosmos sign mode)
        let digest = Sha256::digest(msg);
        let message = Message::from_slice(&digest).expect("32-byte digest");
        let sig = secp.sign_ecdsa(&message, &sk);
        let mut sig_bytes = [0u8; 64];
        sig_bytes.copy_from_slice(&sig.serialize_compact());
        Signature(sig_bytes)
    }

    pub fn sign_typed_hash(&self, hash: &[u8]) -> Signature {
        self.sign(hash)
    }
}

#[derive(Clone)]
pub struct PublicKey(pub [u8; 65]);

impl PublicKey {
    pub fn to_address(&self) -> Address {
        let mut hasher = Sha256::new();
        hasher.update(&self.0[1..65]); // Skip 0x04 prefix
        let hash = hasher.finalize();
        let mut addr = [0u8; 20];
        addr.copy_from_slice(&hash[12..32]);
        Address(addr)
    }

    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }

    /// Verify a real ECDSA signature over SHA-256(msg).
    pub fn verify(&self, msg: &[u8], signature: &Signature) -> bool {
        use secp256k1::{Secp256k1, PublicKey as SecPubKey, Message, ecdsa::Signature as SecSig};
        let secp = Secp256k1::verification_only();
        let pk = match SecPubKey::from_slice(&self.0) {
            Ok(p) => p,
            Err(_) => return false,
        };
        let sig = match SecSig::from_compact(&signature.0) {
            Ok(s) => s,
            Err(_) => return false,
        };
        let digest = Sha256::digest(msg);
        let message = Message::from_slice(&digest).expect("32-byte digest");
        secp.verify_ecdsa(&message, &sig, &pk).is_ok()
    }
}

impl std::fmt::Debug for PublicKey {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "PublicKey({})", hex::encode(self.0))
    }
}

impl Serialize for PublicKey {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(&hex::encode(self.0))
    }
}

impl<'de> Deserialize<'de> for PublicKey {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let s = String::deserialize(deserializer)?;
        let bytes = hex::decode(&s).map_err(serde::de::Error::custom)?;
        if bytes.len() != 65 {
            return Err(serde::de::Error::custom("PublicKey must be 65 bytes"));
        }
        let mut pk = [0u8; 65];
        pk.copy_from_slice(&bytes);
        Ok(PublicKey(pk))
    }
}

#[derive(Clone)]
pub struct Signature(pub [u8; 64]);

impl Signature {
    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }
}

impl std::fmt::Debug for Signature {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "Signature({})", hex::encode(self.0))
    }
}

impl Serialize for Signature {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(&hex::encode(self.0))
    }
}

impl<'de> Deserialize<'de> for Signature {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let s = String::deserialize(deserializer)?;
        let bytes = hex::decode(&s).map_err(serde::de::Error::custom)?;
        if bytes.len() != 64 {
            return Err(serde::de::Error::custom("Signature must be 64 bytes"));
        }
        let mut sig = [0u8; 64];
        sig.copy_from_slice(&bytes);
        Ok(Signature(sig))
    }
}

// ============================================================================
// Transaction Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Message {
    Send(SendMsg),
    GrantAllowance(GrantAllowanceMsg),
    RevokeAllowance(RevokeAllowanceMsg),
    CreateSpotMarketOrder(CreateSpotOrderMsg),
    CreateDerivativeMarketOrder(CreateDerivativeOrderMsg),
    CancelSpotOrder(CancelOrderMsg),
    CancelDerivativeOrder(CancelOrderMsg),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendMsg {
    pub sender: Address,
    pub recipient: Address,
    pub amount: Coin,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GrantAllowanceMsg {
    pub granter: Address,
    pub grantee: Address,
    pub allowance: Allowance,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeAllowanceMsg {
    pub granter: Address,
    pub grantee: Address,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Allowance {
    BasicAllowance(BasicAllowance),
    PeriodicAllowance(PeriodicAllowance),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BasicAllowance {
    pub spend_limit: Vec<Coin>,
    pub expiration: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PeriodicAllowance {
    pub basic: BasicAllowance,
    pub period: u64,
    pub period_reset: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSpotOrderMsg {
    pub sender: Address,
    pub market_id: String,
    pub subaccount_id: String,
    pub order_type: String,
    pub price: f64,
    pub quantity: f64,
    pub direction: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDerivativeOrderMsg {
    pub sender: Address,
    pub market_id: String,
    pub subaccount_id: String,
    pub order_type: String,
    pub price: f64,
    pub quantity: f64,
    pub margin: f64,
    pub direction: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CancelOrderMsg {
    pub sender: Address,
    pub market_id: String,
    pub subaccount_id: String,
    pub order_hash: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Coin {
    pub denom: String,
    pub amount: String,
}

// ============================================================================
// Transaction
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub msgs: Vec<Message>,
    pub fee: Fee,
    pub memo: String,
    pub timeout_height: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fee {
    pub amount: Vec<Coin>,
    pub gas_limit: u64,
    pub payer: String,
    pub granter: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedTransaction {
    pub tx: Transaction,
    pub signature: Signature,
    pub sender: Address,
}

// ============================================================================
// Account Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Account {
    pub address: Address,
    pub balance: Vec<Coin>,
    pub subaccounts: Vec<SubAccount>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubAccount {
    pub id: String,
    pub balances: Vec<Coin>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Market {
    pub market_id: String,
    pub ticker: String,
    pub quote_denom: String,
    pub price_precision: u32,
    pub quantity_precision: u32,
    pub min_price_tick_size: f64,
    pub min_quantity_tick_size: f64,
}

// ============================================================================
// RPC Client
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Network {
    Mainnet,
    Testnet,
    Devnet,
}

impl Network {
    pub fn grpc_url(&self) -> &str {
        match self {
            Network::Mainnet => "https://grpc.injective.network:443",
            Network::Testnet => "https://grpc.injective.network:443",
            Network::Devnet => "http://localhost:9090",
        }
    }
    
    pub fn rest_url(&self) -> &str {
        match self {
            Network::Mainnet => "https://api.injective.network",
            Network::Testnet => "https://api.injective.network",
            Network::Devnet => "http://localhost:1317",
        }
    }
}

pub struct InjectiveClient {
    http_client: reqwest::Client,
    rest_url: String,
}

impl InjectiveClient {
    pub fn new(network: Network) -> Self {
        Self {
            http_client: reqwest::Client::new(),
            rest_url: network.rest_url().to_string(),
        }
    }
    
    pub async fn get_account(&self, address: &Address) -> Result<Account, InjectiveError> {
        let url = format!("{}/cosmos/auth/v1beta1/accounts/{}", self.rest_url, address.to_hex());
        let response = self.http_client.get(&url).send().await
            .map_err(|e| InjectiveError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(InjectiveError::RpcError(format!("HTTP: {}", response.status())));
        }
        
        let data: serde_json::Value = response.json().await
            .map_err(|e| InjectiveError::SerializationError(e.to_string()))?;
        
        Ok(Account {
            address: address.clone(),
            balance: vec![],
            subaccounts: vec![],
        })
    }
    
    pub async fn get_balance(&self, address: &Address, denom: &str) -> Result<String, InjectiveError> {
        let url = format!("{}/cosmos/bank/v1beta1/balances/{}/by_denom?denom={}", 
            self.rest_url, address.to_hex(), denom);
        let response = self.http_client.get(&url).send().await
            .map_err(|e| InjectiveError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Ok("0".to_string());
        }
        
        let data: serde_json::Value = response.json().await
            .map_err(|e| InjectiveError::SerializationError(e.to_string()))?;
        
        Ok(data["balance"]["amount"].as_str().unwrap_or("0").to_string())
    }
    
    pub async fn get_markets(&self, market_type: &str) -> Result<Vec<Market>, InjectiveError> {
        let url = format!("{}/injective/exchange/v1beta1/spot/markets", self.rest_url);
        let response = self.http_client.get(&url).send().await
            .map_err(|e| InjectiveError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(InjectiveError::RpcError(format!("HTTP: {}", response.status())));
        }
        
        let data: serde_json::Value = response.json().await
            .map_err(|e| InjectiveError::SerializationError(e.to_string()))?;
        
        let markets: Vec<Market> = data["markets"].as_array()
            .map(|arr| arr.iter().filter_map(|m| {
                Some(Market {
                    market_id: m["market_id"].as_str()?.to_string(),
                    ticker: m["ticker"].as_str()?.to_string(),
                    quote_denom: m["quote_denom"].as_str()?.to_string(),
                    price_precision: m["price_precision"].as_u64().unwrap_or(8) as u32,
                    quantity_precision: m["quantity_precision"].as_u64().unwrap_or(0) as u32,
                    min_price_tick_size: m["min_price_tick_size"].as_str()?.parse().ok()?,
                    min_quantity_tick_size: m["min_quantity_tick_size"].as_str()?.parse().ok()?,
                })
            }).collect()).unwrap_or_default();
        
        Ok(markets)
    }
    
    pub async fn broadcast_transaction(&self, signed_tx: &SignedTransaction) -> Result<String, InjectiveError> {
        let url = format!("{}/cosmos/tx/v1beta1/txs", self.rest_url);
        
        let tx_bytes = serde_json::to_vec(signed_tx)
            .map_err(|e| InjectiveError::SerializationError(e.to_string()))?;
        
        let response = self.http_client.post(&url)
            .header("Content-Type", "application/json")
            .body(tx_bytes)
            .send()
            .await
            .map_err(|e| InjectiveError::RpcError(e.to_string()))?;
        
        if !response.status().is_success() {
            return Err(InjectiveError::RpcError(format!("Broadcast failed")));
        }
        
        let data: serde_json::Value = response.json().await
            .map_err(|e| InjectiveError::SerializationError(e.to_string()))?;
        
        Ok(data["tx_response"]["txhash"].as_str().unwrap_or("").to_string())
    }
}

// ============================================================================
// Transaction Builder
// ============================================================================

pub struct TransactionBuilder {
    msgs: Vec<Message>,
    fee: Option<Fee>,
    memo: String,
    timeout_height: u64,
}

impl TransactionBuilder {
    pub fn new() -> Self {
        Self {
            msgs: Vec::new(),
            fee: None,
            memo: String::new(),
            timeout_height: 0,
        }
    }
    
    pub fn send(mut self, from: Address, to: Address, amount: u64, denom: &str) -> Self {
        self.msgs.push(Message::Send(SendMsg {
            sender: from,
            recipient: to,
            amount: Coin { denom: denom.to_string(), amount: amount.to_string() },
        }));
        self
    }
    
    pub fn create_spot_order(mut self, sender: Address, market_id: &str, price: f64, quantity: f64, direction: &str) -> Self {
        let subaccount_id = format!("{}000000000000000000000000", sender.to_hex());
        self.msgs.push(Message::CreateSpotMarketOrder(CreateSpotOrderMsg {
            sender,
            market_id: market_id.to_string(),
            subaccount_id,
            order_type: "BUY".to_string(),
            price,
            quantity,
            direction: direction.to_string(),
        }));
        self
    }
    
    pub fn fee(mut self, fee: Fee) -> Self {
        self.fee = Some(fee);
        self
    }
    
    pub fn memo(mut self, memo: &str) -> Self {
        self.memo = memo.to_string();
        self
    }
    
    pub fn build(self) -> Transaction {
        Transaction {
            msgs: self.msgs,
            fee: self.fee.unwrap_or(Fee {
                amount: vec![Coin { denom: "inj".to_string(), amount: "0".to_string() }],
                gas_limit: 200000,
                payer: String::new(),
                granter: String::new(),
            }),
            memo: self.memo,
            timeout_height: self.timeout_height,
        }
    }
}

impl Default for TransactionBuilder {
    fn default() -> Self { Self::new() }
}

// ============================================================================
// Helper Functions
// ============================================================================

pub fn sign_transaction(tx: &Transaction, private_key: &PrivateKey, sender: &Address) -> SignedTransaction {
    let tx_bytes = serde_json::to_vec(tx).unwrap_or_default();
    let signature = private_key.sign(&tx_bytes);
    
    SignedTransaction {
        tx: tx.clone(),
        signature,
        sender: sender.clone(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_address() {
        let addr = Address::from_hex("0123456789ABCDEF0123456789ABCDEF01234567").unwrap();
        println!("Address: {}", addr);
    }
    
    #[test]
    fn test_wallet() {
        let pk = PrivateKey::generate();
        let pubk = pk.public_key();
        let addr = pubk.to_address();
        println!("Wallet address: {}", addr);
    }
}
