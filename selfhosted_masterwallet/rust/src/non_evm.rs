//! non_evm.rs — real non-EVM address generation + signing.
//!
//! Port of master_wallet/backend/non_evm_crypto.go:
//!   * Solana: SLIP-0010 Ed25519 derivation (all-hardened), base58 address,
//!     Ed25519 message signing
//!   * Bitcoin: BIP-32/44 secp256k1 → P2PKH (RIPEMD160 + base58check),
//!     Bitcoin signed-message format (compact recoverable, base64)
//!   * Cosmos: secp256k1 → bech32 (BIP-173) address, amino JSON signing

use ed25519_dalek::{Signer, SigningKey};
use hmac::{Hmac, Mac};
use k256::elliptic_curve::sec1::ToEncodedPoint;
use k256::SecretKey;
use ripemd::Ripemd160;
use sha2::{Digest, Sha256, Sha512};

use crate::crypto;

pub const BASE58_ALPHABET: &[u8; 58] =
    b"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
const BECH32_CHARSET: &[u8; 32] = b"qpzry9x8gf2tvdw0s3jn54khce6mua7l";

/// Parse "m/44'/501'/0'/0'" style paths into hardened-flagged indices.
pub fn parse_path(path: &str) -> Result<Vec<u32>, String> {
    let path = path.trim();
    if path.is_empty() {
        return Err("empty derivation path".into());
    }
    let segments: Vec<&str> = path.split('/').collect();
    let start = if segments[0] == "m" || segments[0] == "M" { 1 } else { 0 };
    let mut out = Vec::new();
    for seg in &segments[start..] {
        if seg.is_empty() {
            return Err(format!("invalid derivation path segment in {path}"));
        }
        let hardened = seg.ends_with('\'') || seg.ends_with('h') || seg.ends_with('H');
        let num_str = if hardened { &seg[..seg.len() - 1] } else { *seg };
        let mut index: u32 = num_str
            .parse()
            .map_err(|_| format!("invalid derivation index: {seg}"))?;
        if index >= 0x8000_0000 {
            return Err(format!("derivation index out of range: {seg}"));
        }
        if hardened {
            index += 0x8000_0000;
        }
        out.push(index);
    }
    if out.is_empty() {
        return Err("invalid derivation path".into());
    }
    Ok(out)
}

// ---------------------------------------------------------------------------
// SLIP-0010 Ed25519 (Solana)
// ---------------------------------------------------------------------------

fn slip10_master(seed: &[u8]) -> ([u8; 32], [u8; 32]) {
    let mut mac = <Hmac<Sha512> as Mac>::new_from_slice(b"ed25519 seed").unwrap();
    mac.update(seed);
    let i = mac.finalize().into_bytes();
    let mut k = [0u8; 32];
    let mut c = [0u8; 32];
    k.copy_from_slice(&i[..32]);
    c.copy_from_slice(&i[32..]);
    (k, c)
}

/// SLIP-10 derivation for Ed25519 — every segment MUST be hardened.
pub fn slip10_derive(seed: &[u8], path: &str) -> Result<[u8; 32], String> {
    let indices = parse_path(path)?;
    let (mut key, mut chain) = slip10_master(seed);
    for index in indices {
        if index & 0x8000_0000 == 0 {
            return Err("ed25519 SLIP-10 requires hardened-only derivation".into());
        }
        let mut data = Vec::with_capacity(37);
        data.push(0u8);
        data.extend_from_slice(&key);
        data.extend_from_slice(&index.to_be_bytes());
        let mut mac = <Hmac<Sha512> as Mac>::new_from_slice(&chain).unwrap();
        mac.update(&data);
        let i = mac.finalize().into_bytes();
        key.copy_from_slice(&i[..32]);
        chain.copy_from_slice(&i[32..]);
    }
    Ok(key)
}

/// SolanaAddressFromSeed — base58(ed25519 public key).
pub fn solana_address_from_seed(seed: &[u8], path: &str) -> Result<String, String> {
    let key = slip10_derive(seed, path)?;
    let sk = SigningKey::from_bytes(&key);
    Ok(base58_encode(sk.verifying_key().as_bytes()))
}

/// SolanaSign — real Ed25519 signature (64 bytes).
pub fn solana_sign(seed: &[u8], path: &str, message: &[u8]) -> Result<Vec<u8>, String> {
    let key = slip10_derive(seed, path)?;
    let sk = SigningKey::from_bytes(&key);
    Ok(sk.sign(message).to_bytes().to_vec())
}

// ---------------------------------------------------------------------------
// Base58 / base58check (Bitcoin)
// ---------------------------------------------------------------------------

/// Base58 encoding (Bitcoin alphabet).
pub fn base58_encode(data: &[u8]) -> String {
    let zeros = data.iter().take_while(|&&b| b == 0).count();
    // big-number base conversion
    let mut digits: Vec<u8> = Vec::new(); // little-endian base-58 digits
    for &b in &data[zeros..] {
        let mut carry = b as u32;
        for d in digits.iter_mut() {
            let cur = (*d as u32) * 256 + carry;
            *d = (cur % 58) as u8;
            carry = cur / 58;
        }
        while carry > 0 {
            digits.push((carry % 58) as u8);
            carry /= 58;
        }
    }
    let mut out: String = std::iter::repeat('1').take(zeros).collect();
    for d in digits.iter().rev() {
        out.push(BASE58_ALPHABET[*d as usize] as char);
    }
    out
}

/// Base58 decoding (Bitcoin alphabet).
pub fn base58_decode(s: &str) -> Result<Vec<u8>, String> {
    let zeros = s.bytes().take_while(|&b| b == b'1').count();
    let mut bytes: Vec<u8> = Vec::new(); // little-endian base-256
    for c in s[zeros..].bytes() {
        let v = BASE58_ALPHABET
            .iter()
            .position(|&x| x == c)
            .ok_or("invalid base58 character")? as u32;
        let mut carry = v;
        for b in bytes.iter_mut() {
            let cur = (*b as u32) * 58 + carry;
            *b = (cur % 256) as u8;
            carry = cur / 256;
        }
        while carry > 0 {
            bytes.push((carry % 256) as u8);
            carry /= 256;
        }
    }
    let be: Vec<u8> = bytes.iter().rev().cloned().collect();
    let first = be.iter().position(|&b| b != 0).unwrap_or(be.len());
    let mut out = vec![0u8; zeros];
    out.extend_from_slice(&be[first..]);
    Ok(out)
}

/// Base58CheckEncode — payload + 4-byte double-SHA256 checksum → base58.
pub fn base58check_encode(version: u8, payload: &[u8]) -> String {
    let mut data = vec![version];
    data.extend_from_slice(payload);
    let checksum = sha256d(&data);
    data.extend_from_slice(&checksum[..4]);
    base58_encode(&data)
}

/// Base58CheckDecode — verify checksum, return (version, payload).
pub fn base58check_decode(s: &str) -> Result<(u8, Vec<u8>), String> {
    let data = base58_decode(s)?;
    if data.len() < 5 {
        return Err("base58check payload too short".into());
    }
    let (body, checksum) = data.split_at(data.len() - 4);
    if sha256d(body)[..4] != *checksum {
        return Err("base58check checksum mismatch".into());
    }
    Ok((body[0], body[1..].to_vec()))
}

pub fn sha256d(data: &[u8]) -> [u8; 32] {
    let first = Sha256::digest(data);
    Sha256::digest(first).into()
}

/// HASH160 — RIPEMD160(SHA256(data)).
pub fn hash160(data: &[u8]) -> [u8; 20] {
    let sha = Sha256::digest(data);
    Ripemd160::digest(sha).into()
}

/// BitcoinAddressFromSeed — BIP-32/44 → compressed pubkey → P2PKH (0x00 mainnet).
pub fn btc_address_from_seed(seed: &[u8], path: &str) -> Result<String, String> {
    let key = crypto::derive_private_key_from_path(seed, path)?;
    Ok(btc_address_from_privkey(&key))
}

/// P2PKH address for a raw secp256k1 private key (mainnet, version 0x00).
pub fn btc_address_from_privkey(priv_key: &[u8; 32]) -> String {
    let sk = SecretKey::from_slice(priv_key).expect("valid secp256k1 key");
    let compressed = sk.public_key().to_encoded_point(true);
    let h160 = hash160(compressed.as_bytes());
    base58check_encode(0x00, &h160)
}

/// Bitcoin compactSize varint.
fn varint(n: usize) -> Vec<u8> {
    if n < 253 {
        vec![n as u8]
    } else if n <= 0xffff {
        let mut v = vec![0xfd];
        v.extend_from_slice(&(n as u16).to_le_bytes());
        v
    } else if n <= 0xffff_ffff {
        let mut v = vec![0xfe];
        v.extend_from_slice(&(n as u32).to_le_bytes());
        v
    } else {
        let mut v = vec![0xff];
        v.extend_from_slice(&(n as u64).to_le_bytes());
        v
    }
}

/// BTCSign — Bitcoin signed message format: base64(header || r || s) where
/// header = 27 + recid + 4 (compressed pubkey flag).
pub fn btc_sign(seed: &[u8], path: &str, message: &[u8]) -> Result<String, String> {
    use base64::Engine;
    let key = crypto::derive_private_key_from_path(seed, path)?;
    let magic = b"\x18Bitcoin Signed Message:\n";
    let mut preimage = varint(magic.len());
    preimage.extend_from_slice(magic);
    preimage.extend_from_slice(&varint(message.len()));
    preimage.extend_from_slice(message);
    let hash = sha256d(&preimage);

    let sk = k256::ecdsa::SigningKey::from_slice(&key).map_err(|e| e.to_string())?;
    let (sig, recid) = sk.sign_prehash_recoverable(&hash).map_err(|e| e.to_string())?;
    let header = 27 + recid.is_y_odd() as u8 + 4;
    let mut compact = vec![header];
    compact.extend_from_slice(&sig.to_bytes());
    Ok(base64::engine::general_purpose::STANDARD.encode(compact))
}

// ---------------------------------------------------------------------------
// Bech32 (BIP-173) — Cosmos
// ---------------------------------------------------------------------------

const BECH32_GEN: [u32; 5] = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

fn bech32_hrp_expand(hrp: &str) -> Vec<u8> {
    let mut out = Vec::with_capacity(hrp.len() * 2 + 1);
    for b in hrp.bytes() {
        out.push(b >> 5);
    }
    out.push(0);
    for b in hrp.bytes() {
        out.push(b & 31);
    }
    out
}

fn bech32_polymod(values: &[u8]) -> u32 {
    let mut chk: u32 = 1;
    for &v in values {
        let top = chk >> 25;
        chk = ((chk & 0x1ffffff) << 5) ^ v as u32;
        for (i, g) in BECH32_GEN.iter().enumerate() {
            if (top >> i) & 1 == 1 {
                chk ^= g;
            }
        }
    }
    chk
}

fn bech32_create_checksum(hrp: &str, data: &[u8]) -> [u8; 6] {
    let mut values = bech32_hrp_expand(hrp);
    values.extend_from_slice(data);
    values.extend_from_slice(&[0u8; 6]);
    let polymod = bech32_polymod(&values) ^ 1;
    let mut out = [0u8; 6];
    for i in 0..6 {
        out[i] = ((polymod >> (5 * (5 - i))) & 31) as u8;
    }
    out
}

/// Convert between bit groups (BIP-173 convertbits).
pub fn convert_bits(data: &[u8], from_bits: u32, to_bits: u32, pad: bool) -> Result<Vec<u8>, String> {
    let mut acc: u32 = 0;
    let mut bits: u32 = 0;
    let maxv: u32 = (1 << to_bits) - 1;
    let max_acc: u32 = (1 << (from_bits + to_bits - 1)) - 1;
    let mut out = Vec::new();
    for &b in data {
        let value = b as u32;
        if value >> from_bits != 0 {
            return Err("convertbits: value out of range".into());
        }
        acc = ((acc << from_bits) | value) & max_acc;
        bits += from_bits;
        while bits >= to_bits {
            bits -= to_bits;
            out.push(((acc >> bits) & maxv) as u8);
        }
    }
    if pad {
        if bits > 0 {
            out.push(((acc << (to_bits - bits)) & maxv) as u8);
        }
    } else if bits >= from_bits || ((acc << (to_bits - bits)) & maxv) != 0 {
        return Err("convertbits: invalid padding".into());
    }
    Ok(out)
}

/// Bech32Encode — BIP-173. `data` is the raw 8-bit payload (20-byte hash160).
pub fn bech32_encode(hrp: &str, data: &[u8]) -> Result<String, String> {
    if hrp.is_empty() || hrp.len() > 83 {
        return Err("invalid bech32 hrp".into());
    }
    let hrp = hrp.to_lowercase();
    let conv = convert_bits(data, 8, 5, true)?;
    let checksum = bech32_create_checksum(&hrp, &conv);
    let mut out = String::with_capacity(hrp.len() + 1 + conv.len() + 6);
    out.push_str(&hrp);
    out.push('1');
    for &d in conv.iter().chain(checksum.iter()) {
        out.push(BECH32_CHARSET[d as usize] as char);
    }
    Ok(out)
}

/// Bech32Decode — BIP-173, returns (hrp, 8-bit payload).
pub fn bech32_decode(s: &str) -> Result<(String, Vec<u8>), String> {
    if s.len() > 90 || s.len() < 8 {
        return Err("invalid bech32 length".into());
    }
    if s.bytes().any(|b| !(33..=126).contains(&b)) {
        return Err("invalid bech32 character range".into());
    }
    let lower = s.to_lowercase();
    let pos = lower.rfind('1').ok_or("missing bech32 separator")?;
    if pos == 0 || pos + 7 > lower.len() {
        return Err("invalid bech32 separator position".into());
    }
    let hrp = &lower[..pos];
    let mut data = Vec::with_capacity(lower.len() - pos - 1);
    for c in lower[pos + 1..].bytes() {
        let v = BECH32_CHARSET
            .iter()
            .position(|&x| x == c)
            .ok_or("invalid bech32 data character")? as u8;
        data.push(v);
    }
    let mut check = bech32_hrp_expand(hrp);
    check.extend_from_slice(&data);
    if bech32_polymod(&check) != 1 {
        return Err("bech32 checksum mismatch".into());
    }
    let payload5 = &data[..data.len() - 6];
    Ok((hrp.to_string(), convert_bits(payload5, 5, 8, false)?))
}

/// CosmosAddressFromSeed — secp256k1 → compressed pubkey → hash160 → bech32.
pub fn cosmos_address_from_seed(seed: &[u8], path: &str, prefix: &str) -> Result<String, String> {
    let key = crypto::derive_private_key_from_path(seed, path)?;
    let sk = SecretKey::from_slice(&key).map_err(|e| e.to_string())?;
    let compressed = sk.public_key().to_encoded_point(true);
    let h160 = hash160(compressed.as_bytes());
    bech32_encode(prefix, &h160)
}

/// CosmosSign — amino-style canonical JSON sign doc, sha256, secp256k1 (64-byte r||s).
pub fn cosmos_sign(
    seed: &[u8],
    path: &str,
    chain_id: &str,
    account_number: u64,
    sequence: u64,
    msg_json: &str,
) -> Result<Vec<u8>, String> {
    let key = crypto::derive_private_key_from_path(seed, path)?;
    let doc = serde_json::json!({
        "account_number": account_number.to_string(),
        "chain_id": chain_id,
        "fee": {"amount": [], "gas": "200000"},
        "memo": "",
        "msgs": [serde_json::from_str::<serde_json::Value>(msg_json).map_err(|e| e.to_string())?],
        "sequence": sequence.to_string(),
    });
    let doc_bytes = serde_json::to_vec(&doc).map_err(|e| e.to_string())?;
    let hash: [u8; 32] = Sha256::digest(&doc_bytes).into();
    let sk = k256::ecdsa::SigningKey::from_slice(&key).map_err(|e| e.to_string())?;
    let (sig, _recid) = sk.sign_prehash_recoverable(&hash).map_err(|e| e.to_string())?;
    Ok(sig.to_bytes().to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;

    const TEST_MNEMONIC: &str = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";

    #[test]
    fn slip10_official_vector() {
        // SLIP-0010 ed25519 test vector 1: seed 000102030405060708090a0b0c0d0e0f, m/0'
        let seed = hex::decode("000102030405060708090a0b0c0d0e0f").unwrap();
        let key = slip10_derive(&seed, "m/0'").unwrap();
        assert_eq!(
            hex::encode(key),
            "68e0fe46dfb67e368c75379acec591dad19df3cde26e63b93a8e704f1dade7a3"
        );
        // m/0'/1'
        let key2 = slip10_derive(&seed, "m/0'/1'").unwrap();
        assert_eq!(
            hex::encode(key2),
            "b1d0bad404bf35da785a64ca1ac54b2617211d2777696fbffaf208f746ae84f2"
        );
        // Non-hardened segment must be rejected for ed25519.
        assert!(slip10_derive(&seed, "m/0").is_err());
    }

    #[test]
    fn solana_sign_verify_roundtrip() {
        let seed = crypto::mnemonic_to_seed(TEST_MNEMONIC);
        let path = "m/44'/501'/0'/0'";
        let addr = solana_address_from_seed(&seed, path).unwrap();
        assert!(!addr.is_empty());
        assert!(addr.len() >= 32 && addr.len() <= 44);

        let key = slip10_derive(&seed, path).unwrap();
        let sk = SigningKey::from_bytes(&key);
        let msg = b"self-hosted master wallet";
        let sig_bytes = solana_sign(&seed, path, msg).unwrap();
        assert_eq!(sig_bytes.len(), 64);
        let sig = ed25519_dalek::Signature::from_bytes(&sig_bytes.try_into().unwrap());
        use ed25519_dalek::Verifier;
        sk.verifying_key().verify(msg, &sig).unwrap();
    }

    #[test]
    fn base58check_known_vector() {
        // All-zero hash160, version 0x00 → the well-known Bitcoin address.
        let addr = base58check_encode(0x00, &[0u8; 20]);
        assert_eq!(addr, "1111111111111111111114oLvT2");
        let (version, payload) = base58check_decode(&addr).unwrap();
        assert_eq!(version, 0x00);
        assert_eq!(payload, vec![0u8; 20]);
        // Corrupted checksum must fail.
        assert!(base58check_decode("1111111111111111111114oLvT3").is_err());
    }

    #[test]
    fn base58_roundtrip() {
        for case in [&b""[..], b"a", b"hello world", &[0, 0, 1, 2, 3][..]] {
            let enc = base58_encode(case);
            let dec = base58_decode(&enc).unwrap();
            assert_eq!(dec, case);
        }
    }

    #[test]
    fn btc_p2pkh_privkey_one_vector() {
        // Private key = 1 → well-known mainnet P2PKH address.
        let mut key = [0u8; 32];
        key[31] = 1;
        assert_eq!(
            btc_address_from_privkey(&key),
            "1BgGZ9tcN4rm9KBzDn7KprQz87SZ26SAMH"
        );
    }

    #[test]
    fn btc_sign_produces_valid_base64() {
        use base64::Engine;
        let seed = crypto::mnemonic_to_seed(TEST_MNEMONIC);
        let sig_b64 = btc_sign(&seed, "m/44'/0'/0'/0/0", b"hello bitcoin").unwrap();
        let raw = base64::engine::general_purpose::STANDARD.decode(&sig_b64).unwrap();
        assert_eq!(raw.len(), 65);
        assert!(raw[0] == 31 || raw[0] == 32, "compressed header 27+recid+4");
    }

    #[test]
    fn bech32_bip173_vector() {
        // BIP-173 valid checksum example: hrp "a", empty data.
        assert_eq!(bech32_encode("a", &[]).unwrap(), "a12uel5l");
    }

    #[test]
    fn bech32_cosmos_roundtrip() {
        let h160 = hash160(&hex::decode("0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798").unwrap());
        let addr = bech32_encode("cosmos", &h160).unwrap();
        assert!(addr.starts_with("cosmos1"));
        let (hrp, payload) = bech32_decode(&addr).unwrap();
        assert_eq!(hrp, "cosmos");
        assert_eq!(payload, h160.to_vec());
        // Uppercase encoding must decode too (BIP-173 mixed case forbidden, all-upper ok).
        let (hrp2, payload2) = bech32_decode(&addr.to_uppercase()).unwrap();
        assert_eq!(hrp2, "cosmos");
        assert_eq!(payload2, h160.to_vec());
        // Corrupted checksum must fail closed.
        let mut bad = addr.clone().into_bytes();
        let n = bad.len();
        bad[n - 1] = if bad[n - 1] == b'q' { b'p' } else { b'q' };
        assert!(bech32_decode(std::str::from_utf8(&bad).unwrap()).is_err());
    }

    #[test]
    fn cosmos_address_deterministic() {
        let seed = crypto::mnemonic_to_seed(TEST_MNEMONIC);
        let a1 = cosmos_address_from_seed(&seed, "m/44'/118'/0'/0/0", "cosmos").unwrap();
        let a2 = cosmos_address_from_seed(&seed, "m/44'/118'/0'/0/0", "cosmos").unwrap();
        assert_eq!(a1, a2);
        assert!(a1.starts_with("cosmos1"));
        let osmo = cosmos_address_from_seed(&seed, "m/44'/118'/0'/0/0", "osmo").unwrap();
        assert!(osmo.starts_with("osmo1"));
        // Same key, different prefix → same payload.
        let (_, p1) = bech32_decode(&a1).unwrap();
        let (_, p2) = bech32_decode(&osmo).unwrap();
        assert_eq!(p1, p2);
    }

    #[test]
    fn cosmos_sign_vector_shape() {
        let seed = crypto::mnemonic_to_seed(TEST_MNEMONIC);
        let sig = cosmos_sign(&seed, "m/44'/118'/0'/0/0", "cosmoshub-4", 1, 0, r#"{"@type":"/cosmos.bank.v1beta1.MsgSend"}"#).unwrap();
        assert_eq!(sig.len(), 64);
    }
}
