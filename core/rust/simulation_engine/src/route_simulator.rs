//! Route Simulator
//!
//! Discovers swap routes by breadth-first search over the pools registered
//! with [`SwapSimulator`], then prices each hop with the exact
//! constant-product formula. Outputs are cumulative, so a two-hop route is
//! quoted with the first hop's proceeds feeding the second.
//!
//! Nothing here is precomputed or hardcoded: the graph is whatever pools the
//! caller registered, and a route only exists if every hop in it does.

use std::collections::{HashSet, VecDeque};

use crate::swap_simulator::{amount_out, SwapSimulator};

/// Gas attributed to one swap hop. Hops differ by pool implementation, but a
/// router needs a per-hop figure to rank routes; callers that need exact gas
/// should override the returned `total_gas`.
pub const GAS_PER_HOP: u128 = 150_000;

/// Maximum number of hops considered for a single route.
pub const MAX_HOPS: usize = 3;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RouteStep {
    pub from_token: String,
    pub to_token: String,
    /// `token0-token1` in pool order.
    pub pool: String,
    pub amount_in: u128,
    pub amount_out: u128,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Route {
    pub steps: Vec<RouteStep>,
    /// Output after the final hop.
    pub total_output: u128,
    pub total_gas: u128,
}

pub struct RouteSimulator {
    swap: std::sync::Arc<SwapSimulator>,
}

impl RouteSimulator {
    /// Build a router over the given swap simulator's pool set.
    pub fn new(swap: std::sync::Arc<SwapSimulator>) -> Self {
        Self { swap }
    }

    /// Every token that appears in at least one registered pool.
    fn tokens(&self) -> Vec<String> {
        let mut seen = HashSet::new();
        for pool in self.swap.pools() {
            seen.insert(pool.token0);
            seen.insert(pool.token1);
        }
        seen.into_iter().collect()
    }

    /// Adjacent tokens reachable from `token` in one hop.
    fn neighbours(&self, token: &str) -> Vec<(String, String)> {
        let mut out = Vec::new();
        for pool in self.swap.pools() {
            if pool.token0 == token {
                out.push((pool.token1.clone(), format!("{}-{}", pool.token0, pool.token1)));
            } else if pool.token1 == token {
                out.push((pool.token0.clone(), format!("{}-{}", pool.token0, pool.token1)));
            }
        }
        out
    }

    /// Price a token path hop by hop, returning `None` if any hop is unpriced.
    fn price_path(&self, path: &[String], amount_in: u128) -> Option<Vec<RouteStep>> {
        let mut steps = Vec::with_capacity(path.len().saturating_sub(1));
        let mut current = amount_in;

        for pair in path.windows(2) {
            let (from, to) = (&pair[0], &pair[1]);
            let (reserve_in, reserve_out, fee_bps) = self.swap.reserves_of(from, to)?;
            let out = amount_out(reserve_in, reserve_out, current, fee_bps)?;

            steps.push(RouteStep {
                from_token: from.clone(),
                to_token: to.clone(),
                pool: format!("{}-{}", from, to),
                amount_in: current,
                amount_out: out,
            });
            current = out;
        }

        if steps.is_empty() {
            return None;
        }
        Some(steps)
    }

    /// All routes from `from` to `to` up to [`MAX_HOPS`], priced and ordered
    /// best output first.
    pub fn get_routes(&self, from: &str, to: &str, amount_in: u128) -> Vec<Route> {
        if from == to || amount_in == 0 {
            return Vec::new();
        }

        let mut routes = Vec::new();
        // Breadth-first over token paths; `visited` is per-path so a token may
        // appear again on a different path.
        let mut queue: VecDeque<Vec<String>> = VecDeque::new();
        queue.push_back(vec![from.to_string()]);

        while let Some(path) = queue.pop_front() {
            if path.len() > MAX_HOPS {
                continue;
            }
            let last = path.last().cloned().unwrap_or_default();

            if last == to {
                if let Some(steps) = self.price_path(&path, amount_in) {
                    let total_output = steps.last().map(|s| s.amount_out).unwrap_or(0);
                    routes.push(Route {
                        total_gas: GAS_PER_HOP * steps.len() as u128,
                        steps,
                        total_output,
                    });
                }
                continue;
            }

            for (next, _) in self.neighbours(&last) {
                if path.contains(&next) {
                    continue;
                }
                let mut extended = path.clone();
                extended.push(next);
                queue.push_back(extended);
            }
        }

        routes.sort_by(|a, b| {
            b.total_output
                .cmp(&a.total_output)
                .then_with(|| a.steps.len().cmp(&b.steps.len()))
        });
        routes
    }

    /// Best-priced route, or `None` when no path exists.
    pub fn find_optimal_route(&self, from: &str, to: &str, amount_in: u128) -> Option<Route> {
        self.get_routes(from, to, amount_in).into_iter().next()
    }

    /// Distinct tokens the router can currently reach from `from`.
    pub fn reachable_tokens(&self, from: &str) -> Vec<String> {
        let mut seen: HashSet<String> = HashSet::new();
        let mut queue: VecDeque<String> = VecDeque::new();
        queue.push_back(from.to_string());
        seen.insert(from.to_string());

        while let Some(token) = queue.pop_front() {
            for (next, _) in self.neighbours(&token) {
                if seen.insert(next.clone()) {
                    queue.push_back(next);
                }
            }
        }

        let mut out: Vec<String> = seen.into_iter().collect();
        out.sort();
        out
    }

    /// Every token in the pool graph.
    pub fn known_tokens(&self) -> Vec<String> {
        let mut tokens = self.tokens();
        tokens.sort();
        tokens
    }
}
