//! TigerAdmin Rust Backend - Main Entry Point
//! High-performance admin operations

use axum::{
    Router,
    routing::{get, post, put, delete},
    middleware,
    Json, Extension,
};
use std::net::SocketAddr;
use std::sync::Arc;
use tower_http::cors::{CorsLayer, Any};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

mod models;
mod handlers;
mod db;
mod auth;
mod error;
mod domain;

use handlers::*;
use domain::domain_routes;
use db::DbPool;
use auth::AuthState;

#[derive(Clone)]
struct AppState {
    db: DbPool,
    auth: AuthState,
}

#[tokio::main]
async fn main() {
    // Initialize logging
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info,tiger_admin=debug".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    tracing::info!("Starting TigerAdmin Rust Backend...");

    // Initialize database
    let db = db::create_pool().await.expect("Failed to create database pool");
    
    // Run migrations
    db::run_migrations(&db).await.expect("Failed to run migrations");

    // Initialize auth state
    let auth = auth::AuthState::new();

    // Create app state
    let state = AppState {
        db: db.clone(),
        auth: auth.clone(),
    };

    // CORS configuration
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    // Build router
    let app = Router::new()
        // Health check
        .route("/health", get(health_check))
        
        // Auth routes
        .route("/api/v1/auth/login", post(login))
        .route("/api/v1/auth/logout", post(logout))
        .route("/api/v1/auth/refresh", post(refresh_token))
        .route("/api/v1/auth/2fa/setup", post(setup_2fa))
        .route("/api/v1/auth/2fa/verify", post(verify_2fa))
        
        // Admin routes (protected)
        .route("/api/v1/admins", get(list_admins))
        .route("/api/v1/admins", post(create_admin))
        .route("/api/v1/admins/:id", get(get_admin))
        .route("/api/v1/admins/:id", put(update_admin))
        .route("/api/v1/admins/:id", delete(delete_admin))
        .route("/api/v1/admins/:id/suspend", post(suspend_admin))
        .route("/api/v1/admins/:id/activate", post(activate_admin))
        
        // User routes
        .route("/api/v1/users", get(list_users))
        .route("/api/v1/users/:id", get(get_user))
        .route("/api/v1/users/:id", put(update_user))
        .route("/api/v1/users/:id/ban", post(ban_user))
        .route("/api/v1/users/:id/unban", post(unban_user))
        .route("/api/v1/users/:id/suspend", post(suspend_user))
        
        // KYC routes
        .route("/api/v1/kyc", get(list_kyc))
        .route("/api/v1/kyc/:id", get(get_kyc))
        .route("/api/v1/kyc/:id/approve", post(approve_kyc))
        .route("/api/v1/kyc/:id/reject", post(reject_kyc))
        
        // Transaction routes
        .route("/api/v1/transactions", get(list_transactions))
        .route("/api/v1/transactions/:id", get(get_transaction))
        .route("/api/v1/transactions/:id/flag", post(flag_transaction))
        .route("/api/v1/transactions/:id/unflag", post(unflag_transaction))
        
        // Withdrawal routes
        .route("/api/v1/withdrawals", get(list_withdrawals))
        .route("/api/v1/withdrawals/:id", get(get_withdrawal))
        .route("/api/v1/withdrawals/:id/approve", post(approve_withdrawal))
        .route("/api/v1/withdrawals/:id/reject", post(reject_withdrawal))
        .route("/api/v1/withdrawals/:id/process", post(process_withdrawal))
        
        // Token routes
        .route("/api/v1/tokens", get(list_tokens))
        .route("/api/v1/tokens/:id", get(get_token))
        .route("/api/v1/tokens", post(create_token))
        .route("/api/v1/tokens/:id", put(update_token))
        .route("/api/v1/tokens/:id", delete(delete_token))
        .route("/api/v1/tokens/:id/verify", post(verify_token))
        
        // Pair routes
        .route("/api/v1/pairs", get(list_pairs))
        .route("/api/v1/pairs/:id", get(get_pair))
        .route("/api/v1/pairs", post(create_pair))
        .route("/api/v1/pairs/:id", put(update_pair))
        .route("/api/v1/pairs/:id/halt", post(halt_pair))
        .route("/api/v1/pairs/:id/activate", post(activate_pair))
        
        // Blockchain routes
        .route("/api/v1/blockchains", get(list_blockchains))
        .route("/api/v1/blockchains/:id", get(get_blockchain))
        .route("/api/v1/blockchains", post(create_blockchain))
        .route("/api/v1/blockchains/:id", put(update_blockchain))
        
        // Fee routes
        .route("/api/v1/fees", get(list_fees))
        .route("/api/v1/fees", post(create_fee))
        .route("/api/v1/fees/:id", put(update_fee))
        
        // White label routes
        .route("/api/v1/whitelabels", get(list_whitelabels))
        .route("/api/v1/whitelabels/:id", get(get_whitelabel))
        .route("/api/v1/whitelabels", post(create_whitelabel))
        .route("/api/v1/whitelabels/:id", put(update_whitelabel))
        .route("/api/v1/whitelabels/:id/activate", post(activate_whitelabel))
        .route("/api/v1/whitelabels/:id/suspend", post(suspend_whitelabel))
        
        // Ticket routes
        .route("/api/v1/tickets", get(list_tickets))
        .route("/api/v1/tickets/:id", get(get_ticket))
        .route("/api/v1/tickets", post(create_ticket))
        .route("/api/v1/tickets/:id/status", put(update_ticket_status))
        .route("/api/v1/tickets/:id/assign", put(assign_ticket))
        .route("/api/v1/tickets/:id/messages", post(add_ticket_message))
        
        // Analytics routes
        .route("/api/v1/analytics/dashboard", get(dashboard_stats))
        .route("/api/v1/analytics/users", get(user_analytics))
        .route("/api/v1/analytics/transactions", get(transaction_analytics))
        .route("/api/v1/analytics/revenue", get(revenue_analytics))
        
        // Audit routes
        .route("/api/v1/audit-logs", get(list_audit_logs))
        .route("/api/v1/audit-logs/export", post(export_audit_logs))
        
        // Feature flags
        .route("/api/v1/feature-flags", get(list_feature_flags))
        .route("/api/v1/feature-flags", post(create_feature_flag))
        .route("/api/v1/feature-flags/:id", put(update_feature_flag))
        .route("/api/v1/feature-flags/:id", delete(delete_feature_flag))
        
        // Notifications
        .route("/api/v1/notifications", get(list_notifications))
        .route("/api/v1/notifications/:id/read", put(mark_notification_read))
        .route("/api/v1/notifications/broadcast", post(broadcast_notification))
        
        // IP whitelist
        .route("/api/v1/ip-whitelist", get(list_ip_whitelist))
        .route("/api/v1/ip-whitelist", post(add_ip_whitelist))
        .route("/api/v1/ip-whitelist/:id", delete(remove_ip_whitelist))
        
        // Backups
        .route("/api/v1/backups", get(list_backups))
        .route("/api/v1/backups", post(create_backup))
        .route("/api/v1/backups/:id/restore", post(restore_backup))
        .route("/api/v1/backups/:id", delete(delete_backup))
        
        // Webhooks
        .route("/api/v1/webhooks", get(list_webhooks))
        .route("/api/v1/webhooks", post(create_webhook))
        .route("/api/v1/webhooks/:id", put(update_webhook))
        .route("/api/v1/webhooks/:id/test", post(test_webhook))
        .route("/api/v1/webhooks/:id", delete(delete_webhook))
        
        // Middleware
        .layer(cors)
        .layer(Extension(Arc::new(state)));

    // Merge the 12 admin domain proxy routes (futures, options, copy-trading,
    // convert, onramp, offramp, p2p-clients, p2p-merchants, partners, rewards,
    // marketing, roles) which forward every call to the admin/go backend on
    // localhost:9093 with the inbound Bearer JWT.
    let app = app.merge(domain_routes());

    // Start server
    let addr = SocketAddr::from(([0, 0, 0, 0], 9095));
    tracing::info!("TigerAdmin Rust Backend listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

// Health check
async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "healthy",
        "service": "tiger-admin-rust",
        "version": "1.0.0"
    }))
}
