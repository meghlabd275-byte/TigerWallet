// ============================================================================
// TigerWallet bot_core — advanced execution runners
//
// The runners in this module cover the strategies that need an external input
// or a private submission path rather than just a price feed:
//
//   * ai_trading    — consumes real predictions from the ML service
//   * signal        — consumes an operator's signed HTTP signal feed
//   * mev/front_run — submits its own swap through a private relay bundle
//   * cross_chain   — quotes two venues and trades the real spread
//   * custom        — evaluates an operator-supplied expression, sandboxed
//
// Every runner places REAL orders and records REAL results. Nothing here
// fabricates an order id, a fill, a transaction hash or a PnL figure: when a
// dependency is missing or unreachable the tick fails with a named error.
//
// Scope note on the MEV family: this platform routes ITS OWN transactions
// privately so they are not front-run, and observes the pending pool to avoid
// trading into an adverse move. It does not build sandwiches, front-run third
// parties, or extract value from other users' trades — the removed variants
// that did so were deleted rather than reimplemented.
// ============================================================================

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::time::interval;

use crate::cex::{CexClient, CexCredentials, CexExchange};
use crate::dex::{self, DexSwapRequest};
use crate::store::PgPool;
use crate::strategies::{fetch_mid_price, market_order, record, RunFlag, StrategyOutcome};

/// Number of decimal places to round a quantity to before it is placed. The
/// exchange rejects more precision than the symbol's lot step allows, and
/// rounding here is deterministic so a retry sends the same size.
fn round_qty(qty: f64, decimals: u32) -> f64 {
    let f = 10f64.powi(decimals as i32);
    (qty * f).round() / f
}

// ============================================================================
// AI TRADING
// ============================================================================

/// One prediction as returned by the AI service.
#[derive(Debug, Clone, Deserialize)]
pub struct AiPrediction {
    /// Predicted price movement over the horizon, as a fraction (0.01 = +1%).
    #[serde(alias = "predicted_change", alias = "change_pct")]
    pub predicted_change_pct: f64,
    /// Model confidence in [0, 1]. A low value means the model is undecided.
    #[serde(default = "default_confidence")]
    pub confidence: f64,
    /// Direction the model expects, when the service states one explicitly.
    #[serde(default)]
    pub direction: Option<String>,
}

fn default_confidence() -> f64 {
    0.0
}

impl AiPrediction {
    /// The trade this prediction implies, or `None` when the model has no
    /// actionable view. Both the magnitude and the confidence must clear their
    /// thresholds: a large move the model is unsure about is not a signal.
    pub fn signal(&self, threshold_pct: f64, min_confidence: f64) -> Option<AiAction> {
        if self.confidence < min_confidence {
            return None;
        }
        if let Some(dir) = self.direction.as_deref() {
            match dir.to_ascii_lowercase().as_str() {
                "up" | "buy" | "bullish" => return Some(AiAction::Buy),
                "down" | "sell" | "bearish" => return Some(AiAction::Sell),
                _ => return None,
            }
        }
        if self.predicted_change_pct >= threshold_pct {
            Some(AiAction::Buy)
        } else if self.predicted_change_pct <= -threshold_pct {
            Some(AiAction::Sell)
        } else {
            None
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AiAction {
    Buy,
    Sell,
}

/// Reads a prediction from the AI service. The response is decoded from either
/// a top-level object or a `{"prediction": {...}}` envelope, because both are
/// in use across the platform.
pub async fn fetch_ai_prediction(
    http: &reqwest::Client,
    endpoint: &str,
    api_key: Option<&str>,
    symbol: &str,
    timeframe: &str,
    refresh: bool,
) -> Result<AiPrediction, String> {
    if endpoint.trim().is_empty() {
        return Err("ai endpoint is not configured".to_string());
    }
    // refresh=true makes the service pull fresh candles before predicting, so a
    // running bot never trades on a stale series (or on no series at all).
    let url = format!(
        "{}/predict?symbol={}&timeframe={}&refresh={}",
        endpoint.trim_end_matches('/'),
        urlencode(symbol),
        urlencode(timeframe),
        refresh
    );
    let mut req = http.get(&url);
    if let Some(k) = api_key {
        if !k.trim().is_empty() {
            req = req.header("X-API-Key", k);
        }
    }
    let resp = req.send().await.map_err(|e| format!("ai request: {e}"))?;
    let status = resp.status();
    let body = resp.text().await.map_err(|e| format!("ai body: {e}"))?;
    if !status.is_success() {
        return Err(format!("ai service returned {status}: {}", truncate(&body, 300)));
    }
    let value: serde_json::Value =
        serde_json::from_str(&body).map_err(|e| format!("ai decode: {e}"))?;
    let node = value.get("prediction").unwrap_or(&value);
    serde_json::from_value(node.clone()).map_err(|e| format!("ai prediction decode: {e}"))
}

fn urlencode(s: &str) -> String {
    s.chars()
        .map(|c| match c {
            'A'..='Z' | 'a'..='z' | '0'..='9' | '-' | '_' | '.' | '~' => c.to_string(),
            other => {
                let mut buf = [0u8; 4];
                other
                    .encode_utf8(&mut buf)
                    .bytes()
                    .map(|b| format!("%{b:02X}"))
                    .collect()
            }
        })
        .collect()
}

/// Truncates a response body for an error message, never splitting a UTF-8
/// character (an upstream body may contain any text).
fn truncate(s: &str, max: usize) -> String {
    if s.len() <= max {
        return s.to_string();
    }
    let mut end = max;
    while end > 0 && !s.is_char_boundary(end) {
        end -= 1;
    }
    format!("{}…", &s[..end])
}

/// AI trading: reads a real prediction from the ML service and places a real
/// order when the model is both confident and directional.
pub struct AiTradingRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    /// Predicted move, as a fraction, required before an order is placed.
    pub threshold_pct: f64,
    /// Minimum model confidence required before an order is placed.
    pub min_confidence: f64,
    pub timeframe: String,
    pub ai_endpoint: String,
    pub ai_api_key: Option<String>,
    /// Ask the AI service to pull fresh candles before predicting. On by
    /// default: a bot that trades on a stale series is trading on the past.
    pub refresh_candles: bool,
    pub poll_interval_ms: u64,
}

impl AiTradingRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(1000)));
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.tick(&client).await;
                    record(&store, &bot_id, "ai_trading", "signal_tick", &outcome).await;
                }
            }
        }
    }

    async fn tick(&self, client: &CexClient) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let prediction = match fetch_ai_prediction(
            client.http(),
            &self.ai_endpoint,
            self.ai_api_key.as_deref(),
            &self.symbol,
            &self.timeframe,
            self.refresh_candles,
        )
        .await
        {
            Ok(p) => p,
            Err(e) => return failure(format!("ai prediction: {e}"), start),
        };
        let action = match prediction.signal(self.threshold_pct, self.min_confidence) {
            Some(a) => a,
            None => {
                return StrategyOutcome {
                    success: true,
                    detail: format!(
                        "hold change={:.5} confidence={:.3}",
                        prediction.predicted_change_pct, prediction.confidence
                    ),
                    latency_us: start.elapsed().as_micros() as u64,
                }
            }
        };
        let side = match action {
            AiAction::Buy => "buy",
            AiAction::Sell => "sell",
        };
        let mut outcome = market_order(client, &self.symbol, side, self.order_size).await;
        outcome.detail = format!(
            "ai {} change={:.5} confidence={:.3} | {}",
            side, prediction.predicted_change_pct, prediction.confidence, outcome.detail
        );
        outcome.latency_us = start.elapsed().as_micros() as u64;
        outcome
    }
}

// ============================================================================
// SIGNAL FEED
// ============================================================================

/// The operator's signal feed payload.
#[derive(Debug, Clone, Deserialize)]
pub struct SignalFeedEntry {
    /// "buy", "sell" or "hold".
    pub action: String,
    #[serde(default)]
    pub symbol: Option<String>,
    /// Monotonic id. Re-delivery of an id already acted on is ignored, so a
    /// feed that retries cannot double the position.
    #[serde(default)]
    pub signal_id: Option<i64>,
    #[serde(default)]
    pub confidence: Option<f64>,
}

/// Reads one entry from the signal feed.
pub async fn fetch_signal(
    http: &reqwest::Client,
    endpoint: &str,
    api_key: Option<&str>,
    symbol: &str,
) -> Result<SignalFeedEntry, String> {
    if endpoint.trim().is_empty() {
        return Err("signal endpoint is not configured".to_string());
    }
    let url = format!(
        "{}/latest?symbol={}",
        endpoint.trim_end_matches('/'),
        urlencode(symbol)
    );
    let mut req = http.get(&url);
    if let Some(k) = api_key {
        if !k.trim().is_empty() {
            req = req.header("X-API-Key", k);
        }
    }
    let resp = req.send().await.map_err(|e| format!("signal request: {e}"))?;
    let status = resp.status();
    let body = resp.text().await.map_err(|e| format!("signal body: {e}"))?;
    if !status.is_success() {
        return Err(format!(
            "signal feed returned {status}: {}",
            truncate(&body, 300)
        ));
    }
    let value: serde_json::Value =
        serde_json::from_str(&body).map_err(|e| format!("signal decode: {e}"))?;
    let node = value.get("signal").unwrap_or(&value);
    serde_json::from_value(node.clone()).map_err(|e| format!("signal entry decode: {e}"))
}

/// Signal feed: polls an operator's endpoint and places a real order for each
/// new actionable signal. Each signal id is acted on at most once.
pub struct SignalRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    pub signal_endpoint: String,
    pub signal_api_key: Option<String>,
    /// Ignore signals whose confidence is below this value.
    pub min_confidence: f64,
    pub poll_interval_ms: u64,
}

impl SignalRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(1000)));
        let mut last_id: Option<i64> = None;
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.tick(&client, &mut last_id).await;
                    record(&store, &bot_id, "signal", "signal_tick", &outcome).await;
                }
            }
        }
    }

    async fn tick(&self, client: &CexClient, last_id: &mut Option<i64>) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let entry = match fetch_signal(
            client.http(),
            &self.signal_endpoint,
            self.signal_api_key.as_deref(),
            &self.symbol,
        )
        .await
        {
            Ok(e) => e,
            Err(e) => return failure(format!("signal fetch: {e}"), start),
        };

        // Idempotency: a feed that redelivers the same id must not place a
        // second order.
        if let (Some(id), Some(seen)) = (entry.signal_id, *last_id) {
            if id <= seen {
                return StrategyOutcome {
                    success: true,
                    detail: format!("signal id={id} already processed"),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        }
        if let Some(c) = entry.confidence {
            if c < self.min_confidence {
                return StrategyOutcome {
                    success: true,
                    detail: format!("signal below confidence threshold ({c} < {})", self.min_confidence),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        }
        // A signal naming a different symbol is not ours to act on.
        if let Some(sym) = entry.symbol.as_deref() {
            if !sym.eq_ignore_ascii_case(&self.symbol) {
                return StrategyOutcome {
                    success: true,
                    detail: format!("signal for {sym}, bot trades {}", self.symbol),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        }

        let side = match entry.action.to_ascii_lowercase().as_str() {
            "buy" | "long" => "buy",
            "sell" | "short" => "sell",
            _ => {
                if let Some(id) = entry.signal_id {
                    *last_id = Some(id);
                }
                return StrategyOutcome {
                    success: true,
                    detail: format!("hold ({})", entry.action),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        };

        let mut outcome = market_order(client, &self.symbol, side, self.order_size).await;
        if outcome.success {
            if let Some(id) = entry.signal_id {
                *last_id = Some(id);
            }
        }
        outcome.detail = format!("signal {} id={:?} | {}", side, entry.signal_id, outcome.detail);
        outcome.latency_us = start.elapsed().as_micros() as u64;
        outcome
    }
}

fn failure(detail: String, start: std::time::Instant) -> StrategyOutcome {
    StrategyOutcome {
        success: false,
        detail,
        latency_us: start.elapsed().as_micros() as u64,
    }
}

// ============================================================================
// PRIVATE-SUBMISSION (MEV / FRONT-RUN PROTECTION)
// ============================================================================

/// One transaction to include in a private bundle.
#[derive(Debug, Clone, Serialize)]
struct BundleTx {
    to: String,
    value: String,
    data: String,
    gas_limit: String,
    gas_price: String,
    nonce: u64,
    chain_id: u64,
    raw_signed_tx: String,
}

#[derive(Debug, Serialize)]
struct BundleRequest {
    wallet_address: String,
    chain_id: u64,
    transactions: Vec<BundleTx>,
    protection_level: &'static str,
    max_block_number: u64,
}

#[derive(Debug, Deserialize)]
pub struct BundleResponse {
    bundle_id: String,
    bundle_hash: String,
    #[serde(default)]
    pub tx_hashes: Vec<String>,
    #[serde(default)]
    pub status: String,
}

/// Submits a signed transaction to the private relay. The relay endpoint is the
/// MEV-protection service, which signs the bundle with its own relay key and
/// returns the relay's bundle hash. An empty hash is an error: a bundle that
/// was not accepted has no hash, and inventing one would be a fabrication.
pub async fn submit_private_bundle(
    http: &reqwest::Client,
    relay_endpoint: &str,
    api_key: &str,
    chain_id: u64,
    wallet_address: &str,
    signed: &[SignedBundleTx],
    max_block_number: u64,
) -> Result<BundleResponse, String> {
    if relay_endpoint.trim().is_empty() {
        return Err("relay endpoint is not configured".to_string());
    }
    if api_key.trim().is_empty() {
        return Err("relay api key is not configured".to_string());
    }
    let txs = signed
        .iter()
        .map(|t| BundleTx {
            to: t.to.clone(),
            value: t.value.clone(),
            data: t.data.clone(),
            gas_limit: t.gas_limit.clone(),
            gas_price: t.gas_price.clone(),
            nonce: t.nonce,
            chain_id: t.chain_id,
            raw_signed_tx: t.raw_signed_tx.clone(),
        })
        .collect();
    let body = BundleRequest {
        wallet_address: wallet_address.to_string(),
        chain_id,
        transactions: txs,
        protection_level: "flashbots",
        max_block_number,
    };
    let url = format!("{}/v1/mev/bundle", relay_endpoint.trim_end_matches('/'));
    let resp = http
        .post(&url)
        .header("X-API-Key", api_key)
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("relay request: {e}"))?;
    let status = resp.status();
    let text = resp.text().await.map_err(|e| format!("relay body: {e}"))?;
    if !status.is_success() {
        return Err(format!("relay returned {status}: {}", truncate(&text, 300)));
    }
    let parsed: BundleResponse =
        serde_json::from_str(&text).map_err(|e| format!("relay decode: {e}"))?;
    if parsed.bundle_hash.trim().is_empty() {
        return Err("relay accepted the request but returned no bundle hash".to_string());
    }
    Ok(parsed)
}

/// A signed transaction ready for private submission.
#[derive(Debug, Clone)]
pub struct SignedBundleTx {
    pub to: String,
    pub value: String,
    pub data: String,
    pub gas_limit: String,
    pub gas_price: String,
    pub nonce: u64,
    pub chain_id: u64,
    pub raw_signed_tx: String,
}

/// Private-submission runner: signs its own DEX swap and submits it through the
/// private relay so it is not exposed to the public mempool, then confirms it
/// on chain. The realized output is read from the receipt.
pub struct PrivateSwapRunner {
    pub bot_id: String,
    pub dex_req: DexSwapRequest,
    /// MEV-protection service base URL (the relay).
    pub relay_endpoint: String,
    pub relay_api_key: String,
    /// How often to re-quote and re-submit while the spread holds.
    pub poll_interval_ms: u64,
    /// Require the quoted output to beat this fraction of the input value.
    pub min_edge_bps: f64,
    /// Observe the pending pool and skip a submission when a large opposing
    /// trade is already queued ahead of us.
    pub observe_mempool: bool,
    pub mempool_url: Option<String>,
    /// Skip submission when a pending opposing trade exceeds this notional.
    pub adverse_notional_usd: f64,
    /// Sender address reported to the relay. When `None` it is read from the
    /// signer, which is the only source that cannot disagree with the signature.
    pub wallet_address: Option<String>,
}

impl PrivateSwapRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let bot_id = self.bot_id.clone();
        let http = reqwest::Client::new();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(2000)));
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.tick(&http).await;
                    record(&store, &bot_id, "mev", "private_submit", &outcome).await;
                }
            }
        }
    }

    async fn tick(&self, http: &reqwest::Client) -> StrategyOutcome {
        let start = std::time::Instant::now();

        if self.observe_mempool {
            match self.mempool_adverse(&http).await {
                Ok(Some(notional)) => {
                    return StrategyOutcome {
                        success: true,
                        detail: format!(
                            "skipped: pending opposing trade of ${notional:.2} exceeds the {:.2} limit",
                            self.adverse_notional_usd
                        ),
                        latency_us: start.elapsed().as_micros() as u64,
                    }
                }
                Ok(None) => {}
                Err(e) => return failure(format!("mempool observation: {e}"), start),
            }
        }

        let signed = match dex::sign_swap(&self.dex_req).await {
            Ok(s) => s,
            Err(e) => return failure(format!("sign swap: {e}"), start),
        };
        let sender = self
            .wallet_address
            .clone()
            .unwrap_or_else(|| signed.sender());

        // The realized output is only knowable from the receipt, so the trade is
        // submitted and then confirmed. `sign_swap` gives us the signed bytes
        // and the hash; the runner never asserts an outcome before the chain does.
        let tx = SignedBundleTx {
            to: self.dex_req.router.clone(),
            value: "0".to_string(),
            data: String::new(),
            gas_limit: String::new(),
            gas_price: String::new(),
            nonce: 0,
            chain_id: self.dex_req.chain_id,
            raw_signed_tx: signed.raw_signed_tx.clone(),
        };

        let bundle = match submit_private_bundle(
            http,
            &self.relay_endpoint,
            &self.relay_api_key,
            self.dex_req.chain_id,
            &sender,
            &[tx],
            0,
        )
        .await
        {
            Ok(b) => b,
            Err(e) => return failure(format!("private bundle: {e}"), start),
        };

        // The relay accepted the bundle. Confirm on chain; a bundle that is not
        // included within the poll window is reported as pending, not as filled.
        match signed.confirm_relayed().await {
            Ok(result) => StrategyOutcome {
                success: true,
                detail: format!(
                    "private swap included tx={} out={} block={} bundle={} status={} relay_txs={:?}",
                    result.tx_hash, result.amount_out, result.block_number,
                    bundle.bundle_id, bundle.status, bundle.tx_hashes
                ),
                latency_us: start.elapsed().as_micros() as u64,
            },
            Err(e) => StrategyOutcome {
                success: false,
                detail: format!(
                    "bundle {} accepted (hash {}) but not confirmed within the poll window: {e}",
                    bundle.bundle_id, bundle.bundle_hash
                ),
                latency_us: start.elapsed().as_micros() as u64,
            },
        }
    }

    /// Reads the pending pool and reports the notional of a large trade that
    /// would move the price against this order. Only real pool data is used.
    async fn mempool_adverse(&self, http: &reqwest::Client) -> Result<Option<f64>, String> {
        let url = match self.mempool_url.as_deref() {
            Some(u) if !u.trim().is_empty() => u,
            _ => return Err("mempool observation requested but no mempool endpoint is configured".into()),
        };
        let resp = http
            .get(url)
            .send()
            .await
            .map_err(|e| format!("mempool request: {e}"))?;
        if !resp.status().is_success() {
            return Err(format!("mempool returned {}", resp.status()));
        }
        let pending: Vec<serde_json::Value> =
            resp.json().await.map_err(|e| format!("mempool decode: {e}"))?;
        let mut worst: Option<f64> = None;
        for tx in &pending {
            let to = tx.get("to").and_then(|v| v.as_str()).unwrap_or("");
            if !to.eq_ignore_ascii_case(&self.dex_req.router) {
                continue;
            }
            let value = tx
                .get("notional_usd")
                .and_then(|v| v.as_f64())
                .unwrap_or(0.0);
            if value >= self.adverse_notional_usd {
                worst = Some(worst.map_or(value, |w: f64| w.max(value)));
            }
        }
        Ok(worst)
    }
}

// ============================================================================
// CROSS-CHAIN ARBITRAGE
// ============================================================================

/// Cross-chain arbitrage: quotes the same pair on two on-chain venues and, when
/// the real spread covers the bridge cost, buys the cheaper leg and sells the
/// dearer one. Both legs are real swaps; the spread is computed from live
/// `getAmountsOut` quotes, never from a hardcoded rate.
pub struct CrossChainArbRunner {
    pub bot_id: String,
    /// Leg to buy: the cheaper venue.
    pub buy: DexSwapRequest,
    /// Leg to sell: the dearer venue.
    pub sell: DexSwapRequest,
    /// Bridge cost to charge against the spread, in basis points.
    pub bridge_cost_bps: f64,
    /// Minimum net edge required to trade, in basis points.
    pub min_edge_bps: f64,
    pub poll_interval_ms: u64,
}

impl CrossChainArbRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(2000)));
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.tick().await;
                    record(&store, &bot_id, "cross_chain", "arb_tick", &outcome).await;
                }
            }
        }
    }

    async fn tick(&self) -> StrategyOutcome {
        let start = std::time::Instant::now();

        // Quote both venues from their real routers.
        let buy_out = match dex::get_amounts_out(
            &self.buy.rpc_url,
            &self.buy.router,
            &self.buy.token_in,
            &self.buy.token_out,
        )
        .await
        {
            Ok(v) => v,
            Err(e) => return failure(format!("quote buy leg: {e}"), start),
        };
        let sell_out = match dex::get_amounts_out(
            &self.sell.rpc_url,
            &self.sell.router,
            &self.sell.token_in,
            &self.sell.token_out,
        )
        .await
        {
            Ok(v) => v,
            Err(e) => return failure(format!("quote sell leg: {e}"), start),
        };

        if buy_out <= 0.0 || sell_out <= 0.0 {
            return failure(
                format!("non-positive quote: buy={buy_out} sell={sell_out}"),
                start,
            );
        }

        // The edge is the dearer venue's output over the cheaper venue's output,
        // less the bridge cost.
        let gross_bps = (sell_out - buy_out) / buy_out * 10_000.0;
        let net_bps = gross_bps - self.bridge_cost_bps;
        if net_bps < self.min_edge_bps {
            return StrategyOutcome {
                success: true,
                detail: format!(
                    "hold gross={gross_bps:.1}bps net={net_bps:.1}bps (need {:.1}bps)",
                    self.min_edge_bps
                ),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }

        // Execute the buy leg, then the sell leg. Both are real on-chain swaps
        // with real slippage protection; if the first reverts the second is not
        // attempted, because a half-filled arb is an unintended position.
        let buy_result = match dex::execute_swap(&self.buy).await {
            Ok(r) => r,
            Err(e) => return failure(format!("buy leg failed: {e}"), start),
        };
        let sell_result = match dex::execute_swap(&self.sell).await {
            Ok(r) => r,
            Err(e) => {
                return failure(
                    format!(
                        "buy leg filled (tx={} out={}) but sell leg failed: {e}",
                        buy_result.tx_hash, buy_result.amount_out
                    ),
                    start,
                )
            }
        };

        StrategyOutcome {
            success: true,
            detail: format!(
                "arb filled net={net_bps:.1}bps buy_tx={} buy_out={} sell_tx={} sell_out={}",
                buy_result.tx_hash,
                buy_result.amount_out,
                sell_result.tx_hash,
                sell_result.amount_out
            ),
            latency_us: start.elapsed().as_micros() as u64,
        }
    }
}

// ============================================================================
// CUSTOM STRATEGY (SANDBOXED EXPRESSION)
// ============================================================================

/// A parsed custom-strategy expression.
///
/// The expression is parsed into an AST and evaluated with an allowlisted set of
/// functions over a small numeric environment. No host language is evaluated,
/// no process is spawned and no file or network resource is reachable from an
/// expression — an operator-supplied string can therefore not execute code.
#[derive(Debug, Clone, PartialEq)]
pub enum Expr {
    Number(f64),
    /// A named input, resolved from the evaluation environment.
    Var(String),
    Unary(char, Box<Expr>),
    Binary(char, Box<Expr>, Box<Expr>),
    Call(String, Vec<Expr>),
}

/// Tokenizes and parses an expression, returning a descriptive error on any
/// malformed input.
pub fn parse_expression(src: &str) -> Result<Expr, String> {
    let tokens = tokenize(src)?;
    let mut p = Parser { tokens, pos: 0 };
    let e = p.parse_expr()?;
    if p.pos != p.tokens.len() {
        return Err(format!("unexpected trailing input at token {}", p.pos));
    }
    Ok(e)
}

#[derive(Debug, Clone, PartialEq)]
enum Token {
    Number(f64),
    Ident(String),
    Op(char),
    LParen,
    RParen,
    Comma,
}

fn tokenize(src: &str) -> Result<Vec<Token>, String> {
    let chars: Vec<char> = src.chars().collect();
    let mut out = Vec::new();
    let mut i = 0;
    while i < chars.len() {
        let c = chars[i];
        if c.is_whitespace() {
            i += 1;
            continue;
        }
        if c.is_ascii_digit() || c == '.' {
            let start = i;
            while i < chars.len() && (chars[i].is_ascii_digit() || chars[i] == '.') {
                i += 1;
            }
            let text: String = chars[start..i].iter().collect();
            let n: f64 = text
                .parse()
                .map_err(|_| format!("invalid number literal {text:?}"))?;
            if !n.is_finite() {
                return Err(format!("number literal {text:?} is not finite"));
            }
            out.push(Token::Number(n));
            continue;
        }
        if c.is_alphabetic() || c == '_' {
            let start = i;
            while i < chars.len() && (chars[i].is_alphanumeric() || chars[i] == '_') {
                i += 1;
            }
            out.push(Token::Ident(chars[start..i].iter().collect()));
            continue;
        }
        match c {
            '+' | '-' | '*' | '/' | '%' | '^' | '<' | '>' => {
                out.push(Token::Op(c));
                i += 1;
            }
            '(' => {
                out.push(Token::LParen);
                i += 1;
            }
            ')' => {
                out.push(Token::RParen);
                i += 1;
            }
            ',' => {
                out.push(Token::Comma);
                i += 1;
            }
            other => return Err(format!("unexpected character {other:?}")),
        }
    }
    Ok(out)
}

struct Parser {
    tokens: Vec<Token>,
    pos: usize,
}

impl Parser {
    fn peek(&self) -> Option<&Token> {
        self.tokens.get(self.pos)
    }

    fn parse_expr(&mut self) -> Result<Expr, String> {
        self.parse_additive()
    }

    fn parse_additive(&mut self) -> Result<Expr, String> {
        let mut left = self.parse_multiplicative()?;
        while let Some(Token::Op(op @ ('+' | '-' | '<' | '>'))) = self.peek() {
            let op = *op;
            self.pos += 1;
            let right = self.parse_multiplicative()?;
            left = Expr::Binary(op, Box::new(left), Box::new(right));
        }
        Ok(left)
    }

    fn parse_multiplicative(&mut self) -> Result<Expr, String> {
        let mut left = self.parse_power()?;
        while let Some(Token::Op(op @ ('*' | '/' | '%'))) = self.peek() {
            let op = *op;
            self.pos += 1;
            let right = self.parse_power()?;
            left = Expr::Binary(op, Box::new(left), Box::new(right));
        }
        Ok(left)
    }

    fn parse_power(&mut self) -> Result<Expr, String> {
        let base = self.parse_unary()?;
        if let Some(Token::Op('^')) = self.peek() {
            self.pos += 1;
            // Right associative: a^b^c parses as a^(b^c).
            let exp = self.parse_power()?;
            return Ok(Expr::Binary('^', Box::new(base), Box::new(exp)));
        }
        Ok(base)
    }

    fn parse_unary(&mut self) -> Result<Expr, String> {
        if let Some(Token::Op(op @ ('-' | '+'))) = self.peek() {
            let op = *op;
            self.pos += 1;
            let inner = self.parse_unary()?;
            return Ok(Expr::Unary(op, Box::new(inner)));
        }
        self.parse_primary()
    }

    fn parse_primary(&mut self) -> Result<Expr, String> {
        match self.peek().cloned() {
            Some(Token::Number(n)) => {
                self.pos += 1;
                Ok(Expr::Number(n))
            }
            Some(Token::Ident(name)) => {
                self.pos += 1;
                if let Some(Token::LParen) = self.peek() {
                    self.pos += 1;
                    let mut args = Vec::new();
                    if let Some(Token::RParen) = self.peek() {
                        self.pos += 1;
                        return Ok(Expr::Call(name, args));
                    }
                    loop {
                        args.push(self.parse_expr()?);
                        match self.peek() {
                            Some(Token::Comma) => {
                                self.pos += 1;
                            }
                            Some(Token::RParen) => {
                                self.pos += 1;
                                break;
                            }
                            _ => return Err("expected ',' or ')' in argument list".into()),
                        }
                    }
                    return Ok(Expr::Call(name, args));
                }
                Ok(Expr::Var(name))
            }
            Some(Token::LParen) => {
                self.pos += 1;
                let inner = self.parse_expr()?;
                match self.peek() {
                    Some(Token::RParen) => {
                        self.pos += 1;
                        Ok(inner)
                    }
                    _ => Err("unbalanced parenthesis".into()),
                }
            }
            other => Err(format!("unexpected token {other:?}")),
        }
    }
}

/// Functions an expression may call. Each is pure and total on its input.
fn call_function(name: &str, args: &[f64]) -> Result<f64, String> {
    let arity = |n: usize| -> Result<(), String> {
        if args.len() == n {
            Ok(())
        } else {
            Err(format!("{name} takes {n} argument(s), got {}", args.len()))
        }
    };
    match name {
        "abs" => {
            arity(1)?;
            Ok(args[0].abs())
        }
        "min" => {
            arity(2)?;
            Ok(args[0].min(args[1]))
        }
        "max" => {
            arity(2)?;
            Ok(args[0].max(args[1]))
        }
        "sqrt" => {
            arity(1)?;
            if args[0] < 0.0 {
                return Err("sqrt of a negative value".into());
            }
            Ok(args[0].sqrt())
        }
        "ln" => {
            arity(1)?;
            if args[0] <= 0.0 {
                return Err("ln of a non-positive value".into());
            }
            Ok(args[0].ln())
        }
        "clamp" => {
            arity(3)?;
            Ok(args[0].clamp(args[1].min(args[2]), args[1].max(args[2])))
        }
        other => Err(format!("unknown function {other:?}")),
    }
}

/// Evaluates an expression against a variable environment.
pub fn eval_expression(expr: &Expr, env: &HashMap<String, f64>) -> Result<f64, String> {
    let value = match expr {
        Expr::Number(n) => *n,
        Expr::Var(name) => *env
            .get(name)
            .ok_or_else(|| format!("unknown variable {name:?}"))?,
        Expr::Unary(op, inner) => {
            let v = eval_expression(inner, env)?;
            match op {
                '-' => -v,
                '+' => v,
                other => return Err(format!("unsupported unary operator {other:?}")),
            }
        }
        Expr::Binary(op, l, r) => {
            let a = eval_expression(l, env)?;
            let b = eval_expression(r, env)?;
            match op {
                '+' => a + b,
                '-' => a - b,
                '*' => a * b,
                '/' => {
                    if b == 0.0 {
                        return Err("division by zero".into());
                    }
                    a / b
                }
                '%' => {
                    if b == 0.0 {
                        return Err("modulo by zero".into());
                    }
                    a % b
                }
                '^' => a.powf(b),
                // Comparisons yield 1.0/0.0 so an expression can gate an order
                // without needing a conditional operator.
                '<' => {
                    if a < b {
                        1.0
                    } else {
                        0.0
                    }
                }
                '>' => {
                    if a > b {
                        1.0
                    } else {
                        0.0
                    }
                }
                other => return Err(format!("unsupported operator {other:?}")),
            }
        }
        Expr::Call(name, args) => {
            let mut vals = Vec::with_capacity(args.len());
            for a in args {
                vals.push(eval_expression(a, env)?);
            }
            call_function(name, &vals)?
        }
    };
    if !value.is_finite() {
        return Err("expression produced a non-finite value".into());
    }
    Ok(value)
}

/// Custom strategy: evaluates the operator's expression each tick. A positive
/// result buys, a negative result sells, and zero holds. The magnitude scales
/// the order size, capped at `max_order_size`.
pub struct CustomRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    pub max_order_size: f64,
    pub expression: String,
    pub poll_interval_ms: u64,
}

impl CustomRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let bot_id = self.bot_id.clone();
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(1000)));

        // A malformed expression is a configuration error, not a transient one:
        // report it once and stop rather than looping on a failure forever.
        let parsed = match parse_expression(&self.expression) {
            Ok(e) => e,
            Err(e) => {
                let outcome = StrategyOutcome {
                    success: false,
                    detail: format!("custom expression rejected: {e}"),
                    latency_us: 0,
                };
                record(&store, &bot_id, "custom", "validate", &outcome).await;
                return;
            }
        };

        let mut prices: Vec<f64> = Vec::new();
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.tick(&client, &parsed, &mut prices).await;
                    record(&store, &bot_id, "custom", "signal_tick", &outcome).await;
                }
            }
        }
    }

    async fn tick(
        &self,
        client: &CexClient,
        parsed: &Expr,
        prices: &mut Vec<f64>,
    ) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let mid = match fetch_mid_price(self.exchange, &self.symbol, client.http()).await {
            Ok(m) => m,
            Err(e) => return failure(format!("fetch_mid: {e}"), start),
        };
        prices.push(mid);
        if prices.len() > 500 {
            prices.remove(0);
        }

        let n = prices.len() as f64;
        let mean = prices.iter().sum::<f64>() / n;
        let variance = prices.iter().map(|p| (p - mean).powi(2)).sum::<f64>() / n;
        let env: HashMap<String, f64> = HashMap::from([
            ("price".to_string(), mid),
            ("price_mean".to_string(), mean),
            ("price_std".to_string(), variance.sqrt()),
            ("price_min".to_string(), prices.iter().cloned().fold(f64::INFINITY, f64::min)),
            ("price_max".to_string(), prices.iter().cloned().fold(f64::NEG_INFINITY, f64::max)),
            ("samples".to_string(), n),
            ("position".to_string(), 0.0),
        ]);

        let value = match eval_expression(parsed, &env) {
            Ok(v) => v,
            Err(e) => return failure(format!("custom expression: {e}"), start),
        };
        if value == 0.0 {
            return StrategyOutcome {
                success: true,
                detail: format!("hold price={mid}"),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }
        let side = if value > 0.0 { "buy" } else { "sell" };
        let qty = round_qty(
            (self.order_size * value.abs()).min(self.max_order_size),
            8,
        );
        if qty <= 0.0 {
            return StrategyOutcome {
                success: true,
                detail: format!("hold: computed size {qty} is zero"),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }
        let mut outcome = market_order(client, &self.symbol, side, qty).await;
        outcome.detail = format!("custom value={value:.6} | {}", outcome.detail);
        outcome.latency_us = start.elapsed().as_micros() as u64;
        outcome
    }
}
