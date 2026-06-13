//! API Gateway Handlers

use crate::error::Error;
use crate::models::*;
use chrono::{DateTime, Utc};
use std::sync::{Arc, RwLock};
use std::collections::HashMap;
use uuid::Uuid;

/// Rate Limiter
pub struct RateLimiter {
    requests: RwLock<HashMap<String, Vec<DateTime>>>,
    limit: u64,
}

impl RateLimiter {
    pub fn new(limit: u64) -> Self {
        Self {
            requests: RwLock::new(HashMap::new()),
            limit,
        }
    }

    pub fn check(&self, client_id: &str) -> Result<(), Error> {
        let mut requests = self.requests.write().unwrap();
        let now = Utc::now();
        
        let client_requests = requests.entry(client_id.to_string()).or_insert_with(Vec::new);
        client_requests.retain(|t| (now - *t).num_seconds() < 60);
        
        if client_requests.len() as u64 >= self.limit {
            return Err(Error::RateLimitExceeded);
        }
        
        client_requests.push(now);
        Ok(())
    }
}

/// Request Router
pub struct Router {
    routes: RwLock<HashMap<String, String>>,
}

impl Router {
    pub fn new() -> Self {
        let router = Self {
            routes: RwLock::new(HashMap::new()),
        };
        router.initialize_routes();
        router
    }

    fn initialize_routes(&self) {
        let mut routes = self.routes.write().unwrap();
        
        routes.insert("/api/v1/auth".to_string(), "auth_service".to_string());
        routes.insert("/api/v1/wallet".to_string(), "wallet_service".to_string());
        routes.insert("/api/v1/swap".to_string(), "swap_service".to_string());
        routes.insert("/api/v1/staking".to_string(), "staking_service".to_string());
        routes.insert("/api/v1/nft".to_string(), "nft_service".to_string());
    }

    pub fn route(&self, path: &str) -> Result<String, Error> {
        let routes = self.routes.read().unwrap();
        
        routes.get(path)
            .cloned()
            .ok_or_else(|| Error::NotFound(format!("Route not found: {}", path)))
    }
}

impl Default for Router {
    fn default() -> Self {
        Self::new()
    }
}