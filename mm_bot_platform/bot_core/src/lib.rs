// TigerSwap Bot Core - REAL execution plane.
//
// bot_core is the EXECUTION plane. It receives dispatch commands from bot_api
// (the control plane) over an internal HTTP channel on port 8472. Secrets are
// passed via the dispatch request (decrypted by bot_api using AES-GCM, then
// forwarded here). This crate wires up:
//   * dex  - real on-chain Uniswap V2 swaps via ethers + secp256k1 (EIP-155)
//   * cex  - real HMAC-SHA256 signed REST order placement (Binance/OKX/Bybit/Kraken)
//   * store - real PostgreSQL persistence (bot_trades, bot_executions)
//   * strategies - real MarketMaker / Arbitrage / Sniper async dispatch
//   * bot_types - shared data types
//
// There are NO stubs, mocks, or fabricated values: any RPC/DB failure returns
// an error and the HTTP layer surfaces it (fail-closed).

pub mod bot_types;
pub mod cex;
pub mod dex;
pub mod store;
pub mod strategies;

pub use bot_types::*;
pub use cex::*;
pub use dex::*;
pub use store::*;
pub use strategies::*;
