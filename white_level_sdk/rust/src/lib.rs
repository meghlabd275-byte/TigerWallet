//! White Level SDK - External Hosting Connection Library
//! 
//! This SDK enables White Level products to connect to TigerWallet Super Admin
//! from independently hosted environments.

pub mod client;
pub mod config;
pub mod connection;
pub mod errors;
pub mod fetcher;
pub mod permissions;
pub mod types;

pub use client::WhiteLevelClient;
pub use config::Config;
pub use connection::ConnectionManager;
pub use errors::{SdkError, Result};
pub use fetcher::FetcherManager;
pub use permissions::PermissionManager;
pub use types::*;
