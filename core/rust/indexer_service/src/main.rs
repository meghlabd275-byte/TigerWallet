//! Indexer Service Binary
//!
//! Runs the on-chain block indexer. Real block data must come from an EVM RPC
//! endpoint (eth_getBlockByNumber). Provide it via the INDEXER_RPC_URL env
//! var. If no RPC URL is configured, the service refuses to index fabricated
//! blocks and exits with an error (fail-closed) — it never invents fake
//! blocks/miners/hashes.

use indexer_service::Indexer;
use std::env;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("TigerWallet Indexer Service");
    println!("===========================");

    let rpc_url = env::var("INDEXER_RPC_URL").unwrap_or_default();
    if rpc_url.is_empty() {
        eprintln!(
            "INDEXER_RPC_URL is not set; refusing to index fabricated blocks.              Set INDEXER_RPC_URL to an EVM RPC endpoint (e.g.              https://eth.llamarpc.com) to start indexing real blocks."
        );
        std::process::exit(1);
    }

    // The Indexer library currently indexes blocks passed to it via
    // index_block(); a real RPC fetch loop (eth_getBlockByNumber over the
    // configured endpoint) must be wired here to feed real blocks in. Until
    // that loop is implemented we do NOT index any data.
    let _indexer = Indexer::new();
    println!("Indexer initialized against RPC: {}", rpc_url);
    eprintln!(
        "Real block fetch loop not yet wired; no blocks indexed.          Wire eth_getBlockByNumber polling against INDEXER_RPC_URL before use."
    );
    std::process::exit(1);
}
