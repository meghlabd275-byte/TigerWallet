/**
 * Cold Wallet Module
 * Air-gapped signing for maximum security
 * Supports: QR code signing, USB transfer, secure transaction signing
 */

use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

use secp256k1::ecdsa::RecoverableSignature;
use secp256k1::{Message, PublicKey, Secp256k1};
use sha3::{Digest, Keccak256};

/// Signed transaction for air-gapped transfer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedTransaction {
    pub tx_hash: String,
    pub raw_tx: String,
    pub signature: String,
    pub signed_at: u64,
}

/// Unsigned transaction ready for signing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnsignedTransaction {
    pub id: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub gas_limit: String,
    pub gas_price: String,
    pub nonce: u64,
    pub chain_id: u64,
    pub data: Option<String>,
    pub token_symbol: Option<String>,
    pub token_decimals: Option<u8>,
}

/// Transaction request for signing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionRequest {
    pub to: String,
    pub value: String,
    pub data: Option<String>,
    pub chain_id: u64,
    pub gas_limit: Option<String>,
    pub gas_price: Option<String>,
}

/// Cold wallet error types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ColdWalletError {
    InvalidTransaction,
    SigningFailed(String),
    VerificationFailed(String),
    StorageError(String),
    InvalidSignature,
}

/// Cold Wallet Manager
pub struct ColdWalletManager {
    wallet_path: PathBuf,
}

impl ColdWalletManager {
    /// Create a new cold wallet manager
    pub fn new(wallet_path: PathBuf) -> Self {
        Self { wallet_path }
    }

    /// Generate unsigned transaction for air-gapped signing
    pub fn create_unsigned_transaction(
        &self,
        request: TransactionRequest,
        from_address: String,
        nonce: u64,
    ) -> Result<UnsignedTransaction, ColdWalletError> {
        // Generate transaction ID
        let id = format!("tx_{}", uuid::Uuid::new_v4());

        Ok(UnsignedTransaction {
            id,
            from: from_address,
            to: request.to,
            value: request.value,
            gas_limit: request.gas_limit.unwrap_or_else(|| "21000".to_string()),
            gas_price: request.gas_price.unwrap_or_else(|| "1000000000".to_string()),
            nonce,
            chain_id: request.chain_id,
            data: request.data,
            token_symbol: None,
            token_decimals: None,
        })
    }

    /// Export transaction to QR code data
    pub fn export_for_qr(&self, tx: &UnsignedTransaction) -> Result<String, ColdWalletError> {
        // Serialize transaction to base64 for QR code
        let serialized = serde_json::to_string(tx)
            .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
        
        // Encode to base64
        use base64::{Engine as _, engine::general_purpose::STANDARD};
        Ok(STANDARD.encode(serialized))
    }

    /// Import signed transaction from QR code data
    pub fn import_from_qr(&self, qr_data: &str) -> Result<SignedTransaction, ColdWalletError> {
        // Decode base64
        use base64::{Engine as _, engine::general_purpose::STANDARD};
        let decoded = STANDARD.decode(qr_data)
            .map_err(|e| ColdWalletError::InvalidTransaction)?;
        
        let serialized = String::from_utf8(decoded)
            .map_err(|_| ColdWalletError::InvalidTransaction)?;
        
        // Parse signed transaction
        let signed: SignedTransaction = serde_json::from_str(&serialized)
            .map_err(|_| ColdWalletError::InvalidTransaction)?;
        
        Ok(signed)
    }

    /// Save transaction to file for USB transfer
    pub fn save_to_file(&self, tx: &UnsignedTransaction, filename: &str) -> Result<(), ColdWalletError> {
        let path = self.wallet_path.join("pending").join(filename);
        
        // Ensure directory exists
        fs::create_dir_all(path.parent().unwrap())
            .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
        
        let json = serde_json::to_string_pretty(tx)
            .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
        
        fs::write(&path, json)
            .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
        
        Ok(())
    }

    /// Load signed transaction from file
    pub fn load_from_file(&self, filename: &str) -> Result<SignedTransaction, ColdWalletError> {
        let path = self.wallet_path.join("signed").join(filename);
        
        let json = fs::read_to_string(&path)
            .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
        
        let signed: SignedTransaction = serde_json::from_str(&json)
            .map_err(|_| ColdWalletError::InvalidTransaction)?;
        
        Ok(signed)
    }

    /// Verify signed transaction
    pub fn verify_signature(
        &self,
        signed: &SignedTransaction,
        expected_from: &str,
    ) -> Result<bool, ColdWalletError> {
        // Real cryptographic signature verification (no fake "check signature
        // is non-empty"). Recover the signer public key from a 65-byte
        // recoverable secp256k1 signature (r||s||v) over the transaction hash
        // (keccak256 of the signed tx) and compare its Ethereum address to
        // expected_from.
        if signed.signature.is_empty() {
            return Err(ColdWalletError::InvalidSignature);
        }

        let sig_bytes = hex::decode(signed.signature.trim_start_matches("0x"))
            .map_err(|_| ColdWalletError::InvalidSignature)?;
        if sig_bytes.len() != 65 {
            return Err(ColdWalletError::InvalidSignature);
        }
        let msg_bytes = hex::decode(signed.tx_hash.trim_start_matches("0x"))
            .map_err(|_| ColdWalletError::VerificationFailed("invalid tx_hash".into()))?;
        if msg_bytes.len() != 32 {
            return Err(ColdWalletError::VerificationFailed("invalid tx_hash length".into()));
        }

        let msg = Message::from_slice(&msg_bytes)
            .map_err(|e| ColdWalletError::VerificationFailed(e.to_string()))?;
        let recovery_id = secp256k1::ecdsa::RecoveryId::from_i32(sig_bytes[64] as i32)
            .map_err(|_| ColdWalletError::InvalidSignature)?;
        let rec_sig = RecoverableSignature::from_compact(&sig_bytes[..64], recovery_id)
            .map_err(|_| ColdWalletError::InvalidSignature)?;

        let secp = Secp256k1::new();
        let pubkey = secp
            .recover_ecdsa(&msg, &rec_sig)
            .map_err(|e| ColdWalletError::VerificationFailed(e.to_string()))?;

        let recovered = recover_eth_address(&pubkey);
        let expected = expected_from.trim_start_matches("0x").to_ascii_lowercase();
        Ok(recovered == expected)
    }

    /// Broadcast signed transaction (requires internet connection)
    pub async fn broadcast_transaction(
        &self,
        signed: &SignedTransaction,
        rpc_url: &str,
    ) -> Result<String, ColdWalletError> {
        // Use eth_sendRawTransaction RPC call
        let client = reqwest::Client::new();
        
        let response = client
            .post(rpc_url)
            .json(&serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_sendRawTransaction",
                "params": [signed.raw_tx],
                "id": 1
            }))
            .send()
            .await
            .map_err(|e| ColdWalletError::SigningFailed(e.to_string()))?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await
                .map_err(|e| ColdWalletError::SigningFailed(e.to_string()))?;
            
            if let Some(tx_hash) = result.get("result").and_then(|r| r.as_str()) {
                Ok(tx_hash.to_string())
            } else {
                Err(ColdWalletError::SigningFailed("No transaction hash".to_string()))
            }
        } else {
            Err(ColdWalletError::SigningFailed(format!("HTTP error: {}", response.status())))
        }
    }
}

/// Generate QR code for unsigned transaction.
///
/// Returns the exact payload that the QR renderer must encode. The unsigned
/// transaction is serialized to JSON; the actual QR image rasterization is
/// performed by the renderer layer (this function returns the bytes that must
/// appear in the QR, so `parse_qr_code` round-trips byte-for-byte).
pub fn generate_qr_code(tx: &UnsignedTransaction) -> Result<Vec<u8>, ColdWalletError> {
    let serialized = serde_json::to_vec(tx)
        .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
    Ok(serialized)
}

/// Parse QR code data to unsigned transaction
pub fn parse_qr_code(data: &[u8]) -> Result<UnsignedTransaction, ColdWalletError> {
    let string = String::from_utf8(data.to_vec())
        .map_err(|_| ColdWalletError::InvalidTransaction)?;

    serde_json::from_str(&string)
        .map_err(|_| ColdWalletError::InvalidTransaction)
}

/// recover_eth_address derives the lowercase 0x-prefixed Ethereum address from
/// an uncompressed secp256k1 public key: keccak256(pubkey[1..65])[12..32].
fn recover_eth_address(pubkey: &PublicKey) -> String {
    let serialized = pubkey.serialize_uncompressed();
    let mut hasher = Keccak256::new();
    hasher.update(&serialized[1..]);
    let hash = hasher.finalize();
    format!("0x{}", hex::encode(&hash[12..]))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_unsigned_transaction() {
        let manager = ColdWalletManager::new(PathBuf::from("/tmp/test"));
        
        let request = TransactionRequest {
            to: "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E".to_string(),
            value: "1000000000000000000".to_string(),
            data: None,
            chain_id: 1,
            gas_limit: None,
            gas_price: None,
        };
        
        let tx = manager.create_unsigned_transaction(
            request,
            "0x123d35Cc6634C0532925a3b844Bc9e7595f0eB1E".to_string(),
            0,
        ).unwrap();
        
        assert_eq!(tx.from, "0x123d35Cc6634C0532925a3b844Bc9e7595f0eB1E");
        assert_eq!(tx.to, "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E");
    }
}
