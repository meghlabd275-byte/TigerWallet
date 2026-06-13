//! Forwarder service for EIP-2771

use crate::GaslessError;
use std::collections::HashMap;
use tokio::sync::RwLock;

/// Forwarder service
pub struct ForwarderService {
    /// Trusted forwarders
    forwarders: HashMap<String, bool>,
}

impl ForwarderService {
    pub fn new() -> Self {
        Self {
            forwarders: HashMap::new(),
        }
    }

    /// Add forwarder
    pub async fn add(&mut self, address: &str) -> Result<(), GaslessError> {
        self.forwarders.insert(address.to_string(), true);
        Ok(())
    }

    /// Remove forwarder
    pub async fn remove(&mut self, address: &str) -> Result<(), GaslessError> {
        self.forwarders.remove(address);
        Ok(())
    }

    /// Is trusted
    pub async fn is_trusted(&self, address: &str) -> bool {
        self.forwarders.get(address).copied().unwrap_or(false)
    }
}

impl Default for ForwarderService {
    fn default() -> Self {
        Self::new()
    }
}