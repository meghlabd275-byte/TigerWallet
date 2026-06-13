//! WalletConnect Protocol Implementation

use crate::error::Error;
use crate::models::*;
use chrono::{DateTime, Utc};
use std::sync::{Arc, RwLock};
use std::collections::HashMap;
use uuid::Uuid;

/// WalletConnect Service
pub struct WalletConnectService {
    sessions: RwLock<HashMap<String, WalletConnectSession>>,
}

impl WalletConnectService {
    pub fn new() -> Self {
        Self {
            sessions: RwLock::new(HashMap::new()),
        }
    }

    pub fn create_session(&self, peer_meta: PeerMeta, accounts: Vec<String>, chain_id: u64) -> Result<WalletConnectSession, Error> {
        let session = WalletConnectSession {
            topic: Uuid::new_v4().to_string(),
            peer_meta,
            accounts,
            chain_id,
            created_at: Utc::now(),
        };
        
        let mut sessions = self.sessions.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        sessions.insert(session.topic.clone(), session.clone());
        
        Ok(session)
    }

    pub fn get_session(&self, topic: &str) -> Result<WalletConnectSession, Error> {
        let sessions = self.sessions.read()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        sessions.get(topic)
            .cloned()
            .ok_or_else(|| Error::NotFound(format!("Session not found: {}", topic)))
    }

    pub fn update_session(&self, topic: &str, accounts: Vec<String>, chain_id: u64) -> Result<WalletConnectSession, Error> {
        let mut sessions = self.sessions.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        if let Some(session) = sessions.get_mut(topic) {
            session.accounts = accounts;
            session.chain_id = chain_id;
            return Ok(session.clone());
        }
        
        Err(Error::NotFound(format!("Session not found: {}", topic)))
    }

    pub fn delete_session(&self, topic: &str) -> Result<(), Error> {
        let mut sessions = self.sessions.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        sessions.remove(topic)
            .ok_or_else(|| Error::NotFound(format!("Session not found: {}", topic)))?;
        
        Ok(())
    }
}

impl Default for WalletConnectService {
    fn default() -> Self {
        Self::new()
    }
}