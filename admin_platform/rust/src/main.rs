/**
 * TigerWallet Admin Platform - Rust High-Speed Server
 * Ultra-low latency REST API server
 */

use actix_cors::Cors;
use actix_web::{web, App, HttpResponse, HttpServer, middleware};
use actix_web::dev::Server;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::Mutex;
use tracing::{info, error, Level};
use tracing_subscriber::FmtSubscriber;

// Import services
mod config;
mod database;
mod services;

use config::Config;
use database::Database;
use services::*;

pub struct AppState {
    pub db: Arc<Mutex<Database>>,
    pub config: Config,
}

// JSON Response types
#[derive(Debug, Serialize, Deserialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub data: Option<T>,
    pub error: Option<String>,
    pub meta: Option<PaginationMeta>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PaginationMeta {
    pub page: i32,
    pub limit: i32,
    pub total: i32,
    pub total_pages: i32,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ErrorResponse {
    pub error: String,
    pub message: String,
}

// Health check endpoint
async fn health_check() -> HttpResponse {
    HttpResponse::Ok().json(ApiResponse::<String> {
        success: true,
        data: Some("Healthy".to_string()),
        error: None,
        meta: None,
    })
}

// Not found handler
async fn not_found() -> HttpResponse {
    HttpResponse::NotFound().json(ApiResponse::<()> {
        success: false,
        data: None,
        error: Some("Endpoint not found".to_string()),
        meta: None,
    })
}

// Auth endpoints
async fn login(
    state: web::Data<AppState>,
    req: web::Json<LoginRequest>,
) -> HttpResponse {
    let auth_service = AuthService::new(state.db.clone(), state.config.jwt_secret.clone());
    
    match auth_service.login(&req.email, &req.password).await {
        Ok(response) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(response),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::Unauthorized().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn logout(
    state: web::Data<AppState>,
    _req: web::HttpRequest,
) -> HttpResponse {
    HttpResponse::Ok().json(ApiResponse::<String> {
        success: true,
        data: Some("Logged out successfully".to_string()),
        error: None,
        meta: None,
    })
}

async fn get_current_admin(
    state: web::Data<AppState>,
    _req: web::HttpRequest,
) -> HttpResponse {
    // In production, get admin from JWT token
    let admin_service = AdminService::new(state.db.clone());
    
    match admin_service.get_admin("default").await {
        Ok(admin) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(admin),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::NotFound().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

// User endpoints
async fn get_users(
    state: web::Data<AppState>,
    params: web::Query<GetUsersParams>,
) -> HttpResponse {
    let user_service = UserService::new(state.db.clone());
    
    let page = params.page.unwrap_or(1);
    let limit = params.limit.unwrap_or(20);
    
    match user_service.list_users(page, limit, params.status.clone(), params.search.clone()).await {
        Ok((users, total)) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(users),
            error: None,
            meta: Some(PaginationMeta {
                page,
                limit,
                total,
                total_pages: (total + limit - 1) / limit,
            }),
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn get_user(
    state: web::Data<AppState>,
    path: web::Path<String>,
) -> HttpResponse {
    let user_service = UserService::new(state.db.clone());
    
    match user_service.get_user(&path).await {
        Ok(user) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(user),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::NotFound().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn update_user(
    state: web::Data<AppState>,
    path: web::Path<String>,
    req: web::Json<UpdateUserRequest>,
) -> HttpResponse {
    let user_service = UserService::new(state.db.clone());
    
    match user_service.update_user(&path, req.into_inner()).await {
        Ok(user) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(user),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn suspend_user(
    state: web::Data<AppState>,
    path: web::Path<String>,
    req: web::Json<SuspendRequest>,
) -> HttpResponse {
    let user_service = UserService::new(state.db.clone());
    
    match user_service.suspend_user(&path, &req.reason).await {
        Ok(_) => HttpResponse::Ok().json(ApiResponse::<String> {
            success: true,
            data: Some("User suspended successfully".to_string()),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn ban_user(
    state: web::Data<AppState>,
    path: web::Path<String>,
    req: web::Json<BanRequest>,
) -> HttpResponse {
    let user_service = UserService::new(state.db.clone());
    
    match user_service.ban_user(&path, &req.reason).await {
        Ok(_) => HttpResponse::Ok().json(ApiResponse::<String> {
            success: true,
            data: Some("User banned successfully".to_string()),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

// KYC endpoints
async fn get_kyc_submissions(
    state: web::Data<AppState>,
    params: web::Query<GetKYCParams>,
) -> HttpResponse {
    let kyc_service = KYCService::new(state.db.clone());
    
    let page = params.page.unwrap_or(1);
    let limit = params.limit.unwrap_or(20);
    
    match kyc_service.list_kyc(page, limit, params.status, params.level).await {
        Ok((submissions, total)) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(submissions),
            error: None,
            meta: Some(PaginationMeta {
                page,
                limit,
                total,
                total_pages: (total + limit - 1) / limit,
            }),
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn approve_kyc(
    state: web::Data<AppState>,
    path: web::Path<String>,
    req: web::Json<ApproveKYCRequest>,
) -> HttpResponse {
    let kyc_service = KYCService::new(state.db.clone());
    
    match kyc_service.approve_kyc(&path, req.notes.clone()).await {
        Ok(_) => HttpResponse::Ok().json(ApiResponse::<String> {
            success: true,
            data: Some("KYC approved successfully".to_string()),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn reject_kyc(
    state: web::Data<AppState>,
    path: web::Path<String>,
    req: web::Json<RejectKYCRequest>,
) -> HttpResponse {
    let kyc_service = KYCService::new(state.db.clone());
    
    match kyc_service.reject_kyc(&path, &req.reason).await {
        Ok(_) => HttpResponse::Ok().json(ApiResponse::<String> {
            success: true,
            data: Some("KYC rejected successfully".to_string()),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

// Token endpoints
async fn get_tokens(
    state: web::Data<AppState>,
    params: web::Query<GetTokensParams>,
) -> HttpResponse {
    let token_service = TokenService::new(state.db.clone());
    
    let page = params.page.unwrap_or(1);
    let limit = params.limit.unwrap_or(20);
    
    match token_service.list_tokens(page, limit, params.status, params.chain.clone()).await {
        Ok((tokens, total)) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(tokens),
            error: None,
            meta: Some(PaginationMeta {
                page,
                limit,
                total,
                total_pages: (total + limit - 1) / limit,
            }),
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn create_token(
    state: web::Data<AppState>,
    req: web::Json<CreateTokenRequest>,
) -> HttpResponse {
    let token_service = TokenService::new(state.db.clone());
    
    match token_service.create_token(req.into_inner()).await {
        Ok(token) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(token),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn verify_token(
    state: web::Data<AppState>,
    path: web::Path<String>,
) -> HttpResponse {
    let token_service = TokenService::new(state.db.clone());
    
    match token_service.verify_token(&path).await {
        Ok(_) => HttpResponse::Ok().json(ApiResponse::<String> {
            success: true,
            data: Some("Token verified successfully".to_string()),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

async fn delete_token(
    state: web::Data<AppState>,
    path: web::Path<String>,
) -> HttpResponse {
    let token_service = TokenService::new(state.db.clone());
    
    match token_service.delete_token(&path).await {
        Ok(_) => HttpResponse::Ok().json(ApiResponse::<String> {
            success: true,
            data: Some("Token deleted successfully".to_string()),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

// Dashboard endpoint
async fn get_dashboard(
    state: web::Data<AppState>,
) -> HttpResponse {
    let analytics_service = AnalyticsService::new(state.db.clone());
    
    match analytics_service.get_dashboard_stats().await {
        Ok(stats) => HttpResponse::Ok().json(ApiResponse {
            success: true,
            data: Some(stats),
            error: None,
            meta: None,
        }),
        Err(e) => HttpResponse::InternalServerError().json(ApiResponse::<()> {
            success: false,
            data: None,
            error: Some(e.message),
            meta: None,
        }),
    }
}

// Request types
#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct GetUsersParams {
    pub page: Option<i32>,
    pub limit: Option<i32>,
    pub status: Option<String>,
    pub search: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateUserRequest {
    pub username: Option<String>,
    pub email: Option<String>,
    pub phone: Option<String>,
    pub status: Option<String>,
    pub kyc_status: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SuspendRequest {
    pub reason: String,
}

#[derive(Debug, Deserialize)]
pub struct BanRequest {
    pub reason: String,
}

#[derive(Debug, Deserialize)]
pub struct GetKYCParams {
    pub page: Option<i32>,
    pub limit: Option<i32>,
    pub status: Option<String>,
    pub level: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct ApproveKYCRequest {
    pub notes: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RejectKYCRequest {
    pub reason: String,
}

#[derive(Debug, Deserialize)]
pub struct GetTokensParams {
    pub page: Option<i32>,
    pub limit: Option<i32>,
    pub status: Option<String>,
    pub chain: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CreateTokenRequest {
    pub name: String,
    pub symbol: String,
    pub contract_addr: String,
    pub decimals: i32,
    pub chain_id: String,
    pub logo_url: Option<String>,
    pub website: Option<String>,
}

fn configure_services(cfg: &mut web::ServiceConfig) {
    // Health check
    cfg.service(
        web::resource("/health")
            .route(web::get().to(health_check))
    );
    
    // Auth routes
    cfg.service(
        web::resource("/api/v1/auth/login")
            .route(web::post().to(login))
    );
    cfg.service(
        web::resource("/api/v1/auth/logout")
            .route(web::post().to(logout))
    );
    cfg.service(
        web::resource("/api/v1/auth/me")
            .route(web::get().to(get_current_admin))
    );
    
    // User routes
    cfg.service(
        web::resource("/api/v1/users")
            .route(web::get().to(get_users))
    );
    cfg.service(
        web::resource("/api/v1/users/{id}")
            .route(web::get().to(get_user))
            .route(web::put().to(update_user))
    );
    cfg.service(
        web::resource("/api/v1/users/{id}/suspend")
            .route(web::post().to(suspend_user))
    );
    cfg.service(
        web::resource("/api/v1/users/{id}/ban")
            .route(web::post().to(ban_user))
    );
    
    // KYC routes
    cfg.service(
        web::resource("/api/v1/kyc")
            .route(web::get().to(get_kyc_submissions))
    );
    cfg.service(
        web::resource("/api/v1/kyc/{id}/approve")
            .route(web::post().to(approve_kyc))
    );
    cfg.service(
        web::resource("/api/v1/kyc/{id}/reject")
            .route(web::post().to(reject_kyc))
    );
    
    // Token routes
    cfg.service(
        web::resource("/api/v1/tokens")
            .route(web::get().to(get_tokens))
            .route(web::post().to(create_token))
    );
    cfg.service(
        web::resource("/api/v1/tokens/{id}/verify")
            .route(web::post().to(verify_token))
    );
    cfg.service(
        web::resource("/api/v1/tokens/{id}")
            .route(web::delete().to(delete_token))
    );
    
    // Dashboard
    cfg.service(
        web::resource("/api/v1/dashboard")
            .route(web::get().to(get_dashboard))
    );
    
    // Fallback
    cfg.default_service(web::route().to(not_found));
}

pub fn create_server(config: Config, db: Database) -> Server {
    let bind_addr = format!("{}:{}", config.server_host, config.server_port);
    let db = Arc::new(Mutex::new(db));
    
    let server = HttpServer::new(move || {
        let cors = Cors::permissive();
        
        App::new()
            .wrap(cors)
            .wrap(middleware::Logger::default())
            .app_data(web::Data::new(AppState {
                db: db.clone(),
                config: config.clone(),
            }))
            .configure(configure_services)
    })
    .bind(&bind_addr)
    .expect("Failed to bind server")
    .disable_signals()
    .run();
    
    server
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    // Initialize logging
    let subscriber = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .with_target(false)
        .finish();
    
    tracing::subscriber::set_global_default(subscriber)
        .expect("Failed to set tracing subscriber");
    
    info!("Starting TigerWallet Admin Server...");
    
    // Load configuration
    let config = Config::from_env()
        .expect("Failed to load configuration");
    
    info!("Connecting to PostgreSQL: {}", config.database_host);
    
    // Initialize database
    let db = Database::new(&config)
        .await
        .expect("Failed to connect to database");
    
    info!("Database connected successfully");
    
    // Create and run server
    let server = create_server(config, db);
    
    info!("Server started on http://0.0.0.0:8080");
    
    server.await
    
    Ok(())
}
