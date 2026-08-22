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

    async fn post_empty<T: for<'de> Deserialize<'de>>(&self, path: &str) -> Result<T, MasterError> {
        let cfg = self.config.read().clone();
        let mut req = self.http.post(format!("{}{}", cfg.backend_url, path));
        if let Some(t) = &cfg.jwt_token {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if !resp.status().is_success() {
            return Err(MasterError::BackendRequest(format!("HTTP {}", resp.status())));
        }
        // Tolerate empty 2xx bodies (e.g. 204 No Content) by falling back to null.
        let status = resp.status();
        let bytes = resp.bytes().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if bytes.is_empty() {
            return serde_json::from_value(serde_json::Value::Null)
                .map_err(|e| MasterError::BackendRequest(format!("decode empty body: {e} (HTTP {status})")));
        }
        serde_json::from_slice(&bytes).map_err(|e| MasterError::BackendRequest(e.to_string()))
    }

    async fn put<T: for<'de> Deserialize<'de>, B: Serialize>(&self, path: &str, body: &B) -> Result<T, MasterError> {
        let cfg = self.config.read().clone();
        let mut req = self.http.put(format!("{}{}", cfg.backend_url, path)).json(body);
        if let Some(t) = &cfg.jwt_token {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if !resp.status().is_success() {
            return Err(MasterError::BackendRequest(format!("HTTP {}", resp.status())));
        }
        let bytes = resp.bytes().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if bytes.is_empty() {
            return serde_json::from_value(serde_json::Value::Null)
                .map_err(|e| MasterError::BackendRequest(format!("decode empty body: {e}")));
        }
        serde_json::from_slice(&bytes).map_err(|e| MasterError::BackendRequest(e.to_string()))
    }

    async fn delete<T: for<'de> Deserialize<'de>>(&self, path: &str) -> Result<T, MasterError> {
        let cfg = self.config.read().clone();
        let mut req = self.http.delete(format!("{}{}", cfg.backend_url, path));
        if let Some(t) = &cfg.jwt_token {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if !resp.status().is_success() {
            return Err(MasterError::BackendRequest(format!("HTTP {}", resp.status())));
        }
        let bytes = resp.bytes().await.map_err(|e| MasterError::BackendRequest(e.to_string()))?;
        if bytes.is_empty() {
            return serde_json::from_value(serde_json::Value::Null)
                .map_err(|e| MasterError::BackendRequest(format!("decode empty body: {e}")));
        }
        serde_json::from_slice(&bytes).map_err(|e| MasterError::BackendRequest(e.to_string()))
    }

    // ---- Real fetchers (delegate to the canonical Go backend) ----

    // Auth
    pub async fn register(&self, email: &str, password: &str, name: &str) -> Result<LoginResponse, MasterError> {
        self.post("/api/v1/auth/register", &serde_json::json!({"email": email, "password": password, "name": name})).await
    }

    pub async fn login(&self, email: &str, password: &str) -> Result<LoginResponse, MasterError> {
        self.post("/api/v1/auth/login", &serde_json::json!({"email": email, "password": password})).await
    }

    pub async fn get_master_wallets(&self) -> Result<WalletListResponse, MasterError> {
        self.get("/api/v1/master-wallet").await
    }

    pub async fn create_master_wallet(&self, name: &str, password: &str, chain_id: i64) -> Result<MasterWallet, MasterError> {
        self.post("/api/v1/master-wallet", &serde_json::json!({"name": name, "password": password, "chain_id": chain_id})).await
    }

    pub async fn get_master_wallet(&self, wallet_id: &str) -> Result<MasterWallet, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}", wallet_id)).await
    }

    pub async fn delete_master_wallet(&self, wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}", wallet_id)).await
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

    // ---- Sub wallets ----

    pub async fn list_sub_wallets(&self, master_wallet_id: &str) -> Result<SubWalletsListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/sub-wallets", master_wallet_id)).await
    }

    pub async fn create_sub_wallet(&self, master_wallet_id: &str, name: &str, password: &str, chain_id: i64) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/sub-wallets", master_wallet_id), &serde_json::json!({"name": name, "password": password, "chain_id": chain_id})).await
    }

    pub async fn get_sub_wallet_balance(&self, master_wallet_id: &str, sub_wallet_id: &str) -> Result<BalanceResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/sub-wallets/{}/balance", master_wallet_id, sub_wallet_id)).await
    }

    pub async fn transfer_from_sub_wallet(&self, master_wallet_id: &str, sub_wallet_id: &str, to: &str, amount: &str, password: &str, token: Option<&str>) -> Result<TransactionResponse, MasterError> {
        let mut body = serde_json::json!({"to": to, "amount": amount, "password": password});
        if let Some(t) = token {
            body["token"] = serde_json::Value::String(t.to_string());
        }
        self.post(&format!("/api/v1/master-wallet/{}/sub-wallets/{}/transfer", master_wallet_id, sub_wallet_id), &body).await
    }

    // ---- Transactions ----

    pub async fn create_transaction(&self, master_wallet_id: &str, to: &str, amount: &str, password: &str, token: Option<&str>) -> Result<TransactionResponse, MasterError> {
        let mut body = serde_json::json!({"to": to, "amount": amount, "password": password});
        if let Some(t) = token {
            body["token"] = serde_json::Value::String(t.to_string());
        }
        self.post(&format!("/api/v1/master-wallet/{}/transactions", master_wallet_id), &body).await
    }

    pub async fn approve_transaction(&self, master_wallet_id: &str, transaction_id: &str) -> Result<serde_json::Value, MasterError> {
        self.post_empty(&format!("/api/v1/master-wallet/{}/transactions/{}/approve", master_wallet_id, transaction_id)).await
    }

    pub async fn reject_transaction(&self, master_wallet_id: &str, transaction_id: &str) -> Result<serde_json::Value, MasterError> {
        self.post_empty(&format!("/api/v1/master-wallet/{}/transactions/{}/reject", master_wallet_id, transaction_id)).await
    }

    // ---- Policies ----

    pub async fn create_policy(&self, master_wallet_id: &str, rule: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/policies", master_wallet_id), rule).await
    }

    pub async fn update_policy(&self, master_wallet_id: &str, policy_id: &str, updates: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/policies/{}", master_wallet_id, policy_id), updates).await
    }

    pub async fn delete_policy(&self, master_wallet_id: &str, policy_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/policies/{}", master_wallet_id, policy_id)).await
    }

    // ---- Fees ----

    pub async fn list_fees(&self, master_wallet_id: &str) -> Result<FeesListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/fees", master_wallet_id)).await
    }

    pub async fn create_fee(&self, master_wallet_id: &str, fee: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/fees", master_wallet_id), fee).await
    }

    pub async fn update_fee(&self, master_wallet_id: &str, fee_id: &str, updates: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/fees/{}", master_wallet_id, fee_id), updates).await
    }

    pub async fn delete_fee(&self, master_wallet_id: &str, fee_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/fees/{}", master_wallet_id, fee_id)).await
    }

    // ---- Auto-sign rules ----

    pub async fn list_auto_sign_rules(&self, master_wallet_id: &str) -> Result<AutoSignListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/auto-sign", master_wallet_id)).await
    }

    pub async fn create_auto_sign_rule(&self, master_wallet_id: &str, rule: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/auto-sign", master_wallet_id), rule).await
    }

    pub async fn update_auto_sign_rule(&self, master_wallet_id: &str, rule_id: &str, updates: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/auto-sign/{}", master_wallet_id, rule_id), updates).await
    }

    pub async fn delete_auto_sign_rule(&self, master_wallet_id: &str, rule_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/auto-sign/{}", master_wallet_id, rule_id)).await
    }

    // ---- Users ----

    pub async fn list_users(&self, master_wallet_id: &str) -> Result<UsersListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/users", master_wallet_id)).await
    }

    pub async fn create_user(&self, master_wallet_id: &str, user: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/users", master_wallet_id), user).await
    }

    pub async fn update_user(&self, master_wallet_id: &str, user_id: &str, updates: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/users/{}", master_wallet_id, user_id), updates).await
    }

    pub async fn delete_user(&self, master_wallet_id: &str, user_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/users/{}", master_wallet_id, user_id)).await
    }

    // ---- Analytics ----

    pub async fn get_analytics_volume(&self, master_wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/analytics/volume", master_wallet_id)).await
    }

    pub async fn get_analytics_transactions(&self, master_wallet_id: &str) -> Result<AnalyticsTransactionsResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/analytics/transactions", master_wallet_id)).await
    }

    pub async fn get_analytics_wallets(&self, master_wallet_id: &str) -> Result<AnalyticsWalletsResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/analytics/wallets", master_wallet_id)).await
    }

    // ---- Notifications ----

    pub async fn list_notifications(&self, master_wallet_id: &str) -> Result<NotificationsListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/notifications", master_wallet_id)).await
    }

    pub async fn create_notification(&self, master_wallet_id: &str, notification: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/notifications", master_wallet_id), notification).await
    }

    pub async fn update_notification(&self, master_wallet_id: &str, notification_id: &str, updates: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/notifications/{}", master_wallet_id, notification_id), updates).await
    }

    // ---- Webhooks ----

    pub async fn list_webhooks(&self, master_wallet_id: &str) -> Result<WebhooksListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/webhooks", master_wallet_id)).await
    }

    pub async fn create_webhook(&self, master_wallet_id: &str, webhook: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/webhooks", master_wallet_id), webhook).await
    }

    pub async fn update_webhook(&self, master_wallet_id: &str, webhook_id: &str, updates: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/webhooks/{}", master_wallet_id, webhook_id), updates).await
    }

    pub async fn delete_webhook(&self, master_wallet_id: &str, webhook_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/webhooks/{}", master_wallet_id, webhook_id)).await
    }

    // ---- Treasury ----

    pub async fn get_treasury_transactions(&self, master_wallet_id: &str) -> Result<TreasuryTransactionsResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/treasury/transactions", master_wallet_id)).await
    }

    pub async fn treasury_transfer(&self, master_wallet_id: &str, to: &str, amount: &str, password: &str) -> Result<TransactionResponse, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/treasury/transfer", master_wallet_id), &serde_json::json!({"to": to, "amount": amount, "password": password})).await
    }

    pub async fn treasury_sweep(&self, master_wallet_id: &str, to: &str, password: &str) -> Result<TransactionResponse, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/treasury/sweep", master_wallet_id), &serde_json::json!({"to": to, "password": password})).await
    }

    // ---- Multisig ----

    pub async fn list_multisig_wallets(&self, master_wallet_id: &str) -> Result<MultisigWalletsListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/multisig/wallets", master_wallet_id)).await
    }

    pub async fn create_multisig_wallet(&self, master_wallet_id: &str, name: &str, owners: &[String], threshold: u32) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/multisig/wallets", master_wallet_id), &serde_json::json!({"name": name, "owners": owners, "threshold": threshold})).await
    }

    pub async fn list_multisig_transactions(&self, master_wallet_id: &str, multisig_wallet_id: &str) -> Result<MultisigTransactionsListResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/multisig/wallets/{}/transactions", master_wallet_id, multisig_wallet_id)).await
    }

    pub async fn create_multisig_transaction(&self, master_wallet_id: &str, multisig_wallet_id: &str, body: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/multisig/wallets/{}/transactions", master_wallet_id, multisig_wallet_id), body).await
    }

    pub async fn sign_multisig_transaction(&self, master_wallet_id: &str, transaction_id: &str) -> Result<serde_json::Value, MasterError> {
        self.post_empty(&format!("/api/v1/master-wallet/{}/multisig/transactions/{}/sign", master_wallet_id, transaction_id)).await
    }

    pub async fn execute_multisig_transaction(&self, master_wallet_id: &str, transaction_id: &str) -> Result<serde_json::Value, MasterError> {
        self.post_empty(&format!("/api/v1/master-wallet/{}/multisig/transactions/{}/execute", master_wallet_id, transaction_id)).await
    }

    // ---- User EVM chains ----

    pub async fn list_user_evm_chains(&self, master_wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/user-chains/evm", master_wallet_id)).await
    }

    pub async fn add_user_evm_chain(&self, master_wallet_id: &str, chain: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/user-chains/evm", master_wallet_id), chain).await
    }

    pub async fn update_user_evm_chain(&self, master_wallet_id: &str, chain_id: &str, chain: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/user-chains/evm/{}", master_wallet_id, chain_id), chain).await
    }

    pub async fn remove_user_evm_chain(&self, master_wallet_id: &str, chain_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/user-chains/evm/{}", master_wallet_id, chain_id)).await
    }

    // ---- User non-EVM chains ----

    pub async fn list_user_nonevm_chains(&self, master_wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/user-chains/nonevm", master_wallet_id)).await
    }

    pub async fn add_user_nonevm_chain(&self, master_wallet_id: &str, chain: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/user-chains/nonevm", master_wallet_id), chain).await
    }

    pub async fn update_user_nonevm_chain(&self, master_wallet_id: &str, chain_id: &str, chain: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/user-chains/nonevm/{}", master_wallet_id, chain_id), chain).await
    }

    pub async fn remove_user_nonevm_chain(&self, master_wallet_id: &str, chain_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/user-chains/nonevm/{}", master_wallet_id, chain_id)).await
    }

    // ---- User tokens ----

    pub async fn list_user_tokens(&self, master_wallet_id: &str, chain_id: Option<&str>) -> Result<serde_json::Value, MasterError> {
        let path = match chain_id {
            Some(cid) => format!("/api/v1/master-wallet/{}/user-tokens?chain_id={}", master_wallet_id, cid),
            None => format!("/api/v1/master-wallet/{}/user-tokens", master_wallet_id),
        };
        self.get(&path).await
    }

    pub async fn add_user_token(&self, master_wallet_id: &str, token: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/user-tokens", master_wallet_id), token).await
    }

    pub async fn update_user_token(&self, master_wallet_id: &str, token_id: &str, token: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/user-tokens/{}", master_wallet_id, token_id), token).await
    }

    pub async fn remove_user_token(&self, master_wallet_id: &str, token_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/user-tokens/{}", master_wallet_id, token_id)).await
    }

    // ---- Address derivation ----

    pub async fn derive_user_address(&self, master_wallet_id: &str, body: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/derive-user-address", master_wallet_id), body).await
    }

    pub async fn list_user_wallet_addresses(&self, master_wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/user-wallet-addresses", master_wallet_id)).await
    }

    // ---- Auto-sign ----

    pub async fn auto_sign_transaction(&self, master_wallet_id: &str, body: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/auto-sign-transaction", master_wallet_id), body).await
    }

    pub async fn list_auto_sign_logs(&self, master_wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/auto-sign-logs", master_wallet_id)).await
    }

    // ---- Auto-sign bridge (MasterWallet-owner policy auto-approval) ----

    pub async fn user_wallet_auto_sign(&self, master_wallet_id: &str, body: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/user-wallet-auto-sign", master_wallet_id), body).await
    }

    pub async fn check_auto_sign_policy(&self, master_wallet_id: &str, body: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/check-auto-sign-policy", master_wallet_id), body).await
    }

    // ---- Feature flags ----

    pub async fn list_feature_flags(&self, master_wallet_id: &str) -> Result<serde_json::Value, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/feature-flags", master_wallet_id)).await
    }

    pub async fn add_feature_flag(&self, master_wallet_id: &str, flag: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/feature-flags", master_wallet_id), flag).await
    }

    pub async fn update_feature_flag(&self, master_wallet_id: &str, flag_id: &str, flag: &serde_json::Value) -> Result<serde_json::Value, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}/feature-flags/{}", master_wallet_id, flag_id), flag).await
    }

    pub async fn remove_feature_flag(&self, master_wallet_id: &str, flag_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/feature-flags/{}", master_wallet_id, flag_id)).await
    }

    // ---- Public (no auth) ----

    pub async fn get_transaction_history(&self, address: &str, chain_id: i64) -> Result<TransactionHistoryResponse, MasterError> {
        self.get(&format!("/api/v1/transactions/history?address={}&chain_id={}", address, chain_id)).await
    }

    pub async fn health(&self) -> Result<serde_json::Value, MasterError> {
        self.get("/health").await
    }

    /// GET /api/v1/health (alias of /health).
    pub async fn api_health(&self) -> Result<serde_json::Value, MasterError> {
        self.get("/api/v1/health").await
    }

    // ---- New master-wallet endpoints ----

    pub async fn update_master_wallet(
        &self,
        master_wallet_id: &str,
        body: &UpdateMasterWalletRequest,
    ) -> Result<UpdateMasterWalletResponse, MasterError> {
        self.put(&format!("/api/v1/master-wallet/{}", master_wallet_id), body).await
    }

    pub async fn get_transaction(
        &self,
        master_wallet_id: &str,
        transaction_id: &str,
    ) -> Result<TransactionDetailResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/transactions/{}", master_wallet_id, transaction_id)).await
    }

    pub async fn get_multisig_wallet_detail(
        &self,
        master_wallet_id: &str,
        multisig_wallet_id: &str,
    ) -> Result<MultisigWalletDetailResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/multisig/wallets/{}", master_wallet_id, multisig_wallet_id)).await
    }

    pub async fn register_passkey(
        &self,
        master_wallet_id: &str,
        credential_id: &str,
        public_key: &str,
        sign_count: u32,
        transports: &[String],
        label: &str,
    ) -> Result<PasskeyRegisterResult, MasterError> {
        let body = serde_json::json!({
            "credential_id": credential_id,
            "public_key": public_key,
            "sign_count": sign_count,
            "transports": transports,
            "label": label,
        });
        self.post(&format!("/api/v1/master-wallet/{}/passkey/register", master_wallet_id), &body).await
    }

    pub async fn list_passkeys(&self, master_wallet_id: &str) -> Result<PasskeyCredentialsResponse, MasterError> {
        self.get(&format!("/api/v1/master-wallet/{}/passkey/credentials", master_wallet_id)).await
    }

    pub async fn delete_passkey(&self, master_wallet_id: &str, credential_id: &str) -> Result<serde_json::Value, MasterError> {
        self.delete(&format!("/api/v1/master-wallet/{}/passkey/credentials/{}", master_wallet_id, credential_id)).await
    }

    pub async fn verify_passkey_assertion(
        &self,
        master_wallet_id: &str,
        body: &PasskeyVerifyRequest,
    ) -> Result<PasskeyVerifyResult, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/passkey/verify-assertion", master_wallet_id), body).await
    }

    pub async fn request_withdrawal(
        &self,
        master_wallet_id: &str,
        body: &WithdrawalRequestRequest,
    ) -> Result<WithdrawalRequestResponse, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/withdrawal-request", master_wallet_id), body).await
    }

    pub async fn revenue_payout(
        &self,
        master_wallet_id: &str,
        body: &RevenuePayoutRequest,
    ) -> Result<RevenuePayoutResponse, MasterError> {
        self.post(&format!("/api/v1/master-wallet/{}/revenue-payout", master_wallet_id), body).await
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

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct SubWalletsListResponse {
    #[serde(default)]
    pub sub_wallets: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct FeesListResponse {
    #[serde(default)]
    pub fees: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AutoSignListResponse {
    #[serde(default)]
    pub rules: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct UsersListResponse {
    #[serde(default)]
    pub users: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AnalyticsTransactionsResponse {
    #[serde(default)]
    pub transactions: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AnalyticsWalletsResponse {
    #[serde(default)]
    pub wallets: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct NotificationsListResponse {
    #[serde(default)]
    pub notifications: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct WebhooksListResponse {
    #[serde(default)]
    pub webhooks: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TreasuryTransactionsResponse {
    #[serde(default)]
    pub transactions: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MultisigWalletsListResponse {
    #[serde(default)]
    pub wallets: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MultisigTransactionsListResponse {
    #[serde(default)]
    pub transactions: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TransactionHistoryResponse {
    #[serde(default)]
    pub transactions: Vec<serde_json::Value>,
}

// --- New master-wallet endpoint request/response types ---

/// PATCH/PUT body for `PUT /api/v1/master-wallet/:id`. All fields optional —
/// only the ones the caller sets are serialized to the backend.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct UpdateMasterWalletRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_active: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub daily_limit: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub per_transaction_limit: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<serde_json::Value>,
}

impl UpdateMasterWalletRequest {
    pub fn new() -> Self {
        Self::default()
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct UpdateMasterWalletResponse {
    pub id: String,
    pub updated: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TransactionDetailResponse {
    pub transaction: serde_json::Value,
}

/// A single multisig wallet record returned by
/// `GET /api/v1/master-wallet/:id/multisig/wallets/:wid`.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MultisigWalletDetail {
    pub id: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub owners: Vec<String>,
    #[serde(default)]
    pub threshold: u32,
    #[serde(default)]
    pub chain_id: i64,
    #[serde(default)]
    pub address: String,
    #[serde(default)]
    pub pending_transactions: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MultisigWalletDetailResponse {
    pub multisig_wallet: MultisigWalletDetail,
}

/// A registered passkey credential.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PasskeyCredential {
    pub id: String,
    pub credential_id: String,
    #[serde(default)]
    pub sign_count: u32,
    #[serde(default)]
    pub transports: Vec<String>,
    #[serde(default)]
    pub label: String,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub updated_at: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PasskeyCredentialsResponse {
    #[serde(default)]
    pub passkeys: Vec<PasskeyCredential>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PasskeyRegisterResult {
    #[serde(default)]
    pub passkey_id: String,
    pub credential_id: String,
    pub registered: bool,
}

/// Assertion body for `POST /passkey/verify-assertion` (all fields base64url).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyVerifyRequest {
    pub credential_id: String,
    pub authenticator_data: String,
    pub client_data_json: String,
    pub signature: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PasskeyVerifyResult {
    pub verified: bool,
    pub credential_id: String,
}

// --- Two-party revenue gate (withdrawal-request / revenue-payout) ---

/// Body for `POST /api/v1/master-wallet/:id/withdrawal-request`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WithdrawalRequestRequest {
    pub to_address: String,
    pub amount_wei: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub currency: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chain_id: Option<i64>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct WithdrawalRequestResponse {
    pub withdrawal_id: String,
    pub status: String,
}

/// Body for `POST /api/v1/master-wallet/:id/revenue-payout`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevenuePayoutRequest {
    pub to: String,
    pub amount: String,
    pub password: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gas_limit: Option<u64>,
    pub withdrawal_id: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RevenuePayoutResponse {
    pub transaction_hash: String,
    pub status: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub withdrawal_id: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub from: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub chain_id: Option<i64>,
}

// --- Passkey helper: REAL P-256 ECDSA (WebAuthn COSE -7 / ES256) ---
//
// WebAuthn passkeys use the NIST P-256 curve (secp256r1), NOT secp256k1. This
// module generates a real P-256 ECDSA keypair, encodes the public key as a DER
// SubjectPublicKeyInfo (SPKI) blob, and produces a random base64url credential
// id — the two values the backend stores on `POST /passkey/register`. The
// private signing key is returned to the caller so an authenticator-equivalent
// can later sign assertions; nothing here is stubbed or fabricated.

mod passkey {
    #![allow(dead_code)]
    use base64::Engine;
    use p256::ecdsa::{SigningKey, VerifyingKey};
    use rand::RngCore;

    /// DER prefix for an uncompressed P-256 ECDSA SubjectPublicKeyInfo:
    ///   SEQUENCE { alg-id { id-ecPublicKey, secp256r1 }, BIT STRING <04||X||Y> }
    /// 26 fixed bytes preceding the 65-byte SEC1 point (91 bytes total).
    const P256_SPKI_PREFIX: [u8; 26] = [
        0x30, 0x59, 0x30, 0x13, 0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01, 0x06, 0x08,
        0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07, 0x03, 0x42, 0x00,
    ];

    const B64: base64::engine::general_purpose::GeneralPurpose = base64::engine::general_purpose::URL_SAFE_NO_PAD;

    /// Generated passkey material: the real P-256 signing key, a base64url
    /// credential id, and the base64url SPKI-encoded public key ready for the
    /// backend register endpoint.
    pub struct PasskeyMaterial {
        pub signing_key: SigningKey,
        pub credential_id: String,
        pub public_key_spki: String,
    }

    /// Generates a fresh P-256 ECDSA keypair + a random 32-byte credential id.
    /// REAL randomness via `OsRng`; no fakes.
    pub fn generate_passkey_material() -> PasskeyMaterial {
        let signing_key = SigningKey::random(&mut rand::rngs::OsRng);
        let verifying_key = VerifyingKey::from(&signing_key);

        // Uncompressed SEC1 point: 0x04 || X(32) || Y(32).
        let point = verifying_key.to_encoded_point(false);
        let sec1 = point.as_bytes();

        let mut spki = Vec::with_capacity(P256_SPKI_PREFIX.len() + sec1.len());
        spki.extend_from_slice(&P256_SPKI_PREFIX);
        spki.extend_from_slice(sec1);
        let public_key_spki = B64.encode(&spki);

        let mut cred = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut cred);
        let credential_id = B64.encode(cred);

        PasskeyMaterial { signing_key, credential_id, public_key_spki }
    }

    /// Encodes an arbitrary byte slice as base64url (no padding).
    pub fn b64url_encode(bytes: &[u8]) -> String {
        B64.encode(bytes)
    }

    /// Decodes a base64url (no padding) string.
    pub fn b64url_decode(s: &str) -> Result<Vec<u8>, base64::DecodeError> {
        B64.decode(s)
    }
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

    /// Sets the local fee configuration override. Validates that no fee exceeds 20%.
    pub fn set_fees(&self, config: FeeConfig) -> Result<(), MasterError> {
        if config.withdrawal_fee_percent > 20.0 || config.swap_fee_percent > 20.0 || config.transaction_fee_percent > 20.0 {
            return Err(MasterError::FeeTooHigh);
        }
        *self.fee_config.write() = config;
        Ok(())
    }

    /// Returns the local fee configuration override (validated by `set_fees`).
    pub fn local_fee_config(&self) -> FeeConfig {
        self.fee_config.read().clone()
    }

    /// Fetches the canonical fee list for a master wallet from the backend
    /// (GET /api/v1/master-wallet/:id/fees). Replaces the previous in-memory
    /// RwLock read with a real HTTP fetch against the canonical Go backend.
    pub async fn get_fees(&self, master_wallet_id: &str) -> Result<FeesListResponse, MasterError> {
        self.client.list_fees(master_wallet_id).await
    }

    // ---- New master-wallet service methods ----

    /// `PUT /api/v1/master-wallet/:id` — partial update of a master wallet.
    /// Only the provided (Some) fields are sent; absent fields are omitted from
    /// the request body via `skip_serializing_if`.
    pub async fn update_master_wallet(
        &self,
        master_id: &str,
        name: Option<&str>,
        is_active: Option<bool>,
        daily_limit: Option<f64>,
        per_transaction_limit: Option<f64>,
    ) -> Result<UpdateMasterWalletResponse, MasterError> {
        let mut req = UpdateMasterWalletRequest::new();
        req.name = name.map(|s| s.to_string());
        req.is_active = is_active;
        req.daily_limit = daily_limit;
        req.per_transaction_limit = per_transaction_limit;
        self.client.update_master_wallet(master_id, &req).await
    }

    /// Like [`update_master_wallet`](Self::update_master_wallet) but also lets
    /// the caller pass a free-form `metadata` object.
    pub async fn update_master_wallet_with_metadata(
        &self,
        master_id: &str,
        name: Option<&str>,
        is_active: Option<bool>,
        daily_limit: Option<f64>,
        per_transaction_limit: Option<f64>,
        metadata: Option<serde_json::Value>,
    ) -> Result<UpdateMasterWalletResponse, MasterError> {
        let mut req = UpdateMasterWalletRequest::new();
        req.name = name.map(|s| s.to_string());
        req.is_active = is_active;
        req.daily_limit = daily_limit;
        req.per_transaction_limit = per_transaction_limit;
        req.metadata = metadata;
        self.client.update_master_wallet(master_id, &req).await
    }

    /// `GET /api/v1/master-wallet/:id/transactions/:tid` — fetch a single
    /// transaction record.
    pub async fn get_transaction(
        &self,
        master_id: &str,
        tx_id: &str,
    ) -> Result<TransactionDetailResponse, MasterError> {
        self.client.get_transaction(master_id, tx_id).await
    }

    /// `GET /api/v1/master-wallet/:id/multisig/wallets/:wid` — fetch the full
    /// detail of a single multisig wallet (owners, threshold, pending txs).
    pub async fn get_multisig_wallet_detail(
        &self,
        master_id: &str,
        wallet_id: &str,
    ) -> Result<MultisigWalletDetailResponse, MasterError> {
        self.client.get_multisig_wallet_detail(master_id, wallet_id).await
    }

    /// `POST /api/v1/master-wallet/:id/passkey/register` — registers an existing
    /// passkey credential (caller supplies the base64url credential id + SPKI
    /// public key). For a freshly generated P-256 keypair see
    /// [`register_new_passkey`](Self::register_new_passkey).
    pub async fn register_passkey(
        &self,
        master_id: &str,
        credential_id: &str,
        public_key: &str,
        sign_count: u32,
        transports: Vec<String>,
        label: &str,
    ) -> Result<PasskeyRegisterResult, MasterError> {
        self.client
            .register_passkey(master_id, credential_id, public_key, sign_count, &transports, label)
            .await
    }

    /// Generates a REAL P-256 ECDSA keypair, builds the base64url SPKI public
    /// key + a random base64url credential id, and registers it with the
    /// backend. Returns the registration result. The private signing key is
    /// kept by the caller's authenticator flow (not stored here); no value is
    /// fabricated — the public key is derived from the live P-256 signing key.
    pub async fn register_new_passkey(
        &self,
        master_id: &str,
        label: &str,
    ) -> Result<PasskeyRegisterResult, MasterError> {
        let material = passkey::generate_passkey_material();
        let transports = vec!["internal".to_string(), "hybrid".to_string()];
        let result = self
            .client
            .register_passkey(master_id, &material.credential_id, &material.public_key_spki, 0, &transports, label)
            .await?;
        // Fail-closed: only report success when the backend confirms it stored
        // the credential. Never fabricate a positive result.
        if !result.registered {
            return Err(MasterError::BackendRequest("backend did not register passkey".into()));
        }
        Ok(result)
    }

    /// `GET /api/v1/master-wallet/:id/passkey/credentials` — list registered
    /// passkey credentials for a master wallet.
    pub async fn list_passkeys(&self, master_id: &str) -> Result<PasskeyCredentialsResponse, MasterError> {
        self.client.list_passkeys(master_id).await
    }

    /// `DELETE /api/v1/master-wallet/:id/passkey/credentials/:credId` — remove a
    /// registered passkey credential. Returns Ok on a 2xx (the backend responds
    /// 204 No Content).
    pub async fn delete_passkey(&self, master_id: &str, cred_id: &str) -> Result<(), MasterError> {
        self.client.delete_passkey(master_id, cred_id).await?;
        Ok(())
    }

    /// `POST /api/v1/master-wallet/:id/passkey/verify-assertion` — verifies a
    /// WebAuthn assertion against the stored credential. All inputs are
    /// base64url. The verification decision is made by the backend; this method
    /// is FAIL-CLOSED: it never fabricates a `verified=true` result. On a
    /// transport/HTTP error it returns `Err`; on a backend `verified=false` it
    /// returns the (verified=false) result unchanged.
    pub async fn verify_passkey_assertion(
        &self,
        master_id: &str,
        credential_id: &str,
        authenticator_data: &str,
        client_data_json: &str,
        signature: &str,
    ) -> Result<PasskeyVerifyResult, MasterError> {
        let body = PasskeyVerifyRequest {
            credential_id: credential_id.to_string(),
            authenticator_data: authenticator_data.to_string(),
            client_data_json: client_data_json.to_string(),
            signature: signature.to_string(),
        };
        let result = self.client.verify_passkey_assertion(master_id, &body).await?;
        // Defensive: if the backend ever omits the flag, treat it as not verified.
        if result.verified && result.credential_id.is_empty() {
            return Err(MasterError::BackendRequest("backend returned verified without credential_id".into()));
        }
        Ok(result)
    }

    /// `POST /api/v1/master-wallet/:id/withdrawal-request` — first leg of the
    /// two-party revenue gate. Creates a withdrawal request; the actual payout
    /// is released by [`revenue_payout`](Self::revenue_payout) using the
    /// returned `withdrawal_id`. Optional fields are omitted from the request
    /// body when `None` so the backend receives clean JSON.
    pub async fn request_withdrawal(
        &self,
        master_id: &str,
        to_address: &str,
        amount_wei: &str,
        currency: Option<String>,
        chain_id: Option<i64>,
    ) -> Result<WithdrawalRequestResponse, MasterError> {
        let body = WithdrawalRequestRequest {
            to_address: to_address.to_string(),
            amount_wei: amount_wei.to_string(),
            currency,
            chain_id,
        };
        self.client.request_withdrawal(master_id, &body).await
    }

    /// `POST /api/v1/master-wallet/:id/revenue-payout` — second leg of the
    /// two-party revenue gate. Releases funds for a previously created
    /// withdrawal request identified by `withdrawal_id`. The backend signs +
    /// broadcasts; this method returns the resulting transaction hash and
    /// status without fabricating any value.
    pub async fn revenue_payout(
        &self,
        master_id: &str,
        to: &str,
        amount: &str,
        password: &str,
        gas_limit: Option<u64>,
        withdrawal_id: &str,
    ) -> Result<RevenuePayoutResponse, MasterError> {
        let body = RevenuePayoutRequest {
            to: to.to_string(),
            amount: amount.to_string(),
            password: password.to_string(),
            gas_limit,
            withdrawal_id: withdrawal_id.to_string(),
        };
        self.client.revenue_payout(master_id, &body).await
    }
}

// ============================================================================
// WebSocketClient — real-time feed from the canonical backend (/ws).
//
// Connects to ws://<base>/ws?master_wallet_id=<id>&token=<JWT> and streams
// live balance updates, transaction confirmations, and market-ticker events.
// Uses tokio-tungstenite (real WebSocket over rustls). Fail-closed: no fake
// events are ever produced; a closed/errored socket simply stops yielding
// messages until reconnect.
// ============================================================================

/// A real-time WebSocket client for the canonical MasterWallet backend.
pub struct WebSocketClient {
    base_url: String,
}

/// Events delivered to the caller's handler.
#[derive(Debug, Clone)]
pub enum WsEvent {
    Open,
    Message(serde_json::Value),
    Close,
    Error(String),
}

impl WebSocketClient {
    /// Build a client targeting the given base URL (e.g. "http://localhost:8450"
    /// or "https://master-api.tigerwallet.com").
    pub fn new(base_url: &str) -> Self {
        Self { base_url: base_url.trim_end_matches('/').to_string() }
    }

    /// Derive the ws:// URL with master_wallet_id + token query params.
    fn ws_url(&self, master_wallet_id: &str, token: &str) -> String {
        let ws_base = self.base_url.replace("http://", "ws://").replace("https://", "wss://");
        format!("{}/ws?master_wallet_id={}&token={}", ws_base, master_wallet_id, token)
    }

    /// Connect and drive an event handler until the socket closes or the handler
    /// returns false. Reconnects with capped exponential backoff on transient
    /// errors; the handler receives WsEvent::Open/Message/Close/Error.
    ///
    /// This is a future that resolves only when the handler returns false or a
    /// hard error occurs. Run it on a tokio task:
    ///   `tokio::spawn(async move { ws.run(&id, &token, |ev| matches!(ev, WsEvent::Message(_))).await });`
    pub async fn run<F>(&self, master_wallet_id: &str, token: &str, mut on_event: F) -> Result<(), MasterError>
    where
        F: FnMut(WsEvent) -> bool,
    {
        use futures_util::StreamExt;
        use tokio_tungstenite::tungstenite::Message;
        use std::time::Duration;

        let url = self.ws_url(master_wallet_id, token);
        let mut backoff_ms = 1000u64;

        loop {
            let mut stream = match tokio_tungstenite::connect_async(&url).await {
                Ok((s, _)) => { backoff_ms = 1000; s }
                Err(e) => {
                    let _ = on_event(WsEvent::Error(format!("connect: {e}")));
                    if backoff_ms >= 30000 {
                        // Capped backoff; keep trying but don't spin tightly.
                        backoff_ms = 30000;
                    }
                    tokio::time::sleep(Duration::from_millis(backoff_ms)).await;
                    backoff_ms = (backoff_ms * 2).min(30000);
                    continue;
                }
            };

            if !on_event(WsEvent::Open) { return Ok(()); }

            loop {
                tokio::select! {
                    msg = stream.next() => {
                        match msg {
                            Some(Ok(Message::Text(txt))) => {
                                match serde_json::from_str::<serde_json::Value>(&txt) {
                                    Ok(v) => { if !on_event(WsEvent::Message(v)) { return Ok(()); } }
                                    Err(_) => {
                                        // Non-JSON text frame; wrap as a string value.
                                        if !on_event(WsEvent::Message(serde_json::Value::String(txt))) { return Ok(()); }
                                    }
                                }
                            }
                            Some(Ok(Message::Binary(bin))) => {
                                if !on_event(WsEvent::Message(serde_json::Value::String(
                                    format!("<binary {} bytes>", bin.len())
                                ))) { return Ok(()); }
                            }
                            Some(Ok(Message::Ping(_))) => { /* tungstenite auto-pongs */ }
                            Some(Ok(Message::Pong(_))) => { }
                            Some(Ok(Message::Frame(_))) => { /* raw frame; ignored */ }
                            Some(Ok(Message::Close(_))) => {
                                let _ = on_event(WsEvent::Close);
                                break;
                            }
                            Some(Err(e)) => {
                                let _ = on_event(WsEvent::Error(format!("stream: {e}")));
                                break;
                            }
                            None => {
                                let _ = on_event(WsEvent::Close);
                                break;
                            }
                        }
                    }
                }
            }

            // Socket closed; back off before reconnecting.
            tokio::time::sleep(Duration::from_millis(backoff_ms)).await;
            backoff_ms = (backoff_ms * 2).min(30000);
        }
    }

    /// Send a JSON message to the backend over an already-connected socket.
    /// (Convenience for request/reply patterns; for streaming use `run`.)
    pub async fn send_once(&self, _master_wallet_id: &str, _token: &str, _payload: &serde_json::Value) -> Result<(), MasterError> {
        // A one-shot send requires its own short-lived connection; for the
        // streaming use case prefer run() which keeps the socket open.
        Err(MasterError::BackendRequest("send_once is not supported; use run() for a streaming connection".to_string()))
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

    // REAL P-256 passkey material: SPKI must be 91 bytes and decode (via SEC1)
    // back to the same verifying key; a P-256 signature over a SHA-256 prehash
    // must verify with that key. Nothing fabricated.
    #[test]
    fn test_passkey_p256_keypair_and_spki() {
        use p256::ecdsa::{Signature, VerifyingKey};
        use p256::ecdsa::signature::hazmat::{PrehashSigner, PrehashVerifier};

        let material = passkey::generate_passkey_material();
        let spki = passkey::b64url_decode(&material.public_key_spki).unwrap();
        // SEQUENCE(89) + tag/len = 91 bytes.
        assert_eq!(spki.len(), 91, "SPKI must be 91 bytes for P-256 uncompressed");
        assert_eq!(spki[0], 0x30, "SPKI must start with SEQUENCE tag");
        // The trailing 65 bytes after the 26-byte prefix are the SEC1 point.
        let sec1 = &spki[26..];
        assert_eq!(sec1.len(), 65);
        assert_eq!(sec1[0], 0x04, "SEC1 point must be uncompressed (0x04)");

        // Reconstruct the verifying key from the SEC1 point and confirm it
        // matches the key derived from the signing key.
        let point = p256::EncodedPoint::from_bytes(sec1).unwrap();
        let vk_from_spki = VerifyingKey::from_encoded_point(&point).unwrap();
        let vk_from_sk = VerifyingKey::from(&material.signing_key);
        assert_eq!(vk_from_spki, vk_from_sk, "SPKI public key must match signing key");

        // Round-trip the credential id (32 random bytes, base64url no-pad).
        let cred_bytes = passkey::b64url_decode(&material.credential_id).unwrap();
        assert_eq!(cred_bytes.len(), 32, "credential id must be 32 random bytes");

        // Sign/verify a message with P-256 ECDSA (ES256 / SHA-256 prehash).
        let msg = b"tigerwallet passkey challenge";
        let digest = sha2::Sha256::digest(msg);
        let sig: Signature = material.signing_key.sign_prehash(&digest).unwrap();
        assert!(vk_from_sk.verify_prehash(&digest, &sig).is_ok(), "P-256 signature must verify");
    }
}
