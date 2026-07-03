//! Launch-readiness controls shared by backend, mobile, extension, and ops.

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum LaunchCategory {
    CriticalProduct,
    WalletUx,
    Security,
    Infrastructure,
    LegalCompliance,
    Operations,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LaunchGate {
    pub id: &'static str,
    pub category: LaunchCategory,
    pub description: &'static str,
    pub required_for_mainnet: bool,
    pub passed: bool,
}

pub fn default_launch_gates() -> Vec<LaunchGate> {
    vec![
        LaunchGate { id: "aa-smart-wallet", category: LaunchCategory::CriticalProduct, description: "ERC-4337 smart account validation, session keys, key rotation, spending limits", required_for_mainnet: true, passed: false },
        LaunchGate { id: "gasless-relayer", category: LaunchCategory::CriticalProduct, description: "EIP-2771/ERC-4337 relayer policies, sponsor budgets, nonce/deadline checks", required_for_mainnet: true, passed: false },
        LaunchGate { id: "guardian-recovery", category: LaunchCategory::CriticalProduct, description: "Guardian social recovery with timelock, threshold approvals, cancellation", required_for_mainnet: true, passed: false },
        LaunchGate { id: "no-mock-wallets", category: LaunchCategory::Security, description: "Backend must not return mock mnemonics or random production wallet data", required_for_mainnet: true, passed: false },
        LaunchGate { id: "real-market-data", category: LaunchCategory::WalletUx, description: "Balances, chart data, prices, portfolio history, and NFTs come from real providers", required_for_mainnet: true, passed: false },
        LaunchGate { id: "transaction-simulation", category: LaunchCategory::Security, description: "Human-readable transaction preview, approval scanner, Permit2 warnings, malicious DApp detection", required_for_mainnet: true, passed: false },
        LaunchGate { id: "ci-security", category: LaunchCategory::Infrastructure, description: "SAST, dependency audit, secret scanning, contract coverage, mobile/extension builds, SBOM", required_for_mainnet: true, passed: false },
        LaunchGate { id: "legal-docs", category: LaunchCategory::LegalCompliance, description: "Terms, privacy, regional restrictions, KYC/AML policy for fiat/P2P/custody", required_for_mainnet: true, passed: false },
        LaunchGate { id: "ops-runbooks", category: LaunchCategory::Operations, description: "Incident response, key compromise, RPC outage, bridge/swap incident, support escalation", required_for_mainnet: true, passed: false },
    ]
}

pub fn blocking_gates(gates: &[LaunchGate]) -> Vec<&LaunchGate> {
    gates
        .iter()
        .filter(|g| g.required_for_mainnet && !g.passed)
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_gates_block_mainnet_until_passed() {
        let gates = default_launch_gates();
        assert!(blocking_gates(&gates).len() >= 8);
    }
}
