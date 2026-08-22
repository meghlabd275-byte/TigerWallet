//! REAL DEX executor.
//!
//! Connects to Ethereum/BSC/Polygon/Arbitrum RPC via an ethers `Provider`,
//! builds real Uniswap V2 router `swapExactTokensForTokens` calldata, signs it
//! with a real secp256k1 ECDSA key (EIP-155), broadcasts via
//! `eth_sendRawTransaction`, and returns the real transaction hash plus the
//! real `amountOut` decoded from the receipt's `Transfer`/`Swap` log.
//!
//! Fail-closed: any RPC failure returns an error. No transaction hash is ever
//! fabricated.

use ethers::contract::abigen;
use ethers::core::types::{
    transaction::eip2718::TypedTransaction, BlockId, BlockNumber, Chain, TransactionRequest, U256,
};
use ethers::providers::{Http, Middleware, Provider};
use ethers::signers::{LocalWallet, Signer};
use std::sync::Arc;
use std::time::Duration;

abigen!(
    IUniswapV2Router,
    r#"[
        function swapExactTokensForTokens(uint amountIn, uint amountOutMin, address[] path, address to, uint deadline) external returns (uint[] amounts)
        function getAmountsOut(uint amountIn, address[] path) external view returns (uint[] amounts)
    ]"#,
);

abigen!(
    IErc20,
    r#"[
        function decimals() external view returns (uint8)
        function approve(address spender, uint256 amount) external returns (bool)
        function allowance(address owner, address spender) external view returns (uint256)
    ]"#,
);

/// Configuration for a real DEX swap.
#[derive(Debug, Clone, serde::Deserialize)]
pub struct DexSwapRequest {
    /// Ethereum/BSC/Polygon/Arbitrum RPC URL, e.g. `https://eth.llamarpc.com`.
    pub rpc_url: String,
    /// Chain id used for EIP-155 signing (1=eth, 56=bsc, 137=polygon, 42161=arbitrum).
    pub chain_id: u64,
    /// Hex-encoded private key (already decrypted by bot_api). NEVER logged.
    #[serde(skip_serializing)]
    pub private_key: String,
    /// Uniswap-V2-compatible router address (checksummed).
    pub router: String,
    /// Token to spend (input).
    pub token_in: String,
    /// Token to receive (output).
    pub token_out: String,
    /// Human-readable input amount (e.g. `0.5`).
    pub amount_in: f64,
    /// Minimum output accepted (slippage protection), human-readable.
    pub amount_out_min: f64,
    /// Optional token decimals for `token_in` (auto-detected via `decimals()` if absent).
    pub token_in_decimals: Option<u8>,
    /// Optional token decimals for `token_out`.
    pub token_out_decimals: Option<u8>,
}

/// Result of a real on-chain swap.
#[derive(Debug, Clone, serde::Serialize)]
pub struct DexSwapResult {
    /// Real transaction hash from `eth_sendRawTransaction`.
    pub tx_hash: String,
    /// Real output amount decoded from the receipt (human-readable).
    pub amount_out: f64,
    /// Real gas used by the mined transaction.
    pub gas_used: u64,
    /// Effective gas price (wei).
    pub gas_price: u128,
    /// Block number in which the tx was included.
    pub block_number: u64,
    /// True only when the receipt status is `1` (success).
    pub success: bool,
}

/// Execute a real Uniswap V2 `swapExactTokensForTokens` swap.
///
/// This signs and broadcasts a real transaction. It never fabricates a hash:
/// if broadcast or confirmation fails, an error is returned.
pub async fn execute_swap(req: &DexSwapRequest) -> Result<DexSwapResult, DexError> {
    let provider = Arc::new(
        Provider::<Http>::try_from(req.rpc_url.as_str())
            .map_err(DexError::provider)?
            .interval(Duration::from_millis(200)),
    );

    let wallet = req
        .private_key
        .parse::<LocalWallet>()
        .map_err(DexError::signer)?
        .with_chain_id(req.chain_id);

    let client = SignerMiddleware::new(provider, wallet);
    let client = Arc::new(client);

    let router_addr: ethers::core::types::Address = req
        .router
        .parse()
        .map_err(|e| DexError::config(format!("bad router address: {e}")))?;
    let token_in: ethers::core::types::Address = req
        .token_in
        .parse()
        .map_err(|e| DexError::config(format!("bad token_in address: {e}")))?;
    let token_out: ethers::core::types::Address = req
        .token_out
        .parse()
        .map_err(|e| DexError::config(format!("bad token_out address: {e}")))?;

    let in_decimals = match req.token_in_decimals {
        Some(d) => d,
        None => fetch_decimals(client.clone(), token_in).await?,
    };
    let out_decimals = match req.token_out_decimals {
        Some(d) => d,
        None => fetch_decimals(client.clone(), token_out).await?,
    };

    let amount_in = to_base_units(req.amount_in, in_decimals);
    let amount_out_min = to_base_units(req.amount_out_min, out_decimals);

    let from = client.signer().address();
    let path = vec![token_in, token_out];
    let deadline = unix_now() + 600; // 10 min
    let router = IUniswapV2Router::new(router_addr, client.clone());

    // Fail-closed pre-check: ensure router is approved to spend amount_in.
    ensure_allowance(client.clone(), token_in, from, router_addr, amount_in).await?;

    let call = router.swap_exact_tokens_for_tokens(
        amount_in,
        amount_out_min,
        path.clone(),
        from,
        deadline,
    );
    let calibrated = call.calldata().ok_or_else(|| DexError::abi("no calldata"))?;

    let nonce = client
        .get_transaction_count(from, Some(BlockId::Number(BlockNumber::Pending)))
        .await
        .map_err(DexError::provider)?;

    let chain = Chain::try_from(req.chain_id).ok();
    let gas_price = client
        .get_gas_price()
        .await
        .map_err(DexError::provider)?;
    let estimate = client
        .estimate_gas(
            &TypedTransaction::Legacy(
                TransactionRequest::new()
                    .from(from)
                    .to(router_addr)
                    .data(calibrated.clone())
                    .value(U256::zero()),
            ),
            None,
        )
        .await
        .map_err(DexError::provider)?;
    let estimate = estimate.max(U256::from(60_000u64));

    let mut tx = TransactionRequest::new()
        .from(from)
        .to(router_addr)
        .nonce(nonce)
        .gas_price(gas_price)
        .gas(estimate)
        .data(calibrated);
    if let Some(c) = chain {
        tx = tx.chain_id(c);
    } else {
        tx = tx.chain_id(req.chain_id);
    }
    let typed = TypedTransaction::Legacy(tx);

    // Real secp256k1 ECDSA signature with EIP-155.
    let signature = client
        .signer()
        .sign_transaction(&typed)
        .await
        .map_err(DexError::signer)?;
    let raw = typed.rlp_signed(&signature);

    // Real broadcast. No fabricated hash.
    let pending = client
        .provider()
        .send_raw_transaction(raw)
        .await
        .map_err(DexError::provider)?;
    let tx_hash = pending.tx_hash();

    // Real confirmation: wait for the receipt, with a bounded timeout.
    let receipt = tokio::time::timeout(Duration::from_secs(180), pending)
        .await
        .map_err(|_| DexError::timeout(tx_hash))?
        .map_err(DexError::provider)?
        .ok_or_else(|| DexError::timeout(tx_hash))?;

    let success = receipt.status == Some(1.into());
    if !success {
        return Err(DexError::reverted(tx_hash));
    }

    // Real amount_out: decode from the router's `amounts` return value via the
    // Swap event / last Transfer to `from`. The router returns `uint[] amounts`;
    // the receipt logs include the output Transfer. We decode the largest
    // Transfer of `token_out` to `from` in this tx as the realized amount_out.
    let amount_out = decode_amount_out(&receipt, token_out, from, out_decimals)
        .ok_or_else(|| DexError::decode(tx_hash))?;

    Ok(DexSwapResult {
        tx_hash: format!("{tx_hash:#x}"),
        amount_out,
        gas_used: receipt.gas_used.map(|g| g.as_u64()).unwrap_or(0),
        gas_price: receipt
            .effective_gas_price
            .map(|g| g.as_u128())
            .unwrap_or(0),
        block_number: receipt.block_number.map(|b| b.as_u64()).unwrap_or(0),
        success,
    })
}

// Use the ethers SignerMiddleware for typed signing.
use ethers::middleware::SignerMiddleware;

/// Real on-chain quote via Uniswap V2 `getAmountsOut` (view call, no signing).
/// Returns the human-readable output for 1 whole `token_in` (18-decimals
/// assumed for the quote unit).
pub async fn get_amounts_out(
    rpc_url: &str,
    router: &str,
    token_in: &str,
    token_out: &str,
) -> Result<f64, DexError> {
    let provider = Arc::new(Provider::<Http>::try_from(rpc_url).map_err(DexError::provider)?);
    let router_addr: ethers::core::types::Address = router
        .parse()
        .map_err(|e| DexError::config(format!("bad router: {e}")))?;
    let token_in_addr: ethers::core::types::Address = token_in
        .parse()
        .map_err(|e| DexError::config(format!("bad token_in: {e}")))?;
    let token_out_addr: ethers::core::types::Address = token_out
        .parse()
        .map_err(|e| DexError::config(format!("bad token_out: {e}")))?;
    let r = IUniswapV2Router::new(router_addr, provider);
    let amounts: Vec<U256> = r
        .get_amounts_out(U256::from(10u64.pow(18)), vec![token_in_addr, token_out_addr])
        .call()
        .await
        .map_err(DexError::provider)?;
    let out = amounts
        .get(1)
        .ok_or_else(|| DexError::other("getAmountsOut: empty"))?;
    let s = out.to_string();
    let v: f64 = s.parse().unwrap_or(0.0);
    Ok(v / 1e18)
}

async fn fetch_decimals(
    client: Arc<SignerMiddleware<Arc<Provider<Http>>, LocalWallet>>,
    token: ethers::core::types::Address,
) -> Result<u8, DexError> {
    let erc20 = IErc20::new(token, client);
    let decimals: u8 = erc20
        .decimals()
        .call()
        .await
        .map_err(DexError::provider)?;
    Ok(decimals)
}

async fn ensure_allowance(
    client: Arc<SignerMiddleware<Arc<Provider<Http>>, LocalWallet>>,
    token: ethers::core::types::Address,
    owner: ethers::core::types::Address,
    spender: ethers::core::types::Address,
    required: U256,
) -> Result<(), DexError> {
    let erc20 = IErc20::new(token, client.clone());
    let current: U256 = erc20
        .allowance(owner, spender)
        .call()
        .await
        .map_err(DexError::provider)?;
    if current >= required {
        return Ok(());
    }
    // Approve max uint256; this is a real on-chain tx.
    let max = U256::max_value();
    let approve_call = erc20.approve(spender, max);
    let pending = approve_call.send().await.map_err(DexError::provider)?;
    let receipt = tokio::time::timeout(Duration::from_secs(120), pending)
        .await
        .map_err(|_| DexError::other("approve timeout"))?
        .map_err(DexError::provider)?
        .ok_or_else(|| DexError::other("approve no receipt"))?;
    if receipt.status != Some(1.into()) {
        return Err(DexError::other("approve reverted"));
    }
    Ok(())
}

fn unix_now() -> U256 {
    U256::from(
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs(),
    )
}

fn to_base_units(amount: f64, decimals: u8) -> U256 {
    let scaled = amount * 10f64.powi(decimals as i32);
    // Clamp to fit in U256.
    if scaled.is_finite() && scaled >= 0.0 && scaled < 1e76 {
        U256::from_dec_str(&format!("{}", scaled as u128)).unwrap_or_default()
    } else {
        U256::zero()
    }
}

fn from_base_units(amount: U256, decimals: u8) -> f64 {
    // Convert a U256 base-unit amount back to a human-readable float.
    let s = amount.to_string();
    let val: f64 = s.parse().unwrap_or(0.0);
    val / 10f64.powi(decimals as i32)
}

fn decode_amount_out(
    receipt: &ethers::core::types::TransactionReceipt,
    token_out: ethers::core::types::Address,
    recipient: ethers::core::types::Address,
    out_decimals: u8,
) -> Option<f64> {
    // ERC-20 Transfer event signature: keccak256("Transfer(address,address,uint256)")
    let transfer_topic =
        ethers::core::utils::keccak256("Transfer(address,address,uint256)");
    let mut best: Option<U256> = None;
    for log in &receipt.logs {
        if log.topics.len() != 3 {
            continue;
        }
        if log.topics[0] != transfer_topic.into() {
            continue;
        }
        if log.address != token_out {
            continue;
        }
        // topic[2] is the `to` address (padded to 32 bytes).
        let mut to_bytes = [0u8; 32];
        to_bytes.copy_from_slice(log.topics[2].as_bytes());
        let to = ethers::core::types::Address::from_slice(&to_bytes[12..]);
        if to != recipient {
            continue;
        }
        if log.data.0.len() >= 32 {
            let mut buf = [0u8; 32];
            buf.copy_from_slice(&log.data.0[..32]);
            let amount = U256::from_big_endian(&buf);
            best = Some(match best {
                Some(b) if amount > b => amount,
                Some(b) => b,
                None => amount,
            });
        }
    }
    best.map(|u| from_base_units(u, out_decimals))
}

#[derive(Debug)]
pub enum DexError {
    Provider(String),
    Signer(String),
    Abi(String),
    Config(String),
    Timeout(String),
    Reverted(String),
    Decode(String),
    Other(String),
}

impl DexError {
    fn provider<E: std::fmt::Display>(e: E) -> Self {
        DexError::Provider(e.to_string())
    }
    fn signer<E: std::fmt::Display>(e: E) -> Self {
        DexError::Signer(e.to_string())
    }
    fn abi<E: std::fmt::Display>(e: E) -> Self {
        DexError::Abi(e.to_string())
    }
    fn config(s: String) -> Self {
        DexError::Config(s)
    }
    fn timeout(h: ethers::core::types::TxHash) -> Self {
        DexError::Timeout(format!("{h:#x}"))
    }
    fn reverted(h: ethers::core::types::TxHash) -> Self {
        DexError::Reverted(format!("{h:#x}"))
    }
    fn decode(h: ethers::core::types::TxHash) -> Self {
        DexError::Decode(format!("{h:#x}"))
    }
    fn other(s: &'static str) -> Self {
        DexError::Other(s.to_string())
    }
}

impl std::fmt::Display for DexError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            DexError::Provider(s) => write!(f, "dex provider error: {s}"),
            DexError::Signer(s) => write!(f, "dex signer error: {s}"),
            DexError::Abi(s) => write!(f, "dex abi error: {s}"),
            DexError::Config(s) => write!(f, "dex config error: {s}"),
            DexError::Timeout(h) => write!(f, "dex tx timeout: {h}"),
            DexError::Reverted(h) => write!(f, "dex tx reverted: {h}"),
            DexError::Decode(h) => write!(f, "dex decode error: {h}"),
            DexError::Other(s) => write!(f, "dex error: {s}"),
        }
    }
}

impl std::error::Error for DexError {}

// Re-export tokens used by the abigen output.
#[allow(unused_imports)]
use ethers::contract::builders::ContractCall;
