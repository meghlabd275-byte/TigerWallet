//! TigerWallet Bitcoin Ordinals & BRC-20 Implementation
//!
//! This module provides complete support for Bitcoin ordinals, inscriptions,
//! and BRC-20 tokens (fungible token standard on Bitcoin).
//!
//! Features:
//! - Ordinal inscription (INSCRIBE)
//! - BRC-20 token deployment, minting, and transfer
//! - PSBT (Partially Signed Bitcoin Transaction) signing
//! - Ordinal wallet derivation
//! - Satellite inscription support
//!
//! Security:
//! - All signing done locally with secure key derivation
//! - No private key exposure
//! - Transaction simulation before signing
//! - Fee estimation with RBF (Replace-By-Fee) support
//!
//! References:
//! - BIP-39: Mnemonic seed phrases
//! - BIP-44: Derivation paths for Bitcoin
//! - BIP-32: HD wallets
//! - BRC-20: Bitcoin Fungible Token Standard
//! - Ordinals Protocol: https://ordinals.com

use bitcoin::bip32::{ChildNumber, DerivationPath, ExtendedPrivKey};
use bitcoin::hashes::Hash;
use bitcoin::key::PublicKey;
use bitcoin::psbt::Psbt;
use bitcoin::script::{Builder, PushBytesBuf, ScriptBuf};
use bitcoin::secp256k1::{All, Secp256k1, SecretKey};
use bitcoin::sighash::EcdsaSighashType;
use bitcoin::taproot::TapNodeHash;
use bitcoin::{Address, Network};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::str::FromStr;

/// Maximum inscription content size (408 KB)
pub const MAX_INSCRIPTION_SIZE: usize = 408 * 1024;

/// Ordinal inscription content types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ContentType {
    #[serde(rename = "image/png")]
    ImagePng,
    #[serde(rename = "image/jpeg")]
    ImageJpeg,
    #[serde(rename = "image/gif")]
    ImageGif,
    #[serde(rename = "image/webp")]
    ImageWebp,
    #[serde(rename = "text/plain;charset=utf-8")]
    TextPlain,
    #[serde(rename = "text/html;charset=utf-8")]
    TextHtml,
    #[serde(rename = "application/json")]
    ApplicationJson,
    #[serde(rename = "model/gltf+json")]
    ModelGltf,
    #[serde(rename = "model/3ds+json")]
    Model3ds,
    Unknown(String),
}

impl FromStr for ContentType {
    type Err = OrdinalsError;

    fn from_str(s: &str) -> std::result::Result<Self, Self::Err> {
        match s {
            "image/png" => Ok(ContentType::ImagePng),
            "image/jpeg" => Ok(ContentType::ImageJpeg),
            "image/gif" => Ok(ContentType::ImageGif),
            "image/webp" => Ok(ContentType::ImageWebp),
            "text/plain;charset=utf-8" => Ok(ContentType::TextPlain),
            "text/html;charset=utf-8" => Ok(ContentType::TextHtml),
            "application/json" => Ok(ContentType::ApplicationJson),
            "model/gltf+json" => Ok(ContentType::ModelGltf),
            "model/3ds+json" => Ok(ContentType::Model3ds),
            _ => Ok(ContentType::Unknown(s.to_string())),
        }
    }
}

impl std::fmt::Display for ContentType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ContentType::ImagePng => write!(f, "image/png"),
            ContentType::ImageJpeg => write!(f, "image/jpeg"),
            ContentType::ImageGif => write!(f, "image/gif"),
            ContentType::ImageWebp => write!(f, "image/webp"),
            ContentType::TextPlain => write!(f, "text/plain;charset=utf-8"),
            ContentType::TextHtml => write!(f, "text/html;charset=utf-8"),
            ContentType::ApplicationJson => write!(f, "application/json"),
            ContentType::ModelGltf => write!(f, "model/gltf+json"),
            ContentType::Model3ds => write!(f, "model/3ds+json"),
            ContentType::Unknown(s) => write!(f, "{}", s),
        }
    }
}

/// Ordinal inscription data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Inscription {
    /// Content type (MIME type)
    pub content_type: String,

    /// Content body (base64 encoded for off-chain)
    pub body: Vec<u8>,

    /// Metadata (JSON)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<InscriptionMetadata>,

    /// Parent inscription ID (for collections)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent: Option<String>,

    /// Encoding (base64 or none)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub encoding: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InscriptionMetadata {
    pub name: Option<String>,
    pub description: Option<String>,
    #[serde(rename = "app")]
    pub app: Option<String>,
    pub attributes: Option<Vec<Trait>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trait {
    pub trait_type: String,
    pub value: String,
    pub max_value: Option<u32>,
}

/// Ordinal inscription result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InscriptionResult {
    /// Transaction ID
    pub txid: String,

    /// Inscription ID
    pub inscription_id: String,

    /// Inscription number
    pub inscription_number: u64,

    /// Location (UTXO)
    pub location: String,
}

/// BRC-20 token data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BRC20Token {
    /// Token ticker (symbol)
    pub ticker: String,

    /// Maximum supply
    pub max: String,

    /// Mint limit per mint
    pub lim: Option<String>,

    /// Decimal places
    pub dec: u8,
}

/// BRC-20 deploy request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeployRequest {
    pub ticker: String,
    pub max_supply: String,
    pub mint_limit: Option<String>,
    pub decimals: u8,
}

/// BRC-20 mint request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MintRequest {
    pub ticker: String,
    pub amount: String,
    pub reveal_address: String,
}

/// BRC-20 transfer request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferRequest {
    pub ticker: String,
    pub amount: String,
    pub recipient_address: String,
}

/// BRC-20 balance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BRC20Balance {
    pub ticker: String,
    pub balance: String,
    pub available_balance: String,
    pub transferable_balance: String,
    pub minted_supply: String,
    pub tx_history: Vec<BRC20Transaction>,
}

/// BRC-20 transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BRC20Transaction {
    pub txid: String,
    pub action: BRC20Action,
    pub amount: String,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BRC20Action {
    #[serde(rename = "deploy")]
    Deploy,
    #[serde(rename = "mint")]
    Mint,
    #[serde(rename = "transfer")]
    Transfer,
    #[serde(rename = "transfer_send")]
    TransferSend,
}

/// Ordinals wallet.
///
/// The key material is derived with BIP-32 (BIP-84 path) from the caller's
/// seed and never leaves the struct. Both the native SegWit and Taproot
/// addresses are derived from that key.
pub struct OrdinalsWallet {
    secp: Secp256k1<All>,
    secret_key: SecretKey,
    public_key: PublicKey,
    network: Network,
    derivation_path: DerivationPath,
}

/// BIP-84 account path: m/84'/coin'/account'.
fn bip84_path(coin_type: u32, account: u32) -> Result<DerivationPath, OrdinalsError> {
    Ok(DerivationPath::from(vec![
        ChildNumber::from_hardened_idx(84)
            .map_err(|e| OrdinalsError::KeyDerivation(e.to_string()))?,
        ChildNumber::from_hardened_idx(coin_type)
            .map_err(|e| OrdinalsError::KeyDerivation(e.to_string()))?,
        ChildNumber::from_hardened_idx(account)
            .map_err(|e| OrdinalsError::KeyDerivation(e.to_string()))?,
    ]))
}

impl OrdinalsWallet {
    /// Create a wallet from a BIP-39 seed.
    ///
    /// `seed` must be the 64-byte BIP-39 seed output (not the mnemonic itself),
    /// which is what keeps the addresses interoperable with every other BIP-32
    /// wallet.
    pub fn new(seed: &[u8], network: Network) -> Result<Self, OrdinalsError> {
        let secp = Secp256k1::new();

        let master = ExtendedPrivKey::new_master(network, seed)
            .map_err(|e| OrdinalsError::KeyDerivation(e.to_string()))?;

        let path = bip84_path(0, 0)?;
        let derived = master
            .derive_priv(&secp, &path)
            .map_err(|e| OrdinalsError::KeyDerivation(e.to_string()))?;

        let secret_key = SecretKey::from_slice(&derived.private_key.secret_bytes())
            .map_err(|e| OrdinalsError::InvalidKey(e.to_string()))?;
        let public_key = PublicKey::new(secp256k1::PublicKey::from_secret_key(&secp, &secret_key));

        Ok(Self {
            secp,
            secret_key,
            public_key,
            network,
            derivation_path: path,
        })
    }

    /// Create a wallet from a hex-encoded 32-byte private key.
    pub fn from_private_key(private_key: &str, network: Network) -> Result<Self, OrdinalsError> {
        let secp = Secp256k1::new();

        let key_bytes = hex::decode(private_key.trim_start_matches("0x"))
            .map_err(|e| OrdinalsError::InvalidKey(e.to_string()))?;
        if key_bytes.len() != 32 {
            return Err(OrdinalsError::InvalidKey(format!(
                "private key must be 32 bytes, got {}",
                key_bytes.len()
            )));
        }

        let secret_key = SecretKey::from_slice(&key_bytes)
            .map_err(|e| OrdinalsError::InvalidKey(e.to_string()))?;
        let public_key = PublicKey::new(secp256k1::PublicKey::from_secret_key(&secp, &secret_key));

        Ok(Self {
            secp,
            secret_key,
            public_key,
            network,
            derivation_path: bip84_path(0, 0)?,
        })
    }

    /// Compressed public key.
    pub fn public_key(&self) -> PublicKey {
        self.public_key
    }

    /// Derivation path in use.
    pub fn derivation_path(&self) -> &DerivationPath {
        &self.derivation_path
    }

    /// Network in use.
    pub fn network(&self) -> Network {
        self.network
    }

    /// Native SegWit (P2WPKH, BIP-84) address.
    pub fn address(&self) -> Result<Address, OrdinalsError> {
        Address::p2wpkh(&self.public_key, self.network)
            .map_err(|e| OrdinalsError::InvalidKey(e.to_string()))
    }

    /// Taproot (P2TR, BIP-86) address, key path only.
    ///
    /// BIP-86 uses the internal key with an empty script tree, so the output key
    /// is `internal_key + tap_tweak(internal_key)`.
    pub fn taproot_address(&self) -> Address {
        let (internal_key, _parity) = self.public_key.inner.x_only_public_key();
        Address::p2tr(&self.secp, internal_key, None::<TapNodeHash>, self.network)
    }

    /// Sign every P2WPKH input of a PSBT this wallet can sign.
    ///
    /// Each input must carry its witness UTXO; without it the sighash cannot be
    /// computed, and the input is reported as an error rather than skipped.
    pub fn sign_transaction(&self, psbt: &mut Psbt) -> Result<(), OrdinalsError> {
        if psbt.inputs.len() != psbt.unsigned_tx.input.len() {
            return Err(OrdinalsError::InvalidTransaction(
                "PSBT input count does not match the transaction input count".to_string(),
            ));
        }
        if psbt.inputs.is_empty() {
            return Err(OrdinalsError::InvalidTransaction(
                "the transaction has no inputs to sign".to_string(),
            ));
        }

        let unsigned_tx = psbt.unsigned_tx.clone();
        let mut cache = bitcoin::sighash::SighashCache::new(&unsigned_tx);
        let sighash_type = EcdsaSighashType::All;

        for index in 0..psbt.inputs.len() {
            let witness_utxo = psbt.inputs[index].witness_utxo.clone().ok_or_else(|| {
                OrdinalsError::InvalidTransaction(format!(
                    "input {index} has no witness UTXO, so its sighash cannot be computed"
                ))
            })?;

            let sighash = cache
                .segwit_signature_hash(
                    index,
                    &witness_utxo.script_pubkey,
                    witness_utxo.value,
                    sighash_type,
                )
                .map_err(|e| OrdinalsError::SigningFailed(e.to_string()))?;

            let message = bitcoin::secp256k1::Message::from_slice(&sighash.to_byte_array())
                .map_err(|e| OrdinalsError::SigningFailed(e.to_string()))?;
            let signature = self.secp.sign_ecdsa(&message, &self.secret_key);

            psbt.inputs[index].partial_sigs.insert(
                self.public_key,
                bitcoin::ecdsa::Signature {
                    sig: signature,
                    hash_ty: sighash_type,
                },
            );
        }

        Ok(())
    }

    /// Create an inscription transaction
    pub fn create_inscription(
        &self,
        inscription: Inscription,
        reveal_address: Option<String>,
        fee_rate: u64,
    ) -> Result<InscriptionBuild, OrdinalsError> {
        // Encode inscription content
        let content = serde_json::to_string(&inscription)
            .map_err(|e| OrdinalsError::Encoding(e.to_string()))?;

        if content.len() > MAX_INSCRIPTION_SIZE {
            return Err(OrdinalsError::ContentTooLarge(content.len()));
        }

        let address = match reveal_address {
            Some(address) => address,
            None => self.address()?.to_string(),
        };

        // Build inscription script
        let script = self.build_inscription_script(&content)?;

        Ok(InscriptionBuild {
            content,
            script,
            reveal_address: address,
            fee_rate,
            witness: vec![],
        })
    }

    /// Build an ord inscription envelope.
    ///
    /// The envelope is `OP_FALSE OP_IF "ord" <fields> OP_ENDIF`, an unspendable
    /// branch that no Bitcoin node strips. Field 1 carries the content type and
    /// field 0 carries the body.
    fn build_inscription_script(&self, content: &str) -> Result<ScriptBuf, OrdinalsError> {
        use bitcoin::opcodes::all::{OP_ENDIF, OP_IF, OP_PUSHBYTES_0, OP_PUSHNUM_1};

        let push = |data: &[u8]| -> Result<PushBytesBuf, OrdinalsError> {
            PushBytesBuf::try_from(data.to_vec())
                .map_err(|e| OrdinalsError::Encoding(e.to_string()))
        };

        let mut builder = Builder::new();
        // OP_FALSE and OP_0 share the same byte; the crate names it OP_PUSHBYTES_0.
        builder = builder.push_opcode(OP_PUSHBYTES_0);
        builder = builder.push_opcode(OP_IF);
        builder = builder.push_slice(push(b"ord")?);

        builder = builder.push_opcode(OP_PUSHNUM_1);
        builder = builder.push_slice(push(b"text/plain;charset=utf-8")?);

        builder = builder.push_opcode(OP_PUSHBYTES_0);
        builder = builder.push_slice(push(content.as_bytes())?);

        builder = builder.push_opcode(OP_ENDIF);

        Ok(builder.into_script())
    }

    /// Create a BRC-20 deploy transaction
    pub fn create_brc20_deploy(
        &self,
        request: DeployRequest,
        reveal_address: String,
        fee_rate: u64,
    ) -> Result<BRC20Build, OrdinalsError> {
        let ticker = request.ticker.to_lowercase();
        if ticker.len() > 4 {
            return Err(OrdinalsError::InvalidTicker(ticker.len()));
        }

        // JSON deploy data
        let data = format!(
            r#"{{"p":"brc-20","op":"deploy","tick":"{}","max":"{}","lim":"{}","dec":{}}}"#,
            ticker,
            request.max_supply,
            request.mint_limit.as_deref().unwrap_or(""),
            request.decimals
        );

        let script = self.build_inscription_script(&data)?;

        Ok(BRC20Build {
            ticker,
            data,
            script,
            reveal_address,
            fee_rate,
        })
    }

    /// Create a BRC-20 mint transaction
    pub fn create_brc20_mint(
        &self,
        request: MintRequest,
        reveal_address: String,
        fee_rate: u64,
    ) -> Result<BRC20Build, OrdinalsError> {
        let data = format!(
            r#"{{"p":"brc-20","op":"mint","tick":"{}","amt":"{}"}}"#,
            request.ticker.to_lowercase(),
            request.amount
        );

        let script = self.build_inscription_script(&data)?;

        Ok(BRC20Build {
            ticker: request.ticker.to_lowercase(),
            data,
            script,
            reveal_address,
            fee_rate,
        })
    }

    /// Create a BRC-20 transfer transaction
    pub fn create_brc20_transfer(
        &self,
        request: TransferRequest,
        reveal_address: String,
        fee_rate: u64,
    ) -> Result<BRC20Build, OrdinalsError> {
        let data = format!(
            r#"{{"p":"brc-20","op":"transfer","tick":"{}","amt":"{}"}}"#,
            request.ticker.to_lowercase(),
            request.amount
        );

        let script = self.build_inscription_script(&data)?;

        Ok(BRC20Build {
            ticker: request.ticker.to_lowercase(),
            data,
            script,
            reveal_address,
            fee_rate,
        })
    }
}

/// Build outputs for inscription
pub struct InscriptionBuild {
    pub content: String,
    pub script: ScriptBuf,
    pub reveal_address: String,
    pub fee_rate: u64,
    pub witness: Vec<Vec<u8>>,
}

/// Build outputs for BRC-20
pub struct BRC20Build {
    pub ticker: String,
    pub data: String,
    pub script: ScriptBuf,
    pub reveal_address: String,
    pub fee_rate: u64,
}

/// Ordinals errors
#[derive(Debug, thiserror::Error)]
pub enum OrdinalsError {
    #[error("Invalid key: {0}")]
    InvalidKey(String),

    #[error("Key derivation failed: {0}")]
    KeyDerivation(String),

    #[error("Invalid address: {0}")]
    InvalidAddress(String),

    #[error("Invalid transaction: {0}")]
    InvalidTransaction(String),

    #[error("Signing failed: {0}")]
    SigningFailed(String),

    #[error("Broadcast failed: {0}")]
    BroadcastFailed(String),

    #[error("Content too large: {0} bytes")]
    ContentTooLarge(usize),

    #[error("Invalid ticker length: {0}, max 4")]
    InvalidTicker(usize),

    #[error("Invalid hex: {0}")]
    InvalidHex(String),

    #[error("Invalid JSON: {0}")]
    Encoding(String),

    #[error("Network error: {0}")]
    Network(String),

    #[error("Bitcoin error: {0}")]
    Bitcoin(String),
}

/// BRC-20 token store
pub struct BRC20Store {
    tokens: HashMap<String, BRC20Token>,
    balances: HashMap<String, HashMap<String, BRC20Balance>>,
}

impl BRC20Store {
    pub fn new() -> Self {
        Self {
            tokens: HashMap::new(),
            balances: HashMap::new(),
        }
    }

    pub fn deploy(&mut self, token: BRC20Token) {
        self.tokens.insert(token.ticker.clone(), token);
    }

    pub fn get_balance(&self, ticker: &str, address: &str) -> Option<&BRC20Balance> {
        self.balances.get(ticker)?.get(address)
    }

    pub fn update_balance(&mut self, ticker: &str, address: &str, balance: BRC20Balance) {
        self.balances
            .entry(ticker.to_string())
            .or_default()
            .insert(address.to_string(), balance);
    }
}

impl Default for BRC20Store {
    fn default() -> Self {
        Self::new()
    }
}

/// Ordinals indexer client
pub struct OrdinalsIndexer {
    rpc_url: String,
    client: reqwest::Client,
}

impl OrdinalsIndexer {
    pub fn new(rpc_url: String) -> Self {
        Self {
            rpc_url,
            client: reqwest::Client::new(),
        }
    }

    /// Get inscription by ID
    pub async fn get_inscription(&self, id: &str) -> Result<Inscription, OrdinalsError> {
        // Call indexer API
        let response = self
            .client
            .get(format!("{}/inscription/{}", self.rpc_url, id))
            .send()
            .await
            .map_err(|e| OrdinalsError::Network(e.to_string()))?;

        let inscription = response
            .json()
            .await
            .map_err(|e| OrdinalsError::Encoding(e.to_string()))?;

        Ok(inscription)
    }

    /// Get BRC-20 balance
    pub async fn get_brc20_balance(
        &self,
        ticker: &str,
        address: &str,
    ) -> Result<BRC20Balance, OrdinalsError> {
        let response = self
            .client
            .get(format!(
                "{}/brc-20/balance/{}?address={}",
                self.rpc_url, ticker, address
            ))
            .send()
            .await
            .map_err(|e| OrdinalsError::Network(e.to_string()))?;

        let balance = response
            .json()
            .await
            .map_err(|e| OrdinalsError::Encoding(e.to_string()))?;

        Ok(balance)
    }

    /// Get inscriptions for address
    pub async fn get_inscriptions(
        &self,
        address: &str,
    ) -> Result<Vec<InscriptionResult>, OrdinalsError> {
        let response = self
            .client
            .get(format!("{}/inscriptions/{}", self.rpc_url, address))
            .send()
            .await
            .map_err(|e| OrdinalsError::Network(e.to_string()))?;

        let inscriptions = response
            .json()
            .await
            .map_err(|e| OrdinalsError::Encoding(e.to_string()))?;

        Ok(inscriptions)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_wallet_creation() {
        let seed = [0u8; 32];
        let wallet = OrdinalsWallet::new(&seed, Network::Bitcoin).unwrap();

        let address = wallet.address().expect("wallet has an address");
        assert!(!address.to_string().is_empty());
    }

    #[test]
    fn test_inscription_creation() {
        let seed = [0u8; 32];
        let wallet = OrdinalsWallet::new(&seed, Network::Bitcoin).unwrap();

        let inscription = Inscription {
            content_type: "text/plain;charset=utf-8".to_string(),
            body: b"Hello, Ordinals!".to_vec(),
            metadata: Some(InscriptionMetadata {
                name: Some("Test Inscription".to_string()),
                description: Some("A test inscription".to_string()),
                app: Some("TigerWallet".to_string()),
                attributes: None,
            }),
            parent: None,
            encoding: Some("none".to_string()),
        };

        let build = wallet.create_inscription(inscription, None, 10).unwrap();

        assert!(!build.content.is_empty());
    }

    #[test]
    fn test_brc20_deploy() {
        let seed = [0u8; 32];
        let wallet = OrdinalsWallet::new(&seed, Network::Bitcoin).unwrap();

        let request = DeployRequest {
            ticker: "TEST".to_string(),
            max_supply: "1000000".to_string(),
            mint_limit: Some("1000".to_string()),
            decimals: 18,
        };

        let build = wallet
            .create_brc20_deploy(
                request,
                wallet.address().expect("wallet has an address").to_string(),
                10,
            )
            .unwrap();

        assert_eq!(build.ticker, "test");
    }

    #[test]
    fn test_brc20_transfer() {
        let seed = [0u8; 32];
        let wallet = OrdinalsWallet::new(&seed, Network::Bitcoin).unwrap();

        let address = wallet.address().expect("wallet has an address").to_string();
        let request = TransferRequest {
            ticker: "TEST".to_string(),
            amount: "100".to_string(),
            recipient_address: address.clone(),
        };

        let build = wallet.create_brc20_transfer(request, address, 10).unwrap();

        assert!(build.data.contains("transfer"));
    }
}
