//! TigerWallet Admin Panel - Main Entry Point
use anyhow::Result;
use axum::Router;
use std::net::SocketAddr;
use tower_http::cors::{Any, CorsLayer};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use tiger_admin_panel::{api, services::*};

#[derive(Clone)]
pub struct AppState {
    pub auth_service: AuthService,
    pub user_service: UserService,
    pub kyc_service: KycService,
    pub transaction_service: TransactionService,
    pub withdrawal_service: WithdrawalService,
    pub token_service: TokenService,
    pub blockchain_service: BlockchainService,
    pub fee_service: FeeService,
    pub webhook_service: WebhookService,
    pub notification_service: NotificationService,
    pub ticket_service: TicketService,
    pub white_label_service: WhiteLabelService,
    pub feature_flag_service: FeatureFlagService,
}

#[derive(Debug, axum::response::IntoResponse)]
pub struct AppError(anyhow::Error);

impl<E> From<E> for AppError
where
    E: Into<anyhow::Error>,
{
    fn from(err: E) -> Self {
        Self(err.into())
    }
}

impl axum::response::Response for AppError {
    fn into_response(self) -> axum::response::Response {
        (
            axum::http::StatusCode::INTERNAL_SERVER_ERROR,
            axum::Json(serde_json::json!({
                "success": false,
                "error": self.0.to_string()
            })),
        )
            .into_response()
    }
}

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info,tiger_admin_panel=debug".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    tracing::info!("Starting TigerWallet Admin Panel (Rust Backend)");

    // Create application state
    let state = AppState {
        auth_service: AuthService,
        user_service: UserService,
        kyc_service: KycService,
        transaction_service: TransactionService,
        withdrawal_service: WithdrawalService,
        token_service: TokenService,
        blockchain_service: BlockchainService,
        fee_service: FeeService,
        webhook_service: WebhookService,
        notification_service: NotificationService,
        ticket_service: TicketService,
        white_label_service: WhiteLabelService,
        feature_flag_service: FeatureFlagService,
    };

    // CORS configuration
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    // Build router
    let app = api::router(state).layer(cors);

    // Start server
    let addr = SocketAddr::from(([0, 0, 0, 0], 3001));
    tracing::info!("Server listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
