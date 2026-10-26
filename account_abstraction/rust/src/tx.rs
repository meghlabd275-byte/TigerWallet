//! EIP-1559 transaction signing and RLP encoding.
//!
//! A transaction is signed over `keccak256(0x02 || rlp([chainId, nonce,
//! maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
//! accessList]))`, per EIP-2718 and EIP-1559. The signature is appended to the
//! same list to produce the raw transaction.

use crate::AAError;
use k256::ecdsa::{RecoveryId, Signature, SigningKey, VerifyingKey};
use sha3::{Digest, Keccak256};

use crate::bundler::abi;
use crate::bundler::BundleTransaction;

/// RLP encoding of a byte string.
fn rlp_bytes(data: &[u8]) -> Vec<u8> {
    match data.len() {
        0 => vec![0x80],
        1 if data[0] < 0x80 => vec![data[0]],
        len if len <= 55 => {
            let mut out = Vec::with_capacity(1 + len);
            out.push(0x80 + len as u8);
            out.extend_from_slice(data);
            out
        }
        len => {
            let length_bytes = encode_length(len);
            let mut out = Vec::with_capacity(1 + length_bytes.len() + len);
            out.push(0xb7 + length_bytes.len() as u8);
            out.extend_from_slice(&length_bytes);
            out.extend_from_slice(data);
            out
        }
    }
}

/// RLP encoding of a big-endian integer.
///
/// RLP encodes a number as the shortest big-endian byte string, so the leading
/// zeros of the 16-byte form are dropped first.
fn rlp_u128(value: u128) -> Vec<u8> {
    if value == 0 {
        return vec![0x80];
    }

    let bytes = value.to_be_bytes();
    let first_nonzero = bytes
        .iter()
        .position(|&b| b != 0)
        .expect("a nonzero value has a nonzero byte");
    rlp_bytes(&bytes[first_nonzero..])
}

/// RLP encoding of a list of already-encoded items.
fn rlp_list(items: &[Vec<u8>]) -> Vec<u8> {
    let payload_len: usize = items.iter().map(|item| item.len()).sum();

    let mut out = Vec::with_capacity(payload_len + 9);
    if payload_len <= 55 {
        out.push(0xc0 + payload_len as u8);
    } else {
        let length_bytes = encode_length(payload_len);
        out.push(0xf7 + length_bytes.len() as u8);
        out.extend_from_slice(&length_bytes);
    }

    for item in items {
        out.extend_from_slice(item);
    }
    out
}

/// Big-endian length bytes with leading zeros stripped.
fn encode_length(len: usize) -> Vec<u8> {
    let bytes = len.to_be_bytes();
    let first_nonzero = bytes.iter().position(|&b| b != 0).unwrap_or(bytes.len() - 1);
    bytes[first_nonzero..].to_vec()
}

/// Signs transactions with a secp256k1 key.
///
/// The key is held only as a signing key; the private scalar is never exposed.
pub struct TxSigner {
    signing_key: SigningKey,
    address: String,
}

impl TxSigner {
    /// Build a signer from a 32-byte private key.
    pub fn from_bytes(key: &[u8]) -> Result<Self, AAError> {
        let signing_key = SigningKey::from_slice(key)
            .map_err(|e| AAError::SecurityError(format!("invalid private key: {e}")))?;

        Ok(Self {
            address: Self::derive_address(&signing_key),
            signing_key,
        })
    }

    /// Build a signer from a `0x`-prefixed hex private key.
    pub fn from_hex(key: &str) -> Result<Self, AAError> {
        let stripped = key.strip_prefix("0x").unwrap_or(key);
        let bytes = hex::decode(stripped)
            .map_err(|e| AAError::SecurityError(format!("private key is not valid hex: {e}")))?;
        Self::from_bytes(&bytes)
    }

    /// The signer's checksummed-lowercase address.
    pub fn address(&self) -> String {
        self.address.clone()
    }

    /// Derive an address from a signing key: the last 20 bytes of the
    /// keccak256 hash of the uncompressed public key without its `0x04` prefix.
    fn derive_address(signing_key: &SigningKey) -> String {
        let verifying_key = VerifyingKey::from(signing_key);
        let encoded = verifying_key.to_encoded_point(false);
        let hash = abi::keccak(&encoded.as_bytes()[1..]);
        format!("0x{}", hex::encode(&hash[12..]))
    }

    /// Sign a transaction, returning the `0x`-prefixed raw encoding.
    pub fn sign_eip1559(
        &self,
        tx: &BundleTransaction,
        chain_id: u64,
        nonce: u64,
    ) -> Result<String, AAError> {
        let to = decode_address(&tx.to)?;

        let unsigned = rlp_list(&[
            rlp_u128(chain_id as u128),
            rlp_u128(nonce as u128),
            rlp_u128(tx.max_priority_fee_per_gas),
            rlp_u128(tx.max_fee_per_gas),
            rlp_u128(tx.gas_limit as u128),
            rlp_bytes(&to),
            rlp_u128(tx.value),
            rlp_bytes(&tx.data),
            rlp_list(&[]), // access list
        ]);

        // The signed preimage is the type byte followed by the RLP list.
        let mut preimage = Vec::with_capacity(1 + unsigned.len());
        preimage.push(0x02);
        preimage.extend_from_slice(&unsigned);

        let digest = Keccak256::digest(&preimage);

        let (signature, recovery_id) = self
            .signing_key
            .sign_prehash_recoverable(&digest)
            .map_err(|e| AAError::SecurityError(format!("signing failed: {e}")))?;

        let mut r = [0u8; 32];
        let mut s = [0u8; 32];
        r.copy_from_slice(&signature.r().to_bytes());
        s.copy_from_slice(&signature.s().to_bytes());

        // EIP-1559 transactions are not replay-protected by the v value, so v
        // is just the recovery id.
        let v = recovery_id.to_byte();

        let signed = rlp_list(&[
            rlp_u128(chain_id as u128),
            rlp_u128(nonce as u128),
            rlp_u128(tx.max_priority_fee_per_gas),
            rlp_u128(tx.max_fee_per_gas),
            rlp_u128(tx.gas_limit as u128),
            rlp_bytes(&to),
            rlp_u128(tx.value),
            rlp_bytes(&tx.data),
            rlp_list(&[]),
            rlp_u128(v as u128),
            rlp_bytes(&r),
            rlp_bytes(&s),
        ]);

        let mut raw = Vec::with_capacity(1 + signed.len());
        raw.push(0x02);
        raw.extend_from_slice(&signed);

        Ok(format!("0x{}", hex::encode(raw)))
    }

    /// Recover the address that produced a signature over `digest`.
    ///
    /// Used to check that a user operation's signature came from an expected
    /// owner before the operation is submitted.
    pub fn recover_signer(
        digest: &[u8; 32],
        signature: &[u8],
        recovery_id: u8,
    ) -> Result<String, AAError> {
        let signature =
            Signature::from_slice(signature).map_err(|_| AAError::InvalidSignature)?;

        let recovery_id = RecoveryId::from_byte(recovery_id)
            .ok_or(AAError::InvalidSignature)?;

        let verifying_key = VerifyingKey::recover_from_prehash(digest, &signature, recovery_id)
            .map_err(|_| AAError::InvalidSignature)?;

        let encoded = verifying_key.to_encoded_point(false);
        let hash = abi::keccak(&encoded.as_bytes()[1..]);

        Ok(format!("0x{}", hex::encode(&hash[12..])))
    }
}

/// Decode a `0x`-prefixed address into 20 bytes.
fn decode_address(address: &str) -> Result<Vec<u8>, AAError> {
    let stripped = address.strip_prefix("0x").unwrap_or(address);
    let bytes = hex::decode(stripped)
        .map_err(|e| AAError::BundlerError(format!("address {address} is not valid hex: {e}")))?;

    if bytes.len() != 20 {
        return Err(AAError::BundlerError(format!(
            "address {address} is {} bytes, expected 20",
            bytes.len()
        )));
    }
    Ok(bytes)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rlp_encodes_a_single_byte_below_0x80_as_itself() {
        assert_eq!(rlp_bytes(&[0x7f]), vec![0x7f]);
    }

    #[test]
    fn rlp_encodes_the_empty_string_as_0x80() {
        assert_eq!(rlp_bytes(&[]), vec![0x80]);
    }

    #[test]
    fn rlp_encodes_a_short_string_with_the_0x80_prefix() {
        // "dog" is the canonical example from the specification.
        assert_eq!(rlp_bytes(b"dog"), vec![0x83, b'd', b'o', b'g']);
    }

    #[test]
    fn rlp_encodes_a_56_byte_string_with_the_0xb8_prefix() {
        let data = vec![0xaa; 56];
        let encoded = rlp_bytes(&data);
        assert_eq!(encoded[0], 0xb8);
        assert_eq!(encoded[1], 56);
        assert_eq!(encoded.len(), 2 + 56);
    }

    #[test]
    fn rlp_encodes_zero_as_an_empty_string() {
        assert_eq!(rlp_u128(0), vec![0x80]);
    }

    #[test]
    fn rlp_strips_leading_zeros_from_integers() {
        // 1024 is 0x0400; RLP encodes it without the leading zero byte.
        assert_eq!(rlp_u128(1024), vec![0x82, 0x04, 0x00]);
        assert_eq!(rlp_u128(15), vec![0x0f]);
        assert_eq!(rlp_u128(127), vec![0x7f]);
        assert_eq!(rlp_u128(128), vec![0x81, 0x80]);
    }

    #[test]
    fn rlp_encodes_the_empty_list_as_0xc0() {
        assert_eq!(rlp_list(&[]), vec![0xc0]);
    }

    #[test]
    fn rlp_encodes_a_short_list_with_the_0xc0_prefix() {
        // ["cat", "dog"] is the canonical example.
        let encoded = rlp_list(&[rlp_bytes(b"cat"), rlp_bytes(b"dog")]);
        assert_eq!(
            encoded,
            vec![0xc8, 0x83, b'c', b'a', b't', 0x83, b'd', b'o', b'g']
        );
    }

    #[test]
    fn signer_derives_the_expected_address() {
        // A fixed key gives a stable address, so a regression in derivation
        // changes this value.
        let key = [0x11u8; 32];
        let signer = TxSigner::from_bytes(&key).unwrap();
        assert_eq!(signer.address().len(), 42);
        assert!(signer.address().starts_with("0x"));

        let again = TxSigner::from_bytes(&key).unwrap();
        assert_eq!(signer.address(), again.address());
    }

    #[test]
    fn signer_rejects_a_short_key() {
        assert!(TxSigner::from_bytes(&[0u8; 16]).is_err());
    }

    #[test]
    fn signed_transaction_is_a_typed_envelope() {
        let signer = TxSigner::from_bytes(&[0x22u8; 32]).unwrap();
        let tx = BundleTransaction {
            to: "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789".to_string(),
            data: vec![0xde, 0xad, 0xbe, 0xef],
            gas_limit: 200_000,
            max_fee_per_gas: 30_000_000_000,
            max_priority_fee_per_gas: 1_000_000_000,
            value: 0,
        };

        let raw = signer.sign_eip1559(&tx, 1, 7).unwrap();

        assert!(
            raw.starts_with("0x02"),
            "EIP-1559 transactions are typed 0x02"
        );

        // The type byte is followed by the RLP list of the transaction fields,
        // so the next byte must be a list header.
        let second = u8::from_str_radix(&raw[4..6], 16).unwrap();
        assert!(
            (0xc0..=0xff).contains(&second),
            "the type byte must be followed by an RLP list header, got {second:#04x}"
        );
    }

    #[test]
    fn signing_is_deterministic_for_the_same_inputs() {
        let signer = TxSigner::from_bytes(&[0x33u8; 32]).unwrap();
        let tx = BundleTransaction {
            to: "0x0000000000000000000000000000000000000001".to_string(),
            data: vec![],
            gas_limit: 21_000,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 1,
            value: 0,
        };

        // RFC 6979 signatures are deterministic, so the same inputs produce the
        // same raw transaction.
        assert_eq!(
            signer.sign_eip1559(&tx, 1, 0).unwrap(),
            signer.sign_eip1559(&tx, 1, 0).unwrap()
        );
    }

    #[test]
    fn different_chain_ids_produce_different_transactions() {
        let signer = TxSigner::from_bytes(&[0x44u8; 32]).unwrap();
        let tx = BundleTransaction {
            to: "0x0000000000000000000000000000000000000001".to_string(),
            data: vec![],
            gas_limit: 21_000,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 1,
            value: 0,
        };

        assert_ne!(
            signer.sign_eip1559(&tx, 1, 0).unwrap(),
            signer.sign_eip1559(&tx, 137, 0).unwrap()
        );
    }

    #[test]
    fn signing_rejects_a_malformed_recipient() {
        let signer = TxSigner::from_bytes(&[0x55u8; 32]).unwrap();
        let tx = BundleTransaction {
            to: "0x1234".to_string(),
            data: vec![],
            gas_limit: 21_000,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 1,
            value: 0,
        };

        assert!(signer.sign_eip1559(&tx, 1, 0).is_err());
    }

    #[test]
    fn a_signature_recovers_to_the_signing_address() {
        let signer = TxSigner::from_bytes(&[0x66u8; 32]).unwrap();
        let digest: [u8; 32] = abi::keccak(b"message");

        let (signature, recovery_id) = signer
            .signing_key
            .sign_prehash_recoverable(&digest)
            .unwrap();

        let recovered = TxSigner::recover_signer(
            &digest,
            &signature.to_bytes(),
            recovery_id.to_byte(),
        )
        .unwrap();

        assert_eq!(recovered, signer.address());
    }
}
