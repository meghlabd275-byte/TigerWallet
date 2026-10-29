//! Wallet Module - BIP32/BIP44 HD wallet implementation
//!
//! Derives secp256k1 keys along BIP-0044 paths. Address formats follow the
//! canonical rules per chain family: EVM chains use Keccak-256 of the
//! uncompressed public key with EIP-55 checksum, Bitcoin uses HASH160 +
//! Base58Check, Solana uses Base58 of the raw Ed25519-style 32-byte point.

use crate::{Mnemonic, WalletError};
use hmac::{Hmac, Mac};
use ripemd::Ripemd160;
use secp256k1::{PublicKey, Secp256k1, SecretKey};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256, Sha512};
use sha3::Keccak256;
use zeroize::Zeroize;

type HmacSha512 = Hmac<Sha512>;

/// Supported chains
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum Chain {
    Ethereum,
    BinanceSmartChain,
    Polygon,
    Avalanche,
    Arbitrum,
    Optimism,
    Solana,
    Bitcoin,
    Base,
    ArbitrumNova,
}

impl Chain {
    pub fn chain_id(&self) -> u64 {
        match self {
            Chain::Ethereum => 1,
            Chain::BinanceSmartChain => 56,
            Chain::Polygon => 137,
            Chain::Avalanche => 43114,
            Chain::Arbitrum => 42161,
            Chain::Optimism => 10,
            Chain::Solana => 0,
            Chain::Bitcoin => 0,
            Chain::Base => 8453,
            Chain::ArbitrumNova => 42170,
        }
    }

    pub fn coin_type(&self) -> u32 {
        match self {
            Chain::Ethereum | Chain::BinanceSmartChain | Chain::Polygon
            | Chain::Avalanche | Chain::Arbitrum | Chain::Optimism
            | Chain::Base | Chain::ArbitrumNova => 60,
            Chain::Solana => 501,
            Chain::Bitcoin => 0,
        }
    }

    pub fn is_evm(&self) -> bool {
        !matches!(self, Chain::Solana | Chain::Bitcoin)
    }

    pub fn from_str(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "bsc" | "binancesmartchain" | "bnb" => Chain::BinanceSmartChain,
            "polygon" | "matic" => Chain::Polygon,
            "avalanche" | "avax" => Chain::Avalanche,
            "arbitrum" | "arb" => Chain::Arbitrum,
            "optimism" | "op" => Chain::Optimism,
            "solana" | "sol" => Chain::Solana,
            "bitcoin" | "btc" => Chain::Bitcoin,
            "base" => Chain::Base,
            "arbitrumnova" | "nova" => Chain::ArbitrumNova,
            _ => Chain::Ethereum,
        }
    }
}

/// An extended key at a single BIP32 node: private key plus chain code.
pub struct ExtendedKey {
    private_key: [u8; 32],
    chain_code: [u8; 32],
}

impl Drop for ExtendedKey {
    fn drop(&mut self) {
        self.private_key.zeroize();
        self.chain_code.zeroize();
    }
}

/// HD Wallet bound to a single chain.
#[derive(Debug, Clone)]
pub struct HDWallet {
    mnemonic: Mnemonic,
    seed: [u8; 64],
    chain: Chain,
}

impl HDWallet {
    /// Generate new wallet for chain
    pub fn generate(chain: Chain) -> Self {
        let mnemonic = Mnemonic::generate(256).expect("256-bit entropy is valid");
        let seed = mnemonic.to_seed(None);
        Self { mnemonic, seed, chain }
    }

    /// Create from mnemonic
    pub fn from_mnemonic(mnemonic: Mnemonic, chain: Chain) -> Self {
        let seed = mnemonic.to_seed(None);
        Self { mnemonic, seed, chain }
    }

    /// Create from raw seed
    pub fn from_seed(seed: [u8; 64], chain: Chain) -> Self {
        let mnemonic = Mnemonic::from_entropy(seed[..32].to_vec())
            .unwrap_or_else(|_| Mnemonic::from_entropy(vec![0u8; 16]).expect("16 bytes valid"));
        Self { mnemonic, seed, chain }
    }

    pub fn chain(&self) -> Chain {
        self.chain
    }

    /// Get default address
    pub fn default_address(&self) -> Result<String, WalletError> {
        self.derive_address(44, self.chain.coin_type(), 0)
    }

    /// Derive the BIP44 account address: m/44'/coin'/account'/0/0
    pub fn derive_address(
        &self,
        purpose: u32,
        coin: u32,
        account: u32,
    ) -> Result<String, WalletError> {
        let key = self.derive_path(&[
            purpose | 0x8000_0000,
            coin | 0x8000_0000,
            account | 0x8000_0000,
            0,
            0,
        ])?;
        self.address_from_private_key(&key.private_key)
    }

    /// Derive an address at a full BIP44 path: m/44'/coin'/account'/change/index
    pub fn derive_address_at(
        &self,
        account: u32,
        change: u32,
        index: u32,
    ) -> Result<String, WalletError> {
        let key = self.derive_path(&[
            44 | 0x8000_0000,
            self.chain.coin_type() | 0x8000_0000,
            account | 0x8000_0000,
            change,
            index,
        ])?;
        self.address_from_private_key(&key.private_key)
    }

    /// Derive the raw private key for a BIP44 account path.
    pub fn derive_private_key_for_account(&self, account: u32) -> Result<[u8; 32], WalletError> {
        let key = self.derive_path(&[
            44 | 0x8000_0000,
            self.chain.coin_type() | 0x8000_0000,
            account | 0x8000_0000,
            0,
            0,
        ])?;
        Ok(key.private_key)
    }

    /// Derive a BIP32 child key along the given index path.
    fn derive_path(&self, path: &[u32]) -> Result<ExtendedKey, WalletError> {
        let mut key = master_key_from_seed(&self.seed);
        for &index in path {
            key = derive_child(&key, index)?;
        }
        Ok(key)
    }

    /// Convert a private key into the address format for this chain.
    fn address_from_private_key(&self, private_key: &[u8; 32]) -> Result<String, WalletError> {
        let secp = Secp256k1::new();
        let secret = SecretKey::from_slice(private_key)
            .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;
        let public = PublicKey::from_secret_key(&secp, &secret);
        let uncompressed = public.serialize_uncompressed();

        if self.chain.is_evm() {
            Ok(evm_address(&uncompressed))
        } else if self.chain == Chain::Bitcoin {
            Ok(bitcoin_p2pkh(&uncompressed))
        } else {
            // Solana: base58 of the 32-byte public key.
            let compressed = public.serialize();
            Ok(bs58::encode(&compressed[1..]).into_string())
        }
    }

    /// Sign a 32-byte digest, returning the 65-byte r||s||v signature.
    pub fn sign_digest(&self, digest: &[u8; 32]) -> Result<[u8; 65], WalletError> {
        let private_key = self.derive_private_key_for_account(0)?;
        let secp = Secp256k1::new();
        let secret = SecretKey::from_slice(&private_key)
            .map_err(|e| WalletError::SigningFailed(e.to_string()))?;
        let message = secp256k1::Message::from_digest(*digest);
        let signature = secp.sign_ecdsa_recoverable(&message, &secret);
        let (recovery_id, compact) = signature.serialize_compact();

        let mut out = [0u8; 65];
        out[..64].copy_from_slice(&compact[..]);
        out[64] = recovery_id.to_i32() as u8;
        Ok(out)
    }

    /// Sign an arbitrary message using the EIP-191 personal_sign scheme.
    pub fn sign(&self, message: &[u8]) -> Result<[u8; 65], WalletError> {
        self.sign_digest(&eip191_digest(message))
    }

    /// Sign a transaction payload, returning a hex-encoded signature.
    pub fn sign_transaction(&self, tx: &[u8]) -> Result<String, WalletError> {
        Ok(hex::encode(self.sign(tx)?))
    }

    /// Get mnemonic phrase
    pub fn mnemonic_phrase(&self) -> String {
        self.mnemonic.phrase()
    }
}

impl Default for HDWallet {
    fn default() -> Self {
        Self::generate(Chain::Ethereum)
    }
}

/// Compute the EIP-191 personal_sign digest for a message.
pub fn eip191_digest(message: &[u8]) -> [u8; 32] {
    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let mut hasher = Keccak256::new();
    hasher.update(prefix.as_bytes());
    hasher.update(message);
    hasher.finalize().into()
}

/// BIP32 master key from seed: I = HMAC-SHA512(key="Bitcoin seed", data=seed).
pub fn master_key_from_seed(seed: &[u8]) -> ExtendedKey {
    let mut mac = HmacSha512::new_from_slice(b"Bitcoin seed").expect("HMAC accepts any key length");
    mac.update(seed);
    let i = mac.finalize().into_bytes();

    let mut private_key = [0u8; 32];
    let mut chain_code = [0u8; 32];
    private_key.copy_from_slice(&i[..32]);
    chain_code.copy_from_slice(&i[32..]);
    ExtendedKey { private_key, chain_code }
}

/// BIP32 CKDpriv: derive one child key.
pub fn derive_child(parent: &ExtendedKey, index: u32) -> Result<ExtendedKey, WalletError> {
    let mut mac = HmacSha512::new_from_slice(&parent.chain_code)
        .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;

    if index >= 0x8000_0000 {
        // Hardened: data = 0x00 || ser256(k_par) || ser32(i)
        mac.update(&[0u8]);
        mac.update(&parent.private_key);
    } else {
        // Normal: data = serP(point(k_par)) || ser32(i)
        let secp = Secp256k1::new();
        let secret = SecretKey::from_slice(&parent.private_key)
            .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;
        let public = PublicKey::from_secret_key(&secp, &secret);
        mac.update(&public.serialize());
    }
    mac.update(&index.to_be_bytes());

    let i = mac.finalize().into_bytes();
    let il = &i[..32];

    // child = (IL + k_par) mod n
    let tweak = secp256k1::Scalar::from_be_bytes(il
        .try_into()
        .map_err(|_| WalletError::DerivationFailed("bad scalar".into()))?);
    let tweak = match tweak {
        Ok(t) => t,
        Err(_) => return Err(WalletError::DerivationFailed("invalid tweak".into())),
    };
    let parent_secret = SecretKey::from_slice(&parent.private_key)
        .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;
    let child_secret = parent_secret
        .add_tweak(&tweak)
        .map_err(|e| WalletError::DerivationFailed(e.to_string()))?;

    let mut chain_code = [0u8; 32];
    chain_code.copy_from_slice(&i[32..]);
    Ok(ExtendedKey {
        private_key: child_secret.secret_bytes(),
        chain_code,
    })
}

/// EVM address: last 20 bytes of Keccak-256(uncompressed pubkey without 0x04),
/// rendered with EIP-55 mixed-case checksum.
pub fn evm_address(uncompressed_public_key: &[u8; 65]) -> String {
    let mut hasher = Keccak256::new();
    hasher.update(&uncompressed_public_key[1..]);
    let hash = hasher.finalize();
    to_eip55(&hex::encode(&hash[12..]))
}

/// Apply the EIP-55 mixed-case checksum to a 40-character lowercase hex address.
pub fn to_eip55(address_hex: &str) -> String {
    let lower = address_hex.trim_start_matches("0x").to_lowercase();
    debug_assert_eq!(lower.len(), 40, "EVM address must be 20 bytes");

    let mut case_hasher = Keccak256::new();
    case_hasher.update(lower.as_bytes());
    let case_hash = hex::encode(case_hasher.finalize());

    let checksummed: String = lower
        .chars()
        .enumerate()
        .map(|(i, c)| {
            if c.is_ascii_digit() {
                c
            } else {
                let nibble = u8::from_str_radix(&case_hash[i..i + 1], 16).unwrap_or(0);
                if nibble >= 8 {
                    c.to_ascii_uppercase()
                } else {
                    c
                }
            }
        })
        .collect();

    format!("0x{}", checksummed)
}

/// Bitcoin P2PKH address: Base58Check(0x00 || HASH160(compressed pubkey)).
pub fn bitcoin_p2pkh(uncompressed_public_key: &[u8; 65]) -> String {
    let compressed = compress_public_key(uncompressed_public_key);
    let sha = Sha256::digest(compressed);
    let hash160 = Ripemd160::digest(sha);

    let mut payload = Vec::with_capacity(21);
    payload.push(0x00);
    payload.extend_from_slice(&hash160);

    let checksum = Sha256::digest(Sha256::digest(&payload));
    payload.extend_from_slice(&checksum[..4]);
    bs58::encode(payload).into_string()
}

/// Compress an uncompressed SEC1 public key.
pub fn compress_public_key(uncompressed: &[u8; 65]) -> [u8; 33] {
    let mut out = [0u8; 33];
    out[0] = if uncompressed[64] % 2 == 0 { 0x02 } else { 0x03 };
    out[1..].copy_from_slice(&uncompressed[1..33]);
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chain_ids() {
        assert_eq!(Chain::Ethereum.chain_id(), 1);
        assert_eq!(Chain::Polygon.chain_id(), 137);
        assert_eq!(Chain::Arbitrum.chain_id(), 42161);
    }

    /// BIP-0032 test vector 1: seed 000102030405060708090a0b0c0d0e0f
    #[test]
    fn test_bip32_vector_1() {
        let seed = hex::decode("000102030405060708090a0b0c0d0e0f").unwrap();
        let master = master_key_from_seed(&seed);
        assert_eq!(
            hex::encode(master.private_key),
            "e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b35"
        );
        assert_eq!(
            hex::encode(master.chain_code),
            "873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d508"
        );

        // m/0'
        let child = derive_child(&master, 0x8000_0000).unwrap();
        assert_eq!(
            hex::encode(child.private_key),
            "edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afea"
        );
        assert_eq!(
            hex::encode(child.chain_code),
            "47fdacbd0f1097043b78c63c20c34ef4ed9a111d980047ad16282c7ae6236141"
        );

        // m/0'/1
        let child = derive_child(&child, 1).unwrap();
        assert_eq!(
            hex::encode(child.private_key),
            "3c6cb8d0f6a264c91ea8b5030fadaa8e538b020f0a387421a12de9319dc93368"
        );
    }

    /// Known BIP-44 vector: the canonical Trezor test mnemonic must yield the
    /// well-known first Ethereum address.
    #[test]
    fn test_ethereum_address_vector() {
        let mnemonic = Mnemonic::from_phrase(
            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
        )
        .unwrap();
        let wallet = HDWallet::from_mnemonic(mnemonic, Chain::Ethereum);
        assert_eq!(
            wallet.default_address().unwrap(),
            "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
        );
    }

    /// Same mnemonic at m/44'/60'/0'/0/1 (second address of account 0).
    #[test]
    fn test_ethereum_address_vector_index_1() {
        let mnemonic = Mnemonic::from_phrase(
            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
        )
        .unwrap();
        let wallet = HDWallet::from_mnemonic(mnemonic, Chain::Ethereum);
        assert_eq!(
            wallet.derive_address_at(0, 0, 1).unwrap(),
            "0x6Fac4D18c912343BF86fa7049364Dd4E424Ab9C0"
        );
    }

    /// m/44'/60'/1'/0/0 (account 1) must differ from account 0.
    #[test]
    fn test_ethereum_account_1_is_distinct() {
        let mnemonic = Mnemonic::from_phrase(
            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
        )
        .unwrap();
        let wallet = HDWallet::from_mnemonic(mnemonic, Chain::Ethereum);
        let account_1 = wallet.derive_address_at(1, 0, 0).unwrap();
        assert_ne!(account_1, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
        assert_eq!(account_1.len(), 42);
    }

    /// Bitcoin P2PKH for the same mnemonic (BIP-44 coin type 0).
    #[test]
    fn test_bitcoin_address_vector() {
        let mnemonic = Mnemonic::from_phrase(
            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
        )
        .unwrap();
        let wallet = HDWallet::from_mnemonic(mnemonic, Chain::Bitcoin);
        assert_eq!(
            wallet.default_address().unwrap(),
            "1LqBGSKuX5yYUonjxT5qGfpUsXKYYWeabA"
        );
    }

    #[test]
    fn test_wallet_generation() {
        let wallet = HDWallet::generate(Chain::Ethereum);
        let address = wallet.default_address().unwrap();
        assert!(address.starts_with("0x"));
        assert_eq!(address.len(), 42);
    }

    #[test]
    fn test_distinct_accounts() {
        let wallet = HDWallet::generate(Chain::Ethereum);
        assert_ne!(
            wallet.derive_address(44, 60, 0).unwrap(),
            wallet.derive_address(44, 60, 1).unwrap()
        );
    }

    #[test]
    fn test_signing_roundtrip() {
        let wallet = HDWallet::generate(Chain::Ethereum);
        let signature = wallet.sign(b"test message").unwrap();
        assert_eq!(signature.len(), 65);
        assert!(signature[64] <= 1, "recovery id must be 0 or 1");
    }

    /// Official EIP-55 test vectors from the specification.
    #[test]
    fn test_eip55_official_vectors() {
        let vectors = [
            "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
            "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
            "0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
            "0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",
        ];
        for expected in vectors {
            assert_eq!(to_eip55(expected), expected);
            // Lowercased input must checksum back to the same mixed-case form.
            assert_eq!(to_eip55(&expected.to_lowercase()), expected);
        }
    }
}
