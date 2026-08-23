//! API routes. Auth is handled natively (real bcrypt + sqlx + HS256 JWT);
//! every `/api/v1/admin/*` route is a transparent proxy to the real Go
//! super-admin backend. No stubs, no fabricated data.

use axum::{
    http::StatusCode,
    middleware,
    response::{IntoResponse, Response},
    routing::{delete, get, post, put},
    Json, Router,
};
use sqlx::{postgres::PgPoolOptions, PgPool};
use std::sync::OnceLock;

use crate::domain::{self, proxy_domain, AppState, DomainRouterExt};
use crate::middleware::auth::{self, jwt_auth_middleware};
use crate::models::*;
use crate::services::auth_service;

/// Health check.
pub async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({ "status": "ok", "service": "super_admin_rust" }))
}

/// Postgres pool built lazily from `DATABASE_URL`. `None` (fail-closed) when
/// the variable is unset or the URL is invalid.
static DB_POOL: OnceLock<Option<PgPool>> = OnceLock::new();

fn db_pool() -> Option<&'static PgPool> {
    DB_POOL
        .get_or_init(|| {
            std::env::var("DATABASE_URL")
                .ok()
                .filter(|s| !s.is_empty())
                .and_then(|url| {
                    PgPoolOptions::new()
                        .max_connections(10)
                        .connect_lazy(&url)
                        .ok()
                })
        })
        .as_ref()
}

/// Real login: bcrypt verification against the `admins` table and real HS256
/// JWT issuance. 503 when DATABASE_URL/JWT_SECRET are not configured, 401 on
/// invalid credentials.
pub async fn login(Json(req): Json<LoginRequest>) -> Response {
    let (pool, secret) = match (db_pool(), auth::jwt_secret()) {
        (Some(pool), Some(secret)) => (pool, secret),
        _ => {
            return (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(serde_json::json!({
                    "error": "DATABASE_URL and JWT_SECRET must be configured"
                })),
            )
                .into_response();
        }
    };

    match auth_service::login(pool, secret, req).await {
        Ok(resp) => Json(ApiResponse::success(resp)).into_response(),
        Err(e) => (
            StatusCode::UNAUTHORIZED,
            Json(serde_json::json!({ "error": e.to_string() })),
        )
            .into_response(),
    }
}

/// Build the axum router.
///
/// - `/health` and `/api/v1/auth/*` are public.
/// - `/api/v1/auth/login` is served natively with real DB-backed auth.
/// - `/api/v1/auth/register` is proxied to the governed Go backend (account
///   creation is never done locally).
/// - Everything under `/api/v1/admin/*` requires a valid JWT and is proxied
///   to the Go backend.
pub fn router() -> Router {
    let state = AppState::new();

    // Protected admin subtree: real JWT validation, then transparent proxy.
    let admin = Router::new()
        .route("/api/v1/admin/summary", get(proxy_domain))
        .route("/api/v1/admin/logs", get(proxy_domain))
        .route(
            "/api/v1/admin/platform/approve_transaction",
            post(proxy_domain),
        )
        .route(
            "/api/v1/admin/platform/reject_transaction",
            post(proxy_domain),
        )
        .route("/api/v1/admin/moneyflow/summary", get(proxy_domain))
        .route("/api/v1/admin/cards/summary", get(proxy_domain))
        .route("/api/v1/admin/cards/summary/export", get(proxy_domain))
        .route("/api/v1/admin/bots/executions", get(proxy_domain))
        .route("/api/v1/admin/bots/positions", get(proxy_domain))
        .route("/api/v1/admin/bots/trades", get(proxy_domain))
        .route("/api/v1/admin/bots/pnl", get(proxy_domain))
        .route("/api/v1/admin/bots/position/summary", get(proxy_domain))
        .register_domains()
        // Generic catch-all proxy for domain subtrees
        .route("/api/v1/admin/:domain", get(proxy_domain))
        .route("/api/v1/admin/:domain", post(proxy_domain))
        .route("/api/v1/admin/:domain/:id", get(proxy_domain))
        .route("/api/v1/admin/:domain/:id", post(proxy_domain))
        .route("/api/v1/admin/:domain/:id", put(proxy_domain))
        .route("/api/v1/admin/:domain/:id", delete(proxy_domain))
        .route(
            "/api/v1/admin/:domain/:id/:action",
            get(proxy_domain),
        )
        .route(
            "/api/v1/admin/:domain/:id/:action",
            post(proxy_domain),
        )
        .route_layer(middleware::from_fn(jwt_auth_middleware));

    Router::new()
        .route("/health", get(health_check))
        .route("/api/v1/auth/login", post(login))
        .route("/api/v1/auth/register", post(domain::proxy_auth_register))
        .merge(admin)
        .with_state(state)
}
