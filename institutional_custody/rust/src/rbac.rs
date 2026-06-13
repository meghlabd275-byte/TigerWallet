//! RBAC module for Institutional Custody

use crate::{CustodyError, Role, RoleInfo};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use chrono::{DateTime, Utc};

/// Role assignment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoleAssignment {
    pub account: String,
    pub role: Role,
    pub granted_at: DateTime<Utc>,
    pub granted_by: String,
    pub expires_at: Option<DateTime<Utc>>,
}

/// Permission
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Permission {
    pub resource: String,
    pub action: String,
    pub conditions: Vec<String>,
}

/// RBAC service
pub struct RbacService {
    /// Role assignments
    assignments: HashMap<String, HashMap<String, RoleAssignment>>,
    /// Permissions
    permissions: HashMap<String, Vec<Permission>>,
}

impl RbacService {
    pub fn new() -> Self {
        Self {
            assignments: HashMap::new(),
            permissions: HashMap::new(),
        }
    }

    /// Initialize wallet
    pub async fn initialize_wallet(
        &self,
        wallet: &str,
        owners: &[String],
        signers: &[String],
        threshold: u32,
    ) -> Result<(), CustodyError> {
        let wallet_roles = self.assignments
            .entry(wallet.to_string())
            .or_insert_with(HashMap::new);
        
        // Grant owner roles
        for owner in owners {
            wallet_roles.insert(owner.clone(), RoleAssignment {
                account: owner.clone(),
                role: Role::Owner,
                granted_at: Utc::now(),
                granted_by: wallet.to_string(),
                expires_at: None,
            });
        }
        
        // Grant signer roles
        for signer in signers {
            wallet_roles.insert(signer.clone(), RoleAssignment {
                account: signer.clone(),
                role: Role::Signer,
                granted_at: Utc::now(),
                granted_by: wallet.to_string(),
                expires_at: None,
            });
        }
        
        Ok(())
    }

    /// Grant role
    pub async fn grant_role(
        &self,
        wallet: &str,
        account: &str,
        role: Role,
    ) -> Result<(), CustodyError> {
        let wallet_roles = self.assignments
            .entry(wallet.to_string())
            .or_insert_with(HashMap::new);
        
        if wallet_roles.contains_key(account) {
            return Err(CustodyError::RbacError("Role already granted".to_string()));
        }
        
        wallet_roles.insert(account.to_string(), RoleAssignment {
            account: account.to_string(),
            role,
            granted_at: Utc::now(),
            granted_by: wallet.to_string(),
            expires_at: None,
        });
        
        Ok(())
    }

    /// Revoke role
    pub async fn revoke_role(
        &self,
        wallet: &str,
        account: &str,
    ) -> Result<(), CustodyError> {
        let wallet_roles = self.assignments
            .get(wallet)
            .ok_or(CustodyError::RbacError("Wallet not found".to_string()))?;
        
        if !wallet_roles.contains_key(account) {
            return Err(CustodyError::RbacError("No role".to_string()));
        }
        
        Ok(())
    }

    /// Check has role
    pub async fn has_role(
        &self,
        wallet: &str,
        account: &str,
        role: Role,
    ) -> bool {
        if let Some(wallet_roles) = self.assignments.get(wallet) {
            if let Some(assignment) = wallet_roles.get(account) {
                return assignment.role == role;
            }
        }
        false
    }

    /// Get roles
    pub async fn get_roles(
        &self,
        wallet: &str,
    ) -> Result<Vec<RoleInfo>, CustodyError> {
        let wallet_roles = self.assignments
            .get(wallet)
            .ok_or(CustodyError::RbacError("Wallet not found".to_string()))?;
        
        let mut roles = vec![];
        
        for assignment in wallet_roles.values() {
            roles.push(RoleInfo {
                account: assignment.account.clone(),
                role: assignment.role,
                granted_at: assignment.granted_at,
            });
        }
        
        Ok(roles)
    }

    /// Check permission
    pub async fn check_permission(
        &self,
        wallet: &str,
        account: &str,
        resource: &str,
        action: &str,
    ) -> Result<bool, CustodyError> {
        let wallet_roles = self.assignments
            .get(wallet)
            .ok_or(CustodyError::RbacError("Wallet not found".to_string()))?;
        
        if let Some(assignment) = wallet_roles.get(account) {
            // Owner and admin have all permissions
            if assignment.role == Role::Owner || assignment.role == Role::Admin {
                return Ok(true);
            }
            
            // Check specific permissions
            if let Some(perms) = self.permissions.get(wallet) {
                for perm in perms {
                    if perm.resource == resource && perm.action == action {
                        return Ok(true);
                    }
                }
            }
        }
        
        Ok(false)
    }
}

impl Default for RbacService {
    fn default() -> Self {
        Self::new()
    }
}