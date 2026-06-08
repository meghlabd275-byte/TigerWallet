//! TigerWallet Multi-Device Sync
use serde::{Deserialize, Serialize};
use parking_lot::RwLock;
use std::collections::HashMap;
use uuid::Uuid;
use chrono::{DateTime, Utc};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Device {
    pub id: String,
    pub user_id: String,
    pub name: String,
    pub platform: String,
    pub last_sync: DateTime<Utc>,
    pub verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncState {
    pub user_id: String,
    pub device_id: String,
    pub state_hash: String,
    pub timestamp: i64,
    pub version: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Conflict {
    pub key: String,
    pub local_value: String,
    pub remote_value: String,
    pub resolution: ConflictResolution,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum ConflictResolution {
    Local,
    Remote,
    Merge,
    Manual,
}

pub struct SyncEngine {
    devices: RwLock<HashMap<String, Device>>,
    states: RwLock<HashMap<String, SyncState>>,
}

impl SyncEngine {
    pub fn new() -> Self {
        Self {
            devices: RwLock::new(HashMap::new()),
            states: RwLock::new(HashMap::new()),
        }
    }

    pub fn register_device(&self, user_id: &str, name: &str, platform: &str) -> Device {
        let device = Device {
            id: Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            name: name.to_string(),
            platform: platform.to_string(),
            last_sync: Utc::now(),
            verified: false,
        };

        self.devices.write().insert(device.id.clone(), device.clone());
        device
    }

    pub fn sync_state(&self, user_id: &str, device_id: &str, new_hash: &str) -> Result<SyncState, SyncError> {
        let key = format!("{}:{}", user_id, device_id);
        let states = self.states.read();
        
        if let Some(current) = states.get(&key) {
            if current.state_hash != new_hash {
                return Err(SyncError::Conflict(current.clone()));
            }
        }
        
        let new_state = SyncState {
            user_id: user_id.to_string(),
            device_id: device_id.to_string(),
            state_hash: new_hash.to_string(),
            timestamp: Utc::now().timestamp(),
            version: states.get(&key).map(|s| s.version + 1).unwrap_or(1),
        };
        
        drop(states);
        self.states.write().insert(key, new_state.clone());
        
        Ok(new_state)
    }

    pub fn resolve_conflict(&self, conflict: &Conflict) -> ConflictResolution {
        match conflict.resolution {
            ConflictResolution::Manual => ConflictResolution::Manual,
            _ => conflict.resolution,
        }
    }
}

impl Default for SyncEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, thiserror::Error)]
pub enum SyncError {
    #[error("Conflict detected: {0:?}")]
    Conflict(SyncState),
    
    #[error("Device not found")]
    DeviceNotFound,
}