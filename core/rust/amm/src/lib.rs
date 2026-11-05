//! TigerSwap AMM Engine - Production-Ready Concentrated Liquidity
//! 
//! COMPLETELY SELF-CONTAINED implementation with:
//! - Constant product AMM (x*y=k)
//! - Concentrated liquidity (Uniswap V3 style)
//! - Stable swaps
//! - Fee tiers with proper growth calculations
//! - Tick management with crossing logic
//! - Position management for liquidity providers
//! - Full swap execution with price impact
//! 
//! NO external dependencies for core logic.

mod math;
mod pool;
mod factory;
mod swap;
mod tick;
mod position;

pub use math::{FullMath, BitMath, PriceMath, Q96, Q128, MAX_UINT256, TickMath};
pub use pool::{PoolCore, PoolConfig, SwapResult, FEE_TIERS, PoolState};
pub use factory::{AMMFactory, SwapRouter};
pub use swap::{SwapExecutor, SwapParams};
pub use tick::{Tick, TickMap};
pub use position::{Position, PositionId};

/// Fee tier constants in basis points (with 6 decimal precision)
pub mod fee {
    /// 0.01% - stable pairs (USDC/USDT, etc.)
    pub const STABLE: u32 = 100;
    /// 0.05% - low volatility pairs
    pub const LOW: u32 = 500;
    /// 0.30% - standard pairs (ETH/USDC)
    pub const MEDIUM: u32 = 3000;
    /// 1.00% - exotic pairs
    pub const HIGH: u32 = 10000;
    /// Custom fee tier
    pub const CUSTOM: u32 = 0;
}

/// Tick spacing for fee tiers
pub mod tick_spacing {
    pub const STABLE: i32 = 1;
    pub const LOW: i32 = 10;
    pub const MEDIUM: i32 = 60;
    pub const HIGH: i32 = 200;
}

/// Maximum tick value
pub const MAX_TICK: i32 = 887272;
pub const MIN_TICK: i32 = -887272;

/// Convert fee basis points to fee tier
pub fn fee_tier_from_bps(bps: u32) -> u32 {
    match bps {
        100 => fee::STABLE,
        500 => fee::LOW,
        3000 => fee::MEDIUM,
        10000 => fee::HIGH,
        _ => fee::CUSTOM,
    }
}

/// Convert fee tier to human readable percentage
pub fn fee_tier_to_percentage(tier: u32) -> f64 {
    tier as f64 / 1000000.0 * 100.0
}

#[cfg(test)]
mod tests {
    use super::*;
    use num_bigint::BigUint;
    use num_traits::Zero;

    #[test]
    fn test_sqrt_price_from_tick() {
        let sqrt_price = PriceMath::get_sqrt_price_at_tick(0);
        assert_eq!(sqrt_price, *Q96);
    }

    #[test]
    fn test_tick_from_sqrt_price() {
        let sqrt_price = PriceMath::get_sqrt_price_at_tick(100);
        let tick = PriceMath::get_tick_at_sqrt_price(&sqrt_price);
        assert!(tick >= 99 && tick <= 101);
    }

    #[test]
    fn test_pool_creation() {
        let config = PoolConfig {
            token0: "0xA".to_string(),
            token1: "0xB".to_string(),
            fee: fee::MEDIUM,
            tick_spacing: tick_spacing::MEDIUM as u32,
            sqrt_price_x96: Some(Q96.clone()),
        };
        
        let factory = AMMFactory::new();
        let pool = factory.create_pool(config);
        
        assert!(pool.is_ok());
    }

    #[test]
    fn test_fee_tier_conversion() {
        assert_eq!(fee_tier_to_percentage(fee::STABLE), 0.01);
        assert_eq!(fee_tier_to_percentage(fee::LOW), 0.05);
        assert_eq!(fee_tier_to_percentage(fee::MEDIUM), 0.3);
        assert_eq!(fee_tier_to_percentage(fee::HIGH), 1.0);
    }

    fn position(tick_lower: i32, tick_upper: i32, liquidity: u64) -> Position {
        Position::new(
            PositionId::new(1),
            "0xowner".to_string(),
            "0xA".to_string(),
            "0xB".to_string(),
            tick_lower,
            tick_upper,
            BigUint::from(liquidity),
        )
    }

    #[test]
    fn position_holds_only_token0_below_its_range() {
        let pos = position(-100, 100, 1_000_000);
        // A price below tick_lower is entirely token0.
        let below = PriceMath::get_sqrt_price_at_tick(-200);
        let (amount0, amount1) = pos.get_value_at_sqrt_price(&below);
        assert!(!amount0.is_zero());
        assert!(amount1.is_zero());
    }

    #[test]
    fn position_holds_only_token1_above_its_range() {
        let pos = position(-100, 100, 1_000_000);
        // A price above tick_upper is entirely token1.
        let above = PriceMath::get_sqrt_price_at_tick(200);
        let (amount0, amount1) = pos.get_value_at_sqrt_price(&above);
        assert!(amount0.is_zero());
        assert!(!amount1.is_zero());
    }

    #[test]
    fn position_holds_both_tokens_inside_its_range() {
        let pos = position(-100, 100, 1_000_000);
        let inside = PriceMath::get_sqrt_price_at_tick(0);
        let (amount0, amount1) = pos.get_value_at_sqrt_price(&inside);
        // The price sits exactly in the middle of a symmetric range, so both
        // sides are non-zero and equal to the full-range halves.
        assert!(!amount0.is_zero());
        assert!(!amount1.is_zero());
    }

    #[test]
    fn position_value_depends_on_its_liquidity() {
        // The old get_value returned its arguments unchanged, so liquidity
        // never affected the result.
        let small = position(-100, 100, 1_000);
        let large = position(-100, 100, 1_000_000);
        let price = PriceMath::get_sqrt_price_at_tick(0);

        let (small0, small1) = small.get_value_at_sqrt_price(&price);
        let (large0, large1) = large.get_value_at_sqrt_price(&price);

        assert!(large0 > small0);
        assert!(large1 > small1);
    }

    #[test]
    fn an_empty_position_is_worth_nothing() {
        let pos = position(-100, 100, 0);
        let price = PriceMath::get_sqrt_price_at_tick(0);
        let (amount0, amount1) = pos.get_value_at_sqrt_price(&price);
        assert!(amount0.is_zero());
        assert!(amount1.is_zero());
    }

    #[test]
    fn test_tick_range() {
        assert!(MAX_TICK > 0);
        assert!(MIN_TICK < 0);
        assert_eq!(MIN_TICK, -MAX_TICK);
    }
}
