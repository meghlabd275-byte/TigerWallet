//! Gas Simulator
//!
//! Prices gas from caller-supplied figures. Gas *prices* are chain state and
//! must come from an oracle or RPC; this module deliberately ships no
//! per-chain default, because a guessed price silently under- or over-states
//! the cost of a transaction. Callers that cannot reach an oracle get `None`
//! and must surface that, rather than quoting a fabricated number.

use std::collections::HashMap;
use std::sync::RwLock;

/// Gas consumed by a native-value transfer (EIP-20 of a plain send).
pub const GAS_NATIVE_TRANSFER: u64 = 21_000;
/// Gas consumed by an ERC-20 transfer, net of the recipient's storage state.
pub const GAS_TOKEN_TRANSFER: u64 = 65_000;
/// Gas consumed by an ERC-20 approval.
pub const GAS_TOKEN_APPROVAL: u64 = 46_000;

/// Floor for an ERC-20 transfer to a fresh (zero-balance) recipient: an SSTORE
/// from zero to non-zero costs 20k more than the warm case.
pub const GAS_TOKEN_TRANSFER_NEW_RECIPIENT: u64 = 85_000;

/// Where a gas figure came from. A caller can then decide whether a quote is
/// authoritative enough to act on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GasSource {
    /// Measured by `eth_estimateGas`.
    Estimate,
    /// Read from the node's current gas price.
    Oracle,
    /// Supplied by the operator as a static configuration.
    Configured,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct GasPrice {
    /// Wei per gas unit.
    pub wei: u128,
    pub source: GasSource,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct GasCost {
    pub gas: u64,
    pub price_wei: u128,
    pub source: GasSource,
}

impl GasCost {
    /// Total cost in wei, or `None` if it would overflow `u128`.
    pub fn total_wei(&self) -> Option<u128> {
        (self.gas as u128).checked_mul(self.price_wei)
    }
}

/// Per-operation gas limits. Defaults are the EVM's own intrinsic costs, which
/// are protocol constants, not chain state.
pub struct GasSimulator {
    limits: RwLock<HashMap<String, u64>>,
    prices: RwLock<HashMap<String, GasPrice>>,
}

impl GasSimulator {
    pub fn new() -> Self {
        let limits = RwLock::new(HashMap::new());
        {
            let mut map = limits.write().unwrap();
            map.insert("transfer".to_string(), GAS_NATIVE_TRANSFER);
            map.insert("token_transfer".to_string(), GAS_TOKEN_TRANSFER);
            map.insert("approve".to_string(), GAS_TOKEN_APPROVAL);
        }

        Self {
            limits,
            prices: RwLock::new(HashMap::new()),
        }
    }

    /// Override the gas limit for an operation.
    pub fn set_gas_limit(&self, operation: &str, gas: u64) {
        self.limits
            .write()
            .unwrap()
            .insert(operation.to_string(), gas);
    }

    /// Record an observed gas price for a chain.
    pub fn set_gas_price(&self, chain: &str, wei: u128, source: GasSource) {
        self.prices
            .write()
            .unwrap()
            .insert(chain.to_string(), GasPrice { wei, source });
    }

    /// Gas price for a chain, or `None` when none has been observed.
    pub fn get_gas_price(&self, chain: &str) -> Option<GasPrice> {
        self.prices.read().unwrap().get(chain).copied()
    }

    /// Gas limit registered for an operation, if any.
    pub fn gas_limit(&self, operation: &str) -> Option<u64> {
        self.limits.read().unwrap().get(operation).copied()
    }

    /// Cost of an operation on a chain, or `None` when either the limit or the
    /// price is unknown.
    pub fn estimate_cost(&self, chain: &str, operation: &str, gas: Option<u64>) -> Option<GasCost> {
        let gas = gas.or_else(|| self.gas_limit(operation))?;
        let price = self.get_gas_price(chain)?;
        Some(GasCost {
            gas,
            price_wei: price.wei,
            source: price.source,
        })
    }

    /// Gas for a swap: the caller's measured estimate when available, else the
    /// router's per-hop constant times the hop count.
    pub fn estimate_swap_gas(&self, measured: Option<u64>, hops: usize) -> Option<u64> {
        match measured {
            Some(gas) if gas > 0 => Some(gas),
            _ => {
                let hops = u64::try_from(hops).ok()?;
                if hops == 0 {
                    return None;
                }
                crate::route_simulator::GAS_PER_HOP
                    .checked_mul(hops as u128)
                    .and_then(|g| u64::try_from(g).ok())
            }
        }
    }

    /// Gas for a transfer: 21k for native value, 65k for a token transfer to a
    /// funded recipient, 85k when the recipient's balance slot is empty.
    pub fn estimate_transfer_gas(&self, is_native: bool, recipient_has_balance: bool) -> u64 {
        if is_native {
            GAS_NATIVE_TRANSFER
        } else if recipient_has_balance {
            GAS_TOKEN_TRANSFER
        } else {
            GAS_TOKEN_TRANSFER_NEW_RECIPIENT
        }
    }

    /// Gas for an arbitrary contract method, using a caller-supplied estimate
    /// or a previously configured limit for that method.
    pub fn estimate_contract_gas(&self, method: &str, measured: Option<u64>) -> Option<u64> {
        measured
            .filter(|gas| *gas > 0)
            .or_else(|| self.gas_limit(method))
    }
}

impl Default for GasSimulator {
    fn default() -> Self {
        Self::new()
    }
}
