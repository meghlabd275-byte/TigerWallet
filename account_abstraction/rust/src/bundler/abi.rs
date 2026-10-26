//! ABI encoding for ERC-4337 v0.7 `PackedUserOperation`.
//!
//! The EntryPoint packs the gas fields into two `bytes32` words, so the
//! off-chain encoding has to do the same or the hash it produces will not match
//! the one the contract computes.

use crate::{AAError, UserOperation};
use sha3::{Digest, Keccak256};

/// keccak256 of `data`.
pub fn keccak(data: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update(data);
    hasher.finalize().into()
}

/// Left-pad a `u128` into a 32-byte word.
fn word_u128(value: u128) -> [u8; 32] {
    let mut out = [0u8; 32];
    out[16..].copy_from_slice(&value.to_be_bytes());
    out
}

/// Left-pad a `u64` into a 32-byte word.
fn word_u64(value: u64) -> [u8; 32] {
    word_u128(value as u128)
}

/// Decode a `0x`-prefixed hex string, ignoring its checksum casing.
fn decode_hex(value: &str) -> Result<Vec<u8>, AAError> {
    let stripped = value.strip_prefix("0x").unwrap_or(value);
    if stripped.is_empty() {
        return Ok(Vec::new());
    }
    // An odd number of digits is not valid hex; reject rather than guess.
    if stripped.len() % 2 != 0 {
        return Err(AAError::BundlerError(format!(
            "{value} is not valid hex: odd number of digits"
        )));
    }
    hex::decode(stripped).map_err(|e| AAError::BundlerError(format!("{value} is not valid hex: {e}")))
}

/// Encode an address into a 32-byte word.
fn encode_address(address: &str) -> Result<[u8; 32], AAError> {
    let bytes = decode_hex(address)?;
    if bytes.len() != 20 {
        return Err(AAError::BundlerError(format!(
            "address {address} is {} bytes, expected 20",
            bytes.len()
        )));
    }
    let mut out = [0u8; 32];
    out[12..].copy_from_slice(&bytes);
    Ok(out)
}

/// A user operation with its gas fields packed as the EntryPoint expects.
struct PackedOp {
    sender: [u8; 32],
    nonce: [u8; 32],
    init_code: Vec<u8>,
    call_data: Vec<u8>,
    account_gas_limits: [u8; 32],
    pre_verification_gas: [u8; 32],
    gas_fees: [u8; 32],
    paymaster_and_data: Vec<u8>,
    signature: Vec<u8>,
}

/// Pack a user operation the way the v0.7 EntryPoint does.
fn pack(op: &UserOperation) -> Result<PackedOp, AAError> {
    // initCode is the factory address followed by its calldata.
    let init_code = match &op.init_code {
        Some(init) => {
            let mut bytes = decode_hex(&init.factory)?;
            if bytes.len() != 20 {
                return Err(AAError::BundlerError(
                    "factory address must be 20 bytes".to_string(),
                ));
            }
            bytes.extend_from_slice(&init.data);
            bytes
        }
        None => Vec::new(),
    };

    // accountGasLimits packs verificationGasLimit (high) and callGasLimit (low).
    let mut account_gas_limits = [0u8; 32];
    account_gas_limits[..16].copy_from_slice(&(op.verification_gas_limit as u128).to_be_bytes());
    account_gas_limits[16..].copy_from_slice(&(op.call_gas_limit as u128).to_be_bytes());

    // gasFees packs maxPriorityFeePerGas (high) and maxFeePerGas (low).
    let mut gas_fees = [0u8; 32];
    gas_fees[..16].copy_from_slice(&(op.max_priority_fee_per_gas as u128).to_be_bytes());
    gas_fees[16..].copy_from_slice(&(op.max_fee_per_gas as u128).to_be_bytes());

    // paymasterAndData is the paymaster address followed by its data.
    let paymaster_and_data = op.paymaster_data.clone().unwrap_or_default();

    Ok(PackedOp {
        sender: encode_address(&op.sender)?,
        nonce: word_u64(op.nonce),
        init_code,
        call_data: op.call_data.clone().unwrap_or_default(),
        account_gas_limits,
        pre_verification_gas: word_u64(op.pre_verification_gas),
        gas_fees,
        paymaster_and_data,
        signature: op.signature.clone(),
    })
}

/// Compute the ERC-4337 user operation hash.
///
/// This mirrors the EntryPoint:
/// `keccak256(abi.encode(hash(op), entryPoint, chainId))` where `hash(op)`
/// covers the packed fields.
pub fn user_op_hash(op: &UserOperation, entry_point: &str, chain_id: u64) -> Result<[u8; 32], AAError> {
    let packed = pack(op)?;

    let mut inner = Vec::with_capacity(8 * 32);
    inner.extend_from_slice(&packed.sender);
    inner.extend_from_slice(&packed.nonce);
    inner.extend_from_slice(&keccak(&packed.init_code));
    inner.extend_from_slice(&keccak(&packed.call_data));
    inner.extend_from_slice(&packed.account_gas_limits);
    inner.extend_from_slice(&packed.pre_verification_gas);
    inner.extend_from_slice(&packed.gas_fees);
    inner.extend_from_slice(&keccak(&packed.paymaster_and_data));

    let inner_hash = keccak(&inner);

    let mut outer = Vec::with_capacity(3 * 32);
    outer.extend_from_slice(&inner_hash);
    outer.extend_from_slice(&encode_address(entry_point)?);
    outer.extend_from_slice(&word_u64(chain_id));

    Ok(keccak(&outer))
}

/// Encode a dynamic `bytes` value: length word followed by right-padded data.
fn encode_dynamic_bytes(data: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(32 + data.len().div_ceil(32) * 32);
    out.extend_from_slice(&word_u64(data.len() as u64));
    out.extend_from_slice(data);

    let padding = (32 - (data.len() % 32)) % 32;
    out.extend(std::iter::repeat_n(0u8, padding));
    out
}

/// Encode one `PackedUserOperation` tuple.
///
/// A tuple with dynamic members is a head of nine words (five static values and
/// four offsets) followed by the dynamic contents. Offsets are measured from
/// the start of the tuple encoding.
fn encode_op_tuple(packed: &PackedOp) -> Vec<u8> {
    const HEAD_WORDS: usize = 9;
    let head_len = HEAD_WORDS * 32;

    let init_code = encode_dynamic_bytes(&packed.init_code);
    let call_data = encode_dynamic_bytes(&packed.call_data);
    let paymaster_and_data = encode_dynamic_bytes(&packed.paymaster_and_data);
    let signature = encode_dynamic_bytes(&packed.signature);

    let mut out = Vec::with_capacity(
        head_len + init_code.len() + call_data.len() + paymaster_and_data.len() + signature.len(),
    );

    // Head: five static words.
    out.extend_from_slice(&packed.sender);
    out.extend_from_slice(&packed.nonce);

    // Then the four offsets, each relative to the tuple start.
    let mut offset = head_len;
    out.extend_from_slice(&word_u64(offset as u64));
    offset += init_code.len();
    out.extend_from_slice(&word_u64(offset as u64));
    offset += call_data.len();
    out.extend_from_slice(&packed.account_gas_limits);
    out.extend_from_slice(&packed.pre_verification_gas);
    out.extend_from_slice(&packed.gas_fees);
    out.extend_from_slice(&word_u64(offset as u64));
    offset += paymaster_and_data.len();
    out.extend_from_slice(&word_u64(offset as u64));

    // Tail: the dynamic contents in the same order as the offsets.
    out.extend_from_slice(&init_code);
    out.extend_from_slice(&call_data);
    out.extend_from_slice(&paymaster_and_data);
    out.extend_from_slice(&signature);

    out
}

/// The `handleOps` function selector, derived from its signature.
pub fn handle_ops_selector() -> [u8; 4] {
    let signature =
        "handleOps((address,uint256,bytes,bytes,bytes32,uint256,bytes32,bytes,bytes)[],address)";
    let hash = keccak(signature.as_bytes());
    [hash[0], hash[1], hash[2], hash[3]]
}

/// Encode `handleOps(PackedUserOperation[] ops, address beneficiary)`.
pub fn encode_handle_ops(ops: &[UserOperation]) -> Result<Vec<u8>, AAError> {
    let tuples: Vec<Vec<u8>> = ops
        .iter()
        .map(|op| pack(op).map(|packed| encode_op_tuple(&packed)))
        .collect::<Result<_, _>>()?;

    // The array is a dynamic parameter, so its head entry is an offset. With two
    // parameters the head is two words, placing the array at 0x40.
    let mut body = Vec::new();
    body.extend_from_slice(&word_u64(tuples.len() as u64));

    // Each element of a dynamic array is itself dynamic, so the array body is a
    // list of offsets to each tuple, measured from after the length word.
    let mut element_offset = tuples.len() * 32;
    for tuple in &tuples {
        body.extend_from_slice(&word_u64(element_offset as u64));
        element_offset += tuple.len();
    }
    for tuple in &tuples {
        body.extend_from_slice(tuple);
    }

    let mut out = Vec::with_capacity(4 + 64 + body.len());
    out.extend_from_slice(&handle_ops_selector());
    out.extend_from_slice(&word_u64(64)); // offset of the ops array
    out.extend_from_slice(&word_u64(0)); // beneficiary (zero address)
    out.extend_from_slice(&body);

    Ok(out)
}

/// Encode a single user operation, for callers that want the tuple alone.
pub fn encode_user_op(op: &UserOperation) -> Result<Vec<u8>, AAError> {
    Ok(encode_op_tuple(&pack(op)?))
}

#[cfg(test)]
mod tests {
    use super::*;

    /// An operation matching the vector read from the deployed v0.7 EntryPoint.
    fn op() -> UserOperation {
        UserOperation {
            sender: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e".to_string(),
            nonce: 0,
            init_code: None,
            call_data: None,
            call_gas_limit: 100_000,
            verification_gas_limit: 150_000,
            pre_verification_gas: 21_000,
            max_fee_per_gas: 100_000_000_000,
            max_priority_fee_per_gas: 1_000_000_000,
            paymaster_data: None,
            signature: vec![],
        }
    }

    #[test]
    fn user_op_hash_matches_the_deployed_v07_entry_point() {
        // Read from `getUserOpHash` on the canonical v0.7 EntryPoint at
        // 0x0000000071727De22E5E9d8BAf0edAc6f37da032 (chain 1) for the
        // operation above. If the packing ever drifts from the contract, this
        // fails.
        let hash = user_op_hash(
            &op(),
            "0x0000000071727De22E5E9d8BAf0edAc6f37da032",
            1,
        )
        .unwrap();

        assert_eq!(
            hex::encode(hash),
            "bcdd35b5495841f65623e710c01a50266b4083a611885774de6c567f54ee0c99"
        );
    }

    #[test]
    fn handle_ops_selector_matches_the_canonical_function() {
        // keccak256("handleOps((address,uint256,bytes,bytes,bytes32,uint256,bytes32,bytes,bytes)[],address)")
        assert_eq!(hex::encode(handle_ops_selector()), "765e827f");
    }

    #[test]
    fn user_op_hash_is_deterministic_and_chain_bound() {
        let entry_point = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789";
        let a = user_op_hash(&op(), entry_point, 1).unwrap();
        let b = user_op_hash(&op(), entry_point, 1).unwrap();
        assert_eq!(a, b, "the same inputs must hash the same");

        let other_chain = user_op_hash(&op(), entry_point, 137).unwrap();
        assert_ne!(a, other_chain, "the hash must bind the chain id");

        let other_entry_point = "0x0000000000000000000000000000000000000001";
        let other = user_op_hash(&op(), other_entry_point, 1).unwrap();
        assert_ne!(a, other, "the hash must bind the EntryPoint");
    }

    #[test]
    fn user_op_hash_changes_with_the_nonce() {
        let entry_point = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789";

        let mut first = op();
        first.nonce = 1;
        let mut second = op();
        second.nonce = 2;

        assert_ne!(
            user_op_hash(&first, entry_point, 1).unwrap(),
            user_op_hash(&second, entry_point, 1).unwrap()
        );
    }

    #[test]
    fn user_op_hash_rejects_a_malformed_sender() {
        let mut bad = op();
        bad.sender = "0x1234".to_string();
        assert!(user_op_hash(&bad, "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789", 1).is_err());
    }

    #[test]
    fn user_op_hash_rejects_odd_length_hex() {
        let mut bad = op();
        bad.sender = "0x742d35cc6634c0532925a3b844bc454e4438f44".to_string();
        assert!(user_op_hash(&bad, "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789", 1).is_err());
    }

    #[test]
    fn account_gas_limits_packs_verification_high_and_call_low() {
        let packed = pack(&op()).unwrap();
        let verification = u128::from_be_bytes(packed.account_gas_limits[..16].try_into().unwrap());
        let call = u128::from_be_bytes(packed.account_gas_limits[16..].try_into().unwrap());

        assert_eq!(verification, 150_000);
        assert_eq!(call, 100_000);
    }

    #[test]
    fn dynamic_bytes_are_length_prefixed_and_padded() {
        let encoded = encode_dynamic_bytes(&[0xaa; 32]);
        assert_eq!(encoded.len(), 64);
        assert_eq!(&encoded[..32], &word_u64(32));

        // A value that is not a multiple of 32 is padded to the next word.
        let encoded = encode_dynamic_bytes(&[0xaa; 33]);
        assert_eq!(encoded.len(), 96);
        assert_eq!(&encoded[..32], &word_u64(33));
    }

    #[test]
    fn handle_ops_starts_with_the_selector_and_a_well_formed_head() {
        let encoded = encode_handle_ops(&[op()]).unwrap();

        assert_eq!(&encoded[..4], &handle_ops_selector());
        assert_eq!(&encoded[4..36], &word_u64(64), "ops array offset");
        assert_eq!(&encoded[36..68], &word_u64(0), "beneficiary");
        assert_eq!(&encoded[68..100], &word_u64(1), "one operation");
        assert_eq!(&encoded[100..132], &word_u64(32), "first element offset");
    }

    #[test]
    fn tuple_offsets_point_at_the_dynamic_contents() {
        let packed = pack(&op()).unwrap();
        let encoded = encode_op_tuple(&packed);

        // Nine head words.
        assert_eq!(encoded.len() % 32, 0);

        let offset_at = |index: usize| {
            let start = index * 32;
            let word: [u8; 32] = encoded[start..start + 32].try_into().unwrap();
            u128::from_be_bytes(word[16..].try_into().unwrap()) as usize
        };

        // Fields 0 and 1 are sender and nonce; 2 is the initCode offset.
        let init_code_offset = offset_at(2);
        let call_data_offset = offset_at(3);
        let paymaster_offset = offset_at(7);
        let signature_offset = offset_at(8);

        assert_eq!(init_code_offset, 9 * 32, "initCode follows the head");

        // initCode and callData are both empty, so each is one length word.
        assert_eq!(call_data_offset, init_code_offset + 32);
        assert_eq!(paymaster_offset, call_data_offset + 32);

        // paymasterAndData is empty.
        assert_eq!(signature_offset, paymaster_offset + 32);

        // Every offset lands on a word boundary inside the encoding.
        for offset in [
            init_code_offset,
            call_data_offset,
            paymaster_offset,
            signature_offset,
        ] {
            assert!(offset < encoded.len(), "offset {offset} is out of range");
            assert_eq!(offset % 32, 0, "offset {offset} is not word aligned");
        }
    }

    #[test]
    fn init_code_concatenates_factory_and_data() {
        let mut with_init = op();
        with_init.init_code = Some(crate::FactoryCall {
            factory: "0x00000000000000000000000000000000000000ff".to_string(),
            data: vec![0xde, 0xad],
        });

        let packed = pack(&with_init).unwrap();
        assert_eq!(packed.init_code.len(), 22);
        assert_eq!(&packed.init_code[20..], &[0xde, 0xad]);
    }
}
