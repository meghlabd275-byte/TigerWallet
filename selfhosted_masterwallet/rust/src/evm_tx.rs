//! evm_tx.rs — real EVM transaction construction + secp256k1 signing.
//!
//! Port of master_wallet/backend/handlers.go SignTransaction internals:
//!   * Legacy EIP-155 and EIP-1559 (type-2) signing via RLP + keccak256
//!   * EIP-191 personal message signing
//!   * ERC-20 transfer(address,uint256) calldata
//!   * human → wei conversion and a minimal JSON-RPC client

use k256::ecdsa::{RecoveryId, Signature, SigningKey, VerifyingKey};
use sha3::{Digest, Keccak256};

use crate::crypto;
use crate::rlp;

/// Parameters for an EVM transaction to sign.
#[derive(Clone, Debug)]
pub struct TxParams {
    pub chain_id: u64,
    pub nonce: u64,
    pub gas_limit: u64,
    /// Recipient (0x-prefixed 20-byte hex). Empty for contract creation.
    pub to: String,
    /// Value in wei, decimal string.
    pub value_wei: String,
    /// Raw calldata.
    pub data: Vec<u8>,
    /// Legacy gas price in wei (decimal string). Used when eip1559 == false.
    pub gas_price_wei: String,
    /// EIP-1559 fee fields (decimal wei strings). Used when eip1559 == true.
    pub max_priority_fee_wei: String,
    pub max_fee_wei: String,
    pub eip1559: bool,
}

/// A signed transaction: raw bytes (ready for eth_sendRawTransaction) and hash.
pub struct SignedTx {
    pub raw: Vec<u8>,
    pub hash: [u8; 32],
    pub v: u64,
    pub r: [u8; 32],
    pub s: [u8; 32],
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Parse a decimal string into a minimal big-endian 256-bit byte string.
/// Only digits are allowed; returns empty vec for "0".
pub fn dec_to_be(s: &str) -> Result<Vec<u8>, String> {
    let s = s.trim();
    if s.is_empty() || !s.bytes().all(|b| b.is_ascii_digit()) {
        return Err(format!("invalid decimal integer: {s}"));
    }
    // Long-division base 10^19 → 256-bit BE, implemented over a 4x u64 limb buf.
    let mut limbs = [0u64; 4]; // little-endian limbs
    for chunk in s.as_bytes().chunks(19) {
        let chunk_str = std::str::from_utf8(chunk).unwrap();
        let chunk_val: u64 = chunk_str
            .parse()
            .map_err(|_| format!("invalid decimal integer: {s}"))?;
        // limbs = limbs * 10^len(chunk) + chunk_val
        let mul = 10u64.pow(chunk.len() as u32);
        let mut carry: u128 = chunk_val as u128;
        for limb in limbs.iter_mut() {
            let cur = (*limb as u128) * (mul as u128) + carry;
            *limb = cur as u64;
            carry = cur >> 64;
        }
        if carry != 0 {
            return Err("value exceeds 256 bits".into());
        }
    }
    let mut be = [0u8; 32];
    for (i, limb) in limbs.iter().enumerate() {
        be[24 - i * 8..32 - i * 8].copy_from_slice(&limb.to_be_bytes());
    }
    Ok(rlp::trim_be(&be).to_vec())
}

/// humanToWei — human decimal string ("1.5") → base units ("1500000000000000000").
/// Port of the canonical helper (no floats — exact decimal shifting).
pub fn human_to_wei(amount: &str, decimals: u32) -> Result<String, String> {
    let amount = amount.trim();
    if amount.is_empty() {
        return Err("empty amount".into());
    }
    let (int_part, frac_part) = match amount.split_once('.') {
        Some((i, f)) => (i, f),
        None => (amount, ""),
    };
    if !int_part.bytes().all(|b| b.is_ascii_digit())
        || !frac_part.bytes().all(|b| b.is_ascii_digit())
    {
        return Err(format!("invalid amount: {amount}"));
    }
    if frac_part.len() > decimals as usize {
        return Err(format!("too many decimal places (max {decimals})"));
    }
    let mut digits = int_part.trim_start_matches('0').to_string();
    if digits.is_empty() {
        digits.push('0');
    }
    digits.push_str(frac_part);
    digits.extend(std::iter::repeat('0').take(decimals as usize - frac_part.len()));
    let trimmed = digits.trim_start_matches('0');
    Ok(if trimmed.is_empty() { "0".into() } else { trimmed.into() })
}

/// ERC-20 transfer(address,uint256) calldata.
pub fn erc20_transfer_calldata(to: &[u8; 20], amount_be: &[u8]) -> Vec<u8> {
    let mut data = vec![0xa9, 0x05, 0x9c, 0xbb];
    data.extend_from_slice(&[0u8; 12]);
    data.extend_from_slice(to);
    data.extend_from_slice(&[0u8; 32]);
    let start = data.len() - amount_be.len().min(32);
    data[start..].copy_from_slice(&amount_be[amount_be.len().saturating_sub(32)..]);
    data
}

/// Parse "0x..." or bare hex into exactly N bytes.
pub fn parse_hex_fixed<const N: usize>(s: &str) -> Result<[u8; N], String> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    let bytes = hex::decode(s).map_err(|e| e.to_string())?;
    if bytes.len() != N {
        return Err(format!("expected {N} bytes, got {}", bytes.len()));
    }
    let mut out = [0u8; N];
    out.copy_from_slice(&bytes);
    Ok(out)
}

// ---------------------------------------------------------------------------
// Signing
// ---------------------------------------------------------------------------

fn sign_recoverable(priv_key: &[u8; 32], sighash: &[u8; 32]) -> Result<(Signature, RecoveryId), String> {
    let sk = SigningKey::from_slice(priv_key).map_err(|e| e.to_string())?;
    sk.sign_prehash_recoverable(sighash).map_err(|e| e.to_string())
}

/// Sign a transaction (legacy EIP-155 or EIP-1559 type-2) and return raw bytes.
pub fn sign_transaction(priv_key: &[u8; 32], p: &TxParams) -> Result<SignedTx, String> {
    let to_bytes = if p.to.is_empty() {
        Vec::new()
    } else {
        parse_hex_fixed::<20>(&p.to)?.to_vec()
    };
    let value = dec_to_be(&p.value_wei)?;
    let chain_be = dec_to_be(&p.chain_id.to_string())?;

    let mut payload: Vec<u8> = Vec::new();
    let sighash: [u8; 32];
    let signed_prefix: Option<u8>;
    if p.eip1559 {
        let tip = dec_to_be(&p.max_priority_fee_wei)?;
        let cap = dec_to_be(&p.max_fee_wei)?;
        let mut body = Vec::new();
        rlp::encode_uint_be(&chain_be, &mut body);
        rlp::encode_u64(p.nonce, &mut body);
        rlp::encode_uint_be(&tip, &mut body);
        rlp::encode_uint_be(&cap, &mut body);
        rlp::encode_u64(p.gas_limit, &mut body);
        rlp::encode_bytes(&to_bytes, &mut body);
        rlp::encode_uint_be(&value, &mut body);
        rlp::encode_bytes(&p.data, &mut body);
        rlp::encode_list(&[], &mut body); // empty access list
        rlp::encode_list(&body, &mut payload);
        let mut preimage = vec![0x02u8];
        preimage.extend_from_slice(&payload);
        let h = Keccak256::digest(&preimage);
        sighash = h.into();
        signed_prefix = Some(0x02);
    } else {
        let gas_price = dec_to_be(&p.gas_price_wei)?;
        let mut body = Vec::new();
        rlp::encode_u64(p.nonce, &mut body);
        rlp::encode_uint_be(&gas_price, &mut body);
        rlp::encode_u64(p.gas_limit, &mut body);
        rlp::encode_bytes(&to_bytes, &mut body);
        rlp::encode_uint_be(&value, &mut body);
        rlp::encode_bytes(&p.data, &mut body);
        rlp::encode_uint_be(&chain_be, &mut body);
        rlp::encode_u64(0, &mut body);
        rlp::encode_u64(0, &mut body);
        rlp::encode_list(&body, &mut payload);
        let h = Keccak256::digest(&payload);
        sighash = h.into();
        signed_prefix = None;
    }

    let (sig, recid) = sign_recoverable(priv_key, &sighash)?;
    let r_bytes: [u8; 32] = sig.r().to_bytes().into();
    let s_bytes: [u8; 32] = sig.s().to_bytes().into();
    let y_parity = recid.is_y_odd() as u64;

    let v: u64 = if p.eip1559 {
        y_parity
    } else {
        p.chain_id * 2 + 35 + y_parity
    };

    // Rebuild the payload with the signature appended.
    if p.eip1559 {
        // Strip the outer list encoding, append v/r/s, re-encode.
        let mut body = Vec::new();
        rlp::encode_uint_be(&chain_be, &mut body);
        rlp::encode_u64(p.nonce, &mut body);
        rlp::encode_uint_be(&dec_to_be(&p.max_priority_fee_wei)?, &mut body);
        rlp::encode_uint_be(&dec_to_be(&p.max_fee_wei)?, &mut body);
        rlp::encode_u64(p.gas_limit, &mut body);
        rlp::encode_bytes(&to_bytes, &mut body);
        rlp::encode_uint_be(&value, &mut body);
        rlp::encode_bytes(&p.data, &mut body);
        rlp::encode_list(&[], &mut body);
        rlp::encode_u64(v, &mut body);
        rlp::encode_bytes(&r_bytes, &mut body);
        rlp::encode_bytes(&s_bytes, &mut body);
        let mut raw = vec![signed_prefix.unwrap()];
        rlp::encode_list(&body, &mut raw);
        let h = Keccak256::digest(&raw);
        Ok(SignedTx { hash: h.into(), raw, v, r: r_bytes, s: s_bytes })
    } else {
        let mut body = Vec::new();
        rlp::encode_u64(p.nonce, &mut body);
        rlp::encode_uint_be(&dec_to_be(&p.gas_price_wei)?, &mut body);
        rlp::encode_u64(p.gas_limit, &mut body);
        rlp::encode_bytes(&to_bytes, &mut body);
        rlp::encode_uint_be(&value, &mut body);
        rlp::encode_bytes(&p.data, &mut body);
        rlp::encode_uint_be(&dec_to_be(&v.to_string())?, &mut body);
        rlp::encode_bytes(&r_bytes, &mut body);
        rlp::encode_bytes(&s_bytes, &mut body);
        let mut raw = Vec::new();
        rlp::encode_list(&body, &mut raw);
        let h = Keccak256::digest(&raw);
        Ok(SignedTx { hash: h.into(), raw, v, r: r_bytes, s: s_bytes })
    }
}

/// EIP-191 personal_sign: keccak256("\x19Ethereum Signed Message:\n" + len + msg).
pub fn personal_sign(priv_key: &[u8; 32], message: &[u8]) -> Result<Vec<u8>, String> {
    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let mut preimage = prefix.into_bytes();
    preimage.extend_from_slice(message);
    let hash: [u8; 32] = Keccak256::digest(&preimage).into();
    let (sig, recid) = sign_recoverable(priv_key, &hash)?;
    let mut out = sig.to_bytes().to_vec();
    out.push(recid.is_y_odd() as u8);
    Ok(out)
}

/// Ecrecover — recover the EIP-55 address of the signer of a 32-byte digest.
/// `sig` is 65 bytes r||s||v where v ∈ {0,1,27,28} or EIP-155 values.
pub fn ecrecover(digest: &[u8; 32], sig: &[u8]) -> Result<String, String> {
    if sig.len() != 65 {
        return Err("signature must be 65 bytes".into());
    }
    let mut v = sig[64] as u64;
    if v >= 35 {
        v = (v - 35) % 2; // EIP-155
    } else if v >= 27 {
        v -= 27;
    }
    if v > 1 {
        return Err("invalid recovery id".into());
    }
    let signature = Signature::from_slice(&sig[..64]).map_err(|e| e.to_string())?;
    let recid = RecoveryId::new(v == 1, false);
    let vk = VerifyingKey::recover_from_prehash(digest, &signature, recid)
        .map_err(|e| e.to_string())?;
    Ok(crypto::public_key_to_address(&vk.into()))
}

// ---------------------------------------------------------------------------
// Minimal JSON-RPC client (real RPC only — fail-closed)
// ---------------------------------------------------------------------------

/// JSON-RPC call; returns the `result` field or an error. Never fabricates.
pub async fn rpc_call(rpc_url: &str, method: &str, params: serde_json::Value) -> Result<serde_json::Value, String> {
    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(15))
        .build()
        .map_err(|e| e.to_string())?;
    let body = serde_json::json!({
        "jsonrpc": "2.0", "id": 1, "method": method, "params": params,
    });
    let resp: serde_json::Value = client
        .post(rpc_url)
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("rpc request failed: {e}"))?
        .json()
        .await
        .map_err(|e| format!("rpc response decode failed: {e}"))?;
    if let Some(err) = resp.get("error") {
        return Err(format!("rpc error: {err}"));
    }
    resp.get("result")
        .cloned()
        .ok_or_else(|| "rpc response missing result".to_string())
}

/// eth_getTransactionCount (pending) → nonce.
pub async fn rpc_get_nonce(rpc_url: &str, address: &str) -> Result<u64, String> {
    let r = rpc_call(rpc_url, "eth_getTransactionCount", serde_json::json!([address, "pending"])).await?;
    let s = r.as_str().ok_or("nonce result not a string")?;
    u64::from_str_radix(s.trim_start_matches("0x"), 16).map_err(|e| e.to_string())
}

/// eth_gasPrice → wei decimal string.
pub async fn rpc_gas_price(rpc_url: &str) -> Result<String, String> {
    let r = rpc_call(rpc_url, "eth_gasPrice", serde_json::json!([])).await?;
    let s = r.as_str().ok_or("gasPrice result not a string")?;
    hex_quantity_to_dec(s)
}

/// eth_maxPriorityFeePerGas → wei decimal string.
pub async fn rpc_max_priority_fee(rpc_url: &str) -> Result<String, String> {
    let r = rpc_call(rpc_url, "eth_maxPriorityFeePerGas", serde_json::json!([])).await?;
    let s = r.as_str().ok_or("maxPriorityFeePerGas result not a string")?;
    hex_quantity_to_dec(s)
}

/// eth_sendRawTransaction → tx hash.
pub async fn rpc_send_raw_transaction(rpc_url: &str, raw: &[u8]) -> Result<String, String> {
    let r = rpc_call(
        rpc_url,
        "eth_sendRawTransaction",
        serde_json::json!([format!("0x{}", hex::encode(raw))]),
    )
    .await?;
    r.as_str().map(|s| s.to_string()).ok_or("tx hash result not a string".into())
}

/// eth_getBalance → wei decimal string.
pub async fn rpc_get_balance(rpc_url: &str, address: &str) -> Result<String, String> {
    let r = rpc_call(rpc_url, "eth_getBalance", serde_json::json!([address, "latest"])).await?;
    let s = r.as_str().ok_or("balance result not a string")?;
    hex_quantity_to_dec(s)
}

/// Convert a hex QUANTITY string ("0x…") to a decimal string.
pub fn hex_quantity_to_dec(s: &str) -> Result<String, String> {
    let h = s.trim().trim_start_matches("0x");
    let h = h.trim_start_matches('0');
    if h.is_empty() {
        return Ok("0".into());
    }
    // base-16 → base-10 via repeated multiply-add over decimal digits
    let mut digits = vec![0u8]; // little-endian decimal digits
    for c in h.bytes() {
        let v = (c as char).to_digit(16).ok_or("invalid hex")? as u8;
        let mut carry = v as u32;
        for d in digits.iter_mut() {
            let cur = (*d as u32) * 16 + carry;
            *d = (cur % 10) as u8;
            carry = cur / 10;
        }
        while carry > 0 {
            digits.push((carry % 10) as u8);
            carry /= 10;
        }
    }
    Ok(digits.iter().rev().map(|d| (b'0' + d) as char).collect())
}

/// weiToFloat-style: wei decimal string → human units (string, exact decimal shift).
pub fn wei_to_human(wei: &str, decimals: u32) -> String {
    let wei = wei.trim_start_matches('0');
    let wei = if wei.is_empty() { "0" } else { wei };
    let d = decimals as usize;
    if wei.len() <= d {
        let zeros = d + 1 - wei.len();
        let mut s = String::from("0.");
        s.extend(std::iter::repeat('0').take(zeros - 1));
        s.push_str(wei);
        let s = s.trim_end_matches('0');
        return s.trim_end_matches('.').to_string();
    }
    let (int_part, frac) = wei.split_at(wei.len() - d);
    let frac = frac.trim_end_matches('0');
    if frac.is_empty() {
        int_part.to_string()
    } else {
        format!("{int_part}.{frac}")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn privkey_one() -> [u8; 32] {
        let mut k = [0u8; 32];
        k[31] = 1;
        k
    }

    #[test]
    fn human_to_wei_exact() {
        assert_eq!(human_to_wei("1.5", 18).unwrap(), "1500000000000000000");
        assert_eq!(human_to_wei("0", 18).unwrap(), "0");
        assert_eq!(human_to_wei("0.000000000000000001", 18).unwrap(), "1");
        assert_eq!(human_to_wei("12", 6).unwrap(), "12000000");
        assert!(human_to_wei("1.1234567", 6).is_err());
        assert!(human_to_wei("abc", 18).is_err());
    }

    #[test]
    fn dec_be_roundtrip() {
        assert_eq!(dec_to_be("0").unwrap(), Vec::<u8>::new());
        assert_eq!(dec_to_be("1").unwrap(), vec![1]);
        assert_eq!(dec_to_be("256").unwrap(), vec![1, 0]);
        // 2^255 fits, 2^256 does not
        assert!(dec_to_be("57896044618658097711785492504343953926634992332820282019728792003956564819968").is_ok());
        assert!(dec_to_be("115792089237316195423570985008687907853269984665640564039457584007913129639936").is_err());
    }

    #[test]
    fn legacy_sign_recovers_signer() {
        let key = privkey_one();
        let expected_from = crypto::private_key_to_address(&key).unwrap();
        let tx = TxParams {
            chain_id: 1,
            nonce: 0,
            gas_limit: 21000,
            to: "0x3535353535353535353535353535353535353535".into(),
            value_wei: "1000000000000000000".into(),
            data: vec![],
            gas_price_wei: "20000000000".into(),
            max_priority_fee_wei: String::new(),
            max_fee_wei: String::new(),
            eip1559: false,
        };
        let signed = sign_transaction(&key, &tx).unwrap();
        assert!(signed.v == 37 || signed.v == 38, "EIP-155 v must be 37/38, got {}", signed.v);
        assert_ne!(signed.r, [0u8; 32]);
        assert_ne!(signed.s, [0u8; 32]);
        assert!(signed.raw.len() > 100);

        // Recompute the sighash and recover the signer.
        let mut body = Vec::new();
        rlp::encode_u64(0, &mut body);
        rlp::encode_uint_be(&dec_to_be("20000000000").unwrap(), &mut body);
        rlp::encode_u64(21000, &mut body);
        rlp::encode_bytes(&parse_hex_fixed::<20>("0x3535353535353535353535353535353535353535").unwrap(), &mut body);
        rlp::encode_uint_be(&dec_to_be("1000000000000000000").unwrap(), &mut body);
        rlp::encode_bytes(&[], &mut body);
        rlp::encode_uint_be(&[1], &mut body);
        rlp::encode_u64(0, &mut body);
        rlp::encode_u64(0, &mut body);
        let mut payload = Vec::new();
        rlp::encode_list(&body, &mut payload);
        let sighash: [u8; 32] = Keccak256::digest(&payload).into();
        let mut sig65 = Vec::with_capacity(65);
        sig65.extend_from_slice(&signed.r);
        sig65.extend_from_slice(&signed.s);
        sig65.push(signed.v as u8);
        let recovered = ecrecover(&sighash, &sig65).unwrap();
        assert_eq!(recovered, expected_from);
    }

    #[test]
    fn eip1559_sign_recovers_signer() {
        let key = privkey_one();
        let expected_from = crypto::private_key_to_address(&key).unwrap();
        let tx = TxParams {
            chain_id: 1,
            nonce: 9,
            gas_limit: 21000,
            to: "0x3535353535353535353535353535353535353535".into(),
            value_wei: "1000000000000000000".into(),
            data: vec![],
            gas_price_wei: String::new(),
            max_priority_fee_wei: "1500000000".into(),
            max_fee_wei: "30000000000".into(),
            eip1559: true,
        };
        let signed = sign_transaction(&key, &tx).unwrap();
        assert_eq!(signed.raw[0], 0x02);
        assert!(signed.v <= 1, "type-2 v must be y-parity 0/1, got {}", signed.v);

        // Recompute sighash, recover signer.
        let mut body = Vec::new();
        rlp::encode_uint_be(&[1], &mut body);
        rlp::encode_u64(9, &mut body);
        rlp::encode_uint_be(&dec_to_be("1500000000").unwrap(), &mut body);
        rlp::encode_uint_be(&dec_to_be("30000000000").unwrap(), &mut body);
        rlp::encode_u64(21000, &mut body);
        rlp::encode_bytes(&parse_hex_fixed::<20>("0x3535353535353535353535353535353535353535").unwrap(), &mut body);
        rlp::encode_uint_be(&dec_to_be("1000000000000000000").unwrap(), &mut body);
        rlp::encode_bytes(&[], &mut body);
        rlp::encode_list(&[], &mut body);
        let mut inner = Vec::new();
        rlp::encode_list(&body, &mut inner);
        let mut preimage = vec![0x02u8];
        preimage.extend_from_slice(&inner);
        let sighash: [u8; 32] = Keccak256::digest(&preimage).into();
        let mut sig65 = Vec::with_capacity(65);
        sig65.extend_from_slice(&signed.r);
        sig65.extend_from_slice(&signed.s);
        sig65.push(signed.v as u8);
        let recovered = ecrecover(&sighash, &sig65).unwrap();
        assert_eq!(recovered, expected_from);
    }

    #[test]
    fn personal_sign_known_vector() {
        // privkey = 1 → address 0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf
        let key = privkey_one();
        assert_eq!(
            crypto::private_key_to_address(&key).unwrap(),
            "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"
        );
        let sig = personal_sign(&key, b"hello").unwrap();
        assert_eq!(sig.len(), 65);
        // Recover from the EIP-191 digest.
        let prefix = b"\x19Ethereum Signed Message:\n5";
        let mut preimage = prefix.to_vec();
        preimage.extend_from_slice(b"hello");
        let digest: [u8; 32] = Keccak256::digest(&preimage).into();
        let recovered = ecrecover(&digest, &sig).unwrap();
        assert_eq!(recovered, "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf");
    }

    #[test]
    fn erc20_calldata_layout() {
        let to = parse_hex_fixed::<20>("0x3535353535353535353535353535353535353535").unwrap();
        let data = erc20_transfer_calldata(&to, &dec_to_be("1000").unwrap());
        assert_eq!(data.len(), 68);
        assert_eq!(&data[..4], &[0xa9, 0x05, 0x9c, 0xbb]);
        assert_eq!(&data[16..36], &to);
        assert_eq!(&data[66..68], &[0x03, 0xe8]);
    }

    #[test]
    fn hex_quantity_conversion() {
        assert_eq!(hex_quantity_to_dec("0x0").unwrap(), "0");
        assert_eq!(hex_quantity_to_dec("0xde0b6b3a7640000").unwrap(), "1000000000000000000");
        assert_eq!(hex_quantity_to_dec("0xff").unwrap(), "255");
    }
}
