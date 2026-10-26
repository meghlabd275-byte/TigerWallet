//! Account management for Account Abstraction.
//!
//! Addresses are derived the way the EntryPoint's factory derives them, with
//! CREATE2: `keccak256(0xff ++ factory ++ salt ++ keccak256(initCode))[12..]`.
//! The init code is the factory's `createAccount` calldata, so a wallet can
//! show the counterfactual address before the account is deployed.

use crate::AAError;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// `createAccount(address owner, uint256 salt)`.
const CREATE_ACCOUNT_SELECTOR: [u8; 4] = [0x5f, 0xbf, 0xb9, 0xcf];
/// `getAddress(address owner, uint256 salt)`.
const GET_ADDRESS_SELECTOR: [u8; 4] = [0x8c, 0xb8, 0x4e, 0x18];

/// Left-pad a `u128` into a 32-byte word.
fn word_u128(value: u128) -> [u8; 32] {
    let mut out = [0u8; 32];
    out[16..].copy_from_slice(&value.to_be_bytes());
    out
}

/// Decode a `0x`-prefixed address into 20 bytes.
fn decode_address(address: &str) -> Result<[u8; 20], AAError> {
    let stripped = address.strip_prefix("0x").unwrap_or(address);
    if stripped.len() != 40 {
        return Err(AAError::FactoryError(format!(
            "{address} is not a 20-byte address"
        )));
    }

    let bytes = hex::decode(stripped)
        .map_err(|e| AAError::FactoryError(format!("{address} is not valid hex: {e}")))?;

    let mut out = [0u8; 20];
    out.copy_from_slice(&bytes);
    Ok(out)
}

/// Account factory for deterministic address generation.
///
/// The canonical `SimpleAccountFactory` derives an account address from
/// `CREATE2(salt, keccak256(abi.encodePacked(proxyCreationCode, abi.encode(
/// implementation, abi.encodeCall(initialize, owner)))), factory)`. The proxy
/// creation code is large contract bytecode, so the address is resolved by
/// asking the deployed factory rather than by reimplementing that encoding
/// off-chain. When no factory is reachable, address derivation fails closed
/// instead of returning an address that no contract would deploy at.
#[derive(Default)]
pub struct AccountFactory {
    factory_address: Option<String>,
    rpc: Option<crate::bundler::JsonRpcClient>,
}

impl AccountFactory {
    pub fn new() -> Self {
        Self {
            factory_address: None,
            rpc: None,
        }
    }

    /// Set the factory address.
    pub fn set_factory_address(&mut self, address: String) {
        self.factory_address = Some(address);
    }

    /// Get the factory address.
    pub fn get_factory_address(&self) -> String {
        self.factory_address.clone().unwrap_or_default()
    }

    /// Set the JSON-RPC endpoint used to query the factory.
    pub fn set_rpc_endpoint(&mut self, url: impl Into<String>) {
        self.rpc = Some(crate::bundler::JsonRpcClient::new(url));
    }

    /// Whether an address can be resolved.
    pub fn can_resolve(&self) -> bool {
        self.factory_address.is_some() && self.rpc.is_some()
    }

    /// The factory address, or an error when it has not been configured.
    fn factory(&self) -> Result<[u8; 20], AAError> {
        let address = self.factory_address.as_deref().ok_or_else(|| {
            AAError::FactoryError("no account factory address is configured".to_string())
        })?;

        decode_address(address)
    }

    /// `createAccount(owner, salt)` calldata for a single owner.
    pub fn create_account_calldata(&self, owner: &str, salt: u64) -> Result<Vec<u8>, AAError> {
        let owner = decode_address(owner)?;

        let mut data = Vec::with_capacity(4 + 64);
        data.extend_from_slice(&CREATE_ACCOUNT_SELECTOR);

        // The address is left-padded to a word.
        let mut owner_word = [0u8; 32];
        owner_word[12..].copy_from_slice(&owner);
        data.extend_from_slice(&owner_word);

        data.extend_from_slice(&word_u128(salt as u128));
        Ok(data)
    }

    /// `getAddress(owner, salt)` calldata.
    pub fn get_address_calldata(&self, owner: &str, salt: u64) -> Result<Vec<u8>, AAError> {
        let owner = decode_address(owner)?;

        let mut data = Vec::with_capacity(4 + 64);
        data.extend_from_slice(&GET_ADDRESS_SELECTOR);

        let mut owner_word = [0u8; 32];
        owner_word[12..].copy_from_slice(&owner);
        data.extend_from_slice(&owner_word);

        data.extend_from_slice(&word_u128(salt as u128));
        Ok(data)
    }

    /// The init code the first user operation carries: the factory address
    /// followed by its `createAccount` calldata.
    pub fn compute_init_code(&self, owner: &str, salt: u64) -> Result<Vec<u8>, AAError> {
        let factory = self.factory()?;

        let mut data = Vec::with_capacity(20 + 4 + 64);
        data.extend_from_slice(&factory);
        data.extend_from_slice(&self.create_account_calldata(owner, salt)?);
        Ok(data)
    }

    /// The counterfactual account address, as the deployed factory computes it.
    pub async fn compute_address(
        &self,
        owner: &str,
        salt: u64,
        _chain_id: u64,
    ) -> Result<String, AAError> {
        let factory = self.factory_address.as_deref().ok_or_else(|| {
            AAError::FactoryError("no account factory address is configured".to_string())
        })?;

        let rpc = self.rpc.as_ref().ok_or_else(|| {
            AAError::FactoryError(
                "no JSON-RPC endpoint is configured, so the account address cannot be resolved"
                    .to_string(),
            )
        })?;

        let calldata = self.get_address_calldata(owner, salt)?;
        let return_data = rpc
            .call_contract(factory, &format!("0x{}", hex::encode(&calldata)))
            .await?;

        parse_address_result(&return_data)
    }
}

/// Decode an ABI-encoded `address` return value.
fn parse_address_result(return_data: &str) -> Result<String, AAError> {
    let stripped = return_data.strip_prefix("0x").unwrap_or(return_data);

    // A returned address occupies the low 20 bytes of a 32-byte word.
    if stripped.len() < 64 {
        return Err(AAError::FactoryError(format!(
            "getAddress returned {} hex digits, expected at least 64",
            stripped.len()
        )));
    }

    let word = &stripped[..64];
    let address = &word[24..64];

    // An all-zero address means the factory has no implementation configured.
    if address.chars().all(|c| c == '0') {
        return Err(AAError::FactoryError(
            "the factory returned the zero address".to_string(),
        ));
    }

    Ok(format!("0x{address}"))
}

/// Account state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountState {
    pub address: String,
    pub nonce: u64,
    pub deposit: u64,
    pub locked: bool,
    pub lock_time: Option<DateTime<Utc>>,
    pub owners: Vec<String>,
    pub threshold: u32,
    pub guardians: Vec<String>,
    pub recovery_threshold: u32,
    pub daily_limit: u64,
    pub daily_spent: u64,
    pub last_reset: DateTime<Utc>,
    pub whitelisted_calls: Vec<String>,
    pub deployed: bool,
    pub created_at: DateTime<Utc>,
}

impl AccountState {
    pub fn new(address: String, owners: Vec<String>, threshold: u32) -> Self {
        Self {
            address,
            nonce: 0,
            deposit: 0,
            locked: false,
            lock_time: None,
            owners,
            threshold,
            guardians: vec![],
            recovery_threshold: 2,
            daily_limit: 0,
            daily_spent: 0,
            last_reset: Utc::now(),
            whitelisted_calls: vec![],
            deployed: false,
            created_at: Utc::now(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const FACTORY: &str = "0x9406Cc6185a346906296840746125a0E44976454";
    const OWNER: &str = "0x742d35Cc6634C0532925a3b844Bc454e4438f44e";

    fn factory() -> AccountFactory {
        let mut factory = AccountFactory::new();
        factory.set_factory_address(FACTORY.to_string());
        factory
    }

    #[test]
    fn an_unconfigured_factory_fails_closed() {
        let factory = AccountFactory::new();

        // With no factory address there is no init code to build.
        assert!(factory.compute_init_code(OWNER, 0).is_err());
        assert!(!factory.can_resolve());
    }

    #[tokio::test]
    async fn resolving_without_an_rpc_endpoint_fails_closed() {
        // Without a reachable node there is no correct address to return, so
        // this must error rather than fabricate one.
        assert!(factory().compute_address(OWNER, 0, 1).await.is_err());
    }

    #[test]
    fn a_factory_with_an_endpoint_reports_it_can_resolve() {
        let mut factory = factory();
        assert!(!factory.can_resolve());
        factory.set_rpc_endpoint("http://127.0.0.1:8545");
        assert!(factory.can_resolve());
    }

    #[test]
    fn a_malformed_owner_is_rejected() {
        let factory = factory();
        assert!(factory.create_account_calldata("0x1234", 0).is_err());
        assert!(factory.get_address_calldata("0x1234", 0).is_err());
        assert!(factory.compute_init_code("0x1234", 0).is_err());
    }

    #[test]
    fn init_code_is_the_factory_then_its_calldata() {
        let init_code = factory().compute_init_code(OWNER, 7).unwrap();

        assert_eq!(&init_code[..20], &decode_address(FACTORY).unwrap());
        assert_eq!(&init_code[20..24], &CREATE_ACCOUNT_SELECTOR);
        assert_eq!(init_code.len(), 20 + 4 + 64);
    }

    #[test]
    fn the_calldata_encodes_the_owner_and_salt() {
        let data = factory().create_account_calldata(OWNER, 7).unwrap();

        assert_eq!(&data[..4], &CREATE_ACCOUNT_SELECTOR);
        // The owner sits in the low 20 bytes of the first word.
        assert_eq!(&data[16..36], &decode_address(OWNER).unwrap());

        // The salt word follows the owner word; its value sits in the low 16
        // bytes.
        let salt = u128::from_be_bytes(data[52..68].try_into().unwrap());
        assert_eq!(salt, 7);
    }

    #[test]
    fn get_address_calldata_uses_its_own_selector() {
        let data = factory().get_address_calldata(OWNER, 0).unwrap();
        assert_eq!(&data[..4], &GET_ADDRESS_SELECTOR);
    }

    #[test]
    fn a_returned_address_is_decoded_from_its_word() {
        // The factory returns the address in the low 20 bytes of a word.
        let encoded = format!(
            "0x000000000000000000000000{}",
            "2bea269f9305577f5a9f6a9fdf04c969368879e7"
        );

        assert_eq!(
            parse_address_result(&encoded).unwrap(),
            "0x2bea269f9305577f5a9f6a9fdf04c969368879e7"
        );
    }

    #[test]
    fn a_short_return_value_is_rejected() {
        assert!(parse_address_result("0x1234").is_err());
    }

    #[test]
    fn a_zero_address_return_is_rejected() {
        assert!(parse_address_result(&format!("0x{}", "0".repeat(64))).is_err());
    }
}
