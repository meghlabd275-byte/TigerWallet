// TigerSwap MM Bot core library — exposes the 18 bot-type definitions and the
// advanced strategy implementations as a reusable library so the bot_api
// control plane and other consumers can reference real strategy logic.
//
// Previously bot_types.rs and strategies/mod.rs were orphaned (no `mod`
// declaration) and never compiled. They are now wired as a library crate so
// the strategy logic is live, type-checked, and maintainable.

pub mod bot_types;
pub mod strategies;

pub use bot_types::*;
pub use strategies::*;
