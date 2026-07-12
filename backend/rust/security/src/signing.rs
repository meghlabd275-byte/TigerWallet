use crate::crypto::{Crypto, CryptoError};
use secp256k1::{Secp256k1, SecretKey, PublicKey, Message, Signature, All};
use serde::{Serialize, Deserialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub chain_id: u64,
    pub nonce: u64,
    pub to: String,
    pub value: String,
    pub data: Vec<u8>,
    pub gas_limit: u64,
    pub gas_price: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedTransaction {
    pub raw: Vec<u8>,
    pub hash: String,
    pub v: u8,
    pub r: String,
    pub s: String,
}

pub struct TransactionSigner;

impl TransactionSigner {
    pub fn sign_evm_transaction(
        tx: &Transaction,
        private_key: &[u8]
    ) -> Result<SignedTransaction, CryptoError> {
        let secp = Secp256k1::new();
        
        // Create secret key
        let secret_key = SecretKey::from_slice(private_key)
            .map_err(|_| CryptoError::InvalidKey)?;
        
        // Encode transaction
        let encoded = Self::encode_evm_transaction(tx)?;
        
        // Hash the transaction
        let tx_hash = Crypto::keccak256(&encoded);
        
        // Sign
        let message = Message::from_slice(&tx_hash)
            .map_err(|_| CryptoError::SigningFailed)?;
        
        let signature = secp.sign_message(&message, &secret_key);
        
        // Calculate v value
        let v = if tx.chain_id > 0 {
            (tx.chain_id * 2 + 35) as u8
        } else {
            27
        };
        
        Ok(SignedTransaction {
            raw: encoded,
            hash: format!("0x{}", hex::encode(tx_hash)),
            v,
            r: format!("0x{}", hex::encode(&signature.r()[..])),
            s: format!("0x{}", hex::encode(&signature.s()[..])),
        })
    }
    
    fn encode_evm_transaction(tx: &Transaction) -> Result<Vec<u8>, CryptoError> {
        use rlp::{RlpStream, Encodable};
        
        let mut stream = RlpStream::new();
        stream.begin_list(9);
        stream.append(&tx.nonce);
        stream.append(&tx.gas_price);
        stream.append(&tx.gas_limit);
        stream.append(&tx.to);
        stream.append(&tx.value);
        stream.append(&tx.data);
        stream.append(&tx.chain_id);
        stream.append(&0u8);
        stream.append(&0u8);
        
        Ok(stream.out().to_vec())
    }
    
    pub fn verify_evm_signature(
        message: &[u8],
        signature: &[u8],
        public_key: &[u8]
    ) -> Result<bool, CryptoError> {
        Crypto::verify_signature(message, signature, public_key)
    }
}

pub struct MessageSigner;

impl MessageSigner {
    pub fn personal_sign(message: &[u8], private_key: &[u8]) -> Result<String, CryptoError> {
        // Ethereum personal sign: keccak256("\x19Ethereum Signed Message:\n" + len(message) + message)
        let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
        let prefixed_message = [&prefix.as_bytes(), message].concat();
        let hash = Crypto::keccak256(&prefixed_message);
        
        let secp = Secp256k1::new();
        let secret_key = SecretKey::from_slice(private_key)
            .map_err(|_| CryptoError::InvalidKey)?;
        
        let message = Message::from_slice(&hash)
            .map_err(|_| CryptoError::SigningFailed)?;
        
        let signature = secp.sign_message(&message, &secret_key);
        
        // Add Ethereum-specific recovery byte
        let mut sig = signature.to_vec();
        sig.push(0); // v = 0
        
        Ok(format!("0x{}", hex::encode(sig)))
    }
    
    pub fn verify_personal_sign(
        message: &[u8],
        signature: &[u8],
        address: &str
    ) -> Result<bool, CryptoError> {
        // Recover public key from signature
        let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
        let prefixed_message = [&prefix.as_bytes(), message].concat();
        let hash = Crypto::keccak256(&prefixed_message);
        
        if signature.len() < 64 {
            return Err(CryptoError::InvalidSignature);
        }
        
        let v = signature[64];
        let recovery_id = if v < 27 { v } else { v - 27 };
        
        let secp = Secp256k1::new();
        let sig = Signature::from_compact(&signature[..64])
            .map_err(|_| CryptoError::InvalidSignature)?;
        
        let message = Message::from_slice(&hash)
            .map_err(|_| CryptoError::SigningFailed)?;
        
        // Try to recover public key
        let public_key = secp.recover(&message, &sig, recovery_id.into())
            .map_err(|_| CryptoError::InvalidSignature)?;
        
        let derived_address = Crypto::pubkey_to_address(&public_key.serialize_uncompressed());
        
        Ok(derived_address.to_lowercase() == address.to_lowercase())
    }
    
    pub fn sign_typed_data(
        data: &[u8],
        private_key: &[u8]
    ) -> Result<String, CryptoError> {
        // EIP-712 signing
        let domain_separator = Crypto::keccak256(b"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)");
        let hash = Crypto::keccak256(data);
        
        let secp = Secp256k1::new();
        let secret_key = SecretKey::from_slice(private_key)
            .map_err(|_| CryptoError::InvalidKey)?;
        
        let message = Message::from_slice(&hash)
            .map_err(|_| CryptoError::SigningFailed)?;
        
        let signature = secp.sign_message(&message, &secret_key);
        
        Ok(format!("0x{}", hex::encode(signature.to_vec())))
    }
}
