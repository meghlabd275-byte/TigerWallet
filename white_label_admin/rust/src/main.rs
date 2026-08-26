//! TigerWallet Admin - Main Entry Point
use anyhow::Result;
use std::net::SocketAddr;
use tower_http::cors::{Any, CorsLayer};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info,tiger_admin=debug".into()))
        .with(tracing_subscriber::fmt::layer())
        .init();

    tracing::info!("Starting TigerWallet Admin (Rust Backend)");

    // Build the PgPool-backed AppState and run idempotent schema migrations so
    // the admin tables exist (mirrors the Go backend's runMigrations).
    let state = tiger_admin::database::build_state().await?;
    tiger_admin::database::run_migrations(&state.pool).await?;

    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    let app = tiger_admin::api::router(state).layer(cors);

    let addr = SocketAddr::from(([0, 0, 0, 0], 8456));
    tracing::info!("Server listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
