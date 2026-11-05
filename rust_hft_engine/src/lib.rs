//! TigerWallet high-frequency trading engine.
//!
//! Two components:
//!
//! * `backtesting` — replays historical bars through a strategy and reports
//!   the resulting trades and risk metrics.
//! * `strategy_marketplace` — a registry of strategy listings, reviews and
//!   orders.
//!
//! The backtester previously reported metrics that could not be computed from
//! the data it held, and a private stand-in `chrono` module hardcoded every
//! date it was asked to format. Both are now derived from the inputs.

pub mod backtesting;
pub mod strategy_marketplace;
