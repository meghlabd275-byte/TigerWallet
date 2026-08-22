//! Library crate root for the TigerBots Rust client.
//!
//! The full client lives in [`bot_api`]; this module re-exports its public
//! surface so callers can `use tigerbots_client::BotsClient;`.

pub mod bot_api;

pub use bot_api::{BotError, BotsClient, BOTS_API_DEFAULT_URL};
