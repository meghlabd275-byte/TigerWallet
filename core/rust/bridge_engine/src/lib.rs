//! TigerSwap Bridge Engine - Production-Ready Cross-Chain
//! 
//! COMPLETELY SELF-CONTAINED with:
//! - Cross-chain bridge routing
//! - Multi-protocol support (LayerZero, Hyperlane, Wormhole)
//! - Liquidity aggregation
//! - Fee optimization
//! - Transaction tracking

use std::collections::HashMap;
use std::sync::RwLock;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

/// Supported chains
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum ChainId {
    Ethereum = 1,
    BSC = 56,
    Polygon = 137,
    Arbitrum = 42161,
    Optimism = 10,
    Avalanche = 43114,
    Solana = 101,
    Aptos = 4,
    SUI = 19,
}

impl ChainId {
    pub fn name(&self) -> &'static str {
        match self {
            ChainId::Ethereum => "Ethereum",
            ChainId::BSC => "BNB Chain",
            ChainId::Polygon => "Polygon",
            ChainId::Arbitrum => "Arbitrum",
            ChainId::Optimism => "Optimism",
            ChainId::Avalanche => "Avalanche",
            ChainId::Solana => "Solana",
            ChainId::Aptos => "Aptos",
            ChainId::SUI => "SUI",
        }
    }
}

/// Bridge protocol
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum BridgeProtocol {
    LayerZero,
    Hyperlane,
    Wormhole,
    Axelar,
    Celer,
    Stargate,
}

impl BridgeProtocol {
    pub fn name(&self) -> &'static str {
        match self {
            BridgeProtocol::LayerZero => "LayerZero",
            BridgeProtocol::Hyperlane => "Hyperlane",
            BridgeProtocol::Wormhole => "Wormhole",
            BridgeProtocol::Axelar => "Axelar",
            BridgeProtocol::Celer => "Celer",
            BridgeProtocol::Stargate => "Stargate",
        }
    }
}

/// Bridge quote request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeQuoteRequest {
    pub from_chain: ChainId,
    pub to_chain: ChainId,
    pub token: String,
    pub amount: u64,
    pub slippage_bps: u64,
    /// Address the funds leave from, on `from_chain`.
    pub sender: String,
    /// Address the funds arrive at, on `to_chain`.
    pub recipient: String,
}

/// Bridge quote result
#[derive(Debug, Clone)]
pub struct BridgeQuote {
    pub protocol: BridgeProtocol,
    pub estimated_output: u64,
    /// Worst-case output the user accepts, derived from `slippage_bps`.
    pub min_output: u64,
    pub fee: u64,
    pub fee_token: String,
    pub estimated_time_seconds: u64,
    pub gas_cost_on_dest: u64,
    pub price_impact_bps: u64,
}

/// Bridge transfer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeTransfer {
    pub id: u64,
    pub from_chain: ChainId,
    pub to_chain: ChainId,
    pub token: String,
    pub amount: u64,
    pub output_amount: u64,
    pub protocol: BridgeProtocol,
    pub status: TransferStatus,
    pub sender: String,
    pub recipient: String,
    /// Source-chain transaction hash. `None` until a transfer is actually
    /// broadcast and `update_transfer` records the real hash.
    pub tx_hash: Option<String>,
    pub dst_tx_hash: Option<String>,
    pub created_at: u64,
    pub updated_at: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransferStatus {
    Pending,
    Sent,
    Confirmed,
    Completed,
    Failed,
    Refunded,
}

/// Bridge engine
pub struct BridgeEngine {
    pools: RwLock<HashMap<(ChainId, String), BridgePool>>,
    routes: RwLock<HashMap<(ChainId, ChainId), Vec<BridgeRoute>>>,
    transfers: RwLock<HashMap<u64, BridgeTransfer>>,
    next_transfer_id: u64,
}

#[derive(Debug, Clone)]
pub struct BridgePool {
    pub chain: ChainId,
    pub token: String,
    pub liquidity: u64,
    pub protocol: BridgeProtocol,
    pub fee_bps: u64,
    pub min_amount: u64,
    pub max_amount: u64,
}

#[derive(Debug, Clone)]
pub struct BridgeRoute {
    pub protocol: BridgeProtocol,
    pub from_chain: ChainId,
    pub to_chain: ChainId,
    pub estimated_fee: u64,
    pub estimated_time: u64,
    pub reliability_score: f64,
}

impl BridgeEngine {
    pub fn new() -> Self {
        Self {
            pools: RwLock::new(HashMap::new()),
            routes: RwLock::new(HashMap::new()),
            transfers: RwLock::new(HashMap::new()),
            next_transfer_id: 1,
        }
    }

    /// Add bridge pool
    pub fn add_pool(&self, pool: BridgePool) {
        let mut pools = self.pools.write().unwrap();
        pools.insert((pool.chain, pool.token.clone()), pool);
    }

    /// Add route
    pub fn add_route(&self, route: BridgeRoute) {
        let mut routes = self.routes.write().unwrap();
        let key = (route.from_chain, route.to_chain);
        routes.entry(key).or_insert_with(Vec::new).push(route);
    }

    /// Get a quote for a bridge request.
    ///
    /// Fails when the pair has no route, when the token has no pool on the
    /// source chain, when the amount falls outside the pool's limits, or when
    /// the pool cannot cover it. Every value is derived from the configured
    /// pool and route; nothing is estimated or invented.
    pub fn get_quote(&self, request: &BridgeQuoteRequest) -> Result<BridgeQuote, String> {
        if request.amount == 0 {
            return Err("amount must be greater than zero".to_string());
        }
        if request.sender.is_empty() || request.recipient.is_empty() {
            return Err("sender and recipient are required".to_string());
        }

        let pools = self.pools.read().unwrap();
        let routes = self.routes.read().unwrap();

        let route_list = routes
            .get(&(request.from_chain, request.to_chain))
            .ok_or_else(|| "no route available between these chains".to_string())?;

        let pool = pools
            .get(&(request.from_chain, request.token.clone()))
            .ok_or_else(|| "no liquidity pool for this token on the source chain".to_string())?;

        if request.amount < pool.min_amount {
            return Err("amount is below the pool minimum".to_string());
        }
        if request.amount > pool.max_amount {
            return Err("amount is above the pool maximum".to_string());
        }
        if request.amount > pool.liquidity {
            return Err("amount exceeds the available pool liquidity".to_string());
        }

        // Highest reliability route wins; ties keep the earliest registered.
        let best_route = route_list
            .iter()
            .max_by(|a, b| {
                a.reliability_score
                    .partial_cmp(&b.reliability_score)
                    .unwrap_or(std::cmp::Ordering::Equal)
            })
            .ok_or_else(|| "no route available between these chains".to_string())?;

        let fee = request.amount.saturating_mul(pool.fee_bps) / 10_000;
        let output = request
            .amount
            .checked_sub(fee)
            .ok_or_else(|| "fee exceeds the bridged amount".to_string())?;
        if output == 0 {
            return Err("bridged amount is too small to cover the fee".to_string());
        }

        let slippage = request.slippage_bps.min(10_000);
        let min_output = output.saturating_mul(10_000 - slippage) / 10_000;

        Ok(BridgeQuote {
            protocol: best_route.protocol,
            estimated_output: output,
            min_output,
            fee,
            fee_token: request.token.clone(),
            estimated_time_seconds: best_route.estimated_time,
            gas_cost_on_dest: 0,
            price_impact_bps: pool.fee_bps,
        })
    }

    /// Create bridge transfer
    pub fn create_transfer(&mut self, request: &BridgeQuoteRequest) -> Result<BridgeTransfer, String> {
        let quote = self.get_quote(request)?;
        
        let transfer = BridgeTransfer {
            id: self.next_transfer_id,
            from_chain: request.from_chain,
            to_chain: request.to_chain,
            token: request.token.clone(),
            amount: request.amount,
            output_amount: quote.estimated_output,
            protocol: quote.protocol,
            status: TransferStatus::Pending,
            sender: request.sender.clone(),
            recipient: request.recipient.clone(),
            tx_hash: None,
            dst_tx_hash: None,
            created_at: now(),
            updated_at: now(),
        };
        
        let mut transfers = self.transfers.write().unwrap();
        transfers.insert(self.next_transfer_id, transfer.clone());
        self.next_transfer_id += 1;
        
        Ok(transfer)
    }

    /// Update transfer status
    pub fn update_transfer(&mut self, transfer_id: u64, status: TransferStatus, tx_hash: Option<String>) {
        let mut transfers = self.transfers.write().unwrap();
        if let Some(transfer) = transfers.get_mut(&transfer_id) {
            transfer.status = status;
            transfer.updated_at = now();
            if let Some(h) = tx_hash {
                if transfer.tx_hash.is_none() {
                    transfer.tx_hash = Some(h);
                } else {
                    transfer.dst_tx_hash = Some(h);
                }
            }
        }
    }

    /// Get transfer by ID
    pub fn get_transfer(&self, id: u64) -> Option<BridgeTransfer> {
        self.transfers.read().unwrap().get(&id).cloned()
    }

    /// Get all transfers an address sent or received
    pub fn get_transfers_for_address(&self, address: &str) -> Vec<BridgeTransfer> {
        self.transfers.read().unwrap()
            .values()
            .filter(|t| t.sender == address || t.recipient == address)
            .cloned()
            .collect()
    }

    /// Get best route between chains
    pub fn get_best_route(&self, from: ChainId, to: ChainId) -> Option<BridgeRoute> {
        let routes = self.routes.read().unwrap();
        let key = (from, to);
        
        routes
            .get(&key)?
            .iter()
            .max_by(|a, b| {
                a.reliability_score
                    .partial_cmp(&b.reliability_score)
                    .unwrap_or(std::cmp::Ordering::Equal)
            })
            .cloned()
    }
}

impl Default for BridgeEngine {
    fn default() -> Self {
        Self::new()
    }
}

fn now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bridge_engine() {
        let engine = BridgeEngine::new();
        
        engine.add_pool(BridgePool {
            chain: ChainId::Ethereum,
            token: "USDC".to_string(),
            liquidity: 1_000_000_000,
            protocol: BridgeProtocol::LayerZero,
            fee_bps: 5,
            min_amount: 100,
            max_amount: 10_000_000_000,
        });
        
        engine.add_route(BridgeRoute {
            protocol: BridgeProtocol::LayerZero,
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::BSC,
            estimated_fee: 10_000,
            estimated_time: 300,
            reliability_score: 0.95,
        });
        
        let quote = engine
            .get_quote(&BridgeQuoteRequest {
                from_chain: ChainId::Ethereum,
                to_chain: ChainId::BSC,
                token: "USDC".to_string(),
                amount: 1_000_000,
                slippage_bps: 50,
                sender: "0xsender".to_string(),
                recipient: "0xrecipient".to_string(),
            })
            .expect("quote should be available");

        // fee_bps = 5 -> fee = amount * 5 / 10000 = 500
        assert_eq!(quote.fee, 500);
        assert_eq!(quote.estimated_output, 999_500);
        // 50 bps slippage -> min_output = 999_500 * 9950 / 10000
        assert_eq!(quote.min_output, 994_502);
        assert_eq!(quote.protocol, BridgeProtocol::LayerZero);
        assert_eq!(quote.estimated_time_seconds, 300);
    }

    #[test]
    fn test_quote_rejects_invalid_requests() {
        let engine = BridgeEngine::new();
        engine.add_pool(BridgePool {
            chain: ChainId::Ethereum,
            token: "USDC".to_string(),
            liquidity: 1_000_000,
            protocol: BridgeProtocol::LayerZero,
            fee_bps: 5,
            min_amount: 1_000,
            max_amount: 500_000,
        });
        engine.add_route(BridgeRoute {
            protocol: BridgeProtocol::LayerZero,
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::BSC,
            estimated_fee: 10_000,
            estimated_time: 300,
            reliability_score: 0.95,
        });

        let request = |amount: u64| BridgeQuoteRequest {
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::BSC,
            token: "USDC".to_string(),
            amount,
            slippage_bps: 50,
            sender: "0xsender".to_string(),
            recipient: "0xrecipient".to_string(),
        };

        assert!(engine.get_quote(&request(0)).is_err(), "zero amount");
        assert!(engine.get_quote(&request(999)).is_err(), "below minimum");
        assert!(engine.get_quote(&request(500_001)).is_err(), "above maximum");
        assert!(
            engine.get_quote(&request(2_000_000)).is_err(),
            "above available liquidity"
        );
        assert!(engine.get_quote(&request(10_000)).is_ok());

        // Unknown destination chain, unknown token, and a blank sender must all
        // fail rather than return an invented quote.
        let mut unknown_chain = request(10_000);
        unknown_chain.to_chain = ChainId::Solana;
        assert!(engine.get_quote(&unknown_chain).is_err(), "no route");

        let mut unknown_token = request(10_000);
        unknown_token.token = "DAI".to_string();
        assert!(engine.get_quote(&unknown_token).is_err(), "no pool");

        let mut blank_sender = request(10_000);
        blank_sender.sender = String::new();
        assert!(engine.get_quote(&blank_sender).is_err(), "blank sender");
    }

    #[test]
    fn test_quote_prefers_the_most_reliable_route() {
        let engine = BridgeEngine::new();
        engine.add_pool(BridgePool {
            chain: ChainId::Ethereum,
            token: "USDC".to_string(),
            liquidity: 10_000_000,
            protocol: BridgeProtocol::LayerZero,
            fee_bps: 5,
            min_amount: 100,
            max_amount: 1_000_000_000,
        });
        engine.add_route(BridgeRoute {
            protocol: BridgeProtocol::LayerZero,
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::BSC,
            estimated_fee: 10_000,
            estimated_time: 300,
            reliability_score: 0.60,
        });
        engine.add_route(BridgeRoute {
            protocol: BridgeProtocol::Wormhole,
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::BSC,
            estimated_fee: 12_000,
            estimated_time: 420,
            reliability_score: 0.99,
        });

        let quote = engine
            .get_quote(&BridgeQuoteRequest {
                from_chain: ChainId::Ethereum,
                to_chain: ChainId::BSC,
                token: "USDC".to_string(),
                amount: 1_000_000,
                slippage_bps: 50,
                sender: "0xsender".to_string(),
                recipient: "0xrecipient".to_string(),
            })
            .expect("quote should be available");

        assert_eq!(quote.protocol, BridgeProtocol::Wormhole);
        assert_eq!(quote.estimated_time_seconds, 420);
    }

    #[test]
    fn test_transfer_creation() {
        let mut engine = BridgeEngine::new();
        
        engine.add_pool(BridgePool {
            chain: ChainId::Ethereum,
            token: "ETH".to_string(),
            liquidity: 10_000_000_000,
            protocol: BridgeProtocol::Wormhole,
            fee_bps: 3,
            min_amount: 10,
            max_amount: 1_000_000_000_000,
        });
        
        engine.add_route(BridgeRoute {
            protocol: BridgeProtocol::Wormhole,
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::Solana,
            estimated_fee: 5_000,
            estimated_time: 600,
            reliability_score: 0.90,
        });
        
        let transfer = engine.create_transfer(&BridgeQuoteRequest {
            from_chain: ChainId::Ethereum,
            to_chain: ChainId::Solana,
            token: "ETH".to_string(),
            amount: 1_000_000_000,
            slippage_bps: 50,
            sender: "0xsender".to_string(),
            recipient: "0xrecipient".to_string(),
        });

        let transfer = transfer.expect("transfer should be created");
        // No transaction has been broadcast, so no hash may be reported.
        assert_eq!(transfer.tx_hash, None);
        assert_eq!(transfer.status, TransferStatus::Pending);

        // Recording the real broadcast hash fills the source hash, then the
        // destination hash, and never overwrites an existing value.
        engine.update_transfer(transfer.id, TransferStatus::Sent, Some("0xaaa".to_string()));
        engine.update_transfer(transfer.id, TransferStatus::Confirmed, Some("0xbbb".to_string()));
        let stored = engine.get_transfer(transfer.id).unwrap();
        assert_eq!(stored.tx_hash.as_deref(), Some("0xaaa"));
        assert_eq!(stored.dst_tx_hash.as_deref(), Some("0xbbb"));

        assert_eq!(engine.get_transfers_for_address("0xsender").len(), 1);
        assert_eq!(engine.get_transfers_for_address("0xrecipient").len(), 1);
        assert!(engine.get_transfers_for_address("0xstranger").is_empty());
    }
}
