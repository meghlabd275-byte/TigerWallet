//! API routes for TigerWallet Admin Panel
use axum::{
    routing::{get, post, put, delete},
    Router,
    extract::{Path, Query, State},
    response::Json,
};
use serde::Deserialize;
use uuid::Uuid;

use crate::models::*;
use crate::services::*;

// Health check
pub async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "healthy",
        "service": "tiger-admin-panel-rust"
    }))
}

// ==================== Auth Routes ====================

pub async fn register(
    State(state): State<AppState>,
    Json(req): Json<CreateAdminRequest>,
) -> Result<Json<ApiResponse<AdminUser>>, AppError> {
    let admin = state.auth_service.register(req).await?;
    Ok(Json(ApiResponse::success(admin)))
}

pub async fn login(
    State(state): State<AppState>,
    Json(req): Json<LoginRequest>,
) -> Result<Json<ApiResponse<LoginResponse>>, AppError> {
    let response = state.auth_service.login(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_admins(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<AdminUser>>>, AppError> {
    let admins = state.auth_service.list_admins().await?;
    Ok(Json(ApiResponse::success(admins)))
}

pub async fn get_admin(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<AdminUser>>, AppError> {
    let admin = state.auth_service.get_admin(id).await?;
    Ok(Json(ApiResponse::success(admin)))
}

pub async fn delete_admin(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.auth_service.delete_admin(id).await?;
    Ok(Json(ApiResponse::success(())))
}

// ==================== User Routes ====================

#[derive(Deserialize)]
pub struct PaginationParams {
    page: Option<i32>,
    page_size: Option<i32>,
}

pub async fn get_users(
    State(state): State<AppState>,
    Query(params): Query<PaginationParams>,
) -> Result<Json<ApiResponse<PaginatedResponse<User>>>, AppError> {
    let page = params.page.unwrap_or(1);
    let page_size = params.page_size.unwrap_or(50);
    let result = state.user_service.list_users(page, page_size).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn get_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<User>>, AppError> {
    let user = state.user_service.get_user(id).await?;
    Ok(Json(ApiResponse::success(user)))
}

pub async fn ban_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.user_service.ban_user(id).await?;
    Ok(Json(ApiResponse::success(())))
}

pub async fn unban_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.user_service.unban_user(id).await?;
    Ok(Json(ApiResponse::success(())))
}

// ==================== KYC Routes ====================

pub async fn get_kyc_requests(
    State(state): State<AppState>,
    Query(params): Query<PaginationParams>,
) -> Result<Json<ApiResponse<PaginatedResponse<KycRequest>>>, AppError> {
    let page = params.page.unwrap_or(1);
    let page_size = params.page_size.unwrap_or(50);
    let result = state.kyc_service.list_requests(page, page_size).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn approve_kyc(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.kyc_service.approve(id).await?;
    Ok(Json(ApiResponse::success(())))
}

pub async fn reject_kyc(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.kyc_service.reject(id).await?;
    Ok(Json(ApiResponse::success(())))
}

// ==================== Transaction Routes ====================

pub async fn get_transactions(
    State(state): State<AppState>,
    Query(params): Query<PaginationParams>,
) -> Result<Json<ApiResponse<PaginatedResponse<Transaction>>>, AppError> {
    let page = params.page.unwrap_or(1);
    let page_size = params.page_size.unwrap_or(50);
    let result = state.transaction_service.list_transactions(page, page_size).await?;
    Ok(Json(ApiResponse::success(result)))
}

// ==================== Withdrawal Routes ====================

pub async fn get_withdrawals(
    State(state): State<AppState>,
    Query(params): Query<PaginationParams>,
) -> Result<Json<ApiResponse<PaginatedResponse<Withdrawal>>>, AppError> {
    let page = params.page.unwrap_or(1);
    let page_size = params.page_size.unwrap_or(50);
    let result = state.withdrawal_service.list_withdrawals(page, page_size).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn approve_withdrawal(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.withdrawal_service.approve(id).await?;
    Ok(Json(ApiResponse::success(())))
}

pub async fn reject_withdrawal(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.withdrawal_service.reject(id).await?;
    Ok(Json(ApiResponse::success(())))
}

// ==================== Token Routes ====================

pub async fn get_tokens(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<Token>>>, AppError> {
    let tokens = state.token_service.list_tokens().await?;
    Ok(Json(ApiResponse::success(tokens)))
}

pub async fn create_token(
    State(state): State<AppState>,
    Json(token): Json<Token>,
) -> Result<Json<ApiResponse<Token>>, AppError> {
    let token = state.token_service.create_token(token).await?;
    Ok(Json(ApiResponse::success(token)))
}

// ==================== Blockchain Routes ====================

pub async fn get_blockchains(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<Blockchain>>>, AppError> {
    let blockchains = state.blockchain_service.list_blockchains().await?;
    Ok(Json(ApiResponse::success(blockchains)))
}

pub async fn create_blockchain(
    State(state): State<AppState>,
    Json(bc): Json<Blockchain>,
) -> Result<Json<ApiResponse<Blockchain>>, AppError> {
    let bc = state.blockchain_service.create_blockchain(bc).await?;
    Ok(Json(ApiResponse::success(bc)))
}

// ==================== Fee Routes ====================

pub async fn get_fees(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<FeeStructure>>>, AppError> {
    let fees = state.fee_service.list_fees().await?;
    Ok(Json(ApiResponse::success(fees)))
}

// ==================== Webhook Routes ====================

pub async fn get_webhooks(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<Webhook>>>, AppError> {
    let webhooks = state.webhook_service.list_webhooks().await?;
    Ok(Json(ApiResponse::success(webhooks)))
}

// ==================== Notification Routes ====================

pub async fn get_notifications(
    State(state): State<AppState>,
    Query(params): Query<PaginationParams>,
) -> Result<Json<ApiResponse<PaginatedResponse<Notification>>>, AppError> {
    let page = params.page.unwrap_or(1);
    let page_size = params.page_size.unwrap_or(50);
    let result = state.notification_service.list_notifications(page, page_size).await?;
    Ok(Json(ApiResponse::success(result)))
}

// ==================== Ticket Routes ====================

pub async fn get_tickets(
    State(state): State<AppState>,
    Query(params): Query<PaginationParams>,
) -> Result<Json<ApiResponse<PaginatedResponse<Ticket>>>, AppError> {
    let page = params.page.unwrap_or(1);
    let page_size = params.page_size.unwrap_or(50);
    let result = state.ticket_service.list_tickets(page, page_size).await?;
    Ok(Json(ApiResponse::success(result)))
}

// ==================== White Label Routes ====================

pub async fn get_white_labels(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<WhiteLabel>>>, AppError> {
    let white_labels = state.white_label_service.list_white_labels().await?;
    Ok(Json(ApiResponse::success(white_labels)))
}

// ==================== Stats Routes ====================

pub async fn get_stats(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<PlatformStats>>, AppError> {
    let stats = state.user_service.get_stats().await?;
    Ok(Json(ApiResponse::success(stats)))
}

// ==================== Feature Flag Routes ====================

pub async fn get_feature_flags(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<Vec<FeatureFlag>>>, AppError> {
    let flags = state.feature_flag_service.list_flags().await?;
    Ok(Json(ApiResponse::success(flags)))
}

// ==================== Router ====================

use crate::AppState;

pub fn router(state: AppState) -> Router {
    Router::new()
        // Health
        .route("/health", get(health_check))
        
        // Auth (public)
        .route("/api/v1/auth/register", post(register))
        .route("/api/v1/auth/login", post(login))
        
        // Admin management (protected)
        .route("/api/v1/admin/admins", get(get_admins))
        .route("/api/v1/admin/admins/:id", get(get_admin).delete(delete_admin))
        
        // Users
        .route("/api/v1/admin/users", get(get_users))
        .route("/api/v1/admin/users/:id", get(get_user))
        .route("/api/v1/admin/users/:id/ban", post(ban_user))
        .route("/api/v1/admin/users/:id/unban", post(unban_user))
        
        // KYC
        .route("/api/v1/admin/kyc", get(get_kyc_requests))
        .route("/api/v1/admin/kyc/:id/approve", post(approve_kyc))
        .route("/api/v1/admin/kyc/:id/reject", post(reject_kyc))
        
        // Transactions
        .route("/api/v1/admin/transactions", get(get_transactions))
        
        // Withdrawals
        .route("/api/v1/admin/withdrawals", get(get_withdrawals))
        .route("/api/v1/admin/withdrawals/:id/approve", post(approve_withdrawal))
        .route("/api/v1/admin/withdrawals/:id/reject", post(reject_withdrawal))
        
        // Tokens
        .route("/api/v1/admin/tokens", get(get_tokens).post(create_token))
        
        // Blockchains
        .route("/api/v1/admin/blockchains", get(get_blockchains).post(create_blockchain))
        
        // Fees
        .route("/api/v1/admin/fees", get(get_fees))
        
        // Webhooks
        .route("/api/v1/admin/webhooks", get(get_webhooks))
        
        // Notifications
        .route("/api/v1/admin/notifications", get(get_notifications))
        
        // Tickets
        .route("/api/v1/admin/tickets", get(get_tickets))
        
        // White Labels
        .route("/api/v1/admin/white-labels", get(get_white_labels))
        
        // Stats
        .route("/api/v1/admin/stats", get(get_stats))
        
        // Feature Flags
        .route("/api/v1/admin/feature-flags", get(get_feature_flags))
        
        .with_state(state)
}
