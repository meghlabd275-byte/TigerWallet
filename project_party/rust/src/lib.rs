//! Library crate root for the TigerProjectParty Rust client.
//!
//! The full client lives in [`party_api`]; this module re-exports its public
//! surface so callers can `use tigerparty_client::PartyClient;`.

pub mod party_api;

pub use party_api::{PartyClient, PartyError, PARTY_API_DEFAULT_URL};
