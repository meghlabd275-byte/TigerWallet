use crate::database::Database;
use crate::error::Error;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Admin {
    pub id: String,
    pub username: String,
    pub email: String,
    pub role: AdminRole,
    pub status: AdminStatus,
    pub permissions: Vec<String>,
    pub two_factor_enabled: bool,
    pub security_level: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AdminRole {
    SuperAdmin,
    Admin,
    Manager,
    Support,
    Analyst,
    Moderator,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AdminStatus {
    Active,
    Suspended,
    Inactive,
    Pending,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAdminRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    pub role: AdminRole,
    pub permissions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateAdminRequest {
    pub username: Option<String>,
    pub email: Option<String>,
    pub role: Option<AdminRole>,
    pub permissions: Option<Vec<String>>,
    pub status: Option<AdminStatus>,
}

pub struct AdminService {
    db: Database,
}

impl AdminService {
    pub fn new(db: Database) -> Self {
        Self { db }
    }

    pub async fn create_admin(&self, req: CreateAdminRequest) -> Result<Admin, Error> {
        let id = uuid::Uuid::new_v4().to_string();
        let now = Utc::now();
        
        let admin = Admin {
            id: id.clone(),
            username: req.username,
            email: req.email,
            role: req.role,
            status: AdminStatus::Active,
            permissions: req.permissions,
            two_factor_enabled: false,
            security_level: 1,
            created_at: now,
            updated_at: now,
        };
        
        Ok(admin)
    }

    pub async fn get_admin(&self, id: &str) -> Result<Option<Admin>, Error> {
        Ok(Some(Admin {
            id: id.to_string(),
            username: "admin".to_string(),
            email: "admin@tigerwallet.com".to_string(),
            role: AdminRole::SuperAdmin,
            status: AdminStatus::Active,
            permissions: vec!["*".to_string()],
            two_factor_enabled: false,
            security_level: 4,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        }))
    }

    pub async fn update_admin(&self, id: &str, req: UpdateAdminRequest) -> Result<Admin, Error> {
        let mut admin = self.get_admin(id).await?.ok_or(Error::NotFound)?;
        
        if let Some(username) = req.username {
            admin.username = username;
        }
        if let Some(email) = req.email {
            admin.email = email;
        }
        if let Some(role) = req.role {
            admin.role = role;
        }
        if let Some(permissions) = req.permissions {
            admin.permissions = permissions;
        }
        if let Some(status) = req.status {
            admin.status = status;
        }
        
        admin.updated_at = Utc::now();
        
        Ok(admin)
    }

    pub async fn delete_admin(&self, id: &str) -> Result<bool, Error> {
        Ok(true)
    }

    pub async fn list_admins(&self, page: i32, limit: i32) -> Result<Vec<Admin>, Error> {
        Ok(vec![])
    }

    pub async fn update_permissions(&self, id: &str, permissions: Vec<String>) -> Result<bool, Error> {
        Ok(true)
    }

    pub async fn has_permission(&self, admin_id: &str, permission: &str) -> Result<bool, Error> {
        let admin = self.get_admin(admin_id).await?.ok_or(Error::NotFound)?;
        
        if admin.role == AdminRole::SuperAdmin {
            return Ok(true);
        }
        
        Ok(admin.permissions.iter().any(|p| p == "*" || p == permission))
    }

    pub async fn suspend_admin(&self, id: &str) -> Result<bool, Error> {
        Ok(true)
    }

    pub async fn activate_admin(&self, id: &str) -> Result<bool, Error> {
        Ok(true)
    }
}
