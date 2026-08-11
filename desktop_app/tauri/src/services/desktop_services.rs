/**
 * Desktop Trading Service
 * Order Book, Trading Charts, Positions for Tauri Desktop
 */

use serde::{Deserialize, Serialize};
use std::sync::Mutex;

// ============================================================================
// Trading Models
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderBook {
    pub bids: Vec<Order>,
    pub asks: Vec<Order>,
    pub timestamp: i64,
    pub symbol: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Order {
    pub price: f64,
    pub amount: f64,
    pub total: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Candlestick {
    pub timestamp: i64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
    pub volume: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub id: String,
    pub symbol: String,
    pub side: String,
    pub amount: f64,
    pub entry_price: f64,
    pub current_price: f64,
    pub unrealized_pnl: f64,
    pub leverage: i32,
    pub liquidation_price: f64,
    pub margin: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OpenOrder {
    pub id: String,
    pub symbol: String,
    pub side: String,
    pub order_type: String,
    pub price: f64,
    pub amount: f64,
    pub filled_amount: f64,
    pub status: String,
    pub created_at: i64,
}

// ============================================================================
// Trading Service
// ============================================================================

pub struct TradingService {
    client: reqwest::Client,
    base_url: String,
}

impl TradingService {
    pub fn new() -> Self {
        Self {
            client: reqwest::Client::new(),
            base_url: "http://localhost:8443/api/v1/trading".to_string(),
        }
    }

    pub async fn get_order_book(&self, symbol: &str, limit: i32) -> Result<OrderBook, String> {
        let url = format!("{}/orderbook?symbol={}&limit={}", self.base_url, symbol, limit);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let order_book: OrderBook = response.json().await.map_err(|e| e.to_string())?;
            Ok(order_book)
        } else {
            Err("Failed to get order book".to_string())
        }
    }

    pub async fn get_candlesticks(&self, symbol: &str, interval: &str, limit: i32) -> Result<Vec<Candlestick>, String> {
        let url = format!("{}/klines?symbol={}&interval={}&limit={}", self.base_url, symbol, interval, limit);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let candlesticks: Vec<Candlestick> = response.json().await.map_err(|e| e.to_string())?;
            Ok(candlesticks)
        } else {
            Err("Failed to get candlesticks".to_string())
        }
    }

    pub async fn get_positions(&self, wallet_address: &str) -> Result<Vec<Position>, String> {
        let url = format!("{}/positions/{}", self.base_url, wallet_address);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let positions: Vec<Position> = response.json().await.map_err(|e| e.to_string())?;
            Ok(positions)
        } else {
            Err("Failed to get positions".to_string())
        }
    }

    pub async fn get_open_orders(&self, wallet_address: &str) -> Result<Vec<OpenOrder>, String> {
        let url = format!("{}/orders/{}?status=open", self.base_url, wallet_address);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let orders: Vec<OpenOrder> = response.json().await.map_err(|e| e.to_string())?;
            Ok(orders)
        } else {
            Err("Failed to get open orders".to_string())
        }
    }

    pub async fn place_market_order(&self, wallet_address: &str, symbol: &str, side: &str, amount: f64, leverage: i32) -> Result<String, String> {
        let url = format!("{}/orders", self.base_url);
        
        let body = serde_json::json!({
            "wallet_address": wallet_address,
            "symbol": symbol,
            "side": side,
            "type": "market",
            "amount": amount,
            "leverage": leverage
        });
        
        let response = self.client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result["txHash"].as_str().unwrap_or("").to_string())
        } else {
            Err("Failed to place order".to_string())
        }
    }

    pub async fn place_limit_order(&self, wallet_address: &str, symbol: &str, side: &str, price: f64, amount: f64, leverage: i32) -> Result<String, String> {
        let url = format!("{}/orders", self.base_url);
        
        let body = serde_json::json!({
            "wallet_address": wallet_address,
            "symbol": symbol,
            "side": side,
            "type": "limit",
            "price": price,
            "amount": amount,
            "leverage": leverage
        });
        
        let response = self.client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result["txHash"].as_str().unwrap_or("").to_string())
        } else {
            Err("Failed to place order".to_string())
        }
    }
}

// ============================================================================
// MEV Protection Service
// ============================================================================

pub struct MEVProtectionService {
    client: reqwest::Client,
}

impl MEVProtectionService {
    pub fn new() -> Self {
        Self {
            client: reqwest::Client::new(),
        }
    }

    pub async fn detect_sandwich_attack(&self, tx_hash: &str) -> Result<serde_json::Value, String> {
        let url = format!("http://localhost:8443/api/v1/mev/detect-sandwich?tx={}", tx_hash);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result)
        } else {
            Err("Failed to detect sandwich attack".to_string())
        }
    }

    pub async fn simulate_transaction(&self, from: &str, to: &str, data: &str, value: &str) -> Result<serde_json::Value, String> {
        let url = "http://localhost:8443/api/v1/mev/simulate".to_string();
        
        let body = serde_json::json!({
            "from": from,
            "to": to,
            "data": data,
            "value": value
        });
        
        let response = self.client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result)
        } else {
            Err("Failed to simulate transaction".to_string())
        }
    }

    pub async fn submit_with_protection(&self, signed_tx: &str, protection_level: &str) -> Result<String, String> {
        let url = "http://localhost:8443/api/v1/mev/submit".to_string();
        
        let body = serde_json::json!({
            "signed_tx": signed_tx,
            "protection_level": protection_level
        });
        
        let response = self.client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result["txHash"].as_str().unwrap_or("").to_string())
        } else {
            Err("Failed to submit with protection".to_string())
        }
    }
}

// ============================================================================
// Session Keys Service
// ============================================================================

pub struct SessionKeysService {
    client: reqwest::Client,
}

impl SessionKeysService {
    pub fn new() -> Self {
        Self {
            client: reqwest::Client::new(),
        }
    }

    pub async fn generate(&self, wallet_address: &str, dapp_url: &str, permissions: Vec<String>, expires_in: i64) -> Result<serde_json::Value, String> {
        let url = "http://localhost:8443/api/v1/session-keys".to_string();
        
        let body = serde_json::json!({
            "wallet_address": wallet_address,
            "dapp_url": dapp_url,
            "permissions": permissions,
            "expires_in": expires_in
        });
        
        let response = self.client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result)
        } else {
            Err("Failed to generate session key".to_string())
        }
    }

    pub async fn list(&self, wallet_address: &str) -> Result<Vec<serde_json::Value>, String> {
        let url = format!("http://localhost:8443/api/v1/session-keys/{}", wallet_address);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: Vec<serde_json::Value> = response.json().await.map_err(|e| e.to_string())?;
            Ok(result)
        } else {
            Err("Failed to list session keys".to_string())
        }
    }

    pub async fn revoke(&self, wallet_address: &str, session_key_id: &str) -> Result<bool, String> {
        let url = format!("http://localhost:8443/api/v1/session-keys/{}", session_key_id);
        
        let body = serde_json::json!({
            "wallet_address": wallet_address
        });
        
        let response = self.client
            .delete(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        Ok(response.status().is_success())
    }
}

// ============================================================================
// Gas Optimization Service
// ============================================================================

pub struct GasOptimizationService {
    client: reqwest::Client,
}

impl GasOptimizationService {
    pub fn new() -> Self {
        Self {
            client: reqwest::Client::new(),
        }
    }

    pub async fn get_prices(&self, chain: &str) -> Result<serde_json::Value, String> {
        let url = format!("http://localhost:8443/api/v1/gas/prices?chain={}", chain);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: serde_json::Value = response.json().await.map_err(|e| e.to_string())?;
            Ok(result)
        } else {
            Err("Failed to get gas prices".to_string())
        }
    }

    pub async fn get_suggestions(&self, from: &str, to: &str, data: &str) -> Result<Vec<serde_json::Value>, String> {
        let url = "http://localhost:8443/api/v1/gas/optimize".to_string();
        
        let body = serde_json::json!({
            "from": from,
            "to": to,
            "data": data
        });
        
        let response = self.client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        
        if response.status().is_success() {
            let result: Vec<serde_json::Value> = response.json().await.map_err(|e| e.to_string())?;
            Ok(result)
        } else {
            Err("Failed to get optimization suggestions".to_string())
        }
    }
}

// ============================================================================
// Widget SDK Service
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Widget {
    pub widget_type: String,
    pub data: serde_json::Value,
}

pub struct WidgetSDKService;

impl WidgetSDKService {
    pub fn create_balance_widget(wallet_address: &str) -> Widget {
        Widget {
            widget_type: "balance".to_string(),
            data: serde_json::json!({
                "wallet_address": wallet_address,
                "update_url": format!("http://localhost:8443/api/v1/wallet/{}/balance", wallet_address)
            }),
        }
    }

    pub fn create_price_widget(token: &str) -> Widget {
        Widget {
            widget_type: "price".to_string(),
            data: serde_json::json!({
                "token": token,
                "update_url": format!("http://localhost:8443/api/v1/prices/{}", token)
            }),
        }
    }

    pub fn create_portfolio_widget(wallet_address: &str) -> Widget {
        Widget {
            widget_type: "portfolio".to_string(),
            data: serde_json::json!({
                "wallet_address": wallet_address,
                "update_url": format!("http://localhost:8443/api/v1/wallet/{}/portfolio", wallet_address)
            }),
        }
    }

    pub fn create_quick_send_widget() -> Widget {
        Widget {
            widget_type: "quick_send".to_string(),
            data: serde_json::json!({
                "actions": ["send", "swap", "bridge"]
            }),
        }
    }
}
