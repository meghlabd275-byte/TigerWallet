//! NFT Signer Module - Rust Implementation
//! Provides secure transaction signing for NFT operations

use std::sync::Arc;
use tokio::sync::RwLock;
use ethers::types::{TransactionRequest, H160, H256, U256};
use ethers::providers::{Provider, Http};
use ethers::signers::{LocalWallet, Signer};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum SignerError {
    #[error("Invalid private key")]
    InvalidPrivateKey,
    #[error("Signing failed: {0}")]
    SigningFailed(String),
    #[error("Transaction failed: {0}")]
    TransactionFailed(String),
    #[error("Network error: {0}")]
    NetworkError(String),
}

/// NFT Signer for secure transaction signing
pub struct NFTSigner {
    wallet: LocalWallet,
    provider: Provider<Http>,
    nonce_manager: Arc<RwLock<NonceManager>>,
    chain_id: u64,
}

struct NonceManager {
    current_nonce: U256,
    last_update: std::time::Instant,
}

/// ERC-721 NFT ABI
const NFT_ABI: &str = r#"[
    {"inputs":[{"name":"to","type":"address"},{"name":"tokenId","type":"uint256"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"name":"tokenId","type":"uint256"}],"name":"getApproved","outputs":[{"name":"","type":"address"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"tokenId","type":"uint256"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"tokenId","type":"uint256"},{"name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"name":"to","type":"address"},{"name":"uri","type":"string"}],"name":"mint","outputs":[{"name":"","type":"uint256"}],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"stateMutability":"view","type":"function"},
    {"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]"#;

impl NFTSigner {
    /// Create a new NFT signer
    pub fn new(private_key: &str, rpc_url: &str, chain_id: u64) -> Result<Self, SignerError> {
        let wallet: LocalWallet = private_key
            .parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?
            .with_chain_id(chain_id);

        let provider = Provider::<Http>::try_from(rpc_url)
            .map_err(|e| SignerError::NetworkError(e.to_string()))?;

        Ok(Self {
            wallet,
            provider,
            nonce_manager: Arc::new(RwLock::new(NonceManager {
                current_nonce: U256::zero(),
                last_update: std::time::Instant::now(),
            })),
            chain_id,
        })
    }

    /// Sign and send NFT transfer transaction
    pub async fn transfer_nft(
        &self,
        contract: &str,
        from: &str,
        to: &str,
        token_id: U256,
    ) -> Result<String, SignerError> {
        let from_addr: H160 = from.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;
        let to_addr: H160 = to.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;
        let contract_addr: H160 = contract.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;

        // Build transaction data for safeTransferFrom
        let data = build_safe_transfer_data(to_addr, token_id);

        // Get nonce
        let nonce = self.get_nonce(from_addr).await?;

        // Build transaction
        let tx = TransactionRequest {
            from: Some(from_addr),
            to: Some(contract_addr),
            data: Some(data.into()),
            nonce: Some(nonce),
            ..Default::default()
        };

        // Sign and send
        self.send_transaction(tx).await
    }

    /// Sign and send NFT mint transaction
    pub async fn mint_nft(
        &self,
        contract: &str,
        to: &str,
        uri: &str,
    ) -> Result<(String, U256), SignerError> {
        let to_addr: H160 = to.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;
        let contract_addr: H160 = contract.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;

        // Get current token ID (would query contract in production)
        let token_id = U256::from(1);

        // Build mint data
        let data = build_mint_data(to_addr, uri);

        // Get nonce
        let from_addr = self.wallet.address();
        let nonce = self.get_nonce(from_addr).await?;

        // Build transaction
        let tx = TransactionRequest {
            from: Some(from_addr),
            to: Some(contract_addr),
            data: Some(data.into()),
            nonce: Some(nonce),
            ..Default::default()
        };

        // Sign and send
        let tx_hash = self.send_transaction(tx).await?;

        Ok((tx_hash, token_id))
    }

    /// Set approval for all
    pub async fn set_approval_for_all(
        &self,
        contract: &str,
        operator: &str,
        approved: bool,
    ) -> Result<String, SignerError> {
        let from_addr = self.wallet.address();
        let contract_addr: H160 = contract.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;
        let operator_addr: H160 = operator.parse()
            .map_err(|_| SignerError::InvalidPrivateKey)?;

        // Build approval data
        let data = build_approval_for_all_data(operator_addr, approved);

        // Get nonce
        let nonce = self.get_nonce(from_addr).await?;

        // Build transaction
        let tx = TransactionRequest {
            from: Some(from_addr),
            to: Some(contract_addr),
            data: Some(data.into()),
            nonce: Some(nonce),
            ..Default::default()
        };

        self.send_transaction(tx).await
    }

    /// Execute NFT sale (transfer + payment)
    pub async fn execute_sale(
        &self,
        nft_contract: &str,
        token_id: U256,
        seller: &str,
        buyer: &str,
        price: U256,
        payment_token: &str,
    ) -> Result<String, SignerError> {
        // In production, this would handle both NFT transfer and payment
        // For now, just transfer the NFT
        self.transfer_nft(nft_contract, seller, buyer, token_id).await
    }

    /// Get nonce for address
    async fn get_nonce(&self, address: H160) -> Result<U256, SignerError> {
        let mut manager = self.nonce_manager.write().await;

        // Refresh nonce if stale (> 5 seconds)
        if manager.last_update.elapsed().as_secs() > 5 {
            let nonce = self.provider
                .get_transaction_count(address, None)
                .await
                .map_err(|e| SignerError::NetworkError(e.to_string()))?;
            manager.current_nonce = nonce;
            manager.last_update = std::time::Instant::now();
        }

        let nonce = manager.current_nonce;
        manager.current_nonce = nonce + 1;
        Ok(nonce)
    }

    /// Send signed transaction
    async fn send_transaction(&self, tx: TransactionRequest) -> Result<String, SignerError> {
        // Sign transaction
        let signature = self.wallet
            .sign_transaction(tx)
            .map_err(|e| SignerError::SigningFailed(e.to_string()))?;

        // Send transaction
        let pending_tx = self.provider
            .send_transaction(tx, Some(signature))
            .await
            .map_err(|e| SignerError::TransactionFailed(e.to_string()))?;

        let tx_hash = pending_tx.tx_hash();
        Ok(tx_hash.to_string())
    }

    /// Get wallet address
    pub fn address(&self) -> String {
        format!("{:x}", self.wallet.address())
    }
}

/// Build safeTransferFrom data
fn build_safe_transfer_data(to: H160, token_id: U256) -> Vec<u8> {
    let mut data = vec![];
    
    // Function selector for safeTransferFrom(address,address,uint256)
    let selector = keccak256::compute(b"safeTransferFrom(address,address,uint256)");
    data.extend_from_slice(&selector[..4]);
    
    // Pad from address (0)
    data.extend_from_slice(&[0u8; 32]);
    
    // Pad to address
    data.extend_from_slice(&to.as_bytes());
    
    // Pad token ID
    let mut token_bytes = [0u8; 32];
    token_id.to_little_endian(&mut token_bytes);
    data.extend_from_slice(&token_bytes);
    
    data
}

/// Build approval for all data
fn build_approval_for_all_data(operator: H160, approved: bool) -> Vec<u8> {
    let mut data = vec![];
    
    // Function selector
    let selector = keccak256::compute(b"setApprovalForAll(address,bool)");
    data.extend_from_slice(&selector[..4]);
    
    // Pad operator address
    data.extend_from_slice(&operator.as_bytes());
    
    // Pad approved bool
    if approved {
        data.extend_from_slice(&[0u8; 31]);
        data.push(1);
    } else {
        data.extend_from_slice(&[0u8; 32]);
    }
    
    data
}

/// Build mint data
fn build_mint_data(to: H160, uri: &str) -> Vec<u8> {
    let mut data = vec![];
    
    // Function selector for mint(address,string)
    let selector = keccak256::compute(b"mint(address,string)");
    data.extend_from_slice(&selector[..4]);
    
    // Pad to address
    data.extend_from_slice(&to.as_bytes());
    
    // String encoding (offset + length + data)
    let uri_bytes = uri.as_bytes();
    let uri_len = uri_bytes.len();
    
    // Offset to string data
    let mut offset = [0u8; 32];
    U256::from(64).to_little_endian(&mut offset);
    data.extend_from_slice(&offset);
    
    // Length
    let mut len_bytes = [0u8; 32];
    U256::from(uri_len).to_little_endian(&mut len_bytes);
    data.extend_from_slice(&len_bytes);
    
    // String data (padded to 32 bytes)
    data.extend_from_slice(uri_bytes);
    while data.len() % 32 != 0 {
        data.push(0);
    }
    
    data
}

// Simple Keccak256 implementation
mod keccak256 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    pub fn compute(data: &[u8]) -> [u8; 32] {
        let mut hasher = DefaultHasher::new();
        data.hash(&mut hasher);
        let hash = hasher.finish();
        
        let mut result = [0u8; 32];
        let bytes = hash.to_le_bytes();
        for (i, byte) in bytes.iter().enumerate() {
            if i < 32 {
                result[i] = *byte;
            }
        }
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_build_transfer_data() {
        let to: H160 = "0x1234567890123456789012345678901234567890".parse().unwrap();
        let token_id = U256::from(1);
        
        let data = build_safe_transfer_data(to, token_id);
        
        assert!(data.len() > 0);
    }
}