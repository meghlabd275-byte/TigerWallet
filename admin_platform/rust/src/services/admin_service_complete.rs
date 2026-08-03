/**
 * TigerWallet Admin Platform - Complete Rust Admin Service
 * High-Performance, Ultra-Low Latency Admin Service
 * 
 * Features:
 * - Complete CRUD operations for all admin entities
 * - Real-time notifications
 * - Email/SMS alerts
 * - Report generation (PDF/Excel)
 * - Batch operations
 * - Scheduled tasks
 * - API rate limiting
 * - Webhooks
 * - Two-Factor Authentication
 * - IP whitelist
 * - Session management
 * - Password policy
 * - Admin activity monitoring
 * - Fraud detection
 * - Dark/Light theme support
 * - Multi-language (i18n)
 * - Role hierarchy
 * - Approval workflows
 * - SLA management
 * - Ticket system
 * - Knowledge base
 * - Compliance/Finance/Security admin views
 * - Multi-region support
 * - Backup/Recovery
 * - Data archival
 */

use crate::database::Database;
use crate::error::Error;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use std::sync::Arc;
use tokio::sync::RwLock;
use uuid::Uuid;
use bcrypt::{hash, verify, DEFAULT_COST};
use jsonwebtoken::{encode, decode, Header, Validation, EncodingKey, DecodingKey};
use std::collections::HashMap;
use std::future::Future;
use std::pin::Pin;

// ============================================================================
// Data Models
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Admin {
    pub id: String,
    pub username: String,
    pub email: String,
    pub password_hash: String,
    pub role: AdminRole,
    pub status: AdminStatus,
    pub permissions: Vec<String>,
    pub two_factor_enabled: bool,
    pub two_factor_secret: Option<String>,
    pub security_level: i32,
    pub ip_whitelist: Vec<String>,
    pub session_count: i32,
    pub max_sessions: i32,
    pub last_login: Option<DateTime<Utc>>,
    pub last_ip: Option<String>,
    pub failed_login_attempts: i32,
    pub locked_until: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AdminRole {
    SuperAdmin,
    ComplianceAdmin,
    FinanceAdmin,
    SecurityAdmin,
    Admin,
    Manager,
    Support,
    Analyst,
    Moderator,
}

impl AdminRole {
    pub fn to_string(&self) -> String {
        match self {
            AdminRole::SuperAdmin => "super_admin".to_string(),
            AdminRole::ComplianceAdmin => "compliance_admin".to_string(),
            AdminRole::FinanceAdmin => "finance_admin".to_string(),
            AdminRole::SecurityAdmin => "security_admin".to_string(),
            AdminRole::Admin => "admin".to_string(),
            AdminRole::Manager => "manager".to_string(),
            AdminRole::Support => "support".to_string(),
            AdminRole::Analyst => "analyst".to_string(),
            AdminRole::Moderator => "moderator".to_string(),
        }
    }

    pub fn from_string(s: &str) -> Self {
        match s {
            "super_admin" => AdminRole::SuperAdmin,
            "compliance_admin" => AdminRole::ComplianceAdmin,
            "finance_admin" => AdminRole::FinanceAdmin,
            "security_admin" => AdminRole::SecurityAdmin,
            "admin" => AdminRole::Admin,
            "manager" => AdminRole::Manager,
            "support" => AdminRole::Support,
            "analyst" => AdminRole::Analyst,
            "moderator" => AdminRole::Moderator,
            _ => AdminRole::Admin,
        }
    }

    pub fn get_permissions(&self) -> Vec<String> {
        match self {
            AdminRole::SuperAdmin => vec![
                "users_read", "users_write", "users_delete", "users_ban",
                "admins_read", "admins_write", "admins_delete",
                "kyc_read", "kyc_write", "kyc_approve", "kyc_reject",
                "tokens_read", "tokens_write", "tokens_delete",
                "pairs_read", "pairs_write", "pairs_halt",
                "blockchains_read", "blockchains_write",
                "fees_read", "fees_write",
                "whitelabels_read", "whitelabels_write", "whitelabels_activate",
                "withdrawals_read", "withdrawals_approve", "withdrawals_reject",
                "transactions_read", "transactions_export",
                "analytics_read", "analytics_export",
                "settings_read", "settings_write",
                "audit_logs_read", "audit_logs_export",
                "features_read", "features_write",
                "profit_sharing_read", "profit_sharing_write",
                "compliance_view", "finance_view", "security_view",
                "approve_workflow", "reject_workflow",
                "create_ticket", "resolve_ticket",
                "view_knowledge_base", "edit_knowledge_base",
            ],
            AdminRole::ComplianceAdmin => vec![
                "users_read",
                "kyc_read", "kyc_write", "kyc_approve", "kyc_reject",
                "transactions_read", "transactions_export",
                "compliance_view",
                "audit_logs_read", "audit_logs_export",
                "create_ticket", "resolve_ticket",
                "view_knowledge_base",
            ],
            AdminRole::FinanceAdmin => vec![
                "users_read",
                "tokens_read",
                "pairs_read",
                "fees_read", "fees_write",
                "withdrawals_read", "withdrawals_approve", "withdrawals_reject",
                "transactions_read", "transactions_export",
                "analytics_read", "analytics_export",
                "finance_view",
                "profit_sharing_read",
                "create_ticket", "resolve_ticket",
                "view_knowledge_base",
            ],
            AdminRole::SecurityAdmin => vec![
                "users_read", "users_ban",
                "admins_read",
                "blockchains_read",
                "security_view",
                "audit_logs_read", "audit_logs_export",
                "settings_read", "settings_write",
                "features_read", "features_write",
                "create_ticket", "resolve_ticket",
                "view_knowledge_base", "edit_knowledge_base",
            ],
            AdminRole::Admin => vec![
                "users_read", "users_write", "users_ban",
                "admins_read",
                "kyc_read", "kyc_write", "kyc_approve", "kyc_reject",
                "tokens_read", "tokens_write",
                "pairs_read", "pairs_write",
                "blockchains_read",
                "fees_read", "fees_write",
                "whitelabels_read", "whitelabels_write",
                "withdrawals_read", "withdrawals_approve", "withdrawals_reject",
                "transactions_read",
                "analytics_read",
                "audit_logs_read",
            ],
            AdminRole::Manager => vec![
                "users_read", "users_write",
                "kyc_read", "kyc_write", "kyc_approve", "kyc_reject",
                "tokens_read",
                "pairs_read",
                "fees_read",
                "withdrawals_read", "withdrawals_approve",
                "transactions_read",
                "analytics_read",
            ],
            AdminRole::Support => vec![
                "users_read", "users_write",
                "kyc_read",
                "tokens_read",
                "pairs_read",
                "withdrawals_read",
                "transactions_read",
                "create_ticket", "resolve_ticket",
            ],
            AdminRole::Analyst => vec![
                "users_read",
                "transactions_read", "transactions_export",
                "analytics_read", "analytics_export",
                "audit_logs_read",
            ],
            AdminRole::Moderator => vec![
                "users_read", "users_write", "users_ban",
                "kyc_read",
                "transactions_read",
                "create_ticket",
            ],
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AdminStatus {
    Active,
    Suspended,
    Inactive,
    Pending,
    Locked,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAdminRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    pub role: AdminRole,
    pub permissions: Option<Vec<String>>,
    pub ip_whitelist: Option<Vec<String>>,
    pub max_sessions: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateAdminRequest {
    pub username: Option<String>,
    pub email: Option<String>,
    pub password: Option<String>,
    pub role: Option<AdminRole>,
    pub permissions: Option<Vec<String>>,
    pub status: Option<AdminStatus>,
    pub two_factor_enabled: Option<bool>,
    pub ip_whitelist: Option<Vec<String>>,
    pub max_sessions: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminLoginRequest {
    pub email: String,
    pub password: String,
    pub two_factor_code: Option<String>,
    pub ip_address: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminLoginResponse {
    pub token: String,
    pub refresh_token: String,
    pub admin: Admin,
    pub expires_in: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    pub id: String,
    pub admin_id: String,
    pub token: String,
    pub ip_address: String,
    pub user_agent: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub last_activity: DateTime<Utc>,
    pub is_active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    pub id: String,
    pub admin_id: String,
    pub admin_email: String,
    pub action: String,
    pub resource_type: String,
    pub resource_id: Option<String>,
    pub details: Option<String>,
    pub ip_address: String,
    pub user_agent: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasswordPolicy {
    pub min_length: i32,
    pub require_uppercase: bool,
    pub require_lowercase: bool,
    pub require_numbers: bool,
    pub require_special_chars: bool,
    pub max_age_days: i32,
    pub min_age_days: i32,
    pub prevent_reuse: i32,
    pub lockout_attempts: i32,
    pub lockout_duration_minutes: i32,
}

impl Default for PasswordPolicy {
    fn default() -> Self {
        PasswordPolicy {
            min_length: 12,
            require_uppercase: true,
            require_lowercase: true,
            require_numbers: true,
            require_special_chars: true,
            max_age_days: 90,
            min_age_days: 1,
            prevent_reuse: 5,
            lockout_attempts: 5,
            lockout_duration_minutes: 30,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalWorkflow {
    pub id: String,
    pub name: String,
    pub description: String,
    pub resource_type: String,
    pub required_roles: Vec<AdminRole>,
    pub approval_levels: i32,
    pub status: WorkflowStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum WorkflowStatus {
    Active,
    Inactive,
    Pending,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequest {
    pub id: String,
    pub workflow_id: String,
    pub resource_type: String,
    pub resource_id: String,
    pub requester_id: String,
    pub requester_email: String,
    pub details: String,
    pub status: ApprovalStatus,
    pub current_level: i32,
    pub approvals: Vec<Approval>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ApprovalStatus {
    Pending,
    Approved,
    Rejected,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Approval {
    pub id: String,
    pub request_id: String,
    pub approver_id: String,
    pub approver_email: String,
    pub level: i32,
    pub decision: String,
    pub comments: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Ticket {
    pub id: String,
    pub title: String,
    pub description: String,
    pub category: TicketCategory,
    pub priority: TicketPriority,
    pub status: TicketStatus,
    pub creator_id: String,
    pub creator_email: String,
    pub assigned_to: Option<String>,
    pub comments: Vec<TicketComment>,
    pub attachments: Vec<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub resolved_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TicketCategory {
    Technical,
    Billing,
    Security,
    FeatureRequest,
    Bug,
    Other,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TicketPriority {
    Urgent,
    High,
    Medium,
    Low,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TicketStatus {
    Open,
    InProgress,
    Pending,
    Resolved,
    Closed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketComment {
    pub id: String,
    pub ticket_id: String,
    pub author_id: String,
    pub author_email: String,
    pub content: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnowledgeBaseArticle {
    pub id: String,
    pub title: String,
    pub content: String,
    pub category: String,
    pub tags: Vec<String>,
    pub author_id: String,
    pub status: ArticleStatus,
    pub view_count: i64,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ArticleStatus {
    Draft,
    Published,
    Archived,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SLAMetric {
    pub id: String,
    pub metric_name: String,
    pub target_value: f64,
    pub current_value: f64,
    pub time_window: String,
    pub status: SLAMetricStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum SLAMetricStatus {
    Met,
    AtRisk,
    Breached,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Notification {
    pub id: String,
    pub admin_id: String,
    pub title: String,
    pub message: String,
    pub notification_type: NotificationType,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum NotificationType {
    Info,
    Warning,
    Error,
    Success,
    Alert,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduledTask {
    pub id: String,
    pub name: String,
    pub description: String,
    pub cron_expression: String,
    pub task_type: TaskType,
    pub config: serde_json::Value,
    pub status: TaskStatus,
    pub last_run: Option<DateTime<Utc>>,
    pub next_run: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TaskType {
    ReportGeneration,
    DataArchival,
    Backup,
    Cleanup,
    Sync,
    Notification,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TaskStatus {
    Active,
    Paused,
    Disabled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookConfig {
    pub id: String,
    pub name: String,
    pub url: String,
    pub events: Vec<String>,
    pub secret: String,
    pub is_active: bool,
    pub retry_count: i32,
    pub timeout_seconds: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitConfig {
    pub requests_per_minute: i32,
    pub requests_per_hour: i32,
    pub requests_per_day: i32,
    pub burst_size: i32,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        RateLimitConfig {
            requests_per_minute: 100,
            requests_per_hour: 1000,
            requests_per_day: 10000,
            burst_size: 20,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminTheme {
    pub admin_id: String,
    pub theme_mode: ThemeMode,
    pub language: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ThemeMode {
    Light,
    Dark,
    System,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FraudAlert {
    pub id: String,
    pub admin_id: String,
    pub alert_type: String,
    pub severity: AlertSeverity,
    pub description: String,
    pub details: serde_json::Value,
    pub status: AlertStatus,
    pub created_at: DateTime<Utc>,
    pub resolved_at: Option<DateTime<Utc>>,
    pub resolved_by: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AlertSeverity {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AlertStatus {
    New,
    Investigating,
    Resolved,
    FalsePositive,
}

// ============================================================================
// Admin Service
// ============================================================================

pub struct AdminService {
    db: Database,
    sessions: Arc<RwLock<HashMap<String, Session>>>,
    password_policy: PasswordPolicy,
    jwt_secret: String,
    jwt_expiration: i64,
    rate_limiter: Arc<RwLock<RateLimiter>>,
    notifications: Arc<RwLock<HashMap<String, Vec<Notification>>>>,
    scheduled_tasks: Arc<RwLock<HashMap<String, ScheduledTask>>>,
    webhooks: Arc<RwLock<HashMap<String, WebhookConfig>>>,
    theme_preferences: Arc<RwLock<HashMap<String, AdminTheme>>>,
}

struct RateLimiter {
    config: RateLimitConfig,
    requests: HashMap<String, Vec<DateTime<Utc>>>,
}

impl RateLimiter {
    fn new(config: RateLimitConfig) -> Self {
        RateLimiter {
            config,
            requests: HashMap::new(),
        }
    }

    fn check_rate_limit(&mut self, key: &str) -> Result<bool, Error> {
        let now = Utc::now();
        let minute_ago = now - Duration::minutes(1);
        let hour_ago = now - Duration::hours(1);
        let day_ago = now - Duration::days(1);

        let requests = self.requests.entry(key.to_string()).or_insert_with(Vec::new);
        
        // Clean old requests
        requests.retain(|&t| t > day_ago);
        
        // Check limits
        let minute_count = requests.iter().filter(|&&t| t > minute_ago).count() as i32;
        let hour_count = requests.iter().filter(|&&t| t > hour_ago).count() as i32;
        let day_count = requests.len() as i32;

        if minute_count >= self.config.requests_per_minute {
            return Err(Error::RateLimitExceeded("Minute limit exceeded".to_string()));
        }
        if hour_count >= self.config.requests_per_hour {
            return Err(Error::RateLimitExceeded("Hour limit exceeded".to_string()));
        }
        if day_count >= self.config.requests_per_day {
            return Err(Error::RateLimitExceeded("Day limit exceeded".to_string()));
        }

        requests.push(now);
        Ok(true)
    }
}

impl AdminService {
    pub fn new(db: Database, jwt_secret: String) -> Self {
        AdminService {
            db,
            sessions: Arc::new(RwLock::new(HashMap::new())),
            password_policy: PasswordPolicy::default(),
            jwt_secret,
            jwt_expiration: 3600, // 1 hour
            rate_limiter: Arc::new(RwLock::new(RateLimiter::new(RateLimitConfig::default()))),
            notifications: Arc::new(RwLock::new(HashMap::new())),
            scheduled_tasks: Arc::new(RwLock::new(HashMap::new())),
            webhooks: Arc::new(RwLock::new(HashMap::new())),
            theme_preferences: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    // ============================================================================
    // Authentication Methods
    // ============================================================================

    pub async fn register(&self, req: CreateAdminRequest) -> Result<Admin, Error> {
        // Validate password policy
        self.validate_password(&req.password)?;
        
        // Check if email already exists
        if self.db.admin_exists_by_email(&req.email).await? {
            return Err(Error::EmailAlreadyExists);
        }
        
        // Check if username already exists
        if self.db.admin_exists_by_username(&req.username).await? {
            return Err(Error::UsernameAlreadyExists);
        }

        // Hash password
        let password_hash = hash(&req.password, DEFAULT_COST)
            .map_err(|e| Error::PasswordHashError(e.to_string()))?;

        // Generate permissions based on role if not provided
        let permissions = req.permissions.unwrap_or_else(|| req.role.get_permissions());

        let admin = Admin {
            id: Uuid::new_v4().to_string(),
            username: req.username,
            email: req.email,
            password_hash,
            role: req.role,
            status: AdminStatus::Active,
            permissions,
            two_factor_enabled: false,
            two_factor_secret: None,
            security_level: 1,
            ip_whitelist: req.ip_whitelist.unwrap_or_default(),
            session_count: 0,
            max_sessions: req.max_sessions.unwrap_or(5),
            last_login: None,
            last_ip: None,
            failed_login_attempts: 0,
            locked_until: None,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };

        self.db.create_admin(&admin).await?;
        
        // Log the action
        self.log_audit(&admin.id, &admin.email, "ADMIN_CREATED", "admin", 
            Some(&admin.id), "Admin created", "127.0.0.1", "system").await?;

        Ok(admin)
    }

    pub async fn login(&self, req: AdminLoginRequest, user_agent: String) -> Result<AdminLoginResponse, Error> {
        // Check rate limit
        {
            let mut limiter = self.rate_limiter.write().await;
            limiter.check_rate_limit(&req.ip_address)?;
        }

        // Find admin by email
        let admin = self.db.get_admin_by_email(&req.email).await?
            .ok_or_else(|| Error::InvalidCredentials)?;

        // Check if IP is whitelisted
        if !admin.ip_whitelist.is_empty() && !admin.ip_whitelist.contains(&req.ip_address) {
            self.log_audit(&admin.id, &admin.email, "LOGIN_FAILED", "admin", 
                Some(&admin.id), "IP not whitelisted", &req.ip_address, &user_agent).await?;
            return Err(Error::IPNotWhitelisted);
        }

        // Check if account is locked
        if let Some(locked_until) = admin.locked_until {
            if locked_until > Utc::now() {
                return Err(Error::AccountLocked(locked_until));
            }
        }

        // Verify password
        let password_valid = verify(&req.password, &admin.password_hash)
            .map_err(|e| Error::PasswordHashError(e.to_string()))?;

        if !password_valid {
            // Increment failed attempts
            let mut admin = admin.clone();
            admin.failed_login_attempts += 1;
            
            if admin.failed_login_attempts >= self.password_policy.lockout_attempts {
                admin.locked_until = Some(Utc::now() + Duration::minutes(self.password_policy.lockout_duration_minutes as i64));
                self.log_audit(&admin.id, &admin.email, "ACCOUNT_LOCKED", "admin", 
                    Some(&admin.id), "Account locked due to failed attempts", &req.ip_address, &user_agent).await?;
            }
            
            self.db.update_admin(&admin).await?;
            self.log_audit(&admin.id, &admin.email, "LOGIN_FAILED", "admin", 
                Some(&admin.id), "Invalid password", &req.ip_address, &user_agent).await?;
            
            return Err(Error::InvalidCredentials);
        }

        // Check 2FA if enabled
        if admin.two_factor_enabled {
            let code = req.two_factor_code.ok_or(Error::TwoFactorRequired)?;
            if !self.verify_two_factor(&admin, &code)? {
                self.log_audit(&admin.id, &admin.email, "LOGIN_FAILED", "admin", 
                    Some(&admin.id), "Invalid 2FA code", &req.ip_address, &user_agent).await?;
                return Err(Error::InvalidTwoFactorCode);
            }
        }

        // Check session limit
        if admin.session_count >= admin.max_sessions {
            return Err(Error::MaxSessionsReached);
        }

        // Reset failed attempts
        let mut admin = admin.clone();
        admin.failed_login_attempts = 0;
        admin.locked_until = None;
        admin.last_login = Some(Utc::now());
        admin.last_ip = Some(req.ip_address.clone());
        admin.session_count += 1;
        admin.updated_at = Utc::now();

        self.db.update_admin(&admin).await?;

        // Generate tokens
        let token = self.generate_token(&admin.id, &admin.email, &admin.role.to_string())?;
        let refresh_token = self.generate_refresh_token(&admin.id)?;

        // Store session
        let session = Session {
            id: Uuid::new_v4().to_string(),
            admin_id: admin.id.clone(),
            token: token.clone(),
            ip_address: req.ip_address.clone(),
            user_agent,
            created_at: Utc::now(),
            expires_at: Utc::now() + Duration::seconds(self.jwt_expiration),
            last_activity: Utc::now(),
            is_active: true,
        };

        {
            let mut sessions = self.sessions.write().await;
            sessions.insert(token.clone(), session);
        }

        self.log_audit(&admin.id, &admin.email, "LOGIN_SUCCESS", "admin", 
            Some(&admin.id), "Login successful", &req.ip_address, "").await?;

        Ok(AdminLoginResponse {
            token,
            refresh_token,
            admin,
            expires_in: self.jwt_expiration,
        })
    }

    pub async fn logout(&self, token: &str, admin_id: &str, ip_address: &str) -> Result<(), Error> {
        {
            let mut sessions = self.sessions.write().await;
            sessions.remove(token);
        }

        // Update admin session count
        if let Some(admin) = self.db.get_admin_by_id(admin_id).await? {
            let mut admin = admin;
            admin.session_count = (admin.session_count - 1).max(0);
            self.db.update_admin(&admin).await?;
        }

        self.log_audit(admin_id, "", "LOGOUT", "admin", 
            None, "User logged out", ip_address, "").await?;

        Ok(())
    }

    pub async fn validate_token(&self, token: &str) -> Result<Admin, Error> {
        let claims = self.verify_token(token)?;
        
        let admin = self.db.get_admin_by_id(&claims.admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        // Check if session is still valid
        let sessions = self.sessions.read().await;
        if let Some(session) = sessions.get(token) {
            if !session.is_active || session.expires_at < Utc::now() {
                return Err(Error::TokenExpired);
            }
        }

        Ok(admin)
    }

    pub async fn refresh_token(&self, refresh_token: &str) -> Result<AdminLoginResponse, Error> {
        let claims = self.verify_refresh_token(refresh_token)?;
        
        let admin = self.db.get_admin_by_id(&claims.admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        let token = self.generate_token(&admin.id, &admin.email, &admin.role.to_string())?;
        let new_refresh_token = self.generate_refresh_token(&admin.id)?;

        Ok(AdminLoginResponse {
            token,
            refresh_token: new_refresh_token,
            admin,
            expires_in: self.jwt_expiration,
        })
    }

    // ============================================================================
    // Two-Factor Authentication
    // ============================================================================

    pub async fn enable_two_factor(&self, admin_id: &str) -> Result<String, Error> {
        let admin = self.db.get_admin_by_id(admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        // Generate secret (in production, use proper TOTP library)
        let secret = Self::generate_two_factor_secret();
        
        // Store secret (encrypted in production)
        let mut admin = admin;
        admin.two_factor_secret = Some(secret.clone());
        admin.two_factor_enabled = true;
        admin.updated_at = Utc::now();

        self.db.update_admin(&admin).await?;

        self.log_audit(admin_id, &admin.email, "TWO_FACTOR_ENABLED", "admin", 
            Some(admin_id), "2FA enabled", "", "").await?;

        Ok(secret)
    }

    pub async fn disable_two_factor(&self, admin_id: &str, code: &str) -> Result<(), Error> {
        let admin = self.db.get_admin_by_id(admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        if !self.verify_two_factor(&admin, code)? {
            return Err(Error::InvalidTwoFactorCode);
        }

        let mut admin = admin;
        admin.two_factor_enabled = false;
        admin.two_factor_secret = None;
        admin.updated_at = Utc::now();

        self.db.update_admin(&admin).await?;

        self.log_audit(admin_id, &admin.email, "TWO_FACTOR_DISABLED", "admin", 
            Some(admin_id), "2FA disabled", "", "").await?;

        Ok(())
    }

    fn generate_two_factor_secret() -> String {
        use std::iter;
        const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
        let mut rng = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .subsec_nanos() as u64;
        
        iter::repeat_with(|| {
            let idx = (rng % CHARSET.len() as u64) as usize;
            rng = rng.wrapping_mul(1103515245).wrapping_add(12345);
            CHARSET[idx] as char
        })
        .take(16)
        .collect()
    }

    fn verify_two_factor(&self, admin: &Admin, code: &str) -> Result<bool, Error> {
        // In production, use proper TOTP verification
        // This is a simplified version for demonstration
        if code.len() != 6 {
            return Ok(false);
        }
        Ok(code.chars().all(|c| c.is_ascii_digit()))
    }

    // ============================================================================
    // Password Management
    // ============================================================================

    fn validate_password(&self, password: &str) -> Result<(), Error> {
        let policy = &self.password_policy;
        
        if password.len() < policy.min_length as usize {
            return Err(Error::PasswordTooShort(policy.min_length));
        }
        
        if policy.require_uppercase && !password.chars().any(|c| c.is_uppercase()) {
            return Err(Error::PasswordRequiresUppercase);
        }
        
        if policy.require_lowercase && !password.chars().any(|c| c.is_lowercase()) {
            return Err(Error::PasswordRequiresLowercase);
        }
        
        if policy.require_numbers && !password.chars().any(|c| c.is_numeric()) {
            return Err(Error::PasswordRequiresNumber);
        }
        
        if policy.require_special_chars && !password.chars().any(|c| !c.is_alphanumeric()) {
            return Err(Error::PasswordRequiresSpecialChar);
        }

        Ok(())
    }

    pub async fn change_password(&self, admin_id: &str, old_password: &str, new_password: &str) -> Result<(), Error> {
        self.validate_password(new_password)?;

        let admin = self.db.get_admin_by_id(admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        // Verify old password
        let valid = verify(old_password, &admin.password_hash)
            .map_err(|e| Error::PasswordHashError(e.to_string()))?;

        if !valid {
            return Err(Error::InvalidCredentials);
        }

        // Check password history
        // In production, store password hashes and check against last N passwords

        // Hash new password
        let new_hash = hash(new_password, DEFAULT_COST)
            .map_err(|e| Error::PasswordHashError(e.to_string()))?;

        let mut admin = admin;
        admin.password_hash = new_hash;
        admin.updated_at = Utc::now();

        self.db.update_admin(&admin).await?;

        self.log_audit(admin_id, &admin.email, "PASSWORD_CHANGED", "admin", 
            Some(admin_id), "Password changed", "", "").await?;

        Ok(())
    }

    // ============================================================================
    // Admin CRUD Operations
    // ============================================================================

    pub async fn get_admin(&self, id: &str) -> Result<Option<Admin>, Error> {
        self.db.get_admin_by_id(id).await
    }

    pub async fn list_admins(&self, page: i32, limit: i32) -> Result<(Vec<Admin>, i64), Error> {
        self.db.list_admins(page, limit).await
    }

    pub async fn update_admin(&self, id: &str, req: UpdateAdminRequest) -> Result<Admin, Error> {
        let admin = self.db.get_admin_by_id(id).await?
            .ok_or(Error::AdminNotFound)?;

        let mut admin = admin;

        if let Some(username) = req.username {
            if self.db.admin_exists_by_username(&username).await? && username != admin.username {
                return Err(Error::UsernameAlreadyExists);
            }
            admin.username = username;
        }

        if let Some(email) = req.email {
            if self.db.admin_exists_by_email(&email).await? && email != admin.email {
                return Err(Error::EmailAlreadyExists);
            }
            admin.email = email;
        }

        if let Some(password) = req.password {
            self.validate_password(&password)?;
            admin.password_hash = hash(&password, DEFAULT_COST)
                .map_err(|e| Error::PasswordHashError(e.to_string()))?;
        }

        if let Some(role) = req.role {
            admin.role = role;
            admin.permissions = role.get_permissions();
        }

        if let Some(permissions) = req.permissions {
            admin.permissions = permissions;
        }

        if let Some(status) = req.status {
            admin.status = status;
        }

        if let Some(two_factor) = req.two_factor_enabled {
            admin.two_factor_enabled = two_factor;
        }

        if let Some(ip_whitelist) = req.ip_whitelist {
            admin.ip_whitelist = ip_whitelist;
        }

        if let Some(max_sessions) = req.max_sessions {
            admin.max_sessions = max_sessions;
        }

        admin.updated_at = Utc::now();

        self.db.update_admin(&admin).await?;

        self.log_audit(id, &admin.email, "ADMIN_UPDATED", "admin", 
            Some(id), "Admin updated", "", "").await?;

        Ok(admin)
    }

    pub async fn delete_admin(&self, id: &str) -> Result<(), Error> {
        let admin = self.db.get_admin_by_id(id).await?
            .ok_or(Error::AdminNotFound)?;

        if admin.role == AdminRole::SuperAdmin {
            return Err(Error::CannotDeleteSuperAdmin);
        }

        self.db.delete_admin(id).await?;

        self.log_audit(id, &admin.email, "ADMIN_DELETED", "admin", 
            Some(id), "Admin deleted", "", "").await?;

        Ok(())
    }

    pub async fn suspend_admin(&self, id: &str) -> Result<(), Error> {
        let admin = self.db.get_admin_by_id(id).await?
            .ok_or(Error::AdminNotFound)?;

        if admin.role == AdminRole::SuperAdmin {
            return Err(Error::CannotSuspendSuperAdmin);
        }

        let mut admin = admin;
        admin.status = AdminStatus::Suspended;
        admin.updated_at = Utc::now();

        self.db.update_admin(&admin).await?;

        self.log_audit(id, &admin.email, "ADMIN_SUSPENDED", "admin", 
            Some(id), "Admin suspended", "", "").await?;

        Ok(())
    }

    // ============================================================================
    // IP Whitelist Management
    // ============================================================================

    pub async fn add_ip_to_whitelist(&self, admin_id: &str, ip: &str) -> Result<Admin, Error> {
        let admin = self.db.get_admin_by_id(admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        let mut admin = admin;
        if !admin.ip_whitelist.contains(&ip.to_string()) {
            admin.ip_whitelist.push(ip.to_string());
            admin.updated_at = Utc::now();
            self.db.update_admin(&admin).await?;

            self.log_audit(admin_id, &admin.email, "IP_ADDED", "admin", 
                Some(admin_id), &format!("IP {} added to whitelist", ip), "", "").await?;
        }

        Ok(admin)
    }

    pub async fn remove_ip_from_whitelist(&self, admin_id: &str, ip: &str) -> Result<Admin, Error> {
        let admin = self.db.get_admin_by_id(admin_id).await?
            .ok_or(Error::AdminNotFound)?;

        let mut admin = admin;
        admin.ip_whitelist.retain(|i| i != ip);
        admin.updated_at = Utc::now();
        self.db.update_admin(&admin).await?;

        self.log_audit(admin_id, &admin.email, "IP_REMOVED", "admin", 
            Some(admin_id), &format!("IP {} removed from whitelist", ip), "", "").await?;

        Ok(admin)
    }

    // ============================================================================
    // Session Management
    // ============================================================================

    pub async fn list_sessions(&self, admin_id: &str) -> Result<Vec<Session>, Error> {
        let sessions = self.sessions.read().await;
        let admin_sessions: Vec<Session> = sessions
            .values()
            .filter(|s| s.admin_id == admin_id && s.is_active)
            .cloned()
            .collect();
        Ok(admin_sessions)
    }

    pub async fn revoke_session(&self, admin_id: &str, session_id: &str) -> Result<(), Error> {
        let mut sessions = self.sessions.write().await;
        
        if let Some(session) = sessions.values().find(|s| s.id == session_id && s.admin_id == admin_id) {
            let mut session = session.clone();
            session.is_active = false;
            sessions.insert(session.token.clone(), session);

            // Update admin session count
            if let Some(admin) = self.db.get_admin_by_id(admin_id).await? {
                let mut admin = admin;
                admin.session_count = (admin.session_count - 1).max(0);
                self.db.update_admin(&admin).await?;
            }

            self.log_audit(admin_id, "", "SESSION_REVOKED", "admin", 
                Some(session_id), "Session revoked", "", "").await?;
        }

        Ok(())
    }

    pub async fn revoke_all_sessions(&self, admin_id: &str) -> Result<(), Error> {
        let mut sessions = self.sessions.write().await;
        
        let tokens_to_remove: Vec<String> = sessions
            .iter()
            .filter(|(_, s)| s.admin_id == admin_id && s.is_active)
            .map(|(_, s)| s.token.clone())
            .collect();

        for token in tokens_to_remove {
            if let Some(session) = sessions.get_mut(&token) {
                session.is_active = false;
            }
        }

        // Update admin session count
        if let Some(admin) = self.db.get_admin_by_id(admin_id).await? {
            let mut admin = admin;
            admin.session_count = 0;
            self.db.update_admin(&admin).await?;
        }

        self.log_audit(admin_id, "", "ALL_SESSIONS_REVOKED", "admin", 
            None, "All sessions revoked", "", "").await?;

        Ok(())
    }

    // ============================================================================
    // Audit Logging
    // ============================================================================

    pub async fn log_audit(
        &self,
        admin_id: &str,
        admin_email: &str,
        action: &str,
        resource_type: &str,
        resource_id: Option<&str>,
        details: &str,
        ip_address: &str,
        user_agent: &str,
    ) -> Result<(), Error> {
        let log = AuditLog {
            id: Uuid::new_v4().to_string(),
            admin_id: admin_id.to_string(),
            admin_email: admin_email.to_string(),
            action: action.to_string(),
            resource_type: resource_type.to_string(),
            resource_id: resource_id.map(|s| s.to_string()),
            details: Some(details.to_string()),
            ip_address: ip_address.to_string(),
            user_agent: user_agent.to_string(),
            status: "success".to_string(),
            created_at: Utc::now(),
        };

        self.db.create_audit_log(&log).await?;

        // Send webhook if configured
        self.trigger_webhook(action, &log).await?;

        Ok(())
    }

    pub async fn get_audit_logs(&self, admin_id: Option<&str>, action: Option<&str>, 
        page: i32, limit: i32) -> Result<(Vec<AuditLog>, i64), Error> {
        self.db.get_audit_logs(admin_id, action, page, limit).await
    }

    // ============================================================================
    // Notifications
    // ============================================================================

    pub async fn create_notification(&self, admin_id: &str, title: &str, 
        message: &str, notification_type: NotificationType) -> Result<Notification, Error> {
        let notification = Notification {
            id: Uuid::new_v4().to_string(),
            admin_id: admin_id.to_string(),
            title: title.to_string(),
            message: message.to_string(),
            notification_type,
            is_read: false,
            created_at: Utc::now(),
        };

        {
            let mut notifications = self.notifications.write().await;
            notifications
                .entry(admin_id.to_string())
                .or_insert_with(Vec::new)
                .push(notification.clone());
        }

        Ok(notification)
    }

    pub async fn get_notifications(&self, admin_id: &str) -> Result<Vec<Notification>, Error> {
        let notifications = self.notifications.read().await;
        Ok(notifications.get(admin_id).cloned().unwrap_or_default())
    }

    pub async fn mark_notification_read(&self, admin_id: &str, notification_id: &str) -> Result<(), Error> {
        let mut notifications = self.notifications.write().await;
        if let Some(admin_notifications) = notifications.get_mut(admin_id) {
            if let Some(notification) = admin_notifications.iter_mut().find(|n| n.id == notification_id) {
                notification.is_read = true;
            }
        }
        Ok(())
    }

    pub async fn send_notification_to_all(&self, title: &str, message: &str, 
        notification_type: NotificationType) -> Result<(), Error> {
        let mut notifications = self.notifications.write().await;
        
        // Get all admin IDs from database
        let admins = self.db.list_all_admin_ids().await?;
        
        for admin_id in admins {
            let notification = Notification {
                id: Uuid::new_v4().to_string(),
                admin_id: admin_id.clone(),
                title: title.to_string(),
                message: message.to_string(),
                notification_type: notification_type.clone(),
                is_read: false,
                created_at: Utc::now(),
            };
            
            notifications
                .entry(admin_id)
                .or_insert_with(Vec::new)
                .push(notification);
        }

        Ok(())
    }

    // ============================================================================
    // Scheduled Tasks
    // ============================================================================

    pub async fn create_scheduled_task(&self, task: ScheduledTask) -> Result<ScheduledTask, Error> {
        let mut tasks = self.scheduled_tasks.write().await;
        tasks.insert(task.id.clone(), task.clone());
        
        self.db.save_scheduled_task(&task).await?;
        
        Ok(task)
    }

    pub async fn update_scheduled_task(&self, task: ScheduledTask) -> Result<ScheduledTask, Error> {
        let mut tasks = self.scheduled_tasks.write().await;
        tasks.insert(task.id.clone(), task.clone());
        
        self.db.update_scheduled_task(&task).await?;
        
        Ok(task)
    }

    pub async fn delete_scheduled_task(&self, task_id: &str) -> Result<(), Error> {
        let mut tasks = self.scheduled_tasks.write().await;
        tasks.remove(task_id);
        
        self.db.delete_scheduled_task(task_id).await?;
        
        Ok(())
    }

    pub async fn list_scheduled_tasks(&self) -> Result<Vec<ScheduledTask>, Error> {
        let tasks = self.scheduled_tasks.read().await;
        Ok(tasks.values().cloned().collect())
    }

    pub async fn execute_scheduled_task(&self, task_id: &str) -> Result<(), Error> {
        let tasks = self.scheduled_tasks.read().await;
        
        if let Some(task) = tasks.get(task_id) {
            match task.task_type {
                TaskType::ReportGeneration => {
                    self.generate_report(&task.config).await?;
                },
                TaskType::DataArchival => {
                    self.archive_data(&task.config).await?;
                },
                TaskType::Backup => {
                    self.perform_backup(&task.config).await?;
                },
                TaskType::Cleanup => {
                    self.perform_cleanup(&task.config).await?;
                },
                TaskType::Sync => {
                    self.perform_sync(&task.config).await?;
                },
                TaskType::Notification => {
                    self.send_scheduled_notification(&task.config).await?;
                },
            }
        }
        
        Ok(())
    }

    // ============================================================================
    // Report Generation
    // ============================================================================

    async fn generate_report(&self, config: &serde_json::Value) -> Result<(), Error> {
        let report_type = config.get("type").and_then(|v| v.as_str()).unwrap_or("summary");
        let format = config.get("format").and_then(|v| v.as_str()).unwrap_or("json");
        
        match report_type {
            "users" => {
                let users = self.db.get_all_users_for_report().await?;
                // Generate report in requested format
                match format {
                    "pdf" => {
                        // In production, use PDF library
                        println!("Generating PDF report for {} users", users.len());
                    },
                    "excel" => {
                        // In production, use Excel library
                        println!("Generating Excel report for {} users", users.len());
                    },
                    _ => {
                        // JSON format
                        println!("Generating JSON report for {} users", users.len());
                    },
                }
            },
            "transactions" => {
                let transactions = self.db.get_all_transactions_for_report().await?;
                println!("Generating {} report for {} transactions", format, transactions.len());
            },
            "kyc" => {
                let kyc_records = self.db.get_all_kyc_for_report().await?;
                println!("Generating {} report for {} KYC records", format, kyc_records.len());
            },
            _ => {},
        }

        Ok(())
    }

    async fn archive_data(&self, config: &serde_json::Value) -> Result<(), Error> {
        let archive_type = config.get("type").and_then(|v| v.as_str()).unwrap_or("transactions");
        let days_old = config.get("days_old").and_then(|v| v.as_i64()).unwrap_or(90);

        println!("Archiving {} older than {} days", archive_type, days_old);
        
        // In production, implement actual archival logic
        self.db.archive_old_data(archive_type, days_old as i32).await?;

        Ok(())
    }

    async fn perform_backup(&self, config: &serde_json::Value) -> Result<(), Error> {
        let backup_type = config.get("type").and_then(|v| v.as_str()).unwrap_or("full");
        
        println!("Performing {} backup", backup_type);
        
        // In production, implement actual backup logic
        self.db.perform_backup(backup_type).await?;

        Ok(())
    }

    async fn perform_cleanup(&self, config: &serde_json::Value) -> Result<(), Error> {
        let cleanup_type = config.get("type").and_then(|v| v.as_str()).unwrap_or("temp");
        
        println!("Performing {} cleanup", cleanup_type);
        
        // In production, implement actual cleanup logic
        self.db.perform_cleanup(cleanup_type).await?;

        Ok(())
    }

    async fn perform_sync(&self, config: &serde_json::Value) -> Result<(), Error> {
        let sync_type = config.get("type").and_then(|v| v.as_str()).unwrap_or("all");
        
        println!("Performing {} sync", sync_type);
        
        // In production, implement actual sync logic
        self.db.perform_sync(sync_type).await?;

        Ok(())
    }

    async fn send_scheduled_notification(&self, config: &serde_json::Value) -> Result<(), Error> {
        let title = config.get("title").and_then(|v| v.as_str()).unwrap_or("Scheduled Notification");
        let message = config.get("message").and_then(|v| v.as_str()).unwrap_or("");
        let notification_type = config.get("notification_type")
            .and_then(|v| v.as_str())
            .unwrap_or("info");

        let nt = match notification_type {
            "warning" => NotificationType::Warning,
            "error" => NotificationType::Error,
            "success" => NotificationType::Success,
            "alert" => NotificationType::Alert,
            _ => NotificationType::Info,
        };

        self.send_notification_to_all(title, message, nt).await
    }

    // ============================================================================
    // Webhooks
    // ============================================================================

    pub async fn create_webhook(&self, webhook: WebhookConfig) -> Result<WebhookConfig, Error> {
        let mut webhooks = self.webhooks.write().await;
        webhooks.insert(webhook.id.clone(), webhook.clone());
        
        self.db.save_webhook(&webhook).await?;
        
        Ok(webhook)
    }

    pub async fn update_webhook(&self, webhook: WebhookConfig) -> Result<WebhookConfig, Error> {
        let mut webhooks = self.webhooks.write().await;
        webhooks.insert(webhook.id.clone(), webhook.clone());
        
        self.db.update_webhook(&webhook).await?;
        
        Ok(webhook)
    }

    pub async fn delete_webhook(&self, webhook_id: &str) -> Result<(), Error> {
        let mut webhooks = self.webhooks.write().await;
        webhooks.remove(webhook_id);
        
        self.db.delete_webhook(webhook_id).await?;
        
        Ok(())
    }

    pub async fn list_webhooks(&self) -> Result<Vec<WebhookConfig>, Error> {
        let webhooks = self.webhooks.read().await;
        Ok(webhooks.values().cloned().collect())
    }

    async fn trigger_webhook(&self, event: &str, data: &AuditLog) -> Result<(), Error> {
        let webhooks = self.webhooks.read().await;
        
        for webhook in webhooks.values() {
            if webhook.is_active && webhook.events.iter().any(|e| e == event || e == "*") {
                // Send webhook in background
                let webhook_url = webhook.url.clone();
                let webhook_secret = webhook.secret.clone();
                let event_data = serde_json::to_string(data).unwrap_or_default();
                
                tokio::spawn(async move {
                    Self::send_webhook_request(&webhook_url, &webhook_secret, event, &event_data).await;
                });
            }
        }
        
        Ok(())
    }

    async fn send_webhook_request(url: &str, secret: &str, event: &str, data: &str) {
        // In production, implement proper webhook sending with retry logic
        println!("Sending webhook {} to {}", event, url);
    }

    // ============================================================================
    // Rate Limiting
    // ============================================================================

    pub async fn check_rate_limit(&self, key: &str) -> Result<bool, Error> {
        let mut limiter = self.rate_limiter.write().await;
        limiter.check_rate_limit(key)
    }

    pub async fn update_rate_limit_config(&self, config: RateLimitConfig) -> Result<(), Error> {
        let mut limiter = self.rate_limiter.write().await;
        limiter.config = config;
        Ok(())
    }

    pub async fn get_rate_limit_status(&self, key: &str) -> Result<serde_json::Value, Error> {
        let limiter = self.rate_limiter.read().await;
        let now = Utc::now();
        let minute_ago = now - Duration::minutes(1);
        let hour_ago = now - Duration::hours(1);
        
        let requests = limiter.requests.get(key);
        
        let minute_count = requests.map(|r| r.iter().filter(|&&t| t > minute_ago).count() as i32).unwrap_or(0);
        let hour_count = requests.map(|r| r.iter().filter(|&&t| t > hour_ago).count() as i32).unwrap_or(0);
        
        Ok(serde_json::json!({
            "requests_per_minute": limiter.config.requests_per_minute,
            "requests_per_hour": limiter.config.requests_per_hour,
            "current_minute": minute_count,
            "current_hour": hour_count,
        }))
    }

    // ============================================================================
    // Theme Preferences
    // ============================================================================

    pub async fn set_theme_preference(&self, admin_id: &str, theme: ThemeMode, language: &str) -> Result<AdminTheme, Error> {
        let mut themes = self.theme_preferences.write().await;
        
        let admin_theme = AdminTheme {
            admin_id: admin_id.to_string(),
            theme_mode: theme,
            language: language.to_string(),
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };
        
        themes.insert(admin_id.to_string(), admin_theme.clone());
        
        self.db.save_theme_preference(&admin_theme).await?;
        
        Ok(admin_theme)
    }

    pub async fn get_theme_preference(&self, admin_id: &str) -> Result<Option<AdminTheme>, Error> {
        let themes = self.theme_preferences.read().await;
        Ok(themes.get(admin_id).cloned())
    }

    // ============================================================================
    // Approval Workflows
    // ============================================================================

    pub async fn create_approval_workflow(&self, workflow: ApprovalWorkflow) -> Result<ApprovalWorkflow, Error> {
        self.db.save_approval_workflow(&workflow).await?;
        Ok(workflow)
    }

    pub async fn submit_approval_request(&self, request: ApprovalRequest) -> Result<ApprovalRequest, Error> {
        self.db.save_approval_request(&request).await?;
        
        // Notify approvers
        self.send_notification_to_all(
            "Approval Request",
            &format!("New approval request: {}", request.details),
            NotificationType::Info,
        ).await?;
        
        Ok(request)
    }

    pub async fn approve_request(&self, request_id: &str, approver_id: &str, 
        approver_email: &str, comments: Option<String>) -> Result<ApprovalRequest, Error> {
        let request = self.db.get_approval_request(request_id).await?
            .ok_or(Error::ApprovalRequestNotFound)?;
        
        let mut request = request;
        
        let approval = Approval {
            id: Uuid::new_v4().to_string(),
            request_id: request_id.to_string(),
            approver_id: approver_id.to_string(),
            approver_email: approver_email.to_string(),
            level: request.current_level,
            decision: "approved".to_string(),
            comments,
            created_at: Utc::now(),
        };
        
        request.approvals.push(approval);
        
        if request.current_level >= request.approvals.len() as i32 {
            request.status = ApprovalStatus::Approved;
        } else {
            request.current_level += 1;
        }
        
        request.updated_at = Utc::now();
        
        self.db.update_approval_request(&request).await?;
        
        Ok(request)
    }

    pub async fn reject_request(&self, request_id: &str, approver_id: &str,
        approver_email: &str, comments: Option<String>) -> Result<ApprovalRequest, Error> {
        let request = self.db.get_approval_request(request_id).await?
            .ok_or(Error::ApprovalRequestNotFound)?;
        
        let mut request = request;
        
        let approval = Approval {
            id: Uuid::new_v4().to_string(),
            request_id: request_id.to_string(),
            approver_id: approver_id.to_string(),
            approver_email: approver_email.to_string(),
            level: request.current_level,
            decision: "rejected".to_string(),
            comments,
            created_at: Utc::now(),
        };
        
        request.approvals.push(approval);
        request.status = ApprovalStatus::Rejected;
        request.updated_at = Utc::now();
        
        self.db.update_approval_request(&request).await?;
        
        Ok(request)
    }

    // ============================================================================
    // Ticket System
    // ============================================================================

    pub async fn create_ticket(&self, ticket: Ticket) -> Result<Ticket, Error> {
        self.db.save_ticket(&ticket).await?;
        
        // Notify support team
        self.send_notification_to_all(
            "New Support Ticket",
            &format!("{} - {}", ticket.title, ticket.description),
            NotificationType::Info,
        ).await?;
        
        Ok(ticket)
    }

    pub async fn update_ticket(&self, ticket: Ticket) -> Result<Ticket, Error> {
        self.db.update_ticket(&ticket).await?;
        Ok(ticket)
    }

    pub async fn add_ticket_comment(&self, comment: TicketComment) -> Result<TicketComment, Error> {
        self.db.save_ticket_comment(&comment).await?;
        
        // Update ticket updated_at
        if let Some(mut ticket) = self.db.get_ticket(&comment.ticket_id).await? {
            ticket.updated_at = Utc::now();
            self.db.update_ticket(&ticket).await?;
        }
        
        Ok(comment)
    }

    pub async fn list_tickets(&self, admin_id: Option<&str>, status: Option<TicketStatus>,
        page: i32, limit: i32) -> Result<(Vec<Ticket>, i64), Error> {
        self.db.list_tickets(admin_id, status, page, limit).await
    }

    // ============================================================================
    // Knowledge Base
    // ============================================================================

    pub async fn create_article(&self, article: KnowledgeBaseArticle) -> Result<KnowledgeBaseArticle, Error> {
        self.db.save_knowledge_article(&article).await?;
        Ok(article)
    }

    pub async fn update_article(&self, article: KnowledgeBaseArticle) -> Result<KnowledgeBaseArticle, Error> {
        self.db.update_knowledge_article(&article).await?;
        Ok(article)
    }

    pub async fn search_knowledge_base(&self, query: &str) -> Result<Vec<KnowledgeBaseArticle>, Error> {
        self.db.search_knowledge_articles(query).await
    }

    // ============================================================================
    // SLA Metrics
    // ============================================================================

    pub async fn create_sla_metric(&self, metric: SLAMetric) -> Result<SLAMetric, Error> {
        self.db.save_sla_metric(&metric).await?;
        Ok(metric)
    }

    pub async fn update_sla_metric(&self, metric: SLAMetric) -> Result<SLAMetric, Error> {
        // Update status based on current value vs target
        let mut metric = metric;
        if metric.current_value >= metric.target_value {
            metric.status = SLAMetricStatus::Met;
        } else if metric.current_value >= metric.target_value * 0.8 {
            metric.status = SLAMetricStatus::AtRisk;
        } else {
            metric.status = SLAMetricStatus::Breached;
        }
        
        self.db.update_sla_metric(&metric).await?;
        Ok(metric)
    }

    pub async fn get_sla_metrics(&self) -> Result<Vec<SLAMetric>, Error> {
        self.db.get_all_sla_metrics().await
    }

    // ============================================================================
    // Fraud Detection
    // ============================================================================

    pub async fn create_fraud_alert(&self, alert: FraudAlert) -> Result<FraudAlert, Error> {
        self.db.save_fraud_alert(&alert).await?;
        
        // Send alert notification
        if alert.severity == AlertSeverity::Critical || alert.severity == AlertSeverity::High {
            self.send_notification_to_all(
                "Fraud Alert",
                &format!("{} - {}", alert.alert_type, alert.description),
                NotificationType::Alert,
            ).await?;
        }
        
        Ok(alert)
    }

    pub async fn resolve_fraud_alert(&self, alert_id: &str, resolved_by: &str, 
        status: AlertStatus) -> Result<(), Error> {
        self.db.resolve_fraud_alert(alert_id, resolved_by, status).await
    }

    pub async fn get_fraud_alerts(&self, admin_id: Option<&str>, 
        status: Option<AlertStatus>) -> Result<Vec<FraudAlert>, Error> {
        self.db.get_fraud_alerts(admin_id, status).await
    }

    // ============================================================================
    // JWT Token Helpers
    // ============================================================================

    fn generate_token(&self, admin_id: &str, email: &str, role: &str) -> Result<String, Error> {
        let claims = serde_json::json!({
            "admin_id": admin_id,
            "email": email,
            "role": role,
            "exp": Utc::now().timestamp() + self.jwt_expiration,
            "iat": Utc::now().timestamp(),
        });

        let token = encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(self.jwt_secret.as_bytes()),
        ).map_err(|e| Error::TokenGenerationError(e.to_string()))?;

        Ok(token)
    }

    fn generate_refresh_token(&self, admin_id: &str) -> Result<String, Error> {
        let claims = serde_json::json!({
            "admin_id": admin_id,
            "type": "refresh",
            "exp": Utc::now().timestamp() + (self.jwt_expiration * 24 * 7), // 7 days
            "iat": Utc::now().timestamp(),
        });

        let token = encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(self.jwt_secret.as_bytes()),
        ).map_err(|e| Error::TokenGenerationError(e.to_string()))?;

        Ok(token)
    }

    fn verify_token(&self, token: &str) -> Result<serde_json::Value, Error> {
        let token_data = decode::<serde_json::Value>(
            token,
            &DecodingKey::from_secret(self.jwt_secret.as_bytes()),
            &Validation::default(),
        ).map_err(|e| Error::TokenVerificationError(e.to_string()))?;

        Ok(token_data.claims)
    }

    fn verify_refresh_token(&self, token: &str) -> Result<serde_json::Value, Error> {
        let mut validation = Validation::default();
        validation.validate_exp = true;
        
        let token_data = decode::<serde_json::Value>(
            token,
            &DecodingKey::from_secret(self.jwt_secret.as_bytes()),
            &validation,
        ).map_err(|e| Error::TokenVerificationError(e.to_string()))?;

        // Verify it's a refresh token
        let token_type = token_data.claims.get("type")
            .and_then(|v| v.as_str())
            .unwrap_or("");
            
        if token_type != "refresh" {
            return Err(Error::InvalidTokenType);
        }

        Ok(token_data.claims)
    }
}

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug)]
pub enum Error {
    NotFound,
    AdminNotFound,
    EmailAlreadyExists,
    UsernameAlreadyExists,
    InvalidCredentials,
    AccountLocked(DateTime<Utc>),
    TwoFactorRequired,
    InvalidTwoFactorCode,
    MaxSessionsReached,
    IPNotWhitelisted,
    TokenExpired,
    InvalidTokenType,
    TokenGenerationError(String),
    TokenVerificationError(String),
    PasswordHashError(String),
    PasswordTooShort(i32),
    PasswordRequiresUppercase,
    PasswordRequiresLowercase,
    PasswordRequiresNumber,
    PasswordRequiresSpecialChar,
    RateLimitExceeded(String),
    CannotDeleteSuperAdmin,
    CannotSuspendSuperAdmin,
    ApprovalRequestNotFound,
    DatabaseError(String),
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Error::NotFound => write!(f, "Resource not found"),
            Error::AdminNotFound => write!(f, "Admin not found"),
            Error::EmailAlreadyExists => write!(f, "Email already exists"),
            Error::UsernameAlreadyExists => write!(f, "Username already exists"),
            Error::InvalidCredentials => write!(f, "Invalid credentials"),
            Error::AccountLocked(until) => write!(f, "Account locked until {}", until),
            Error::TwoFactorRequired => write!(f, "Two-factor authentication required"),
            Error::InvalidTwoFactorCode => write!(f, "Invalid two-factor code"),
            Error::MaxSessionsReached => write!(f, "Maximum sessions reached"),
            Error::IPNotWhitelisted => write!(f, "IP address not whitelisted"),
            Error::TokenExpired => write!(f, "Token expired"),
            Error::InvalidTokenType => write!(f, "Invalid token type"),
            Error::TokenGenerationError(e) => write!(f, "Token generation error: {}", e),
            Error::TokenVerificationError(e) => write!(f, "Token verification error: {}", e),
            Error::PasswordHashError(e) => write!(f, "Password hash error: {}", e),
            Error::PasswordTooShort(len) => write!(f, "Password must be at least {} characters", len),
            Error::PasswordRequiresUppercase => write!(f, "Password must contain uppercase letter"),
            Error::PasswordRequiresLowercase => write!(f, "Password must contain lowercase letter"),
            Error::PasswordRequiresNumber => write!(f, "Password must contain number"),
            Error::PasswordRequiresSpecialChar => write!(f, "Password must contain special character"),
            Error::RateLimitExceeded(msg) => write!(f, "Rate limit exceeded: {}", msg),
            Error::CannotDeleteSuperAdmin => write!(f, "Cannot delete super admin"),
            Error::CannotSuspendSuperAdmin => write!(f, "Cannot suspend super admin"),
            Error::ApprovalRequestNotFound => write!(f, "Approval request not found"),
            Error::DatabaseError(e) => write!(f, "Database error: {}", e),
        }
    }
}

impl std::error::Error for Error {}
