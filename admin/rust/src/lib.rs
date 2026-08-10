//! TigerAdmin Rust Library

pub mod models;
pub mod handlers;
pub mod db;
pub mod auth;
pub mod error;

pub use models::*;
pub use handlers::*;
pub use db::*;
pub use auth::*;
pub use error::*;

use db::DbPool;
use auth::AuthState;

#[derive(Clone)]
pub struct AppState {
    pub db: DbPool,
    pub auth: AuthState,
}
