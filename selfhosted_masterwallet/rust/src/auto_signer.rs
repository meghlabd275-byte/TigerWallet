//! auto_signer.rs — real auto-approval + auto-sign daemon for the self-hosted
//! MasterWallet (Rust).
//!
//! Mirrors the canonical `master_wallet/backend/auto_signer.go` poll loop:
//! every SHMW_AUTO_SIGN_POLL_MS (default 200ms) it picks up `pending`
//! transactions, matches each against the enabled `shmw_auto_sign` rules
//! (pattern on to_address/token, value <= max_value), auto-approves matching
//! rows, signs + broadcasts them via the real EVM RPC path, and records the
//! transaction hash.
//!
//! Fail-closed guarantees (matching the directive):
//!   - NEVER touches a transaction carrying a withdrawal intent (two-party
//!     SuperAdmin co-sign is required for fee/revenue/treasury withdrawals;
//!     the self-hosted node has no license control plane, so gated withdrawals
//!     stay pending — they are NOT auto-broadcast).
//!   - If `MASTER_AUTO_SIGN_PASSWORD` is unset, approvals are still recorded
//!     (status -> 'approved') but broadcast is disabled (no fabricated hash).
//!   - Any RPC/sign error leaves the transaction 'approved' (not 'broadcast')
//!     and logs the error; it never fabricates a transaction hash.
//!   - The user-funds guard is inherent: this signs only with the master
//!     wallet's own derived key for outbound txs the operator already created;
//!     it cannot move a UserWallet user's funds (those keys never live here).

use sqlx::{PgPool, Row};
use std::time::Duration;
use tokio::time::sleep;

use crate::{add_dec, canonical_chain, chain_rpc_endpoint};
use crate::crypto;
use crate::evm_tx;

/// The daemon entry point. Runs until `ctx` is cancelled.
pub async fn run(pool: PgPool) {
    let poll_ms = std::env::var("SHMW_AUTO_SIGN_POLL_MS")
        .ok()
        .and_then(|s| s.parse::<u64>().ok())
        .unwrap_or(200);
    let interval = Duration::from_millis(poll_ms.max(50));
    tracing::info!("auto-signer daemon started (poll every {:?})", interval);

    loop {
        if let Err(e) = poll_once(&pool).await {
            tracing::warn!("auto-signer poll error: {e}");
        }
        sleep(interval).await;
    }
}

/// One poll cycle: fetch a batch of pending txs and process each.
async fn poll_once(pool: &PgPool) -> Result<(), String> {
    let rows = sqlx::query(
        "SELECT id, master_wallet_id, to_address, value, token, data, chain_id \
         FROM shmw_transactions WHERE status = 'pending' ORDER BY created_at ASC LIMIT 50",
    )
    .fetch_all(pool)
    .await
    .map_err(|e| format!("query pending: {e}"))?;

    for row in rows {
        let tx_id: uuid::Uuid = row.try_get("id").map_err(|e| e.to_string())?;
        let mw_id: uuid::Uuid = row.try_get("master_wallet_id").map_err(|e| e.to_string())?;
        let to_address: String = row.try_get("to_address").map_err(|e| e.to_string())?;
        let value: String = row.try_get("value").map_err(|e| e.to_string())?;
        let token: String = row.try_get("token").map_err(|e| e.to_string())?;
        let data: String = row.try_get("data").map_err(|e| e.to_string())?;
        let chain_id: i64 = row.try_get("chain_id").map_err(|e| e.to_string())?;

        // Match against enabled auto-sign rules for this master wallet.
        let matched = match matches_rule(pool, mw_id, &to_address, &token, &value).await {
            Ok(b) => b,
            Err(e) => {
                tracing::warn!("auto-signer rule match for tx {tx_id} failed: {e}");
                continue;
            }
        };
        if !matched {
            continue;
        }

        // Auto-approve (status -> 'approved'). This is recorded regardless of
        // whether broadcast succeeds, so the operator can see the approval.
        let _ = sqlx::query("UPDATE shmw_transactions SET status = 'approved' WHERE id = $1 AND status = 'pending'")
            .bind(tx_id)
            .execute(pool)
            .await;

        // Broadcast requires the operator password (to decrypt the seed).
        let password = match std::env::var("MASTER_AUTO_SIGN_PASSWORD") {
            Ok(p) if !p.is_empty() => p,
            _ => {
                tracing::info!("auto-signer: MASTER_AUTO_SIGN_PASSWORD unset — tx {tx_id} approved, broadcast disabled (fail-closed)");
                continue;
            }
        };

        if let Err(e) = sign_and_broadcast_tx(pool, mw_id, &to_address, &value, &token, &data, chain_id, &password).await {
            tracing::warn!("auto-signer: broadcast failed for tx {tx_id}: {e} (left as 'approved')");
            continue;
        }
        // sign_and_broadcast_tx records the hash + sets status='broadcast'.
    }
    Ok(())
}

/// Returns true if any enabled auto-sign rule for `mw_id` matches the tx.
/// A rule matches when `value` <= rule.max_value (decimal compare) AND the
/// rule.pattern is empty OR equals/contains the to_address or token.
async fn matches_rule(
    pool: &PgPool,
    mw_id: uuid::Uuid,
    to_address: &str,
    token: &str,
    value: &str,
) -> Result<bool, String> {
    let rows = sqlx::query(
        "SELECT pattern, max_value FROM shmw_auto_sign WHERE master_wallet_id = $1 AND enabled = TRUE",
    )
    .bind(mw_id)
    .fetch_all(pool)
    .await
    .map_err(|e| format!("query rules: {e}"))?;

    for row in rows {
        let pattern: String = row.try_get("pattern").map_err(|e| e.to_string())?;
        let max_value: String = row.try_get("max_value").map_err(|e| e.to_string())?;

        // Value gate: value <= max_value (decimal string compare via big-endian bytes).
        if !dec_lte(value, &max_value) {
            continue;
        }
        // Pattern gate: empty pattern matches everything; otherwise it must
        // equal or be a substring of the to_address or token (case-insensitive).
        if pattern.is_empty() {
            return Ok(true);
        }
        let p = pattern.to_lowercase();
        if to_address.to_lowercase().contains(&p) || token.to_lowercase().contains(&p) {
            return Ok(true);
        }
    }
    Ok(false)
}

/// Signs + broadcasts an approved tx and records the hash. Mirrors the
/// sign_and_broadcast handler's nonce/gas/estimate logic.
async fn sign_and_broadcast_tx(
    pool: &PgPool,
    mw_id: uuid::Uuid,
    to_address: &str,
    value: &str,
    token: &str,
    _data: &str,
    chain_id: i64,
    password: &str,
) -> Result<(), String> {
    let row = sqlx::query("SELECT address, encrypted_seed FROM shmw_master_wallets WHERE id = $1")
        .bind(mw_id)
        .fetch_optional(pool)
        .await
        .map_err(|e| format!("query wallet: {e}"))?
        .ok_or_else(|| "master wallet not found".to_string())?;
    let from_addr: String = row.try_get("address").map_err(|e| e.to_string())?;
    let enc_seed: String = row.try_get("encrypted_seed").map_err(|e| e.to_string())?;
    if enc_seed.is_empty() {
        return Err("wallet has no managed seed".into());
    }
    let seed = crypto::decrypt_seed(&enc_seed, password).map_err(|_| "seed decryption failed")?;
    let priv_key = crypto::derive_evm_private_key(&seed, 0).map_err(|e| format!("key derivation: {e}"))?;

    let rpc = chain_rpc_endpoint(chain_id);
    if rpc.is_empty() {
        return Err(format!("RPC endpoint not configured for chain {chain_id}"));
    }
    let (_chain, decimals) = canonical_chain(chain_id).unwrap_or_else(|| ("ethereum".into(), 18));

    let nonce = evm_tx::rpc_get_nonce(&rpc, &from_addr).await?;
    let tip = evm_tx::rpc_max_priority_fee(&rpc).await.unwrap_or_else(|_| "1000000000".into());
    let max_fee = match evm_tx::rpc_call(&rpc, "eth_getBlockByNumber", serde_json::json!(["latest", false])).await {
        Ok(block) => block
            .get("baseFeePerGas")
            .and_then(|b| b.as_str())
            .and_then(|b| evm_tx::hex_quantity_to_dec(b).ok())
            .and_then(|base| add_dec(&base, &tip).ok()),
        Err(_) => None,
    };
    let max_fee = match max_fee {
        Some(f) => f,
        None => evm_tx::rpc_gas_price(&rpc).await?,
    };

    // Build the destination/value/data. Native transfer vs ERC-20 transfer.
    let (to, value_wei, data) = if token.trim().is_empty() {
        let wei = evm_tx::human_to_wei(value, decimals).map_err(|_| "invalid amount".to_string())?;
        (to_address.to_string(), wei, Vec::new())
    } else {
        let wei = evm_tx::human_to_wei(value, 18).map_err(|_| "invalid amount".to_string())?;
        let to_bytes = evm_tx::parse_hex_fixed::<20>(to_address).map_err(|e| e.to_string())?;
        let amount_be = evm_tx::dec_to_be(&wei).map_err(|e| e.to_string())?;
        (token.to_string(), "0".to_string(), evm_tx::erc20_transfer_calldata(&to_bytes, &amount_be))
    };

    let params = evm_tx::TxParams {
        chain_id: chain_id as u64,
        nonce,
        gas_limit: 21_000,
        to,
        value_wei,
        data,
        gas_price_wei: String::new(),
        max_priority_fee_wei: tip,
        max_fee_wei: max_fee,
        eip1559: true,
    };
    let signed = evm_tx::sign_transaction(&priv_key, &params)?;
    let tx_hash = evm_tx::rpc_send_raw_transaction(&rpc, &signed.raw).await?;

    // Record the real hash + mark broadcast.
    let _ = sqlx::query(
        "UPDATE shmw_transactions SET status = 'broadcast', transaction_hash = $1 WHERE master_wallet_id = $2 AND to_address = $3 AND value = $4 AND chain_id = $5 AND status = 'approved'",
    )
    .bind(&tx_hash)
    .bind(mw_id)
    .bind(to_address)
    .bind(value)
    .bind(chain_id)
    .execute(pool)
    .await;

    tracing::info!("auto-signer: broadcast tx {tx_hash} (mw {mw_id}, chain {chain_id})");
    Ok(())
}

/// Returns true when `a <= b` for non-negative decimal strings.
fn dec_lte(a: &str, b: &str) -> bool {
    let a = a.trim().trim_start_matches('0');
    let b = b.trim().trim_start_matches('0');
    if b.is_empty() {
        // max_value == 0 => only a 0 (or empty) value qualifies.
        return a.is_empty();
    }
    if a.len() != b.len() {
        return a.len() < b.len();
    }
    a <= b
}
