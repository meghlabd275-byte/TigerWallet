//! Permission management for White Level SDK

use crate::types::*;
use crate::errors::{Result, SdkError};
use crate::config::Config;
use parking_lot::RwLock;
use std::collections::HashMap;
use std::sync::Arc;

/// Permission manager
pub struct PermissionManager {
    config: Config,
    permissions: Arc<RwLock<HashMap<(WhiteLevelProduct, FetcherType), Permission>>>,
    http_client: reqwest::Client,
}

impl PermissionManager {
    /// Create new permission manager
    pub fn new(config: Config) -> Self {
        Self {
            config,
            permissions: Arc::new(RwLock::new(HashMap::new())),
            http_client: reqwest::Client::new(),
        }
    }

    /// Load permissions from Super Admin
    pub async fn load_permissions(&self, client_id: &str) -> Result<Vec<Permission>> {
        let url = format!("{}/api/v1/permissions/{}", self.config.super_admin_url, client_id);
        
        let response = self.http_client
            .get(&url)
            .header("X-API-Key", &self.config.api_key)
            .send()
            .await?;

        if !response.status().is_success() {
            let error_text = response.text().await.unwrap_or_default();
            return Err(SdkError::PermissionDenied(error_text));
        }

        let permissions: Vec<Permission> = response.json().await?;

        // Update cache
        {
            let mut perms = self.permissions.write();
            for perm in &permissions {
                perms.insert((perm.product, perm.fetcher), perm.clone());
            }
        }

        Ok(permissions)
    }

    /// Check if client has permission for specific fetcher
    pub fn has_permission(&self, product: WhiteLevelProduct, fetcher: FetcherType, required_level: PermissionLevel) -> bool {
        let perms = self.permissions.read();
        
        if let Some(perm) = perms.get(&(product, fetcher)) {
            return perm.is_enabled && self.level_sufficient(perm.level, required_level);
        }
        
        false
    }

    /// Check permission for specific product
    pub fn check_product_permission(&self, product: WhiteLevelProduct, level: PermissionLevel) -> Vec<FetcherType> {
        let perms = self.permissions.read();
        let mut allowed = Vec::new();

        for ((prod, fetcher), perm) in perms.iter() {
            if *prod == product && perm.is_enabled && self.level_sufficient(perm.level, level) {
                allowed.push(*fetcher);
            }
        }

        allowed
    }

    /// Get all enabled fetchers for product
    pub fn get_enabled_fetchers(&self, product: WhiteLevelProduct) -> Vec<FetcherType> {
        let perms = self.permissions.read();
        let mut fetchers = Vec::new();

        for ((prod, fetcher), perm) in perms.iter() {
            if *prod == product && perm.is_enabled {
                fetchers.push(*fetcher);
            }
        }

        fetchers
    }

    /// Update single permission (from event)
    pub fn update_permission(&self, permission: Permission) {
        let mut perms = self.permissions.write();
        perms.insert((permission.product, permission.fetcher), permission);
    }

    /// Clear all permissions
    pub fn clear(&self) {
        let mut perms = self.permissions.write();
        perms.clear();
    }

    /// Get all permissions
    pub fn get_all(&self) -> Vec<Permission> {
        let perms = self.permissions.read();
        perms.values().cloned().collect()
    }

    /// Helper to compare permission levels
    fn level_sufficient(&self, actual: PermissionLevel, required: PermissionLevel) -> bool {
        use PermissionLevel::*;
        
        match (actual, required) {
            (_, None) => true,
            (SuperAdmin, _) => true,
            (Admin, Admin) => true,
            (Admin, Execute) => true,
            (Admin, Write) => true,
            (Admin, Read) => true,
            (Execute, Execute) => true,
            (Execute, Write) => true,
            (Execute, Read) => true,
            (Write, Write) => true,
            (Write, Read) => true,
            (Read, Read) => true,
            _ => false,
        }
    }
}

/// Default permission levels for products
pub fn get_default_permissions(product: WhiteLevelProduct) -> Vec<Permission> {
    match product {
        WhiteLevelProduct::MasterWallet => vec![
            Permission { product, fetcher: FetcherType::Prices, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::Balances, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::Transactions, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::UserData, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::MarketData, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::Blockchain, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::TokenInfo, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::KYC, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::NftData, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::GasPrice, level: PermissionLevel::SuperAdmin, is_enabled: true },
            Permission { product, fetcher: FetcherType::NetworkStatus, level: PermissionLevel::SuperAdmin, is_enabled: true },
        ],
        WhiteLevelProduct::UserWallet => vec![
            Permission { product, fetcher: FetcherType::Prices, level: PermissionLevel::Read, is_enabled: true },
            Permission { product, fetcher: FetcherType::Balances, level: PermissionLevel::Read, is_enabled: true },
            Permission { product, fetcher: FetcherType::Transactions, level: PermissionLevel::Write, is_enabled: true },
            Permission { product, fetcher: FetcherType::UserData, level: PermissionLevel::Read, is_enabled: true },
        ],
        WhiteLevelProduct::Bots | WhiteLevelProduct::BotsClients => vec![
            Permission { product, fetcher: FetcherType::Prices, level: PermissionLevel::Read, is_enabled: true },
            Permission { product, fetcher: FetcherType::MarketData, level: PermissionLevel::Read, is_enabled: true },
            Permission { product, fetcher: FetcherType::Blockchain, level: PermissionLevel::Execute, is_enabled: true },
        ],
        WhiteLevelProduct::ProjectParty => vec![
            Permission { product, fetcher: FetcherType::TokenInfo, level: PermissionLevel::Admin, is_enabled: true },
            Permission { product, fetcher: FetcherType::MarketData, level: PermissionLevel::Read, is_enabled: true },
            Permission { product, fetcher: FetcherType::Blockchain, level: PermissionLevel::Read, is_enabled: true },
            Permission { product, fetcher: FetcherType::KYC, level: PermissionLevel::Admin, is_enabled: true },
        ],
    }
}
