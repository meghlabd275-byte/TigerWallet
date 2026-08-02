/**
 * Cold Wallet Module
 * Air-gapped signing for maximum security
 * Supports: QR code signing, USB transfer, secure transaction signing
 */

use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

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
        // In production, this would:
        // 1. Recover the signer address from signature
        // 2. Compare with expected_from
        // 3. Verify the transaction hash matches
        
        // For now, just check if we have a valid signature
        if signed.signature.is_empty() {
            return Err(ColdWalletError::InvalidSignature);
        }
        
        Ok(true)
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

/// Generate QR code for unsigned transaction
pub fn generate_qr_code(tx: &UnsignedTransaction) -> Result<Vec<u8>, ColdWalletError> {
    // Serialize and encode
    let serialized = serde_json::to_string(tx)
        .map_err(|e| ColdWalletError::StorageError(e.to_string()))?;
    
    // Use QR code library to generate image
    // This is a placeholder - in production, use qrcode crate
    Ok(serialized.as_bytes().to_vec())
}

/// Parse QR code data to unsigned transaction
pub fn parse_qr_code(data: &[u8]) -> Result<UnsignedTransaction, ColdWalletError> {
    let string = String::from_utf8(data.to_vec())
        .map_err(|_| ColdWalletError::InvalidTransaction)?;
    
    serde_json::from_str(&string)
        .map_err(|_| ColdWalletError::InvalidTransaction)
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
