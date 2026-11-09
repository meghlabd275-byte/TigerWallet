//! TigerWallet Bitcoin Service
//!
//! Bitcoin protocol support built on the reference encodings:
//! - Addresses: base58check (P2PKH/P2SH) and bech32/bech32m (P2WPKH/P2WSH/P2TR)
//! - Transaction serialization, including SegWit, and BIP-143 sighash
//! - PSBT, following BIP-174
//! - Stacks (STX) addresses, following c32check
//! - Lightning BOLT-11 invoice decoding
//! - Ordinals / BRC-20 inscription envelopes
//! - Bitcoin Core JSON-RPC for anything that needs the network

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum BitcoinError {
    #[error("RPC error: {0}")]
    RPCError(String),

    #[error("Invalid address: {0}")]
    InvalidAddress(String),

    #[error("Invalid transaction: {0}")]
    InvalidTransaction(String),

    #[error("Signing error: {0}")]
    SigningError(String),

    #[error("Ordinals error: {0}")]
    OrdinalsError(String),

    #[error("Lightning error: {0}")]
    LightningError(String),

    #[error("Network error: {0}")]
    Network(String),

    #[error("Parse error: {0}")]
    Parse(String),

    #[error("Unsupported: {0}")]
    Unsupported(String),
}

pub type Result<T> = std::result::Result<T, BitcoinError>;

// ============================================================================
// Bitcoin Network Configuration
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Network {
    Mainnet,
    Testnet,
    Signet,
    Regtest,
}

impl Network {
    /// Base58 version byte for a P2PKH address.
    pub fn p2pkh_version(self) -> u8 {
        match self {
            Network::Mainnet => 0x00,
            _ => 0x6f,
        }
    }

    /// Base58 version byte for a P2SH address.
    pub fn p2sh_version(self) -> u8 {
        match self {
            Network::Mainnet => 0x05,
            _ => 0xc4,
        }
    }

    /// Bech32 human-readable part.
    pub fn bech32_hrp(self) -> &'static str {
        match self {
            Network::Mainnet => "bc",
            Network::Testnet => "tb",
            Network::Signet => "tb",
            Network::Regtest => "bcrt",
        }
    }
}

// ============================================================================
// Hash Functions
// ============================================================================

fn sha256_hash(data: &[u8]) -> Vec<u8> {
    use sha2::{Digest, Sha256};
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

fn ripemd160_hash(data: &[u8]) -> Vec<u8> {
    use ripemd::{Digest, Ripemd160};
    let mut hasher = Ripemd160::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

fn double_sha256(data: &[u8]) -> Vec<u8> {
    sha256_hash(&sha256_hash(data))
}

/// Hash160, as used for P2PKH and P2SH.
pub fn hash160(data: &[u8]) -> Vec<u8> {
    ripemd160_hash(&sha256_hash(data))
}

// ============================================================================
// Bech32 / Bech32m (BIP-173, BIP-350)
// ============================================================================

mod bech32_impl {
    const CHARSET: &[u8; 32] = b"qpzry9x8gf2tvdw0s3jn54khce6mua7l";
    const GENERATOR: [u32; 5] = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

    /// Checksum constant for bech32 (BIP-173).
    const BECH32_CONST: u32 = 1;
    /// Checksum constant for bech32m (BIP-350).
    const BECH32M_CONST: u32 = 0x2bc830a3;

    #[derive(Debug, Clone, Copy, PartialEq, Eq)]
    pub enum Variant {
        Bech32,
        Bech32m,
    }

    impl Variant {
        fn constant(self) -> u32 {
            match self {
                Variant::Bech32 => BECH32_CONST,
                Variant::Bech32m => BECH32M_CONST,
            }
        }
    }

    fn polymod(values: &[u8]) -> u32 {
        let mut checksum: u32 = 1;
        for value in values {
            let top = checksum >> 25;
            checksum = ((checksum & 0x1ffffff) << 5) ^ u32::from(*value);

            for (index, generator) in GENERATOR.iter().enumerate() {
                if (top >> index) & 1 == 1 {
                    checksum ^= generator;
                }
            }
        }
        checksum
    }

    fn hrp_expand(hrp: &str) -> Vec<u8> {
        let bytes = hrp.as_bytes();
        let mut expanded = Vec::with_capacity(bytes.len() * 2 + 1);

        expanded.extend(bytes.iter().map(|byte| byte >> 5));
        expanded.push(0);
        expanded.extend(bytes.iter().map(|byte| byte & 0x1f));
        expanded
    }

    fn create_checksum(hrp: &str, data: &[u8], variant: Variant) -> Vec<u8> {
        let mut values = hrp_expand(hrp);
        values.extend_from_slice(data);
        values.extend_from_slice(&[0, 0, 0, 0, 0, 0]);

        let polymod = polymod(&values) ^ variant.constant();
        (0..6)
            .map(|index| ((polymod >> (5 * (5 - index))) & 0x1f) as u8)
            .collect()
    }

    fn verify_checksum(hrp: &str, data: &[u8]) -> Option<Variant> {
        let mut values = hrp_expand(hrp);
        values.extend_from_slice(data);

        match polymod(&values) {
            BECH32_CONST => Some(Variant::Bech32),
            BECH32M_CONST => Some(Variant::Bech32m),
            _ => None,
        }
    }

    /// Regroup bits, `from` bits per element to `to` bits per element.
    pub fn convert_bits(data: &[u8], from: u32, to: u32, pad: bool) -> Option<Vec<u8>> {
        let mut accumulator: u32 = 0;
        let mut bits: u32 = 0;
        let mut out = Vec::new();
        let max_value: u32 = (1 << to) - 1;
        let max_accumulator: u32 = (1 << (from + to - 1)) - 1;

        for value in data {
            let value = u32::from(*value);
            if value >> from != 0 {
                return None;
            }

            accumulator = ((accumulator << from) | value) & max_accumulator;
            bits += from;

            while bits >= to {
                bits -= to;
                out.push(((accumulator >> bits) & max_value) as u8);
            }
        }

        if pad {
            if bits > 0 {
                out.push(((accumulator << (to - bits)) & max_value) as u8);
            }
        } else if bits >= from || ((accumulator << (to - bits)) & max_value) != 0 {
            // Non-zero padding means the encoding was not canonical.
            return None;
        }

        Some(out)
    }

    pub fn encode(hrp: &str, data: &[u8], variant: Variant) -> String {
        let mut result = String::with_capacity(hrp.len() + 1 + data.len() + 6);
        result.push_str(hrp);
        result.push('1');

        for value in data {
            result.push(CHARSET[*value as usize] as char);
        }

        for value in create_checksum(hrp, data, variant) {
            result.push(CHARSET[value as usize] as char);
        }

        result
    }

    /// Decode a bech32/bech32m string into its human-readable part, data and
    /// variant.
    ///
    /// BIP-173 caps a segwit address at 90 characters. BOLT-11 invoices are
    /// much longer and use the same encoding without that cap, so
    /// `decode_unbounded` is provided for them.
    pub fn decode(input: &str) -> Option<(String, Vec<u8>, Variant)> {
        decode_with_limit(input, Some(90))
    }

    /// Decode without the segwit address length limit.
    pub fn decode_unbounded(input: &str) -> Option<(String, Vec<u8>, Variant)> {
        decode_with_limit(input, None)
    }

    fn decode_with_limit(
        input: &str,
        max_len: Option<usize>,
    ) -> Option<(String, Vec<u8>, Variant)> {
        // Mixed case is invalid, and every character must be printable ASCII.
        let has_lower = input.chars().any(|c| c.is_ascii_lowercase());
        let has_upper = input.chars().any(|c| c.is_ascii_uppercase());
        if has_lower && has_upper {
            return None;
        }

        if max_len.map(|limit| input.len() > limit).unwrap_or(false)
            || input.chars().any(|c| !(33..=126).contains(&(c as u32)))
        {
            return None;
        }

        let lowered = input.to_ascii_lowercase();
        let separator = lowered.rfind('1')?;

        // The human-readable part needs at least one character, and the data
        // part at least the six checksum characters.
        if separator == 0 || lowered.len() - separator - 1 < 6 {
            return None;
        }

        let hrp = lowered[..separator].to_string();
        let data: Option<Vec<u8>> = lowered[separator + 1..]
            .bytes()
            .map(|byte| CHARSET.iter().position(|c| *c == byte).map(|index| index as u8))
            .collect();
        let data = data?;

        let variant = verify_checksum(&hrp, &data)?;

        Some((hrp, data[..data.len() - 6].to_vec(), variant))
    }
}

pub use bech32_impl::Variant as Bech32Variant;

// ============================================================================
// Base58Check
// ============================================================================

fn base58check_encode(version: u8, payload: &[u8]) -> String {
    let mut data = Vec::with_capacity(payload.len() + 5);
    data.push(version);
    data.extend_from_slice(payload);

    let checksum = double_sha256(&data);
    data.extend_from_slice(&checksum[..4]);

    bs58::encode(&data).into_string()
}

fn base58check_decode(address: &str) -> Result<(u8, Vec<u8>)> {
    let decoded = bs58::decode(address)
        .into_vec()
        .map_err(|e| BitcoinError::InvalidAddress(e.to_string()))?;

    // Version byte, payload, and four checksum bytes.
    if decoded.len() < 5 {
        return Err(BitcoinError::InvalidAddress(
            "address is too short".to_string(),
        ));
    }

    let (body, checksum) = decoded.split_at(decoded.len() - 4);
    if double_sha256(body)[..4] != *checksum {
        return Err(BitcoinError::InvalidAddress(
            "checksum mismatch".to_string(),
        ));
    }

    Ok((body[0], body[1..].to_vec()))
}

// ============================================================================
// Bitcoin Address Types
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AddressType {
    /// Pay to Public Key Hash
    P2PKH,
    /// Pay to Script Hash
    P2SH,
    /// Pay to Witness Public Key Hash
    P2WPKH,
    /// Pay to Witness Script Hash
    P2WSH,
    /// Pay to Taproot
    P2TR,
}

impl AddressType {
    /// Witness version, if this is a SegWit address.
    pub fn witness_version(self) -> Option<u8> {
        match self {
            AddressType::P2WPKH | AddressType::P2WSH => Some(0),
            AddressType::P2TR => Some(1),
            _ => None,
        }
    }
}

/// A Bitcoin address together with the script it locks.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BitcoinAddress {
    pub network: Network,
    pub address_type: AddressType,
    /// The scriptPubKey that pays this address.
    pub script_pub_key: Vec<u8>,
    /// The 20- or 32-byte hash the script commits to.
    pub hash: Vec<u8>,
}

impl BitcoinAddress {
    /// Decode an address of any supported type.
    ///
    /// The network is not taken on trust: it is derived from the version byte
    /// or the bech32 human-readable part, and must match the network the caller
    /// expects.
    pub fn decode(address: &str, network: Network) -> Result<Self> {
        let trimmed = address.trim();
        if trimmed.is_empty() {
            return Err(BitcoinError::InvalidAddress("address is empty".to_string()));
        }

        // SegWit and Taproot addresses are bech32 or bech32m.
        if let Some((hrp, data, variant)) = bech32_impl::decode(trimmed) {
            return Self::from_witness(&hrp, &data, variant, network);
        }

        let (version, payload) = base58check_decode(trimmed)?;

        let address_type = if version == network.p2pkh_version() {
            AddressType::P2PKH
        } else if version == network.p2sh_version() {
            AddressType::P2SH
        } else {
            return Err(BitcoinError::InvalidAddress(format!(
                "version byte 0x{version:02x} does not belong to {network:?}"
            )));
        };

        if payload.len() != 20 {
            return Err(BitcoinError::InvalidAddress(
                "a P2PKH/P2SH payload must be 20 bytes".to_string(),
            ));
        }

        let script_pub_key = match address_type {
            AddressType::P2PKH => {
                let mut script = vec![0x76, 0xa9, 0x14];
                script.extend_from_slice(&payload);
                script.extend_from_slice(&[0x88, 0xac]);
                script
            }
            _ => {
                let mut script = vec![0xa9, 0x14];
                script.extend_from_slice(&payload);
                script.push(0x87);
                script
            }
        };

        Ok(BitcoinAddress {
            network,
            address_type,
            script_pub_key,
            hash: payload,
        })
    }

    /// Build an address from a decoded witness program.
    fn from_witness(
        hrp: &str,
        data: &[u8],
        variant: Bech32Variant,
        network: Network,
    ) -> Result<Self> {
        if hrp != network.bech32_hrp() {
            return Err(BitcoinError::InvalidAddress(format!(
                "human-readable part {hrp:?} does not belong to {network:?}"
            )));
        }

        let (version, program_groups) = data
            .split_first()
            .ok_or_else(|| BitcoinError::InvalidAddress("empty witness program".to_string()))?;
        let version = *version;

        // The version is one 5-bit group; the program is the remaining groups
        // regrouped into bytes.
        let program = bech32_impl::convert_bits(program_groups, 5, 8, false).ok_or_else(|| {
            BitcoinError::InvalidAddress("the witness program is not canonically padded".to_string())
        })?;

        if version > 16 {
            return Err(BitcoinError::InvalidAddress(
                "witness version must be 0..=16".to_string(),
            ));
        }

        // BIP-350: version 0 uses bech32, versions 1..=16 use bech32m.
        let expected_variant = if version == 0 {
            Bech32Variant::Bech32
        } else {
            Bech32Variant::Bech32m
        };
        if variant != expected_variant {
            return Err(BitcoinError::InvalidAddress(format!(
                "witness version {version} must use {expected_variant:?}"
            )));
        }

        if !(2..=40).contains(&program.len()) {
            return Err(BitcoinError::InvalidAddress(
                "witness program must be 2..=40 bytes".to_string(),
            ));
        }

        // Version 0 programs are exactly 20 or 32 bytes.
        if version == 0 && program.len() != 20 && program.len() != 32 {
            return Err(BitcoinError::InvalidAddress(
                "a version 0 program must be 20 or 32 bytes".to_string(),
            ));
        }

        let address_type = match (version, program.len()) {
            (0, 20) => AddressType::P2WPKH,
            (0, 32) => AddressType::P2WSH,
            (1, 32) => AddressType::P2TR,
            _ => {
                return Err(BitcoinError::InvalidAddress(format!(
                    "no supported address type for witness version {version} with a {}-byte program",
                    program.len()
                )))
            }
        };

        let script_pub_key = if version == 0 {
            let mut script = vec![0x00, program.len() as u8];
            script.extend_from_slice(&program);
            script
        } else {
            let mut script = vec![0x50 + version, program.len() as u8];
            script.extend_from_slice(&program);
            script
        };

        Ok(BitcoinAddress {
            network,
            address_type,
            script_pub_key,
            hash: program,
        })
    }

    /// Create a P2PKH address from a public key hash.
    pub fn p2pkh(pubkey_hash: &[u8], network: Network) -> Result<Self> {
        if pubkey_hash.len() != 20 {
            return Err(BitcoinError::InvalidAddress(
                "a public key hash must be 20 bytes".to_string(),
            ));
        }

        let mut script = vec![0x76, 0xa9, 0x14];
        script.extend_from_slice(pubkey_hash);
        script.extend_from_slice(&[0x88, 0xac]);

        Ok(BitcoinAddress {
            network,
            address_type: AddressType::P2PKH,
            script_pub_key: script,
            hash: pubkey_hash.to_vec(),
        })
    }

    /// Create a P2PKH address from a compressed public key.
    pub fn p2pkh_from_pubkey(public_key: &[u8], network: Network) -> Result<Self> {
        Self::p2pkh(&hash160(public_key), network)
    }

    /// Create a P2SH address from a script hash.
    pub fn p2sh(script_hash: &[u8], network: Network) -> Result<Self> {
        if script_hash.len() != 20 {
            return Err(BitcoinError::InvalidAddress(
                "a script hash must be 20 bytes".to_string(),
            ));
        }

        let mut script = vec![0xa9, 0x14];
        script.extend_from_slice(script_hash);
        script.push(0x87);

        Ok(BitcoinAddress {
            network,
            address_type: AddressType::P2SH,
            script_pub_key: script,
            hash: script_hash.to_vec(),
        })
    }

    /// Create a P2SH address that wraps a redeem script.
    pub fn p2sh_for_script(redeem_script: &[u8], network: Network) -> Result<Self> {
        Self::p2sh(&hash160(redeem_script), network)
    }

    /// Create a P2WPKH address from a public key hash.
    pub fn p2wpkh(pubkey_hash: &[u8], network: Network) -> Result<Self> {
        if pubkey_hash.len() != 20 {
            return Err(BitcoinError::InvalidAddress(
                "a public key hash must be 20 bytes".to_string(),
            ));
        }

        let mut script = vec![0x00, 0x14];
        script.extend_from_slice(pubkey_hash);

        Ok(BitcoinAddress {
            network,
            address_type: AddressType::P2WPKH,
            script_pub_key: script,
            hash: pubkey_hash.to_vec(),
        })
    }

    /// Create a P2WPKH address from a compressed public key.
    pub fn p2wpkh_from_pubkey(public_key: &[u8], network: Network) -> Result<Self> {
        if public_key.len() != 33 {
            return Err(BitcoinError::InvalidAddress(
                "a compressed public key must be 33 bytes".to_string(),
            ));
        }

        Self::p2wpkh(&hash160(public_key), network)
    }

    /// Create a P2WSH address from a witness script.
    pub fn p2wsh(witness_script: &[u8], network: Network) -> Result<Self> {
        let hash = sha256_hash(witness_script);

        let mut script = vec![0x00, 0x20];
        script.extend_from_slice(&hash);

        Ok(BitcoinAddress {
            network,
            address_type: AddressType::P2WSH,
            script_pub_key: script,
            hash,
        })
    }

    /// Create a P2TR address from a 32-byte x-only output key.
    pub fn p2tr(output_key: &[u8], network: Network) -> Result<Self> {
        if output_key.len() != 32 {
            return Err(BitcoinError::InvalidAddress(
                "a Taproot output key must be 32 bytes".to_string(),
            ));
        }

        let mut script = vec![0x51, 0x20];
        script.extend_from_slice(output_key);

        Ok(BitcoinAddress {
            network,
            address_type: AddressType::P2TR,
            script_pub_key: script,
            hash: output_key.to_vec(),
        })
    }

    /// Encode this address.
    pub fn encode(&self) -> String {
        match self.address_type {
            AddressType::P2PKH => base58check_encode(self.network.p2pkh_version(), &self.hash),
            AddressType::P2SH => base58check_encode(self.network.p2sh_version(), &self.hash),
            AddressType::P2WPKH | AddressType::P2WSH | AddressType::P2TR => {
                let version = self.address_type.witness_version().unwrap_or(0);
                let variant = if version == 0 {
                    Bech32Variant::Bech32
                } else {
                    Bech32Variant::Bech32m
                };

                // The witness version is one 5-bit group of its own; the
                // program is converted from bytes separately. Packing them
                // together would shift the version into the program.
                let mut data = Vec::with_capacity(self.hash.len() + 1);
                data.push(version);
                data.extend_from_slice(
                    &bech32_impl::convert_bits(&self.hash, 8, 5, true)
                        .expect("8-bit data always converts into 5-bit groups"),
                );

                bech32_impl::encode(self.network.bech32_hrp(), &data, variant)
            }
        }
    }

    /// Check that a string is a valid address for a network.
    pub fn validate(address: &str, network: Network) -> bool {
        Self::decode(address, network).is_ok()
    }
}

impl std::fmt::Display for BitcoinAddress {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(&self.encode())
    }
}

// ============================================================================
// Compact integers
// ============================================================================

/// Bitcoin's `CompactSize` (varint) encoding.
pub fn varint(n: u64) -> Vec<u8> {
    match n {
        0..=0xfc => vec![n as u8],
        0xfd..=0xffff => {
            let mut out = vec![0xfd];
            out.extend_from_slice(&(n as u16).to_le_bytes());
            out
        }
        0x1_0000..=0xffff_ffff => {
            let mut out = vec![0xfe];
            out.extend_from_slice(&(n as u32).to_le_bytes());
            out
        }
        _ => {
            let mut out = vec![0xff];
            out.extend_from_slice(&n.to_le_bytes());
            out
        }
    }
}

/// Read a `CompactSize` value, returning it and advancing the cursor.
fn read_varint(data: &[u8], cursor: &mut usize) -> Result<u64> {
    let first = *data
        .get(*cursor)
        .ok_or_else(|| BitcoinError::Parse("truncated compact size".to_string()))?;
    *cursor += 1;

    fn take(data: &[u8], cursor: &mut usize, count: usize) -> Result<Vec<u8>> {
        let slice = data
            .get(*cursor..*cursor + count)
            .ok_or_else(|| BitcoinError::Parse("truncated compact size".to_string()))?;
        *cursor += count;
        Ok(slice.to_vec())
    }

    let value = match first {
        0xfd => u64::from(u16::from_le_bytes(
            take(data, cursor, 2)?.try_into().unwrap(),
        )),
        0xfe => u64::from(u32::from_le_bytes(
            take(data, cursor, 4)?.try_into().unwrap(),
        )),
        0xff => u64::from_le_bytes(take(data, cursor, 8)?.try_into().unwrap()),
        other => u64::from(other),
    };

    // Values that could have used a shorter form are non-canonical, and
    // accepting them lets two encodings share one transaction id.
    let canonical = match value {
        0..=0xfc => first != 0xfd && first != 0xfe && first != 0xff,
        0xfd..=0xffff => first == 0xfd,
        0x1_0000..=0xffff_ffff => first == 0xfe,
        _ => first == 0xff,
    };
    if !canonical {
        return Err(BitcoinError::Parse(
            "non-canonical compact size".to_string(),
        ));
    }

    Ok(value)
}

// ============================================================================
// Transaction Building
// ============================================================================

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OutPoint {
    /// Transaction id in internal byte order (little-endian).
    pub txid: [u8; 32],
    pub vout: u32,
}

impl OutPoint {
    /// Display order is the reverse of internal order.
    pub fn txid_hex(&self) -> String {
        let mut reversed = self.txid;
        reversed.reverse();
        hex::encode(reversed)
    }

    fn serialize(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(36);
        out.extend_from_slice(&self.txid);
        out.extend_from_slice(&self.vout.to_le_bytes());
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TxInput {
    pub previous_output: OutPoint,
    pub script_sig: Vec<u8>,
    pub sequence: u32,
    /// Witness stack for this input. Empty for legacy inputs.
    pub witness: Vec<Vec<u8>>,
}

impl TxInput {
    fn serialize(&self) -> Vec<u8> {
        let mut out = self.previous_output.serialize();
        out.extend_from_slice(&varint(self.script_sig.len() as u64));
        out.extend_from_slice(&self.script_sig);
        out.extend_from_slice(&self.sequence.to_le_bytes());
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TxOutput {
    pub value: u64,
    pub script_pub_key: Vec<u8>,
}

impl TxOutput {
    fn serialize(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(self.script_pub_key.len() + 9);
        out.extend_from_slice(&self.value.to_le_bytes());
        out.extend_from_slice(&varint(self.script_pub_key.len() as u64));
        out.extend_from_slice(&self.script_pub_key);
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BitcoinTransaction {
    pub version: i32,
    pub inputs: Vec<TxInput>,
    pub outputs: Vec<TxOutput>,
    pub lock_time: u32,
}

impl Default for BitcoinTransaction {
    fn default() -> Self {
        Self::new()
    }
}

impl BitcoinTransaction {
    pub fn new() -> Self {
        BitcoinTransaction {
            version: 2,
            inputs: Vec::new(),
            outputs: Vec::new(),
            lock_time: 0,
        }
    }

    /// Add an input spending `txid:vout`, with no scriptSig and a final
    /// sequence number.
    pub fn add_input(&mut self, txid: [u8; 32], vout: u32, script_sig: Vec<u8>) {
        self.inputs.push(TxInput {
            previous_output: OutPoint { txid, vout },
            script_sig,
            sequence: 0xffff_ffff,
            witness: Vec::new(),
        });
    }

    /// Add an input from a displayed transaction id.
    pub fn add_input_hex(&mut self, txid: &str, vout: u32) -> Result<()> {
        let mut bytes = hex::decode(txid.trim_start_matches("0x"))
            .map_err(|e| BitcoinError::InvalidTransaction(e.to_string()))?;

        if bytes.len() != 32 {
            return Err(BitcoinError::InvalidTransaction(
                "a transaction id must be 32 bytes".to_string(),
            ));
        }

        bytes.reverse();
        let mut txid_bytes = [0u8; 32];
        txid_bytes.copy_from_slice(&bytes);
        self.add_input(txid_bytes, vout, Vec::new());
        Ok(())
    }

    pub fn add_output(&mut self, value: u64, script_pub_key: Vec<u8>) {
        self.outputs.push(TxOutput {
            value,
            script_pub_key,
        });
    }

    /// Whether any input carries witness data.
    pub fn has_witness(&self) -> bool {
        self.inputs.iter().any(|input| !input.witness.is_empty())
    }

    /// Serialize without witness data. This is what a transaction id commits to.
    pub fn serialize_legacy(&self) -> Vec<u8> {
        let mut result = Vec::new();

        result.extend_from_slice(&self.version.to_le_bytes());
        result.extend_from_slice(&varint(self.inputs.len() as u64));
        for input in &self.inputs {
            result.extend_from_slice(&input.serialize());
        }

        result.extend_from_slice(&varint(self.outputs.len() as u64));
        for output in &self.outputs {
            result.extend_from_slice(&output.serialize());
        }

        result.extend_from_slice(&self.lock_time.to_le_bytes());
        result
    }

    /// Serialize in the form that is broadcast, including witness data when
    /// present.
    pub fn serialize(&self) -> Vec<u8> {
        if !self.has_witness() {
            return self.serialize_legacy();
        }

        let mut result = Vec::new();

        result.extend_from_slice(&self.version.to_le_bytes());
        // SegWit marker and flag.
        result.extend_from_slice(&[0x00, 0x01]);

        result.extend_from_slice(&varint(self.inputs.len() as u64));
        for input in &self.inputs {
            result.extend_from_slice(&input.serialize());
        }

        result.extend_from_slice(&varint(self.outputs.len() as u64));
        for output in &self.outputs {
            result.extend_from_slice(&output.serialize());
        }

        for input in &self.inputs {
            result.extend_from_slice(&varint(input.witness.len() as u64));
            for item in &input.witness {
                result.extend_from_slice(&varint(item.len() as u64));
                result.extend_from_slice(item);
            }
        }

        result.extend_from_slice(&self.lock_time.to_le_bytes());
        result
    }

    /// Transaction id, in display order.
    pub fn txid(&self) -> String {
        let mut hash = double_sha256(&self.serialize_legacy());
        hash.reverse();
        hex::encode(hash)
    }

    /// Transaction id in internal byte order.
    pub fn txid_bytes(&self) -> [u8; 32] {
        let hash = double_sha256(&self.serialize_legacy());
        let mut txid = [0u8; 32];
        txid.copy_from_slice(&hash);
        txid
    }

    /// Virtual size, which is what fee rates are quoted against.
    pub fn vsize(&self) -> usize {
        let weight = if self.has_witness() {
            let base = self.serialize_legacy().len();
            let total = self.serialize().len();
            base * 3 + total
        } else {
            self.serialize_legacy().len() * 4
        };

        weight.div_ceil(4)
    }

    /// Parse a transaction in either legacy or SegWit form.
    pub fn deserialize(data: &[u8]) -> Result<Self> {
        let mut cursor = 0usize;

        let version = i32::from_le_bytes(
            data.get(0..4)
                .ok_or_else(|| BitcoinError::Parse("truncated version".to_string()))?
                .try_into()
                .unwrap(),
        );
        cursor += 4;

        // A zero input count is the SegWit marker, followed by the flag.
        let mut segwit = false;
        let input_count = read_varint(data, &mut cursor)?;
        let input_count = if input_count == 0 {
            let flag = *data
                .get(cursor)
                .ok_or_else(|| BitcoinError::Parse("missing segwit flag".to_string()))?;
            if flag != 0x01 {
                return Err(BitcoinError::Parse(format!(
                    "unsupported segwit flag 0x{flag:02x}"
                )));
            }
            cursor += 1;
            segwit = true;
            read_varint(data, &mut cursor)?
        } else {
            input_count
        };

        let mut inputs = Vec::with_capacity(input_count as usize);
        for _ in 0..input_count {
            let txid_slice = data
                .get(cursor..cursor + 32)
                .ok_or_else(|| BitcoinError::Parse("truncated input txid".to_string()))?;
            let mut txid = [0u8; 32];
            txid.copy_from_slice(txid_slice);
            cursor += 32;

            let vout = u32::from_le_bytes(
                data.get(cursor..cursor + 4)
                    .ok_or_else(|| BitcoinError::Parse("truncated input vout".to_string()))?
                    .try_into()
                    .unwrap(),
            );
            cursor += 4;

            let script_len = read_varint(data, &mut cursor)? as usize;
            let script_sig = data
                .get(cursor..cursor + script_len)
                .ok_or_else(|| BitcoinError::Parse("truncated scriptSig".to_string()))?
                .to_vec();
            cursor += script_len;

            let sequence = u32::from_le_bytes(
                data.get(cursor..cursor + 4)
                    .ok_or_else(|| BitcoinError::Parse("truncated sequence".to_string()))?
                    .try_into()
                    .unwrap(),
            );
            cursor += 4;

            inputs.push(TxInput {
                previous_output: OutPoint { txid, vout },
                script_sig,
                sequence,
                witness: Vec::new(),
            });
        }

        let output_count = read_varint(data, &mut cursor)?;
        let mut outputs = Vec::with_capacity(output_count as usize);
        for _ in 0..output_count {
            let value = u64::from_le_bytes(
                data.get(cursor..cursor + 8)
                    .ok_or_else(|| BitcoinError::Parse("truncated output value".to_string()))?
                    .try_into()
                    .unwrap(),
            );
            cursor += 8;

            let script_len = read_varint(data, &mut cursor)? as usize;
            let script_pub_key = data
                .get(cursor..cursor + script_len)
                .ok_or_else(|| BitcoinError::Parse("truncated scriptPubKey".to_string()))?
                .to_vec();
            cursor += script_len;

            outputs.push(TxOutput {
                value,
                script_pub_key,
            });
        }

        if segwit {
            for input in inputs.iter_mut() {
                let item_count = read_varint(data, &mut cursor)?;
                for _ in 0..item_count {
                    let item_len = read_varint(data, &mut cursor)? as usize;
                    let item = data
                        .get(cursor..cursor + item_len)
                        .ok_or_else(|| BitcoinError::Parse("truncated witness item".to_string()))?
                        .to_vec();
                    cursor += item_len;
                    input.witness.push(item);
                }
            }
        }

        let lock_time = u32::from_le_bytes(
            data.get(cursor..cursor + 4)
                .ok_or_else(|| BitcoinError::Parse("truncated lock time".to_string()))?
                .try_into()
                .unwrap(),
        );
        cursor += 4;

        if cursor != data.len() {
            return Err(BitcoinError::Parse(
                "trailing bytes after the transaction".to_string(),
            ));
        }

        Ok(BitcoinTransaction {
            version,
            inputs,
            outputs,
            lock_time,
        })
    }
}

// ============================================================================
// BIP-143 sighash
// ============================================================================

/// Signature hash types.
pub mod sighash {
    pub const ALL: u32 = 0x01;
    pub const NONE: u32 = 0x02;
    pub const SINGLE: u32 = 0x03;
    pub const ANYONECANPAY: u32 = 0x80;
}

/// Build the P2WPKH scriptCode for a public key hash.
///
/// This is the script itself, `OP_DUP OP_HASH160 <20 bytes> OP_EQUALVERIFY
/// OP_CHECKSIG`. The sighash serializer prepends its length, so the prefix must
/// not be included here.
pub fn p2wpkh_script_code(pubkey_hash: &[u8]) -> Result<Vec<u8>> {
    if pubkey_hash.len() != 20 {
        return Err(BitcoinError::SigningError(
            "a P2WPKH scriptCode needs a 20-byte public key hash".to_string(),
        ));
    }

    let mut script = vec![0x76, 0xa9, 0x14];
    script.extend_from_slice(pubkey_hash);
    script.extend_from_slice(&[0x88, 0xac]);
    Ok(script)
}

/// Compute the BIP-143 signature hash for a SegWit v0 input.
///
/// `script_code` is the script being satisfied, without a length prefix;
/// `amount` is the value of the output being spent, which the signature commits
/// to. The returned value is the 32-byte digest to sign.
pub fn segwit_v0_sighash(
    tx: &BitcoinTransaction,
    input_index: usize,
    script_code: &[u8],
    amount: u64,
    sighash_type: u32,
) -> Result<[u8; 32]> {
    if input_index >= tx.inputs.len() {
        return Err(BitcoinError::SigningError(format!(
            "input index {input_index} is out of range"
        )));
    }

    let base_type = sighash_type & 0x1f;
    let anyone_can_pay = sighash_type & sighash::ANYONECANPAY != 0;

    if base_type == sighash::SINGLE && input_index >= tx.outputs.len() {
        // BIP-143 makes this combination invalid rather than undefined.
        return Err(BitcoinError::SigningError(
            "SIGHASH_SINGLE without a matching output".to_string(),
        ));
    }

    if base_type != sighash::ALL && base_type != sighash::NONE && base_type != sighash::SINGLE {
        return Err(BitcoinError::SigningError(format!(
            "unsupported sighash type {base_type:#x}"
        )));
    }

    // hashPrevouts: every input's outpoint, unless ANYONECANPAY is set.
    let mut prevouts = Vec::new();
    if !anyone_can_pay {
        for input in &tx.inputs {
            prevouts.extend_from_slice(&input.previous_output.serialize());
        }
    }

    // hashSequence: every input's sequence, but only for a plain ALL.
    let mut sequences = Vec::new();
    if !anyone_can_pay && base_type == sighash::ALL {
        for input in &tx.inputs {
            sequences.extend_from_slice(&input.sequence.to_le_bytes());
        }
    }

    // hashOutputs: all outputs for ALL, only the matching one for SINGLE, and
    // nothing for NONE.
    let mut outputs = Vec::new();
    match base_type {
        sighash::ALL => {
            for output in &tx.outputs {
                outputs.extend_from_slice(&output.serialize());
            }
        }
        sighash::SINGLE => {
            if input_index < tx.outputs.len() {
                outputs.extend_from_slice(&tx.outputs[input_index].serialize());
            }
        }
        _ => {}
    }

    let mut preimage = Vec::with_capacity(156 + script_code.len());
    preimage.extend_from_slice(&tx.version.to_le_bytes());
    preimage.extend_from_slice(&double_sha256(&prevouts));
    preimage.extend_from_slice(&double_sha256(&sequences));
    preimage.extend_from_slice(&tx.inputs[input_index].previous_output.serialize());
    preimage.extend_from_slice(&varint(script_code.len() as u64));
    preimage.extend_from_slice(script_code);
    preimage.extend_from_slice(&amount.to_le_bytes());
    preimage.extend_from_slice(&tx.inputs[input_index].sequence.to_le_bytes());
    preimage.extend_from_slice(&double_sha256(&outputs));
    preimage.extend_from_slice(&tx.lock_time.to_le_bytes());
    preimage.extend_from_slice(&sighash_type.to_le_bytes());

    let digest = double_sha256(&preimage);
    let mut out = [0u8; 32];
    out.copy_from_slice(&digest);
    Ok(out)
}

// ============================================================================
// PSBT (BIP-174)
// ============================================================================

/// One key/value entry in a PSBT map.
type MapEntry = (Vec<u8>, Vec<u8>);

/// A BIP-174 Partially Signed Bitcoin Transaction.
///
/// Maps are kept as raw key/value pairs so unknown proprietary fields survive a
/// round trip, which is what lets a PSBT pass through this library on its way
/// between a coordinator and a hardware signer.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Psbt {
    /// Global key/value pairs, excluding the unsigned transaction.
    pub global: Vec<MapEntry>,
    /// Per-input key/value pairs.
    pub inputs: Vec<Vec<MapEntry>>,
    /// Per-output key/value pairs.
    pub outputs: Vec<Vec<MapEntry>>,
    /// The unsigned transaction, from the `PSBT_GLOBAL_UNSIGNED_TX` field.
    pub unsigned_tx: BitcoinTransaction,
}

impl Psbt {
    pub const MAGIC: [u8; 5] = [0x70, 0x73, 0x62, 0x74, 0xff];

    /// Start a PSBT for an unsigned transaction.
    pub fn new(unsigned_tx: BitcoinTransaction) -> Self {
        let inputs = vec![Vec::new(); unsigned_tx.inputs.len()];
        let outputs = vec![Vec::new(); unsigned_tx.outputs.len()];

        Psbt {
            global: Vec::new(),
            inputs,
            outputs,
            unsigned_tx,
        }
    }

    /// The `PSBT_GLOBAL_VERSION` field, if present.
    pub fn version(&self) -> Option<u32> {
        self.global
            .iter()
            .find(|(key, _)| key.len() == 1 && key[0] == 0xfb)
            .and_then(|(_, value)| {
                if value.len() == 4 {
                    Some(u32::from_le_bytes(value.as_slice().try_into().unwrap()))
                } else {
                    None
                }
            })
    }

    /// Set the witness UTXO an input spends.
    pub fn set_witness_utxo(&mut self, input_index: usize, output: &TxOutput) -> Result<()> {
        let entry = self
            .inputs
            .get_mut(input_index)
            .ok_or_else(|| BitcoinError::InvalidTransaction("input index out of range".into()))?;

        let mut value = Vec::with_capacity(output.script_pub_key.len() + 9);
        value.extend_from_slice(&output.value.to_le_bytes());
        value.extend_from_slice(&varint(output.script_pub_key.len() as u64));
        value.extend_from_slice(&output.script_pub_key);

        entry.push((vec![0x01], value));
        Ok(())
    }

    /// Read the witness UTXO recorded for an input.
    pub fn witness_utxo(&self, input_index: usize) -> Option<TxOutput> {
        let (_, value) = self
            .inputs
            .get(input_index)?
            .iter()
            .find(|(key, _)| key.len() == 1 && key[0] == 0x01)?;

        if value.len() < 9 {
            return None;
        }

        let amount = u64::from_le_bytes(value[..8].try_into().ok()?);
        let mut cursor = 8;
        let script_len = read_varint(value, &mut cursor).ok()? as usize;
        let script = value.get(cursor..cursor + script_len)?.to_vec();

        Some(TxOutput {
            value: amount,
            script_pub_key: script,
        })
    }

    /// Record a partial signature for an input.
    pub fn add_partial_signature(
        &mut self,
        input_index: usize,
        public_key: &[u8],
        signature: &[u8],
    ) -> Result<()> {
        let entry = self
            .inputs
            .get_mut(input_index)
            .ok_or_else(|| BitcoinError::InvalidTransaction("input index out of range".into()))?;

        let mut key = vec![0x02];
        key.extend_from_slice(public_key);
        entry.push((key, signature.to_vec()));
        Ok(())
    }

    /// Partial signatures recorded for an input, keyed by public key.
    pub fn partial_signatures(&self, input_index: usize) -> Vec<(Vec<u8>, Vec<u8>)> {
        self.inputs
            .get(input_index)
            .map(|entries| {
                entries
                    .iter()
                    .filter(|(key, _)| key.len() == 34 && key[0] == 0x02)
                    .map(|(key, value)| (key[1..].to_vec(), value.clone()))
                    .collect()
            })
            .unwrap_or_default()
    }

    /// Whether every input carries a final scriptSig or witness.
    pub fn is_finalized(&self) -> bool {
        if self.inputs.is_empty() {
            return false;
        }

        self.inputs.iter().all(|entries| {
            entries
                .iter()
                .any(|(key, _)| key.len() == 1 && (key[0] == 0x07 || key[0] == 0x08))
        })
    }

    /// Build the final scriptSig or witness for every input that has a partial
    /// signature and a known witness UTXO.
    ///
    /// Only P2WPKH is finalized here: it is the one form whose script is fully
    /// determined by the public key hash, so the result can be checked without
    /// consulting a script interpreter. Anything else is left untouched rather
    /// than guessed at.
    pub fn finalize(&mut self) -> Result<()> {
        for index in 0..self.inputs.len() {
            if self
                .inputs[index]
                .iter()
                .any(|(key, _)| key.len() == 1 && (key[0] == 0x07 || key[0] == 0x08))
            {
                continue;
            }

            let Some(utxo) = self.witness_utxo(index) else {
                continue;
            };

            // P2WPKH locks OP_0 <20-byte hash>.
            if utxo.script_pub_key.len() != 22
                || utxo.script_pub_key[0] != 0x00
                || utxo.script_pub_key[1] != 0x14
            {
                continue;
            }

            let expected_hash = &utxo.script_pub_key[2..22];
            let Some((public_key, signature)) = self.partial_signatures(index).into_iter().next()
            else {
                continue;
            };

            if hash160(&public_key) != expected_hash {
                return Err(BitcoinError::SigningError(format!(
                    "the partial signature for input {index} was made with a key that does not match the output being spent"
                )));
            }

            // Witness stack: <signature> <public key>.
            let mut witness = Vec::new();
            witness.extend_from_slice(&varint(2));
            for item in [&signature, &public_key] {
                witness.extend_from_slice(&varint(item.len() as u64));
                witness.extend_from_slice(item);
            }

            self.inputs[index].push((vec![0x08], witness));
        }

        Ok(())
    }

    /// Apply every finalized input to the unsigned transaction.
    pub fn extract(&self) -> Result<BitcoinTransaction> {
        let mut tx = self.unsigned_tx.clone();

        for (index, entries) in self.inputs.iter().enumerate() {
            let script_sig = entries
                .iter()
                .find(|(key, _)| key.len() == 1 && key[0] == 0x07)
                .map(|(_, value)| value.clone());
            let witness = entries
                .iter()
                .find(|(key, _)| key.len() == 1 && key[0] == 0x08)
                .map(|(_, value)| value.clone());

            if script_sig.is_none() && witness.is_none() {
                return Err(BitcoinError::SigningError(format!(
                    "input {index} has not been finalized"
                )));
            }

            if let Some(script_sig) = script_sig {
                tx.inputs[index].script_sig = script_sig;
            }

            if let Some(witness) = witness {
                tx.inputs[index].witness = parse_witness(&witness)?;
            }
        }

        Ok(tx)
    }

    /// Serialize to the binary PSBT form.
    pub fn serialize(&self) -> Vec<u8> {
        let mut out = Vec::new();
        out.extend_from_slice(&Self::MAGIC);

        // The unsigned transaction is always the first global entry.
        let tx_bytes = self.unsigned_tx.serialize_legacy();
        out.extend_from_slice(&[0x01, 0x00]);
        out.extend_from_slice(&varint(tx_bytes.len() as u64));
        out.extend_from_slice(&tx_bytes);

        for (key, value) in &self.global {
            write_entry(&mut out, key, value);
        }
        out.push(0x00);

        for entries in &self.inputs {
            for (key, value) in entries {
                write_entry(&mut out, key, value);
            }
            out.push(0x00);
        }

        for entries in &self.outputs {
            for (key, value) in entries {
                write_entry(&mut out, key, value);
            }
            out.push(0x00);
        }

        out
    }

    /// Serialize to the base64 form used by wallets and QR codes.
    pub fn to_base64(&self) -> String {
        use base64::Engine;
        base64::engine::general_purpose::STANDARD.encode(self.serialize())
    }

    /// Parse the binary PSBT form.
    pub fn deserialize(data: &[u8]) -> Result<Self> {
        if data.len() < Self::MAGIC.len() || data[..Self::MAGIC.len()] != Self::MAGIC {
            return Err(BitcoinError::Parse("missing PSBT magic".to_string()));
        }

        let mut cursor = Self::MAGIC.len();
        let global_entries = read_map(data, &mut cursor)?;

        // The unsigned transaction must come first, and there must be exactly
        // one.
        let (first_key, first_value) = global_entries
            .first()
            .ok_or_else(|| BitcoinError::Parse("PSBT has no global map".to_string()))?;
        if first_key != &vec![0x00] {
            return Err(BitcoinError::Parse(
                "PSBT_GLOBAL_UNSIGNED_TX must be the first global entry".to_string(),
            ));
        }
        let unsigned_tx = BitcoinTransaction::deserialize(first_value)?;

        let mut psbt = Psbt {
            global: global_entries[1..].to_vec(),
            inputs: Vec::new(),
            outputs: Vec::new(),
            unsigned_tx,
        };

        for _ in 0..psbt.unsigned_tx.inputs.len() {
            psbt.inputs.push(read_map(data, &mut cursor)?);
        }

        for _ in 0..psbt.unsigned_tx.outputs.len() {
            psbt.outputs.push(read_map(data, &mut cursor)?);
        }

        if cursor != data.len() {
            return Err(BitcoinError::Parse(
                "trailing bytes after the PSBT".to_string(),
            ));
        }

        Ok(psbt)
    }

    /// Parse the base64 form.
    pub fn from_base64(encoded: &str) -> Result<Self> {
        use base64::Engine;
        let bytes = base64::engine::general_purpose::STANDARD
            .decode(encoded.trim())
            .map_err(|e| BitcoinError::Parse(e.to_string()))?;
        Self::deserialize(&bytes)
    }
}

fn write_entry(out: &mut Vec<u8>, key: &[u8], value: &[u8]) {
    out.extend_from_slice(&varint(key.len() as u64));
    out.extend_from_slice(key);
    out.extend_from_slice(&varint(value.len() as u64));
    out.extend_from_slice(value);
}

/// Read one map, up to and including its `0x00` separator.
fn read_map(data: &[u8], cursor: &mut usize) -> Result<Vec<MapEntry>> {
    let mut entries = Vec::new();
    let mut seen: std::collections::HashSet<Vec<u8>> = std::collections::HashSet::new();

    loop {
        let key_len = read_varint(data, cursor)? as usize;

        if key_len == 0 {
            return Ok(entries);
        }

        let key = data
            .get(*cursor..*cursor + key_len)
            .ok_or_else(|| BitcoinError::Parse("truncated PSBT key".to_string()))?
            .to_vec();
        *cursor += key_len;

        // Duplicate keys make the PSBT ambiguous.
        if !seen.insert(key.clone()) {
            return Err(BitcoinError::Parse(
                "duplicate key in a PSBT map".to_string(),
            ));
        }

        let value_len = read_varint(data, cursor)? as usize;
        let value = data
            .get(*cursor..*cursor + value_len)
            .ok_or_else(|| BitcoinError::Parse("truncated PSBT value".to_string()))?
            .to_vec();
        *cursor += value_len;

        entries.push((key, value));
    }
}

/// Parse a serialized witness stack.
fn parse_witness(data: &[u8]) -> Result<Vec<Vec<u8>>> {
    let mut cursor = 0usize;
    let count = read_varint(data, &mut cursor)?;
    let mut items = Vec::with_capacity(count as usize);

    for _ in 0..count {
        let len = read_varint(data, &mut cursor)? as usize;
        let item = data
            .get(cursor..cursor + len)
            .ok_or_else(|| BitcoinError::Parse("truncated witness item".to_string()))?
            .to_vec();
        cursor += len;
        items.push(item);
    }

    if cursor != data.len() {
        return Err(BitcoinError::Parse(
            "trailing bytes after the witness".to_string(),
        ));
    }

    Ok(items)
}

/// Serialize a witness stack.
pub fn serialize_witness(items: &[Vec<u8>]) -> Vec<u8> {
    let mut out = Vec::new();
    out.extend_from_slice(&varint(items.len() as u64));
    for item in items {
        out.extend_from_slice(&varint(item.len() as u64));
        out.extend_from_slice(item);
    }
    out
}

// ============================================================================
// BIP-32 extended keys
// ============================================================================

/// A BIP-32 extended public key.
///
/// Derivation itself lives in the key-management crate; this type is the
/// serialization used to move an account xpub between components, and the
/// non-hardened child derivation that goes with it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExtendedPublicKey {
    pub depth: u8,
    pub parent_fingerprint: [u8; 4],
    pub child_number: u32,
    pub chain_code: [u8; 32],
    pub public_key: [u8; 33],
}

impl ExtendedPublicKey {
    /// Serialize to the Base58Check form (`xpub`, `ypub`, `zpub`, ...).
    pub fn encode(&self, version: u32) -> String {
        let mut payload = Vec::with_capacity(78);
        payload.extend_from_slice(&version.to_be_bytes());
        payload.push(self.depth);
        payload.extend_from_slice(&self.parent_fingerprint);
        payload.extend_from_slice(&self.child_number.to_be_bytes());
        payload.extend_from_slice(&self.chain_code);
        payload.extend_from_slice(&self.public_key);

        let checksum = double_sha256(&payload);
        payload.extend_from_slice(&checksum[..4]);
        bs58::encode(&payload).into_string()
    }

    /// Parse a Base58Check extended public key, returning the version bytes so
    /// the caller can decide which script type it describes.
    pub fn decode(encoded: &str) -> Result<(u32, Self)> {
        let decoded = bs58::decode(encoded)
            .into_vec()
            .map_err(|e| BitcoinError::Parse(e.to_string()))?;

        if decoded.len() != 82 {
            return Err(BitcoinError::Parse(
                "an extended key is 78 bytes plus a 4-byte checksum".to_string(),
            ));
        }

        let (body, checksum) = decoded.split_at(78);
        if double_sha256(body)[..4] != *checksum {
            return Err(BitcoinError::Parse(
                "extended key checksum mismatch".to_string(),
            ));
        }

        let version = u32::from_be_bytes(body[..4].try_into().unwrap());

        let mut parent_fingerprint = [0u8; 4];
        parent_fingerprint.copy_from_slice(&body[5..9]);

        let mut chain_code = [0u8; 32];
        chain_code.copy_from_slice(&body[13..45]);

        let mut public_key = [0u8; 33];
        public_key.copy_from_slice(&body[45..78]);

        if public_key[0] != 0x02 && public_key[0] != 0x03 {
            return Err(BitcoinError::Parse(
                "an extended public key must carry a compressed public key".to_string(),
            ));
        }

        Ok((
            version,
            ExtendedPublicKey {
                depth: body[4],
                parent_fingerprint,
                child_number: u32::from_be_bytes(body[9..13].try_into().unwrap()),
                chain_code,
                public_key,
            },
        ))
    }

    /// Derive a non-hardened child, following BIP-32.
    ///
    /// `index` must be below `2^31`: a hardened child can only be derived from
    /// a private key.
    pub fn derive_child(&self, index: u32) -> Result<Self> {
        if index >= 0x8000_0000 {
            return Err(BitcoinError::SigningError(
                "a hardened child cannot be derived from an extended public key".to_string(),
            ));
        }

        use hmac::{Hmac, Mac};
        type HmacSha512 = Hmac<sha2::Sha512>;

        let mut mac = HmacSha512::new_from_slice(&self.chain_code)
            .map_err(|e| BitcoinError::SigningError(e.to_string()))?;
        mac.update(&self.public_key);
        mac.update(&index.to_be_bytes());
        let digest = mac.finalize().into_bytes();

        let (left, right) = digest.split_at(32);

        // The child key is IL + parent point; the addition happens on the curve,
        // which needs the key type from the signing crate.
        let parent = secp256k1::PublicKey::from_slice(&self.public_key)
            .map_err(|e| BitcoinError::SigningError(e.to_string()))?;
        let tweak = secp256k1::Scalar::from_be_bytes(
            left.try_into()
                .map_err(|_| BitcoinError::SigningError("bad tweak".to_string()))?,
        )
        .map_err(|e| BitcoinError::SigningError(e.to_string()))?;

        let child = parent
            .add_exp_tweak(secp256k1::SECP256K1, &tweak)
            .map_err(|e| BitcoinError::SigningError(e.to_string()))?;

        let mut chain_code = [0u8; 32];
        chain_code.copy_from_slice(right);

        Ok(ExtendedPublicKey {
            depth: self.depth.saturating_add(1),
            parent_fingerprint: fingerprint(&self.public_key),
            child_number: index,
            chain_code,
            public_key: child.serialize(),
        })
    }
}

/// The first four bytes of HASH160, used to identify a parent key.
pub fn fingerprint(public_key: &[u8]) -> [u8; 4] {
    let hash = hash160(public_key);
    let mut out = [0u8; 4];
    out.copy_from_slice(&hash[..4]);
    out
}

// ============================================================================
// Stacks (STX) Integration
// ============================================================================

/// c32check, the base-32 encoding Stacks addresses use.
///
/// This is a port of the reference algorithm in `@stacks/encoding`: the
/// alphabet is Crockford base-32, the payload is encoded over hex digits, and
/// the four-byte checksum is `SHA256(SHA256(version || payload))[..4]`.
pub mod c32 {
    use sha2::{Digest, Sha256};

    const ALPHABET: &[u8; 32] = b"0123456789ABCDEFGHJKMNPQRSTVWXYZ";
    const HEX: &[u8; 16] = b"0123456789abcdef";

    fn hex_value(digit: u8) -> usize {
        match digit {
            b'0'..=b'9' => (digit - b'0') as usize,
            b'a'..=b'f' => (digit - b'a' + 10) as usize,
            _ => 0,
        }
    }

    /// Index of `character` in the alphabet.
    fn digit(character: u8) -> Option<u8> {
        ALPHABET
            .iter()
            .position(|c| *c == character.to_ascii_uppercase())
            .map(|index| index as u8)
    }

    /// Encode bytes with the Stacks base-32 variant.
    fn encode_bytes(data: &[u8]) -> String {
        let hex: Vec<u8> = data
            .iter()
            .flat_map(|byte| [HEX[(byte >> 4) as usize], HEX[(byte & 0x0f) as usize]])
            .collect();

        let mut out: Vec<u8> = Vec::new();
        let mut carry: usize = 0;

        for index in (0..hex.len()).rev() {
            if carry < 4 {
                let current = hex_value(hex[index]) >> carry;
                let next = if index != 0 { hex_value(hex[index - 1]) } else { 0 };
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

        String::from_utf8(result).expect("the alphabet is ASCII")
    }

    /// Decode a Stacks base-32 string into bytes.
    fn decode_bytes(input: &str) -> Option<Vec<u8>> {
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

    /// Encode `version || payload` as a c32check string, without the leading
    /// `"S"`.
    pub fn encode_check(version: u8, payload: &[u8]) -> String {
        if version >= 32 {
            return String::new();
        }

        let mut data = Vec::with_capacity(payload.len() + 4);
        data.extend_from_slice(payload);
        data.extend_from_slice(&checksum(version, payload));

        format!("{}{}", ALPHABET[version as usize] as char, encode_bytes(&data))
    }

    /// Decode a c32check string, verifying the checksum.
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
}

/// Stacks address version bytes, from `@stacks/transactions`.
pub mod stacks_versions {
    /// Mainnet single-signature (starts with `SP`).
    pub const MAINNET_SINGLE_SIG: u8 = 22;
    /// Mainnet contract (starts with `SM`).
    pub const MAINNET_CONTRACT: u8 = 20;
    /// Testnet single-signature (starts with `ST`).
    pub const TESTNET_SINGLE_SIG: u8 = 26;
    /// Testnet contract (starts with `SN`).
    pub const TESTNET_CONTRACT: u8 = 21;
}

/// A decoded Stacks address.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StacksAddress {
    /// c32 version byte.
    pub version: u8,
    /// The 20-byte HASH160 of the spending condition.
    pub hash_bytes: Vec<u8>,
}

impl StacksAddress {
    /// Decode a Stacks address, verifying its c32check checksum.
    pub fn decode(address: &str) -> Result<Self> {
        let (version, payload) = c32::decode_check(address.trim()).ok_or_else(|| {
            BitcoinError::InvalidAddress("not a valid c32check Stacks address".to_string())
        })?;

        let known = matches!(
            version,
            stacks_versions::MAINNET_SINGLE_SIG
                | stacks_versions::MAINNET_CONTRACT
                | stacks_versions::TESTNET_SINGLE_SIG
                | stacks_versions::TESTNET_CONTRACT
        );
        if !known {
            return Err(BitcoinError::InvalidAddress(format!(
                "unknown Stacks version byte {version}"
            )));
        }

        if payload.len() != 20 {
            return Err(BitcoinError::InvalidAddress(
                "a Stacks address payload must be 20 bytes".to_string(),
            ));
        }

        Ok(StacksAddress {
            version,
            hash_bytes: payload,
        })
    }

    /// Encode this address.
    pub fn encode(&self) -> String {
        format!("S{}", c32::encode_check(self.version, &self.hash_bytes))
    }

    /// Whether this is a mainnet address.
    pub fn is_mainnet(&self) -> bool {
        self.version == stacks_versions::MAINNET_SINGLE_SIG
            || self.version == stacks_versions::MAINNET_CONTRACT
    }

    /// Whether this is a contract address rather than a single-signature one.
    pub fn is_contract(&self) -> bool {
        self.version == stacks_versions::MAINNET_CONTRACT
            || self.version == stacks_versions::TESTNET_CONTRACT
    }

    /// The Bitcoin address that controls this Stacks address.
    ///
    /// Stacks reuses the Bitcoin HASH160 directly, so the P2PKH encoding of the
    /// same bytes is the controlling address.
    pub fn to_bitcoin_address(&self) -> Result<BitcoinAddress> {
        let network = if self.is_mainnet() {
            Network::Mainnet
        } else {
            Network::Testnet
        };

        BitcoinAddress::p2pkh(&self.hash_bytes, network)
    }
}

// ============================================================================
// Stacks node client
// ============================================================================

pub struct StacksService {
    rpc_url: String,
    http: reqwest::blocking::Client,
}

impl StacksService {
    pub fn new(rpc_url: String) -> Self {
        StacksService {
            rpc_url: rpc_url.trim_end_matches('/').to_string(),
            http: reqwest::blocking::Client::builder()
                .timeout(std::time::Duration::from_secs(15))
                .build()
                .expect("the HTTP client builds"),
        }
    }

    fn get_json(&self, path: &str) -> Result<serde_json::Value> {
        let url = format!("{}{}", self.rpc_url, path);

        let response = self
            .http
            .get(&url)
            .send()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        let status = response.status();
        let body = response
            .text()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        if !status.is_success() {
            return Err(BitcoinError::RPCError(format!(
                "stacks node returned {status}: {body}"
            )));
        }

        serde_json::from_str(&body).map_err(|e| BitcoinError::Parse(e.to_string()))
    }

    /// Read the STX balance of an address, in micro-STX.
    ///
    /// The node reports this as a decimal string, so it is parsed as an integer
    /// rather than through a float.
    pub fn get_balance(&self, address: &str) -> Result<u64> {
        let decoded = StacksAddress::decode(address)?;
        let path = format!(
            "/v2/accounts/{}?proof=0",
            decoded.encode()
        );

        let body = self.get_json(&path)?;
        let balance = body["balance"]
            .as_str()
            .ok_or_else(|| BitcoinError::RPCError("the response has no balance".to_string()))?;

        balance
            .parse::<u64>()
            .map_err(|e| BitcoinError::Parse(format!("balance {balance:?} is not an integer: {e}")))
    }

    /// Read the nonce of an address, which the next transaction must use.
    pub fn get_nonce(&self, address: &str) -> Result<u64> {
        let decoded = StacksAddress::decode(address)?;
        let body = self.get_json(&format!("/v2/accounts/{}?proof=0", decoded.encode()))?;

        body["nonce"]
            .as_u64()
            .ok_or_else(|| BitcoinError::RPCError("the response has no nonce".to_string()))
    }

    /// Read the assets an address holds.
    pub fn get_balances(&self, address: &str) -> Result<Vec<StacksAsset>> {
        let decoded = StacksAddress::decode(address)?;
        let body = self.get_json(&format!("/extended/v1/address/{}/balances", decoded.encode()))?;

        let mut assets = Vec::new();

        if let Some(stx) = body["stx"]["balance"].as_str() {
            assets.push(StacksAsset {
                asset_identifier: "STX".to_string(),
                balance: stx.to_string(),
            });
        }

        for group in ["fungible_tokens", "non_fungible_tokens"] {
            if let Some(map) = body[group].as_object() {
                for (identifier, entry) in map {
                    assets.push(StacksAsset {
                        asset_identifier: identifier.clone(),
                        balance: entry["balance"].as_str().unwrap_or("0").to_string(),
                    });
                }
            }
        }

        Ok(assets)
    }

    /// Read a contract's data, which includes its source and clarity version.
    pub fn get_contract(&self, contract_id: &str) -> Result<serde_json::Value> {
        // A contract id is `<principal>.<contract-name>`; both parts are
        // restricted to the characters Stacks allows, so the path cannot be
        // steered elsewhere.
        if !contract_id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '.' || c == '-' || c == '_')
        {
            return Err(BitcoinError::InvalidAddress(
                "a contract id may only contain alphanumerics, '.', '-' and '_'".to_string(),
            ));
        }

        self.get_json(&format!("/v2/contracts/source/{contract_id}"))
    }

    /// Broadcast a signed transaction, returning the transaction id the node
    /// accepted.
    pub fn broadcast(&self, signed_tx_hex: &str) -> Result<String> {
        let url = format!("{}/v2/transactions", self.rpc_url);

        let response = self
            .http
            .post(&url)
            .header("content-type", "application/octet-stream")
            .body(signed_tx_hex.to_string())
            .send()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        let status = response.status();
        let body = response
            .text()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        if !status.is_success() {
            return Err(BitcoinError::RPCError(format!(
                "stacks node rejected the transaction with {status}: {body}"
            )));
        }

        // The node replies with a JSON string holding the transaction id.
        let txid: serde_json::Value =
            serde_json::from_str(&body).map_err(|e| BitcoinError::Parse(e.to_string()))?;

        txid.as_str()
            .map(|id| id.to_string())
            .ok_or_else(|| BitcoinError::Parse(format!("unexpected broadcast response: {body}")))
    }
}

/// One asset balance reported by the Stacks node.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StacksAsset {
    pub asset_identifier: String,
    pub balance: String,
}

/// An NFT held by an address.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NftAsset {
    /// Contract that defines the asset, in `<principal>.<name>` form.
    pub asset_identifier: String,
    /// The token's serial number within that contract.
    pub token_id: String,
    /// A URI for the token's metadata, when the contract exposes one.
    pub metadata_uri: Option<String>,
}

// ============================================================================
// Ordinals and BRC-20
// ============================================================================

/// An inscription envelope, as defined by the ordinals protocol.
///
/// The envelope is the payload that gets embedded in a tapscript witness:
/// `OP_FALSE OP_IF "ord" OP_1 <content-type> OP_0 <body> OP_ENDIF`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Inscription {
    pub content_type: String,
    pub body: Vec<u8>,
    /// Optional pointer to an input, a parent inscription, or metadata.
    pub parents: Vec<Vec<u8>>,
    pub metadata: Option<Vec<u8>>,
}

impl Inscription {
    /// Build the tapscript that carries this inscription.
    pub fn to_envelope(&self) -> Vec<u8> {
        let mut script = vec![0x00, 0x63]; // OP_FALSE OP_IF

        push_data(&mut script, b"ord");
        script.push(0x51); // OP_1, the tag for content-type
        push_data(&mut script, self.content_type.as_bytes());

        for parent in &self.parents {
            script.push(0x52); // OP_2, the tag for parent
            push_data(&mut script, parent);
        }

        if let Some(metadata) = &self.metadata {
            script.push(0x55); // OP_5, the tag for metadata
            push_data(&mut script, metadata);
        }

        script.push(0x00); // OP_0, the tag for the body
        push_data(&mut script, &self.body);

        script.push(0x68); // OP_ENDIF
        script
    }
}

/// Push `data` onto a script, using the shortest valid push opcode.
fn push_data(script: &mut Vec<u8>, data: &[u8]) {
    match data.len() {
        0 => script.push(0x00),
        1..=75 => {
            script.push(data.len() as u8);
            script.extend_from_slice(data);
        }
        76..=255 => {
            script.push(0x4c);
            script.push(data.len() as u8);
            script.extend_from_slice(data);
        }
        256..=65535 => {
            script.push(0x4d);
            script.extend_from_slice(&(data.len() as u16).to_le_bytes());
            script.extend_from_slice(data);
        }
        _ => {
            script.push(0x4e);
            script.extend_from_slice(&(data.len() as u32).to_le_bytes());
            script.extend_from_slice(data);
        }
    }
}

/// A BRC-20 operation, which is JSON embedded in an inscription body.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Brc20Operation {
    Deploy {
        tick: String,
        max: String,
        lim: Option<String>,
        decimals: Option<String>,
    },
    Mint {
        tick: String,
        amount: String,
    },
    Transfer {
        tick: String,
        amount: String,
    },
}

impl Brc20Operation {
    /// Build the inscription body for this operation.
    pub fn to_inscription_body(&self) -> String {
        match self {
            Brc20Operation::Deploy {
                tick,
                max,
                lim,
                decimals,
            } => {
                let mut value = serde_json::json!({
                    "p": "brc-20",
                    "op": "deploy",
                    "tick": tick,
                    "max": max,
                });

                if let Some(lim) = lim {
                    value["lim"] = serde_json::Value::String(lim.clone());
                }
                if let Some(decimals) = decimals {
                    value["dec"] = serde_json::Value::String(decimals.clone());
                }

                value.to_string()
            }
            Brc20Operation::Mint { tick, amount } => serde_json::json!({
                "p": "brc-20",
                "op": "mint",
                "tick": tick,
                "amt": amount,
            })
            .to_string(),
            Brc20Operation::Transfer { tick, amount } => serde_json::json!({
                "p": "brc-20",
                "op": "transfer",
                "tick": tick,
                "amt": amount,
            })
            .to_string(),
        }
    }

    /// Parse an inscription body back into an operation.
    pub fn from_inscription_body(body: &str) -> Result<Self> {
        let value: serde_json::Value =
            serde_json::from_str(body).map_err(|e| BitcoinError::Parse(e.to_string()))?;

        if value["p"] != "brc-20" {
            return Err(BitcoinError::Parse(
                "not a brc-20 inscription".to_string(),
            ));
        }

        let text = |field: &str| -> Result<String> {
            value[field]
                .as_str()
                .map(|s| s.to_string())
                .ok_or_else(|| BitcoinError::Parse(format!("missing field {field:?}")))
        };

        match value["op"].as_str() {
            Some("deploy") => Ok(Brc20Operation::Deploy {
                tick: text("tick")?,
                max: text("max")?,
                lim: value["lim"].as_str().map(|s| s.to_string()),
                decimals: value["dec"].as_str().map(|s| s.to_string()),
            }),
            Some("mint") => Ok(Brc20Operation::Mint {
                tick: text("tick")?,
                amount: text("amt")?,
            }),
            Some("transfer") => Ok(Brc20Operation::Transfer {
                tick: text("tick")?,
                amount: text("amt")?,
            }),
            other => Err(BitcoinError::Parse(format!(
                "unknown brc-20 operation {other:?}"
            ))),
        }
    }

    /// Whether the operation's numeric fields are well formed.
    ///
    /// The indexers are strict about this, and an inscription with a malformed
    /// amount is treated as invalid rather than minted.
    pub fn validate(&self) -> Result<()> {
        fn check_number(field: &str, value: &str) -> Result<()> {
            if value.is_empty() || !value.chars().all(|c| c.is_ascii_digit()) {
                return Err(BitcoinError::OrdinalsError(format!(
                    "{field} must be a non-empty integer string, got {value:?}"
                )));
            }

            // Leading zeros are rejected by the indexers.
            if value.len() > 1 && value.starts_with('0') {
                return Err(BitcoinError::OrdinalsError(format!(
                    "{field} must not have leading zeros, got {value:?}"
                )));
            }

            Ok(())
        }

        fn check_tick(tick: &str) -> Result<()> {
            // Four bytes, UTF-8, no spaces.
            if tick.is_empty() || tick.len() > 4 || tick.contains(' ') {
                return Err(BitcoinError::OrdinalsError(format!(
                    "a ticker is one to four bytes with no spaces, got {tick:?}"
                )));
            }
            Ok(())
        }

        match self {
            Brc20Operation::Deploy {
                tick,
                max,
                lim,
                decimals,
            } => {
                check_tick(tick)?;
                check_number("max", max)?;
                if let Some(lim) = lim {
                    check_number("lim", lim)?;
                }
                if let Some(decimals) = decimals {
                    check_number("dec", decimals)?;
                    if decimals.parse::<u8>().map(|d| d > 18).unwrap_or(true) {
                        return Err(BitcoinError::OrdinalsError(
                            "decimals must be 0..=18".to_string(),
                        ));
                    }
                }
            }
            Brc20Operation::Mint { tick, amount } | Brc20Operation::Transfer { tick, amount } => {
                check_tick(tick)?;
                check_number("amt", amount)?;
            }
        }

        Ok(())
    }
}

/// Read ordinals data from an ord server.
pub struct OrdinalsService {
    base_url: String,
    http: reqwest::blocking::Client,
}

impl OrdinalsService {
    pub fn new(base_url: String) -> Self {
        OrdinalsService {
            base_url: base_url.trim_end_matches('/').to_string(),
            http: reqwest::blocking::Client::builder()
                .timeout(std::time::Duration::from_secs(20))
                .build()
                .expect("the HTTP client builds"),
        }
    }

    fn get_json(&self, path: &str) -> Result<serde_json::Value> {
        let response = self
            .http
            .get(format!("{}{}", self.base_url, path))
            .send()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        let status = response.status();
        let body = response
            .text()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        if !status.is_success() {
            return Err(BitcoinError::RPCError(format!(
                "the ord server returned {status}: {body}"
            )));
        }

        serde_json::from_str(&body).map_err(|e| BitcoinError::Parse(e.to_string()))
    }

    /// Read an inscription by id.
    pub fn get_inscription(&self, inscription_id: &str) -> Result<OrdinalInscription> {
        if !inscription_id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == 'i' || c == ':' || c == 'x')
        {
            return Err(BitcoinError::OrdinalsError(
                "an inscription id is <txid>i<index>".to_string(),
            ));
        }

        let value = self.get_json(&format!("/inscription/{inscription_id}"))?;

        Ok(OrdinalInscription {
            id: inscription_id.to_string(),
            inscription_id: inscription_id.to_string(),
            address: value["address"].as_str().unwrap_or_default().to_string(),
            content_type: value["content_type"]
                .as_str()
                .unwrap_or("application/octet-stream")
                .to_string(),
            content_length: value["content_length"].as_u64(),
            genesis_height: value["genesis_height"].as_u64(),
            timestamp: value["timestamp"].as_u64().unwrap_or(0),
            sat: value["sat"].as_u64(),
        })
    }

    /// Read the inscriptions held by an address.
    pub fn get_address_inscriptions(&self, address: &str) -> Result<Vec<OrdinalInscription>> {
        // Addresses are bech32 or base58; reject anything with path characters
        // so the request cannot be steered.
        if !address
            .chars()
            .all(|c| c.is_ascii_alphanumeric())
        {
            return Err(BitcoinError::InvalidAddress(
                "an address may only contain alphanumerics".to_string(),
            ));
        }

        let value = self.get_json(&format!("/address/{address}/inscriptions"))?;

        let Some(entries) = value["inscriptions"].as_array() else {
            return Ok(Vec::new());
        };

        let mut out = Vec::with_capacity(entries.len());
        for entry in entries {
            out.push(OrdinalInscription {
                id: entry["id"].as_str().unwrap_or_default().to_string(),
                inscription_id: entry["id"].as_str().unwrap_or_default().to_string(),
                address: address.to_string(),
                content_type: entry["content_type"]
                    .as_str()
                    .unwrap_or("application/octet-stream")
                    .to_string(),
                content_length: entry["content_length"].as_u64(),
                genesis_height: entry["genesis_height"].as_u64(),
                timestamp: entry["timestamp"].as_u64().unwrap_or(0),
                sat: entry["sat"].as_u64(),
            });
        }

        Ok(out)
    }

    /// Read a BRC-20 ticker's current state.
    pub fn get_brc20_token(&self, tick: &str) -> Result<Brc20Token> {
        let value = self.get_json(&format!("/brc-20/tokens/{tick}"))?;

        let number = |field: &str| -> String { value[field].as_str().unwrap_or("0").to_string() };

        Ok(Brc20Token {
            tick: tick.to_string(),
            max_supply: number("max_supply"),
            minted_supply: number("minted_supply"),
            limit_per_mint: value["limit_per_mint"].as_str().map(|s| s.to_string()),
            decimals: value["decimals"].as_u64().unwrap_or(0) as u8,
            holders: value["holders"].as_u64().unwrap_or(0),
            deploy_timestamp: value["deploy_timestamp"].as_u64().unwrap_or(0),
        })
    }

    /// Read an address's BRC-20 balances.
    pub fn get_brc20_balances(&self, address: &str) -> Result<Vec<Brc20Balance>> {
        if !address.chars().all(|c| c.is_ascii_alphanumeric()) {
            return Err(BitcoinError::InvalidAddress(
                "an address may only contain alphanumerics".to_string(),
            ));
        }

        let value = self.get_json(&format!("/brc-20/balances/{address}"))?;

        let Some(entries) = value["results"].as_array() else {
            return Ok(Vec::new());
        };

        Ok(entries
            .iter()
            .map(|entry| Brc20Balance {
                tick: entry["ticker"].as_str().unwrap_or_default().to_string(),
                overall: entry["overall_balance"]
                    .as_str()
                    .unwrap_or("0")
                    .to_string(),
                transferable: entry["transferable_balance"]
                    .as_str()
                    .unwrap_or("0")
                    .to_string(),
            })
            .collect())
    }

    /// Read an address's BRC-20 balance for one ticker.
    pub fn get_brc20_balance(&self, address: &str, tick: &str) -> Result<String> {
        Ok(self
            .get_brc20_balances(address)?
            .into_iter()
            .find(|balance| balance.tick == tick)
            .map(|balance| balance.overall)
            .unwrap_or_else(|| "0".to_string()))
    }
}

/// An inscription as reported by an ord server.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OrdinalInscription {
    pub id: String,
    pub inscription_id: String,
    pub address: String,
    pub content_type: String,
    pub content_length: Option<u64>,
    pub genesis_height: Option<u64>,
    pub timestamp: u64,
    pub sat: Option<u64>,
}

/// A BRC-20 ticker's state.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Brc20Token {
    pub tick: String,
    pub max_supply: String,
    pub minted_supply: String,
    pub limit_per_mint: Option<String>,
    pub decimals: u8,
    pub holders: u64,
    pub deploy_timestamp: u64,
}

/// An address's BRC-20 holdings for one ticker.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Brc20Balance {
    pub tick: String,
    pub overall: String,
    pub transferable: String,
}

// ============================================================================
// Lightning Network (BOLT-11)
// ============================================================================

/// A decoded BOLT-11 invoice.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Bolt11Invoice {
    /// Network prefix, `lnbc`, `lntb` or `lnbcrt`.
    pub currency: String,
    /// Amount in milli-satoshi, when the invoice is not amount-less.
    pub amount_msat: Option<u64>,
    pub timestamp: u64,
    pub payment_hash: [u8; 32],
    pub description: Option<String>,
    pub description_hash: Option<[u8; 32]>,
    pub expiry_seconds: u64,
    pub min_final_cltv_expiry: u64,
    pub payee: [u8; 33],
    pub signature: [u8; 64],
    /// Every 5-bit data group of the invoice, including the checksum.
    pub data_groups: Vec<u8>,
}

impl Bolt11Invoice {
    /// Whether the invoice has passed its expiry.
    pub fn is_expired(&self, now: u64) -> bool {
        now > self.timestamp.saturating_add(self.expiry_seconds)
    }

    /// The 5-bit groups the payee signed.
    ///
    /// BOLT-11 signs the human-readable part followed by the data groups with
    /// the 104 signature groups replaced by zeros. The caller expands the
    /// human-readable part and appends these groups to rebuild the message.
    pub fn signing_groups(&self) -> Vec<u8> {
        let mut groups = self.data_groups.clone();
        let start = groups.len().saturating_sub(104);
        for group in groups[start..].iter_mut() {
            *group = 0;
        }
        groups
    }
}

/// BOLT-11 amount units, as multipliers on the base 10^0 unit.
///
/// The invoice amount is expressed in bitcoin with a unit suffix: `m` is
/// milli-bitcoin, `u` micro, `n` nano, `p` pico.
fn bolt11_amount_multiplier(unit: Option<char>) -> Option<u64> {
    match unit {
        None => Some(100_000_000_000),      // bitcoin -> msat
        Some('m') => Some(100_000_000),     // milli-bitcoin -> msat
        Some('u') => Some(100_000),         // micro-bitcoin -> msat
        Some('n') => Some(100),             // nano-bitcoin -> msat
        Some('p') => Some(1),               // pico-bitcoin, handled by 10
        _ => None,
    }
}

/// Decode a BOLT-11 invoice.
///
/// This checks the bech32 checksum and the field structure. The signature is
/// returned so the caller can verify it with the payee's public key; it is not
/// verified here because the recovery step belongs with the signing crate.
pub fn decode_bolt11(invoice: &str) -> Result<Bolt11Invoice> {
    let lowered = invoice.trim().to_ascii_lowercase();
    if lowered.len() < 8 {
        return Err(BitcoinError::LightningError("invoice is too short".to_string()));
    }

    // Invoices routinely exceed the 90-character segwit address limit.
    let (hrp, data, _variant) = bech32_impl::decode_unbounded(&lowered)
        .ok_or_else(|| BitcoinError::LightningError("invalid bech32 checksum".to_string()))?;

    let currency = ["lnbcrt", "lnbc", "lntb", "lnsb"]
        .iter()
        .find(|prefix| hrp.starts_with(**prefix))
        .ok_or_else(|| {
            BitcoinError::LightningError(format!("unknown invoice currency {hrp:?}"))
        })?;

    let amount_part = &hrp[currency.len()..];
    let amount_msat = if amount_part.is_empty() {
        None
    } else {
        // A trailing unit letter may be present.
        let (digits, unit) = match amount_part.chars().last() {
            Some(last) if last.is_ascii_alphabetic() => {
                (&amount_part[..amount_part.len() - 1], Some(last))
            }
            _ => (amount_part, None),
        };

        if digits.is_empty() || !digits.chars().all(|c| c.is_ascii_digit()) {
            return Err(BitcoinError::LightningError(format!(
                "invalid amount {amount_part:?}"
            )));
        }

        let value: u64 = digits
            .parse()
            .map_err(|e| BitcoinError::LightningError(format!("invalid amount: {e}")))?;

        let multiplier = bolt11_amount_multiplier(unit).ok_or_else(|| {
            BitcoinError::LightningError(format!("unknown amount unit {unit:?}"))
        })?;

        // The pico unit is a tenth of a milli-satoshi, so an amount that is not
        // a whole number of milli-satoshi cannot be represented.
        let msat = if unit == Some('p') {
            if value % 10 != 0 {
                return Err(BitcoinError::LightningError(
                    "a pico-bitcoin amount must be a whole number of milli-satoshi".to_string(),
                ));
            }
            value / 10
        } else {
            value.checked_mul(multiplier).ok_or_else(|| {
                BitcoinError::LightningError("amount overflows".to_string())
            })?
        };

        Some(msat)
    };

    // The data part is a stream of 5-bit groups. The signature is the final
    // 104 groups; everything before it is the payload.
    const SIGNATURE_GROUPS: usize = 104;
    if data.len() < 7 + SIGNATURE_GROUPS {
        return Err(BitcoinError::LightningError(
            "invoice is too short to hold a signature".to_string(),
        ));
    }

    let (payload, signature_groups) = data.split_at(data.len() - SIGNATURE_GROUPS);
    let signature_bytes = bech32_impl::convert_bits(signature_groups, 5, 8, false)
        .ok_or_else(|| BitcoinError::LightningError("bad signature encoding".to_string()))?;
    if signature_bytes.len() != 65 {
        return Err(BitcoinError::LightningError(
            "the signature must be 65 bytes".to_string(),
        ));
    }

    let mut signature = [0u8; 64];
    signature.copy_from_slice(&signature_bytes[..64]);

    // Fields are not byte aligned: a 35-bit timestamp is followed by 5-bit
    // tags and 10-bit lengths, so the payload is read bit by bit.
    let total_bits = payload.len() * 5;
    let mut bit_cursor = 0usize;

    fn read_bits(payload: &[u8], total_bits: usize, count: usize, cursor: &mut usize) -> Option<u64> {
        if *cursor + count > total_bits {
            return None;
        }

        let mut value = 0u64;
        for _ in 0..count {
            let group = payload[*cursor / 5];
            let bit = (group >> (4 - (*cursor % 5))) & 1;
            value = (value << 1) | u64::from(bit);
            *cursor += 1;
        }
        Some(value)
    }

    let timestamp = read_bits(payload, total_bits, 35, &mut bit_cursor)
        .ok_or_else(|| BitcoinError::LightningError("truncated timestamp".to_string()))?;

    let mut description = None;
    let mut description_hash = None;
    let mut payment_hash = None;
    let mut expiry_seconds = 3600u64;
    let mut min_final_cltv_expiry = 18u64;
    let mut payee = None;

    while bit_cursor + 15 <= total_bits {
        let field_type = read_bits(payload, total_bits, 5, &mut bit_cursor)
            .ok_or_else(|| BitcoinError::LightningError("truncated field tag".to_string()))?;
        let field_groups = read_bits(payload, total_bits, 10, &mut bit_cursor)
            .ok_or_else(|| BitcoinError::LightningError("truncated field length".to_string()))?
            as usize;

        if bit_cursor + field_groups * 5 > total_bits {
            return Err(BitcoinError::LightningError(
                "a field runs past the end of the payload".to_string(),
            ));
        }

        let mut groups = Vec::with_capacity(field_groups);
        for _ in 0..field_groups {
            groups.push(read_bits(payload, total_bits, 5, &mut bit_cursor).unwrap_or(0) as u8);
        }

        // An integer field is folded directly; a byte field is regrouped, which
        // only succeeds when it holds a whole number of bytes.
        let decoded = if is_integer_field(field_type) {
            None
        } else {
            bech32_impl::convert_bits(&groups, 5, 8, false)
        };

        match field_type {
            // p: payment hash
            1 => {
                let bytes = decoded.ok_or_else(|| {
                    BitcoinError::LightningError(
                        "the payment hash is not a whole number of bytes".to_string(),
                    )
                })?;
                if bytes.len() != 32 {
                    return Err(BitcoinError::LightningError(
                        "a payment hash must be 32 bytes".to_string(),
                    ));
                }
                let mut hash = [0u8; 32];
                hash.copy_from_slice(&bytes);
                payment_hash = Some(hash);
            }
            // d: description
            13 => {
                let bytes = decoded.ok_or_else(|| {
                    BitcoinError::LightningError(
                        "the description is not a whole number of bytes".to_string(),
                    )
                })?;
                description = Some(String::from_utf8_lossy(&bytes).to_string());
            }
            // n: payee public key
            19 => {
                let bytes = decoded.ok_or_else(|| {
                    BitcoinError::LightningError(
                        "the payee key is not a whole number of bytes".to_string(),
                    )
                })?;
                if bytes.len() != 33 {
                    return Err(BitcoinError::LightningError(
                        "a payee key must be 33 bytes".to_string(),
                    ));
                }
                let mut key = [0u8; 33];
                key.copy_from_slice(&bytes);
                payee = Some(key);
            }
            // h: description hash
            23 => {
                let bytes = decoded.ok_or_else(|| {
                    BitcoinError::LightningError(
                        "the description hash is not a whole number of bytes".to_string(),
                    )
                })?;
                if bytes.len() != 32 {
                    return Err(BitcoinError::LightningError(
                        "a description hash must be 32 bytes".to_string(),
                    ));
                }
                let mut hash = [0u8; 32];
                hash.copy_from_slice(&bytes);
                description_hash = Some(hash);
            }
            // x: expiry, and c: min_final_cltv_expiry.
            6 | 24 => {
                let value = read_integer_field(&groups).ok_or_else(|| {
                    BitcoinError::LightningError("a numeric field is too large".to_string())
                })?;
                if field_type == 6 {
                    expiry_seconds = value;
                } else {
                    min_final_cltv_expiry = value;
                }
            }
            _ => {}
        }
    }

    Ok(Bolt11Invoice {
        currency: (*currency).to_string(),
        amount_msat,
        timestamp,
        payment_hash: payment_hash.ok_or_else(|| {
            BitcoinError::LightningError("the invoice has no payment hash".to_string())
        })?,
        description,
        description_hash,
        expiry_seconds,
        min_final_cltv_expiry,
        payee: payee
            .ok_or_else(|| BitcoinError::LightningError("the invoice has no payee".to_string()))?,
        signature,
        data_groups: data,
    })
}

/// BOLT-11 field tags that carry an integer rather than a byte string.
fn is_integer_field(field_type: u64) -> bool {
    // x: expiry, and c: min_final_cltv_expiry.
    matches!(field_type, 6 | 24)
}

/// Read an integer field.
///
/// The value is big-endian and occupies a whole number of 5-bit groups, so it
/// is a plain fold of those groups.
fn read_integer_field(groups: &[u8]) -> Option<u64> {
    if groups.is_empty() || groups.len() > 13 {
        return None;
    }

    let mut value = 0u64;
    for group in groups {
        value = value.checked_mul(32)?.checked_add(u64::from(*group))?;
    }

    Some(value)
}

// ============================================================================
// Lightning node client
// ============================================================================

/// A client for an LND REST endpoint.
///
/// Every call carries the macaroon in the `Grpc-Metadata-macaroon` header, and
/// failures are reported rather than papered over: an unreachable node returns
/// an error, never a zero balance.
pub struct LightningService {
    base_url: String,
    macaroon_hex: String,
    http: reqwest::blocking::Client,
}

impl LightningService {
    /// Build a client. `macaroon_hex` is the hex-encoded macaroon.
    pub fn new(base_url: String, macaroon_hex: String) -> Self {
        LightningService {
            base_url: base_url.trim_end_matches('/').to_string(),
            macaroon_hex,
            http: reqwest::blocking::Client::builder()
                .timeout(std::time::Duration::from_secs(30))
                .build()
                .expect("the HTTP client builds"),
        }
    }

    fn request(
        &self,
        method: reqwest::Method,
        path: &str,
        body: Option<serde_json::Value>,
    ) -> Result<serde_json::Value> {
        let mut request = self
            .http
            .request(method, format!("{}{}", self.base_url, path))
            .header("Grpc-Metadata-macaroon", &self.macaroon_hex);

        if let Some(body) = body {
            request = request.json(&body);
        }

        let response = request
            .send()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        let status = response.status();
        let text = response
            .text()
            .map_err(|e| BitcoinError::Network(e.to_string()))?;

        if !status.is_success() {
            return Err(BitcoinError::RPCError(format!(
                "lnd returned {status}: {text}"
            )));
        }

        if text.trim().is_empty() {
            return Ok(serde_json::Value::Null);
        }

        serde_json::from_str(&text).map_err(|e| BitcoinError::Parse(e.to_string()))
    }

    /// Node identity and sync state.
    pub fn get_info(&self) -> Result<NodeInfo> {
        let value = self.request(reqwest::Method::GET, "/v1/getinfo", None)?;

        Ok(NodeInfo {
            pubkey: value["identity_pubkey"].as_str().unwrap_or_default().to_string(),
            alias: value["alias"].as_str().unwrap_or_default().to_string(),
            version: value["version"].as_str().unwrap_or_default().to_string(),
            block_height: value["block_height"].as_u64().unwrap_or(0),
            synced_to_chain: value["synced_to_chain"].as_bool().unwrap_or(false),
            num_active_channels: value["num_active_channels"].as_u64().unwrap_or(0),
            num_peers: value["num_peers"].as_u64().unwrap_or(0),
        })
    }

    /// Wallet balance in satoshi.
    pub fn get_wallet_balance(&self) -> Result<WalletBalance> {
        let value = self.request(reqwest::Method::GET, "/v1/balance/blockchain", None)?;

        let number = |field: &str| -> u64 {
            value[field]
                .as_str()
                .and_then(|s| s.parse().ok())
                .unwrap_or(0)
        };

        Ok(WalletBalance {
            total_sat: number("total_balance"),
            confirmed_sat: number("confirmed_balance"),
            unconfirmed_sat: number("unconfirmed_balance"),
        })
    }

    /// Channel balance in satoshi.
    pub fn get_channel_balance(&self) -> Result<ChannelBalance> {
        let value = self.request(reqwest::Method::GET, "/v1/balance/channels", None)?;

        Ok(ChannelBalance {
            local_sat: value["local_balance"]["sat"].as_u64().unwrap_or(0),
            remote_sat: value["remote_balance"]["sat"].as_u64().unwrap_or(0),
        })
    }

    /// Create an invoice.
    pub fn create_invoice(
        &self,
        amount_msat: u64,
        description: &str,
        expiry_seconds: u64,
    ) -> Result<Bolt11Invoice> {
        if amount_msat == 0 {
            return Err(BitcoinError::LightningError(
                "an invoice needs a non-zero amount".to_string(),
            ));
        }

        let value = self.request(
            reqwest::Method::POST,
            "/v1/invoices",
            Some(serde_json::json!({
                "value_msat": amount_msat.to_string(),
                "memo": description,
                "expiry": expiry_seconds.to_string(),
            })),
        )?;

        let encoded = value["payment_request"]
            .as_str()
            .ok_or_else(|| BitcoinError::RPCError("lnd returned no invoice".to_string()))?;

        // Decode what the node returned rather than trusting the inputs.
        decode_bolt11(encoded)
    }

    /// Pay a BOLT-11 invoice, returning the preimage that proves payment.
    pub fn pay_invoice(&self, invoice: &str, max_fee_msat: Option<u64>) -> Result<String> {
        // The invoice is decoded first, so a malformed one never reaches the
        // node and a failure is reported against the right cause.
        let decoded = decode_bolt11(invoice)?;

        let mut body = serde_json::json!({
            "payment_request": invoice,
            "fee_limit": {
                "fixed_msat": max_fee_msat.unwrap_or(decoded.amount_msat.unwrap_or(0) / 100).to_string()
            },
        });

        if decoded.amount_msat.is_none() {
            return Err(BitcoinError::LightningError(
                "an amount-less invoice needs an explicit amount".to_string(),
            ));
        }

        body["amt_msat"] = serde_json::Value::String(
            decoded.amount_msat.unwrap_or(0).to_string(),
        );

        let value = self.request(reqwest::Method::POST, "/v2/router/send", Some(body))?;

        value["result"]["payment_preimage"]
            .as_str()
            .map(|preimage| preimage.to_string())
            .ok_or_else(|| {
                BitcoinError::RPCError(format!("lnd did not return a preimage: {value}"))
            })
    }

    /// Open a channel to a peer.
    pub fn open_channel(
        &self,
        node_pubkey: &str,
        capacity_sat: u64,
        push_sat: u64,
    ) -> Result<String> {
        if capacity_sat == 0 {
            return Err(BitcoinError::LightningError(
                "a channel needs a non-zero capacity".to_string(),
            ));
        }
        if push_sat >= capacity_sat {
            return Err(BitcoinError::LightningError(
                "the push amount must be smaller than the capacity".to_string(),
            ));
        }

        let value = self.request(
            reqwest::Method::POST,
            "/v1/channels",
            Some(serde_json::json!({
                "node_pubkey_string": node_pubkey,
                "local_funding_amount": capacity_sat.to_string(),
                "push_sat": push_sat.to_string(),
            })),
        )?;

        value["funding_txid_str"]
            .as_str()
            .map(|txid| txid.to_string())
            .ok_or_else(|| BitcoinError::RPCError("lnd returned no funding txid".to_string()))
    }

    /// Close a channel.
    pub fn close_channel(&self, channel_point: &str, force: bool) -> Result<String> {
        let value = self.request(
            reqwest::Method::DELETE,
            "/v1/channels",
            Some(serde_json::json!({
                "channel_point": {
                    "funding_txid_str": channel_point.split(':').next().unwrap_or(channel_point),
                    "output_index": channel_point
                        .split(':')
                        .nth(1)
                        .and_then(|index| index.parse::<u32>().ok())
                        .unwrap_or(0),
                },
                "force": force,
            })),
        )?;

        value["txid"]
            .as_str()
            .map(|txid| txid.to_string())
            .ok_or_else(|| BitcoinError::RPCError("lnd returned no closing txid".to_string()))
    }

    /// List channels.
    pub fn list_channels(&self) -> Result<Vec<ChannelInfo>> {
        let value = self.request(reqwest::Method::GET, "/v1/channels", None)?;

        let Some(channels) = value["channels"].as_array() else {
            return Ok(Vec::new());
        };

        Ok(channels
            .iter()
            .map(|channel| ChannelInfo {
                active: channel["active"].as_bool().unwrap_or(false),
                remote_pubkey: channel["remote_pubkey"].as_str().unwrap_or_default().to_string(),
                channel_point: channel["channel_point"].as_str().unwrap_or_default().to_string(),
                capacity_sat: channel["capacity"].as_str().and_then(|s| s.parse().ok()).unwrap_or(0),
                local_balance_sat: channel["local_balance"]
                    .as_str()
                    .and_then(|s| s.parse().ok())
                    .unwrap_or(0),
                remote_balance_sat: channel["remote_balance"]
                    .as_str()
                    .and_then(|s| s.parse().ok())
                    .unwrap_or(0),
            })
            .collect())
    }
}

/// Node identity and sync state.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NodeInfo {
    pub pubkey: String,
    pub alias: String,
    pub version: String,
    pub block_height: u64,
    pub synced_to_chain: bool,
    pub num_active_channels: u64,
    pub num_peers: u64,
}

/// On-chain wallet balance.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WalletBalance {
    pub total_sat: u64,
    pub confirmed_sat: u64,
    pub unconfirmed_sat: u64,
}

/// Channel balance.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChannelBalance {
    pub local_sat: u64,
    pub remote_sat: u64,
}

/// One open channel.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChannelInfo {
    pub active: bool,
    pub remote_pubkey: String,
    pub channel_point: String,
    pub capacity_sat: u64,
    pub local_balance_sat: u64,
    pub remote_balance_sat: u64,
}

// ============================================================================
// Multi-Signature
// ============================================================================

/// An m-of-n multisig wallet.
///
/// The redeem script is `OP_m <pubkey>... OP_n OP_CHECKMULTISIG`, which is the
/// standard bare multisig form; P2SH and P2WSH addresses wrap its hash.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MultiSigWallet {
    pub threshold: u8,
    /// Compressed public keys, in the order they appear in the redeem script.
    pub public_keys: Vec<Vec<u8>>,
    pub redeem_script: Vec<u8>,
}

impl MultiSigWallet {
    /// Build a wallet from its threshold and public keys.
    ///
    /// The keys are sorted, which is what BIP-67 recommends so that every
    /// cosigner derives the same script regardless of the order they were
    /// collected in.
    pub fn new(threshold: u8, public_keys: Vec<Vec<u8>>) -> Result<Self> {
        if public_keys.is_empty() {
            return Err(BitcoinError::InvalidAddress(
                "a multisig wallet needs at least one public key".to_string(),
            ));
        }

        if threshold == 0 || threshold > public_keys.len() as u8 {
            return Err(BitcoinError::InvalidAddress(format!(
                "a threshold of {threshold} is not usable with {} keys",
                public_keys.len()
            )));
        }

        if public_keys.len() > 16 {
            return Err(BitcoinError::InvalidAddress(
                "at most 16 keys can be listed in a bare multisig script".to_string(),
            ));
        }

        let mut keys = public_keys;
        for key in &keys {
            if key.len() != 33 || (key[0] != 0x02 && key[0] != 0x03) {
                return Err(BitcoinError::InvalidAddress(
                    "every public key must be a 33-byte compressed key".to_string(),
                ));
            }
        }

        // A duplicate key would let one signer count twice.
        let mut unique = keys.clone();
        unique.sort();
        unique.dedup();
        if unique.len() != keys.len() {
            return Err(BitcoinError::InvalidAddress(
                "a public key appears more than once".to_string(),
            ));
        }

        keys.sort();

        let mut redeem_script = Vec::with_capacity(3 + keys.len() * 34);
        redeem_script.push(0x50 + threshold); // OP_1..OP_16
        for key in &keys {
            redeem_script.push(33);
            redeem_script.extend_from_slice(key);
        }
        redeem_script.push(0x50 + keys.len() as u8);
        redeem_script.push(0xae); // OP_CHECKMULTISIG

        Ok(MultiSigWallet {
            threshold,
            public_keys: keys,
            redeem_script,
        })
    }

    /// The P2SH address that pays this wallet.
    pub fn p2sh_address(&self, network: Network) -> Result<BitcoinAddress> {
        BitcoinAddress::p2sh_for_script(&self.redeem_script, network)
    }

    /// The P2WSH address that pays this wallet.
    pub fn p2wsh_address(&self, network: Network) -> Result<BitcoinAddress> {
        BitcoinAddress::p2wsh(&self.redeem_script, network)
    }

    /// Build the scriptSig for a set of signatures.
    ///
    /// `OP_CHECKMULTISIG` pops one extra element because of a long-standing
    /// off-by-one in the original implementation, so the stack must start with
    /// `OP_0`. Signatures must be in the same order as the public keys.
    pub fn build_script_sig(&self, signatures: &[Vec<u8>]) -> Result<Vec<u8>> {
        if signatures.len() < self.threshold as usize {
            return Err(BitcoinError::SigningError(format!(
                "{} signatures were supplied but {} are required",
                signatures.len(),
                self.threshold
            )));
        }

        let mut script = vec![0x00]; // the CHECKMULTISIG dummy
        for signature in signatures.iter().take(self.threshold as usize) {
            if signature.is_empty() || signature.len() > 73 {
                return Err(BitcoinError::SigningError(
                    "a signature must be a DER-encoded value of at most 73 bytes".to_string(),
                ));
            }
            push_data(&mut script, signature);
        }
        Ok(script)
    }

    /// Build the witness stack for a P2WSH spend.
    pub fn build_witness(&self, signatures: &[Vec<u8>]) -> Result<Vec<Vec<u8>>> {
        if signatures.len() < self.threshold as usize {
            return Err(BitcoinError::SigningError(format!(
                "{} signatures were supplied but {} are required",
                signatures.len(),
                self.threshold
            )));
        }

        let mut witness = vec![Vec::new()]; // the CHECKMULTISIG dummy
        witness.extend(
            signatures
                .iter()
                .take(self.threshold as usize)
                .cloned(),
        );
        witness.push(self.redeem_script.clone());
        Ok(witness)
    }

    /// Check a partial signature set against the public keys.
    ///
    /// Each signature must be a valid DER-encoded ECDSA signature by the
    /// corresponding key over `digest`. Returns the number that verified.
    pub fn count_valid_signatures(
        &self,
        digest: &[u8; 32],
        signatures: &[Vec<u8>],
    ) -> Result<usize> {
        use secp256k1::ecdsa::Signature;

        let message = secp256k1::Message::from_slice(digest)
            .map_err(|e| BitcoinError::SigningError(e.to_string()))?;
        let secp = secp256k1::Secp256k1::verification_only();

        let mut verified = 0usize;
        let mut key_index = 0usize;

        // CHECKMULTISIG walks the keys and signatures in order, so a signature
        // must match a key at or after the last matched position.
        for signature in signatures {
            let mut matched = false;

            while key_index < self.public_keys.len() {
                let public_key = secp256k1::PublicKey::from_slice(&self.public_keys[key_index])
                    .map_err(|e| BitcoinError::SigningError(e.to_string()))?;

                if let Ok(signature) = Signature::from_der(signature) {
                    if secp.verify_ecdsa(&message, &signature, &public_key).is_ok() {
                        verified += 1;
                        key_index += 1;
                        matched = true;
                        break;
                    }
                }

                key_index += 1;
            }

            if !matched {
                break;
            }
        }

        Ok(verified)
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    /// BIP-173 test vector: a valid P2WPKH address for a zero key hash.
    const GENESIS_P2PKH: &str = "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa";
    const SATOSHI_PUBKEY: &str =
        "04678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112de5c384df7ba0b8d578a4c702b6bf11d5f";

    fn p2wpkh_vector() -> (Vec<u8>, String) {
        // BIP-173 test vector: P2WPKH for the pubkey hash of 0x0279be66...
        // The address is the one published in the BIP.
        let program = hex::decode("751e76e8199196d454941c45d1b3a323f1433bd6").unwrap();
        (program, "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4".to_string())
    }

    #[test]
    fn base58check_round_trips_the_genesis_address() {
        let decoded = BitcoinAddress::decode(GENESIS_P2PKH, Network::Mainnet).unwrap();

        assert_eq!(decoded.address_type, AddressType::P2PKH);
        assert_eq!(decoded.encode(), GENESIS_P2PKH);
        assert_eq!(decoded.hash.len(), 20);

        // The script is OP_DUP OP_HASH160 <20 bytes> OP_EQUALVERIFY OP_CHECKSIG.
        assert_eq!(&decoded.script_pub_key[..3], &[0x76, 0xa9, 0x14]);
        assert_eq!(&decoded.script_pub_key[23..], &[0x88, 0xac]);
    }

    #[test]
    fn a_corrupted_base58_checksum_is_rejected() {
        // The last character is changed, so the checksum no longer matches.
        let corrupted = "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNb";
        assert!(BitcoinAddress::decode(corrupted, Network::Mainnet).is_err());
    }

    #[test]
    fn an_address_from_the_wrong_network_is_rejected() {
        assert!(BitcoinAddress::decode(GENESIS_P2PKH, Network::Testnet).is_err());
    }

    #[test]
    fn p2wpkh_encodes_to_the_bip173_vector() {
        let (program, expected) = p2wpkh_vector();

        let address = BitcoinAddress::p2wpkh(&program, Network::Mainnet).unwrap();
        assert_eq!(address.encode(), expected);

        let decoded = BitcoinAddress::decode(&expected, Network::Mainnet).unwrap();
        assert_eq!(decoded.address_type, AddressType::P2WPKH);
        assert_eq!(decoded.hash, program);
    }

    #[test]
    fn bech32_rejects_mixed_case_and_bad_checksums() {
        let (_, valid) = p2wpkh_vector();

        // Uppercase is allowed, mixed case is not.
        assert!(BitcoinAddress::decode(&valid.to_uppercase(), Network::Mainnet).is_ok());

        let mixed = format!("{}{}", &valid[..5].to_uppercase(), &valid[5..]);
        assert!(BitcoinAddress::decode(&mixed, Network::Mainnet).is_err());

        // Flipping a character breaks the checksum.
        let mut broken = valid.clone();
        let replacement = if valid.as_bytes()[10] == b'q' { 'p' } else { 'q' };
        broken.replace_range(10..11, &replacement.to_string());
        assert!(BitcoinAddress::decode(&broken, Network::Mainnet).is_err());

        // The human-readable part must match the network.
        assert!(BitcoinAddress::decode(&valid.replace("bc1", "tb1"), Network::Mainnet).is_err());
    }

    #[test]
    fn taproot_uses_bech32m_and_round_trips() {
        // BIP-350 test vector: a P2TR address for an all-zero output key.
        let output_key = [0u8; 32];
        let address = BitcoinAddress::p2tr(&output_key, Network::Mainnet).unwrap();
        let encoded = address.encode();

        assert!(encoded.starts_with("bc1p"), "got {encoded}");
        assert_eq!(encoded.len(), 62);

        let decoded = BitcoinAddress::decode(&encoded, Network::Mainnet).unwrap();
        assert_eq!(decoded.address_type, AddressType::P2TR);
        assert_eq!(decoded.hash, output_key);
    }

    #[test]
    fn a_taproot_address_encoded_as_bech32_is_rejected() {
        let output_key = [1u8; 32];

        // Build the same program but with the bech32 checksum, which BIP-350
        // forbids for witness version 1.
        let mut data = vec![1u8];
        data.extend_from_slice(&output_key);
        let grouped = bech32_impl::convert_bits(&data, 8, 5, true).unwrap();
        let wrong = bech32_impl::encode("bc", &grouped, Bech32Variant::Bech32);

        assert!(BitcoinAddress::decode(&wrong, Network::Mainnet).is_err());
    }

    #[test]
    fn p2wsh_commits_to_sha256_of_the_script() {
        let script = vec![0x51u8]; // OP_TRUE
        let address = BitcoinAddress::p2wsh(&script, Network::Mainnet).unwrap();

        assert_eq!(address.hash, sha256_hash(&script));
        assert_eq!(address.script_pub_key[..2].to_vec(), vec![0x00u8, 0x20]);

        let decoded = BitcoinAddress::decode(&address.encode(), Network::Mainnet).unwrap();
        assert_eq!(decoded.address_type, AddressType::P2WSH);
        assert_eq!(decoded.hash, address.hash);
    }

    #[test]
    fn compact_sizes_are_canonical() {
        assert_eq!(varint(0), vec![0x00]);
        assert_eq!(varint(0xfc), vec![0xfc]);
        assert_eq!(varint(0xfd), vec![0xfd, 0xfd, 0x00]);
        assert_eq!(varint(0xffff), vec![0xfd, 0xff, 0xff]);
        assert_eq!(varint(0x1_0000), vec![0xfe, 0x00, 0x00, 0x01, 0x00]);

        let mut cursor = 0;
        assert_eq!(read_varint(&[0xfd, 0xfd, 0x00], &mut cursor).unwrap(), 0xfd);

        // A value that fits in one byte must not be padded out.
        let mut cursor = 0;
        assert!(read_varint(&[0xfd, 0x01, 0x00], &mut cursor).is_err());
    }

    #[test]
    fn a_transaction_round_trips_through_serialization() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input_hex(
            "b3e2bcc9a5f8a1e4a4c8f2b3a5d0c0e0f0a0b0c0d0e0f0a0b0c0d0e0f0a0b0c0",
            0,
        )
        .unwrap();
        tx.add_output(
            50_000,
            BitcoinAddress::p2wpkh(&[7u8; 20], Network::Mainnet)
                .unwrap()
                .script_pub_key,
        );

        let serialized = tx.serialize();
        let parsed = BitcoinTransaction::deserialize(&serialized).unwrap();

        assert_eq!(parsed, tx);
        assert_eq!(parsed.txid(), tx.txid());
        assert_eq!(parsed.txid().len(), 64);
    }

    #[test]
    fn a_segwit_transaction_keeps_its_witness_and_its_txid() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([1u8; 32], 0, Vec::new());
        tx.add_output(1_000, vec![0x51]);
        tx.inputs[0].witness = vec![vec![0xaa; 71], vec![0x02; 33]];

        let legacy = tx.serialize_legacy();
        let full = tx.serialize();

        // The SegWit form carries the marker and flag and is longer.
        assert_eq!(&full[4..6], &[0x00, 0x01]);
        assert!(full.len() > legacy.len());

        // The transaction id commits only to the legacy form, so it is stable
        // whether or not the witness is attached.
        let mut without_witness = tx.clone();
        without_witness.inputs[0].witness.clear();
        assert_eq!(tx.txid(), without_witness.txid());

        let parsed = BitcoinTransaction::deserialize(&full).unwrap();
        assert_eq!(parsed.inputs[0].witness, tx.inputs[0].witness);
        assert_eq!(parsed.txid(), tx.txid());
    }

    #[test]
    fn deserialization_rejects_trailing_bytes_and_truncation() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([2u8; 32], 1, Vec::new());
        tx.add_output(1, vec![0x51]);

        let mut extended = tx.serialize();
        extended.push(0x00);
        assert!(BitcoinTransaction::deserialize(&extended).is_err());

        let truncated = &tx.serialize()[..tx.serialize().len() - 2];
        assert!(BitcoinTransaction::deserialize(truncated).is_err());
    }

    #[test]
    fn vsize_discounts_witness_data() {
        let mut legacy = BitcoinTransaction::new();
        legacy.add_input([3u8; 32], 0, vec![0x51]);
        legacy.add_output(1, vec![0x51]);

        let mut segwit = legacy.clone();
        segwit.inputs[0].witness = vec![vec![0u8; 72]];

        // The witness adds bytes but they are discounted by a quarter.
        assert!(segwit.vsize() < segwit.serialize().len());
        assert_eq!(legacy.vsize(), legacy.serialize().len());
    }

    #[test]
    fn bip143_sighash_matches_the_published_vector() {
        // BIP-143's worked example: a two-input transaction whose second input
        // is a P2WPKH output worth 6 BTC.
        fn internal(txid_hex: &str) -> [u8; 32] {
            // The serialized transaction carries the internal byte order, so
            // the hex is used exactly as written.
            let mut out = [0u8; 32];
            out.copy_from_slice(&hex::decode(txid_hex).unwrap());
            out
        }

        let mut tx = BitcoinTransaction::new();
        tx.version = 1;
        tx.add_input(
            internal("fff7f7881a8099afa6940d42d1e7f6362bec38171ea3edf433541db4e4ad969f"),
            0,
            Vec::new(),
        );
        tx.inputs[0].sequence = 0xffffffee;
        tx.add_input(
            internal("ef51e1b804cc89d182d279655c3aa89e815b1b309fe287d9b2b55d57b90ec68a"),
            1,
            Vec::new(),
        );
        tx.outputs.push(TxOutput {
            value: 112_340_000,
            script_pub_key: hex::decode("76a9148280b37df378db99f66f85c95a783a76ac7a6d5988ac").unwrap(),
        });
        tx.outputs.push(TxOutput {
            value: 223_450_000,
            script_pub_key: hex::decode("76a9143bde42dbee7e4dbe6a21b2d50ce2f0167faa815988ac").unwrap(),
        });
        // The example's lock time is 0x00000011.
        tx.lock_time = 17;

        // The scriptCode is the raw script; the serializer adds its length.
        let script_code =
            hex::decode("76a9141d0f172a0ecb48aee1be1f2687d2963ae33f71a188ac").unwrap();

        let sighash = segwit_v0_sighash(&tx, 1, &script_code, 600_000_000, sighash::ALL).unwrap();

        assert_eq!(
            hex::encode(sighash),
            "c37af31116d1b27caf68aae9e3ac82f1477929014d5b917657d0eb49478cb670"
        );
    }

    #[test]
    fn p2wpkh_script_code_is_the_raw_script() {
        let hash = [9u8; 20];
        let script = p2wpkh_script_code(&hash).unwrap();

        // No length prefix, and the canonical P2WPKH shape.
        assert_eq!(script.len(), 25);
        assert_eq!(&script[..3], &[0x76, 0xa9, 0x14]);
        assert_eq!(&script[23..], &[0x88, 0xac]);
        assert_eq!(&script[3..23], &hash);

        // The wrong length is refused.
        assert!(p2wpkh_script_code(&[0u8; 19]).is_err());
    }

    #[test]
    fn sighash_none_and_single_zero_the_sequence_and_output_hashes() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([1u8; 32], 0, Vec::new());
        tx.add_input([2u8; 32], 0, Vec::new());
        tx.add_output(1_000, vec![0x51]);
        tx.add_output(2_000, vec![0x51]);

        // SINGLE commits to the paired output, so the digest changes when that
        // output changes but not when an earlier one does.
        let base = segwit_v0_sighash(&tx, 1, &[0x51], 0, sighash::SINGLE).unwrap();

        let mut earlier_changed = tx.clone();
        earlier_changed.outputs[0].value = 9_999;
        assert_eq!(
            segwit_v0_sighash(&earlier_changed, 1, &[0x51], 0, sighash::SINGLE).unwrap(),
            base
        );

        let mut paired_changed = tx.clone();
        paired_changed.outputs[1].value = 9_999;
        assert_ne!(
            segwit_v0_sighash(&paired_changed, 1, &[0x51], 0, sighash::SINGLE).unwrap(),
            base
        );

        // NONE commits to no output at all.
        let none = segwit_v0_sighash(&tx, 1, &[0x51], 0, sighash::NONE).unwrap();
        let mut none_changed = tx.clone();
        none_changed.outputs[1].value = 9_999;
        assert_eq!(
            segwit_v0_sighash(&none_changed, 1, &[0x51], 0, sighash::NONE).unwrap(),
            none
        );

        // ANYONECANPAY commits to neither prevouts nor sequences.
        let acp = segwit_v0_sighash(&tx, 1, &[0x51], 0, sighash::ALL | sighash::ANYONECANPAY)
            .unwrap();
        let mut acp_changed = tx.clone();
        acp_changed.inputs[0].sequence = 1;
        acp_changed.inputs[0].previous_output.vout = 7;
        assert_eq!(
            segwit_v0_sighash(
                &acp_changed,
                1,
                &[0x51],
                0,
                sighash::ALL | sighash::ANYONECANPAY
            )
            .unwrap(),
            acp
        );
    }

    #[test]
    fn sighash_rejects_an_unknown_type() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([3u8; 32], 0, Vec::new());
        tx.add_output(1, vec![0x51]);

        assert!(segwit_v0_sighash(&tx, 0, &[0x51], 0, 0x04).is_err());
    }

    #[test]
    fn sighash_single_without_a_matching_output_is_rejected() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([4u8; 32], 0, Vec::new());
        tx.add_input([5u8; 32], 0, Vec::new());
        tx.add_output(1, vec![0x51]);

        // Input 1 has no output 1 to pair with.
        assert!(segwit_v0_sighash(&tx, 1, &[0x51], 0, sighash::SINGLE).is_err());
    }

    #[test]
    fn psbt_round_trips_with_unknown_fields_preserved() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([6u8; 32], 0, Vec::new());
        tx.add_output(1_000, vec![0x51]);

        let mut psbt = Psbt::new(tx);
        psbt.global.push((vec![0xfc, 0x01], b"proprietary".to_vec()));
        psbt.set_witness_utxo(
            0,
            &TxOutput {
                value: 2_000,
                script_pub_key: vec![0x00, 0x14, 1, 2, 3],
            },
        )
        .unwrap();
        psbt.add_partial_signature(0, &[0x02; 33], &[0x30; 71]).unwrap();

        let encoded = psbt.serialize();
        assert_eq!(&encoded[..5], &Psbt::MAGIC);

        let parsed = Psbt::deserialize(&encoded).unwrap();
        assert_eq!(parsed, psbt);
        assert_eq!(parsed.witness_utxo(0).unwrap().value, 2_000);
        assert_eq!(parsed.partial_signatures(0).len(), 1);

        // The base64 form carries the same bytes.
        let base64 = psbt.to_base64();
        assert_eq!(Psbt::from_base64(&base64).unwrap(), psbt);
    }

    #[test]
    fn psbt_rejects_a_missing_magic_and_a_duplicate_key() {
        assert!(Psbt::deserialize(b"not a psbt").is_err());

        let tx = BitcoinTransaction::new();
        let mut psbt = Psbt::new(tx);
        psbt.global.push((vec![0xfc, 0x01], b"a".to_vec()));
        psbt.global.push((vec![0xfc, 0x01], b"b".to_vec()));

        let encoded = psbt.serialize();
        assert!(Psbt::deserialize(&encoded).is_err());
    }

    #[test]
    fn psbt_finalize_refuses_a_signature_from_the_wrong_key() {
        let pubkey_hash = hash160(&[0x02; 33]);

        let mut tx = BitcoinTransaction::new();
        tx.add_input([7u8; 32], 0, Vec::new());
        tx.add_output(1_000, vec![0x51]);

        let mut psbt = Psbt::new(tx);
        psbt.set_witness_utxo(
            0,
            &TxOutput {
                value: 2_000,
                script_pub_key: {
                    let mut script = vec![0x00, 0x14];
                    script.extend_from_slice(&pubkey_hash);
                    script
                },
            },
        )
        .unwrap();

        // A signature attributed to a different key.
        psbt.add_partial_signature(0, &[0x03; 33], &[0x30; 71])
            .unwrap();

        assert!(psbt.finalize().is_err());
    }

    #[test]
    fn psbt_finalize_builds_a_witness_and_extracts() {
        let public_key = vec![0x02; 33];
        let pubkey_hash = hash160(&public_key);

        let mut tx = BitcoinTransaction::new();
        tx.add_input([8u8; 32], 0, Vec::new());
        tx.add_output(1_000, vec![0x51]);

        let mut psbt = Psbt::new(tx);
        psbt.set_witness_utxo(
            0,
            &TxOutput {
                value: 2_000,
                script_pub_key: {
                    let mut script = vec![0x00, 0x14];
                    script.extend_from_slice(&pubkey_hash);
                    script
                },
            },
        )
        .unwrap();
        psbt.add_partial_signature(0, &public_key, &[0x30; 71])
            .unwrap();

        psbt.finalize().unwrap();
        assert!(psbt.is_finalized());

        let signed = psbt.extract().unwrap();
        assert_eq!(signed.inputs[0].witness.len(), 2);
        assert_eq!(signed.inputs[0].witness[0], vec![0x30; 71]);
        assert_eq!(signed.inputs[0].witness[1], public_key);
    }

    #[test]
    fn extracting_an_unfinalized_psbt_is_refused() {
        let mut tx = BitcoinTransaction::new();
        tx.add_input([9u8; 32], 0, Vec::new());
        tx.add_output(1_000, vec![0x51]);

        let psbt = Psbt::new(tx);
        assert!(psbt.extract().is_err());
    }

    #[test]
    fn extended_public_keys_round_trip_and_derive_non_hardened_children() {
        // BIP-32 test vector 1, the master xpub.
        let xpub = "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8";

        let (version, key) = ExtendedPublicKey::decode(xpub).unwrap();
        assert_eq!(version, 0x0488_b21e);
        assert_eq!(key.depth, 0);
        assert_eq!(key.child_number, 0);

        // Re-encoding reproduces the input exactly.
        assert_eq!(key.encode(version), xpub);

        // A hardened index cannot be derived from a public key.
        assert!(key.derive_child(0x8000_0000).is_err());

        let child = key.derive_child(0).unwrap();
        assert_eq!(child.depth, 1);
        assert_eq!(child.child_number, 0);
        assert_eq!(child.parent_fingerprint, fingerprint(&key.public_key));
    }

    #[test]
    fn extended_key_decoding_rejects_a_bad_checksum() {
        let mut xpub = "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8".to_string();
        xpub.replace_range(xpub.len() - 1.., "9");

        assert!(ExtendedPublicKey::decode(&xpub).is_err());
    }

    #[test]
    fn stacks_addresses_decode_and_re_encode() {
        // The canonical c32check vector: version 22 with a zero payload.
        let expected = "SP000000000000000000002Q6VF78";
        let address = StacksAddress::decode(expected).unwrap();

        assert_eq!(address.version, stacks_versions::MAINNET_SINGLE_SIG);
        assert_eq!(address.hash_bytes, vec![0u8; 20]);
        assert!(address.is_mainnet());
        assert!(!address.is_contract());
        assert_eq!(address.encode(), expected);
    }

    #[test]
    fn stacks_address_decoding_rejects_corruption_and_unknown_versions() {
        let valid = "SP000000000000000000002Q6VF78";

        let mut corrupted = valid.to_string();
        corrupted.replace_range(valid.len() - 1.., "9");
        assert!(StacksAddress::decode(&corrupted).is_err());

        assert!(StacksAddress::decode("not-an-address").is_err());

        // Version 30 is not assigned to any Stacks address form.
        let unknown = format!("S{}", c32::encode_check(30, &[0u8; 20]));
        assert!(StacksAddress::decode(&unknown).is_err());
    }

    #[test]
    fn stacks_address_maps_to_the_controlling_bitcoin_address() {
        let address = StacksAddress::decode("SP000000000000000000002Q6VF78").unwrap();
        let bitcoin = address.to_bitcoin_address().unwrap();

        // A zero hash with the mainnet P2PKH version byte.
        assert_eq!(bitcoin.hash, vec![0u8; 20]);
        assert_eq!(bitcoin.network, Network::Mainnet);
        assert!(bitcoin.encode().starts_with('1'));
    }

    #[test]
    fn brc20_operations_serialize_and_parse() {
        let deploy = Brc20Operation::Deploy {
            tick: "ordi".to_string(),
            max: "21000000".to_string(),
            lim: Some("1000".to_string()),
            decimals: Some("18".to_string()),
        };
        deploy.validate().unwrap();

        let body = deploy.to_inscription_body();
        assert_eq!(Brc20Operation::from_inscription_body(&body).unwrap(), deploy);

        let mint = Brc20Operation::Mint {
            tick: "ordi".to_string(),
            amount: "100".to_string(),
        };
        mint.validate().unwrap();
        let body = mint.to_inscription_body();
        assert_eq!(Brc20Operation::from_inscription_body(&body).unwrap(), mint);

        let transfer = Brc20Operation::Transfer {
            tick: "ordi".to_string(),
            amount: "50".to_string(),
        };
        let body = transfer.to_inscription_body();
        assert_eq!(
            Brc20Operation::from_inscription_body(&body).unwrap(),
            transfer
        );
    }

    #[test]
    fn brc20_validation_rejects_malformed_fields() {
        // Leading zeros are not accepted by the indexers.
        assert!(Brc20Operation::Mint {
            tick: "ordi".to_string(),
            amount: "0100".to_string(),
        }
        .validate()
        .is_err());

        // A non-numeric amount is invalid.
        assert!(Brc20Operation::Transfer {
            tick: "ordi".to_string(),
            amount: "1.5".to_string(),
        }
        .validate()
        .is_err());

        // A ticker is at most four bytes.
        assert!(Brc20Operation::Mint {
            tick: "toolong".to_string(),
            amount: "1".to_string(),
        }
        .validate()
        .is_err());

        // Decimals above 18 are rejected.
        assert!(Brc20Operation::Deploy {
            tick: "ordi".to_string(),
            max: "100".to_string(),
            lim: None,
            decimals: Some("19".to_string()),
        }
        .validate()
        .is_err());
    }

    #[test]
    fn inscription_envelopes_use_the_ordinals_tags() {
        let inscription = Inscription {
            content_type: "text/plain;charset=utf-8".to_string(),
            body: b"hello".to_vec(),
            parents: Vec::new(),
            metadata: None,
        };

        let script = inscription.to_envelope();

        assert_eq!(&script[..2], &[0x00, 0x63]); // OP_FALSE OP_IF
        assert_eq!(*script.last().unwrap(), 0x68); // OP_ENDIF
        assert!(script.windows(3).any(|w| w == b"ord"));
        assert!(script.ends_with(&[0x05, b'h', b'e', b'l', b'l', b'o', 0x68]));
    }

    #[test]
    fn push_data_uses_the_shortest_opcode() {
        let mut script = Vec::new();
        push_data(&mut script, &[]);
        assert_eq!(script, vec![0x00]);

        script.clear();
        push_data(&mut script, &[1, 2, 3]);
        assert_eq!(script, vec![0x03, 1, 2, 3]);

        script.clear();
        push_data(&mut script, &[0u8; 76]);
        assert_eq!(script[0], 0x4c);
        assert_eq!(script[1], 76);
        assert_eq!(script.len(), 78);
    }

    #[test]
    fn multisig_builds_a_sorted_redeem_script_and_addresses() {
        let keys = vec![vec![0x03; 33], vec![0x02; 33], vec![0x03; 33]];

        // The duplicate key is rejected.
        assert!(MultiSigWallet::new(2, keys).is_err());

        let wallet = MultiSigWallet::new(2, vec![vec![0x03; 33], vec![0x02; 33]]).unwrap();

        assert_eq!(wallet.threshold, 2);
        assert_eq!(wallet.redeem_script[0], 0x52); // OP_2
        // The key-count opcode sits immediately before OP_CHECKMULTISIG.
        assert_eq!(
            wallet.redeem_script[wallet.redeem_script.len() - 2],
            0x52
        );
        assert_eq!(*wallet.redeem_script.last().unwrap(), 0xae); // OP_CHECKMULTISIG

        // Keys are sorted, so 0x02 comes before 0x03.
        assert_eq!(wallet.public_keys[0], vec![0x02; 33]);

        let p2sh = wallet.p2sh_address(Network::Mainnet).unwrap();
        assert_eq!(p2sh.address_type, AddressType::P2SH);

        let p2wsh = wallet.p2wsh_address(Network::Mainnet).unwrap();
        assert_eq!(p2wsh.address_type, AddressType::P2WSH);
    }

    #[test]
    fn multisig_rejects_impossible_thresholds() {
        assert!(MultiSigWallet::new(0, vec![vec![0x02; 33]]).is_err());
        assert!(MultiSigWallet::new(2, vec![vec![0x02; 33]]).is_err());
        assert!(MultiSigWallet::new(1, Vec::new()).is_err());

        // A key that is not a compressed public key.
        assert!(MultiSigWallet::new(1, vec![vec![0x04; 65]]).is_err());
    }

    #[test]
    fn multisig_script_sig_starts_with_the_dummy_element() {
        let wallet = MultiSigWallet::new(2, vec![vec![0x02; 33], vec![0x03; 33]]).unwrap();

        let script = wallet
            .build_script_sig(&[vec![0x30; 71], vec![0x30; 72]])
            .unwrap();

        // OP_0, then two pushes.
        assert_eq!(script[0], 0x00);
        assert_eq!(script[1], 71);
        assert_eq!(script[73], 72);

        // Too few signatures is refused.
        assert!(wallet.build_script_sig(&[vec![0x30; 71]]).is_err());
    }

    #[test]
    fn multisig_witness_ends_with_the_redeem_script() {
        let wallet = MultiSigWallet::new(1, vec![vec![0x02; 33], vec![0x03; 33]]).unwrap();
        let witness = wallet.build_witness(&[vec![0x30; 71]]).unwrap();

        assert_eq!(witness.len(), 3);
        assert!(witness[0].is_empty());
        assert_eq!(witness[2], wallet.redeem_script);
    }

    #[test]
    fn multisig_counts_only_signatures_that_verify() {
        use secp256k1::{Message, Secp256k1, SecretKey};

        let secp = Secp256k1::new();
        let secret = SecretKey::from_slice(&[9u8; 32]).unwrap();
        let public = secret.public_key(&secp).serialize().to_vec();

        let wallet = MultiSigWallet::new(1, vec![public]).unwrap();

        let digest = [42u8; 32];
        let message = Message::from_slice(&digest).unwrap();
        let signature = secp
            .sign_ecdsa(&message, &secret)
            .serialize_der()
            .to_vec();

        assert_eq!(
            wallet.count_valid_signatures(&digest, &[signature.clone()]).unwrap(),
            1
        );

        // A signature over a different digest does not count.
        let other_digest = [43u8; 32];
        assert_eq!(
            wallet.count_valid_signatures(&other_digest, &[signature]).unwrap(),
            0
        );
    }

    /// Build an invoice string from a human-readable part and 5-bit groups.
    fn build_bolt11(hrp: &str, groups: &[u8], signature: &[u8; 65]) -> String {
        let mut all = groups.to_vec();
        all.extend_from_slice(&bech32_impl::convert_bits(signature, 8, 5, true).unwrap());
        bech32_impl::encode(hrp, &all, Bech32Variant::Bech32)
    }

    /// A BOLT-11 field: either a byte string or an integer.
    enum Bolt11Field {
        Bytes(Vec<u8>),
        Integer(u64),
    }

    /// A BOLT-11 payload: a 35-bit timestamp followed by tagged fields.
    ///
    /// The fields are not byte aligned, so the payload is assembled as a bit
    /// stream and only then grouped into fives.
    fn bolt11_payload(timestamp: u64, fields: &[(u8, Bolt11Field)]) -> Vec<u8> {
        let mut bits: Vec<bool> = Vec::new();

        fn push(value: u64, count: u32, bits: &mut Vec<bool>) {
            for index in (0..count).rev() {
                bits.push((value >> index) & 1 == 1);
            }
        }

        push(timestamp, 35, &mut bits);

        for (field_type, field) in fields {
            let groups: Vec<u8> = match field {
                // Byte data is regrouped into fives with zero padding, the same
                // way a bech32 witness program is.
                Bolt11Field::Bytes(data) => bech32_impl::convert_bits(data, 8, 5, true).unwrap(),
                // An integer occupies the fewest whole groups that fit it.
                Bolt11Field::Integer(value) => {
                    let significant = 64 - value.leading_zeros();
                    let count = (significant.max(1) as usize).div_ceil(5);

                    (0..count)
                        .rev()
                        .map(|index| ((value >> (5 * index)) & 0x1f) as u8)
                        .collect()
                }
            };

            push(u64::from(*field_type), 5, &mut bits);
            push(groups.len() as u64, 10, &mut bits);
            for group in groups {
                push(u64::from(group), 5, &mut bits);
            }
        }

        while bits.len() % 5 != 0 {
            bits.push(false);
        }

        bits.chunks(5)
            .map(|chunk| chunk.iter().fold(0u8, |acc, bit| (acc << 1) | u8::from(*bit)))
            .collect()
    }

    #[test]
    fn bolt11_decodes_a_well_formed_invoice() {
        let payment_hash = [0x11u8; 32];
        let payee = [0x02u8; 33];
        let mut description = Vec::new();
        description.extend_from_slice(b"coffee");

        let payload = bolt11_payload(
            1_700_000_000,
            &[
                (1, Bolt11Field::Bytes(payment_hash.to_vec())),
                (13, Bolt11Field::Bytes(description)),
                (19, Bolt11Field::Bytes(payee.to_vec())),
                (6, Bolt11Field::Integer(60)),
            ],
        );

        let invoice = build_bolt11("lnbc2500u", &payload, &[0x30u8; 65]);
        let decoded = decode_bolt11(&invoice).unwrap();

        assert_eq!(decoded.currency, "lnbc");
        assert_eq!(decoded.amount_msat, Some(250_000_000));
        assert_eq!(decoded.timestamp, 1_700_000_000);
        assert_eq!(decoded.payment_hash, payment_hash);
        assert_eq!(decoded.payee, payee);
        assert_eq!(decoded.description.as_deref(), Some("coffee"));
        assert_eq!(decoded.expiry_seconds, 60);
    }

    #[test]
    fn bolt11_rejects_a_corrupted_invoice() {
        let payload = bolt11_payload(
            1_700_000_000,
            &[
                (1, Bolt11Field::Bytes(vec![0x22u8; 32])),
                (19, Bolt11Field::Bytes(vec![0x02u8; 33])),
            ],
        );
        let invoice = build_bolt11("lnbc1m", &payload, &[0x30u8; 65]);

        // Flipping a character breaks the bech32 checksum.
        let mut corrupted = invoice.clone();
        let position = corrupted.len() / 2;
        let replacement = if corrupted.as_bytes()[position] == b'q' { 'p' } else { 'q' };
        corrupted.replace_range(position..position + 1, &replacement.to_string());
        assert!(decode_bolt11(&corrupted).is_err());

        assert!(decode_bolt11("not-an-invoice").is_err());

        // An invoice with no payment hash is rejected.
        let without_hash = bolt11_payload(
            1_700_000_000,
            &[(19, Bolt11Field::Bytes(vec![0x02u8; 33]))],
        );
        let invoice = build_bolt11("lnbc", &without_hash, &[0x30u8; 65]);
        assert!(decode_bolt11(&invoice).is_err());

        // So is one with no payee key.
        let without_payee = bolt11_payload(
            1_700_000_000,
            &[(1, Bolt11Field::Bytes(vec![0x22u8; 32]))],
        );
        let invoice = build_bolt11("lnbc", &without_payee, &[0x30u8; 65]);
        assert!(decode_bolt11(&invoice).is_err());
    }

    #[test]
    fn bolt11_amounts_and_expiry_are_interpreted_correctly() {
        let payload = bolt11_payload(
            1_000,
            &[
                (1, Bolt11Field::Bytes(vec![0x33u8; 32])),
                (19, Bolt11Field::Bytes(vec![0x02u8; 33])),
                (6, Bolt11Field::Integer(3_600)),
            ],
        );

        // A unit scales the value into milli-satoshi; no unit means bitcoin.
        let units = [
            ("lnbc2", Some(200_000_000_000u64)),
            ("lnbc2m", Some(200_000_000)),
            ("lnbc2u", Some(200_000)),
            ("lnbc2n", Some(200)),
        ];

        for (hrp, expected) in units {
            let invoice = build_bolt11(hrp, &payload, &[0u8; 65]);
            let decoded = decode_bolt11(&invoice).unwrap();
            assert_eq!(decoded.amount_msat, expected, "hrp {hrp}");
            assert_eq!(decoded.expiry_seconds, 3_600);
        }

        // An invoice with no amount is valid; the payer chooses.
        let invoice = build_bolt11("lnbc", &payload, &[0u8; 65]);
        let decoded = decode_bolt11(&invoice).unwrap();
        assert_eq!(decoded.amount_msat, None);
        assert!(!decoded.is_expired(1_000));
        assert!(decoded.is_expired(1_000 + 3_601));
    }

    #[test]
    fn bolt11_signing_groups_zero_the_signature() {
        let payload = bolt11_payload(
            1_000,
            &[
                (1, Bolt11Field::Bytes(vec![0x44u8; 32])),
                (19, Bolt11Field::Bytes(vec![0x02u8; 33])),
            ],
        );
        let invoice = build_bolt11("lnbc", &payload, &[0x30u8; 65]);
        let decoded = decode_bolt11(&invoice).unwrap();

        let signing = decoded.signing_groups();
        let payload_len = signing.len() - 104;

        assert_eq!(signing.len(), decoded.data_groups.len());
        assert!(signing[..payload_len]
            .iter()
            .eq(decoded.data_groups[..payload_len].iter()));
        assert!(signing[payload_len..].iter().all(|group| *group == 0));
    }

    #[test]
    fn hash160_matches_the_published_vector() {
        // HASH160 of the genesis coinbase public key.
        let pubkey = hex::decode(SATOSHI_PUBKEY).unwrap();
        let hash = hash160(&pubkey);

        // The genesis P2PKH address decodes to the same 20 bytes.
        let address = BitcoinAddress::decode(GENESIS_P2PKH, Network::Mainnet).unwrap();
        assert_eq!(hash, address.hash);
    }
}
