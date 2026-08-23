//! Shared types: feature flags + the per-fetcher guard helper.

use serde::{Deserialize, Serialize};

/// A per-fetcher feature flag pulled from the SuperAdmin control plane. A flag
/// with `fetcher == "*"` disables the whole product; a named fetcher flag
/// disables just that fetcher. Absent flags default to enabled.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureFlag {
    pub id: String,
    pub wl_client_id: String,
    pub product: String,
    pub fetcher: String,
    pub enabled: bool,
}

/// FetcherGuard evaluates a fetcher name against a flag set. Returns true when
/// the fetcher is permitted (no disabling flag present).
pub struct FetcherGuard;

impl FetcherGuard {
    /// Returns true if the fetcher may serve. A whole-product disable ('*')
    /// or a specific fetcher disable makes this false.
    pub fn is_enabled(flags: &[FeatureFlag], product: &str, fetcher: &str) -> bool {
        for f in flags {
            if f.product != product {
                continue;
            }
            if f.fetcher == "*" && !f.enabled {
                return false;
            }
            if f.fetcher == fetcher && !f.enabled {
                return false;
            }
        }
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn whole_product_disable_blocks_all_fetchers() {
        let flags = vec![FeatureFlag {
            id: "1".into(), wl_client_id: "wl".into(), product: "user_wallet".into(),
            fetcher: "*".into(), enabled: false,
        }];
        assert!(!FetcherGuard::is_enabled(&flags, "user_wallet", "balances"));
        assert!(!FetcherGuard::is_enabled(&flags, "user_wallet", "transactions"));
    }

    #[test]
    fn specific_fetcher_disable_blocks_only_that_one() {
        let flags = vec![FeatureFlag {
            id: "1".into(), wl_client_id: "wl".into(), product: "user_wallet".into(),
            fetcher: "swap_quote".into(), enabled: false,
        }];
        assert!(!FetcherGuard::is_enabled(&flags, "user_wallet", "swap_quote"));
        assert!(FetcherGuard::is_enabled(&flags, "user_wallet", "balances"));
    }

    #[test]
    fn absent_flags_default_enabled() {
        assert!(FetcherGuard::is_enabled(&[], "user_wallet", "balances"));
    }
}
