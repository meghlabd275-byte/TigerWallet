//! zkSync Era transactions — EIP-1559 typed transactions with RLP encoding

use crate::crypto::{keccak256, KeyPair};
use crate::types::{Address, ZksyncError};

/// Minimal RLP encoder (used for legacy + typed tx payloads)
fn rlp_bytes(out: &mut Vec<u8>, data: &[u8]) {
    if data.len() == 1 && data[0] < 0x80 {
        out.push(data[0]);
    } else if data.len() <= 55 {
        out.push(0x80 + data.len() as u8);
        out.extend_from_slice(data);
    } else {
        let len_bytes = (data.len() as u64).to_be_bytes();
        let significant = &len_bytes[len_bytes.iter().position(|&b| b != 0).unwrap_or(7)..];
        out.push(0xb7 + significant.len() as u8);
        out.extend_from_slice(significant);
        out.extend_from_slice(data);
    }
}

fn rlp_list(out: &mut Vec<u8>, items: &[Vec<u8>]) {
    let payload_len: usize = items.iter().map(|i| i.len()).sum();
    if payload_len <= 55 {
        out.push(0xc0 + payload_len as u8);
    } else {
        let len_bytes = (payload_len as u64).to_be_bytes();
        let significant = &len_bytes[len_bytes.iter().position(|&b| b != 0).unwrap_or(7)..];
        out.push(0xf7 + significant.len() as u8);
        out.extend_from_slice(significant);
    }
    for item in items {
        out.extend_from_slice(item);
    }
}

fn rlp_u64(out: &mut Vec<u8>, v: u64) {
    if v == 0 {
        rlp_bytes(out, &[]);
    } else {
        let b = v.to_be_bytes();
        let significant = &b[b.iter().position(|&x| x != 0).unwrap_or(7)..];
        rlp_bytes(out, significant);
    }
}

fn rlp_u128(out: &mut Vec<u8>, v: u128) {
    if v == 0 {
        rlp_bytes(out, &[]);
    } else {
        let b = v.to_be_bytes();
        let significant = &b[b.iter().position(|&x| x != 0).unwrap_or(15)..];
        rlp_bytes(out, significant);
    }
}

/// EIP-1559 dynamic-fee transaction (type 0x02)
#[derive(Debug, Clone)]
pub struct Eip1559Transaction {
    pub chain_id: u64,
    pub nonce: u64,
    pub max_priority_fee_per_gas: u128,
    pub max_fee_per_gas: u128,
    pub gas_limit: u64,
    pub to: Option<Address>,
    pub value: u128,
    pub data: Vec<u8>,
    /// Empty access list
    pub access_list: Vec<(Address, Vec<[u8; 32]>)>,
}

impl Eip1559Transaction {
    /// RLP payload for the signing hash (0x02 || rlp([...]))
    pub fn signing_payload(&self) -> Vec<u8> {
        let mut items: Vec<Vec<u8>> = Vec::new();
        let mut buf = Vec::new();
        rlp_u64(&mut buf, self.chain_id);
        items.push(std::mem::take(&mut buf));
        rlp_u64(&mut buf, self.nonce);
        items.push(std::mem::take(&mut buf));
        rlp_u128(&mut buf, self.max_priority_fee_per_gas);
        items.push(std::mem::take(&mut buf));
        rlp_u128(&mut buf, self.max_fee_per_gas);
        items.push(std::mem::take(&mut buf));
        rlp_u64(&mut buf, self.gas_limit);
        items.push(std::mem::take(&mut buf));
        match &self.to {
            Some(addr) => rlp_bytes(&mut buf, addr),
            None => rlp_bytes(&mut buf, &[]),
        }
        items.push(std::mem::take(&mut buf));
        rlp_u128(&mut buf, self.value);
        items.push(std::mem::take(&mut buf));
        rlp_bytes(&mut buf, &self.data);
        items.push(std::mem::take(&mut buf));
        rlp_list(&mut buf, &[]); // empty access list
        items.push(std::mem::take(&mut buf));

        let mut out = vec![0x02u8];
        rlp_list(&mut out, &items);
        out
    }

    /// keccak256(0x02 || rlp(payload)) — the digest that gets signed
    pub fn signing_hash(&self) -> [u8; 32] {
        keccak256(&self.signing_payload())
    }

    /// Sign and produce the raw broadcastable transaction bytes
    pub fn sign(&self, key: &KeyPair) -> Result<SignedTransaction, ZksyncError> {
        let hash = self.signing_hash();
        let sig = key.sign(&hash);

        // Recover the recid (v) by trying both parities against the sender address
        let sender = key.address();
        let sig_bytes = sig.to_bytes();
        let mut r = [0u8; 32];
        let mut s = [0u8; 32];
        r.copy_from_slice(&sig_bytes[..32]);
        s.copy_from_slice(&sig_bytes[32..]);

        let mut recid = None;
        for v in 0u8..2 {
            if let Ok(recovered) = recover_address(&hash, &r, &s, v) {
                if recovered == sender {
                    recid = Some(v);
                    break;
                }
            }
        }
        let y_parity = recid.ok_or_else(|| {
            ZksyncError::InvalidTransaction("failed to recover signer parity".to_string())
        })?;

        Ok(SignedTransaction {
            tx: self.clone(),
            y_parity,
            r,
            s,
        })
    }
}

/// Signed transaction ready for eth_sendRawTransaction
#[derive(Debug, Clone)]
pub struct SignedTransaction {
    pub tx: Eip1559Transaction,
    pub y_parity: u8,
    pub r: [u8; 32],
    pub s: [u8; 32],
}

impl SignedTransaction {
    /// Raw bytes: 0x02 || rlp([..., y_parity, r, s])
    pub fn raw_bytes(&self) -> Vec<u8> {
        let mut items: Vec<Vec<u8>> = Vec::new();
        let mut buf = Vec::new();
        rlp_u64(&mut buf, self.tx.chain_id);
        items.push(std::mem::take(&mut buf));
        rlp_u64(&mut buf, self.tx.nonce);
        items.push(std::mem::take(&mut buf));
        rlp_u128(&mut buf, self.tx.max_priority_fee_per_gas);
        items.push(std::mem::take(&mut buf));
        rlp_u128(&mut buf, self.tx.max_fee_per_gas);
        items.push(std::mem::take(&mut buf));
        rlp_u64(&mut buf, self.tx.gas_limit);
        items.push(std::mem::take(&mut buf));
        match &self.tx.to {
            Some(addr) => rlp_bytes(&mut buf, addr),
            None => rlp_bytes(&mut buf, &[]),
        }
        items.push(std::mem::take(&mut buf));
        rlp_u128(&mut buf, self.tx.value);
        items.push(std::mem::take(&mut buf));
        rlp_bytes(&mut buf, &self.tx.data);
        items.push(std::mem::take(&mut buf));
        rlp_list(&mut buf, &[]);
        items.push(std::mem::take(&mut buf));
        rlp_u64(&mut buf, self.y_parity as u64);
        items.push(std::mem::take(&mut buf));
        rlp_bytes(&mut buf, &self.r);
        items.push(std::mem::take(&mut buf));
        rlp_bytes(&mut buf, &self.s);
        items.push(std::mem::take(&mut buf));

        let mut out = vec![0x02u8];
        rlp_list(&mut out, &items);
        out
    }

    /// 0x-prefixed hex for eth_sendRawTransaction
    pub fn raw_hex(&self) -> String {
        format!("0x{}", hex::encode(self.raw_bytes()))
    }

    /// keccak256 of the raw bytes — the transaction hash
    pub fn hash(&self) -> [u8; 32] {
        keccak256(&self.raw_bytes())
    }
}

/// Recover the sender address from a signature
fn recover_address(
    hash: &[u8; 32],
    r: &[u8; 32],
    s: &[u8; 32],
    v: u8,
) -> Result<Address, ZksyncError> {
    use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
    let mut sig_bytes = [0u8; 64];
    sig_bytes[..32].copy_from_slice(r);
    sig_bytes[32..].copy_from_slice(s);
    let sig = Signature::from_bytes(&sig_bytes.into())
        .map_err(|e| ZksyncError::InvalidTransaction(e.to_string()))?;
    let recid = RecoveryId::new(v != 0, false);
    let vk = VerifyingKey::recover_from_prehash(hash, &sig, recid)
        .map_err(|e| ZksyncError::InvalidTransaction(e.to_string()))?;
    let point = vk.to_encoded_point(false);
    let h = keccak256(&point.as_bytes()[1..65]);
    let mut addr = [0u8; 20];
    addr.copy_from_slice(&h[12..]);
    Ok(addr)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_tx() -> Eip1559Transaction {
        Eip1559Transaction {
            chain_id: 324,
            nonce: 0,
            max_priority_fee_per_gas: 1_000_000_000,
            max_fee_per_gas: 2_000_000_000,
            gas_limit: 21_000,
            to: Some([0xabu8; 20]),
            value: 1_000_000_000_000_000_000,
            data: vec![],
            access_list: vec![],
        }
    }

    #[test]
    fn signed_tx_recovers_sender() {
        let kp = KeyPair::generate();
        let signed = test_tx().sign(&kp).unwrap();
        let recovered = recover_address(
            &signed.tx.signing_hash(),
            &signed.r,
            &signed.s,
            signed.y_parity,
        )
        .unwrap();
        assert_eq!(recovered, kp.address());
    }

    #[test]
    fn raw_hex_is_02_prefixed() {
        let kp = KeyPair::generate();
        let signed = test_tx().sign(&kp).unwrap();
        assert!(signed.raw_hex().starts_with("0x02"));
    }
}
