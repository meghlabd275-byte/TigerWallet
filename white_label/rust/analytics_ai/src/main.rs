//! TigerWallet White Label Advanced Analytics with AI — binary entry point

use actix_web::{web, App, HttpServer, middleware};
use std::sync::Arc;
use std::collections::HashMap;
use tokio::sync::RwLock;
use white_label_analytics_ai::{
    AppState, health, get_time_series, get_prediction, detect_anomalies_handler,
    get_user_behavior, get_revenue_analytics, get_dashboard_widgets,
    get_user_risk_score, create_report, get_analytics_summary,
    init_database, create_analytics_schema,
};

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .init();

    tracing::info!("Starting TigerWallet White Label Analytics AI");

    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet".to_string());

    let db_pool = init_database(&database_url)
        .await
        .expect("Failed to connect to database");

    create_analytics_schema(&db_pool)
        .await
        .expect("Failed to create schema");

    let redis_url = std::env::var("REDIS_URL")
        .unwrap_or_else(|_| "redis://localhost:6379".to_string());

    let redis_client = redis::Client::open(redis_url.as_str())
        .expect("Failed to connect to Redis");

    let state = web::Data::new(AppState {
        db_pool,
        redis_client: Arc::new(redis_client),
        model_cache: Arc::new(RwLock::new(HashMap::new())),
    });

    HttpServer::new(move || {
        App::new()
            .app_data(state.clone())
            .wrap(middleware::DefaultHeaders::new().header("X-Version", "1.0.0"))
            .wrap(middleware::Compress::default())
            .route("/health", web::get().to(health))
            .route("/api/v1/analytics/timeseries", web::get().to(get_time_series))
            .route("/api/v1/analytics/prediction", web::get().to(get_prediction))
            .route("/api/v1/analytics/anomalies", web::get().to(detect_anomalies_handler))
            .route("/api/v1/analytics/user-behavior", web::get().to(get_user_behavior))
            .route("/api/v1/analytics/revenue", web::get().to(get_revenue_analytics))
            .route("/api/v1/analytics/dashboard", web::get().to(get_dashboard_widgets))
            .route("/api/v1/analytics/risk/{userId}", web::get().to(get_user_risk_score))
            .route("/api/v1/analytics/reports", web::post().to(create_report))
            .route("/api/v1/analytics/summary", web::get().to(get_analytics_summary))
    })
    .bind(("0.0.0.0", 8089))?
    .run()
    .await
}
