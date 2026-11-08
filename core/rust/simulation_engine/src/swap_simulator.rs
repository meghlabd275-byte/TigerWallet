//! Swap Simulator
//!
//! Predicts constant-product (x*y=k) swap outputs with exact integer
//! arithmetic, mirroring the on-chain Uniswap-V2 `getAmountOut` formula:
//!
//! ```text
//! amount_in_with_fee = amount_in * (FEE_DENOMINATOR - fee_bps)
//! numerator          = amount_in_with_fee * reserve_out
//! denominator        = reserve_in * FEE_DENOMINATOR + amount_in_with_fee
//! amount_out         = numerator / denominator
//! ```
//!
//! Reserves and amounts are `u128`, so a large token supply cannot overflow
//! the intermediate product the way `f64` did (and `f64` also silently loses
//! integer precision above 2^53).

use std::collections::HashMap;
use std::sync::RwLock;

/// Denominator for basis-point fees: 10_000 bps == 100%.
pub const FEE_DENOMINATOR: u128 = 10_000;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SwapParams {
    pub token_in: String,
    pub token_out: String,
    pub amount_in: u128,
    /// Slippage tolerance in basis points (50 == 0.5%).
    pub slippage_bps: u64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PoolInfo {
    pub token0: String,
    pub token1: String,
    pub reserve0: u128,
    pub reserve1: u128,
    /// Pool fee in basis points (30 == 0.3%).
    pub fee_bps: u64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SwapResult {
    pub success: bool,
    pub expected_output: u128,
    /// `expected_output` reduced by the caller's slippage tolerance.
    pub min_output: u128,
    pub price_impact_bps: u64,
    pub fee_paid: u128,
    pub errors: Vec<String>,
}

impl SwapResult {
    fn failure(message: impl Into<String>) -> Self {
        Self {
            success: false,
            expected_output: 0,
            min_output: 0,
            price_impact_bps: 0,
            fee_paid: 0,
            errors: vec![message.into()],
        }
    }
}

pub struct SwapSimulator {
    pools: RwLock<HashMap<String, PoolInfo>>,
}

/// Canonical pool key: always `token0-token1` in pool order, so a lookup
/// succeeds for a swap in either direction.
pub fn pool_key(token0: &str, token1: &str) -> String {
    format!("{}-{}", token0, token1)
}

/// Exact constant-product output for a single hop, matching on-chain
/// `getAmountOut`. Returns `None` when the hop is impossible (empty reserve,
/// out-of-range fee) or when the intermediate product would overflow `u128`.
pub fn amount_out(
    reserve_in: u128,
    reserve_out: u128,
    amount_in: u128,
    fee_bps: u64,
) -> Option<u128> {
    if amount_in == 0 || reserve_in == 0 || reserve_out == 0 {
        return None;
    }
    if fee_bps as u128 >= FEE_DENOMINATOR {
        return None;
    }

    let amount_in_with_fee = amount_in.checked_mul(FEE_DENOMINATOR - fee_bps as u128)?;
    let numerator = amount_in_with_fee.checked_mul(reserve_out)?;
    let denominator = reserve_in
        .checked_mul(FEE_DENOMINATOR)?
        .checked_add(amount_in_with_fee)?;
    let out = numerator / denominator;

    if out == 0 || out >= reserve_out {
        return None;
    }
    Some(out)
}

impl SwapSimulator {
    pub fn new() -> Self {
        Self {
            pools: RwLock::new(HashMap::new()),
        }
    }

    /// Register (or replace) a pool. `fee_bps` is the pool's fee in basis points.
    pub fn add_pool(
        &self,
        token0: String,
        token1: String,
        reserve0: u128,
        reserve1: u128,
        fee_bps: u64,
    ) {
        let key = pool_key(&token0, &token1);
        self.pools.write().unwrap().insert(
            key,
            PoolInfo {
                token0,
                token1,
                reserve0,
                reserve1,
                fee_bps,
            },
        );
    }

    /// Look up a pool by either token order, returning the reserves oriented
    /// so that index 0 is `token_in`.
    fn reserves_for(&self, token_in: &str, token_out: &str) -> Option<(u128, u128, u64)> {
        let pools = self.pools.read().unwrap();

        if let Some(pool) = pools.get(&pool_key(token_in, token_out)) {
            return Some((pool.reserve0, pool.reserve1, pool.fee_bps));
        }

        if let Some(pool) = pools.get(&pool_key(token_out, token_in)) {
            return Some((pool.reserve1, pool.reserve0, pool.fee_bps));
        }

        None
    }

    /// Exact output for a single hop, or `None` when the pool is unknown,
    /// a reserve is empty, or the arithmetic would overflow.
    pub fn get_amount_out(
        &self,
        token_in: &str,
        token_out: &str,
        amount_in: u128,
    ) -> Option<(u128, u128)> {
        if amount_in == 0 {
            return None;
        }

        let (reserve_in, reserve_out, fee_bps) = self.reserves_for(token_in, token_out)?;
        if reserve_in == 0 || reserve_out == 0 || fee_bps as u128 >= FEE_DENOMINATOR {
            return None;
        }

        let amount_in_with_fee = amount_in.checked_mul(FEE_DENOMINATOR - fee_bps as u128)?;
        let numerator = amount_in_with_fee.checked_mul(reserve_out)?;
        let denominator = reserve_in
            .checked_mul(FEE_DENOMINATOR)?
            .checked_add(amount_in_with_fee)?;
        let amount_out = numerator / denominator;

        if amount_out == 0 || amount_out >= reserve_out {
            return None;
        }

        let fee_paid = amount_in.checked_mul(fee_bps as u128)? / FEE_DENOMINATOR;
        Some((amount_out, fee_paid))
    }

    /// Price impact of a hop in basis points:
    /// `1 - (amount_out / amount_in) / (reserve_out / reserve_in)`.
    fn price_impact_bps(
        amount_in: u128,
        amount_out: u128,
        reserve_in: u128,
        reserve_out: u128,
    ) -> Option<u64> {
        if amount_in == 0 || reserve_in == 0 {
            return None;
        }
        // The execution rate cannot beat the spot rate, so impact is >= 0.
        let execution_scaled = amount_out.checked_mul(reserve_in)?;
        let spot_scaled = amount_in.checked_mul(reserve_out)?;
        if execution_scaled >= spot_scaled {
            return Some(0);
        }
        let impact = (spot_scaled - execution_scaled).checked_mul(FEE_DENOMINATOR)? / spot_scaled;
        Some(impact as u64)
    }

    /// Simulate a swap against the registered pools.
    pub fn simulate(&self, params: &SwapParams) -> SwapResult {
        if params.token_in == params.token_out {
            return SwapResult::failure("token_in and token_out are the same token");
        }
        if params.amount_in == 0 {
            return SwapResult::failure("amount_in must be greater than zero");
        }
        if params.slippage_bps >= FEE_DENOMINATOR as u64 {
            return SwapResult::failure("slippage_bps must be below 10000");
        }

        let (reserve_in, reserve_out, _fee) =
            match self.reserves_for(&params.token_in, &params.token_out) {
                Some(r) => r,
                None => {
                    return SwapResult::failure(format!(
                        "no pool registered for {}-{}",
                        params.token_in, params.token_out
                    ))
                }
            };

        let (expected_output, fee_paid) =
            match self.get_amount_out(&params.token_in, &params.token_out, params.amount_in) {
                Some(v) => v,
                None => {
                    return SwapResult::failure(
                        "pool has insufficient liquidity or the quote overflows",
                    )
                }
            };

        let price_impact_bps =
            Self::price_impact_bps(params.amount_in, expected_output, reserve_in, reserve_out)
                .unwrap_or(0);

        let min_output = expected_output
            .saturating_mul(FEE_DENOMINATOR - params.slippage_bps as u128)
            / FEE_DENOMINATOR;

        SwapResult {
            success: true,
            expected_output,
            min_output,
            price_impact_bps,
            fee_paid,
            errors: Vec::new(),
        }
    }

    /// Snapshot of every registered pool, for route discovery.
    pub fn pools(&self) -> Vec<PoolInfo> {
        self.pools.read().unwrap().values().cloned().collect()
    }

    /// Reserves for a pair in the requested direction, oriented so index 0 is
    /// `token_in`. Used by route discovery, which walks the pool graph.
    pub fn reserves_of(&self, token_in: &str, token_out: &str) -> Option<(u128, u128, u64)> {
        self.reserves_for(token_in, token_out)
    }
}

impl Default for SwapSimulator {
    fn default() -> Self {
        Self::new()
    }
}
