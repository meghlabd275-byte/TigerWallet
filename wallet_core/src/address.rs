//! Multi-chain address derivation utilities.
//!
//! Converts derived public keys / key material into chain-specific addresses:
//! - EVM (Ethereum/BSC/Polygon/etc.): Keccak-256 of the uncompressed pubkey, last 20 bytes.
//! - Bitcoin: P2PKH (Base58Check) and P2WPKH (Bech32) from a compressed pubkey.
//! - Solana: ed25519 public key, Base58.
//! - Cosmos: Bech32 with chain-specific HRP.
//! - TRON: Base58Check of Keccak-256(pubkey)[12:].
//! - Aptos / Sui: hex of the ed25519/derivation pubkey.

use sha3::{Keccak256, Digest};
use ripemd::Ripemd160;
use sha2::Sha256;
use bs58;
use bech32::{ToBase32, Variant};

/// Derive an EVM address (0x-prefixed, 20 bytes / 40 hex chars) from a
/// 64-byte uncompressed secp256k1 public key (X || Y, no 0x04 prefix).
pub fn evm_address_from_pubkey(pubkey_xy: &[u8]) -> Result<String, &'static str> {
    if pubkey_xy.len() != 64 {
        return Err("expected 64-byte uncompressed pubkey (X||Y)");
    }
    let hash = Keccak256::digest(pubkey_xy);
    let addr = &hash[12..]; // last 20 bytes
    Ok(format!("0x{}", hex::encode(addr)))
}

/// Bitcoin P2PKH address (Base58Check) from a 33-byte compressed pubkey.
pub fn bitcoin_p2pkh_address(compressed_pubkey: &[u8], version: u8) -> Result<String, &'static str> {
    if compressed_pubkey.len() != 33 {
        return Err("expected 33-byte compressed pubkey");
    }
    // HASH160 = RIPEMD160(SHA256(pubkey))
    let sha = Sha256::digest(compressed_pubkey);
    let ripe = Ripemd160::digest(&sha);
    let mut payload = vec![version];
    payload.extend_from_slice(&ripe);
    let checksum = &Sha256::digest(&Sha256::digest(&payload))[..4];
    payload.extend_from_slice(checksum);
    Ok(bs58::encode(payload).into_string())
}

/// Bitcoin P2WPKH (native SegWit, Bech32) address from a 33-byte compressed pubkey.
pub fn bitcoin_p2wpkh_address(compressed_pubkey: &[u8], hrp: &str) -> Result<String, &'static str> {
    if compressed_pubkey.len() != 33 {
        return Err("expected 33-byte compressed pubkey");
    }
    let sha = Sha256::digest(compressed_pubkey);
    let ripe = Ripemd160::digest(&sha);
    bech32::encode(hrp, ripe.to_vec().to_base32(), Variant::Bech32)
        .map_err(|_| "bech32 encode failed")
}

/// Solana / Aptos / Sui address from an ed25519 32-byte public key (Base58).
pub fn base58_address(pubkey: &[u8]) -> Result<String, &'static str> {
    if pubkey.is_empty() {
        return Err("empty pubkey");
    }
    Ok(bs58::encode(pubkey).into_string())
}

/// Cosmos / Tendermint Bech32 address from a 32-byte key hash or pubkey.
pub fn cosmos_bech32_address(key_material: &[u8], hrp: &str) -> Result<String, &'static str> {
    if key_material.is_empty() {
        return Err("empty key material");
    }
    // For Cosmos the "key material" is typically the HASH160 of the compressed pubkey.
    bech32::encode(hrp, key_material.to_vec().to_base32(), Variant::Bech32)
        .map_err(|_| "bech32 encode failed")
}

/// TRON Base58Check address from a 64-byte uncompressed secp256k1 pubkey.
pub fn tron_address_from_pubkey(pubkey_xy: &[u8]) -> Result<String, &'static str> {
    if pubkey_xy.len() != 64 {
        return Err("expected 64-byte uncompressed pubkey");
    }
    let hash = Keccak256::digest(pubkey_xy);
    let mut payload = vec![0x41u8]; // TRON mainnet version byte
    payload.extend_from_slice(&hash[12..]);
    let checksum = &Sha256::digest(&Sha256::digest(&payload))[..4];
    payload.extend_from_slice(checksum);
    Ok(bs58::encode(payload).into_string())
}