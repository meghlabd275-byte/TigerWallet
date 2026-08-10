//! TigerWallet White Label Templates
//! High-performance template management system with ultra-low latency

use actix_web::{web, App, HttpResponse, HttpServer, Responder, middleware};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{postgres::PgPoolOptions, Pool, Postgres, Row};
use std::sync::Arc;
use tokio::sync::RwLock;
use uuid::Uuid;

// ============================================================================
// Data Models
// ============================================================================

/// White Label Template
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabelTemplate {
    pub id: String,
    pub name: String,
    pub description: String,
    pub category: TemplateCategory,
    pub template_type: TemplateType,
    pub thumbnail_url: Option<String>,
    pub preview_urls: Vec<String>,
    pub config: TemplateConfig,
    pub features: Vec<String>,
    pub chains_supported: Vec<String>,
    pub pricing: TemplatePricing,
    pub is_featured: bool,
    pub is_premium: bool,
    pub download_count: i64,
    pub rating: f64,
    pub review_count: i64,
    pub status: TemplateStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Template Categories
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TemplateCategory {
    Defi,
    Nft,
    Gaming,
    Enterprise,
    Social,
    Finance,
    Ecommerce,
    Custom,
}

/// Template Types
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TemplateType {
    Starter,
    Professional,
    Enterprise,
    Custom,
}

/// Template Status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TemplateStatus {
    Draft,
    Pending,
    Active,
    Suspended,
    Archived,
}

/// Template Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateConfig {
    pub branding: BrandingConfig,
    pub features: FeatureConfig,
    pub ui_config: UIConfig,
    pub security_config: SecurityConfig,
    pub api_config: APIConfig,
}

/// Branding Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BrandingConfig {
    pub primary_color: String,
    pub secondary_color: String,
    pub accent_color: String,
    pub background_color: String,
    pub text_color: String,
    pub logo_url: Option<String>,
    pub favicon_url: Option<String>,
    pub font_family: Option<String>,
    pub custom_css: Option<String>,
    pub app_name: String,
    pub app_description: Option<String>,
    pub slogan: Option<String>,
}

/// Feature Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureConfig {
    pub swap_enabled: bool,
    pub staking_enabled: bool,
    pub nft_enabled: bool,
    pub bridge_enabled: bool,
    pub defi_enabled: bool,
    pub analytics_enabled: bool,
    pub kyc_enabled: bool,
    pub multi_sig_enabled: bool,
    pub privacy_enabled: bool,
    pub hardware_wallet_enabled: bool,
    pub session_keys_enabled: bool,
    pub paymaster_enabled: bool,
    pub gas_optimization: bool,
    pub mev_protection: bool,
    pub price_alerts: bool,
    pub tax_integration: bool,
}

/// UI Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UIConfig {
    pub theme: String,
    pub language: String,
    pub direction: String,
    pub layout: String,
    pub show_navigation: bool,
    pub show_footer: bool,
    pub custom_pages: Vec<CustomPage>,
    pub widgets: Vec<WidgetConfig>,
}

/// Custom Page
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomPage {
    pub id: String,
    pub title: String,
    pub slug: String,
    pub content: String,
    pub visible: bool,
}

/// Widget Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WidgetConfig {
    pub widget_type: String,
    pub position: String,
    pub config: serde_json::Value,
}

/// Security Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityConfig {
    pub biometric_enabled: bool,
    pub pin_enabled: bool,
    pub two_factor_enabled: bool,
    pub passkey_enabled: bool,
    pub whitelisted_ips: Vec<String>,
    pub max_login_attempts: i32,
    pub session_timeout: i32,
}

/// API Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct APIConfig {
    pub api_key_required: bool,
    pub rate_limit: i32,
    pub webhook_url: Option<String>,
    pub webhook_events: Vec<String>,
}

/// Template Pricing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplatePricing {
    pub price: f64,
    pub currency: String,
    pub billing_cycle: Option<String>,
    pub features_included: Vec<String>,
    pub support_level: String,
}

/// Template Instance (created from template)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateInstance {
    pub id: String,
    pub template_id: String,
    pub white_label_id: String,
    pub name: String,
    pub config: TemplateConfig,
    pub status: InstanceStatus,
    pub deployed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Instance Status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum InstanceStatus {
    Pending,
    Deploying,
    Active,
    Failed,
    Suspended,
}

/// Template Review
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateReview {
    pub id: String,
    pub template_id: String,
    pub user_id: String,
    pub rating: i32,
    pub title: String,
    pub content: String,
    pub pros: Vec<String>,
    pub cons: Vec<String>,
    pub helpful_count: i32,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

// ============================================================================
// Application State
// ============================================================================

pub struct AppState {
    pub db_pool: Pool<Postgres>,
    pub template_cache: Arc<RwLock<std::collections::HashMap<String, WhiteLabelTemplate>>>,
}

// ============================================================================
// Database Functions
// ============================================================================

async fn init_database(database_url: &str) -> Result<Pool<Postgres>, sqlx::Error> {
    let pool = PgPoolOptions::new()
        .max_connections(100)
        .min_connections(10)
        .acquire_timeout(std::time::Duration::from_secs(30))
        .connect(database_url)
        .await?;

    Ok(pool)
}

async fn create_tables(pool: &Pool<Postgres>) -> Result<(), sqlx::Error> {
    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS white_label_templates (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            description TEXT,
            category VARCHAR(50) NOT NULL,
            template_type VARCHAR(50) NOT NULL,
            thumbnail_url TEXT,
            preview_urls JSONB DEFAULT '[]',
            config JSONB NOT NULL,
            features JSONB DEFAULT '[]',
            chains_supported JSONB DEFAULT '[]',
            pricing JSONB NOT NULL,
            is_featured BOOLEAN DEFAULT false,
            is_premium BOOLEAN DEFAULT false,
            download_count BIGINT DEFAULT 0,
            rating DECIMAL(3,2) DEFAULT 0,
            review_count BIGINT DEFAULT 0,
            status VARCHAR(20) DEFAULT 'active',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS template_instances (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            template_id UUID REFERENCES white_label_templates(id),
            white_label_id UUID NOT NULL,
            name VARCHAR(255) NOT NULL,
            config JSONB NOT NULL,
            status VARCHAR(20) DEFAULT 'pending',
            deployed_at TIMESTAMP WITH TIME ZONE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS template_reviews (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            template_id UUID REFERENCES white_label_templates(id),
            user_id UUID NOT NULL,
            rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
            title VARCHAR(255),
            content TEXT,
            pros JSONB DEFAULT '[]',
            cons JSONB DEFAULT '[]',
            helpful_count BIGINT DEFAULT 0,
            status VARCHAR(20) DEFAULT 'pending',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX IF NOT EXISTS idx_templates_category ON white_label_templates(category);
        CREATE INDEX IF NOT EXISTS idx_templates_status ON white_label_templates(status);
        CREATE INDEX IF NOT EXISTS idx_instances_white_label ON template_instances(white_label_id);
        CREATE INDEX IF NOT EXISTS idx_reviews_template ON template_reviews(template_id);
        "#,
    )
    .execute(pool)
    .await?;

    Ok(())
}

// ============================================================================
// API Handlers
// ============================================================================

/// Get all templates
async fn get_templates(
    state: web::Data<AppState>,
    web::Query(params): web::Query<std::collections::HashMap<String, String>>,
) -> impl Responder {
    let category = params.get("category").cloned();
    let template_type = params.get("type").cloned();
    let featured = params.get("featured").cloned();

    let mut query = String::from(
        "SELECT id, name, description, category, template_type, thumbnail_url, preview_urls, 
         config, features, chains_supported, pricing, is_featured, is_premium, 
         download_count, rating, review_count, status, created_at, updated_at 
         FROM white_label_templates WHERE status = 'active'",
    );

    let mut conditions = Vec::new();
    if let Some(cat) = &category {
        conditions.push(format!("category = '{}'", cat));
    }
    if let Some(typ) = &template_type {
        conditions.push(format!("template_type = '{}'", typ));
    }
    if featured == Some("true".to_string()) {
        conditions.push("is_featured = true".to_string());
    }

    if !conditions.is_empty() {
        query.push_str(" AND ");
        query.push_str(&conditions.join(" AND "));
    }

    query.push_str(" ORDER BY download_count DESC, rating DESC");

    let rows = sqlx::query(&query)
        .fetch_all(&state.db_pool)
        .await
        .unwrap_or_default();

    let templates: Vec<WhiteLabelTemplate> = rows
        .iter()
        .map(|row| {
            let config_json: serde_json::Value = row.try_get("config").unwrap_or(serde_json::json!({}));
            let pricing_json: serde_json::Value = row.try_get("pricing").unwrap_or(serde_json::json!({}));
            
            WhiteLabelTemplate {
                id: row.try_get("id").unwrap_or_default(),
                name: row.try_get("name").unwrap_or_default(),
                description: row.try_get("description").unwrap_or_default(),
                category: TemplateCategory::Custom,
                template_type: TemplateType::Starter,
                thumbnail_url: row.try_get("thumbnail_url").unwrap_or_default(),
                preview_urls: vec![],
                config: serde_json::from_value(config_json).unwrap_or(TemplateConfig {
                    branding: BrandingConfig {
                        primary_color: "#000000".to_string(),
                        secondary_color: "#ffffff".to_string(),
                        accent_color: "#000000".to_string(),
                        background_color: "#ffffff".to_string(),
                        text_color: "#000000".to_string(),
                        logo_url: None,
                        favicon_url: None,
                        font_family: None,
                        custom_css: None,
                        app_name: "TigerWallet".to_string(),
                        app_description: None,
                        slogan: None,
                    },
                    features: FeatureConfig {
                        swap_enabled: true,
                        staking_enabled: true,
                        nft_enabled: true,
                        bridge_enabled: true,
                        defi_enabled: true,
                        analytics_enabled: true,
                        kyc_enabled: false,
                        multi_sig_enabled: false,
                        privacy_enabled: false,
                        hardware_wallet_enabled: true,
                        session_keys_enabled: true,
                        paymaster_enabled: false,
                        gas_optimization: true,
                        mev_protection: false,
                        price_alerts: true,
                        tax_integration: false,
                    },
                    ui_config: UIConfig {
                        theme: "light".to_string(),
                        language: "en".to_string(),
                        direction: "ltr".to_string(),
                        layout: "default".to_string(),
                        show_navigation: true,
                        show_footer: true,
                        custom_pages: vec![],
                        widgets: vec![],
                    },
                    security_config: SecurityConfig {
                        biometric_enabled: true,
                        pin_enabled: true,
                        two_factor_enabled: false,
                        passkey_enabled: false,
                        whitelisted_ips: vec![],
                        max_login_attempts: 5,
                        session_timeout: 3600,
                    },
                    api_config: APIConfig {
                        api_key_required: true,
                        rate_limit: 1000,
                        webhook_url: None,
                        webhook_events: vec![],
                    },
                }),
                features: vec![],
                chains_supported: vec![],
                pricing: serde_json::from_value(pricing_json).unwrap_or(TemplatePricing {
                    price: 0.0,
                    currency: "USD".to_string(),
                    billing_cycle: None,
                    features_included: vec![],
                    support_level: "standard".to_string(),
                }),
                is_featured: row.try_get("is_featured").unwrap_or_default(),
                is_premium: row.try_get("is_premium").unwrap_or_default(),
                download_count: row.try_get("download_count").unwrap_or_default(),
                rating: row.try_get("rating").unwrap_or(0.0),
                review_count: row.try_get("review_count").unwrap_or_default(),
                status: TemplateStatus::Active,
                created_at: Utc::now(),
                updated_at: Utc::now(),
            }
        })
        .collect();

    HttpResponse::Ok().json(serde_json::json!({ "templates": templates }))
}

/// Get template by ID
async fn get_template(state: web::Data<AppState>, path: web::Path<String>) -> impl Responder {
    let id = path.into_inner();

    let row = sqlx::query(
        "SELECT id, name, description, category, template_type, thumbnail_url, preview_urls, 
         config, features, chains_supported, pricing, is_featured, is_premium, 
         download_count, rating, review_count, status, created_at, updated_at 
         FROM white_label_templates WHERE id = $1",
    )
    .bind(&id)
    .fetch_optional(&state.db_pool)
    .await;

    match row {
        Ok(Some(row)) => {
            let config_json: serde_json::Value = row.try_get("config").unwrap_or(serde_json::json!({}));
            let pricing_json: serde_json::Value = row.try_get("pricing").unwrap_or(serde_json::json!({}));
            
            let template = WhiteLabelTemplate {
                id: row.try_get("id").unwrap_or_default(),
                name: row.try_get("name").unwrap_or_default(),
                description: row.try_get("description").unwrap_or_default(),
                category: TemplateCategory::Custom,
                template_type: TemplateType::Starter,
                thumbnail_url: row.try_get("thumbnail_url").unwrap_or_default(),
                preview_urls: vec![],
                config: serde_json::from_value(config_json).unwrap_or(TemplateConfig {
                    branding: BrandingConfig {
                        primary_color: "#000000".to_string(),
                        secondary_color: "#ffffff".to_string(),
                        accent_color: "#000000".to_string(),
                        background_color: "#ffffff".to_string(),
                        text_color: "#000000".to_string(),
                        logo_url: None,
                        favicon_url: None,
                        font_family: None,
                        custom_css: None,
                        app_name: "TigerWallet".to_string(),
                        app_description: None,
                        slogan: None,
                    },
                    features: FeatureConfig {
                        swap_enabled: true,
                        staking_enabled: true,
                        nft_enabled: true,
                        bridge_enabled: true,
                        defi_enabled: true,
                        analytics_enabled: true,
                        kyc_enabled: false,
                        multi_sig_enabled: false,
                        privacy_enabled: false,
                        hardware_wallet_enabled: true,
                        session_keys_enabled: true,
                        paymaster_enabled: false,
                        gas_optimization: true,
                        mev_protection: false,
                        price_alerts: true,
                        tax_integration: false,
                    },
                    ui_config: UIConfig {
                        theme: "light".to_string(),
                        language: "en".to_string(),
                        direction: "ltr".to_string(),
                        layout: "default".to_string(),
                        show_navigation: true,
                        show_footer: true,
                        custom_pages: vec![],
                        widgets: vec![],
                    },
                    security_config: SecurityConfig {
                        biometric_enabled: true,
                        pin_enabled: true,
                        two_factor_enabled: false,
                        passkey_enabled: false,
                        whitelisted_ips: vec![],
                        max_login_attempts: 5,
                        session_timeout: 3600,
                    },
                    api_config: APIConfig {
                        api_key_required: true,
                        rate_limit: 1000,
                        webhook_url: None,
                        webhook_events: vec![],
                    },
                }),
                features: vec![],
                chains_supported: vec![],
                pricing: serde_json::from_value(pricing_json).unwrap_or(TemplatePricing {
                    price: 0.0,
                    currency: "USD".to_string(),
                    billing_cycle: None,
                    features_included: vec![],
                    support_level: "standard".to_string(),
                }),
                is_featured: row.try_get("is_featured").unwrap_or_default(),
                is_premium: row.try_get("is_premium").unwrap_or_default(),
                download_count: row.try_get("download_count").unwrap_or_default(),
                rating: row.try_get("rating").unwrap_or(0.0),
                review_count: row.try_get("review_count").unwrap_or_default(),
                status: TemplateStatus::Active,
                created_at: Utc::now(),
                updated_at: Utc::now(),
            };

            HttpResponse::Ok().json(serde_json::json!({ "template": template }))
        }
        Ok(None) => HttpResponse::NotFound().json(serde_json::json!({ "error": "Template not found" })),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({ "error": e.to_string() })),
    }
}

/// Create template
async fn create_template(
    state: web::Data<AppState>,
    payload: web::Json<WhiteLabelTemplate>,
) -> impl Responder {
    let template = payload.into_inner();
    let id = Uuid::new_v4().to_string();
    let now = Utc::now();

    let config_json = serde_json::to_value(&template.config).unwrap_or(serde_json::json!({}));
    let pricing_json = serde_json::to_value(&template.pricing).unwrap_or(serde_json::json!({}));

    let result = sqlx::query(
        r#"
        INSERT INTO white_label_templates (
            id, name, description, category, template_type, thumbnail_url, preview_urls,
            config, features, chains_supported, pricing, is_featured, is_premium,
            download_count, rating, review_count, status, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
        "#,
    )
    .bind(&id)
    .bind(&template.name)
    .bind(&template.description)
    .bind(format!("{:?}", template.category).to_lowercase())
    .bind(format!("{:?}", template.template_type).to_lowercase())
    .bind(&template.thumbnail_url)
    .bind(serde_json::to_string(&template.preview_urls).unwrap_or_default())
    .bind(&config_json)
    .bind(serde_json::to_string(&template.features).unwrap_or_default())
    .bind(serde_json::to_string(&template.chains_supported).unwrap_or_default())
    .bind(&pricing_json)
    .bind(template.is_featured)
    .bind(template.is_premium)
    .bind(template.download_count)
    .bind(template.rating)
    .bind(template.review_count)
    .bind(format!("{:?}", template.status).to_lowercase())
    .bind(now)
    .bind(now)
    .execute(&state.db_pool)
    .await;

    match result {
        Ok(_) => {
            let mut cache = state.template_cache.write().await;
            let mut new_template = template;
            new_template.id = id.clone();
            cache.insert(id.clone(), new_template);

            HttpResponse::Created().json(serde_json::json!({ 
                "template_id": id,
                "message": "Template created successfully" 
            }))
        }
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({ "error": e.to_string() })),
    }
}

/// Deploy template instance
async fn deploy_template(
    state: web::Data<AppState>,
    payload: web::Json<TemplateInstance>,
) -> impl Responder {
    let instance = payload.into_inner();
    let id = Uuid::new_v4().to_string();
    let now = Utc::now();

    let config_json = serde_json::to_value(&instance.config).unwrap_or(serde_json::json!({}));

    let result = sqlx::query(
        r#"
        INSERT INTO template_instances (
            id, template_id, white_label_id, name, config, status, deployed_at, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        "#,
    )
    .bind(&id)
    .bind(&instance.template_id)
    .bind(&instance.white_label_id)
    .bind(&instance.name)
    .bind(&config_json)
    .bind("deploying")
    .bind(now)
    .bind(now)
    .bind(now)
    .execute(&state.db_pool)
    .await;

    match result {
        Ok(_) => HttpResponse::Created().json(serde_json::json!({
            "instance_id": id,
            "message": "Template deployment initiated"
        })),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({ "error": e.to_string() })),
    }
}

/// Get template reviews
async fn get_template_reviews(
    state: web::Data<AppState>,
    path: web::Path<String>,
) -> impl Responder {
    let template_id = path.into_inner();

    let rows = sqlx::query(
        "SELECT id, template_id, user_id, rating, title, content, pros, cons, 
         helpful_count, status, created_at FROM template_reviews 
         WHERE template_id = $1 AND status = 'approved' ORDER BY helpful_count DESC",
    )
    .bind(&template_id)
    .fetch_all(&state.db_pool)
    .await
    .unwrap_or_default();

    let reviews: Vec<TemplateReview> = rows
        .iter()
        .map(|row| TemplateReview {
            id: row.try_get("id").unwrap_or_default(),
            template_id: row.try_get("template_id").unwrap_or_default(),
            user_id: row.try_get("user_id").unwrap_or_default(),
            rating: row.try_get("rating").unwrap_or_default(),
            title: row.try_get("title").unwrap_or_default(),
            content: row.try_get("content").unwrap_or_default(),
            pros: vec![],
            cons: vec![],
            helpful_count: row.try_get("helpful_count").unwrap_or_default(),
            status: row.try_get("status").unwrap_or_default(),
            created_at: Utc::now(),
        })
        .collect();

    HttpResponse::Ok().json(serde_json::json!({ "reviews": reviews }))
}

/// Get template categories
async fn get_categories() -> impl Responder {
    let categories = vec![
        serde_json::json!({ "id": "defi", "name": "DeFi", "count": 0 }),
        serde_json::json!({ "id": "nft", "name": "NFT", "count": 0 }),
        serde_json::json!({ "id": "gaming", "name": "Gaming", "count": 0 }),
        serde_json::json!({ "id": "enterprise", "name": "Enterprise", "count": 0 }),
        serde_json::json!({ "id": "social", "name": "Social", "count": 0 }),
        serde_json::json!({ "id": "finance", "name": "Finance", "count": 0 }),
        serde_json::json!({ "id": "ecommerce", "name": "Ecommerce", "count": 0 }),
        serde_json::json!({ "id": "custom", "name": "Custom", "count": 0 }),
    ];

    HttpResponse::Ok().json(serde_json::json!({ "categories": categories }))
}

/// Health check
async fn health() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "service": "white_label_templates",
        "timestamp": Utc::now().to_rfc3339()
    }))
}

// ============================================================================
// Main Entry Point
// ============================================================================

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    // Initialize logging
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .init();

    tracing::info!("Starting TigerWallet White Label Templates Service");

    // Database configuration
    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet".to_string());

    // Initialize database connection pool
    let db_pool = init_database(&database_url)
        .await
        .expect("Failed to connect to database");

    // Create tables
    create_tables(&db_pool)
        .await
        .expect("Failed to create tables");

    tracing::info!("Database initialized successfully");

    // Create application state
    let state = web::Data::new(AppState {
        db_pool,
        template_cache: Arc::new(RwLock::new(std::collections::HashMap::new())),
    });

    // Start HTTP server
    HttpServer::new(move || {
        App::new()
            .app_data(state.clone())
            .wrap(middleware::DefaultHeaders::new().header("X-Version", "1.0.0"))
            .wrap(middleware::Compress::default())
            .route("/health", web::get().to(health))
            .route("/api/v1/templates", web::get().to(get_templates))
            .route("/api/v1/templates", web::post().to(create_template))
            .route("/api/v1/templates/{id}", web::get().to(get_template))
            .route("/api/v1/templates/{id}/deploy", web::post().to(deploy_template))
            .route("/api/v1/templates/{id}/reviews", web::get().to(get_template_reviews))
            .route("/api/v1/templates/categories", web::get().to(get_categories))
    })
    .bind(("0.0.0.0", 8086))?
    .run()
    .await
}
