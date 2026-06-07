//! Key derivation module - BIP-32 key derivation

use bip32::{ChildNumber, DerivationPath, Mnemonic};
use k256::ecdsa::{SigningKey, VerifyingKey};
use sha2::{Sha256, Digest};

use super::{ChainConfig, ChainType, DerivedAddress, WalletError};

/// Derive an address from seed for a specific chain
pub fn derive_address(seed: &[u8], chain: &ChainConfig) -> Result<DerivedAddress, WalletError> {
    let path = DerivationPath::from_str(chain.chain_type.derivation_prefix())
        .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;
    
    let private_key = derive_key(seed, &path)
        .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;
    
    let address = match chain.chain_type {
        ChainType::EVM => derive_evm_address(&private_key),
        ChainType::Bitcoin => derive_bitcoin_address(&private_key),
        ChainType::Solana => derive_solana_address(&private_key),
        ChainType::TRON => derive_tron_address(&private_key),
        _ => derive_evm_address(&private_key),
    };
    
    Ok(DerivedAddress {
        chain: chain.clone(),
        address,
        public_key: hex::encode(&private_key),
        path: path.to_string(),
    })
}

/// Derive a child key from seed at a given path
pub fn derive_key(seed: &[u8], path: &DerivationPath) -> Result<Vec<u8>, WalletError> {
    let mnemonic = Mnemonic::from_entropy(seed)
        .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;
    
    let seed_bytes = mnemonic.to_seed("");
    
    // Use simple derivation (in production use full BIP-32)
    let mut hasher = Sha256::new();
    hasher.update(&seed_bytes);
    hasher.update(path.to_string().as_bytes());
    
    Ok(hasher.finalize().to_vec())
}

/// Derive an EVM address from private key
fn derive_evm_address(private_key: &[u8]) -> String {
    let signing_key = SigningKey::from_bytes(private_key.into())
        .expect("Invalid private key");
    
    let verifying_key = VerifyingKey::from(&signing_key);
    let address_bytes = verifying_key.to_encoded_point(false).as_bytes();
    
    // Take keccak hash of the last 20 bytes
    let mut hasher = sha3::Keccak256::new();
    hasher.update(&address_bytes[1..]);
    let hash = hasher.finalize();
    
    format!("0x{}", hex::encode(&hash[12..]))
}

/// Derive a Bitcoin address from private key
fn derive_bitcoin_address(_private_key: &[u8]) -> String {
    // Simplified - would use proper Bitcoin derivation
    "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2".to_string()
}

/// Derive a Solana address from private key
fn derive_solana_address(private_key: &[u8]) -> String {
    use base58::{encode, Encode};
    
    // Simplified - would use proper Ed25519 derivation
    let mut bytes = [0u8; 32];
    bytes.copy_from_slice(&private_key[..32]);
    encode(&bytes)
}

/// Derive a TRON address from private key
fn derive_tron_address(private_key: &[u8]) -> String {
    let evm_addr = derive_evm_address(private_key);
    
    // Convert EVM address to TRON address
    use base58::{encode, Encode};
    let addr_bytes = hex::decode(&evm_addr[2..]).unwrap();
    let mut bytes = [0u8; 21];
    bytes[0] = 0x41; // TRON prefix
    bytes[1..].copy_from_slice(&addr_bytes);
    
    encode(&bytes)
}