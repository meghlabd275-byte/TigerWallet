//! Math utilities for AMM calculations
//! 
//! Uses Q96 and Q128 fixed-point arithmetic for precision
//! All calculations are self-contained with no external dependencies

use num_bigint::{BigUint, BigInt};
use num_integer::Integer;
use num_traits::{One, ToPrimitive, Zero};
use std::sync::LazyLock;

/// Q96 constant (2^96)
pub static Q96: LazyLock<BigUint> = LazyLock::new(|| BigUint::from(1u64) << 96);
/// Q128 constant (2^128)
pub static Q128: LazyLock<BigUint> = LazyLock::new(|| BigUint::from(1u64) << 128);
/// Q192 constant (2^192)
pub static Q192: LazyLock<BigUint> = LazyLock::new(|| BigUint::from(1u64) << 192);
/// Max uint256
pub static MAX_UINT256: LazyLock<BigUint> = LazyLock::new(|| {
    BigUint::parse_bytes(
        b"115792089237316195423570985008687907853269984665640564039457584007913129639935",
        10,
    )
    .unwrap()
});
/// Max uint160 (for sqrt price)
pub static MAX_UINT160: LazyLock<BigUint> = LazyLock::new(|| {
    BigUint::parse_bytes(b"1461501637330902918203684832716283019655932542976", 10).unwrap()
});
/// Max uint128
pub static MAX_UINT128: LazyLock<BigUint> = LazyLock::new(|| {
    BigUint::parse_bytes(b"340282366920938463463374607431768211455", 10).unwrap()
});

/// Full math library for precise calculations
pub struct FullMath;

impl FullMath {
    /// Multiply and divide with rounding up
    pub fn mul_div_rounding_up(a: &BigUint, b: &BigUint, divisor: &BigUint) -> BigUint {
        if divisor.is_zero() {
            panic!("Division by zero");
        }
        
        let product = a * b;
        let result = &product / divisor;
        if (&product % divisor) > BigUint::zero() {
            result + BigUint::one()
        } else {
            result
        }
    }

    /// Multiply and divide rounding down
    pub fn mul_div_floor(a: &BigUint, b: &BigUint, divisor: &BigUint) -> BigUint {
        if divisor.is_zero() {
            panic!("Division by zero");
        }
        a * b / divisor
    }

    /// Clip ratio to prevent overflow
    pub fn clip_ratio(value: &BigInt, max_val: &BigInt) -> BigInt {
        if value > max_val {
            max_val.clone()
        } else if value < &-max_val.clone() {
            -max_val.clone()
        } else {
            value.clone()
        }
    }

    /// Safe cast to u128 (returns None if overflow)
    pub fn to_u128(value: &BigUint) -> Option<u128> {
        if value > &*MAX_UINT128 {
            None
        } else {
            value.to_u128()
        }
    }
}

/// Bit math utilities for bit operations
pub struct BitMath;

impl BitMath {
    /// Find most significant bit
    pub fn most_significant_bit(x: &BigUint) -> u32 {
        if x.is_zero() {
            return 0;
        }
        
        let mut msb = 0u32;
        let mut x256 = x.clone();
        
        if &x256 >= &(BigUint::from(1u64) << 128) {
            x256 >>= 128;
            msb += 128;
        }
        if &x256 >= &(BigUint::from(1u64) << 64) {
            x256 >>= 64;
            msb += 64;
        }
        if &x256 >= &(BigUint::from(1u64) << 32) {
            x256 >>= 32;
            msb += 32;
        }
        if &x256 >= &(BigUint::from(1u64) << 16) {
            x256 >>= 16;
            msb += 16;
        }
        if &x256 >= &(BigUint::from(1u64) << 8) {
            x256 >>= 8;
            msb += 8;
        }
        if &x256 >= &(BigUint::from(1u64) << 4) {
            x256 >>= 4;
            msb += 4;
        }
        if &x256 >= &(BigUint::from(1u64) << 2) {
            x256 >>= 2;
            msb += 2;
        }
        if &x256 >= &(BigUint::from(1u64) << 1) {
            msb += 1;
        }
        
        msb
    }

    /// Find least significant bit
    pub fn least_significant_bit(x: &BigUint) -> u32 {
        if x.is_zero() {
            return 255;
        }
        
        let mut lsb = 0u32;
        let mut x256 = x.clone();
        
        while x256.is_even() {
            x256 >>= 1;
            lsb += 1;
        }
        
        lsb
    }
}

/// Price math utilities
pub struct PriceMath;

impl PriceMath {
    /// sqrtPriceX96 for a tick, the Uniswap-V3 TickMath algorithm.
    ///
    /// The previous implementation started `ratio` at 2^96 and divided by 2^96
    /// after every multiply, and inverted the ratio for negative ticks. Both
    /// are wrong: the constants below are Q128 values, so the running product
    /// has to start at 2^128 and be shifted right by 128, and it is the
    /// *positive* branch that inverts (through `MAX_UINT256 / ratio`).
    ///
    /// It also passed byte strings still carrying their `0x` prefix to
    /// `BigUint::parse_bytes`, which returns `None` for those, so every call
    /// panicked on `unwrap()`.
    pub fn get_sqrt_price_at_tick(tick: i32) -> BigUint {
        let abs_tick = tick.unsigned_abs();
        let mut ratio = BigUint::one() << 128u32;

        // Each constant is 1 / 1.0001^(2^i) in Q128, straight from the
        // reference implementation.
        const RATIO_MULTIPLIERS: [&[u8]; 20] = [
            b"fffcb933bd6fad37aa2d162d1a594001",
            b"fff97272373d413259a46990580e213a",
            b"fff2e50f5f656932ef12357cf3c7fdcc",
            b"ffe5caca7e10e4e61c3624eaa0941cd0",
            b"ffcb9843d60f6159c9db58835c926644",
            b"ff973b41fa98c081472e6896dfb254c0",
            b"ff2ea16466c96a3843ec78b326b52861",
            b"fe5dee046a99a2a811c461f1969c3053",
            b"fcbe86c7900a88aedcffc83b479aa3a4",
            b"f987a7253ac413176f2b074cf7815e54",
            b"f3392b0822b70005940c7a398e4b70f3",
            b"e7159475a2c29b7443b29c7fa6e889d9",
            b"d097f3bdfd2022b8845ad8f792aa5825",
            b"a9f746462d870fdf8a65dc1f90e061e5",
            b"70d869a156d2a1b890bb3df62baf32f7",
            b"31be135f97d08fd981231505542fcfa6",
            b"09aa508b5b7a84e1c677de54f3e99bc9",
            b"005d6af8dedb81196699c329225ee604",
            b"0002216e584f5fa1ea926041bedfe98",
            b"000048a170391f7dc42444e8fa2",
        ];

        for (bit, encoded) in RATIO_MULTIPLIERS.iter().enumerate() {
            if abs_tick & (1u32 << bit) != 0 {
                let multiplier = BigUint::parse_bytes(encoded, 16)
                    .expect("ratio multipliers are valid hex");
                ratio = (ratio * multiplier) >> 128u32;
            }
        }

        if tick > 0 {
            ratio = &*MAX_UINT256 / ratio;
        }

        // Q128.128 -> Q128.96, rounding up.
        let shifted = &ratio >> 32u32;
        let remainder_mask = (BigUint::one() << 32u32) - BigUint::one();
        if (&ratio & &remainder_mask).is_zero() {
            shifted
        } else {
            shifted + BigUint::one()
        }
    }

    /// The greatest tick whose sqrt price is at or below `sqrt_price_x96`.
    ///
    /// This is the Uniswap-V3 `getTickAtSqrtRatio` algorithm. The previous
    /// version derived the tick from the bit length alone — `(msb - 96) * 2` —
    /// which is off by roughly a factor of 6900: it reported tick 0 for
    /// `2^96 * 1.0001^100` instead of 100, so every position was compared
    /// against the wrong price.
    pub fn get_tick_at_sqrt_price(sqrt_price_x96: &BigUint) -> i32 {
        if sqrt_price_x96.is_zero() {
            return TickMath::MIN_TICK;
        }

        // Clamp to the valid tick range. The reference contract reverts below
        // MIN_SQRT_RATIO; returning an i32 we cannot, and the intermediate
        // estimate can land one tick outside the range, which would be
        // rejected by every caller that validates ticks.
        if *sqrt_price_x96 < Self::get_sqrt_price_at_tick(TickMath::MIN_TICK) {
            return TickMath::MIN_TICK;
        }
        if *sqrt_price_x96 >= Self::get_sqrt_price_at_tick(TickMath::MAX_TICK) {
            return TickMath::MAX_TICK;
        }

        // Work in Q128.128 so the integer part of log2 lands in the high bits.
        let ratio = sqrt_price_x96 << 32u32;
        let msb = BitMath::most_significant_bit(&ratio);

        // Normalise r into [2^127, 2^128).
        let mut r: BigUint = if msb >= 128 {
            &ratio >> (msb - 127)
        } else {
            &ratio << (127 - msb)
        };

        // log2(ratio) - 128, in Q64.64.
        let mut log_2: BigInt = BigInt::from(msb as i64 - 128) << 64u32;

        // Fourteen more bits of the fractional part.
        for i in (50u32..=63u32).rev() {
            r = (&r * &r) >> 127u32;
            let bit = if (&r >> 128u32).is_zero() { 0u32 } else { 1u32 };
            log_2 |= BigInt::from(bit) << i;
            r >>= bit;
        }

        // Change of base to sqrt(1.0001), still Q128.128.
        let log_sqrt10001 = &log_2
            * BigInt::parse_bytes(b"255738958999603826347141", 10)
                .expect("constant is valid decimal");

        let low_offset = BigInt::parse_bytes(b"3402992956809132418596140100660247210", 10)
            .expect("constant is valid decimal");
        let high_offset = BigInt::parse_bytes(
            b"291339464771989622907027621153398088495",
            10,
        )
        .expect("constant is valid decimal");

        let tick_low = ((&log_sqrt10001 - low_offset) >> 128u32)
            .to_i32()
            .unwrap_or(TickMath::MIN_TICK);
        let tick_high = ((&log_sqrt10001 + high_offset) >> 128u32)
            .to_i32()
            .unwrap_or(TickMath::MAX_TICK);

        // `log_sqrt10001` is only accurate to about one tick, so confirm the
        // upper candidate against the exact price.
        if tick_low == tick_high {
            tick_low
        } else if Self::get_sqrt_price_at_tick(tick_high) <= *sqrt_price_x96 {
            tick_high
        } else {
            tick_low
        }
    }

    /// Get amount0 delta
    pub fn get_amount0_delta(
        sqrt_ratio_ax96: &BigUint,
        sqrt_ratio_bx96: &BigUint,
        liquidity: &BigUint,
        round_up: bool,
    ) -> BigUint {
        let (ratio_a, ratio_b) = if sqrt_ratio_ax96 < sqrt_ratio_bx96 {
            (sqrt_ratio_ax96.clone(), sqrt_ratio_bx96.clone())
        } else {
            (sqrt_ratio_bx96.clone(), sqrt_ratio_ax96.clone())
        };

        let numerator1 = liquidity * Q96.clone();
        let numerator2 = &ratio_b - &ratio_a;

        if round_up {
            FullMath::mul_div_rounding_up(
                &FullMath::mul_div_rounding_up(&numerator1, &numerator2, &ratio_b),
                &BigUint::one(),
                &ratio_a,
            )
        } else {
            FullMath::mul_div_floor(&numerator1, &numerator2, &ratio_b) / &ratio_a
        }
    }

    /// Get amount1 delta
    pub fn get_amount1_delta(
        sqrt_ratio_ax96: &BigUint,
        sqrt_ratio_bx96: &BigUint,
        liquidity: &BigUint,
        round_up: bool,
    ) -> BigUint {
        let (ratio_a, ratio_b) = if sqrt_ratio_ax96 < sqrt_ratio_bx96 {
            (sqrt_ratio_ax96.clone(), sqrt_ratio_bx96.clone())
        } else {
            (sqrt_ratio_bx96.clone(), sqrt_ratio_ax96.clone())
        };

        if round_up {
            FullMath::mul_div_rounding_up(liquidity, &(ratio_b - ratio_a), &*Q96)
        } else {
            FullMath::mul_div_floor(liquidity, &(ratio_b - ratio_a), &*Q96)
        }
    }

    /// Get next sqrt price from input amount (swap token0 in)
    pub fn get_next_sqrt_price_from_input(
        sqrt_price_x96: &BigUint,
        liquidity: &BigUint,
        amount_in: &BigUint,
        zero_for_one: bool,
    ) -> BigUint {
        if zero_for_one {
            // Token0 in, token1 out - price goes down
            Self::get_next_sqrt_price_from_amount0_rounding_up(sqrt_price_x96, liquidity, amount_in)
        } else {
            // Token1 in, token0 out - price goes up
            Self::get_next_sqrt_price_from_amount1_rounding_up(sqrt_price_x96, liquidity, amount_in)
        }
    }

    /// Get next sqrt price from amount0 (rounding up)
    pub fn get_next_sqrt_price_from_amount0_rounding_up(
        sqrt_price_x96: &BigUint,
        liquidity: &BigUint,
        amount_in: &BigUint,
    ) -> BigUint {
        let numerator1 = liquidity << 96; // liquidity * Q96
        let denominator = liquidity * sqrt_price_x96 / amount_in + sqrt_price_x96;
        FullMath::mul_div_rounding_up(&numerator1, &BigUint::one(), &denominator)
    }

    /// Get next sqrt price from amount1 (rounding up)
    pub fn get_next_sqrt_price_from_amount1_rounding_up(
        sqrt_price_x96: &BigUint,
        liquidity: &BigUint,
        amount_in: &BigUint,
    ) -> BigUint {
        let product = amount_in * sqrt_price_x96;
        let numerator = product + (liquidity << 96);
        let denominator = liquidity * 10_000_000_000u64; // Liquidity * Q96
        FullMath::mul_div_rounding_up(&numerator, &BigUint::one(), &denominator)
    }
}

/// Tick math utilities for tick calculations
pub struct TickMath;

impl TickMath {
    /// Get tick from sqrt price (public version)
    pub fn get_tick_at_sqrt_price(sqrt_price_x96: &BigUint) -> i32 {
        PriceMath::get_tick_at_sqrt_price(sqrt_price_x96)
    }

    /// Get sqrt price at tick (public version)
    pub fn get_sqrt_price_at_tick(tick: i32) -> BigUint {
        PriceMath::get_sqrt_price_at_tick(tick)
    }

    /// Minimum tick
    pub const MIN_TICK: i32 = -887272;
    /// Maximum tick
    pub const MAX_TICK: i32 = 887272;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mul_div_rounding_up() {
        let a = BigUint::from(10u64);
        let b = BigUint::from(3u64);
        let divisor = BigUint::from(3u64);
        
        let result = FullMath::mul_div_rounding_up(&a, &b, &divisor);
        assert_eq!(result, BigUint::from(10u64));
    }

    /// Values from the Uniswap-V3 reference `TickMath` test vectors.
    #[test]
    fn tick_math_matches_uniswap_v3_reference_vectors() {
        let q96 = BigUint::parse_bytes(b"79228162514264337593543950336", 10).unwrap();

        // tick 0 is exactly 2^96.
        assert_eq!(PriceMath::get_sqrt_price_at_tick(0), q96);

        // The documented extremes of the tick range.
        assert_eq!(
            PriceMath::get_sqrt_price_at_tick(887272),
            BigUint::parse_bytes(
                b"1461446703485210103287273052203988822378723970342",
                10
            )
            .unwrap()
        );
        assert_eq!(
            PriceMath::get_sqrt_price_at_tick(-887272),
            BigUint::parse_bytes(b"4295128739", 10).unwrap()
        );

        // tick 1 is 2^96 * sqrt(1.0001) = 79232123823359799118286999568.
        assert_eq!(
            PriceMath::get_sqrt_price_at_tick(1),
            BigUint::parse_bytes(b"79232123823359799118286999568", 10).unwrap()
        );
        // tick -1 is 2^96 / sqrt(1.0001) = 79224201403219477170569942574.
        assert_eq!(
            PriceMath::get_sqrt_price_at_tick(-1),
            BigUint::parse_bytes(b"79224201403219477170569942574", 10).unwrap()
        );
    }

    #[test]
    fn tick_at_sqrt_price_matches_uniswap_v3_reference_vectors() {
        let q96 = BigUint::parse_bytes(b"79228162514264337593543950336", 10).unwrap();

        assert_eq!(PriceMath::get_tick_at_sqrt_price(&q96), 0);
        assert_eq!(
            PriceMath::get_tick_at_sqrt_price(
                &BigUint::parse_bytes(b"79232123823359799118286999568", 10).unwrap()
            ),
            1
        );
        assert_eq!(
            PriceMath::get_tick_at_sqrt_price(
                &BigUint::parse_bytes(b"79224201403219477170569942574", 10).unwrap()
            ),
            -1
        );
        // Below the minimum price the tick clamps to MIN_TICK.
        assert_eq!(
            PriceMath::get_tick_at_sqrt_price(&BigUint::from(4295128738u64)),
            TickMath::MIN_TICK
        );
    }

    #[test]
    fn tick_round_trips_across_the_whole_range() {
        // Every tick must map to the greatest tick not above its price, so
        // round-tripping through the price returns the same tick.
        for tick in [
            -887272, -100000, -1000, -100, -1, 0, 1, 100, 1000, 100000, 887272,
        ] {
            let price = PriceMath::get_sqrt_price_at_tick(tick);
            assert_eq!(
                PriceMath::get_tick_at_sqrt_price(&price),
                tick,
                "round trip failed for tick {tick}"
            );
        }
    }

    #[test]
    fn test_msb() {
        assert_eq!(
            BitMath::most_significant_bit(&(BigUint::from(1u64) << 10u32)),
            10
        );
        assert_eq!(BitMath::most_significant_bit(&BigUint::from(1u64)), 0);
    }

    #[test]
    fn test_lsb() {
        assert_eq!(BitMath::least_significant_bit(&BigUint::from(8u64)), 3);
        assert_eq!(BitMath::least_significant_bit(&BigUint::from(1u64)), 0);
    }

    #[test]
    fn test_sqrt_price_at_tick() {
        let sqrt_price = PriceMath::get_sqrt_price_at_tick(0);
        assert_eq!(sqrt_price, *Q96);
    }

    #[test]
    fn test_tick_round_trip() {
        let original_tick = 1000;
        let sqrt_price = PriceMath::get_sqrt_price_at_tick(original_tick);
        let recovered_tick = PriceMath::get_tick_at_sqrt_price(&sqrt_price);
        assert!(recovered_tick >= original_tick - 1 && recovered_tick <= original_tick + 1);
    }
}
