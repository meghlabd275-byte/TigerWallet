//! TigerWallet MasterWallet — Rust core.
//!
//! REAL cryptographic primitives: secp256k1 (k256), keccak256 (sha3), BIP-39
//! mnemonic validation, BIP-32/44 HD derivation, AES-256-GCM seed encryption.
//! Signs EVM transactions + messages with real ECDSA and delegates broadcast /
//! persistence to the canonical Go backend at MASTER_WALLET_API_URL (:8450).
//!
//! No SHA-256 fakes, no random "signatures", no in-memory-only state.

use std::collections::HashMap;
use std::sync::Arc;

use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes256Gcm, Key, Nonce};
use hmac::{Hmac, Mac};
use k256::ecdsa::{SigningKey, VerifyingKey};
use k256::elliptic_curve::sec1::ToEncodedPoint;
use parking_lot::RwLock;
use rand::RngCore;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha512};
use sha3::Keccak256;
use thiserror::Error;

type HmacSha512 = Hmac<Sha512>;

/// Errors returned by the master wallet service.
#[derive(Debug, Error)]
pub enum MasterError {
    #[error("invalid mnemonic: {0}")]
    InvalidMnemonic(String),
    #[error("derivation failed: {0}")]
    DerivationFailed(String),
    #[error("signing failed: {0}")]
    SigningFailed(String),
    #[error("wallet not found")]
    WalletNotFound,
    #[error("encryption error: {0}")]
    Encryption(String),
    #[error("backend request failed: {0}")]
    BackendRequest(String),
    #[error("fee percentage exceeds maximum (20%)")]
    FeeTooHigh,
}

impl From<k256::ecdsa::Error> for MasterError {
    fn from(e: k256::ecdsa::Error) -> Self {
        MasterError::SigningFailed(e.to_string())
    }
}

impl From<k256::elliptic_curve::Error> for MasterError {
    fn from(e: k256::elliptic_curve::Error) -> Self {
        MasterError::DerivationFailed(e.to_string())
    }
}

/// Fee configuration for the master wallet.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FeeConfig {
    pub withdrawal_fee_percent: f64,
    pub swap_fee_percent: f64,
    pub transaction_fee_percent: f64,
    pub minimum_fee: u64,
}

/// A master wallet record (in-memory working copy; canonical state is in the
/// backend PostgreSQL via the HTTP client).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MasterWallet {
    pub id: String,
    pub name: String,
    pub address: String,
    pub public_key: String,
    pub chain_id: i64,
    pub blockchain: String,
    pub wallet_type: String,
    pub created_at: i64,
    pub fee_config: FeeConfig,
}

/// User wallet info under a master wallet.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserWalletInfo {
    pub user_id: String,
    pub wallet_address: String,
    pub chain: String,
    pub balance: u64,
    pub total_received: u64,
    pub total_sent: u64,
    pub is_active: bool,
}

// --- BIP-39 wordlist (the canonical 2048 English words). ---
// Embedded so mnemonic validation uses the real wordlist + checksum.
include!("bip39_wordlist.rs");

/// Validates a BIP-39 mnemonic: every word must be in the wordlist and the
/// checksum (last entropy bits) must match.
pub fn validate_mnemonic(mnemonic: &str) -> bool {
    let words: Vec<&str> = mnemonic.split_whitespace().collect();
    if words.len() != 12 && words.len() != 15 && words.len() != 18 && words.len() != 21 && words.len() != 24 {
        return false;
    }
    let mut bits = String::with_capacity(words.len() * 11);
    for w in &words {
        let idx = match WORDLIST.iter().position(|&x| x == *w) {
            Some(i) => i,
            None => return false,
        };
        bits.push_str(&format!("{:011b}", idx));
    }
    let entropy_bits = (words.len() * 11 * 32) / 33;
    let entropy_str = &bits[..entropy_bits];
    let checksum_str = &bits[entropy_bits..];
    let mut entropy = Vec::with_capacity(entropy_bits / 8);
    for chunk in entropy_str.as_bytes().chunks(8) {
        let b = chunk.iter().fold(0u8, |acc, &c| (acc << 1) | (c - b'0'));
        entropy.push(b);
    }
    // BIP-39 checksum is the first N bits of SHA-256(entropy).
    let hash = sha2::Sha256::digest(&entropy);
    let first_n = (hash[0] >> (8 - words.len() / 3)) as usize;
    let expected: usize = checksum_str.bytes().fold(0, |acc, c| (acc << 1) | ((c - b'0') as usize));
    first_n == expected
}

// --- BIP-32 HD key derivation (secp256k1) ---

const HARDEN_OFFSET: u32 = 0x8000_0000;

/// Derives the secp256k1 private key at a BIP-32 path (e.g. "m/44'/60'/0'/0/0")
/// from a BIP-39 seed. REAL HMAC-SHA512 CKD; no SHA-256-of-seed fakes.
pub fn derive_private_key(seed: &[u8], path: &str) -> Result<Vec<u8>, MasterError> {
    // Master key = HMAC-SHA512("Bitcoin seed", seed)
    let mut mac = <HmacSha512 as Mac>::new_from_slice(b"Bitcoin seed").unwrap();
    mac.update(seed);
    let i = mac.finalize().into_bytes();
    let mut parent_key = i[..32].to_vec();
    let mut parent_chain = i[32..].to_vec();

    let segments = parse_path(path)?;
    for idx in segments {
        let (child, chain) = ckd_priv(&parent_key, &parent_chain, idx)?;
        parent_key = child;
        parent_chain = chain;
    }
    Ok(parent_key)
}

fn parse_path(path: &str) -> Result<Vec<u32>, MasterError> {
    let path = path.trim();
    if path.is_empty() || path == "m" {
        return Ok(vec![]);
    }
    let path = path.strip_prefix("m/").unwrap_or(path);
    let mut out = Vec::new();
    for p in path.split('/') {
        let p = p.trim();
        if p.is_empty() {
            continue;
        }
        let mut hardened = false;
        let p = if p.ends_with('\'') || p.ends_with('h') || p.ends_with('H') {
            hardened = true;
            &p[..p.len() - 1]
        } else {
            p
        };
        let n: u32 = p.parse().map_err(|_| MasterError::DerivationFailed(format!("invalid path segment {}", p)))?;
        out.push(if hardened { n + HARDEN_OFFSET } else { n });
    }
    Ok(out)
}

fn ckd_priv(parent_key: &[u8], parent_chain: &[u8], index: u32) -> Result<(Vec<u8>, Vec<u8>), MasterError> {
    let mut data = Vec::new();
    if index >= HARDEN_OFFSET {
        // Hardened: 0x00 || ser256(kpar) || ser32(i)
        data.push(0u8);
        data.extend_from_slice(parent_key);
        data.extend_from_slice(&index.to_be_bytes());
    } else {
        // Normal: serP(point(kpar)) || ser32(i)
        let secret = k256::SecretKey::from_bytes(parent_key.into())?;
        let pubkey = secret.public_key();
        let comp = pubkey.to_encoded_point(true);
        data.extend_from_slice(comp.as_bytes());
        data.extend_from_slice(&index.to_be_bytes());
    }
    let mut mac = <HmacSha512 as Mac>::new_from_slice(parent_chain).unwrap();
    mac.update(&data);
    let i = mac.finalize().into_bytes();
    let il = &i[..32];
    let ir = &i[32..];
    // child = (il + kpar) mod n — modular addition over the secp256k1 order n.
    // n = FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141
    use num_bigint::BigUint;
    use num_traits::Zero;
    let n = BigUint::parse_bytes(b"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16).unwrap();
    let il_u = BigUint::from_bytes_be(il);
    let parent_u = BigUint::from_bytes_be(parent_key);
    let child = (il_u + parent_u) % &n;
    if child.is_zero() {
        return Err(MasterError::DerivationFailed("invalid child key (zero)".into()));
    }
    let child_bytes = child.to_bytes_be();
    let child_bytes = bigint_pad(&child_bytes, 32);
    Ok((child_bytes, ir.to_vec()))
}

fn bigint_pad(n: &[u8], len: usize) -> Vec<u8> {
    let mut out = vec![0u8; len];
    let start = len.saturating_sub(n.len());
    out[start..].copy_from_slice(n);
    out
}

/// Derives the EVM address for a secp256k1 private key:
/// keccak256(publicKey[1:]) last 20 bytes, EIP-55 checksummed.
pub fn private_key_to_address(priv_key: &[u8]) -> Result<String, MasterError> {
    let secret = k256::SecretKey::from_bytes(priv_key.into())?;
    let pubkey = secret.public_key();
    let encoded = pubkey.to_encoded_point(false);
    let bytes = encoded.as_bytes();
    // uncompressed point: 0x04 || X(32) || Y(32); we hash X||Y
    let pub_xy = &bytes[1..];
    let hash = Keccak256::digest(pub_xy);
    let addr_bytes = &hash[hash.len() - 20..];
    Ok(eip55_checksum(addr_bytes))
}

/// EIP-55 checksum: hash the LOWERCASE hex address and capitalize where the
/// hash nibble is >= 8.
fn eip55_checksum(addr: &[u8]) -> String {
    let lower = hex::encode(addr);
    let hash = Keccak256::digest(lower.as_bytes());
    let hash_hex = hex::encode(hash);
    let mut out = String::with_capacity(42);
    out.push_str("0x");
    for (i, c) in lower.chars().enumerate() {
        if c.is_ascii_digit() {
            out.push(c);
        } else {
            let h = hash_hex.as_bytes().get(i).copied().unwrap_or(b'0');
            let nibble = (h as char).to_digit(16).unwrap_or(0);
            if nibble >= 8 {
                out.push(c.to_ascii_uppercase());
            } else {
                out.push(c);
            }
        }
    }
    out
}

// --- EVM transaction + message signing (real ECDSA) ---

/// Signs a 32-byte prehash with a secp256k1 private key and returns r||s||v
/// (65 bytes). v is the recovery id (0/1). The prehash is expected to be
/// keccak256 (for EVM txs) or the EIP-191 hash (for personal_sign).
pub fn sign_hash(priv_key: &[u8], msg_hash: &[u8]) -> Result<Vec<u8>, MasterError> {
    let signing_key = SigningKey::from_bytes(priv_key.into())?;
    let (sig, rec_id) = signing_key.sign_prehash_recoverable(msg_hash)?;
    let sig_bytes = sig.to_bytes();
    let v = rec_id.to_byte();
    let mut out = Vec::with_capacity(65);
    out.extend_from_slice(&sig_bytes);
    out.push(v);
    Ok(out)
}

/// Signs an EIP-191 personal message (keccak256 of the Ethereum prefix).
pub fn sign_personal_message(priv_key: &[u8], message: &[u8]) -> Result<Vec<u8>, MasterError> {
    let prefixed = format!("\x19Ethereum Signed Message:\n{}{}", message.len(), String::from_utf8_lossy(message));
    let hash = Keccak256::digest(prefixed.as_bytes());
    sign_hash(priv_key, &hash)
}

// Recover the public key from (prehash, signature, recovery_id).
fn recover_pubkey(hash: &[u8], sig_bytes: &[u8], v: u8) -> Option<VerifyingKey> {
    let rec_id = k256::ecdsa::RecoveryId::from_byte(v)?;
    let sig = k256::ecdsa::Signature::from_slice(sig_bytes).ok()?;
    VerifyingKey::recover_from_prehash(hash, &sig, rec_id).ok()
}

// --- Seed encryption: scrypt + AES-256-GCM ---

const SCRYPT_N: u32 = 1 << 18;
const SCRYPT_R: u32 = 8;
const SCRYPT_P: u32 = 1;

/// Encrypts a seed with a password: scrypt key derivation + AES-256-GCM.
/// Returns hex(salt||nonce||ciphertext). Constant-time MAC compare (GCM tag).
pub fn encrypt_seed(seed: &[u8], password: &str) -> Result<String, MasterError> {
    let mut salt = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut salt);
    let dk = scrypt_key(password, &salt)?;
    let key = Key::<Aes256Gcm>::from_slice(&dk);
    let cipher = Aes256Gcm::new(key);
    let mut nonce_bytes = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ct = cipher.encrypt(nonce, seed).map_err(|e| MasterError::Encryption(e.to_string()))?;
    Ok(format!("{}{}{}", hex::encode(salt), hex::encode(nonce_bytes), hex::encode(ct)))
}

/// Decrypts a seed: wrong password fails the GCM auth tag.
pub fn decrypt_seed(enc_hex: &str, password: &str) -> Result<Vec<u8>, MasterError> {
    if enc_hex.len() < 128 {
        return Err(MasterError::Encryption("invalid encrypted seed".into()));
    }
    let salt = hex::decode(&enc_hex[..64]).map_err(|e| MasterError::Encryption(e.to_string()))?;
    let dk = scrypt_key(password, &salt)?;
    let key = Key::<Aes256Gcm>::from_slice(&dk);
    let cipher = Aes256Gcm::new(key);
    let nonce_bytes = hex::decode(&enc_hex[64..88]).map_err(|e| MasterError::Encryption(e.to_string()))?;
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ct = hex::decode(&enc_hex[88..]).map_err(|e| MasterError::Encryption(e.to_string()))?;
    let pt = cipher.decrypt(nonce, ct.as_ref()).map_err(|_| MasterError::Encryption("invalid password (authentication failed)".into()))?;
    Ok(pt)
}

fn scrypt_key(password: &str, salt: &[u8]) -> Result<[u8; 32], MasterError> {
    let mut dk = [0u8; 32];
    scrypt::scrypt(password.as_bytes(), salt, &scrypt::Params::new(log2(SCRYPT_N), SCRYPT_R, SCRYPT_P, 32).unwrap(), &mut dk)
        .map_err(|e| MasterError::Encryption(e.to_string()))?;
    Ok(dk)
}

fn log2(n: u32) -> u8 {
    (31 - n.leading_zeros()) as u8
}

// --- HTTP client to the canonical Go backend ---

/// ClientConfig holds the backend URL + optional JWT token.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientConfig {
    pub backend_url: String,
    pub jwt_token: Option<String>,
}

impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            backend_url: std::env::var("MASTER_WALLET_API_URL").unwrap_or_else(|_| "http://localhost:8450".to_string()),
            jwt_token: None,
        }
    }
}

/// BackendClient is the HTTP client to the canonical MasterWallet Go backend.
pub struct BackendClient {
    http: reqwest::Client,
    config: RwLock<ClientConfig>,
}

impl BackendClient {
    pub fn new(config: ClientConfig) -> Self {
        let http = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .expect("failed to build HTTP client");
        Self { http, config: RwLock::new(config) }
    }

    pub fn set_token(&self, token: String) {
        self.config.write().jwt_token = Some(token);
    }

    async fn get<T: for<'de> Deserialize<'de>>(&self, path: &str) -> Result<T, MasterError> {
        let cfg = self.config.read().clone();
        let mut req = self.http.get(format!("{}{}", cfg.backend_url, path));
        if let Some(t) = &cfg.jwt_token {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if !resp.status().is_success() {
            return Err(MasterError::BackendRequest(format!("HTTP {}", resp.status())));
        }
        resp.json().await.map_err(|e| MasterError::BackendRequest(e.to_string()))
    }

    async fn post<T: for<'de> Deserialize<'de>, B: Serialize>(&self, path: &str, body: &B) -> Result<T, MasterError> {
        let cfg = self.config.read().clone();
        let mut req = self.http.post(format!("{}{}", cfg.backend_url, path)).json(body);
        if let Some(t) = &cfg.jwt_token {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if !resp.status().is_success() {
            return Err(MasterError::BackendRequest(format!("HTTP {}", resp.status())));
        }
        resp.json().await.map_err(|e| MasterError::BackendRequest(e.to_string()))
    }

    // ---- Real fetchers (delegate to the canonical Go backend) ----

    pub async fn login(&self, email: &str, password: &str) -> Result<LoginResponse, MasterError> {
        self.post("/api/v1/auth/login", &serde_json::json!({"email": email, "password": password})).await
    }

    pub async fn get_master_wallets(&self) -> Result<WalletListResponse, MasterError> {
        self.get("/api/v1/master-wallet").await
    }

    pub async fn create_master_wallet(&self, name: &str, password: &str, chain_id: i64) -> Result<MasterWallet, MasterError> {
        self.post("/api/v1/master-wallet", &serde_json::json!({"name": name, "password": password, "chain_id": chain_id})).await
    }

    pub async fn get_balance(&self, wallet_id: &str) -> Result<BalanceResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/balance", wallet_id)).await
    }

    pub async fn sign_transaction(&self, wallet_id: &str, to: &str, amount: &str, password: &str, token: Option<&str>) -> Result<TransactionResponse, MasterError> {
        let mut body = serde_json::json!({"to": to, "amount": amount, "password": password});
        if let Some(t) = token {
            body["token"] = serde_json::Value::String(t.to_string());
        }
        self.post(&format!("/api/v1/master-wallet/{}/sign", wallet_id), &body).await
    }

    pub async fn get_transactions(&self, master_wallet_id: &str) -> Result<TransactionListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/transactions", master_wallet_id)).await
    }

    pub async fn get_treasury_overview(&self, wallet_id: &str) -> Result<TreasuryResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/treasury", wallet_id)).await
    }

    pub async fn get_gas_price(&self, chain_id: i64) -> Result<GasPriceResponse, MasterError> {
        self.get(&format!("/api/v1/gas?chain_id={}", chain_id)).await
    }

    pub async fn get_price(&self, coin_id: &str) -> Result<PriceResponse, MasterError> {
        self.get(&format!("/api/v1/price?coin_id={}", coin_id)).await
    }

    pub async fn get_chains(&self) -> Result<ChainsResponse, MasterError> {
        self.get("/api/v1/chains").await
    }

    pub async fn get_audit_logs(&self, master_wallet_id: &str) -> Result<AuditLogListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/audit", master_wallet_id)).await
    }

    pub async fn get_policies(&self, master_wallet_id: &str) -> Result<PolicyListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/policies", master_wallet_id)).await
    }
}

// --- API response types ---

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct LoginResponse {
    pub token: String,
    pub user_id: String,
    pub email: String,
    pub role: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct WalletListResponse {
    pub wallets: Vec<MasterWallet>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct BalanceResponse {
    pub address: String,
    pub chain_id: i64,
    pub native: NativeBalance,
    #[serde(default)]
    pub tokens: Vec<TokenBalance>,
    #[serde(default)]
    pub usd_value: f64,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct NativeBalance {
    pub symbol: String,
    pub balance: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TokenBalance {
    pub symbol: String,
    pub contract: String,
    pub balance: String,
    pub decimals: i32,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TransactionResponse {
    pub transaction_hash: String,
    pub status: String,
    #[serde(default)]
    pub from: Option<String>,
    #[serde(default)]
    pub chain_id: Option<i64>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TransactionListResponse {
    pub transactions: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TreasuryResponse {
    pub address: String,
    pub chain_id: i64,
    pub total_balance: String,
    #[serde(default)]
    pub total_value_usd: f64,
    pub native_symbol: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct GasPriceResponse {
    pub chain_id: i64,
    pub gas_price: String,
    pub max_fee: String,
    pub priority_fee: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PriceResponse {
    pub coin_id: String,
    pub usd: f64,
    #[serde(default)]
    pub usd_24h_change: f64,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ChainsResponse {
    pub chains: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AuditLogListResponse {
    #[serde(rename = "audit_logs")]
    pub logs: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PolicyListResponse {
    pub policies: Vec<serde_json::Value>,
}

// --- MasterWalletService: orchestrates crypto + backend ---

pub struct MasterWalletService {
    pub client: Arc<BackendClient>,
    fee_config: RwLock<FeeConfig>,
    /// Cached derived private keys (in memory only; never persisted). Keyed by
    /// wallet_id. Production should use HSM-backed signing.
    keys: RwLock<HashMap<String, Vec<u8>>>,
}

impl MasterWalletService {
    pub fn new(backend_url: &str) -> Self {
        let config = ClientConfig { backend_url: backend_url.to_string(), jwt_token: None };
        Self {
            client: Arc::new(BackendClient::new(config)),
            fee_config: RwLock::new(FeeConfig::default()),
            keys: RwLock::new(HashMap::new()),
        }
    }

    pub fn set_token(&self, token: String) {
        self.client.set_token(token);
    }

    /// Creates a master wallet via the backend (which does the real BIP-39/44
    /// derivation + seed encryption). Returns the wallet record + mnemonic.
    pub async fn create_master_wallet(&self, name: &str, password: &str, chain_id: i64) -> Result<MasterWallet, MasterError> {
        let wallet = self.client.create_master_wallet(name, password, chain_id).await?;
        Ok(wallet)
    }

    /// Derives + caches the secp256k1 private key for a wallet from its BIP-39
    /// seed (which the caller decrypts locally). The key is kept in memory only.
    pub fn cache_wallet_key(&self, wallet_id: &str, seed: &[u8], account_index: u32) -> Result<String, MasterError> {
        let path = format!("m/44'/60'/0'/0/{}", account_index);
        let key = derive_private_key(seed, &path)?;
        let addr = private_key_to_address(&key)?;
        self.keys.write().insert(wallet_id.to_string(), key);
        Ok(addr)
    }

    /// Signs a transaction hash with a cached wallet key (real ECDSA secp256k1).
    pub fn sign_transaction(&self, wallet_id: &str, msg_hash: &[u8]) -> Result<Vec<u8>, MasterError> {
        let keys = self.keys.read();
        let key = keys.get(wallet_id).ok_or(MasterError::WalletNotFound)?;
        sign_hash(key, msg_hash)
    }

    /// Signs an EIP-191 personal message with a cached wallet key.
    pub fn sign_message(&self, wallet_id: &str, message: &[u8]) -> Result<Vec<u8>, MasterError> {
        let keys = self.keys.read();
        let key = keys.get(wallet_id).ok_or(MasterError::WalletNotFound)?;
        sign_personal_message(key, message)
    }

    /// Sets the fee configuration. Validates that no fee exceeds 20%.
    pub fn set_fees(&self, config: FeeConfig) -> Result<(), MasterError> {
        if config.withdrawal_fee_percent > 20.0 || config.swap_fee_percent > 20.0 || config.transaction_fee_percent > 20.0 {
            return Err(MasterError::FeeTooHigh);
        }
        *self.fee_config.write() = config;
        Ok(())
    }

    pub fn get_fees(&self) -> FeeConfig {
        self.fee_config.read().clone()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Canonical BIP-44 test vector: abandon...about -> m/44'/60'/0'/0/0.
    #[test]
    fn test_bip44_abandon_vector() {
        // BIP-39 seed for "abandon abandon ... about" (empty passphrase).
        let seed_hex = "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4";
        let seed = hex::decode(seed_hex).unwrap();
        let key = derive_private_key(&seed, "m/44'/60'/0'/0/0").unwrap();
        let addr = private_key_to_address(&key).unwrap();
        assert_eq!(addr.to_lowercase(), "0x9858effd232b4033e47d90003d41ec34ecaeda94");
    }

    #[test]
    fn test_mnemonic_validation() {
        assert!(validate_mnemonic("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"));
        assert!(!validate_mnemonic("invalid mnemonic words here"));
    }

    #[test]
    fn test_seed_encryption_roundtrip() {
        let seed = [42u8; 64];
        let enc = encrypt_seed(&seed, "correct-horse").unwrap();
        let dec = decrypt_seed(&enc, "correct-horse").unwrap();
        assert_eq!(dec, seed);
        // Wrong password fails.
        assert!(decrypt_seed(&enc, "wrong-password").is_err());
    }

    #[test]
    fn test_sign_and_verify() {
        let seed_hex = "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4";
        let seed = hex::decode(seed_hex).unwrap();
        let key = derive_private_key(&seed, "m/44'/60'/0'/0/0").unwrap();
        let msg_hash = Keccak256::digest(b"test message");
        let sig = sign_hash(&key, &msg_hash).unwrap();
        assert_eq!(sig.len(), 65);
        // Verify recovery works: recovering the pubkey from (hash, sig[0..64], v) yields
        // the same verifying key as the signing key.
        let signing_key = SigningKey::from_bytes(key.as_slice().into()).unwrap();
        let expected_vk = VerifyingKey::from(&signing_key);
        let recovered = recover_pubkey(&msg_hash, &sig[..64], sig[64]).unwrap();
        assert_eq!(expected_vk, recovered);
    }

    #[test]
    fn test_fee_cap() {
        let svc = MasterWalletService::new("http://localhost:8450");
        let good = FeeConfig { withdrawal_fee_percent: 1.0, swap_fee_percent: 0.5, transaction_fee_percent: 0.1, minimum_fee: 100 };
        assert!(svc.set_fees(good).is_ok());
        let bad = FeeConfig { withdrawal_fee_percent: 25.0, swap_fee_percent: 0.0, transaction_fee_percent: 0.0, minimum_fee: 0 };
        assert!(svc.set_fees(bad).is_err());
    }
}
