//! Bridge Simulator
//!
//! Quotes a cross-chain transfer from routes the caller registered. Fees,
//! gas and bridge availability are operator- and chain-specific, so none of
//! them are invented here: an unregistered route fails closed with a reason
//! instead of returning a plausible-looking quote.

use std::collections::HashMap;
use std::sync::RwLock;

use crate::gas_simulator::GasSource;
use crate::SimulationResult;

/// Denominator for basis-point fees: 10_000 bps == 100%.
pub const BPS_DENOMINATOR: u128 = 10_000;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BridgeParams {
    pub from_chain: String,
    pub to_chain: String,
    pub token: String,
    pub amount: u128,
}

/// A route the operator has actually enabled, with its real terms.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BridgeRoute {
    pub from_chain: String,
    pub to_chain: String,
    /// Bridge protocol fee in basis points.
    pub fee_bps: u64,
    /// Gas the source-chain transaction will consume.
    pub gas_limit: u64,
    /// Whether the route currently accepts transfers.
    pub enabled: bool,
}

pub struct BridgeSimulator {
    routes: RwLock<HashMap<String, BridgeRoute>>,
    /// Observed source-chain gas prices, keyed by chain name.
    gas_prices: RwLock<HashMap<String, u128>>,
}

fn route_key(from: &str, to: &str) -> String {
    format!("{}-{}", from, to)
}

impl BridgeSimulator {
    pub fn new() -> Self {
        Self {
            routes: RwLock::new(HashMap::new()),
            gas_prices: RwLock::new(HashMap::new()),
        }
    }

    /// Register (or replace) a bridge route.
    pub fn set_route(&self, route: BridgeRoute) {
        let key = route_key(&route.from_chain, &route.to_chain);
        self.routes.write().unwrap().insert(key, route);
    }

    /// Record an observed gas price for a chain (wei per gas unit).
    pub fn set_gas_price(&self, chain: &str, wei: u128) {
        self.gas_prices
            .write()
            .unwrap()
            .insert(chain.to_string(), wei);
    }

    pub fn route(&self, from: &str, to: &str) -> Option<BridgeRoute> {
        self.routes.read().unwrap().get(&route_key(from, to)).cloned()
    }

    /// Quote a transfer. Fails closed with a specific reason when the route is
    /// unknown, disabled, mis-priced, or the arithmetic overflows.
    pub fn simulate(&self, params: &BridgeParams) -> SimulationResult {
        if params.amount == 0 {
            return SimulationResult::failure("amount must be greater than zero");
        }
        if params.from_chain == params.to_chain {
            return SimulationResult::failure("from_chain and to_chain are the same chain");
        }

        let route = match self.route(&params.from_chain, &params.to_chain) {
            Some(route) => route,
            None => {
                return SimulationResult::failure(format!(
                    "no bridge route from {} to {}",
                    params.from_chain, params.to_chain
                ))
            }
        };

        if !route.enabled {
            return SimulationResult::failure(format!(
                "bridge route {}->{} is currently disabled",
                params.from_chain, params.to_chain
            ));
        }

        if route.fee_bps as u128 >= BPS_DENOMINATOR {
            return SimulationResult::failure("bridge fee_bps must be below 10000");
        }

        let fee = match params.amount.checked_mul(route.fee_bps as u128) {
            Some(product) => product / BPS_DENOMINATOR,
            None => return SimulationResult::failure("bridge fee arithmetic overflowed"),
        };

        let output = params.amount - fee;
        if output == 0 {
            return SimulationResult::failure("bridge fee consumes the entire transfer amount");
        }

        let gas_wei = self
            .gas_prices
            .read()
            .unwrap()
            .get(&params.from_chain)
            .copied()
            .and_then(|price| (route.gas_limit as u128).checked_mul(price));

        SimulationResult {
            success: true,
            expected_output: output,
            actual_output: Some(output),
            gas_used: route.gas_limit,
            // Reported in bps as an integer so no floating-point rounding can
            // creep into a fee figure.
            fee_bps: route.fee_bps,
            gas_cost_wei: gas_wei,
            gas_source: gas_wei.map(|_| GasSource::Oracle),
            price_impact_bps: 0,
            errors: Vec::new(),
        }
    }
}

impl Default for BridgeSimulator {
    fn default() -> Self {
        Self::new()
    }
}
