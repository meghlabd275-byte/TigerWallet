//! multisig.rs — threshold multisig wallets (owner ECDSA signature collection,
//! on-chain execute when threshold met). Port of master_wallet/backend/multisig.go.
//!
//! Fail-closed: execution requires a real executor key (env
//! MASTER_WALLET_TREASURY_KEY_HEX) and a real RPC endpoint (per-chain env var).
//! No fabricated hashes, no stubs.

use actix_web::{web, HttpRequest, HttpResponse};
use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use sqlx::types::Json;

use crate::evm_tx;
use crate::AppState;

/// JWT gate (self-hosted pillar convention: all routes are authenticated).
fn authed(req: &HttpRequest, state: &AppState) -> Option<HttpResponse> {
    match crate::require_auth(req.headers(), &state.jwt_secret) {
        Ok(_) => None,
        Err(_) => Some(HttpResponse::Unauthorized().json(serde_json::json!({"error": "unauthorized"}))),
    }
}

#[derive(Deserialize)]
pub struct CreateMultisigWalletRequest {
    pub name: String,
    pub chain_id: i64,
    pub owners: Vec<String>,
    pub threshold: i32,
}

#[derive(Deserialize)]
pub struct CreateMultisigTxRequest {
    pub wallet_id: String,
    pub to_address: String,
    pub value_wei: String,
    #[serde(default)]
    pub data: String,
}

#[derive(Deserialize)]
pub struct SignMultisigRequest {
    pub owner_address: String,
    pub signature: String,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct MultisigSignature {
    pub signer: String,
    pub signature: String,
}

#[derive(Serialize, sqlx::FromRow)]
pub struct MultisigWalletRow {
    pub id: String,
    pub name: String,
    pub chain_id: i64,
    pub threshold: i32,
    pub owners: Vec<String>,
    pub nonce: i64,
    pub created_at: String,
}

#[derive(Serialize, sqlx::FromRow)]
pub struct MultisigTxRow {
    pub id: String,
    pub wallet_id: String,
    pub to_address: String,
    pub value_wei: String,
    pub data: String,
    pub nonce: i64,
    pub signatures: Json<Vec<MultisigSignature>>,
    pub threshold: i32,
    pub status: String,
    pub tx_hash: Option<String>,
    pub chain_id: i64,
    pub created_at: String,
}

fn err400(msg: &str) -> HttpResponse {
    HttpResponse::BadRequest().json(serde_json::json!({"error": msg}))
}

fn err503(msg: &str) -> HttpResponse {
    HttpResponse::ServiceUnavailable().json(serde_json::json!({"error": msg}))
}

/// CreateMultisigWallet POST /api/v1/multisig/wallets
pub async fn create_multisig_wallet(
    state: web::Data<AppState>,
    http_req: HttpRequest,
    req: web::Json<CreateMultisigWalletRequest>,
) -> HttpResponse {
    if let Some(r) = authed(&http_req, &state) {
        return r;
    }
    if req.name.trim().is_empty() {
        return err400("name required");
    }
    if req.owners.len() < 2 {
        return err400("at least 2 owners required");
    }
    if req.threshold < 1 || req.threshold as usize > req.owners.len() {
        return err400("invalid threshold");
    }
    // Validate + dedupe owners (by lowercase address).
    let mut seen = std::collections::HashSet::new();
    let mut owners = Vec::with_capacity(req.owners.len());
    for o in &req.owners {
        let o = o.trim();
        if evm_tx::parse_hex_fixed::<20>(o).is_err() {
            return err400(&format!("invalid owner address: {o}"));
        }
        let lower = o.to_lowercase();
        if seen.insert(lower.clone()) {
            owners.push(lower);
        }
    }
    if (req.threshold as usize) > owners.len() {
        return err400("threshold exceeds unique owner count");
    }

    let id = uuid::Uuid::new_v4().to_string();
    let res = sqlx::query(
        "INSERT INTO shmw_multisig_wallets (id, name, chain_id, owners, threshold, nonce) \
         VALUES ($1,$2,$3,$4,$5,0)",
    )
    .bind(&id)
    .bind(req.name.trim())
    .bind(req.chain_id)
    .bind(&owners)
    .bind(req.threshold)
    .execute(&state.pool)
    .await;
    match res {
        Ok(_) => HttpResponse::Created().json(serde_json::json!({
            "id": id,
            "name": req.name.trim(),
            "chain_id": req.chain_id,
            "owners": owners,
            "threshold": req.threshold,
            "nonce": 0,
        })),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

/// ListMultisigWallets GET /api/v1/multisig/wallets
pub async fn list_multisig_wallets(state: web::Data<AppState>, http_req: HttpRequest) -> HttpResponse {
    if let Some(r) = authed(&http_req, &state) {
        return r;
    }
    let rows = sqlx::query_as::<_, MultisigWalletRow>(
        "SELECT id, name, chain_id, threshold, owners, nonce, created_at::text AS created_at \
         FROM shmw_multisig_wallets ORDER BY created_at DESC LIMIT 200",
    )
    .fetch_all(&state.pool)
    .await;
    match rows {
        Ok(w) => {
            let count = w.len();
            HttpResponse::Ok().json(serde_json::json!({"wallets": w, "count": count}))
        }
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

/// CreateMultisigTransaction POST /api/v1/multisig/transactions
pub async fn create_multisig_transaction(
    state: web::Data<AppState>,
    http_req: HttpRequest,
    req: web::Json<CreateMultisigTxRequest>,
) -> HttpResponse {
    if let Some(r) = authed(&http_req, &state) {
        return r;
    }
    if evm_tx::parse_hex_fixed::<20>(&req.to_address).is_err() {
        return err400("invalid to_address");
    }
    if evm_tx::dec_to_be(&req.value_wei).is_err() {
        return err400("invalid value_wei");
    }
    let data_hex = req.data.trim_start_matches("0x");
    if !data_hex.is_empty() && hex::decode(data_hex).is_err() {
        return err400("invalid data hex");
    }

    let wallet = sqlx::query_as::<_, (i64, i32, i64)>(
        "SELECT chain_id, threshold, nonce FROM shmw_multisig_wallets WHERE id=$1",
    )
    .bind(&req.wallet_id)
    .fetch_optional(&state.pool)
    .await;
    let (chain_id, threshold, nonce) = match wallet {
        Ok(Some(w)) => w,
        Ok(None) => return HttpResponse::NotFound().json(serde_json::json!({"error": "multisig wallet not found"})),
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    };

    let id = uuid::Uuid::new_v4().to_string();
    let to_lower = req.to_address.to_lowercase();
    let res = sqlx::query(
        "INSERT INTO shmw_multisig_transactions \
         (id, wallet_id, to_address, value_wei, data, nonce, signatures, threshold, status, chain_id) \
         VALUES ($1,$2,$3,$4,$5,$6,'[]'::jsonb,$7,'pending',$8)",
    )
    .bind(&id)
    .bind(&req.wallet_id)
    .bind(&to_lower)
    .bind(&req.value_wei)
    .bind(&req.data)
    .bind(nonce)
    .bind(threshold)
    .bind(chain_id)
    .execute(&state.pool)
    .await;
    match res {
        Ok(_) => HttpResponse::Created().json(serde_json::json!({
            "id": id,
            "wallet_id": req.wallet_id,
            "to_address": to_lower,
            "value_wei": req.value_wei,
            "data": req.data,
            "nonce": nonce,
            "threshold": threshold,
            "chain_id": chain_id,
            "status": "pending",
        })),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

/// ListMultisigTransactions GET /api/v1/multisig/wallets/:id/transactions
pub async fn list_multisig_transactions(
    state: web::Data<AppState>,
    http_req: HttpRequest,
    path: web::Path<String>,
) -> HttpResponse {
    if let Some(r) = authed(&http_req, &state) {
        return r;
    }
    let wallet_id = path.into_inner();
    let rows = sqlx::query_as::<_, MultisigTxRow>(
        "SELECT id, wallet_id, to_address, value_wei, data, nonce, signatures, threshold, \
         status, tx_hash, chain_id, created_at::text AS created_at \
         FROM shmw_multisig_transactions WHERE wallet_id=$1 ORDER BY created_at DESC LIMIT 200",
    )
    .bind(&wallet_id)
    .fetch_all(&state.pool)
    .await;
    match rows {
        Ok(t) => {
            let count = t.len();
            HttpResponse::Ok().json(serde_json::json!({"transactions": t, "count": count}))
        }
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

/// The digest that owners sign: keccak256(to || value32 || data || nonce32 || chainID32).
pub fn multisig_tx_digest(
    to: &str,
    value_wei: &str,
    data_hex: &str,
    nonce: i64,
    chain_id: i64,
) -> Result<[u8; 32], String> {
    let to_bytes = evm_tx::parse_hex_fixed::<20>(to)?;
    let value = evm_tx::dec_to_be(value_wei)?;
    if value.len() > 32 {
        return Err("value_wei exceeds 256 bits".into());
    }
    let data = hex::decode(data_hex.trim_start_matches("0x")).map_err(|e| e.to_string())?;

    let mut preimage = Vec::with_capacity(20 + 32 + data.len() + 64);
    preimage.extend_from_slice(&to_bytes);
    preimage.extend_from_slice(&[0u8; 32]);
    let vpos = preimage.len() - value.len();
    preimage[vpos..].copy_from_slice(&value);
    preimage.extend_from_slice(&data);
    preimage.extend_from_slice(&[0u8; 24]);
    preimage.extend_from_slice(&nonce.to_be_bytes());
    preimage.extend_from_slice(&[0u8; 24]);
    preimage.extend_from_slice(&chain_id.to_be_bytes());
    Ok(Keccak256::digest(&preimage).into())
}

/// SignMultisigTransaction POST /api/v1/multisig/transactions/:tx_id/sign
///
/// Verifies the owner's ECDSA signature against the tx digest via ecrecover,
/// confirms the signer is a wallet owner, dedupes by signer, and persists.
pub async fn sign_multisig_transaction(
    state: web::Data<AppState>,
    http_req: HttpRequest,
    path: web::Path<String>,
    req: web::Json<SignMultisigRequest>,
) -> HttpResponse {
    if let Some(r) = authed(&http_req, &state) {
        return r;
    }
    let tx_id = path.into_inner();

    let row = sqlx::query_as::<_, MultisigTxRow>(
        "SELECT id, wallet_id, to_address, value_wei, data, nonce, signatures, threshold, \
         status, tx_hash, chain_id, created_at::text AS created_at \
         FROM shmw_multisig_transactions WHERE id=$1",
    )
    .bind(&tx_id)
    .fetch_optional(&state.pool)
    .await;
    let tx = match row {
        Ok(Some(t)) => t,
        Ok(None) => return HttpResponse::NotFound().json(serde_json::json!({"error": "multisig transaction not found"})),
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    };
    if tx.status == "executed" {
        return err400("transaction already executed");
    }

    let owners = sqlx::query_as::<_, (Vec<String>,)>(
        "SELECT owners FROM shmw_multisig_wallets WHERE id=$1",
    )
    .bind(&tx.wallet_id)
    .fetch_optional(&state.pool)
    .await;
    let owners = match owners {
        Ok(Some((o,))) => o,
        Ok(None) => return HttpResponse::NotFound().json(serde_json::json!({"error": "multisig wallet not found"})),
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    };

    let owner = req.owner_address.trim().to_lowercase();
    if !owners.iter().any(|o| o == &owner) {
        return HttpResponse::Forbidden().json(serde_json::json!({"error": "signer is not an owner of this wallet"}));
    }

    // Decode + verify the owner signature against the tx digest.
    let sig_bytes = match hex::decode(req.signature.trim_start_matches("0x")) {
        Ok(b) if b.len() == 65 => b,
        _ => return err400("signature must be 65 bytes (hex)"),
    };
    let digest = match multisig_tx_digest(&tx.to_address, &tx.value_wei, &tx.data, tx.nonce, tx.chain_id) {
        Ok(d) => d,
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": e})),
    };
    let signer = match evm_tx::ecrecover(&digest, &sig_bytes) {
        Ok(s) => s.to_lowercase(),
        Err(_) => return err400("invalid signature"),
    };
    if signer != owner {
        return err400("signature does not match owner_address");
    }

    // Dedupe by signer: re-signing is idempotent.
    let mut signatures = tx.signatures.0.clone();
    if signatures.iter().any(|s| s.signer == signer) {
        return HttpResponse::Ok().json(serde_json::json!({
            "id": tx.id,
            "signatures": signatures,
            "signatures_count": signatures.len(),
            "threshold": tx.threshold,
            "status": tx.status,
            "deduplicated": true,
        }));
    }
    signatures.push(MultisigSignature {
        signer: signer.clone(),
        signature: format!("0x{}", hex::encode(&sig_bytes)),
    });

    let res = sqlx::query(
        "UPDATE shmw_multisig_transactions SET signatures=$2 WHERE id=$1",
    )
    .bind(&tx.id)
    .bind(Json(&signatures))
    .execute(&state.pool)
    .await;
    match res {
        Ok(_) => HttpResponse::Ok().json(serde_json::json!({
            "id": tx.id,
            "signatures": signatures,
            "signatures_count": signatures.len(),
            "threshold": tx.threshold,
            "status": tx.status,
        })),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

/// ExecuteMultisigTransaction POST /api/v1/multisig/transactions/:tx_id/execute
///
/// Fail-closed: requires threshold met, executor key env set, and a live RPC.
/// Builds, signs (EIP-1559), and broadcasts a real transaction.
pub async fn execute_multisig_transaction(
    state: web::Data<AppState>,
    http_req: HttpRequest,
    path: web::Path<String>,
) -> HttpResponse {
    if let Some(r) = authed(&http_req, &state) {
        return r;
    }
    let tx_id = path.into_inner();
    let row = sqlx::query_as::<_, MultisigTxRow>(
        "SELECT id, wallet_id, to_address, value_wei, data, nonce, signatures, threshold, \
         status, tx_hash, chain_id, created_at::text AS created_at \
         FROM shmw_multisig_transactions WHERE id=$1",
    )
    .bind(&tx_id)
    .fetch_optional(&state.pool)
    .await;
    let tx = match row {
        Ok(Some(t)) => t,
        Ok(None) => return HttpResponse::NotFound().json(serde_json::json!({"error": "multisig transaction not found"})),
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    };
    if tx.status == "executed" {
        return err400("transaction already executed");
    }
    if (tx.signatures.0.len() as i32) < tx.threshold {
        return err400(&format!(
            "threshold not met: {}/{}",
            tx.signatures.0.len(),
            tx.threshold
        ));
    }

    // Fail-closed: executor key must be configured.
    let executor_key_hex = match std::env::var("MASTER_WALLET_TREASURY_KEY_HEX") {
        Ok(k) if !k.trim().is_empty() => k.trim().to_string(),
        _ => {
            return err503("executor key not configured: set MASTER_WALLET_TREASURY_KEY_HEX")
        }
    };
    let executor_key: [u8; 32] = match hex::decode(executor_key_hex.trim_start_matches("0x")) {
        Ok(b) if b.len() == 32 => b.try_into().unwrap(),
        _ => return err503("invalid executor key format (expected 32-byte hex)"),
    };
    let executor_addr = match crate::crypto::private_key_to_address(&executor_key) {
        Ok(a) => a,
        Err(e) => return err503(&format!("invalid executor key: {e}")),
    };

    // Fail-closed: RPC must be configured for this chain.
    let rpc_url = crate::chain_rpc_endpoint(tx.chain_id);
    if rpc_url.is_empty() {
        return err503(&format!("no RPC endpoint configured for chain_id {}", tx.chain_id));
    }

    let nonce = match evm_tx::rpc_get_nonce(&rpc_url, &executor_addr).await {
        Ok(n) => n,
        Err(e) => return HttpResponse::BadGateway().json(serde_json::json!({"error": format!("rpc nonce fetch failed: {e}")})),
    };
    let tip = match evm_tx::rpc_max_priority_fee(&rpc_url).await {
        Ok(f) => f,
        Err(e) => return HttpResponse::BadGateway().json(serde_json::json!({"error": format!("rpc fee fetch failed: {e}")})),
    };
    // maxFee = 2 * tip as a conservative cap (matches the canonical backend).
    let cap = match multiply_dec(&tip, 2) {
        Ok(c) => c,
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": e})),
    };

    let data = hex::decode(tx.data.trim_start_matches("0x")).unwrap_or_default();
    let params = evm_tx::TxParams {
        chain_id: tx.chain_id as u64,
        nonce,
        gas_limit: 200_000,
        to: tx.to_address.clone(),
        value_wei: tx.value_wei.clone(),
        data,
        gas_price_wei: String::new(),
        max_priority_fee_wei: tip,
        max_fee_wei: cap,
        eip1559: true,
    };
    let signed = match evm_tx::sign_transaction(&executor_key, &params) {
        Ok(s) => s,
        Err(e) => return HttpResponse::InternalServerError().json(serde_json::json!({"error": format!("signing failed: {e}")})),
    };
    let tx_hash = match evm_tx::rpc_send_raw_transaction(&rpc_url, &signed.raw).await {
        Ok(h) => h,
        Err(e) => return HttpResponse::BadGateway().json(serde_json::json!({"error": format!("broadcast failed: {e}")})),
    };

    let _ = sqlx::query(
        "UPDATE shmw_multisig_transactions SET status='executed', tx_hash=$2 WHERE id=$1",
    )
    .bind(&tx.id)
    .bind(&tx_hash)
    .execute(&state.pool)
    .await;
    let _ = sqlx::query("UPDATE shmw_multisig_wallets SET nonce=nonce+1 WHERE id=$1")
        .bind(&tx.wallet_id)
        .execute(&state.pool)
        .await;

    HttpResponse::Ok().json(serde_json::json!({
        "id": tx.id,
        "status": "executed",
        "transaction_hash": tx_hash,
        "executor": executor_addr,
    }))
}

/// Multiply a decimal-string integer by a small factor.
fn multiply_dec(s: &str, factor: u64) -> Result<String, String> {
    let be = evm_tx::dec_to_be(s)?;
    // base-256 * factor
    let mut out = be.clone();
    let mut carry: u64 = 0;
    for b in out.iter_mut().rev() {
        let cur = (*b as u64) * factor + carry;
        *b = (cur & 0xff) as u8;
        carry = cur >> 8;
    }
    let mut full = Vec::new();
    while carry > 0 {
        full.push((carry & 0xff) as u8);
        carry >>= 8;
    }
    full.reverse();
    full.extend_from_slice(&out);
    Ok(crate::evm_tx::hex_quantity_to_dec(&format!("0x{}", hex::encode(full)))?)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn digest_stable_and_length_sensitive() {
        let d1 = multisig_tx_digest(
            "0x3535353535353535353535353535353535353535",
            "1000000000000000000",
            "",
            0,
            1,
        )
        .unwrap();
        let d2 = multisig_tx_digest(
            "0x3535353535353535353535353535353535353535",
            "1000000000000000000",
            "",
            0,
            1,
        )
        .unwrap();
        assert_eq!(d1, d2);
        // Any field change changes the digest.
        let d3 = multisig_tx_digest(
            "0x3535353535353535353535353535353535353535",
            "1000000000000000001",
            "",
            0,
            1,
        )
        .unwrap();
        assert_ne!(d1, d3);
        let d4 = multisig_tx_digest(
            "0x3535353535353535353535353535353535353535",
            "1000000000000000000",
            "",
            1,
            1,
        )
        .unwrap();
        assert_ne!(d1, d4);
    }

    #[test]
    fn owner_signature_verifies_via_ecrecover() {
        // Owner with privkey = 1 signs the digest; ecrecover must return their address.
        let mut key = [0u8; 32];
        key[31] = 1;
        let owner = crate::crypto::private_key_to_address(&key).unwrap().to_lowercase();
        let digest = multisig_tx_digest(
            "0x3535353535353535353535353535353535353535",
            "1000000000000000000",
            "0x",
            0,
            1,
        )
        .unwrap();
        let sk = k256::ecdsa::SigningKey::from_slice(&key).unwrap();
        let (sig, recid) = sk.sign_prehash_recoverable(&digest).unwrap();
        let mut sig65 = sig.to_bytes().to_vec();
        sig65.push(recid.is_y_odd() as u8);
        let recovered = evm_tx::ecrecover(&digest, &sig65).unwrap().to_lowercase();
        assert_eq!(recovered, owner);
    }

    #[test]
    fn multiply_dec_works() {
        assert_eq!(multiply_dec("1500000000", 2).unwrap(), "3000000000");
        assert_eq!(multiply_dec("0", 2).unwrap(), "0");
        assert_eq!(multiply_dec("255", 2).unwrap(), "510");
    }
}
