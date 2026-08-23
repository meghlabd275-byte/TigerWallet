//! TigerWallet White-Label SDK — fail-closed license client + Ed25519 verifier.
//!
//! A white-label product (MasterWallet, UserWallet, Bots, ProjectParty) embeds
//! this SDK to phone home to the TigerWallet SuperAdmin license control plane.
//! The product MUST hold a valid, active, signed license token AND keep its
//! heartbeat current; otherwise `is_alive()` returns false and the product
//! MUST refuse to serve any route (fail-closed).
//!
//! Design:
//! - `LicenseVerifier` performs REAL Ed25519 signature verification of the
//!   license token issued by the control plane. Tampering invalidates the
//!   signature. No fake "always valid".
//! - `LicenseClient` runs a background heartbeat loop. On any failure
//!   (network error, 403, revoked/suspended/halted status, stale heartbeat)
//!   the shared `alive` flag flips to false and stays false until a
//!   successful validate/heartbeat re-establishes liveness. The product
//!   NEVER self-resumes — only a successful SuperAdmin-side resume followed
//!   by a fresh validate() can restore liveness.
//! - `FlagCache` holds the per-fetcher feature flags pulled from SuperAdmin
//!   and refreshed on each heartbeat. `is_fetcher_enabled()` is the
//!   per-fetcher granularity gate.

pub mod verifier;
pub mod client;
pub mod types;

pub use verifier::{LicenseVerifier, SignedLicenseToken, LicenseToken};
pub use client::{LicenseClient, AliveGuard};
pub use types::{FeatureFlag, FetcherGuard};
