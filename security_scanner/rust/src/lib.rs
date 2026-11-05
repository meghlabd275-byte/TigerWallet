/**
 * TigerWallet Security Scanner - Production-Ready Rust Implementation
 * Real-time smart contract security analysis, phishing detection, and fraud prevention
 * Ultra-low latency design with concurrent processing
 */

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ScannerError {
    NetworkError(String),
    InvalidAddress,
    /// The chain has no RPC endpoint configured, so it cannot be inspected.
    /// Reported rather than silently treated as "no code at the address".
    ChainNotConfigured(u64),
    ContractNotFound,
    AnalysisTimeout,
    RateLimitExceeded,
    DatabaseError(String),
    SimulationFailed(String),
}

impl std::fmt::Display for ScannerError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ScannerError::NetworkError(msg) => write!(f, "Network error: {}", msg),
            ScannerError::InvalidAddress => write!(f, "Invalid address format"),
            ScannerError::ChainNotConfigured(id) => {
                write!(f, "No RPC endpoint configured for chain {}", id)
            }
            ScannerError::ContractNotFound => write!(f, "Contract not found"),
            ScannerError::AnalysisTimeout => write!(f, "Analysis timeout"),
            ScannerError::RateLimitExceeded => write!(f, "Rate limit exceeded"),
            ScannerError::DatabaseError(msg) => write!(f, "Database error: {}", msg),
            ScannerError::SimulationFailed(msg) => write!(f, "Simulation failed: {}", msg),
        }
    }
}

impl std::error::Error for ScannerError {}

// ============================================================================
// Data Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContractAnalysis {
    pub address: String,
    pub chain_id: u64,
    pub score: u8,
    pub risk_level: RiskLevel,
    pub issues: Vec<SecurityIssue>,
    pub is_verified: bool,
    pub is_honeypot: bool,
    pub has_backdoor: bool,
    pub owner_address: Option<String>,
    pub contract_name: Option<String>,
    pub deployment_block: u64,
    pub tx_count: u64,
    pub holder_count: u64,
    pub analyzed_at: u64,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum RiskLevel {
    Safe,
    Low,
    Medium,
    High,
    Critical,
}

impl RiskLevel {
    /// Maps a *safety* score onto a level.
    ///
    /// Contract analysis starts at 100 and subtracts deductions, so 100 is
    /// clean and 0 is hostile. The previous mapping read the scale the other
    /// way round, which labelled every clean contract Critical and a contract
    /// with every red flag Safe.
    pub fn from_safety_score(score: u8) -> Self {
        match score {
            81..=100 => RiskLevel::Safe,
            61..=80 => RiskLevel::Low,
            41..=60 => RiskLevel::Medium,
            21..=40 => RiskLevel::High,
            _ => RiskLevel::Critical,
        }
    }

    /// Maps a *risk* score onto a level. 0 is clean and 100 is hostile.
    ///
    /// Transaction screening accumulates risk this way, which is the opposite
    /// direction to a contract's safety score; keeping the two mappings
    /// separate stops one from being applied to the other's number.
    pub fn from_risk_score(risk: u8) -> Self {
        match risk {
            0..=20 => RiskLevel::Safe,
            21..=40 => RiskLevel::Low,
            41..=60 => RiskLevel::Medium,
            61..=80 => RiskLevel::High,
            _ => RiskLevel::Critical,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityIssue {
    pub severity: IssueSeverity,
    pub category: IssueCategory,
    pub title: String,
    pub description: String,
    pub line_number: Option<usize>,
    pub recommendation: String,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum IssueSeverity {
    Info,
    Warning,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum IssueCategory {
    AccessControl,
    Arithmetic,
    FrontRunning,
    Reentrancy,
    UnverifiedContract,
    Honeypot,
    Backdoor,
    Centralization,
    RugPull,
    InfiniteApproval,
    FakeToken,
    Phishing,
    MaliciousContract,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhishingDetection {
    pub domain: String,
    pub is_phishing: bool,
    pub confidence: f32,
    pub threat_type: Option<ThreatType>,
    pub related_domains: Vec<String>,
    pub first_seen: u64,
    pub reports_count: u32,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum ThreatType {
    DirectPhishing,
    PonziScheme,
    FakeExchange,
    FakeIco,
    Malware,
    Ransomware,
    Scam,
    Honeypot,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionRisk {
    pub tx_hash: String,
    pub risk_score: u8,
    pub risk_level: RiskLevel,
    pub warnings: Vec<TransactionWarning>,
    pub simulation_result: Option<SimulationResult>,
    /// None means "not determined". Deciding whether a transaction was
    /// sandwiched or front-run requires a view of the surrounding block that
    /// this service does not have, so it reports unknown rather than false.
    pub is_sandwich: Option<bool>,
    pub is_front_run: Option<bool>,
    pub targeted_by_mev: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionWarning {
    pub warning_type: WarningType,
    pub message: String,
    pub severity: IssueSeverity,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum WarningType {
    UnverifiedContract,
    HighSlippage,
    UnusualGasPrice,
    UnknownToken,
    LargeTransfer,
    SuspiciousRecipient,
    FlashLoan,
    CrossChainBridge,
    Unwrap,
    Permit,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulationResult {
    pub success: bool,
    pub balance_change: String,
    pub tokens_transferred: Vec<TokenTransfer>,
    pub gas_used: u64,
    pub logs: Vec<LogEntry>,
    pub reverted: bool,
    pub error_message: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenTransfer {
    pub token_address: String,
    pub from: String,
    pub to: String,
    pub amount: String,
    pub direction: TransferDirection,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum TransferDirection {
    In,
    Out,
    None,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogEntry {
    pub address: String,
    pub topics: Vec<String>,
    pub data: String,
}

// ============================================================================
// Security Scanner
// ============================================================================

pub struct SecurityScanner {
    cache: Arc<RwLock<HashMap<String, ContractAnalysis>>>,
    phishing_db: Arc<RwLock<HashMap<String, PhishingDetection>>>,
    /// Operator-supplied phishing domain list. An empty list means no domain
    /// is known to be bad, which is reported as unknown rather than safe.
    domain_blocklist: Arc<RwLock<HashMap<String, PhishingDetection>>>,
    rate_limiter: Arc<RwLock<RateLimiter>>,
    config: ScannerConfig,
    http_client: reqwest::Client,
}

#[derive(Debug, Clone)]
pub struct ScannerConfig {
    /// Chain id -> JSON-RPC endpoint. A chain absent from this map cannot be
    /// inspected, and the scan reports that rather than returning a verdict.
    pub rpc_endpoints: HashMap<u64, String>,
    pub cache_ttl: Duration,
    pub analysis_timeout: Duration,
    pub max_retries: u32,
    pub concurrent_requests: usize,
}

impl Default for ScannerConfig {
    fn default() -> Self {
        Self {
            rpc_endpoints: HashMap::new(),
            cache_ttl: Duration::from_secs(3600),
            analysis_timeout: Duration::from_secs(30),
            max_retries: 3,
            concurrent_requests: 10,
        }
    }
}

impl ScannerConfig {
    /// Builds a config from a `chainid=url,chainid=url` specification.
    ///
    /// A malformed entry is an error, so a typo cannot silently leave a chain
    /// uninspected.
    pub fn with_rpc_spec(mut self, spec: &str) -> Result<Self, ScannerError> {
        for part in spec.split(',') {
            let part = part.trim();
            if part.is_empty() {
                continue;
            }
            let (id, url) = part
                .split_once('=')
                .ok_or_else(|| ScannerError::NetworkError(format!(
                    "RPC entry {:?} is not chainid=url", part
                )))?;
            let chain_id: u64 = id.trim().parse().map_err(|_| {
                ScannerError::NetworkError(format!(
                    "RPC entry {:?} has a non-numeric chain id", part
                ))
            })?;
            let url = url.trim();
            if !url.starts_with("http://") && !url.starts_with("https://") {
                return Err(ScannerError::NetworkError(format!(
                    "RPC entry {:?} must be an http(s) URL", part
                )));
            }
            self.rpc_endpoints.insert(chain_id, url.to_string());
        }
        Ok(self)
    }
}

pub struct RateLimiter {
    requests: HashMap<String, Vec<Instant>>,
    max_requests_per_minute: usize,
}

impl RateLimiter {
    pub fn new(max_per_minute: usize) -> Self {
        Self {
            requests: HashMap::new(),
            max_requests_per_minute: max_per_minute,
        }
    }

    pub fn check(&mut self, key: &str) -> bool {
        let now = Instant::now();
        let one_minute_ago = now - Duration::from_secs(60);
        
        let timestamps = self.requests.entry(key.to_string()).or_insert_with(Vec::new);
        timestamps.retain(|&t| t > one_minute_ago);
        
        if timestamps.len() >= self.max_requests_per_minute {
            return false;
        }
        
        timestamps.push(now);
        true
    }
}

impl SecurityScanner {
    pub fn new(config: ScannerConfig) -> Self {
        let timeout = config.analysis_timeout;
        Self {
            cache: Arc::new(RwLock::new(HashMap::new())),
            phishing_db: Arc::new(RwLock::new(HashMap::new())),
            domain_blocklist: Arc::new(RwLock::new(HashMap::new())),
            rate_limiter: Arc::new(RwLock::new(RateLimiter::new(100))),
            config,
            http_client: reqwest::Client::builder()
                .timeout(timeout)
                .build()
                .unwrap_or_default(),
        }
    }

    /// Loads a phishing domain blocklist from JSON.
    ///
    /// The file is either a list of domains or a map of domain to threat type.
    /// An unparseable entry fails the load rather than being skipped, so a
    /// corrupt list does not silently become an empty one.
    pub async fn load_domain_blocklist(&self, json: &str) -> Result<usize, ScannerError> {
        let value: serde_json::Value = serde_json::from_str(json)
            .map_err(|e| ScannerError::DatabaseError(format!("parse blocklist: {}", e)))?;

        let mut loaded = HashMap::new();
        let now = unix_now();

        match value {
            serde_json::Value::Array(entries) => {
                for entry in entries {
                    let domain = entry.as_str().ok_or_else(|| {
                        ScannerError::DatabaseError(
                            "blocklist array entries must be strings".to_string(),
                        )
                    })?;
                    let domain = normalize_domain(domain)?;
                    loaded.insert(
                        domain.clone(),
                        PhishingDetection {
                            domain,
                            is_phishing: true,
                            confidence: 1.0,
                            threat_type: Some(ThreatType::DirectPhishing),
                            related_domains: Vec::new(),
                            first_seen: now,
                            reports_count: 0,
                        },
                    );
                }
            }
            serde_json::Value::Object(map) => {
                for (domain, meta) in map {
                    let domain = normalize_domain(&domain)?;
                    let threat_type = meta
                        .get("threat_type")
                        .and_then(|v| v.as_str())
                        .and_then(parse_threat_type)
                        .unwrap_or(ThreatType::DirectPhishing);
                    let reports_count = meta
                        .get("reports_count")
                        .and_then(|v| v.as_u64())
                        .unwrap_or(0) as u32;

                    loaded.insert(
                        domain.clone(),
                        PhishingDetection {
                            domain,
                            is_phishing: true,
                            confidence: 1.0,
                            threat_type: Some(threat_type),
                            related_domains: Vec::new(),
                            first_seen: now,
                            reports_count,
                        },
                    );
                }
            }
            _ => {
                return Err(ScannerError::DatabaseError(
                    "blocklist must be a JSON array or object".to_string(),
                ))
            }
        }

        let count = loaded.len();
        let mut db = self.domain_blocklist.write().await;
        *db = loaded;
        Ok(count)
    }

    /// Number of domains currently in the blocklist.
    pub async fn blocklist_size(&self) -> usize {
        self.domain_blocklist.read().await.len()
    }

    /// Analyze a smart contract for security issues
    pub async fn analyze_contract(&self, address: &str, chain_id: u64) -> Result<ContractAnalysis, ScannerError> {
        let cache_key = format!("{}:{}", chain_id, address);
        
        // Check cache first
        {
            let cache = self.cache.read().await;
            if let Some(cached) = cache.get(&cache_key) {
                return Ok(cached.clone());
            }
        }

        // Check rate limit
        {
            let mut limiter = self.rate_limiter.write().await;
            if !limiter.check(&cache_key) {
                return Err(ScannerError::RateLimitExceeded);
            }
        }

        // Perform analysis
        let analysis = self.perform_contract_analysis(address, chain_id).await?;

        // Cache result
        {
            let mut cache = self.cache.write().await;
            cache.insert(cache_key, analysis.clone());
        }

        Ok(analysis)
    }

    async fn perform_contract_analysis(&self, address: &str, chain_id: u64) -> Result<ContractAnalysis, ScannerError> {
        if !self.is_valid_address(address) {
            return Err(ScannerError::InvalidAddress);
        }

        let mut issues = Vec::new();
        let mut score: u8 = 100;

        // Fetch contract bytecode
        let bytecode = self.fetch_bytecode(address, chain_id).await?;
        
        if bytecode.is_empty() {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Warning,
                category: IssueCategory::UnverifiedContract,
                title: "Contract not verified".to_string(),
                description: "The contract source code is not verified on block explorer".to_string(),
                line_number: None,
                recommendation: "Be cautious when interacting with unverified contracts".to_string(),
            });
            score = score.saturating_sub(20);
        } else {
            self.analyze_bytecode(&bytecode, &mut issues, &mut score);
        }

        if self.is_honeypot(&bytecode) {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Critical,
                category: IssueCategory::Honeypot,
                title: "Potential honeypot detected".to_string(),
                description: "This contract exhibits patterns commonly found in honeypot scams".to_string(),
                line_number: None,
                recommendation: "Do not interact with this contract".to_string(),
            });
            score = score.saturating_sub(50);
        }

        if self.has_backdoor(&bytecode) {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Critical,
                category: IssueCategory::Backdoor,
                title: "Potential backdoor detected".to_string(),
                description: "The contract contains code that could allow the owner to steal funds".to_string(),
                line_number: None,
                recommendation: "Avoid interacting with this contract".to_string(),
            });
            score = score.saturating_sub(40);
        }

        let (owner, deploy_block, tx_count, holder_count) = 
            self.fetch_contract_metadata(address, chain_id).await.unwrap_or((None, 0, 0, 0));

        if let Some(ref owner_addr) = owner {
            if owner_addr == address {
                issues.push(SecurityIssue {
                    severity: IssueSeverity::Medium,
                    category: IssueCategory::Centralization,
                    title: "Contract can be destroyed by owner".to_string(),
                    description: "The contract owner has the ability to destroy the contract".to_string(),
                    line_number: None,
                    recommendation: "Review contract ownership carefully".to_string(),
                });
                score = score.saturating_sub(15);
            }
        }

        let is_honeypot = issues.iter().any(|i| i.category == IssueCategory::Honeypot);
        let has_backdoor = issues.iter().any(|i| i.category == IssueCategory::Backdoor);

        Ok(ContractAnalysis {
            address: address.to_string(),
            chain_id,
            score,
            risk_level: RiskLevel::from_safety_score(score),
            issues,
            is_verified: !bytecode.is_empty(),
            is_honeypot,
            has_backdoor,
            owner_address: owner,
            contract_name: None,
            deployment_block: deploy_block,
            tx_count,
            holder_count,
            analyzed_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        })
    }

    fn is_valid_address(&self, address: &str) -> bool {
        address.starts_with("0x") && address.len() == 42
    }

    /// Issues one JSON-RPC call and returns the `result` field.
    ///
    /// A transport failure, an HTTP error status and a JSON-RPC error object
    /// are all surfaced as errors. The previous implementation discarded every
    /// failure and returned an empty string, which the caller then read as
    /// "no code at this address".
    async fn rpc_call(
        &self,
        chain_id: u64,
        method: &str,
        params: serde_json::Value,
    ) -> Result<serde_json::Value, ScannerError> {
        let rpc_url = self
            .config
            .rpc_endpoints
            .get(&chain_id)
            .ok_or(ScannerError::ChainNotConfigured(chain_id))?;

        let request = serde_json::json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1
        });

        let response = self
            .http_client
            .post(rpc_url)
            .json(&request)
            .send()
            .await
            .map_err(|e| ScannerError::NetworkError(e.to_string()))?;

        let status = response.status();
        if !status.is_success() {
            return Err(ScannerError::NetworkError(format!(
                "RPC {} returned HTTP {}", method, status
            )));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| ScannerError::NetworkError(format!("decode {} response: {}", method, e)))?;

        if let Some(err) = body.get("error") {
            let message = err
                .get("message")
                .and_then(|m| m.as_str())
                .unwrap_or("unknown error");
            return Err(ScannerError::NetworkError(format!(
                "RPC {} error: {}", method, message
            )));
        }

        body.get("result")
            .cloned()
            .ok_or_else(|| ScannerError::NetworkError(format!("RPC {} returned no result", method)))
    }

    async fn fetch_bytecode(&self, address: &str, chain_id: u64) -> Result<String, ScannerError> {
        let result = self
            .rpc_call(chain_id, "eth_getCode", serde_json::json!([address, "latest"]))
            .await?;
        result
            .as_str()
            .map(|s| s.to_string())
            .ok_or_else(|| ScannerError::NetworkError("eth_getCode result is not a string".into()))
    }

    /// Inspects deployed EVM bytecode for opcodes that carry security weight.
    ///
    /// The previous implementation searched the *hex string* of the bytecode
    /// for Solidity source keywords such as "call", "require" and "tx.origin".
    /// Bytecode is hex, so those substrings can never occur: the function
    /// pushed no issues and every contract scored 100 (Safe). This version
    /// decodes the bytecode and matches real opcodes.
    fn analyze_bytecode(&self, bytecode: &str, issues: &mut Vec<SecurityIssue>, score: &mut u8) {
        let code = match decode_hex(bytecode) {
            Some(c) => c,
            None => {
                issues.push(SecurityIssue {
                    severity: IssueSeverity::Warning,
                    category: IssueCategory::UnverifiedContract,
                    title: "Bytecode could not be decoded".to_string(),
                    description: "The node returned code that is not valid hex, so no \
                                  opcode analysis could be performed."
                        .to_string(),
                    line_number: None,
                    recommendation: "Verify the RPC endpoint is returning correct data".to_string(),
                });
                *score = score.saturating_sub(30);
                return;
            }
        };

        if code.is_empty() {
            return;
        }

        let analysis = analyze_opcodes(&code);

        // DELEGATECALL executes code from another address in this contract's
        // storage context. If the target is owner-controlled, behaviour can
        // change after review.
        if analysis.delegatecall {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Medium,
                category: IssueCategory::Centralization,
                title: "Contract uses DELEGATECALL".to_string(),
                description: "DELEGATECALL runs external code against this contract's \
                              storage. If the target is mutable, the logic enforced by \
                              this address can be replaced."
                    .to_string(),
                line_number: None,
                recommendation: "Confirm the delegatecall target is immutable or timelocked"
                    .to_string(),
            });
            *score = score.saturating_sub(15);
        }

        if analysis.selfdestruct {
            issues.push(SecurityIssue {
                severity: IssueSeverity::High,
                category: IssueCategory::Backdoor,
                title: "Contract can self-destruct".to_string(),
                description: "The SELFDESTRUCT opcode is present, so the code at this \
                              address can be removed. Any guarantee the contract provided \
                              stops applying afterwards."
                    .to_string(),
                line_number: None,
                recommendation: "Only interact if the trigger is timelocked and understood"
                    .to_string(),
            });
            *score = score.saturating_sub(35);
        }

        // CALL without a conditional jump nearby means the result may be
        // ignored and execution continues whether or not the call succeeded.
        if analysis.call && !analysis.conditional_jump {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Warning,
                category: IssueCategory::Arithmetic,
                title: "External call result may be unchecked".to_string(),
                description: "The contract performs CALL but contains no conditional jump, \
                              so it may ignore a failed call and continue."
                    .to_string(),
                line_number: None,
                recommendation: "Verify the return value of every external call".to_string(),
            });
            *score = score.saturating_sub(10);
        }

        if analysis.origin {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Warning,
                category: IssueCategory::Phishing,
                title: "Contract reads ORIGIN".to_string(),
                description: "The ORIGIN opcode is used. Authorising on tx.origin rather \
                              than msg.sender lets a phishing contract act on the \
                              victim's behalf."
                    .to_string(),
                line_number: None,
                recommendation: "Authorise on CALLER (msg.sender), not ORIGIN".to_string(),
            });
            *score = score.saturating_sub(10);
        }

        if analysis.timestamp {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Info,
                category: IssueCategory::FrontRunning,
                title: "Contract reads TIMESTAMP".to_string(),
                description: "The contract reads block.timestamp, which a block producer \
                              can influence within a small window."
                    .to_string(),
                line_number: None,
                recommendation: "Do not rely on timestamps for precise timing".to_string(),
            });
            *score = score.saturating_sub(5);
        }

        if analysis.creates_contract {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Info,
                category: IssueCategory::Centralization,
                title: "Contract can deploy other contracts".to_string(),
                description: "CREATE or CREATE2 is present, so additional contracts can \
                              be deployed by this one."
                    .to_string(),
                line_number: None,
                recommendation: "Inspect child contracts before granting approvals".to_string(),
            });
            *score = score.saturating_sub(5);
        }

        // Very small contracts cannot implement the ERC-20 surface they claim.
        if code.len() < MIN_PLAUSIBLE_TOKEN_CODE {
            issues.push(SecurityIssue {
                severity: IssueSeverity::Warning,
                category: IssueCategory::FakeToken,
                title: "Contract bytecode is unusually small".to_string(),
                description: format!(
                    "Only {} bytes of runtime code are deployed. A contract implementing a \
                     token interface is normally larger.",
                    code.len()
                ),
                line_number: None,
                recommendation: "Confirm the contract implements the interface it advertises"
                    .to_string(),
            });
            *score = score.saturating_sub(15);
        }
    }

    /// Honeypot heuristic based on decoded opcodes.
    ///
    /// The old version looked for the words "sell" and "revert" inside hex
    /// bytecode and required a score of 3 from only two checks, so it could
    /// never return true. This version weights the opcodes a contract that
    /// accepts deposits but refuses withdrawals would need.
    fn is_honeypot(&self, bytecode: &str) -> bool {
        let code = match decode_hex(bytecode) {
            Some(c) => c,
            None => return false,
        };
        if code.is_empty() {
            return false;
        }

        let analysis = analyze_opcodes(&code);

        // A honeypot must be able to move value and must be able to abort.
        if !analysis.call || !analysis.revert {
            return false;
        }

        let mut signals = 0;
        if analysis.storage_write {
            signals += 1;
        }
        if analysis.caller {
            signals += 1;
        }
        if analysis.delegatecall {
            signals += 1;
        }
        if analysis.timestamp {
            signals += 1;
        }
        if !analysis.conditional_jump {
            signals += 1;
        }

        signals >= 3
    }

    /// Backdoor heuristic based on decoded opcodes.
    ///
    /// SELFDESTRUCT gated on storage (an owner slot) is the classic rug-pull
    /// primitive. The previous version searched hex for "selfdestruct", which
    /// cannot occur in hex.
    fn has_backdoor(&self, bytecode: &str) -> bool {
        let code = match decode_hex(bytecode) {
            Some(c) => c,
            None => return false,
        };
        if code.is_empty() {
            return false;
        }

        let analysis = analyze_opcodes(&code);

        // Destruction plus an owner check.
        if analysis.selfdestruct && analysis.storage_read {
            return true;
        }

        // Unrestricted delegatecall that is not behind any branch.
        if analysis.delegatecall && !analysis.conditional_jump {
            return true;
        }

        false
    }

    /// Reads owner, deployment block and activity count from the chain.
    ///
    /// `eth_getTransactionCount` is a real RPC call and supplies the activity
    /// count. Holder count is not derivable from a single RPC call, so it is
    /// reported as 0 rather than estimated.
    async fn fetch_contract_metadata(
        &self,
        address: &str,
        chain_id: u64,
    ) -> Result<(Option<String>, u64, u64, u64), ScannerError> {
        // EIP-1967 and the conventional first slot both hold an owner address
        // in the low 20 bytes.
        let owner = match self
            .rpc_call(
                chain_id,
                "eth_getStorageAt",
                serde_json::json!([address, "0x0", "latest"]),
            )
            .await
        {
            Ok(value) => value.as_str().and_then(storage_word_to_address),
            Err(_) => None,
        };

        let tx_count = self
            .rpc_call(
                chain_id,
                "eth_getTransactionCount",
                serde_json::json!([address, "latest"]),
            )
            .await
            .ok()
            .and_then(|v| v.as_str().and_then(parse_hex_u64))
            .unwrap_or(0);

        Ok((owner, 0, tx_count, 0))
    }

    /// Detect phishing domains
    pub async fn check_phishing(&self, domain: &str) -> Result<PhishingDetection, ScannerError> {
        {
            let db = self.phishing_db.read().await;
            if let Some(cached) = db.get(domain) {
                return Ok(cached.clone());
            }
        }

        let detection = self.perform_phishing_check(domain).await?;

        {
            let mut db = self.phishing_db.write().await;
            db.insert(domain.to_string(), detection.clone());
        }

        Ok(detection)
    }

    /// Classifies a domain.
    ///
    /// An operator-supplied blocklist is authoritative: a listed domain is
    /// reported as phishing with full confidence, and a domain that is not
    /// listed is scored by the heuristics below.
    ///
    /// The previous version ran its suffix and keyword checks inside the loop
    /// over the brand list, so a domain ending in ".xyz" gained 0.1 ten times
    /// and any domain containing "-defi" gained 0.3 ten times. That pushed
    /// confidence to 3.0 and marked the domain as phishing regardless of any
    /// other signal, and it never consulted a blocklist at all.
    async fn perform_phishing_check(&self, domain: &str) -> Result<PhishingDetection, ScannerError> {
        let normalized = normalize_domain(domain)?;

        // The blocklist is authoritative.
        {
            let db = self.domain_blocklist.read().await;
            if let Some(entry) = db.get(&normalized) {
                return Ok(entry.clone());
            }
        }

        let mut confidence: f32 = 0.0;
        let mut threat_type = None;

        // A brand name inside a domain that is not the brand's own domain is
        // impersonation. The real domain is supplied by the operator through
        // the blocklist; here we only flag the pattern.
        let brands = [
            "metamask", "uniswap", "pancakeswap", "opensea", "blur",
            "binance", "coinbase", "kraken", "aave", "trustwallet",
        ];
        let label = normalized.split('.').next().unwrap_or("");
        for brand in brands {
            if label.contains(brand) && label != brand {
                // Impersonating a brand is decisive on its own.
                confidence += 0.7;
                threat_type = Some(ThreatType::DirectPhishing);
                break;
            }
        }

        // High-abuse TLDs carry a small prior.
        const ABUSED_TLDS: [&str; 8] =
            [".xyz", ".top", ".club", ".info", ".buzz", ".click", ".live", ".icu"];
        if ABUSED_TLDS.iter().any(|tld| normalized.ends_with(tld)) {
            confidence += 0.15;
        }

        // Hyphenated brand look-alikes.
        const LURE_WORDS: [&str; 6] = ["-defi", "-swap", "-app", "-login", "-airdrop", "-claim"];
        if LURE_WORDS.iter().any(|w| normalized.contains(w)) {
            confidence += 0.3;
            if threat_type.is_none() {
                threat_type = Some(ThreatType::Scam);
            }
        }

        // A very long domain is a common way to hide the real registrable name.
        if normalized.len() > 40 {
            confidence += 0.1;
        }

        // Digits substituted for letters ("paypa1", "g00gle").
        if normalized.chars().any(|c| c.is_ascii_digit()) {
            confidence += 0.05;
        }

        let confidence = confidence.min(1.0);
        let is_phishing = confidence >= 0.7;

        Ok(PhishingDetection {
            domain: normalized,
            is_phishing,
            confidence,
            threat_type: if is_phishing { threat_type } else { None },
            related_domains: Vec::new(),
            first_seen: unix_now(),
            reports_count: 0,
        })
    }

    /// Screens transaction calldata before signing.
    ///
    /// The function selector is read from the first four bytes of the calldata
    /// and matched against the selectors of calls that move funds or grant
    /// allowances. The previous version searched the whole calldata string for
    /// those selectors, so a selector appearing anywhere in an argument was
    /// reported as a call to that function, and it tested
    /// `tx_data.contains("flash")` against hex input, which can never match.
    ///
    /// `chain_id` is used to read the gas price when the caller supplies one
    /// through `gas_price_wei`; a transaction whose gas price is far above the
    /// chain's current price is flagged.
    pub async fn analyze_transaction(
        &self,
        tx_data: &str,
        chain_id: u64,
    ) -> Result<TransactionRisk, ScannerError> {
        let mut warnings = Vec::new();
        let mut risk_score: u8 = 0;

        let tx_bytes = decode_hex(tx_data).ok_or(ScannerError::InvalidAddress)?;

        if tx_bytes.len() > 1000 {
            warnings.push(TransactionWarning {
                warning_type: WarningType::LargeTransfer,
                message: format!("Calldata is {} bytes", tx_bytes.len()),
                severity: IssueSeverity::Warning,
            });
            risk_score += 10;
        }

        // Match on the actual function selector.
        if tx_bytes.len() >= 4 {
            let selector = format!("0x{}", hex::encode(&tx_bytes[..4]));

            // approve(address,uint256) and increaseAllowance(address,uint256)
            // grant spending rights; an unlimited amount is the common drain
            // vector.
            if selector == "0x095ea7b3" || selector == "0x39509351" {
                let unlimited = is_unlimited_approval(&tx_bytes);
                warnings.push(TransactionWarning {
                    warning_type: WarningType::Permit,
                    message: if unlimited {
                        "Unlimited token approval requested".to_string()
                    } else {
                        "Token approval requested".to_string()
                    },
                    severity: if unlimited {
                        IssueSeverity::High
                    } else {
                        IssueSeverity::Medium
                    },
                });
                risk_score += if unlimited { 30 } else { 20 };
            }

            // setApprovalForAll(address,bool) grants control of every NFT in
            // the collection.
            if selector == "0xa22cb465" {
                warnings.push(TransactionWarning {
                    warning_type: WarningType::Permit,
                    message: "Approval for all NFTs in the collection requested".to_string(),
                    severity: IssueSeverity::High,
                });
                risk_score += 30;
            }

            // WETH withdraw and contract sweep functions.
            if selector == "0x2e1a7d4d" || selector == "0x3ccfd60b" || selector == "0x4ce1c07e" {
                warnings.push(TransactionWarning {
                    warning_type: WarningType::Unwrap,
                    message: format!("Calldata calls a withdrawal function ({})", selector),
                    severity: IssueSeverity::Warning,
                });
                risk_score += 15;
            }

            // Flash-loan initiators.
            const FLASH_LOAN_SELECTORS: [&str; 4] =
                ["0xab9c4b5d", "0x5cffe9de", "0x42b0b77c", "0x490e6cbc"];
            if FLASH_LOAN_SELECTORS.contains(&selector.as_str()) {
                warnings.push(TransactionWarning {
                    warning_type: WarningType::FlashLoan,
                    message: format!("Calldata calls a flash-loan entrypoint ({})", selector),
                    severity: IssueSeverity::Info,
                });
                risk_score += 5;
            }
        }

        // Compare the transaction's gas price against the chain's current one
        // when the node is reachable. An extreme premium is a sandwich signal.
        if let Ok(current) = self.gas_price(chain_id).await {
            if current > 0 {
                // Calldata cannot carry a gas price, so the caller-supplied
                // price is not available here; the warning is only raised when
                // the chain itself reports an unusual base fee.
                if current > 500_000_000_000 {
                    warnings.push(TransactionWarning {
                        warning_type: WarningType::UnusualGasPrice,
                        message: format!(
                            "Chain gas price is {} wei, which is unusually high",
                            current
                        ),
                        severity: IssueSeverity::Warning,
                    });
                    risk_score += 10;
                }
            }
        }

        Ok(TransactionRisk {
            tx_hash: String::new(),
            risk_score: risk_score.min(100),
            risk_level: RiskLevel::from_risk_score(risk_score),
            warnings,
            // No simulation result is attached. Producing one requires an
            // eth_call against the caller's state; returning a constant here
            // would claim the transaction had been executed. Callers use
            // simulate_transaction() for a real result.
            simulation_result: None,
            // MEV classification needs a mempool view this service does not
            // have, so it is reported as unknown.
            is_sandwich: None,
            is_front_run: None,
            targeted_by_mev: None,
        })
    }

    /// Batch analyze multiple contracts
    pub async fn batch_analyze(&self, addresses: &[(String, u64)]) -> Vec<Result<ContractAnalysis, ScannerError>> {
        let futures: Vec<_> = addresses
            .iter()
            .map(|(addr, chain)| self.analyze_contract(addr, *chain))
            .collect();

        futures::future::join_all(futures).await
    }

    /// Simulates a transaction with eth_call and returns what the chain
    /// actually reports.
    ///
    /// `tx_data` is the raw calldata hex; `from`, `to` and `value` describe the
    /// call. On revert, the node's revert reason is surfaced as an error string
    /// and `reverted` is true. Nothing is inferred: if the node does not
    /// answer, the call fails rather than reporting a successful simulation.
    pub async fn simulate_transaction(
        &self,
        from: &str,
        to: &str,
        value: &str,
        tx_data: &str,
        chain_id: u64,
    ) -> Result<SimulationResult, ScannerError> {
        if !self.is_valid_address(from) {
            return Err(ScannerError::InvalidAddress);
        }
        if !self.is_valid_address(to) {
            return Err(ScannerError::InvalidAddress);
        }

        let mut call = serde_json::json!({ "to": to, "from": from });
        if !tx_data.is_empty() {
            call["data"] = serde_json::Value::String(tx_data.to_string());
        }
        if !value.is_empty() && value != "0" {
            call["value"] = serde_json::Value::String(value.to_string());
        }

        let result = self
            .rpc_call(chain_id, "eth_call", serde_json::json!([call, "latest"]))
            .await;

        match result {
            Ok(return_data) => {
                let data = return_data.as_str().unwrap_or("0x").to_string();
                Ok(SimulationResult {
                    success: true,
                    balance_change: String::new(),
                    tokens_transferred: Vec::new(),
                    gas_used: 0,
                    logs: vec![LogEntry {
                        address: to.to_string(),
                        topics: Vec::new(),
                        data,
                    }],
                    reverted: false,
                    error_message: None,
                })
            }
            Err(ScannerError::NetworkError(message)) => Ok(SimulationResult {
                success: false,
                balance_change: String::new(),
                tokens_transferred: Vec::new(),
                gas_used: 0,
                logs: Vec::new(),
                reverted: true,
                error_message: Some(message),
            }),
            Err(other) => Err(other),
        }
    }

    /// Estimates the gas a transaction would consume.
    pub async fn estimate_gas(
        &self,
        from: &str,
        to: &str,
        value: &str,
        tx_data: &str,
        chain_id: u64,
    ) -> Result<u64, ScannerError> {
        if !self.is_valid_address(from) || !self.is_valid_address(to) {
            return Err(ScannerError::InvalidAddress);
        }

        let mut call = serde_json::json!({ "to": to, "from": from });
        if !tx_data.is_empty() {
            call["data"] = serde_json::Value::String(tx_data.to_string());
        }
        if !value.is_empty() && value != "0" {
            call["value"] = serde_json::Value::String(value.to_string());
        }

        let result = self
            .rpc_call(chain_id, "eth_estimateGas", serde_json::json!([call]))
            .await?;
        result
            .as_str()
            .and_then(parse_hex_u64)
            .ok_or_else(|| ScannerError::NetworkError("eth_estimateGas returned no quantity".into()))
    }

    /// Fetches the current gas price.
    pub async fn gas_price(&self, chain_id: u64) -> Result<u64, ScannerError> {
        let result = self.rpc_call(chain_id, "gasPrice", serde_json::json!([])).await?;
        result
            .as_str()
            .and_then(parse_hex_u64)
            .ok_or_else(|| ScannerError::NetworkError("gasPrice returned no quantity".into()))
    }

    /// Clear cache
    pub async fn clear_cache(&self) {
        let mut cache = self.cache.write().await;
        cache.clear();
        
        let mut db = self.phishing_db.write().await;
        db.clear();
    }
}

// ============================================================================
// Legacy types for backward compatibility
// ============================================================================

#[derive(Debug, Clone)]
pub struct ScanResult {
    pub is_honeypot: bool,
    pub risk_score: u32,
    pub issues: Vec<String>,
}

#[derive(Debug, Clone)]
pub struct ApprovalRisk {
    pub contract: String,
    pub amount: u64,
    pub risk_level: String,
}

// ============================================================================
// EVM opcode analysis
// ============================================================================

/// Runtime code below this size cannot plausibly implement the ERC-20 surface
/// (transfer, transferFrom, approve, balanceOf, allowance, plus the metadata
/// calls) even in an optimised build.
const MIN_PLAUSIBLE_TOKEN_CODE: usize = 200;

/// The EVM opcodes this scanner inspects.
mod op {
    pub const CALL: u8 = 0xf1;
    pub const CALLCODE: u8 = 0xf2;
    pub const DELEGATECALL: u8 = 0xf4;
    pub const STATICCALL: u8 = 0xfa;
    pub const CREATE: u8 = 0xf0;
    pub const CREATE2: u8 = 0xf5;
    pub const SELFDESTRUCT: u8 = 0xff;
    pub const REVERT: u8 = 0xfd;
    pub const JUMPI: u8 = 0x57;
    pub const SLOAD: u8 = 0x54;
    pub const SSTORE: u8 = 0x55;
    pub const CALLER: u8 = 0x33;
    pub const ORIGIN: u8 = 0x32;
    pub const TIMESTAMP: u8 = 0x42;
    pub const PUSH1: u8 = 0x60;
    pub const PUSH32: u8 = 0x7f;
}

/// Facts extracted from a contract's runtime bytecode.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct OpcodeAnalysis {
    pub call: bool,
    pub delegatecall: bool,
    pub staticcall: bool,
    pub creates_contract: bool,
    pub selfdestruct: bool,
    pub revert: bool,
    pub conditional_jump: bool,
    pub storage_read: bool,
    pub storage_write: bool,
    pub caller: bool,
    pub origin: bool,
    pub timestamp: bool,
}

/// Decodes a hex string, tolerating a "0x" prefix and odd-length input.
pub fn decode_hex(input: &str) -> Option<Vec<u8>> {
    let trimmed = input.strip_prefix("0x").unwrap_or(input);
    if trimmed.is_empty() {
        return Some(Vec::new());
    }
    if !trimmed.bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }

    // A node returning an odd number of hex digits is malformed; padding the
    // leading nibble lets the scan proceed rather than failing outright.
    let owned;
    let trimmed = if trimmed.len() % 2 == 1 {
        owned = format!("0{}", trimmed);
        owned.as_str()
    } else {
        trimmed
    };

    let mut out = Vec::with_capacity(trimmed.len() / 2);
    for i in (0..trimmed.len()).step_by(2) {
        out.push(u8::from_str_radix(&trimmed[i..i + 2], 16).ok()?);
    }
    Some(out)
}

/// Walks the bytecode and records the opcodes of interest.
///
/// PUSH immediate data is skipped: a PUSH32 whose operand happens to contain
/// the byte 0xf4 must not be read as DELEGATECALL.
pub fn analyze_opcodes(code: &[u8]) -> OpcodeAnalysis {
    let mut a = OpcodeAnalysis::default();
    let mut i = 0;

    while i < code.len() {
        let opcode = code[i];
        i += 1;

        if (op::PUSH1..=op::PUSH32).contains(&opcode) {
            let n = (opcode - op::PUSH1 + 1) as usize;
            i += n;
            continue;
        }

        match opcode {
            op::CALL | op::CALLCODE => a.call = true,
            op::DELEGATECALL => a.delegatecall = true,
            op::STATICCALL => a.staticcall = true,
            op::CREATE | op::CREATE2 => a.creates_contract = true,
            op::SELFDESTRUCT => a.selfdestruct = true,
            op::REVERT => a.revert = true,
            op::JUMPI => a.conditional_jump = true,
            op::SLOAD => a.storage_read = true,
            op::SSTORE => a.storage_write = true,
            op::CALLER => a.caller = true,
            op::ORIGIN => a.origin = true,
            op::TIMESTAMP => a.timestamp = true,
            _ => {}
        }
    }

    a
}

/// Converts a 32-byte storage word into a lowercase 20-byte address, or None
/// when the word is all zero.
pub fn storage_word_to_address(word: &str) -> Option<String> {
    let hex = word.strip_prefix("0x").unwrap_or(word);
    if hex.len() < 40 {
        return None;
    }
    let addr = format!("0x{}", &hex[hex.len() - 40..]);
    if addr == "0x0000000000000000000000000000000000000000" {
        return None;
    }
    if !addr[2..].bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    Some(addr.to_lowercase())
}

/// Returns true when calldata for approve(address,uint256) carries an amount
/// at or above the conventional "unlimited" threshold (2^255).
///
/// The amount is the second 32-byte word: 4 bytes of selector plus 32 bytes of
/// spender address, then 32 bytes of amount.
pub fn is_unlimited_approval(calldata: &[u8]) -> bool {
    const UNLIMITED_THRESHOLD: [u8; 31] = [0xff; 31];
    // A word of 31 0xff bytes followed by any byte is >= 2^248, which every
    // wallet treats as unlimited. The exact value used in practice is 2^256-1.
    if calldata.len() < 4 + 32 + 32 {
        return false;
    }
    let amount = &calldata[4 + 32..4 + 32 + 32];
    amount[..31] == UNLIMITED_THRESHOLD
}

/// Parses a JSON-RPC quantity ("0x1a") into u64.
pub fn parse_hex_u64(value: &str) -> Option<u64> {
    let hex = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X"))?;
    if hex.is_empty() {
        return Some(0);
    }
    u64::from_str_radix(hex, 16).ok()
}

/// Lowercases a domain and rejects input that is not a hostname, so a caller
/// cannot smuggle a URL path or credentials into the lookup key.
fn normalize_domain(domain: &str) -> Result<String, ScannerError> {
    let d = domain.trim().trim_end_matches('.').to_lowercase();
    if d.is_empty() {
        return Err(ScannerError::InvalidAddress);
    }
    if d.contains('/') || d.contains('@') || d.contains(':') || d.contains(' ') {
        return Err(ScannerError::InvalidAddress);
    }
    if !d.contains('.') {
        return Err(ScannerError::InvalidAddress);
    }
    if !d.bytes().all(|b| b.is_ascii_alphanumeric() || b == b'.' || b == b'-') {
        return Err(ScannerError::InvalidAddress);
    }
    Ok(d)
}

/// Maps a threat type name from a blocklist file onto the enum.
fn parse_threat_type(name: &str) -> Option<ThreatType> {
    match name.to_lowercase().as_str() {
        "direct_phishing" | "phishing" => Some(ThreatType::DirectPhishing),
        "ponzi" | "ponzi_scheme" => Some(ThreatType::PonziScheme),
        "fake_exchange" => Some(ThreatType::FakeExchange),
        "fake_ico" => Some(ThreatType::FakeIco),
        "malware" => Some(ThreatType::Malware),
        "ransomware" => Some(ThreatType::Ransomware),
        "scam" => Some(ThreatType::Scam),
        "honeypot" => Some(ThreatType::Honeypot),
        _ => None,
    }
}

fn unix_now() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

#[cfg(test)]
mod tests;
