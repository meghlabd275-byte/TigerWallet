//! TigerWallet Account Abstraction Service
//! 
//! ERC-4337 compatible account abstraction with social recovery,
//! multi-sig support, gasless transactions, and institutional custody.
//!
//! # Features
//! - ERC-4337 EntryPoint implementation
//! - Smart Contract Wallet with social recovery
//! - Account Factory with deterministic addresses
//! - Paymaster for gas sponsorship
//! - Bundler for user operation execution
//! - Multi-sig support
//! - Social recovery with guardians
//! - Time-lock recovery
//! - Anti-scam protection
//! - Institutional MPC custody integration

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc};

pub mod account;
pub mod bundler;
pub mod tx;
pub mod paymaster;
pub mod recovery;
pub mod multisig;
pub mod security;

pub use account::*;
pub use bundler::*;
pub use paymaster::*;
pub use recovery::*;
pub use multisig::*;
pub use security::*;

/// Account Abstraction Service
pub struct AccountAbstractionService {
    /// Entry point contract address
    entry_point: RwLock<Option<String>>,
    
    /// Account factory
    factory: RwLock<AccountFactory>,
    
    /// Bundler service
    bundler: RwLock<BundlerService>,
    
    /// Paymaster service
    paymaster: RwLock<PaymasterService>,
    
    /// Recovery service
    recovery: RwLock<RecoveryService>,
    
    /// Multi-sig service
    multisig: RwLock<MultiSigService>,
    
    /// Security module
    security: Arc<RwLock<SecurityModule>>,
    
    /// Account cache
    accounts: RwLock<HashMap<String, SmartAccount>>,
    
    /// Chain ID
    chain_id: u64,
}

impl AccountAbstractionService {
    /// Create new account abstraction service
    pub fn new(chain_id: u64) -> Self {
        Self {
            entry_point: RwLock::new(None),
            factory: RwLock::new(AccountFactory::new()),
            bundler: RwLock::new(BundlerService::new(chain_id)),
            paymaster: RwLock::new(PaymasterService::new()),
            recovery: RwLock::new(RecoveryService::new()),
            multisig: RwLock::new(MultiSigService::new()),
            security: Arc::new(RwLock::new(SecurityModule::new())),
            accounts: RwLock::new(HashMap::new()),
            chain_id,
        }
    }

    /// Configure the RPC endpoint and signing key used to submit bundles.
    ///
    /// Without both, `submit_user_op` fails closed rather than reporting an
    /// operation as submitted.
    pub async fn configure_bundler(
        &self,
        rpc_url: &str,
        private_key: &str,
    ) -> Result<(), AAError> {
        let signer = crate::tx::TxSigner::from_hex(private_key)?;

        let mut bundler = self.bundler.write().await;
        bundler.set_rpc_endpoint(rpc_url);
        bundler.set_signer(signer);
        Ok(())
    }

    /// Initialize with entry point address
    ///
    /// The address is stored on the service and on the bundler, which needs it
    /// to compute operation hashes and to address `handleOps` calls.
    pub async fn initialize(&self, entry_point: String) -> Result<(), AAError> {
        let mut ep = self.entry_point.write().await;
        *ep = Some(entry_point.clone());

        self.bundler.write().await.set_entry_point(entry_point);
        Ok(())
    }

    /// Configure the account factory used to derive account addresses.
    ///
    /// Without a factory, `create_account` fails closed rather than deriving an
    /// address that no contract would ever deploy at.
    pub async fn configure_factory(
        &self,
        factory_address: String,
        rpc_url: &str,
    ) -> Result<(), AAError> {
        let mut factory = self.factory.write().await;
        factory.set_factory_address(factory_address);
        factory.set_rpc_endpoint(rpc_url);
        Ok(())
    }

    /// Create new smart account
    pub async fn create_account(
        &self,
        owners: Vec<String>,
        threshold: u32,
        salt: u64,
    ) -> Result<AccountInfo, AAError> {
        // Validate inputs
        if owners.is_empty() {
            return Err(AAError::InvalidOwners);
        }
        if threshold == 0 || threshold > owners.len() as u32 {
            return Err(AAError::InvalidThreshold);
        }

        // Generate account address
        let account_address = self
            .factory
            .read()
            .await
            .compute_address(&owners[0], salt, self.chain_id)
            .await?;

        // Create account metadata
        let account_id = Uuid::new_v4().to_string();
        let account_info = AccountInfo {
            account_id: account_id.clone(),
            address: account_address.clone(),
            owners: owners.clone(),
            threshold,
            created_at: Utc::now(),
            entry_point: self.entry_point.read().await.clone(),
            chain_id: self.chain_id,
            nonce: 0,
            deposit: 0,
            locked: false,
            guard_ians: vec![],
            recovery_threshold: 2,
            daily_limit: 0,
            whitelisted_calls: vec![],
        };

        // The init code lets the first user operation deploy the account.
        let init_data = self
            .factory
            .read()
            .await
            .compute_init_code(&owners[0], salt)?;

        // Register a signing key for each owner, so the owner can sign user
        // operations. Without this, signing would fail closed.
        {
            let mut security = self.security.write().await;
            for owner in &owners {
                if !security.has_key(owner) {
                    security.generate_key_pair(owner)?;
                }
            }
        }

        let mut smart_account = SmartAccount::new(owners, threshold);
        smart_account.init_data = init_data;

        self.accounts
            .write()
            .await
            .insert(account_address.clone(), smart_account);

        Ok(account_info)
    }

    /// Get account info
    pub async fn get_account(&self, address: &str) -> Result<AccountInfo, AAError> {
        let accounts = self.accounts.read().await;
        
        if let Some(account) = accounts.get(address) {
            Ok(AccountInfo {
                account_id: account.id.clone(),
                address: address.to_string(),
                owners: account.owners.clone(),
                threshold: account.threshold,
                created_at: account.created_at,
                entry_point: self.entry_point.read().await.clone(),
                chain_id: self.chain_id,
                nonce: account.nonce,
                deposit: account.deposit,
                locked: account.locked,
                guard_ians: account.guardians.clone(),
                recovery_threshold: account.recovery_threshold,
                daily_limit: account.daily_limit,
                whitelisted_calls: account.whitelisted_calls.clone(),
            })
        } else {
            Err(AAError::AccountNotFound)
        }
    }

    /// Build user operation
    pub async fn build_user_op(
        &self,
        account: &str,
        calls: Vec<Call>,
        options: UserOpOptions,
    ) -> Result<UserOperation, AAError> {
        let accounts = self.accounts.read().await;
        
        let account_data = accounts.get(account)
            .ok_or(AAError::AccountNotFound)?;
        
        // Build init code if account not deployed
        let init_code = if !account_data.deployed {
            let factory = self.factory.read().await.get_factory_address();
            Some(FactoryCall {
                factory,
                data: account_data.init_data.clone(),
            })
        } else {
            None
        };

        // Build user operation
        let user_op = UserOperation {
            sender: account.to_string(),
            nonce: account_data.nonce,
            init_code,
            call_data: Some(encode_calls(&calls)),
            call_gas_limit: options.call_gas_limit.unwrap_or(100000),
            verification_gas_limit: options.verification_gas_limit.unwrap_or(200000),
            pre_verification_gas: options.pre_verification_gas.unwrap_or(50000),
            max_fee_per_gas: options.max_fee_per_gas.unwrap_or(0),
            max_priority_fee_per_gas: options.max_priority_fee_per_gas.unwrap_or(0),
            paymaster_data: options.paymaster_data,
            signature: vec![],
        };

        Ok(user_op)
    }

    /// Sign user operation
    pub async fn sign_user_op(
        &self,
        user_op: &mut UserOperation,
        signer: &str,
    ) -> Result<Vec<u8>, AAError> {
        // Get account
        let accounts = self.accounts.read().await;
        let account = accounts.get(&user_op.sender)
            .ok_or(AAError::AccountNotFound)?;
        
        // Verify signer is owner
        if !account.owners.contains(&signer.to_string()) {
            return Err(AAError::NotOwner);
        }

        // Get operation hash
        let op_hash = self.get_operation_hash(user_op).await?;
        
        // Sign using security module
        let signature = self.security.read().await.sign(&op_hash, signer).await?;
        
        user_op.signature = signature.clone();
        
        Ok(signature)
    }

    /// Submit user operation to bundler
    pub async fn submit_user_op(
        &self,
        user_op: UserOperation,
    ) -> Result<String, AAError> {
        // Validate user operation
        self.validate_user_op(&user_op).await?;
        
        // Submit to bundler
        self.bundler.write().await.send_user_op(user_op).await
    }

    /// Get operation hash
    ///
    /// Delegates to the bundler's encoder so the hash matches the EntryPoint's
    /// `getUserOpHash`.
    pub async fn get_operation_hash(&self, user_op: &UserOperation) -> Result<Vec<u8>, AAError> {
        let bundler = self.bundler.read().await;
        let hex_hash = bundler.compute_op_hash(user_op)?;
        hex::decode(hex_hash).map_err(|e| AAError::BundlerError(e.to_string()))
    }

    /// Validate user operation
    async fn validate_user_op(&self, user_op: &UserOperation) -> Result<(), AAError> {
        // Check sender is valid
        if user_op.sender.is_empty() {
            return Err(AAError::InvalidSender);
        }
        
        // Check gas limits
        if user_op.call_gas_limit < 21000 {
            return Err(AAError::InsufficientGas);
        }
        
        if user_op.verification_gas_limit < 50000 {
            return Err(AAError::InsufficientVerificationGas);
        }
        
        Ok(())
    }

    /// Start social recovery
    pub async fn start_recovery(
        &self,
        account: &str,
        new_owner: &str,
        guardian: &str,
    ) -> Result<RecoveryRequest, AAError> {
        let mut recovery = self.recovery.write().await;
        
        recovery.start_recovery(
            account,
            new_owner,
            guardian,
            self.chain_id,
        ).await
    }

    /// Confirm recovery
    pub async fn confirm_recovery(
        &self,
        account: &str,
        new_owner: &str,
        guardian: &str,
    ) -> Result<bool, AAError> {
        let mut recovery = self.recovery.write().await;
        
        recovery.confirm_recovery(
            account,
            new_owner,
            guardian,
        ).await
    }

    /// Complete recovery
    pub async fn complete_recovery(
        &self,
        account: &str,
    ) -> Result<String, AAError> {
        let mut recovery = self.recovery.write().await;
        recovery.complete_recovery(account).await
    }

    /// Execute multi-sig transaction
    pub async fn execute_multisig(&self, tx_id: &str) -> Result<String, AAError> {
        self.multisig.write().await.execute(tx_id)
    }

    /// Get bundler status
    pub async fn get_bundler_status(&self) -> BundlerStatus {
        self.bundler.read().await.get_status()
    }

    /// Get paymaster deposit
    pub async fn get_paymaster_deposit(&self, account: &str) -> Result<u64, AAError> {
        let paymaster = self.paymaster.read().await;
        paymaster.get_deposit(account).await
    }

    /// Sponsor user operation
    pub async fn sponsor_user_op(
        &self,
        user_op: &UserOperation,
    ) -> Result<SponsoredOp, AAError> {
        let mut paymaster = self.paymaster.write().await;
        paymaster.sponsor(user_op).await
    }
}

/// Account information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountInfo {
    pub account_id: String,
    pub address: String,
    pub owners: Vec<String>,
    pub threshold: u32,
    pub created_at: DateTime<Utc>,
    pub entry_point: Option<String>,
    pub chain_id: u64,
    pub nonce: u64,
    pub deposit: u64,
    pub locked: bool,
    pub guard_ians: Vec<String>,
    pub recovery_threshold: u32,
    pub daily_limit: u64,
    pub whitelisted_calls: Vec<String>,
}

/// Smart account data
#[derive(Debug, Clone)]
pub struct SmartAccount {
    pub id: String,
    pub owners: Vec<String>,
    pub threshold: u32,
    pub nonce: u64,
    pub deposit: u64,
    pub deployed: bool,
    pub init_data: Vec<u8>,
    pub locked: bool,
    pub lock_time: Option<DateTime<Utc>>,
    pub guardians: Vec<String>,
    pub recovery_threshold: u32,
    pub daily_limit: u64,
    pub daily_spent: u64,
    pub daily_reset: DateTime<Utc>,
    pub whitelisted_calls: Vec<String>,
    pub created_at: DateTime<Utc>,
}

impl SmartAccount {
    pub fn new(owners: Vec<String>, threshold: u32) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            owners,
            threshold,
            nonce: 0,
            deposit: 0,
            deployed: false,
            init_data: vec![],
            locked: false,
            lock_time: None,
            guardians: vec![],
            recovery_threshold: 2,
            daily_limit: 0,
            daily_spent: 0,
            daily_reset: Utc::now(),
            whitelisted_calls: vec![],
            created_at: Utc::now(),
        }
    }
}

/// Call structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Call {
    pub to: String,
    pub value: u64,
    pub data: Vec<u8>,
}

/// User operation options
#[derive(Debug, Clone, Default)]
pub struct UserOpOptions {
    pub call_gas_limit: Option<u64>,
    pub verification_gas_limit: Option<u64>,
    pub pre_verification_gas: Option<u64>,
    /// Maximum total fee per gas unit, in wei.
    pub max_fee_per_gas: Option<u64>,
    /// Maximum priority fee per gas unit, in wei.
    pub max_priority_fee_per_gas: Option<u64>,
    pub paymaster_data: Option<Vec<u8>>,
    pub signature: Option<Vec<u8>>,
}

/// Factory call
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactoryCall {
    pub factory: String,
    pub data: Vec<u8>,
}

/// User operation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserOperation {
    pub sender: String,
    pub nonce: u64,
    pub init_code: Option<FactoryCall>,
    pub call_data: Option<Vec<u8>>,
    pub call_gas_limit: u64,
    pub verification_gas_limit: u64,
    pub pre_verification_gas: u64,
    /// Maximum total fee per gas unit, in wei. The EntryPoint packs this into
    /// the low half of `gasFees`.
    pub max_fee_per_gas: u64,
    /// Maximum priority fee per gas unit, in wei. The EntryPoint packs this
    /// into the high half of `gasFees`.
    pub max_priority_fee_per_gas: u64,
    pub paymaster_data: Option<Vec<u8>>,
    pub signature: Vec<u8>,
}

/// Sponsored operation
#[derive(Debug, Clone)]
pub struct SponsoredOp {
    pub user_op: UserOperation,
    pub sponsor_amount: u64,
    pub expires_at: DateTime<Utc>,
}

/// Recovery request
#[derive(Debug, Clone)]
pub struct RecoveryRequest {
    pub id: String,
    pub account: String,
    pub new_owner: String,
    pub guardians: Vec<String>,
    pub confirmations: u32,
    pub threshold: u32,
    pub started_at: DateTime<Utc>,
    pub unlock_time: DateTime<Utc>,
    pub completed: bool,
}

/// Encode calls to bytes
fn encode_calls(calls: &[Call]) -> Vec<u8> {
    let mut data = vec![];
    
    // Encode number of calls
    data.extend_from_slice(&(calls.len() as u32).to_be_bytes());
    
    for call in calls {
        // Encode each call
        data.extend_from_slice(call.to.as_bytes());
        data.extend_from_slice(&call.value.to_be_bytes());
        data.extend_from_slice(&(call.data.len() as u32).to_be_bytes());
        data.extend_from_slice(&call.data);
    }
    
    data
}

/// AA Error types
#[derive(Debug, thiserror::Error)]
pub enum AAError {
    #[error("Account not found")]
    AccountNotFound,
    
    #[error("Invalid owners")]
    InvalidOwners,
    
    #[error("Invalid threshold")]
    InvalidThreshold,
    
    #[error("Not initialized")]
    NotInitialized,
    
    #[error("Invalid sender")]
    InvalidSender,
    
    #[error("Insufficient gas")]
    InsufficientGas,
    
    #[error("Insufficient verification gas")]
    InsufficientVerificationGas,
    
    #[error("Not owner")]
    NotOwner,
    
    #[error("Invalid signature")]
    InvalidSignature,
    
    #[error("Bundler error: {0}")]
    BundlerError(String),
    
    #[error("Paymaster error: {0}")]
    PaymasterError(String),
    
    #[error("Recovery error: {0}")]
    RecoveryError(String),
    
    #[error("Multi-sig error: {0}")]
    MultiSigError(String),
    
    #[error("Security error: {0}")]
    SecurityError(String),

    #[error("Factory error: {0}")]
    FactoryError(String),
}

