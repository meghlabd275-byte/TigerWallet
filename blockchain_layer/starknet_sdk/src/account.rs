//! Starknet Account
//! 
//! Account abstraction for Starknet (OpenZeppelin Account, etc.)

use std::fmt;
use crate::address::StarknetAddress;
use crate::crypto::{KeyPair, Signature};
use crate::provider::{Provider, FunctionCall, InvokeResult, DeployResult, FeeEstimate};
use crate::transaction::{InvokeTransaction, DeployAccountTransaction};

/// Account type
#[derive(Debug, Clone)]
pub enum AccountType {
    /// OpenZeppelin Account (v0.3.0+)
    OpenZeppelin,
    /// Argent X
    ArgentX,
    /// Braavos
    Braavos,
    /// Generic account
    Generic,
}

/// Starknet Account
pub struct Account {
    /// Address
    address: StarknetAddress,
    /// Key pair for signing
    key_pair: KeyPair,
    /// Account type
    account_type: AccountType,
    /// Class hash
    class_hash: [u8; 32],
    /// Provider
    provider: Option<Provider>,
}

impl Account {
    /// Create new account from private key
    pub fn from_private_key(
        private_key: [u8; 32],
        account_type: AccountType,
        class_hash: [u8; 32],
    ) -> Result<Self, AccountError> {
        let key_pair = KeyPair::from_private_key(private_key)
            .map_err(|e| AccountError::Signing(e.to_string()))?;
        
        // Derive address from public key based on account type
        let address = match account_type {
            AccountType::OpenZeppelin | AccountType::Generic => {
                StarknetAddress::from_public_key(&key_pair.public_key())
            }
            AccountType::ArgentX => {
                // Argent X uses different address derivation
                StarknetAddress::from_public_key(&key_pair.public_key())
            }
            AccountType::Braavos => {
                // Braavos uses proxy pattern
                StarknetAddress::from_public_key(&key_pair.public_key())
            }
        };
        
        Ok(Self {
            address,
            key_pair,
            account_type,
            class_hash,
            provider: None,
        })
    }
    
    /// Create with provider
    pub fn with_provider(mut self, provider: Provider) -> Self {
        self.provider = Some(provider);
        self
    }
    
    /// Get address
    pub fn address(&self) -> &StarknetAddress {
        &self.address
    }
    
    /// Get public key
    pub fn public_key(&self) -> [u8; 32] {
        self.key_pair.public_key()
    }
    
    /// Get private key reference
    pub fn private_key(&self) -> [u8; 32] {
        self.key_pair.private_key()
    }
    
    /// Sign a message
    pub fn sign(&self, message: &[u8]) -> Result<Signature, AccountError> {
        self.key_pair.sign(message).map_err(|e| AccountError::Signing(e.to_string()))
    }
    
    /// Verify signature
    pub fn verify(&self, message: &[u8], signature: &Signature) -> bool {
        self.key_pair.verify(message, signature)
    }
    
    /// Get nonce from network
    pub async fn get_nonce(&self) -> Result<[u8; 32], AccountError> {
        let provider = self.provider.as_ref()
            .ok_or(AccountError::NoProvider)?;
        
        provider.get_nonce(&self.address).await
            .map_err(|e| AccountError::Provider(e.to_string()))
    }
    
    /// Execute a call
    pub async fn execute(
        &self,
        calls: Vec<FunctionCall>,
    ) -> Result<InvokeResult, AccountError> {
        let provider = self.provider.as_ref()
            .ok_or(AccountError::NoProvider)?;
        
        let nonce = self.get_nonce().await?;
        let invoke = InvokeTransaction {
            contract_address: calls[0].contract_address.clone(),
            entry_point_selector: calls[0].entry_point_selector.clone(),
            calldata: calls[0].calldata.clone(),
            max_fee: "0x0".to_string(),
            version: "0x0".to_string(),
            nonce: hex::encode(nonce),
            signature: None,
        };
        provider.add_invoke_transaction(invoke)
            .await
            .map_err(|e| AccountError::Provider(e.to_string()))
    }
    
    /// Execute multiple calls (multicall)
    pub async fn execute_multicall(
        &self,
        calls: Vec<FunctionCall>,
    ) -> Result<InvokeResult, AccountError> {
        // Use multicall contract
        // For now: execute sequentially
        
        if calls.is_empty() {
            return Err(AccountError::NoCalls);
        }
        
        self.execute(calls).await
    }
    
    /// Transfer native tokens (ETH)
    pub async fn transfer(
        &self,
        recipient: &StarknetAddress,
        amount: [u8; 32],
    ) -> Result<InvokeResult, AccountError> {
        let provider = self.provider.as_ref()
            .ok_or(AccountError::NoProvider)?;
        
        // Get ETH contract address (same on all Starknet chains)
        let eth_address = StarknetAddress::from_hex(
            "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc"
        ).map_err(|e| AccountError::Address(e.to_string()))?;
        
        // ERC-20 transfer selector (transfer selector)
        let selector: [u8; 32] = [
            0x84, 0x7e, 0x3d, 0x04, 0x9c, 0x6f, 0x47, 0x7a,
            0xda, 0xf9, 0xc4, 0x5c, 0xd9, 0x20, 0x46, 0x4e,
            0x9e, 0x43, 0x2e, 0x0c, 0xd5, 0x1f, 0x4a, 0xf3,
            0xf2, 0xdc, 0x0a, 0x4e, 0xd1, 0x8e, 0x1f, 0x1e
        ]; // transfer selector hash
        
        // Calldata: [recipient, amount]
        let mut calldata = vec![0u8; 32];
        calldata[0] = recipient.to_felt252()[31];
        let mut amount_le = amount;
        amount_le.reverse();
        calldata.push(amount_le[0]);
        
        let call = FunctionCall {
            contract_address: eth_address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![recipient.to_hex(), hex::encode(amount)],
        };
        
        self.execute(vec![call]).await
    }
    
    /// Deploy account to network
    pub async fn deploy(&self) -> Result<DeployResult, AccountError> {
        let provider = self.provider.as_ref()
            .ok_or(AccountError::NoProvider)?;
        
        // Build deploy account transaction
        let mut deploy = DeployAccountTransaction::new(
            self.class_hash,
            vec![self.key_pair.public_key()], // Constructor: public key
        );
        
        // Sign
        deploy.sign(&self.key_pair);
        
        // Send
        provider.add_deploy_account_transaction(deploy)
            .await
            .map_err(|e| AccountError::Provider(e.to_string()))
    }
    
    /// Estimate fee for transaction
    pub async fn estimate_fee(&self, calls: Vec<FunctionCall>) -> Result<FeeEstimate, AccountError> {
        let provider = self.provider.as_ref()
            .ok_or(AccountError::NoProvider)?;
        
        let nonce = self.get_nonce().await?;
        
        let invokes: Vec<InvokeTransaction> = calls.iter().map(|call| {
            InvokeTransaction {
                contract_address: call.contract_address.clone(),
                entry_point_selector: call.entry_point_selector.clone(),
                calldata: call.calldata.clone(),
                max_fee: "0x0".to_string(),
                version: "0x0".to_string(),
                nonce: hex::encode(nonce),
                signature: None,
            }
        }).collect();
        
        let estimates = provider.estimate_fee(&invokes, None)
            .await
            .map_err(|e| AccountError::Provider(e.to_string()))?;
        
        Ok(estimates.into_iter().next()
            .ok_or(AccountError::EstimationFailed)?)
    }
    
    /// Build a function call
    pub fn call(
        &self,
        contract: StarknetAddress,
        selector: &str,
        calldata: Vec<String>,
    ) -> FunctionCall {
        FunctionCall {
            contract_address: contract.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata,
        }
    }
}

/// Account errors
#[derive(Debug)]
pub enum AccountError {
    NoProvider,
    NoCalls,
    Address(String),
    Provider(String),
    Signing(String),
    EstimationFailed,
}

impl fmt::Display for AccountError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AccountError::NoProvider => write!(f, "No provider configured"),
            AccountError::NoCalls => write!(f, "No calls provided"),
            AccountError::Address(e) => write!(f, "Address error: {}", e),
            AccountError::Provider(e) => write!(f, "Provider error: {}", e),
            AccountError::Signing(e) => write!(f, "Signing error: {}", e),
            AccountError::EstimationFailed => write!(f, "Fee estimation failed"),
        }
    }
}

impl std::error::Error for AccountError {}
