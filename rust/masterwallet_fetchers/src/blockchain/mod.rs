//! Blockchain integration module for MasterWallet
//! Provides real blockchain RPC connections for multi-chain support

pub mod rpc;
pub mod contracts;
pub mod signer;

pub use rpc::*;
pub use contracts::*;
pub use signer::*;
