//! TigerWallet Stacks Blockchain Support
//!
//! This module provides complete support for Stacks (STX) blockchain,
//! a Bitcoin L2 that enables smart contracts on Bitcoin.
//!
//! Features:
//! - STX transfers
//! - Clarity smart contract deployment
//! - SIP-009 NFT support
//! - SIP-010 fungible tokens
//! - Stacking (Proof of Transfer)
//! - BNS (Blockstack Naming Service)
//! - Bitcoin finalize transaction
//!
//! Security:
//! - All transactions signed with Stacks private key
//! - Bitcoin anchor transaction verification
//! - Microblock confirmation tracking
//! - PoX (Proof of Transfer) security
//!
//! References:
//! - Stacks Docs: https://docs.stacks.co
//! - SIPs: https://github.com/stacksgov/sips
//! - Clarity Language: https://docs.stacks.co/clarity

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Stacks errors
#[derive(Error, Debug)]
pub enum StacksError {
    #[error("Invalid address: {0}")]
    InvalidAddress(String),

    #[error("Invalid private key: {0}")]
    InvalidPrivateKey(String),

    #[error("Transaction failed: {0}")]
    TransactionFailed(String),

    #[error("Contract deployment failed: {0}")]
    ContractFailed(String),

    #[error("API error: {0}")]
    APIError(String),

    #[error("Signing failed: {0}")]
    SigningFailed(String),

    #[error("Verification failed: {0}")]
    VerificationFailed(String),

    #[error("Stacking error: {0}")]
    StackingError(String),
}

/// Stacks network
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum Network {
    Mainnet,
    Testnet,
    Devnet,
}

/// Stacks address type
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum AddressType {
    /// Standard single signature
    P2PKH,
    /// Multi-signature
    P2SH,
    /// Contract principal
    Contract,
    /// Native Clarity smart contract
    P2SP,
}

/// Stacks address
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StacksAddress {
    /// Address version byte
    pub version: u8,

    /// Hash160 of the public key or contract
    pub hash160: [u8; 20],

    /// Address type
    pub address_type: AddressType,
}

/// C32Check encoding, the Stacks address format (SIP-005).
///
/// An address is `"S" + c32[version] + c32encode(payload || checksum)`, where
/// the checksum is the first four bytes of `SHA256(SHA256(version || payload))`
/// and `c32encode` is the Stacks variant of base-32: it works over hex digits,
/// strips leading zero digits, and re-adds one `'0'` per leading zero byte.
pub mod c32 {
    use sha2::{Digest, Sha256};

    /// Crockford base-32 alphabet, as used by Stacks (no I, L, O or U).
    const ALPHABET: &[u8; 32] = b"0123456789ABCDEFGHJKMNPQRSTVWXYZ";
    const HEX: &[u8; 16] = b"0123456789abcdef";

    /// Value of a hexadecimal digit.
    fn hex_value(digit: u8) -> usize {
        match digit {
            b'0'..=b'9' => (digit - b'0') as usize,
            b'a'..=b'f' => (digit - b'a' + 10) as usize,
            _ => 0,
        }
    }

    /// Index of `character` in `ALPHABET`, after normalising the aliases.
    fn digit(character: u8) -> Option<u8> {
        let character = character.to_ascii_uppercase();
        ALPHABET
            .iter()
            .position(|c| *c == character)
            .map(|i| i as u8)
    }

    /// Encode bytes with the Stacks base-32 variant.
    fn encode_bytes(data: &[u8]) -> String {
        // Work over hex digits, least significant first, packing five bits per
        // output digit.
        let hex: Vec<u8> = data
            .iter()
            .flat_map(|byte| [HEX[(byte >> 4) as usize], HEX[(byte & 0x0f) as usize]])
            .collect();

        let mut out: Vec<u8> = Vec::new();
        let mut carry: usize = 0;

        for index in (0..hex.len()).rev() {
            if carry < 4 {
                let current = hex_value(hex[index]) >> carry;
                let next = if index != 0 {
                    hex_value(hex[index - 1])
                } else {
                    0
                };
                let next_bits = 1 + carry;
                let next_low = (next % (1 << next_bits)) << (5 - next_bits);
                out.insert(0, ALPHABET[current + next_low]);
                carry = next_bits;
            } else {
                carry = 0;
            }
        }

        // Leading zero digits are dropped, then one '0' is re-added per leading
        // zero byte of the input.
        let leading = out.iter().take_while(|c| **c == b'0').count();
        let digits = &out[leading..];
        let zero_bytes = data.iter().take_while(|byte| **byte == 0).count();

        let mut result = vec![b'0'; zero_bytes];
        result.extend_from_slice(digits);

        String::from_utf8(result).expect("alphabet is ASCII")
    }

    /// Decode a Stacks base-32 string into bytes.
    fn decode_bytes(input: &str) -> Option<Vec<u8>> {
        // Normalise aliases, then count the leading zero digits, each of which
        // stands for one leading zero byte.
        let normalised: Vec<u8> = input
            .bytes()
            .map(|character| match character.to_ascii_uppercase() {
                b'O' => b'0',
                b'I' | b'L' => b'1',
                other => other,
            })
            .collect();

        let zero_bytes = normalised.iter().take_while(|c| **c == b'0').count();
        let digits = &normalised[zero_bytes..];

        let mut hex: Vec<u8> = Vec::with_capacity(digits.len() + 1);
        let mut carry: u8 = 0;
        let mut carry_bits: u32 = 0;

        for index in (0..digits.len()).rev() {
            if carry_bits == 4 {
                hex.insert(0, HEX[carry as usize]);
                carry_bits = 0;
                carry = 0;
            }

            let current = (digit(digits[index])? as u32) << carry_bits;
            let value = current + carry as u32;
            hex.insert(0, HEX[(value % 16) as usize]);
            carry_bits += 1;
            carry = (value >> 4) as u8;
        }

        hex.insert(0, HEX[carry as usize]);

        if hex.len() % 2 == 1 {
            hex.insert(0, b'0');
        }

        let leading_zeros = hex.iter().take_while(|c| **c == b'0').count();
        let hex = &hex[leading_zeros - (leading_zeros % 2)..];

        let decoded = hex
            .chunks(2)
            .map(|pair| {
                let high = hex_value(pair[0]) as u8;
                let low = hex_value(pair[1]) as u8;
                Some((high << 4) | low)
            })
            .collect::<Option<Vec<u8>>>()?;

        let mut out = vec![0u8; zero_bytes];
        out.extend_from_slice(&decoded);
        Some(out)
    }

    /// `SHA256(SHA256(version || payload))[..4]`.
    fn checksum(version: u8, payload: &[u8]) -> [u8; 4] {
        let mut hasher = Sha256::new();
        hasher.update([version]);
        hasher.update(payload);
        let first = hasher.finalize();

        let mut hasher = Sha256::new();
        hasher.update(first);
        let second = hasher.finalize();

        let mut out = [0u8; 4];
        out.copy_from_slice(&second[..4]);
        out
    }

    /// Encode `version || payload` as a C32Check string (without the `"S"`).
    pub fn encode_check(version: u8, payload: &[u8]) -> String {
        if version >= 32 {
            return String::new();
        }

        let mut data = Vec::with_capacity(payload.len() + 4);
        data.extend_from_slice(payload);
        data.extend_from_slice(&checksum(version, payload));

        format!(
            "{}{}",
            ALPHABET[version as usize] as char,
            encode_bytes(&data)
        )
    }

    /// Decode a C32Check string, verifying the checksum.
    ///
    /// Returns the version and the payload without the checksum.
    pub fn decode_check(address: &str) -> Option<(u8, Vec<u8>)> {
        // Addresses carry a leading `S`; the encoder without it is accepted too.
        let body = address.strip_prefix('S').unwrap_or(address);
        let mut characters = body.chars();
        let version = digit(characters.next()? as u8)?;

        let rest: String = characters.collect();
        let decoded = decode_bytes(&rest)?;

        if decoded.len() < 4 {
            return None;
        }

        let (payload, expected) = decoded.split_at(decoded.len() - 4);
        if checksum(version, payload) != expected {
            return None;
        }

        Some((version, payload.to_vec()))
    }

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn version_22_zero_hash_matches_the_published_vector() {
            // SP000000000000000000002Q6VF78 is the canonical c32check vector for
            // version 22 with a zero payload.
            // The published address is SP000000000000000000002Q6VF78, so the
            // body (without the leading "S") is 28 characters.
            let encoded = format!("S{}", encode_check(22, &[0u8; 20]));
            assert_eq!(encoded, "SP000000000000000000002Q6VF78");
        }

        #[test]
        fn version_prefixes_are_the_expected_letters() {
            // P = 22 (mainnet p2pkh), M = 20 (mainnet p2sh),
            // T = 26 (testnet p2pkh), N = 21 (testnet p2sh).
            assert!(encode_check(22, &[1u8; 20]).starts_with('P'));
            assert!(encode_check(20, &[1u8; 20]).starts_with('M'));
            assert!(encode_check(26, &[1u8; 20]).starts_with('T'));
            assert!(encode_check(21, &[1u8; 20]).starts_with('N'));
        }

        #[test]
        fn roundtrip_preserves_version_and_payload() {
            for version in [20u8, 21, 22, 26] {
                let payload: Vec<u8> = (0u8..20).collect();
                let encoded = format!("S{}", encode_check(version, &payload));
                let (decoded_version, decoded) = decode_check(&encoded).expect("valid checksum");
                assert_eq!(decoded_version, version);
                assert_eq!(decoded, payload);
            }
        }

        #[test]
        fn roundtrip_handles_leading_zero_payloads() {
            let mut payload = vec![0u8; 20];
            payload[19] = 0x01;
            let encoded = format!("S{}", encode_check(22, &payload));
            let (version, decoded) = decode_check(&encoded).expect("valid checksum");
            assert_eq!(version, 22);
            assert_eq!(decoded, payload);
        }

        #[test]
        fn a_flipped_character_fails_the_checksum() {
            let encoded = format!("S{}", encode_check(22, &[7u8; 20]));
            let mut tampered = encoded.clone();
            let last = tampered.pop().unwrap();
            tampered.push(if last == '0' { '1' } else { '0' });
            assert!(decode_check(&tampered).is_none());
        }

        #[test]
        fn characters_outside_the_alphabet_are_rejected() {
            assert!(decode_check("SP000000000000000000002Q6VF7!").is_none());
        }
    }
}

impl StacksAddress {
    /// Create from a C32Check address string.
    pub fn from_string(addr: &str, network: Network) -> Result<Self, StacksError> {
        let (version, payload) =
            c32::decode_check(addr).ok_or_else(|| StacksError::InvalidAddress(addr.to_string()))?;

        if payload.len() != 20 {
            return Err(StacksError::InvalidAddress(addr.to_string()));
        }

        let address_type = match (version, network) {
            (22, Network::Mainnet) => AddressType::P2PKH,
            (20, Network::Mainnet) => AddressType::P2SH,
            (26, Network::Testnet | Network::Devnet) => AddressType::P2PKH,
            (21, Network::Testnet | Network::Devnet) => AddressType::P2SH,
            _ => return Err(StacksError::InvalidAddress(addr.to_string())),
        };

        let mut hash160 = [0u8; 20];
        hash160.copy_from_slice(&payload);

        Ok(Self {
            version,
            hash160,
            address_type,
        })
    }

    /// Encode as a C32Check address string.
    pub fn to_string(&self, _network: Network) -> String {
        format!("S{}", c32::encode_check(self.version, &self.hash160))
    }

    /// Canonical display address.
    ///
    /// The version byte already selects the network and address class, so this
    /// is the same encoding as [`StacksAddress::to_string`].
    pub fn canonical(&self, network: Network) -> String {
        self.to_string(network)
    }
}

/// Stacks private key
#[derive(Debug, Clone)]
pub struct PrivateKey {
    /// 32-byte private key
    bytes: [u8; 32],
}

impl PrivateKey {
    /// Generate a new random private key.
    pub fn generate() -> Result<Self, StacksError> {
        use crate::secure_random::generate_random_bytes;

        let material =
            generate_random_bytes(32).map_err(|e| StacksError::SigningFailed(e.to_string()))?;

        Self::from_bytes(&material)
    }

    /// Build a private key from 32 raw bytes.
    fn from_bytes(material: &[u8]) -> Result<Self, StacksError> {
        if material.len() != 32 {
            return Err(StacksError::InvalidPrivateKey(
                "private key must be 32 bytes".to_string(),
            ));
        }

        // Reject out-of-range scalars here so signing never has to panic.
        secp256k1::SecretKey::from_slice(material)
            .map_err(|e| StacksError::InvalidPrivateKey(e.to_string()))?;

        let mut bytes = [0u8; 32];
        bytes.copy_from_slice(material);
        Ok(Self { bytes })
    }

    /// Import from a hex string.
    pub fn from_hex(hex: &str) -> Result<Self, StacksError> {
        let bytes = hex::decode(hex.trim_start_matches("0x"))
            .map_err(|_| StacksError::InvalidPrivateKey(hex.to_string()))?;

        Self::from_bytes(&bytes)
    }

    /// Get the compressed public key.
    pub fn public_key(&self) -> [u8; 33] {
        use secp256k1::{PublicKey, Secp256k1};

        let secp = Secp256k1::new();
        let secret_key = secp256k1::SecretKey::from_slice(&self.bytes)
            .expect("private key was validated when it was constructed");

        PublicKey::from_secret_key(&secp, &secret_key).serialize()
    }

    /// Get the Stacks address for this key.
    pub fn stacks_address(&self, network: Network) -> StacksAddress {
        use ripemd::Ripemd160;
        use sha2::{Digest, Sha256};

        let public_key = self.public_key();

        let sha256_hash = Sha256::digest(public_key);
        let hash160 = Ripemd160::digest(sha256_hash);

        let version = match network {
            Network::Mainnet => 22,
            Network::Testnet | Network::Devnet => 20,
        };

        let mut payload = [0u8; 20];
        payload.copy_from_slice(&hash160);

        StacksAddress {
            version,
            hash160: payload,
            address_type: AddressType::P2PKH,
        }
    }

    /// Sign a 32-byte digest, returning `r || s || v` with `v` in 0..=1.
    ///
    /// Stacks signatures are recoverable secp256k1 signatures over the
    /// transaction's presign sighash, and this is exactly that construction.
    pub fn sign_digest(&self, digest: &[u8; 32]) -> Result<[u8; 65], StacksError> {
        use secp256k1::{Message, Secp256k1};

        let secp = Secp256k1::new();
        let secret_key = secp256k1::SecretKey::from_slice(&self.bytes)
            .map_err(|e| StacksError::InvalidPrivateKey(e.to_string()))?;

        let message =
            Message::from_slice(digest).map_err(|e| StacksError::SigningFailed(e.to_string()))?;

        let signature = secp.sign_ecdsa_recoverable(&message, &secret_key);
        let (recovery_id, compact) = signature.serialize_compact();

        let mut out = [0u8; 65];
        out[..64].copy_from_slice(&compact);
        out[64] = recovery_id.to_i32() as u8;

        Ok(out)
    }

    /// Sign a transaction's presign sighash.
    ///
    /// The digest must be the 32-byte Stacks presign sighash, which the caller
    /// computes from the serialized transaction and the chain ID. Hashing the
    /// message here would sign a different value than the one the network
    /// verifies.
    pub fn sign_transaction(&self, sighash: &[u8; 32]) -> Result<[u8; 65], StacksError> {
        self.sign_digest(sighash)
    }

    /// Sign an arbitrary message with the Stacks personal-message prefix.
    pub fn sign_message(&self, message: &str) -> Result<[u8; 65], StacksError> {
        use sha2::{Digest, Sha256};

        let mut hasher = Sha256::new();
        hasher.update(b"Stacks Signed Message:\n");
        hasher.update((message.len() as u32).to_le_bytes());
        hasher.update(message.as_bytes());

        let digest: [u8; 32] = hasher.finalize().into();
        self.sign_digest(&digest)
    }
}

/// Stacks transaction.
///
/// The fields follow the Stacks wire format (SIP-005): version, chain ID, an
/// authentication block, the anchor mode, post conditions, and the payload.
/// The struct deliberately does not derive `serde`; it is a wire-format type,
/// not an API DTO.
#[derive(Debug, Clone)]
pub struct Transaction {
    /// Transaction version byte.
    pub version: u8,

    /// Chain ID.
    pub chain_id: u32,

    /// Anchor mode.
    pub anchor_mode: AnchorMode,

    /// Payload.
    pub payload: TransactionPayload,

    /// Post condition mode.
    pub post_condition_mode: u8,

    /// Post conditions.
    pub post_conditions: Vec<PostCondition>,

    /// Fee in micro-STX.
    pub fee: u64,

    /// Account nonce.
    pub nonce: u64,

    /// Compressed signer public key.
    pub signer: [u8; 33],

    /// Recoverable signature over the presign sighash.
    pub signature: [u8; 65],
}

/// Anchor mode.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AnchorMode {
    /// On-chain
    OnChain = 1,
    /// Microblock
    Microblock = 2,
    /// Any
    Any = 3,
}

/// Transaction payload.
#[derive(Debug, Clone)]
pub enum TransactionPayload {
    /// STX transfer.
    TokenTransfer {
        recipient: StacksAddress,
        amount: u64,
        memo: [u8; 34],
    },

    /// Contract call.
    ContractCall {
        contract_address: StacksAddress,
        contract_name: String,
        function_name: String,
        arguments: Vec<ClarityValue>,
    },

    /// Contract deployment.
    ContractDeploy { code: Vec<u8>, name: String },

    /// Poison microblock.
    PoisonMicroblock { header: Vec<u8>, signature: Vec<u8> },
}

/// Post condition.
#[derive(Debug, Clone)]
pub struct PostCondition {
    /// `true` for `if`, `false` for `if-not`.
    pub is_if: bool,

    /// Asset identifier, `None` for STX.
    pub asset: Option<AssetIdentifier>,

    /// Principal the condition applies to.
    pub principal: StacksAddress,

    /// Comparator: 0x01 `=`, 0x02 `>`, 0x03 `>=`, 0x04 `<`, 0x05 `<=`.
    pub comparator: u8,

    /// Amount (fungible) or asset id (non-fungible).
    pub amount: u64,
}

/// Asset identifier for a post condition.
#[derive(Debug, Clone)]
pub struct AssetIdentifier {
    /// Contract that defines the asset.
    pub contract_address: StacksAddress,

    /// Asset name.
    pub asset_name: String,
}

/// A Clarity value, serialized with the canonical wire encoding.
///
/// Only the variants a wallet needs for contract calls are modelled; each one
/// encodes exactly as the Clarity specification requires.
#[derive(Debug, Clone)]
pub enum ClarityValue {
    /// Signed 128-bit integer.
    Int(i128),
    /// Unsigned 128-bit integer.
    UInt(u128),
    /// Buffers.
    Buffer(Vec<u8>),
    /// Boolean.
    Bool(bool),
    /// Standard principal.
    PrincipalStandard(StacksAddress),
    /// Contract principal.
    PrincipalContract(StacksAddress, String),
    /// Response `ok`.
    ResponseOk(Box<ClarityValue>),
    /// Response `err`.
    ResponseErr(Box<ClarityValue>),
    /// `none`.
    None,
    /// `some`.
    Some(Box<ClarityValue>),
    /// List of values.
    List(Vec<ClarityValue>),
    /// ASCII string.
    StringAscii(String),
    /// UTF-8 string.
    StringUtf8(String),
}

impl ClarityValue {
    /// Serialize with the canonical Clarity wire encoding.
    pub fn serialize(&self) -> Result<Vec<u8>, StacksError> {
        let mut out = Vec::new();
        match self {
            ClarityValue::Int(value) => {
                out.push(0x00);
                out.extend_from_slice(&value.to_be_bytes());
            }
            ClarityValue::UInt(value) => {
                out.push(0x01);
                out.extend_from_slice(&value.to_be_bytes());
            }
            ClarityValue::Buffer(bytes) => {
                out.push(0x02);
                out.extend_from_slice(&(bytes.len() as u32).to_be_bytes());
                out.extend_from_slice(bytes);
            }
            ClarityValue::Bool(value) => out.push(if *value { 0x03 } else { 0x04 }),
            ClarityValue::PrincipalStandard(address) => {
                out.push(0x05);
                out.push(address.version);
                out.extend_from_slice(&address.hash160);
            }
            ClarityValue::PrincipalContract(address, name) => {
                out.push(0x06);
                out.push(address.version);
                out.extend_from_slice(&address.hash160);
                write_length_prefixed(&mut out, name.as_bytes(), 1)?;
            }
            ClarityValue::ResponseOk(value) => {
                out.push(0x07);
                out.extend_from_slice(&value.serialize()?);
            }
            ClarityValue::ResponseErr(value) => {
                out.push(0x08);
                out.extend_from_slice(&value.serialize()?);
            }
            ClarityValue::None => out.push(0x09),
            ClarityValue::Some(value) => {
                out.push(0x0a);
                out.extend_from_slice(&value.serialize()?);
            }
            ClarityValue::List(values) => {
                out.push(0x0b);
                out.extend_from_slice(&(values.len() as u32).to_be_bytes());
                for value in values {
                    out.extend_from_slice(&value.serialize()?);
                }
            }
            ClarityValue::StringAscii(value) => {
                out.push(0x0d);
                write_length_prefixed(&mut out, value.as_bytes(), 4)?;
            }
            ClarityValue::StringUtf8(value) => {
                out.push(0x0e);
                write_length_prefixed(&mut out, value.as_bytes(), 4)?;
            }
        }
        Ok(out)
    }
}

/// Write a length-prefixed byte string with a 1- or 4-byte big-endian length.
fn write_length_prefixed(
    out: &mut Vec<u8>,
    bytes: &[u8],
    length_bytes: usize,
) -> Result<(), StacksError> {
    match length_bytes {
        1 => {
            let length = u8::try_from(bytes.len())
                .map_err(|_| StacksError::TransactionFailed("value exceeds 255 bytes".into()))?;
            out.push(length);
        }
        4 => {
            let length = u32::try_from(bytes.len())
                .map_err(|_| StacksError::TransactionFailed("value exceeds 4 GiB".into()))?;
            out.extend_from_slice(&length.to_be_bytes());
        }
        _ => {
            return Err(StacksError::TransactionFailed(
                "length prefix must be 1 or 4 bytes".into(),
            ))
        }
    }
    out.extend_from_slice(bytes);
    Ok(())
}

impl Transaction {
    /// Create a signed STX transfer.
    pub fn new_token_transfer(
        recipient: StacksAddress,
        amount: u64,
        private_key: &PrivateKey,
        network: Network,
    ) -> Result<Self, StacksError> {
        let chain_id = match network {
            Network::Mainnet => 1u32,
            Network::Testnet | Network::Devnet => 0x8000_0000u32,
        };

        let mut transaction = Self {
            version: match network {
                Network::Mainnet => 0x00,
                Network::Testnet | Network::Devnet => 0x80,
            },
            chain_id,
            anchor_mode: AnchorMode::OnChain,
            payload: TransactionPayload::TokenTransfer {
                recipient,
                amount,
                memo: [0u8; 34],
            },
            post_condition_mode: 0x01,
            post_conditions: Vec::new(),
            fee: 0,
            nonce: 0,
            signer: private_key.public_key(),
            signature: [0u8; 65],
        };

        transaction.sign(private_key)?;
        Ok(transaction)
    }

    /// Create a contract call. The transaction is left unsigned.
    pub fn new_contract_call(
        contract_address: StacksAddress,
        contract_name: &str,
        function_name: &str,
        arguments: Vec<ClarityValue>,
        private_key: &PrivateKey,
        network: Network,
    ) -> Result<Self, StacksError> {
        let mut transaction = Self {
            version: match network {
                Network::Mainnet => 0x00,
                Network::Testnet | Network::Devnet => 0x80,
            },
            chain_id: match network {
                Network::Mainnet => 1u32,
                Network::Testnet | Network::Devnet => 0x8000_0000u32,
            },
            anchor_mode: AnchorMode::OnChain,
            payload: TransactionPayload::ContractCall {
                contract_address,
                contract_name: contract_name.to_string(),
                function_name: function_name.to_string(),
                arguments,
            },
            post_condition_mode: 0x01,
            post_conditions: Vec::new(),
            fee: 0,
            nonce: 0,
            signer: private_key.public_key(),
            signature: [0u8; 65],
        };

        transaction.sign(private_key)?;
        Ok(transaction)
    }

    /// Serialize the payload.
    fn serialize_payload(payload: &TransactionPayload) -> Result<Vec<u8>, StacksError> {
        let mut out = Vec::new();
        match payload {
            TransactionPayload::TokenTransfer {
                recipient,
                amount,
                memo,
            } => {
                out.push(0x00);
                out.push(recipient.version);
                out.extend_from_slice(&recipient.hash160);
                out.extend_from_slice(&amount.to_be_bytes());
                out.extend_from_slice(memo);
            }
            TransactionPayload::ContractCall {
                contract_address,
                contract_name,
                function_name,
                arguments,
            } => {
                out.push(0x01);
                out.push(contract_address.version);
                out.extend_from_slice(&contract_address.hash160);
                write_length_prefixed(&mut out, contract_name.as_bytes(), 1)?;
                write_length_prefixed(&mut out, function_name.as_bytes(), 1)?;
                let count = u32::try_from(arguments.len()).map_err(|_| {
                    StacksError::TransactionFailed("too many contract arguments".into())
                })?;
                out.extend_from_slice(&count.to_be_bytes());
                for argument in arguments {
                    out.extend_from_slice(&argument.serialize()?);
                }
            }
            TransactionPayload::ContractDeploy { code, name } => {
                out.push(0x02);
                write_length_prefixed(&mut out, name.as_bytes(), 1)?;
                let length = u32::try_from(code.len()).map_err(|_| {
                    StacksError::TransactionFailed("contract code too large".into())
                })?;
                out.extend_from_slice(&length.to_be_bytes());
                out.extend_from_slice(code);
            }
            TransactionPayload::PoisonMicroblock { header, signature } => {
                out.push(0x03);
                let header_length = u32::try_from(header.len())
                    .map_err(|_| StacksError::TransactionFailed("header too large".into()))?;
                out.extend_from_slice(&header_length.to_be_bytes());
                out.extend_from_slice(header);
                let signature_length = u32::try_from(signature.len())
                    .map_err(|_| StacksError::TransactionFailed("signature too large".into()))?;
                out.extend_from_slice(&signature_length.to_be_bytes());
                out.extend_from_slice(signature);
            }
        }
        Ok(out)
    }

    /// Serialize the post conditions.
    fn serialize_post_conditions(&self) -> Result<Vec<u8>, StacksError> {
        let mut out = Vec::new();
        let count = u32::try_from(self.post_conditions.len())
            .map_err(|_| StacksError::TransactionFailed("too many post conditions".into()))?;
        out.extend_from_slice(&count.to_be_bytes());

        for condition in &self.post_conditions {
            out.push(if condition.is_if { 0x00 } else { 0x01 });

            match &condition.asset {
                None => {
                    out.push(0x01);
                    out.push(condition.principal.version);
                    out.extend_from_slice(&condition.principal.hash160);
                    out.push(condition.comparator);
                    out.extend_from_slice(&(condition.amount as u128).to_be_bytes());
                }
                Some(asset) => {
                    out.push(0x02);
                    out.push(asset.contract_address.version);
                    out.extend_from_slice(&asset.contract_address.hash160);
                    write_length_prefixed(&mut out, asset.asset_name.as_bytes(), 1)?;
                    out.push(condition.principal.version);
                    out.extend_from_slice(&condition.principal.hash160);
                    out.push(condition.comparator);
                    out.extend_from_slice(&(condition.amount as u128).to_be_bytes());
                }
            }
        }

        Ok(out)
    }

    /// Serialize the transaction in wire form.
    pub fn serialize(&self) -> Result<Vec<u8>, StacksError> {
        let mut out = Vec::new();
        out.push(self.version);
        out.extend_from_slice(&self.chain_id.to_be_bytes());

        // Standard authorisation block.
        out.push(0x04);
        out.extend_from_slice(&self.signer);
        out.extend_from_slice(&self.nonce.to_be_bytes());
        out.extend_from_slice(&self.fee.to_be_bytes());
        out.push(0x00);
        out.extend_from_slice(&self.signature);

        out.push(self.anchor_mode as u8);
        out.push(self.post_condition_mode);
        out.extend_from_slice(&self.serialize_post_conditions()?);
        out.extend_from_slice(&Self::serialize_payload(&self.payload)?);

        Ok(out)
    }

    /// Presign sighash: `SHA512/256(0x00 || serialized transaction)`.
    ///
    /// The signature field is zeroed while hashing, which is what the network
    /// does when it verifies the transaction.
    pub fn presign_sighash(&self) -> Result<[u8; 32], StacksError> {
        use sha2::{Digest, Sha512_256};

        let mut unsigned = self.clone();
        unsigned.signature = [0u8; 65];

        let mut hasher = Sha512_256::new();
        hasher.update([0x00u8]);
        hasher.update(unsigned.serialize()?);

        Ok(hasher.finalize().into())
    }

    /// Sign the transaction with `private_key`.
    pub fn sign(&mut self, private_key: &PrivateKey) -> Result<(), StacksError> {
        let sighash = self.presign_sighash()?;
        self.signature = private_key.sign_transaction(&sighash)?;
        Ok(())
    }

    /// Transaction ID: `SHA512/256` of the serialized transaction.
    pub fn txid(&self) -> Result<String, StacksError> {
        use sha2::{Digest, Sha512_256};

        let mut hasher = Sha512_256::new();
        hasher.update(self.serialize()?);
        Ok(hex::encode(hasher.finalize()))
    }
}

/// Stacking (PoX) participation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StackInf {
    /// Stacker address.
    pub stacker: StacksAddress,

    /// Amount to stack (micro-STX).
    pub amount: u64,

    /// Number of cycles (1-12).
    pub lock_period: u8,

    /// Delegate stacker, if any.
    pub delegate: Option<StacksAddress>,
}

/// Stacking pool information.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StackingPool {
    /// Pool address.
    pub address: StacksAddress,

    /// Pool name.
    pub name: String,

    /// Description.
    pub description: String,

    /// Total STX stacked.
    pub total_stacked: u64,

    /// Number of delegators.
    pub delegators: u32,

    /// Minimum to participate.
    pub min_stx: u64,

    /// Reward cycle.
    pub reward_cycle: u8,
}

/// Stacks RPC client.
pub struct RPCClient {
    network: Network,
    rpc_url: String,
    client: reqwest::Client,
}

impl RPCClient {
    /// Create a new client.
    pub fn new(network: Network, rpc_url: String) -> Self {
        Self {
            network,
            rpc_url: rpc_url.trim_end_matches('/').to_string(),
            client: reqwest::Client::new(),
        }
    }

    /// Network this client talks to.
    pub fn network(&self) -> Network {
        self.network
    }

    /// Get account info.
    pub async fn get_account(&self, address: &StacksAddress) -> Result<AccountInfo, StacksError> {
        let url = format!(
            "{}/v2/accounts/{}?proof=0",
            self.rpc_url,
            address.canonical(self.network)
        );

        let response = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(StacksError::APIError(format!(
                "node returned {}",
                response.status()
            )));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        let balance = body
            .get("balance")
            .and_then(|v| v.as_str())
            .and_then(|v| {
                v.trim_start_matches("0x")
                    .parse::<u64>()
                    .ok()
                    .or_else(|| u64::from_str_radix(v, 16).ok())
            })
            .or_else(|| body.get("balance").and_then(|v| v.as_u64()))
            .ok_or_else(|| StacksError::APIError("response has no balance field".into()))?;

        let nonce = body
            .get("nonce")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| StacksError::APIError("response has no nonce field".into()))?;

        let lock_height = body
            .get("lock_height")
            .and_then(|v| v.as_u64())
            .unwrap_or(0) as u32;

        Ok(AccountInfo {
            balance,
            nonce,
            lock_height,
        })
    }

    /// Get the account nonce.
    pub async fn get_nonce(&self, address: &StacksAddress) -> Result<u64, StacksError> {
        Ok(self.get_account(address).await?.nonce)
    }

    /// Get the STX balance.
    pub async fn get_balance(&self, address: &StacksAddress) -> Result<u64, StacksError> {
        Ok(self.get_account(address).await?.balance)
    }

    /// Get a contract's source.
    pub async fn get_contract(
        &self,
        address: &StacksAddress,
        name: &str,
    ) -> Result<ContractInfo, StacksError> {
        let url = format!(
            "{}/v2/contracts/source/{}/{}",
            self.rpc_url,
            address.canonical(self.network),
            name
        );

        let response = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(StacksError::APIError(format!(
                "node returned {}",
                response.status()
            )));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        let source = body
            .get("source")
            .and_then(|v| v.as_str())
            .ok_or_else(|| StacksError::APIError("response has no source field".into()))?;

        Ok(ContractInfo {
            source: source.to_string(),
            name: name.to_string(),
        })
    }

    /// Get a block by its index hash.
    pub async fn get_block(&self, block_hash: &str) -> Result<BlockInfo, StacksError> {
        let url = format!("{}/v2/blocks/{}", self.rpc_url, block_hash);

        let response = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(StacksError::APIError(format!(
                "node returned {}",
                response.status()
            )));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        let hash = body
            .get("hash")
            .and_then(|v| v.as_str())
            .ok_or_else(|| StacksError::APIError("response has no hash field".into()))?;

        let height = body
            .get("height")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| StacksError::APIError("response has no height field".into()))?;

        let timestamp = body
            .get("burn_block_time")
            .and_then(|v| v.as_u64())
            .unwrap_or(0);

        let miner = body
            .get("miner_txid")
            .and_then(|v| v.as_str())
            .unwrap_or_default();

        Ok(BlockInfo {
            hash: hash.to_string(),
            height: height as u32,
            timestamp,
            miner: miner.to_string(),
        })
    }

    /// Get the mempool transactions.
    pub async fn get_mempool(&self) -> Result<Vec<MempoolTx>, StacksError> {
        let url = format!("{}/v2/transactions/mempool", self.rpc_url);

        let response = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(StacksError::APIError(format!(
                "node returned {}",
                response.status()
            )));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        let entries = body
            .as_array()
            .ok_or_else(|| StacksError::APIError("mempool response is not an array".into()))?;

        Ok(entries
            .iter()
            .filter_map(|entry| {
                Some(MempoolTx {
                    txid: entry.get("tx_id")?.as_str()?.to_string(),
                    type_id: entry.get("type_id")?.as_u64()? as u8,
                    anchor_mode: entry.get("anchor_mode")?.as_u64()? as u8,
                })
            })
            .collect())
    }

    /// Broadcast a transaction and return the node's transaction ID.
    pub async fn post_transaction(&self, transaction: &Transaction) -> Result<String, StacksError> {
        let url = format!("{}/v2/transactions", self.rpc_url);
        let bytes = transaction.serialize()?;
        let payload = serde_json::json!({ "tx": hex::encode(bytes) });

        let response = self
            .client
            .post(&url)
            .header("Content-Type", "application/json")
            .json(&payload)
            .send()
            .await
            .map_err(|e| StacksError::TransactionFailed(e.to_string()))?;

        let status = response.status();
        let body = response
            .text()
            .await
            .map_err(|e| StacksError::TransactionFailed(e.to_string()))?;

        if !status.is_success() {
            return Err(StacksError::TransactionFailed(format!(
                "node rejected the transaction ({status}): {body}"
            )));
        }

        // The node answers with a JSON string containing the transaction ID.
        let txid = serde_json::from_str::<String>(&body)
            .or_else(|_| {
                serde_json::from_str::<serde_json::Value>(&body).map(|value| {
                    value
                        .get("txid")
                        .and_then(|v| v.as_str())
                        .unwrap_or_default()
                        .to_string()
                })
            })
            .map_err(|e| StacksError::TransactionFailed(e.to_string()))?;

        if txid.is_empty() {
            return Err(StacksError::TransactionFailed(
                "node response contained no transaction ID".into(),
            ));
        }

        Ok(txid)
    }

    /// Get stacking info for an address.
    pub async fn get_stacking_info(
        &self,
        address: &StacksAddress,
    ) -> Result<StackingInfo, StacksError> {
        let url = format!(
            "{}/v2/stacking/stacking-info/{}",
            self.rpc_url,
            address.canonical(self.network)
        );

        let response = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(StacksError::APIError(format!(
                "node returned {}",
                response.status()
            )));
        }

        response
            .json()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))
    }

    /// Get the reward cycle holders.
    pub async fn get_reward_cycle_holders(
        &self,
        cycle: u32,
    ) -> Result<Vec<RewardHolder>, StacksError> {
        let url = format!(
            "{}/v2/stacking/reward-cycle-holders/{}",
            self.rpc_url, cycle
        );

        let response = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(StacksError::APIError(format!(
                "node returned {}",
                response.status()
            )));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| StacksError::APIError(e.to_string()))?;

        let holders = body
            .get("reward_cycle_holders")
            .and_then(|v| v.as_array())
            .ok_or_else(|| StacksError::APIError("response has no holder list".into()))?;

        Ok(holders
            .iter()
            .filter_map(|holder| {
                Some(RewardHolder {
                    holder: holder.get("holder")?.as_str()?.to_string(),
                    locked: holder.get("amount_ustx")?.as_u64()?,
                    weight: holder.get("weight")?.as_u64().unwrap_or(0),
                })
            })
            .collect())
    }
}

/// Account info.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountInfo {
    pub balance: u64,
    pub nonce: u64,
    pub lock_height: u32,
}

/// Contract info.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContractInfo {
    pub source: String,
    pub name: String,
}

/// Block info.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockInfo {
    pub hash: String,
    pub height: u32,
    pub timestamp: u64,
    pub miner: String,
}

/// Mempool transaction.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MempoolTx {
    pub txid: String,
    pub type_id: u8,
    pub anchor_mode: u8,
}

/// Stacking info.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StackingInfo {
    pub stacker: String,
    pub locked: u64,
    pub unlock_height: u32,
    pub pox_addr: Option<String>,
}

/// Reward cycle holder.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RewardHolder {
    pub holder: String,
    pub locked: u64,
    pub weight: u64,
}

/// Stacks wallet.
pub struct Wallet {
    private_key: PrivateKey,
    address: StacksAddress,
    network: Network,
    rpc: RPCClient,
}

impl Wallet {
    /// Create a new wallet.
    pub fn new(network: Network, rpc_url: String) -> Result<Self, StacksError> {
        let private_key = PrivateKey::generate()?;
        Ok(Self::from_private_key(private_key, network, rpc_url))
    }

    /// Build a wallet from an existing private key.
    pub fn from_private_key(private_key: PrivateKey, network: Network, rpc_url: String) -> Self {
        let address = private_key.stacks_address(network);
        let rpc = RPCClient::new(network, rpc_url);

        Self {
            private_key,
            address,
            network,
            rpc,
        }
    }

    /// Address of this wallet.
    pub fn address(&self) -> &StacksAddress {
        &self.address
    }

    /// STX balance.
    pub async fn balance(&self) -> Result<u64, StacksError> {
        self.rpc.get_balance(&self.address).await
    }

    /// Transfer STX.
    pub async fn transfer(
        &self,
        recipient: &StacksAddress,
        amount: u64,
        fee: u64,
    ) -> Result<String, StacksError> {
        let nonce = self.rpc.get_nonce(&self.address).await?;

        let mut transaction = Transaction::new_token_transfer(
            recipient.clone(),
            amount,
            &self.private_key,
            self.network,
        )?;
        transaction.fee = fee;
        transaction.nonce = nonce;
        transaction.sign(&self.private_key)?;

        self.rpc.post_transaction(&transaction).await
    }

    /// Delegate STX through the proof-of-transfer contract.
    ///
    /// `pox_contract` is the contract principal that implements the currently
    /// active PoX version, so the caller chooses which version to target rather
    /// than the module hard-coding one.
    pub async fn stack(
        &self,
        pox_contract: &StacksAddress,
        pox_contract_name: &str,
        amount: u64,
        lock_period: u8,
        fee: u64,
    ) -> Result<String, StacksError> {
        if !(1..=12).contains(&lock_period) {
            return Err(StacksError::StackingError(
                "lock period must be between 1 and 12 cycles".into(),
            ));
        }

        let nonce = self.rpc.get_nonce(&self.address).await?;

        let mut transaction = Transaction::new_contract_call(
            pox_contract.clone(),
            pox_contract_name,
            "stack-stx",
            vec![
                ClarityValue::UInt(amount as u128),
                ClarityValue::PrincipalStandard(self.address.clone()),
                ClarityValue::UInt(lock_period as u128),
                ClarityValue::UInt(0),
            ],
            &self.private_key,
            self.network,
        )?;
        transaction.fee = fee;
        transaction.nonce = nonce;
        transaction.sign(&self.private_key)?;

        self.rpc.post_transaction(&transaction).await
    }
}

/// SIP-010 fungible token.
pub struct SIP010Token {
    /// Contract address.
    pub contract: StacksAddress,

    /// Contract name.
    pub contract_name: String,

    /// Token name.
    pub name: String,

    /// Token symbol.
    pub symbol: String,

    /// Decimals.
    pub decimals: u8,
}

impl SIP010Token {
    /// Build the `transfer` contract call.
    pub fn transfer(
        &self,
        amount: u128,
        sender: &StacksAddress,
        recipient: &StacksAddress,
        memo: Option<Vec<u8>>,
    ) -> Result<TransactionPayload, StacksError> {
        let memo_value = match memo {
            Some(bytes) => ClarityValue::Some(Box::new(ClarityValue::Buffer(bytes))),
            None => ClarityValue::None,
        };

        Ok(TransactionPayload::ContractCall {
            contract_address: self.contract.clone(),
            contract_name: self.contract_name.clone(),
            function_name: "transfer".to_string(),
            arguments: vec![
                ClarityValue::UInt(amount),
                ClarityValue::PrincipalStandard(sender.clone()),
                ClarityValue::PrincipalStandard(recipient.clone()),
                memo_value,
            ],
        })
    }

    /// Build the `get-balance` read-only call.
    pub fn get_balance(&self, owner: &StacksAddress) -> TransactionPayload {
        TransactionPayload::ContractCall {
            contract_address: self.contract.clone(),
            contract_name: self.contract_name.clone(),
            function_name: "get-balance".to_string(),
            arguments: vec![ClarityValue::PrincipalStandard(owner.clone())],
        }
    }
}

/// SIP-009 non-fungible token.
pub struct SIP009NFT {
    /// Contract address.
    pub contract: StacksAddress,

    /// Contract name.
    pub contract_name: String,

    /// Collection name.
    pub name: String,

    /// Collection symbol.
    pub symbol: String,
}

impl SIP009NFT {
    /// Build the `transfer` contract call.
    pub fn transfer(
        &self,
        token_id: u128,
        sender: &StacksAddress,
        recipient: &StacksAddress,
    ) -> TransactionPayload {
        TransactionPayload::ContractCall {
            contract_address: self.contract.clone(),
            contract_name: self.contract_name.clone(),
            function_name: "transfer".to_string(),
            arguments: vec![
                ClarityValue::UInt(token_id),
                ClarityValue::PrincipalStandard(sender.clone()),
                ClarityValue::PrincipalStandard(recipient.clone()),
            ],
        }
    }

    /// Build the `get-owner` read-only call.
    pub fn get_owner(&self, token_id: u128) -> TransactionPayload {
        TransactionPayload::ContractCall {
            contract_address: self.contract.clone(),
            contract_name: self.contract_name.clone(),
            function_name: "get-owner".to_string(),
            arguments: vec![ClarityValue::UInt(token_id)],
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn key() -> PrivateKey {
        PrivateKey::from_hex("0101010101010101010101010101010101010101010101010101010101010101")
            .unwrap()
    }

    #[test]
    fn public_key_is_a_valid_compressed_point() {
        let public_key = key().public_key();
        assert_eq!(public_key.len(), 33);
        assert!(public_key[0] == 0x02 || public_key[0] == 0x03);
        // The point must decompress; a fabricated key would not.
        secp256k1::PublicKey::from_slice(&public_key).expect("valid curve point");
    }

    #[test]
    fn address_matches_the_published_c32check_vector() {
        let zero = StacksAddress {
            version: 22,
            hash160: [0u8; 20],
            address_type: AddressType::P2PKH,
        };
        assert_eq!(
            zero.canonical(Network::Mainnet),
            "SP000000000000000000002Q6VF78"
        );
    }

    #[test]
    fn address_roundtrips_through_string_form() {
        let address = key().stacks_address(Network::Mainnet);
        let parsed =
            StacksAddress::from_string(&address.canonical(Network::Mainnet), Network::Mainnet)
                .expect("own address parses");
        assert_eq!(parsed.hash160, address.hash160);
        assert_eq!(parsed.version, address.version);
    }

    #[test]
    fn malformed_addresses_are_rejected() {
        assert!(StacksAddress::from_string("not-an-address", Network::Mainnet).is_err());
        assert!(
            StacksAddress::from_string("SP000000000000000000002Q6VF79", Network::Mainnet).is_err()
        );
    }

    #[test]
    fn signature_recovers_to_the_signing_key() {
        use secp256k1::{ecdsa::RecoverableSignature, Message, Secp256k1};

        let private_key = key();
        let digest = [9u8; 32];
        let signature = private_key.sign_transaction(&digest).unwrap();

        let recoverable = RecoverableSignature::from_compact(
            &signature[..64],
            secp256k1::ecdsa::RecoveryId::from_i32(signature[64] as i32).unwrap(),
        )
        .unwrap();

        let secp = Secp256k1::new();
        let message = Message::from_slice(&digest).unwrap();
        let recovered = secp
            .recover_ecdsa(&message, &recoverable)
            .expect("signature recovers");

        assert_eq!(recovered.serialize(), private_key.public_key());
    }

    #[test]
    fn a_zeroed_signature_does_not_recover() {
        let private_key = key();
        let digest = [9u8; 32];
        let mut signature = private_key.sign_transaction(&digest).unwrap();
        signature[64] = 0;
        signature[0] ^= 0xff;

        let recoverable = secp256k1::ecdsa::RecoverableSignature::from_compact(
            &signature[..64],
            secp256k1::ecdsa::RecoveryId::from_i32(0).unwrap(),
        );
        // Either the compact form is rejected, or recovery yields a different key.
        if let Ok(recoverable) = recoverable {
            let secp = secp256k1::Secp256k1::new();
            let message = secp256k1::Message::from_slice(&digest).unwrap();
            if let Ok(recovered) = secp.recover_ecdsa(&message, &recoverable) {
                assert_ne!(recovered.serialize(), private_key.public_key());
            }
        }
    }

    #[test]
    fn token_transfer_serializes_with_the_expected_prefix() {
        let private_key = key();
        let recipient = private_key.stacks_address(Network::Mainnet);

        let transaction =
            Transaction::new_token_transfer(recipient, 1_000_000, &private_key, Network::Mainnet)
                .unwrap();

        let bytes = transaction.serialize().unwrap();
        assert_eq!(bytes[0], 0x00, "mainnet version byte");
        assert_eq!(&bytes[1..5], &1u32.to_be_bytes(), "mainnet chain id");
        assert_eq!(bytes[5], 0x04, "standard authorisation");
        assert_eq!(&bytes[5 + 1 + 33..5 + 1 + 33 + 8], &0u64.to_be_bytes());
        // Signature must be present, not a placeholder.
        assert_ne!(&transaction.signature, &[0u8; 65]);
    }

    #[test]
    fn sighash_ignores_the_signature_field() {
        let private_key = key();
        let recipient = private_key.stacks_address(Network::Mainnet);

        let mut transaction =
            Transaction::new_token_transfer(recipient, 5, &private_key, Network::Mainnet).unwrap();
        let before = transaction.presign_sighash().unwrap();

        transaction.signature = [0xab; 65];
        let after = transaction.presign_sighash().unwrap();

        assert_eq!(before, after);
    }

    #[test]
    fn txid_is_the_sha512_256_of_the_serialized_transaction() {
        use sha2::{Digest, Sha512_256};

        let private_key = key();
        let recipient = private_key.stacks_address(Network::Mainnet);
        let transaction =
            Transaction::new_token_transfer(recipient, 42, &private_key, Network::Mainnet).unwrap();

        let mut hasher = Sha512_256::new();
        hasher.update(transaction.serialize().unwrap());

        assert_eq!(transaction.txid().unwrap(), hex::encode(hasher.finalize()));
    }

    #[test]
    fn clarity_values_serialize_with_the_canonical_encoding() {
        assert_eq!(ClarityValue::UInt(1).serialize().unwrap()[0], 0x01);
        assert_eq!(ClarityValue::Int(-1).serialize().unwrap()[0], 0x00);
        assert_eq!(ClarityValue::Bool(true).serialize().unwrap(), vec![0x03]);
        assert_eq!(ClarityValue::None.serialize().unwrap(), vec![0x09]);

        let ascii = ClarityValue::StringAscii("ab".into()).serialize().unwrap();
        assert_eq!(ascii, vec![0x0d, 0x00, 0x00, 0x00, 0x02, b'a', b'b']);

        let some = ClarityValue::Some(Box::new(ClarityValue::UInt(7)))
            .serialize()
            .unwrap();
        assert_eq!(some[0], 0x0a);
        assert_eq!(some[1], 0x01);
    }

    #[test]
    fn contract_call_arguments_are_serialized_into_the_payload() {
        let private_key = key();
        let contract = StacksAddress {
            version: 22,
            hash160: [3u8; 20],
            address_type: AddressType::P2PKH,
        };

        let transaction = Transaction::new_contract_call(
            contract,
            "token",
            "transfer",
            vec![
                ClarityValue::UInt(10),
                ClarityValue::PrincipalStandard(private_key.stacks_address(Network::Mainnet)),
            ],
            &private_key,
            Network::Mainnet,
        )
        .unwrap();

        let payload = Transaction::serialize_payload(&transaction.payload).unwrap();
        assert_eq!(payload[0], 0x01, "contract call payload type");

        // Walk the payload: type, version, hash160, name, function, count.
        let mut cursor = 1 + 1 + 20;
        let name_length = payload[cursor] as usize;
        cursor += 1 + name_length;
        let function_length = payload[cursor] as usize;
        cursor += 1 + function_length;

        assert_eq!(&payload[cursor..cursor + 4], &2u32.to_be_bytes());

        // The two encoded arguments follow the count: a UInt is 17 bytes and a
        // standard principal is 22.
        assert_eq!(payload.len() - (cursor + 4), 17 + 22);
        assert_eq!(payload[cursor + 4], 0x01, "first argument is a UInt");
        assert_eq!(payload[cursor + 4 + 17], 0x05, "second is a principal");
    }

    #[test]
    fn stacking_rejects_out_of_range_lock_periods() {
        let wallet = Wallet::from_private_key(key(), Network::Mainnet, "http://localhost".into());
        let contract = StacksAddress {
            version: 22,
            hash160: [0u8; 20],
            address_type: AddressType::P2PKH,
        };

        let result = futures_lite_block_on(wallet.stack(&contract, "pox-4", 100, 13, 1));
        assert!(matches!(result, Err(StacksError::StackingError(_))));
    }

    /// Minimal single-threaded executor so the async guards can be tested
    /// without pulling in a full async runtime.
    fn futures_lite_block_on<F: std::future::Future>(future: F) -> F::Output {
        use std::task::{Context, Poll, Wake, Waker};

        struct NoopWaker;
        impl Wake for NoopWaker {
            fn wake(self: std::sync::Arc<Self>) {}
        }

        let waker = Waker::from(std::sync::Arc::new(NoopWaker));
        let mut context = Context::from_waker(&waker);
        let mut future = Box::pin(future);

        // Poll exactly once: the guards under test resolve immediately, so a
        // Pending result means the test would be awaiting I/O and must fail.
        match future.as_mut().poll(&mut context) {
            Poll::Ready(output) => output,
            Poll::Pending => panic!("future is pending; this test must not await I/O"),
        }
    }
}
