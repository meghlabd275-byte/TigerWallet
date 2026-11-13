//! Blockchain integration module for MasterWallet
//! Provides real blockchain RPC connections for multi-chain support

pub mod contracts;
pub mod rpc;
pub mod signer;

pub use contracts::*;
pub use rpc::*;
pub use signer::*;
