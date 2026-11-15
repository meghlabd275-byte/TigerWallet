//! Swap execution logic

use super::pool::{PoolCore, SwapResult};
use num_bigint::BigUint;
use num_traits::ToPrimitive;

/// Swap parameters
#[derive(Debug, Clone)]
pub struct SwapParams {
    pub token_in: String,
    pub token_out: String,
    pub amount_in: BigUint,
    pub min_amount_out: BigUint,
    pub recipient: String,
    pub deadline: u64,
}

/// Swap executor for executing swaps across pools
pub struct SwapExecutor {
    pools: Vec<PoolCore>,
}

impl SwapExecutor {
    /// Create a new swap executor
    pub fn new(pools: Vec<PoolCore>) -> Self {
        Self { pools }
    }

    /// Execute a swap
    pub fn execute(&self, params: &SwapParams) -> Result<BigUint, String> {
        if self.pools.is_empty() {
            return Err("No pools available".to_string());
        }

        let mut remaining = params.amount_in.clone();
        let mut total_out = BigUint::from(0u64);

        for pool in &self.pools {
            if remaining == BigUint::from(0u64) {
                break;
            }

            let zero_for_one = pool.token0().to_lowercase() == params.token_in.to_lowercase();
            let result = pool.swap(&remaining, zero_for_one, None)?;
            total_out += result.amount_out.clone();
            remaining = result.amount_out;
        }

        if total_out < params.min_amount_out {
            return Err("Slippage tolerance exceeded".to_string());
        }

        Ok(total_out)
    }

    /// Calculate price impact as a percentage.
    ///
    /// Price impact is how far the average execution price sits below the
    /// marginal price before the trade. It is derived from the pool's own
    /// reserves and fee, so it cannot be reported as zero for a trade that
    /// actually moves the price.
    ///
    /// Returns `None` when the pools cannot price the trade — an empty pool,
    /// a zero input, or a route that produces nothing. A caller that receives
    /// `None` must surface an error rather than print 0%: "we could not
    /// measure this" and "this trade has no impact" are different statements.
    pub fn calculate_price_impact(&self, params: &SwapParams) -> Option<f64> {
        let pool = self.pools.first()?;

        let zero_for_one = pool.token0().eq_ignore_ascii_case(&params.token_in);
        let spot_price = pool.spot_price_from_reserves()?;
        if spot_price <= 0.0 {
            return None;
        }

        let amount_out = pool.quote(&params.amount_in, zero_for_one).ok()?;
        let amount_in = params.amount_in.to_f64()?;
        let amount_out = amount_out.to_f64()?;
        if amount_in <= 0.0 || amount_out <= 0.0 {
            return None;
        }

        // What the trade would return at the marginal price, in the output
        // token. `spot_price` is token1 per token0, so the two directions
        // invert it.
        let fair_amount_out = if zero_for_one {
            amount_in * spot_price
        } else {
            amount_in / spot_price
        };
        if fair_amount_out <= 0.0 {
            return None;
        }

        // A constant-product swap always returns less than the marginal rate,
        // because it pays the fee and moves the price against itself. Express
        // that shortfall as a percentage of the fair output, which is positive
        // in both directions.
        let shortfall = (fair_amount_out - amount_out) / fair_amount_out * 100.0;
        if shortfall < 0.0 {
            // Output above the marginal rate is impossible for this curve, so
            // the inputs are degenerate. Refuse rather than report a negative
            // impact.
            return None;
        }

        Some(shortfall)
    }

    /// Spot price of token0 in token1, from the pool's reserves.
    pub fn get_spot_price(&self) -> f64 {
        self.pools
            .first()
            .and_then(|p| p.spot_price_from_reserves())
            .unwrap_or(0.0)
    }

    /// Get pools
    pub fn pools(&self) -> &[PoolCore] {
        &self.pools
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::math::Q96;

    #[test]
    fn test_swap_params() {
        let params = SwapParams {
            token_in: "0xA".to_string(),
            token_out: "0xB".to_string(),
            amount_in: BigUint::from(1000u64),
            min_amount_out: BigUint::from(900u64),
            recipient: "0xRecipient".to_string(),
            deadline: 0,
        };
        
        assert_eq!(params.amount_in, BigUint::from(1000u64));
    }

    /// A pool with 1_000_000 of token0 against 1_000_000 of token1 (spot price
    /// 1.0) and the 0.30% fee tier.
    fn executor() -> SwapExecutor {
        let pool = PoolCore::new(
            "0xA".to_string(),
            "0xB".to_string(),
            3000,
            60,
            Q96.clone(),
        );
        pool.add_liquidity(BigUint::from(1_000_000u64), BigUint::from(1_000_000u64));
        SwapExecutor::new(vec![pool])
    }

    fn params(amount_in: u64) -> SwapParams {
        SwapParams {
            token_in: "0xA".to_string(),
            token_out: "0xB".to_string(),
            amount_in: BigUint::from(amount_in),
            min_amount_out: BigUint::from(0u64),
            recipient: "0xRecipient".to_string(),
            deadline: 0,
        }
    }

    #[test]
    fn price_impact_is_non_zero_for_a_trade_that_moves_the_price() {
        // The previous implementation multiplied the input by the spot price
        // and divided it back out, so it always returned exactly 0.
        let impact = executor()
            .calculate_price_impact(&params(10_000))
            .expect("a 1% trade against a funded pool must be measurable");

        assert!(impact > 0.0, "expected positive impact, got {impact}");
    }

    #[test]
    fn price_impact_grows_with_trade_size() {
        let small = executor()
            .calculate_price_impact(&params(1_000))
            .expect("small trade measurable");
        let large = executor()
            .calculate_price_impact(&params(100_000))
            .expect("large trade measurable");

        assert!(
            large > small,
            "a 100x larger trade must move the price further: {large} vs {small}"
        );
    }

    #[test]
    fn price_impact_accounts_for_the_fee() {
        // The 0.30% fee is paid on every swap, so even a trade small enough
        // to barely move the reserves cannot execute at the marginal price.
        let impact = executor()
            .calculate_price_impact(&params(1_000))
            .expect("a small trade is still measurable");

        assert!(
            impact > 0.29,
            "the fee tier alone implies ~0.30% impact, got {impact}"
        );
    }

    #[test]
    fn price_impact_is_measured_for_both_directions() {
        let mut reverse = params(10_000);
        reverse.token_in = "0xB".to_string();
        reverse.token_out = "0xA".to_string();

        let impact = executor()
            .calculate_price_impact(&reverse)
            .expect("a reverse-direction trade must be measurable");

        assert!(impact > 0.0, "expected positive impact, got {impact}");
    }

    #[test]
    fn price_impact_fails_closed_without_liquidity() {
        let empty = SwapExecutor::new(vec![PoolCore::new(
            "0xA".to_string(),
            "0xB".to_string(),
            3000,
            60,
            Q96.clone(),
        )]);

        assert!(
            empty.calculate_price_impact(&params(1_000)).is_none(),
            "an unfunded pool must not report an impact figure"
        );
    }

    #[test]
    fn price_impact_fails_closed_for_a_zero_amount() {
        assert!(
            executor().calculate_price_impact(&params(0)).is_none(),
            "a zero-value trade must not report an impact figure"
        );
    }

    #[test]
    fn price_impact_fails_closed_without_pools() {
        let none = SwapExecutor::new(vec![]);

        assert!(none.calculate_price_impact(&params(1_000)).is_none());
    }

    #[test]
    fn quote_matches_the_executed_swap() {
        let executor = executor();
        let quoted = executor.pools[0]
            .quote(&BigUint::from(10_000u64), true)
            .expect("quote against a funded pool");

        // Execute the same swap and confirm the quote was honest.
        let executed = executor
            .execute(&params(10_000))
            .expect("execution against a funded pool");

        assert_eq!(quoted, executed, "a quote must equal the swap it predicts");
    }
}
