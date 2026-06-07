// ============================================================================
// TIGERWALLET SECURITY CENTER
// Transaction simulation, anti-phishing, scam detection, address reputation
// ============================================================================

use std::collections::HashMap;
use serde::{Deserialize, Serialize};

/// Security alert severity
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AlertSeverity {
    Low,
    Medium,
    High,
    Critical,
}

/// Security alert type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AlertType {
    Phishing,
    Scam,
    Malware,
    Suspicious,
    Risk,
    Safe,
}

/// Security alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityAlert {
    pub id: String,
    pub alert_type: AlertType,
    pub severity: AlertSeverity,
    pub message: String,
    pub address: Option<String>,
    pub tx_hash: Option<String>,
    pub timestamp: i64,
    pub resolved: bool,
}

impl SecurityAlert {
    pub fn new(alert_type: AlertType, severity: AlertSeverity, message: &str) -> Self {
        Self {
            id: generate_alert_id(),
            alert_type,
            severity,
            message: message.to_string(),
            address: None,
            tx_hash: None,
            timestamp: current_timestamp(),
            resolved: false,
        }
    }

    pub fn with_address(mut self, address: &str) -> Self {
        self.address = Some(address.to_string());
        self
    }

    pub fn with_tx_hash(mut self, tx_hash: &str) -> Self {
        self.tx_hash = Some(tx_hash.to_string());
        self
    }

    pub fn resolve(&mut self) {
        self.resolved = true;
    }
}

/// Transaction simulation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulationResult {
    pub safe: bool,
    pub changes: Vec<TokenChange>,
    pub warnings: Vec<String>,
    pub errors: Vec<String>,
    pub gas_used: u64,
    pub debug: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenChange {
    pub token: String,
    pub change_type: ChangeType,
    pub amount: String,
    pub usd_value: Option<String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum ChangeType {
    Send,
    Receive,
    Approve,
    Transfer,
}

/// Address reputation score
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReputationScore {
    pub address: String,
    pub score: i32, // -100 to 100
    pub risk_level: RiskLevel,
    pub factors: Vec<RiskFactor>,
    pub tags: Vec<String>,
    pub first_seen: i64,
    pub last_activity: i64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RiskLevel {
    Unknown,
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    pub factor_type: String,
    pub description: String,
    pub weight: i32,
}

/// Transaction simulator
pub struct TransactionSimulator {
    tenderly_key: Option<String>,
    blowfish_key: Option<String>,
}

impl TransactionSimulator {
    pub fn new() -> Self {
        Self {
            tenderly_key: None,
            blowfish_key: None,
        }
    }

    pub fn with_tenderly(mut self, api_key: &str) -> Self {
        self.tenderly_key = Some(api_key.to_string());
        self
    }

    pub fn with_blowfish(mut self, api_key: &str) -> Self {
        self.blowfish_key = Some(api_key.to_string());
        self
    }

    /// Simulate EVM transaction
    pub async fn simulate_evm(
        &self,
        chain_id: u64,
        from: &str,
        to: &str,
        data: &str,
        value: &str,
    ) -> Result<SimulationResult, SimulatorError> {
        // Basic simulation - in production use Tenderly/Blowfish API
        
        let mut warnings = Vec::new();
        let mut errors = Vec::new();
        let mut changes = Vec::new();
        
        // Check if it's a token transfer
        if data.len() >= 10 {
            let selector = &data[..10];
            
            match selector {
                // ERC-20 Transfer
                "0xa9059cbb" => {
                    let token = to; // This is actually the token contract
                    changes.push(TokenChange {
                        token: "NATIVE".to_string(),
                        change_type: ChangeType::Send,
                        amount: value.to_string(),
                        usd_value: None,
                    });
                    warnings.push("Token transfer detected".to_string());
                }
                // ERC-20 Approve
                "0x095ea7b3" => {
                    warnings.push("Token approval detected - verify recipient".to_string());
                }
                // Unknown function
                _ => {
                    warnings.push(format!("Unknown contract call: {}", selector));
                }
            }
        }
        
        // Check for suspicious patterns
        if to == "0x0000000000000000000000000000000000000000" {
            errors.push("Contract creation detected".to_string());
        }
        
        // Check value
        if !value.is_empty() && value != "0x0" {
            changes.push(TokenChange {
                token: "NATIVE".to_string(),
                change_type: ChangeType::Send,
                amount: value.to_string(),
                usd_value: None,
            });
        }
        
        Ok(SimulationResult {
            safe: errors.is_empty(),
            changes,
            warnings,
            errors,
            gas_used: 21000,
            debug: vec![],
        })
    }

    /// Simulate token approval
    pub fn simulate_approval(&self, spender: &str, amount: &str) -> SimulationResult {
        SimulationResult {
            safe: true,
            changes: vec![TokenChange {
                token: "APPROVAL".to_string(),
                change_type: ChangeType::Approve,
                amount: amount.to_string(),
                usd_value: None,
            }],
            warnings: vec![
                format!("Approving {} to spend unlimited tokens", spender),
                "Only approve trusted contracts".to_string(),
            ],
            errors: vec![],
            gas_used: 50000,
            debug: vec![],
        }
    }
}

impl Default for TransactionSimulator {
    fn default() -> Self {
        Self::new()
    }
}

/// Anti-phishing scanner
pub struct AntiPhishingScanner {
    known_phishing: HashMap<String, PhishingInfo>,
}

#[derive(Debug, Clone)]
pub struct PhishingInfo {
    pub domain: String,
    pub target: String,
    pub reported_at: i64,
    pub phishing_type: String,
}

impl AntiPhishingScanner {
    pub fn new() -> Self {
        let mut known_phishing = HashMap::new();
        
        // Add some known phishing patterns (in production, use external API)
        known_phishing.insert(
            "fake-uniswap.com".to_string(),
            PhishingInfo {
                domain: "fake-uniswap.com".to_string(),
                target: "Uniswap".to_string(),
                reported_at: 0,
                phishing_type: "fake exchange".to_string(),
            },
        );
        
        Self { known_phishing }
    }

    /// Scan URL for phishing
    pub fn scan_url(&self, url: &str) -> Option<SecurityAlert> {
        // Check against known phishing domains
        for (domain, info) in &self.known_phishing {
            if url.contains(domain) {
                return Some(SecurityAlert::new(
                    AlertType::Phishing,
                    AlertSeverity::Critical,
                    &format!("Known phishing site targeting {}", info.target),
                ));
            }
        }
        
        // Check for suspicious patterns
        let suspicious_patterns = vec![
            ("-swap.com", "fake DEX"),
            ("-app.com", "fake app"),
            ("login-", "phishing login"),
            ("secure-", "phishing secure"),
        ];
        
        for (pattern, description) in suspicious_patterns {
            if url.contains(pattern) {
                return Some(SecurityAlert::new(
                    AlertType::Suspicious,
                    AlertSeverity::Medium,
                    &format!("Suspicious URL pattern: {}", description),
                ));
            }
        }
        
        None
    }

    /// Scan address for risk
    pub fn scan_address(&self, address: &str) -> ReputationScore {
        let mut factors = Vec::new();
        let mut tags = Vec::new();
        let mut score: i32 = 50; // Start neutral
        
        // Check for zero address
        if address == "0x0000000000000000000000000000000000000000" {
            factors.push(RiskFactor {
                factor_type: "zero_address".to_string(),
                description: "Zero address detected".to_string(),
                weight: -50,
            });
            score -= 50;
            tags.push("zero_address".to_string());
        }
        
        // Check for contract address (starts with 0x0000)
        if address.starts_with("0x0000") {
            factors.push(RiskFactor {
                factor_type: "contract_address".to_string(),
                description: "Contract address detected".to_string(),
                weight: -10,
            });
            score -= 10;
            tags.push("contract".to_string());
        }
        
        // Determine risk level
        let risk_level = if score >= 75 {
            RiskLevel::Low
        } else if score >= 50 {
            RiskLevel::Medium
        } else if score >= 25 {
            RiskLevel::High
        } else {
            RiskLevel::Critical
        };
        
        ReputationScore {
            address: address.to_string(),
            score,
            risk_level,
            factors,
            tags,
            first_seen: 0,
            last_activity: current_timestamp(),
        }
    }
}

impl Default for AntiPhishingScanner {
    fn default() -> Self {
        Self::new()
    }
}

/// Smart contract scanner
pub struct ContractScanner {
    honeypot_signatures: Vec<String>,
    malicious_patterns: Vec<MaliciousPattern>,
}

#[derive(Debug, Clone)]
pub struct MaliciousPattern {
    pub pattern: String,
    pub description: String,
    pub severity: AlertSeverity,
}

impl ContractScanner {
    pub fn new() -> Self {
        let honeypot_signatures = vec![
            "selfdestruct".to_string(),
            "suicide".to_string(),
        ];
        
        let malicious_patterns = vec![
            MaliciousPattern {
                pattern: "delegatecall".to_string(),
                description: "Uses delegatecall - potential vulnerability".to_string(),
                severity: AlertSeverity::Medium,
            },
            MaliciousPattern {
                pattern: "create2".to_string(),
                description: "Uses create2 - potential front-running".to_string(),
                severity: AlertSeverity::Low,
            },
        ];
        
        Self {
            honeypot_signatures,
            malicious_patterns,
        }
    }

    /// Scan contract bytecode for malicious patterns
    pub fn scan_bytecode(&self, bytecode: &str) -> Vec<SecurityAlert> {
        let mut alerts = Vec::new();
        
        // Check for honeypot signatures
        for sig in &self.honeypot_signatures {
            if bytecode.contains(sig) {
                alerts.push(SecurityAlert::new(
                    AlertType::Risk,
                    AlertSeverity::High,
                    &format!("Contract contains: {}", sig),
                ));
            }
        }
        
        // Check malicious patterns
        for pattern in &self.malicious_patterns {
            if bytecode.contains(&pattern.pattern) {
                alerts.push(SecurityAlert::new(
                    AlertType::Suspicious,
                    pattern.severity,
                    &pattern.description,
                ));
            }
        }
        
        alerts
    }

    /// Check if token is verified (in production, check against block explorers)
    pub fn is_verified_token(&self, token: &str) -> bool {
        // In production, check against Etherscan API
        // For now, return false as default
        false
    }
}

impl Default for ContractScanner {
    fn default() -> Self {
        Self::new()
    }
}

/// Helper functions
fn generate_alert_id() -> String {
    use std::time::SystemTime;
    let nanos = SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    format!("alert_{}", nanos)
}

fn current_timestamp() -> i64 {
    use std::time::SystemTime;
    SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64
}

/// Simulator errors
#[derive(Debug, Clone)]
pub enum SimulatorError {
    NetworkError(String),
    ParseError(String),
    SimulationFailed(String),
}

impl std::fmt::Display for SimulatorError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SimulatorError::NetworkError(e) => write!(f, "Network error: {}", e),
            SimulatorError::ParseError(e) => write!(f, "Parse error: {}", e),
            SimulatorError::SimulationFailed(e) => write!(f, "Simulation failed: {}", e),
        }
    }
}

impl std::error::Error for SimulatorError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_transaction_simulator() {
        let simulator = TransactionSimulator::new();
        
        let result = simulator.simulate_approval(
            "0x7426d52352014cFB77c687717cE5AAd7C3aAD86c",
            "115792089237316195423570985008687907853269984665640564039457584007913129639935",
        );
        
        assert!(result.safe);
        assert!(!result.warnings.is_empty());
    }

    #[test]
    fn test_anti_phishing() {
        let scanner = AntiPhishingScanner::new();
        
        let alert = scanner.scan_url("https://fake-uniswap.com/swap");
        assert!(alert.is_some());
        
        let score = scanner.scan_address("0x7426d52352014cFB77c687717cE5AAd7C3aAD86c");
        assert!(score.score >= 0);
    }

    #[test]
    fn test_contract_scanner() {
        let scanner = ContractScanner::new();
        
        let bytecode = "0x608060405234801561001057600080fd5b50";
        let alerts = scanner.scan_bytecode(bytecode);
        
        assert!(alerts.is_empty());
    }
}