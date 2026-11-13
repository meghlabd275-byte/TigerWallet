//! TigerWallet in-house KYC service.
//!
//! Exposes the verification engine over HTTP and persists decisions in
//! PostgreSQL. This is the service `go/wallet_api`'s `kycVerified` gate calls,
//! so its status vocabulary is a contract: the wallet gates P2P trading and
//! card issuance on the value `verified`, which only this service writes.
//!
//! There is no external identity vendor. Verification is performed by
//! `tiger-kyc-engine` against documents the applicant supplies and watchlists
//! the operator loads.

mod auth;
mod handlers;
mod store;

use std::net::SocketAddr;
use std::sync::Arc;

use axum::routing::{get, post, put};
use axum::Router;
use tower_http::limit::RequestBodyLimitLayer;
use tower_http::trace::TraceLayer;
use tracing_subscriber::EnvFilter;

use crate::store::Store;

/// Shared application state.
pub struct AppState {
    pub store: Store,
    /// Shared secret required on the internal API. Requests without it are
    /// rejected, so the service cannot be driven by an unauthenticated caller.
    pub api_token: String,
    /// Secret used to authenticate review actions, held separately so that a
    /// leak of the service token does not grant review authority.
    pub reviewer_token: String,
}

/// Largest accepted request body. Identity payloads are small; this bounds the
/// memory a single request can consume.
const MAX_BODY_BYTES: usize = 256 * 1024;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    let database_url = match std::env::var("KYC_DATABASE_URL") {
        Ok(v) if !v.trim().is_empty() => v,
        _ => {
            // Fail closed. Without a database there is nowhere to record a
            // decision, and an in-memory fallback would silently lose every
            // verification on restart.
            eprintln!("KYC_DATABASE_URL is not set; refusing to start");
            std::process::exit(1);
        }
    };

    let api_token = match std::env::var("KYC_API_TOKEN") {
        Ok(v) if !v.trim().is_empty() => v,
        _ => {
            eprintln!("KYC_API_TOKEN is not set; refusing to start");
            std::process::exit(1);
        }
    };

    let reviewer_token = match std::env::var("KYC_REVIEWER_TOKEN") {
        Ok(v) if !v.trim().is_empty() => v,
        _ => {
            eprintln!("KYC_REVIEWER_TOKEN is not set; refusing to start");
            std::process::exit(1);
        }
    };

    let max_conns: u32 = std::env::var("KYC_MAX_CONNECTIONS")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(16);

    let store = match Store::connect(&database_url, max_conns).await {
        Ok(s) => s,
        Err(e) => {
            eprintln!("cannot connect to PostgreSQL: {e}");
            std::process::exit(1);
        }
    };

    let state = Arc::new(AppState {
        store,
        api_token,
        reviewer_token,
    });

    let app = Router::new()
        .route("/health", get(handlers::health))
        .route("/readyz", get(handlers::readyz))
        // Applicant-facing.
        .route("/api/v1/kyc/submit", post(handlers::submit))
        .route("/api/v1/kyc/status/:user_id", get(handlers::status))
        // Reviewer-facing. Both require the reviewer token.
        .route("/api/v1/kyc/applications", get(handlers::list_applications))
        .route("/api/v1/kyc/review/:user_id", put(handlers::review))
        // Operator-facing watchlist management.
        .route(
            "/api/v1/kyc/watchlist",
            post(handlers::add_watchlist_entry).delete(handlers::remove_watchlist_entry),
        )
        .with_state(state)
        .layer(RequestBodyLimitLayer::new(MAX_BODY_BYTES))
        .layer(TraceLayer::new_for_http());

    let port: u16 = std::env::var("KYC_PORT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(8470);
    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    tracing::info!("tiger-kyc-service listening on {addr}");

    let listener = match tokio::net::TcpListener::bind(addr).await {
        Ok(l) => l,
        Err(e) => {
            eprintln!("cannot bind {addr}: {e}");
            std::process::exit(1);
        }
    };

    if let Err(e) = axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
    {
        eprintln!("server error: {e}");
        std::process::exit(1);
    }
}

async fn shutdown_signal() {
    let ctrl_c = async {
        let _ = tokio::signal::ctrl_c().await;
    };
    #[cfg(unix)]
    let terminate = async {
        if let Ok(mut sig) = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
        {
            sig.recv().await;
        }
    };
    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
    tracing::info!("shutdown signal received");
}
