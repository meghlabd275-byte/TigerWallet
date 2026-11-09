//! Smart contract interfaces for MasterWallet operations
//! Provides ABI encoding and contract interaction

use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use std::collections::HashMap;

/// Contract interface
#[derive(Debug, Clone)]
pub struct Contract {
    pub address: String,
    pub abi: Vec<Function>,
}

/// Function in a contract ABI
#[derive(Debug, Clone)]
pub struct Function {
    pub name: String,
    pub inputs: Vec<Parameter>,
    pub outputs: Vec<Parameter>,
    pub state_mutability: StateMutability,
}

/// Parameter in function signature
#[derive(Debug, Clone)]
pub struct Parameter {
    pub name: String,
    pub param_type: String,
    pub indexed: bool,
}

/// State mutability
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StateMutability {
    Pure,
    View,
    Nonpayable,
    Payable,
}

/// ERC-20 Token contract interface
pub fn erc20_abi() -> Vec<Function> {
    vec![
        Function {
            name: "name".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "symbol".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "decimals".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint8".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "totalSupply".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "balanceOf".to_string(),
            inputs: vec![Parameter { name: "owner".to_string(), param_type: "address".to_string(), indexed: false }],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "transfer".to_string(),
            inputs: vec![
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "bool".to_string(), indexed: false }],
            state_mutability: StateMutability::Nonpayable,
        },
        Function {
            name: "approve".to_string(),
            inputs: vec![
                Parameter { name: "spender".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "bool".to_string(), indexed: false }],
            state_mutability: StateMutability::Nonpayable,
        },
        Function {
            name: "transferFrom".to_string(),
            inputs: vec![
                Parameter { name: "from".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "bool".to_string(), indexed: false }],
            state_mutability: StateMutability::Nonpayable,
        },
        Function {
            name: "allowance".to_string(),
            inputs: vec![
                Parameter { name: "owner".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "spender".to_string(), param_type: "address".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
    ]
}

/// ERC-721 NFT contract interface
pub fn erc721_abi() -> Vec<Function> {
    vec![
        Function {
            name: "name".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "symbol".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "ownerOf".to_string(),
            inputs: vec![Parameter { name: "tokenId".to_string(), param_type: "uint256".to_string(), indexed: false }],
            outputs: vec![Parameter { name: "".to_string(), param_type: "address".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "balanceOf".to_string(),
            inputs: vec![Parameter { name: "owner".to_string(), param_type: "address".to_string(), indexed: false }],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "safeTransferFrom".to_string(),
            inputs: vec![
                Parameter { name: "from".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "tokenId".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![],
            state_mutability: StateMutability::Nonpayable,
        },
    ]
}

/// ERC-1155 Multi-token contract interface
pub fn erc1155_abi() -> Vec<Function> {
    vec![
        Function {
            name: "balanceOf".to_string(),
            inputs: vec![
                Parameter { name: "account".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "id".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "safeTransferFrom".to_string(),
            inputs: vec![
                Parameter { name: "from".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "id".to_string(), param_type: "uint256".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
                Parameter { name: "data".to_string(), param_type: "bytes".to_string(), indexed: false },
            ],
            outputs: vec![],
            state_mutability: StateMutability::Nonpayable,
        },
    ]
}

/// Contract registry for known contracts
pub struct ContractRegistry {
    contracts: HashMap<String, Contract>,
}

impl ContractRegistry {
    pub fn new() -> Self {
        let mut contracts = HashMap::new();
        
        // ERC-20
        contracts.insert("ERC20".to_string(), Contract {
            address: "".to_string(),
            abi: erc20_abi(),
        });
        
        // ERC-721
        contracts.insert("ERC721".to_string(), Contract {
            address: "".to_string(),
            abi: erc721_abi(),
        });
        
        // ERC-1155
        contracts.insert("ERC1155".to_string(), Contract {
            address: "".to_string(),
            abi: erc1155_abi(),
        });
        
        Self { contracts }
    }
    
    pub fn get(&self, name: &str) -> Option<&Contract> {
        self.contracts.get(name)
    }
    
    pub fn register(&mut self, name: &str, address: &str, abi: Vec<Function>) {
        self.contracts.insert(name.to_string(), Contract {
            address: address.to_string(),
            abi,
        });
    }
}

impl Default for ContractRegistry {
    fn default() -> Self {
        Self::new()
    }
}

/// ABI encoder for contract calls
pub struct ABIEncoder;

impl ABIEncoder {
    /// Encode function call data
    pub fn encode(function_name: &str, params: &[&dyn ParamEncoder]) -> Result<Vec<u8>, String> {

        let mut result = Vec::new();
        
        // Function selector (first 4 bytes of keccak256 of function signature)
        let selector = Self::function_selector(function_name);
        result.extend_from_slice(&selector);
        
        // Encode parameters. A malformed argument aborts the whole call
        // rather than producing calldata that would dispatch with the wrong
        // arguments.
        for param in params {
            let word = param.encode()?;
            result.extend_from_slice(&word);
        }
        
        Ok(result)
    }
    
    /// Generate function selector
    ///
    /// The selector is the first four bytes of the Keccak-256 hash of the
    /// canonical function signature. Taking raw bytes of the name, as this once
    /// did, produces selectors that collide across functions and match nothing
    /// on chain.
    pub fn function_selector(function_name: &str) -> [u8; 4] {
        let digest = Keccak256::digest(function_name.as_bytes());
        let mut selector = [0u8; 4];
        selector.copy_from_slice(&digest[..4]);
        selector
    }
}

/// Interface for parameter encoding.
///
/// Encoding is fallible. An earlier version returned a zero word for input it
/// could not parse, which silently turned a mistyped address into the zero
/// address: a token transfer would then burn the funds instead of failing. A
/// value longer than one word also underflowed `32 - len` and panicked. Both
/// now surface as errors that the caller must handle.
pub trait ParamEncoder {
    fn encode(&self) -> Result<Vec<u8>, String>;
}

/// Decodes a 0x-prefixed hex string into a single left-padded ABI word.
fn encode_word(hex_value: &str, kind: &str) -> Result<Vec<u8>, String> {
    let digits = hex_value
        .strip_prefix("0x")
        .or_else(|| hex_value.strip_prefix("0X"))
        .unwrap_or(hex_value);

    let bytes = hex::decode(digits)
        .map_err(|e| format!("{kind} is not valid hex ({hex_value}): {e}"))?;

    if bytes.len() > 32 {
        return Err(format!(
            "{kind} does not fit in one ABI word: {} bytes ({hex_value})",
            bytes.len()
        ));
    }

    let mut word = vec![0u8; 32];
    word[32 - bytes.len()..].copy_from_slice(&bytes);
    Ok(word)
}

/// Address parameter encoder
pub struct AddressEncoder(pub String);

impl ParamEncoder for AddressEncoder {
    fn encode(&self) -> Result<Vec<u8>, String> {
        let word = encode_word(&self.0, "address")?;
        // An address occupies exactly the low 20 bytes; anything longer would
        // silently drop the high bytes when the EVM reads the low 160 bits.
        if self.0.trim_start_matches("0x").len() / 2 > 20 {
            return Err(format!("address is longer than 20 bytes: {}", self.0));
        }
        Ok(word)
    }
}

/// Uint256 parameter encoder
pub struct Uint256Encoder(pub String);

impl ParamEncoder for Uint256Encoder {
    fn encode(&self) -> Result<Vec<u8>, String> {
        encode_word(&self.0, "uint256")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Selectors are fixed by the ABI specification, so these values pin the
    /// encoder to what real contracts dispatch on.
    #[test]
    fn function_selectors_match_the_published_abi() {
        let cases: [(&str, [u8; 4]); 6] = [
            ("transfer(address,uint256)", [0xa9, 0x05, 0x9c, 0xbb]),
            ("balanceOf(address)", [0x70, 0xa0, 0x82, 0x31]),
            ("approve(address,uint256)", [0x09, 0x5e, 0xa7, 0xb3]),
            ("transferFrom(address,address,uint256)", [0x23, 0xb8, 0x72, 0xdd]),
            ("totalSupply()", [0x18, 0x16, 0x0d, 0xdd]),
            ("decimals()", [0x31, 0x3c, 0xe5, 0x67]),
        ];
        for (signature, expected) in cases {
            assert_eq!(
                ABIEncoder::function_selector(signature),
                expected,
                "selector mismatch for {signature}"
            );
        }
    }

    #[test]
    fn selectors_are_distinct_for_distinct_signatures() {
        // The previous byte-prefix scheme mapped every signature sharing a
        // prefix to the same selector; hashing must not.
        let a = ABIEncoder::function_selector("transfer(address,uint256)");
        let b = ABIEncoder::function_selector("transfer(address,uint8)");
        assert_ne!(a, b);
    }

    /// A malformed address must fail the call instead of encoding the zero
    /// address, which would send tokens to a burn address.
    #[test]
    fn encode_rejects_a_malformed_address() {
        let err = ABIEncoder::encode(
            "transfer(address,uint256)",
            &[
                &AddressEncoder("0xZZZZ".to_string()),
                &Uint256Encoder("0x64".to_string()),
            ],
        )
        .expect_err("non-hex address must not encode");
        assert!(err.contains("address"), "unexpected error: {err}");
    }

    /// A value wider than one ABI word must be rejected. It previously
    /// underflowed the left-pad offset and panicked.
    #[test]
    fn encode_rejects_a_value_wider_than_one_word() {
        let too_wide = format!("0x{}", "ab".repeat(33));
        let err = ABIEncoder::encode(
            "transfer(address,uint256)",
            &[
                &AddressEncoder("0x000000000000000000000000000000000000dEaD".to_string()),
                &Uint256Encoder(too_wide),
            ],
        )
        .expect_err("33-byte argument must not encode");
        assert!(err.contains("one ABI word"), "unexpected error: {err}");
    }

    #[test]
    fn encode_rejects_an_over_long_address() {
        let over_long = format!("0x{}", "ab".repeat(21));
        let err = ABIEncoder::encode("transfer(address,uint256)", &[&AddressEncoder(over_long)])
            .expect_err("21-byte address must not encode");
        assert!(err.contains("longer than 20 bytes"), "unexpected error: {err}");
    }

    /// A bare hex string without the 0x prefix is still valid input.
    #[test]
    fn encode_accepts_an_unprefixed_value() {
        let encoded = ABIEncoder::encode(
            "transfer(address,uint256)",
            &[
                &AddressEncoder("000000000000000000000000000000000000dEaD".to_string()),
                &Uint256Encoder("64".to_string()),
            ],
        )
        .expect("unprefixed hex must encode");
        assert_eq!(encoded.len(), 4 + 32 + 32);
        // The address ends in ...dEaD, so its final two bytes are de ad.
        assert_eq!(encoded[4 + 30], 0xde);
        assert_eq!(encoded[4 + 31], 0xad);
        assert_eq!(encoded[4 + 32 + 31], 0x64);
    }

    #[test]
    fn encode_prefixes_the_selector_and_pads_arguments() {
        let address = "0x000000000000000000000000000000000000dEaD";
        let amount = "0x0000000000000000000000000000000000000000000000000000000000000064";
        let encoded = ABIEncoder::encode(
            "transfer(address,uint256)",
            &[&AddressEncoder(address.to_string()), &Uint256Encoder(amount.to_string())],
        )
        .expect("encoding a well-formed call must succeed");

        assert_eq!(&encoded[..4], &[0xa9, 0x05, 0x9c, 0xbb]);
        assert_eq!(encoded.len(), 4 + 32 + 32);
        // Left-padded address in the first word.
        assert_eq!(&encoded[4 + 12..4 + 32], &hex::decode("000000000000000000000000000000000000dead").unwrap()[..]);
        // Amount in the second word.
        assert_eq!(encoded[4 + 63], 0x64);
    }
}
