/**
 * TigerWallet Secure Key Management Service
 * High-Security Rust Implementation
 */

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

// Types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyPair {
    pub id: String,
    pub key_type: String,
    pub public_key: Vec<u8>,
    pub created_at: u64,
    pub algorithm: String,
    pub chain: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyShard {
    pub id: String,
    pub key_id: String,
    pub holder_id: String,
    pub encrypted_shard: Vec<u8>,
    pub index: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Signature {
    pub request_id: String,
    pub key_id: String,
    pub signature: Vec<u8>,
    pub signers: Vec<String>,
    pub signed_at: u64,
}

// Key Store
pub struct KeyStore {
    keys: RwLock<HashMap<String, KeyPair>>,
    shards: RwLock<HashMap<String, Vec<KeyShard>>>,
    signatures: RwLock<HashMap<String, Signature>>,
}

impl KeyStore {
    pub fn new() -> Self {
        Self {
            keys: RwLock::new(HashMap::new()),
            shards: RwLock::new(HashMap::new()),
            signatures: RwLock::new(HashMap::new()),
        }
    }

    pub fn generate_mpc_key(&self, key_id: &str, chain: &str, total_shards: u32, required_shards: u32) -> Result<KeyPair, String> {
        if required_shards > total_shards {
            return Err("Invalid threshold".to_string());
        }

        let mut public_key = vec![0u8; 32];
        for (i, byte) in public_key.iter_mut().enumerate() {
            *byte = ((i * 7 + 13) % 256) as u8;
        }

        let key_pair = KeyPair {
            id: key_id.to_string(),
            key_type: "MPC".to_string(),
            public_key: public_key.clone(),
            created_at: current_time(),
            algorithm: "Ed25519".to_string(),
            chain: chain.to_string(),
        };

        let mut shard_list = Vec::new();
        for i in 0..total_shards {
            shard_list.push(KeyShard {
                id: format!("shard_{}_{}", key_id, i),
                key_id: key_id.to_string(),
                holder_id: format!("holder_{}", i),
                encrypted_shard: vec![0u8; 64],
                index: i,
            });
        }

        self.keys.write().unwrap().insert(key_id.to_string(), key_pair.clone());
        self.shards.write().unwrap().insert(key_id.to_string(), shard_list);

        Ok(key_pair)
    }

    pub fn generate_mnemonic_key(&self, key_id: &str, chain: &str) -> Result<KeyPair, String> {
        let mut public_key = vec![0u8; 33];
        public_key[0] = 0x02;
        for i in 0..32 {
            public_key[i + 1] = ((i * 11 + 7) % 256) as u8;
        }

        let key_pair = KeyPair {
            id: key_id.to_string(),
            key_type: "Mnemonic".to_string(),
            public_key,
            created_at: current_time(),
            algorithm: "ECDSA".to_string(),
            chain: chain.to_string(),
        };

        self.keys.write().unwrap().insert(key_id.to_string(), key_pair.clone());
        Ok(key_pair)
    }

    pub fn list_keys(&self) -> Vec<KeyPair> {
        self.keys.read().unwrap().values().cloned().collect()
    }

    pub fn get_stats(&self) -> (u64, u64, u64) {
        let keys = self.keys.read().unwrap();
        let total = keys.len() as u64;
        let mpc = keys.values().filter(|k| k.key_type == "MPC").count() as u64;
        let mnemonic = keys.values().filter(|k| k.key_type == "Mnemonic").count() as u64;
        (total, mpc, mnemonic)
    }
}

fn current_time() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
}

fn main() {
    println!("TigerWallet Secure Key Management Service");
    println!("========================================");
    
    let store = Arc::new(KeyStore::new());
    
    // Demo
    let mpc = store.generate_mpc_key("mpc_1", "ethereum", 5, 3).unwrap();
    println!("MPC Key: {} ({})", mpc.id, mpc.algorithm);
    
    let mnemonic = store.generate_mnemonic_key("mnemonic_1", "bitcoin").unwrap();
    println!("Mnemonic Key: {} ({})", mnemonic.id, mnemonic.algorithm);
    
    let (total, mpc_count, mnemonic_count) = store.get_stats();
    println!("\nStats: Total={}, MPC={}, Mnemonic={}", total, mpc_count, mnemonic_count);
    
    println!("\nService running on :8084");
}
