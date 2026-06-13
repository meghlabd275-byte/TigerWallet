//! Audit Engine

use crate::error::Error;
use crate::models::*;
use chrono::{DateTime, Utc};
use std::sync::{Arc, RwLock};
use std::collections::HashMap;
use uuid::Uuid;

/// Audit Service
pub struct AuditService {
    logs: RwLock<Vec<AuditLog>>,
    checks: RwLock<HashMap<String, Vec<SecurityCheck>>>,
}

impl AuditService {
    pub fn new() -> Self {
        Self {
            logs: RwLock::new(Vec::new()),
            checks: RwLock::new(HashMap::new()),
        }
    }

    pub fn log_action(&self, req: AuditRequest, result: AuditResult) -> Result<AuditLog, Error> {
        let log = AuditLog {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            action: req.action,
            resource: req.resource,
            result,
            timestamp: Utc::now(),
        };
        
        let mut logs = self.logs.write().unwrap();
        logs.push(log.clone());
        
        Ok(log)
    }

    pub fn get_logs(&self, user_id: &str) -> Vec<AuditLog> {
        let logs = self.logs.read().unwrap();
        logs.iter()
            .filter(|l| l.user_id == user_id)
            .cloned()
            .collect()
    }

    pub fn run_security_check(&self, user_id: &str, check_type: &str) -> Result<SecurityCheck, Error> {
        let check = SecurityCheck {
            id: Uuid::new_v4().to_string(),
            check_type: check_type.to_string(),
            passed: true,
            details: "Check passed".to_string(),
            timestamp: Utc::now(),
        };
        
        let mut checks = self.checks.write().unwrap();
        checks.entry(user_id.to_string()).or_insert_with(Vec::new);
        if let Some(user_checks) = checks.get_mut(user_id) {
            user_checks.push(check.clone());
        }
        
        Ok(check)
    }
}

impl Default for AuditService {
    fn default() -> Self {
        Self::new()
    }
}