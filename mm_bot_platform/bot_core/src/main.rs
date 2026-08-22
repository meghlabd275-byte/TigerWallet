//! TigerSwap bot_core - REAL axum HTTP dispatch server (port 8472).
//!
//! bot_api (the Go control plane) calls `http://localhost:8472/dispatch/*`.
//! Secrets arrive already-decrypted in the dispatch request body; this binary
//! only signs and broadcasts real orders. Fail-closed: if PostgreSQL is
//! unavailable, the health/stats endpoints return 503.

use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;

use axum::{
    extract::State,
    http::StatusCode,
    response::{IntoResponse, Json, Response},
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};
use serde_json::json;
use tokio::sync::{watch, RwLock};
use tokio::task::JoinHandle;
use tracing_subscriber::EnvFilter;

use tigerswap_bot_core::cex::{CexClient, CexCredentials, CexExchange, CexOrderRequest};
use tigerswap_bot_core::dex::{self, DexSwapRequest};
use tigerswap_bot_core::store::{PgPool, TradeRecord};
use tigerswap_bot_core::strategies::{
    ArbitrageRunner, MarketMakerRunner, SniperRunner,
};

const PORT: u16 = 8472;

/// A running strategy task: its JoinHandle plus the run-flag sender used to
/// pause/resume/stop it.
struct BotHandle {
    task: JoinHandle<()>,
    run_tx: watch::Sender<bool>,
    kind: BotKind,
}

#[derive(Debug, Clone, Copy)]
enum BotKind {
    MarketMaker,
    Arbitrage,
    Sniper,
}

#[derive(Default)]
struct AppState {
    pool: Option<Arc<PgPool>>,
    bots: RwLock<HashMap<String, BotHandle>>,
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")))
        .init();

    let pg_config = std::env::var("BOT_CORE_PG").unwrap_or_else(|_| {
        "host=localhost port=5432 user=tigerwallet password=tigerwallet dbname=tigerwallet".to_string()
    });
    let pool = PgPool::new(pg_config);
    let pool_state = match pool.ping().await {
        Ok(()) => {
            tracing::info!("postgres connected");
            Some(pool)
        }
        Err(e) => {
            tracing::error!("postgres unavailable (fail-closed on /health and /stats): {e}");
            None
        }
    };

    let state = Arc::new(AppState {
        pool: pool_state,
        bots: RwLock::new(HashMap::new()),
    });

    let app = Router::new()
        .route("/health", get(health))
        .route("/stats", get(stats))
        .route("/dispatch/start", post(dispatch_start))
        .route("/dispatch/stop", post(dispatch_stop))
        .route("/dispatch/pause", post(dispatch_pause))
        .route("/dispatch/resume", post(dispatch_resume))
        .route("/dispatch/execute", post(dispatch_execute))
        .with_state(state);

    let addr = SocketAddr::from(([0, 0, 0, 0], PORT));
    tracing::info!("bot_core dispatch server listening on {addr}");
    let listener = tokio::net::TcpListener::bind(addr).await.expect("bind 8472");
    axum::serve(listener, app).await.expect("server run");
}

// ---------------- Handlers ----------------

async fn health(State(state): State<Arc<AppState>>) -> Response {
    match &state.pool {
        Some(pool) => match pool.ping().await {
            Ok(()) => (StatusCode::OK, Json(json!({"status":"ok","db":"up"}))).into_response(),
            Err(e) => (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({"status":"degraded","db":"down","error":e.to_string()})),
            )
                .into_response(),
        },
        None => (
            StatusCode::SERVICE_UNAVAILABLE,
            Json(json!({"status":"degraded","db":"unconfigured"})),
        )
            .into_response(),
    }
}

async fn stats(State(state): State<Arc<AppState>>) -> Response {
    let pool = match &state.pool {
        Some(p) => p,
        None => {
            return (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({"error":"postgres unavailable"})),
            )
                .into_response()
        }
    };
    match pool.stats().await {
        Ok(s) => (StatusCode::OK, Json(serde_json::to_value(s).unwrap_or_default())).into_response(),
        Err(e) => (
            StatusCode::SERVICE_UNAVAILABLE,
            Json(json!({"error":e.to_string()})),
        )
            .into_response(),
    }
}

// ---------------- Dispatch DTOs ----------------

#[derive(Debug, Deserialize)]
struct StartMarketMaker {
    bot_id: String,
    exchange: CexExchange,
    #[serde(flatten)]
    creds: CexCredentials,
    base_url: Option<String>,
    symbol: String,
    order_size: f64,
    spread_bps: f64,
    poll_interval_ms: Option<u64>,
}

#[derive(Debug, Deserialize)]
struct StartArbitrage {
    bot_id: String,
    dex_req: DexSwapRequest,
    exchange: CexExchange,
    #[serde(flatten)]
    creds: CexCredentials,
    base_url: Option<String>,
    symbol: String,
    threshold_bps: f64,
    poll_interval_ms: Option<u64>,
}

#[derive(Debug, Deserialize)]
struct StartSniper {
    bot_id: String,
    dex_req: DexSwapRequest,
    mempool_url: String,
    poll_interval_ms: Option<u64>,
    min_target_amount: Option<u64>,
}

#[derive(Debug, Deserialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
enum StartReq {
    MarketMaker(StartMarketMaker),
    Arbitrage(StartArbitrage),
    Sniper(StartSniper),
}

#[derive(Debug, Deserialize)]
struct BotIdReq {
    bot_id: String,
}

#[derive(Debug, Serialize)]
struct DispatchAck {
    ok: bool,
    bot_id: String,
    action: &'static str,
}

async fn dispatch_start(
    State(state): State<Arc<AppState>>,
    Json(req): Json<StartReq>,
) -> Response {
    let pool = match &state.pool {
        Some(p) => Arc::clone(p),
        None => {
            return (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({"error":"postgres unavailable"})),
            )
                .into_response()
        }
    };

    let (bot_id, kind, task, run_tx) = match req {
        StartReq::MarketMaker(m) => {
            let (run_tx, run_rx) = watch::channel(true);
            let runner = MarketMakerRunner {
                bot_id: m.bot_id.clone(),
                exchange: m.exchange,
                creds: m.creds,
                base_url: m.base_url,
                symbol: m.symbol,
                order_size: m.order_size,
                spread_bps: m.spread_bps,
                poll_interval_ms: m.poll_interval_ms.unwrap_or(2000),
            };
            let p = Arc::clone(&pool);
            let task = tokio::spawn(async move { runner.run(run_rx, p).await });
            (m.bot_id, BotKind::MarketMaker, task, run_tx)
        }
        StartReq::Arbitrage(a) => {
            let (run_tx, run_rx) = watch::channel(true);
            let runner = ArbitrageRunner {
                bot_id: a.bot_id.clone(),
                dex_req: a.dex_req,
                exchange: a.exchange,
                creds: a.creds,
                base_url: a.base_url,
                symbol: a.symbol,
                threshold_bps: a.threshold_bps,
                poll_interval_ms: a.poll_interval_ms.unwrap_or(5000),
            };
            let p = Arc::clone(&pool);
            let task = tokio::spawn(async move { runner.run(run_rx, p).await });
            (a.bot_id, BotKind::Arbitrage, task, run_tx)
        }
        StartReq::Sniper(s) => {
            let (run_tx, run_rx) = watch::channel(true);
            let runner = SniperRunner {
                bot_id: s.bot_id.clone(),
                dex_req: s.dex_req,
                mempool_url: s.mempool_url,
                poll_interval_ms: s.poll_interval_ms.unwrap_or(1000),
                min_target_amount: s.min_target_amount.unwrap_or(0),
            };
            let p = Arc::clone(&pool);
            let task = tokio::spawn(async move { runner.run(run_rx, p).await });
            (s.bot_id, BotKind::Sniper, task, run_tx)
        }
    };

    let mut bots = state.bots.write().await;
    if let Some(prev) = bots.insert(
        bot_id.clone(),
        BotHandle {
            task,
            run_tx,
            kind,
        },
    ) {
        // Stop the previous instance: drop the sender to end the loop.
        let _ = prev.run_tx.send(false);
        prev.task.abort();
    }
    (StatusCode::OK, Json(DispatchAck { ok: true, bot_id, action: "start" })).into_response()
}

async fn dispatch_stop(
    State(state): State<Arc<AppState>>,
    Json(req): Json<BotIdReq>,
) -> Response {
    let removed = state.bots.write().await.remove(&req.bot_id);
    if let Some(h) = removed {
        let _ = h.run_tx.send(false);
        h.task.abort();
        (
            StatusCode::OK,
            Json(DispatchAck { ok: true, bot_id: req.bot_id, action: "stop" }),
        )
            .into_response()
    } else {
        (
            StatusCode::NOT_FOUND,
            Json(json!({"error":"bot not found","bot_id":req.bot_id})),
        )
            .into_response()
    }
}

async fn dispatch_pause(
    State(state): State<Arc<AppState>>,
    Json(req): Json<BotIdReq>,
) -> Response {
    let bots = state.bots.read().await;
    if let Some(h) = bots.get(&req.bot_id) {
        let _ = h.run_tx.send(false);
        (
            StatusCode::OK,
            Json(DispatchAck { ok: true, bot_id: req.bot_id, action: "pause" }),
        )
            .into_response()
    } else {
        (
            StatusCode::NOT_FOUND,
            Json(json!({"error":"bot not found","bot_id":req.bot_id})),
        )
            .into_response()
    }
}

async fn dispatch_resume(
    State(state): State<Arc<AppState>>,
    Json(req): Json<BotIdReq>,
) -> Response {
    let bots = state.bots.read().await;
    if let Some(h) = bots.get(&req.bot_id) {
        let _ = h.run_tx.send(true);
        (
            StatusCode::OK,
            Json(DispatchAck { ok: true, bot_id: req.bot_id, action: "resume" }),
        )
            .into_response()
    } else {
        (
            StatusCode::NOT_FOUND,
            Json(json!({"error":"bot not found","bot_id":req.bot_id})),
        )
            .into_response()
    }
}

// ---------------- Single trade execution ----------------

#[derive(Debug, Deserialize)]
#[serde(tag = "venue", rename_all = "lowercase")]
enum ExecuteReq {
    Dex(DexSwapRequest),
    Cex {
        exchange: CexExchange,
        #[serde(flatten)]
        creds: CexCredentials,
        base_url: Option<String>,
        #[serde(flatten)]
        order: CexOrderFields,
    },
}

#[derive(Debug, Deserialize)]
struct CexOrderFields {
    symbol: String,
    side: String,
    order_type: String,
    #[serde(default)]
    price: Option<f64>,
    quantity: f64,
}

async fn dispatch_execute(
    State(state): State<Arc<AppState>>,
    Json(req): Json<ExecuteReq>,
) -> Response {
    let pool = match &state.pool {
        Some(p) => Arc::clone(p),
        None => {
            return (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({"error":"postgres unavailable"})),
            )
                .into_response()
        }
    };

    let start = std::time::Instant::now();
    match req {
        ExecuteReq::Dex(d) => match dex::execute_swap(&d).await {
            Ok(r) => {
                let rec = TradeRecord {
                    bot_id: format!("dex:{}", d.router),
                    tx_hash: r.tx_hash.clone(),
                    amount_in: d.amount_in,
                    amount_out: r.amount_out,
                    fee: r.gas_used as f64 * r.gas_price as f64 / 1e18,
                    profit: r.amount_out - d.amount_in,
                    success: r.success,
                    timestamp: chrono::Utc::now(),
                };
                if let Err(e) = pool.insert_trade(&rec).await {
                    tracing::error!("insert_trade: {e}");
                }
                (
                    StatusCode::OK,
                    Json(json!({
                        "ok": true,
                        "venue": "dex",
                        "tx_hash": r.tx_hash,
                        "amount_out": r.amount_out,
                        "gas_used": r.gas_used,
                        "block_number": r.block_number,
                        "latency_us": start.elapsed().as_micros(),
                    })),
                )
                    .into_response()
            }
            Err(e) => (
                StatusCode::BAD_GATEWAY,
                Json(json!({"ok": false, "venue": "dex", "error": e.to_string()})),
            )
                .into_response(),
        },
        ExecuteReq::Cex {
            exchange,
            creds,
            base_url,
            order,
        } => {
            let client = CexClient::new(exchange, creds, base_url);
            let req = CexOrderRequest {
                base_url: None,
                symbol: order.symbol,
                side: order.side,
                order_type: order.order_type,
                price: order.price,
                quantity: order.quantity,
            };
            match client.place_order(&req).await {
                Ok(r) => {
                    let rec = TradeRecord {
                        bot_id: format!("cex:{exchange:?}"),
                        tx_hash: r.order_id.clone(),
                        amount_in: req.quantity,
                        amount_out: req.price.unwrap_or(0.0) * req.quantity,
                        fee: 0.0,
                        profit: 0.0,
                        success: true,
                        timestamp: chrono::Utc::now(),
                    };
                    if let Err(e) = pool.insert_trade(&rec).await {
                        tracing::error!("insert_trade: {e}");
                    }
                    (
                        StatusCode::OK,
                        Json(json!({
                            "ok": true,
                            "venue": "cex",
                            "order_id": r.order_id,
                            "status": r.status,
                            "latency_us": start.elapsed().as_micros(),
                        })),
                    )
                        .into_response()
                }
                Err(e) => (
                    StatusCode::BAD_GATEWAY,
                    Json(json!({"ok": false, "venue": "cex", "error": e.to_string()})),
                )
                    .into_response(),
            }
        }
    }
}
