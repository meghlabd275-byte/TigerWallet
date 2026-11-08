//! TigerSwap Simulation Engine
//!
//! Pre-flight quotes for swaps, routes, gas and bridges.
//!
//! Every figure produced here is derived from inputs the caller supplied
//! (pool reserves, gas prices, bridge terms). The engine never invents a
//! price, a fee, a gas figure or a success flag: an input it does not have
//! produces a failed result carrying the reason, so a caller can refuse to
//! act rather than acting on a guess.
//!
//! All amounts are integers. Fees and price impact are reported in basis
//! points, which keeps rounding explicit and reproducible.

use std::sync::Arc;

mod bridge_simulator;
mod gas_simulator;
mod route_simulator;
mod swap_simulator;

pub use bridge_simulator::{BridgeParams, BridgeRoute, BridgeSimulator, BPS_DENOMINATOR};
pub use gas_simulator::{
    GasCost, GasPrice, GasSimulator, GasSource, GAS_NATIVE_TRANSFER, GAS_TOKEN_APPROVAL,
    GAS_TOKEN_TRANSFER, GAS_TOKEN_TRANSFER_NEW_RECIPIENT,
};
pub use route_simulator::{Route, RouteSimulator, RouteStep, GAS_PER_HOP, MAX_HOPS};
pub use swap_simulator::{
    amount_out, pool_key, PoolInfo, SwapParams, SwapResult, SwapSimulator, FEE_DENOMINATOR,
};

/// Outcome of a pre-flight check.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SimulationResult {
    pub success: bool,
    pub expected_output: u128,
    /// Output actually observed after execution, when the caller re-runs the
    /// simulation against post-execution state.
    pub actual_output: Option<u128>,
    pub gas_used: u64,
    /// Bridge or pool fee in basis points.
    pub fee_bps: u64,
    /// Gas cost in wei, present only when a gas price was known.
    pub gas_cost_wei: Option<u128>,
    pub gas_source: Option<GasSource>,
    pub price_impact_bps: u64,
    pub errors: Vec<String>,
}

impl SimulationResult {
    /// A failed check. `success` is always false here, so no caller can
    /// mistake a missing input for a passing simulation.
    pub fn failure(reason: impl Into<String>) -> Self {
        Self {
            success: false,
            expected_output: 0,
            actual_output: None,
            gas_used: 0,
            fee_bps: 0,
            gas_cost_wei: None,
            gas_source: None,
            price_impact_bps: 0,
            errors: vec![reason.into()],
        }
    }
}

/// Coordinates the swap, route, gas and bridge simulators over one pool set.
pub struct SimulationEngine {
    swap: Arc<SwapSimulator>,
    route: Arc<RouteSimulator>,
    gas: Arc<GasSimulator>,
    bridge: Arc<BridgeSimulator>,
}

impl SimulationEngine {
    pub fn new() -> Self {
        let swap = Arc::new(SwapSimulator::new());
        Self {
            route: Arc::new(RouteSimulator::new(Arc::clone(&swap))),
            swap,
            gas: Arc::new(GasSimulator::new()),
            bridge: Arc::new(BridgeSimulator::new()),
        }
    }

    pub fn swap_simulator(&self) -> &SwapSimulator {
        &self.swap
    }

    pub fn route_simulator(&self) -> &RouteSimulator {
        &self.route
    }

    pub fn gas_simulator(&self) -> &GasSimulator {
        &self.gas
    }

    pub fn bridge_simulator(&self) -> &BridgeSimulator {
        &self.bridge
    }

    /// Simulate a swap, including the best multi-hop route when the direct
    /// pool is absent or thin.
    ///
    /// `measured_gas` is the caller's `eth_estimateGas` result for this exact
    /// transaction; pass `None` only when a router constant is acceptable.
    pub fn simulate_swap(
        &self,
        params: &SwapParams,
        chain: &str,
        measured_gas: Option<u64>,
    ) -> SimulationResult {
        // Prefer the best discovered route so the quote reflects the path the
        // router would actually take.
        let route = self
            .route
            .find_optimal_route(&params.token_in, &params.token_out, params.amount_in);

        let direct = self.swap.simulate(params);
        let (expected_output, price_impact_bps, hops) = match route {
            Some(route) => (route.total_output, direct.price_impact_bps, route.steps.len()),
            None if direct.success => (direct.expected_output, direct.price_impact_bps, 1),
            None => {
                return SimulationResult::failure(
                    direct
                        .errors
                        .first()
                        .cloned()
                        .unwrap_or_else(|| "no route found".to_string()),
                )
            }
        };

        let gas_used = match self.gas.estimate_swap_gas(measured_gas, hops) {
            Some(gas) => gas,
            None => return SimulationResult::failure("unable to determine gas for swap"),
        };

        let gas_price = self.gas.get_gas_price(chain);
        let gas_cost_wei =
            gas_price.and_then(|price| (gas_used as u128).checked_mul(price.wei));

        // Fee actually charged, as basis points of the input.
        let fee_bps = direct
            .fee_paid
            .checked_mul(FEE_DENOMINATOR)
            .and_then(|scaled| scaled.checked_div(params.amount_in))
            .unwrap_or(0) as u64;

        SimulationResult {
            success: true,
            expected_output,
            actual_output: None,
            gas_used,
            fee_bps,
            gas_cost_wei,
            gas_source: gas_price.map(|price| price.source),
            price_impact_bps,
            errors: Vec::new(),
        }
    }

    /// Simulate a cross-chain bridge transfer.
    pub fn simulate_bridge(&self, params: &BridgeParams) -> SimulationResult {
        self.bridge.simulate(params)
    }
}

impl Default for SimulationEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn engine_with_pool() -> SimulationEngine {
        let engine = SimulationEngine::new();
        engine.swap_simulator().add_pool(
            "USDC".to_string(),
            "ETH".to_string(),
            1_000_000_000_000,
            500_000_000_000,
            30,
        );
        engine
    }

    #[test]
    fn swap_quote_matches_constant_product_formula() {
        let engine = engine_with_pool();
        let params = SwapParams {
            token_in: "USDC".to_string(),
            token_out: "ETH".to_string(),
            amount_in: 1_000_000,
            slippage_bps: 50,
        };

        let result = engine.swap_simulator().simulate(&params);
        assert!(result.success, "errors: {:?}", result.errors);

        // Independently recompute the canonical getAmountOut formula.
        let amount_in_with_fee: u128 = 1_000_000 * (10_000 - 30);
        let numerator = amount_in_with_fee * 500_000_000_000u128;
        let denominator = 1_000_000_000_000u128 * 10_000 + amount_in_with_fee;
        assert_eq!(result.expected_output, numerator / denominator);
        assert_eq!(
            result.min_output,
            result.expected_output * (10_000 - 50) / 10_000
        );
        assert!(result.price_impact_bps > 0, "a real swap moves the price");
    }

    #[test]
    fn swap_is_reversible_by_token_order() {
        let engine = engine_with_pool();
        let forward = engine.swap_simulator().get_amount_out("USDC", "ETH", 1_000_000);
        let reverse = engine.swap_simulator().get_amount_out("ETH", "USDC", 1_000_000);
        assert!(forward.is_some(), "pool must resolve in pool order");
        assert!(reverse.is_some(), "pool must resolve in reverse order");
    }

    #[test]
    fn unknown_pool_fails_closed() {
        let engine = SimulationEngine::new();
        let params = SwapParams {
            token_in: "USDC".to_string(),
            token_out: "ETH".to_string(),
            amount_in: 1_000_000,
            slippage_bps: 50,
        };
        let result = engine.swap_simulator().simulate(&params);
        assert!(!result.success);
        assert!(result.errors[0].contains("no pool registered"));
    }

    #[test]
    fn gas_is_unknown_until_a_price_is_observed() {
        let engine = engine_with_pool();
        assert!(engine.gas_simulator().get_gas_price("ethereum").is_none());

        let params = SwapParams {
            token_in: "USDC".to_string(),
            token_out: "ETH".to_string(),
            amount_in: 1_000_000,
            slippage_bps: 50,
        };
        let without_price = engine.simulate_swap(&params, "ethereum", None);
        assert!(without_price.success);
        assert!(without_price.gas_cost_wei.is_none());

        engine
            .gas_simulator()
            .set_gas_price("ethereum", 20_000_000_000, GasSource::Oracle);
        let with_price = engine.simulate_swap(&params, "ethereum", None);
        assert_eq!(
            with_price.gas_cost_wei,
            Some(with_price.gas_used as u128 * 20_000_000_000)
        );
    }

    #[test]
    fn multi_hop_route_prices_each_hop() {
        let engine = SimulationEngine::new();
        let swap = engine.swap_simulator();
        swap.add_pool("USDC".to_string(), "WETH".to_string(), 1_000_000, 1_000, 30);
        swap.add_pool("WETH".to_string(), "TGR".to_string(), 1_000, 2_000_000, 30);

        let route = engine
            .route_simulator()
            .find_optimal_route("USDC", "TGR", 10_000)
            .expect("a two-hop route exists");

        assert_eq!(route.steps.len(), 2);
        assert_eq!(route.total_gas, GAS_PER_HOP * 2);
        // The second hop consumes the first hop's output.
        assert_eq!(route.steps[0].amount_out, route.steps[1].amount_in);
        assert_eq!(route.total_output, route.steps[1].amount_out);
    }

    #[test]
    fn unreachable_token_yields_no_route() {
        let engine = engine_with_pool();
        assert!(engine
            .route_simulator()
            .find_optimal_route("USDC", "TGR", 10_000)
            .is_none());
        assert_eq!(engine.route_simulator().get_routes("USDC", "TGR", 10_000).len(), 0);
    }

    #[test]
    fn bridge_fails_closed_without_a_route() {
        let engine = SimulationEngine::new();
        let params = BridgeParams {
            from_chain: "ethereum".to_string(),
            to_chain: "solana".to_string(),
            token: "USDC".to_string(),
            amount: 1_000_000,
        };
        let result = engine.simulate_bridge(&params);
        assert!(!result.success);
        assert!(result.errors[0].contains("no bridge route"));
    }

    #[test]
    fn bridge_quote_uses_the_registered_fee() {
        let engine = SimulationEngine::new();
        engine.bridge_simulator().set_route(BridgeRoute {
            from_chain: "ethereum".to_string(),
            to_chain: "solana".to_string(),
            fee_bps: 25,
            gas_limit: 200_000,
            enabled: true,
        });
        engine
            .bridge_simulator()
            .set_gas_price("ethereum", 20_000_000_000);

        let params = BridgeParams {
            from_chain: "ethereum".to_string(),
            to_chain: "solana".to_string(),
            token: "USDC".to_string(),
            amount: 1_000_000,
        };
        let result = engine.simulate_bridge(&params);

        assert!(result.success);
        assert_eq!(result.fee_bps, 25);
        assert_eq!(result.expected_output, 1_000_000 - 1_000_000 * 25 / 10_000);
        assert_eq!(result.gas_cost_wei, Some(200_000u128 * 20_000_000_000));
    }

    #[test]
    fn disabled_bridge_route_is_refused() {
        let engine = SimulationEngine::new();
        engine.bridge_simulator().set_route(BridgeRoute {
            from_chain: "ethereum".to_string(),
            to_chain: "solana".to_string(),
            fee_bps: 25,
            gas_limit: 200_000,
            enabled: false,
        });
        let result = engine.simulate_bridge(&BridgeParams {
            from_chain: "ethereum".to_string(),
            to_chain: "solana".to_string(),
            token: "USDC".to_string(),
            amount: 1_000_000,
        });
        assert!(!result.success);
        assert!(result.errors[0].contains("disabled"));
    }

    #[test]
    fn overflow_refuses_to_quote_instead_of_wrapping() {
        let engine = SimulationEngine::new();
        engine.swap_simulator().add_pool(
            "A".to_string(),
            "B".to_string(),
            u128::MAX,
            u128::MAX,
            30,
        );
        // A reserve this large cannot be multiplied by FEE_DENOMINATOR.
        assert!(engine
            .swap_simulator()
            .get_amount_out("A", "B", 1_000_000)
            .is_none());
    }
}
