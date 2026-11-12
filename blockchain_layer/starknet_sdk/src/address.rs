//! Starknet Address Types
//! 
//! Implementation of Starknet address formats and validation.

use std::fmt;
use hex;
use starknet_crypto::{pedersen_hash, FieldElement};

/// The `STARKNET_CONTRACT_ADDRESS` felt. Starknet mixes this domain separator
/// into every contract address so a hash of one kind cannot be reinterpreted as
/// an address of another.
fn contract_address_prefix() -> Result<FieldElement, AddressError> {
    FieldElement::from_hex_be("0x535441524b4e45545f434f4e54524143545f41444452455353")
        .map_err(|_| AddressError::InvalidFormat)
}

/// Reject bytes at or above the field modulus, which are not valid felts.
fn field_from_bytes(bytes: &[u8; 32]) -> Result<FieldElement, AddressError> {
    FieldElement::from_bytes_be(bytes).map_err(|_| AddressError::TooLong)
}

/// Starknet's `compute_hash_on_elements`: a Pedersen chain seeded and terminated
/// with the element count, so `[a, b]` and `[b, a]` differ and a list cannot
/// collide with its own concatenation.
fn compute_hash_on_elements(elements: &[FieldElement]) -> FieldElement {
    let count = FieldElement::from(elements.len() as u64);
    let mut hash = count;
    for element in elements {
        hash = pedersen_hash(&hash, element);
    }
    pedersen_hash(&hash, &count)
}

/// Starknet's `calculate_contract_address`: the address a contract will have
/// once deployed, so it can be known before deployment and is unchanged after.
pub fn compute_contract_address(
    salt: FieldElement,
    deployer: FieldElement,
    class_hash: FieldElement,
    constructor_calldata_hash: FieldElement,
) -> Result<FieldElement, AddressError> {
    let hash = pedersen_hash(&contract_address_prefix()?, &deployer);
    let hash = pedersen_hash(&hash, &salt);
    let hash = pedersen_hash(&hash, &class_hash);
    Ok(pedersen_hash(&hash, &constructor_calldata_hash))
}

/// Starknet Address (Felt252)
#[derive(Clone, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
pub struct StarknetAddress {
    /// The 252-bit felt value
    value: [u8; 32],
    /// Whether this is a contract address
    is_contract: bool,
}

impl StarknetAddress {
    /// Create from hex string
    pub fn from_hex(hex_str: &str) -> Result<Self, AddressError> {
        let bytes = hex::decode(hex_str.trim_start_matches("0x"))
            .map_err(|_| AddressError::InvalidHex)?;
        
        if bytes.len() > 32 {
            return Err(AddressError::TooLong);
        }
        
        let mut value = [0u8; 32];
        value[32 - bytes.len()..].copy_from_slice(&bytes);
        
        Ok(Self { value, is_contract: false })
    }
    
    /// Create from felt252 value
    pub fn from_felt252(mut value: [u8; 32]) -> Self {
        // Validate value is within field
        // Starknet field: p = 2^251 + 17 * 2^192 + 1
        let max_felt: [u8; 32] = [
            0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        ];
        
        // Validate within field
        let mut is_contract = false;
        for i in 0..32 {
            if value[i] < max_felt[i] {
                break;
            } else if value[i] > max_felt[i] {
                // Set to zero if out of field
                value = [0u8; 32];
                break;
            }
        }
        
        // Contract addresses are deterministic (hash of contract code)
        if value.iter().all(|&b| b == 0) {
            is_contract = false;
        }
        
        Self { value, is_contract }
    }
    
    /// Derive the account address for a single-signer account.
    ///
    /// Starknet accounts have no fixed address: an address is the hash of the
    /// account class plus its constructor arguments, which is why it can be
    /// computed before deployment and stays the same afterwards. This applies
    /// the standard formula with an empty salt and the single `public_key`
    /// constructor argument used by the OpenZeppelin and Argent account classes,
    /// so the caller must supply that class's real hash. Deriving from the
    /// public key alone, as this used to, produced an address that no Starknet
    /// node would ever agree with.
    pub fn from_public_key(
        public_key: &[u8; 32],
        class_hash: &[u8; 32],
    ) -> Result<Self, AddressError> {
        let calldata_hash = compute_hash_on_elements(&[field_from_bytes(public_key)?]);
        let value = compute_contract_address(
            FieldElement::ZERO,
            FieldElement::ZERO,
            field_from_bytes(class_hash)?,
            calldata_hash,
        )?;
        Ok(Self { value: value.to_bytes_be(), is_contract: true })
    }
    
    /// Get as hex string (0x...)
    pub fn to_hex(&self) -> String {
        format!("0x{}", hex::encode(self.value))
    }
    
    /// Get as felt252 bytes
    pub fn to_felt252(&self) -> [u8; 32] {
        self.value
    }
    
    /// Get short address (first 10 chars)
    pub fn to_short(&self) -> String {
        let hex_str = hex::encode(self.value);
        format!("0x{}", &hex_str[..10])
    }
    
    /// Validate address format
    pub fn is_valid(&self) -> bool {
        // Check not zero
        if self.value.iter().all(|&b| b == 0) {
            return false;
        }
        
        // Check within field (simplified)
        // In production: verify < 2^251 + 17*2^192 + 1
        true
    }
    
    /// Set contract flag
    pub fn set_contract(&mut self, is_contract: bool) {
        self.is_contract = is_contract;
    }
    
    /// Check if contract
    pub fn is_contract(&self) -> bool {
        self.is_contract
    }
}

impl fmt::Debug for StarknetAddress {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "StarknetAddress({})", self.to_hex())
    }
}

impl fmt::Display for StarknetAddress {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.to_hex())
    }
}

impl From<[u8; 32]> for StarknetAddress {
    fn from(value: [u8; 32]) -> Self {
        Self::from_felt252(value)
    }
}

impl From<&str> for StarknetAddress {
    fn from(s: &str) -> Self {
        Self::from_hex(s).expect("Invalid Starknet address")
    }
}

impl From<String> for StarknetAddress {
    fn from(s: String) -> Self {
        Self::from_hex(&s).expect("Invalid Starknet address")
    }
}

/// Address errors
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AddressError {
    InvalidHex,
    TooLong,
    InvalidFormat,
    ZeroAddress,
}

impl fmt::Display for AddressError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AddressError::InvalidHex => write!(f, "Invalid hex string"),
            AddressError::TooLong => write!(f, "Address too long"),
            AddressError::InvalidFormat => write!(f, "Invalid address format"),
            AddressError::ZeroAddress => write!(f, "Zero address not allowed"),
        }
    }
}

impl std::error::Error for AddressError {}

/// Starknet chain IDs
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StarknetChainId {
    /// Mainnet
    Mainnet,
    /// Goerli testnet
    Goerli,
    /// Goerli 2 testnet
    Goerli2,
    /// Sepolia testnet
    Sepolia,
    /// Local development
    Local,
}

impl StarknetChainId {
    /// Get chain ID as felt
    pub fn to_felt(&self) -> [u8; 32] {
        match self {
            StarknetChainId::Mainnet => {
                let mut id = [0u8; 32];
                id[31] = 0x01; // SN_MAINNET
                id
            }
            StarknetChainId::Goerli => {
                let mut id = [0u8; 32];
                id[31] = 0x03; // SN_GOERLI
                id
            }
            StarknetChainId::Goerli2 => {
                let mut id = [0u8; 32];
                id[31] = 0x05; // SN_GOERLI2
                id
            }
            StarknetChainId::Sepolia => {
                let mut id = [0u8; 32];
                id[31] = 0x06; // SN_SEPOLIA
                id
            }
            StarknetChainId::Local => {
                let mut id = [0u8; 32];
                id[31] = 0x00; // Local
                id
            }
        }
    }
    
    /// Get RPC URL prefix
    pub fn rpc_prefix(&self) -> &'static str {
        match self {
            StarknetChainId::Mainnet => "mainnet",
            StarknetChainId::Goerli => "goerli",
            StarknetChainId::Goerli2 => "goerli2",
            StarknetChainId::Sepolia => "sepolia",
            StarknetChainId::Local => "localhost",
        }
    }
}

impl Default for StarknetChainId {
    fn default() -> Self {
        StarknetChainId::Mainnet
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fe(hex: &str) -> FieldElement {
        FieldElement::from_hex_be(hex).unwrap()
    }

    /// The canonical Pedersen vector published with starknet-crypto. This pins
    /// the hash the address formula is built on, so a wrong primitive is caught
    /// here rather than as an address that only fails against a live node.
    #[test]
    fn pedersen_hash_matches_canonical_vector() {
        let x = fe("0x03d937c035c878245caf64531a5756109c53068da139362728feb561405371cb");
        let y = fe("0x0208a0a10250e382e1e4bbe2880906c2791bf6275695e02fbbc6aeff9cd8b31a");
        let expected = fe("0x030e480bed5fe53fa909cc0f8c4d99b8f9f2c016be4c41e13a4848797979c662");
        assert_eq!(pedersen_hash(&x, &y), expected);
    }

    /// `compute_hash_on_elements` seeds and terminates with the element count,
    /// which is what stops `[a, b]` colliding with `[b, a]`.
    #[test]
    fn element_hash_is_order_sensitive() {
        let a = fe("0x1");
        let b = fe("0x2");
        assert_ne!(
            compute_hash_on_elements(&[a, b]),
            compute_hash_on_elements(&[b, a])
        );
    }

    /// An empty calldata list must still hash, and to the hash of the count
    /// alone rather than to zero.
    #[test]
    fn element_hash_of_empty_list_is_not_zero() {
        let empty = compute_hash_on_elements(&[]);
        assert_ne!(empty, FieldElement::ZERO);
        assert_eq!(empty, pedersen_hash(&FieldElement::ZERO, &FieldElement::ZERO));
    }

    #[test]
    fn account_address_is_deterministic_and_contract_flagged() {
        let key = [7u8; 32];
        let class_hash = [6u8; 32];
        let a = StarknetAddress::from_public_key(&key, &class_hash).unwrap();
        let b = StarknetAddress::from_public_key(&key, &class_hash).unwrap();
        assert_eq!(a, b);
        assert!(a.is_contract(), "an account address is a deployed contract");
        assert_ne!(a.to_felt252(), [0u8; 32]);
    }

    /// The class hash is part of the address, so two account classes with the
    /// same key land on different addresses. Ignoring it was the old bug.
    #[test]
    fn account_address_depends_on_class_hash() {
        let key = [7u8; 32];
        let one = StarknetAddress::from_public_key(&key, &[1u8; 32]).unwrap();
        let two = StarknetAddress::from_public_key(&key, &[2u8; 32]).unwrap();
        assert_ne!(one, two);
    }

    #[test]
    fn account_address_differs_per_key() {
        let class_hash = [6u8; 32];
        let a = StarknetAddress::from_public_key(&[1u8; 32], &class_hash).unwrap();
        let b = StarknetAddress::from_public_key(&[2u8; 32], &class_hash).unwrap();
        assert_ne!(a, b);
    }

    /// The address must land inside the field. The field modulus is just under
    /// 2^252, so the top four bits of a valid felt are always clear.
    #[test]
    fn account_address_is_within_the_field() {
        let addr = StarknetAddress::from_public_key(&[3u8; 32], &[4u8; 32]).unwrap();
        assert_eq!(addr.to_felt252()[0] & 0xF0, 0x00);
    }

    /// Bytes at or above the modulus are not felts and must be rejected rather
    /// than silently reduced.
    #[test]
    fn rejects_out_of_field_class_hash() {
        // 0x0909... is just above the modulus 0x0800...011, so it is not a felt.
        assert!(StarknetAddress::from_public_key(&[1u8; 32], &[9u8; 32]).is_err());
    }

    #[test]
    fn contract_address_is_stable_for_fixed_inputs() {
        let addr = compute_contract_address(
            FieldElement::ZERO,
            FieldElement::ZERO,
            fe("0x1"),
            fe("0x2"),
        )
        .unwrap();
        assert_eq!(
            addr,
            compute_contract_address(
                FieldElement::ZERO,
                FieldElement::ZERO,
                fe("0x1"),
                fe("0x2"),
            )
            .unwrap()
        );
        assert_ne!(
            addr,
            compute_contract_address(
                FieldElement::ZERO,
                FieldElement::ZERO,
                fe("0x1"),
                fe("0x3"),
            )
            .unwrap()
        );
    }
}
