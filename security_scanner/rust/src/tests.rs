//! Tests for the security scanner.
//!
//! The opcode analyser and the hex helpers are tested directly. The RPC paths
//! are tested against a real HTTP server that speaks JSON-RPC, so request
//! shaping, response parsing and every failure path are covered without
//! mocking the HTTP client.

use super::*;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpListener;

// ============================================================================
// A minimal JSON-RPC server
// ============================================================================

/// Spawns a loopback HTTP server that answers each request with the body the
/// closure returns. The closure receives the parsed JSON-RPC request.
async fn spawn_rpc<F>(handler: F) -> String
where
    F: Fn(serde_json::Value) -> (u16, String) + Send + Sync + 'static,
{
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    let handler = std::sync::Arc::new(handler);

    tokio::spawn(async move {
        loop {
            let (mut socket, _) = match listener.accept().await {
                Ok(s) => s,
                Err(_) => return,
            };
            let handler = handler.clone();

            tokio::spawn(async move {
                let mut buf = vec![0u8; 8192];
                let n = match socket.read(&mut buf).await {
                    Ok(n) => n,
                    Err(_) => return,
                };
                let request = String::from_utf8_lossy(&buf[..n]).to_string();

                // Pull the JSON body out of the HTTP request.
                let body = request
                    .split("\r\n\r\n")
                    .nth(1)
                    .unwrap_or("")
                    .to_string();
                let parsed: serde_json::Value =
                    serde_json::from_str(&body).unwrap_or(serde_json::Value::Null);

                let (status, response_body) = handler(parsed);
                let response = format!(
                    "HTTP/1.1 {} OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                    status,
                    response_body.len(),
                    response_body
                );
                let _ = socket.write_all(response.as_bytes()).await;
                let _ = socket.flush().await;
            });
        }
    });

    format!("http://{}", addr)
}

fn rpc_ok(result: serde_json::Value) -> (u16, String) {
    (
        200,
        serde_json::json!({"jsonrpc": "2.0", "id": 1, "result": result}).to_string(),
    )
}

fn scanner_for(chain_id: u64, url: String) -> SecurityScanner {
    let config = ScannerConfig::default()
        .with_rpc_spec(&format!("{}={}", chain_id, url))
        .expect("rpc spec should parse");
    SecurityScanner::new(config)
}

// ============================================================================
// Hex decoding
// ============================================================================

#[test]
fn decode_hex_handles_prefixes_and_case() {
    assert_eq!(decode_hex("0x"), Some(vec![]));
    assert_eq!(decode_hex(""), Some(vec![]));
    assert_eq!(decode_hex("0x6001"), Some(vec![0x60, 0x01]));
    assert_eq!(decode_hex("6001"), Some(vec![0x60, 0x01]));
    assert_eq!(decode_hex("0x6001FF"), Some(vec![0x60, 0x01, 0xff]));
}

#[test]
fn decode_hex_pads_odd_length() {
    // A node returning an odd digit count is malformed; the leading nibble is
    // padded so the scan can proceed.
    assert_eq!(decode_hex("0x601"), Some(vec![0x06, 0x01]));
}

#[test]
fn decode_hex_rejects_non_hex() {
    assert_eq!(decode_hex("0xzz"), None);
    assert_eq!(decode_hex("0xhello"), None);
    assert_eq!(decode_hex("0x12 34"), None);
}

// ============================================================================
// Opcode analysis
// ============================================================================

#[test]
fn analyze_opcodes_finds_each_opcode() {
    // PUSH1 0x01, then one of each opcode under test.
    let code = vec![
        0x60, 0x01, // PUSH1 0x01
        0xf1, // CALL
        0xf4, // DELEGATECALL
        0xfa, // STATICCALL
        0xf0, // CREATE
        0xff, // SELFDESTRUCT
        0xfd, // REVERT
        0x57, // JUMPI
        0x54, // SLOAD
        0x55, // SSTORE
        0x33, // CALLER
        0x32, // ORIGIN
        0x42, // TIMESTAMP
    ];

    let a = analyze_opcodes(&code);
    assert!(a.call);
    assert!(a.delegatecall);
    assert!(a.staticcall);
    assert!(a.creates_contract);
    assert!(a.selfdestruct);
    assert!(a.revert);
    assert!(a.conditional_jump);
    assert!(a.storage_read);
    assert!(a.storage_write);
    assert!(a.caller);
    assert!(a.origin);
    assert!(a.timestamp);
}

#[test]
fn analyze_opcodes_skips_push_immediates() {
    // PUSH32 whose 32-byte operand is full of opcode bytes that must not be
    // read as instructions. This is the case a substring search gets wrong.
    let mut code = vec![0x7f];
    code.extend(std::iter::repeat(0xf4u8).take(32)); // DELEGATECALL bytes
    code.push(0x00); // STOP

    let a = analyze_opcodes(&code);
    assert!(!a.delegatecall, "PUSH32 operand was misread as an instruction");
    assert!(!a.call);
    assert!(!a.selfdestruct);
}

#[test]
fn analyze_opcodes_skips_short_push_immediates() {
    // PUSH2 0xf1f4, then STOP. Neither byte is an instruction.
    let code = vec![0x61, 0xf1, 0xf4, 0x00];
    let a = analyze_opcodes(&code);
    assert!(!a.call);
    assert!(!a.delegatecall);
}

#[test]
fn analyze_opcodes_handles_truncated_push() {
    // PUSH32 with no operand bytes: the walk must not panic.
    let code = vec![0x7f];
    let a = analyze_opcodes(&code);
    assert_eq!(a, OpcodeAnalysis::default());
}

#[test]
fn analyze_opcodes_empty_input() {
    assert_eq!(analyze_opcodes(&[]), OpcodeAnalysis::default());
}

// ============================================================================
// Storage words and quantities
// ============================================================================

#[test]
fn storage_word_to_address_extracts_low_20_bytes() {
    let word = "0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd";
    assert_eq!(
        storage_word_to_address(word),
        Some("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd".to_string())
    );
}

#[test]
fn storage_word_to_address_rejects_zero_and_short() {
    let zero = "0x0000000000000000000000000000000000000000000000000000000000000000";
    assert_eq!(storage_word_to_address(zero), None);
    assert_eq!(storage_word_to_address("0xabcd"), None);
}

#[test]
fn parse_hex_u64_handles_quantities() {
    assert_eq!(parse_hex_u64("0x0"), Some(0));
    assert_eq!(parse_hex_u64("0x1a"), Some(26));
    assert_eq!(parse_hex_u64("0x5208"), Some(21000));
    assert_eq!(parse_hex_u64("0X10"), Some(16));
    assert_eq!(parse_hex_u64("not-hex"), None);
    assert_eq!(parse_hex_u64("0x"), Some(0));
}

// ============================================================================
// Unlimited approval detection
// ============================================================================

#[test]
fn unlimited_approval_detection() {
    // approve(spender, 2^256-1)
    let mut calldata = vec![0x09, 0x5e, 0xa7, 0xb3];
    calldata.extend([0u8; 32]); // spender
    calldata.extend([0xffu8; 32]); // amount
    assert!(is_unlimited_approval(&calldata));

    // approve(spender, 1000) is bounded.
    let mut bounded = vec![0x09, 0x5e, 0xa7, 0xb3];
    bounded.extend([0u8; 32]);
    bounded.extend([0u8; 31]);
    bounded.push(0xe8);
    assert!(!is_unlimited_approval(&bounded));

    // Truncated calldata is not an unlimited approval.
    assert!(!is_unlimited_approval(&[0x09, 0x5e, 0xa7, 0xb3]));
    assert!(!is_unlimited_approval(&[]));
}

// ============================================================================
// Domain normalisation
// ============================================================================

#[test]
fn normalize_domain_accepts_real_hostnames() {
    assert_eq!(normalize_domain("Example.COM").unwrap(), "example.com");
    assert_eq!(normalize_domain("app.example.com.").unwrap(), "app.example.com");
    assert_eq!(normalize_domain("  example.co.uk  ").unwrap(), "example.co.uk");
}

#[test]
fn normalize_domain_rejects_non_hostnames() {
    for bad in [
        "",
        "   ",
        "no-dot",
        "http://example.com/path",
        "user@example.com",
        "example.com:8080",
        "exa mple.com",
        "example.com/path",
    ] {
        assert!(
            normalize_domain(bad).is_err(),
            "{:?} should have been rejected",
            bad
        );
    }
}

#[test]
fn parse_threat_type_maps_names() {
    assert_eq!(parse_threat_type("phishing"), Some(ThreatType::DirectPhishing));
    assert_eq!(parse_threat_type("DRAINER"), None);
    assert_eq!(parse_threat_type("scam"), Some(ThreatType::Scam));
    assert_eq!(parse_threat_type("honeypot"), Some(ThreatType::Honeypot));
    assert_eq!(parse_threat_type("nonsense"), None);
}

// ============================================================================
// RPC configuration
// ============================================================================

#[test]
fn rpc_spec_parsing() {
    let config = ScannerConfig::default()
        .with_rpc_spec("1=https://eth.example,137=http://polygon.example:8545")
        .unwrap();
    assert_eq!(
        config.rpc_endpoints.get(&1).map(String::as_str),
        Some("https://eth.example")
    );
    assert_eq!(
        config.rpc_endpoints.get(&137).map(String::as_str),
        Some("http://polygon.example:8545")
    );
}

#[test]
fn rpc_spec_rejects_malformed_entries() {
    for bad in ["no-equals", "=https://x.example", "abc=https://x.example", "1=", "1=ftp://x", "1=x.example"] {
        assert!(
            ScannerConfig::default().with_rpc_spec(bad).is_err(),
            "{:?} should have been rejected",
            bad
        );
    }

    let empty = ScannerConfig::default().with_rpc_spec("  ").unwrap();
    assert!(empty.rpc_endpoints.is_empty());
}

// ============================================================================
// Contract analysis over a real RPC server
// ============================================================================

/// Runtime bytecode for a plausible token: large enough to clear the size
/// check, containing CALL and JUMPI so the "unchecked call" rule stays quiet.
fn plausible_token_bytecode() -> String {
    let mut code = Vec::new();
    // PUSH1 .. to pad the contract past the minimum size.
    for _ in 0..220 {
        code.extend_from_slice(&[0x60, 0x00]); // PUSH1 0x00
    }
    code.push(0xf1); // CALL
    code.push(0x57); // JUMPI
    code.push(0x00); // STOP
    format!("0x{}", hex::encode(code))
}

#[tokio::test]
async fn analyze_contract_reports_a_clean_token() {
    let bytecode = plausible_token_bytecode();
    let url = spawn_rpc(move |req| {
        let method = req["method"].as_str().unwrap_or("").to_string();
        match method.as_str() {
            "eth_getCode" => rpc_ok(serde_json::json!(bytecode)),
            "eth_getStorageAt" => rpc_ok(serde_json::json!(
                "0x0000000000000000000000000000000000000000000000000000000000000000"
            )),
            "eth_getTransactionCount" => rpc_ok(serde_json::json!("0x2a")),
            _ => rpc_ok(serde_json::Value::Null),
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let analysis = scanner
        .analyze_contract("0x1111111111111111111111111111111111111111", 1)
        .await
        .expect("analysis should succeed");

    assert_eq!(analysis.score, 100, "a clean contract should score 100");
    assert_eq!(analysis.risk_level, RiskLevel::Safe);
    assert!(analysis.issues.is_empty(), "unexpected issues: {:?}", analysis.issues);
    assert!(!analysis.is_honeypot);
    assert!(!analysis.has_backdoor);
    assert_eq!(analysis.tx_count, 42);
}

#[tokio::test]
async fn analyze_contract_flags_selfdestruct() {
    let mut code = vec![0x60, 0x00]; // PUSH1 0x00
    for _ in 0..220 {
        code.extend_from_slice(&[0x60, 0x00]);
    }
    code.push(0x54); // SLOAD
    code.push(0xff); // SELFDESTRUCT
    let bytecode = format!("0x{}", hex::encode(code));

    let url = spawn_rpc(move |req| {
        let method = req["method"].as_str().unwrap_or("").to_string();
        match method.as_str() {
            "eth_getCode" => rpc_ok(serde_json::json!(bytecode)),
            _ => rpc_ok(serde_json::Value::Null),
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let analysis = scanner
        .analyze_contract("0x2222222222222222222222222222222222222222", 1)
        .await
        .unwrap();

    assert!(analysis.has_backdoor, "SELFDESTRUCT behind storage is a backdoor");
    assert!(
        analysis.issues.iter().any(|i| i.category == IssueCategory::Backdoor),
        "a backdoor issue should be recorded"
    );
    assert!(analysis.score < 100);
}

#[tokio::test]
async fn analyze_contract_flags_delegatecall() {
    let mut code = Vec::new();
    for _ in 0..220 {
        code.extend_from_slice(&[0x60, 0x00]);
    }
    code.push(0xf4); // DELEGATECALL
    code.push(0x57); // JUMPI
    let bytecode = format!("0x{}", hex::encode(code));

    let url = spawn_rpc(move |req| {
        if req["method"] == "eth_getCode" {
            rpc_ok(serde_json::json!(bytecode))
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let analysis = scanner
        .analyze_contract("0x3333333333333333333333333333333333333333", 1)
        .await
        .unwrap();

    assert!(
        analysis.issues.iter().any(|i| i.title.contains("DELEGATECALL")),
        "DELEGATECALL should be reported"
    );
}

#[tokio::test]
async fn analyze_contract_flags_tiny_contract() {
    // 3 bytes of runtime code: cannot implement a token interface.
    let url = spawn_rpc(move |req| {
        if req["method"] == "eth_getCode" {
            rpc_ok(serde_json::json!("0x600160"))
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let analysis = scanner
        .analyze_contract("0x4444444444444444444444444444444444444444", 1)
        .await
        .unwrap();

    assert!(
        analysis.issues.iter().any(|i| i.category == IssueCategory::FakeToken),
        "a 3-byte contract should be flagged"
    );
}

#[tokio::test]
async fn analyze_contract_rejects_invalid_address() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    let err = scanner.analyze_contract("not-an-address", 1).await;
    assert!(matches!(err, Err(ScannerError::InvalidAddress)));
}

#[tokio::test]
async fn analyze_contract_reports_unconfigured_chain() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    let err = scanner
        .analyze_contract("0x1111111111111111111111111111111111111111", 999)
        .await;
    match err {
        Err(ScannerError::ChainNotConfigured(999)) => {}
        other => panic!("expected ChainNotConfigured(999), got {:?}", other),
    }
}

#[tokio::test]
async fn analyze_contract_surfaces_rpc_error() {
    let url = spawn_rpc(|_req| {
        (
            200,
            serde_json::json!({
                "jsonrpc": "2.0",
                "id": 1,
                "error": {"code": -32000, "message": "execution reverted"}
            })
            .to_string(),
        )
    })
    .await;

    let scanner = scanner_for(1, url);
    let err = scanner
        .analyze_contract("0x1111111111111111111111111111111111111111", 1)
        .await;
    match err {
        Err(ScannerError::NetworkError(msg)) => {
            assert!(msg.contains("execution reverted"), "message was {:?}", msg);
        }
        other => panic!("expected a network error, got {:?}", other),
    }
}

#[tokio::test]
async fn analyze_contract_surfaces_http_error_status() {
    let url = spawn_rpc(|_req| (500, "server error".to_string())).await;

    let scanner = scanner_for(1, url);
    let err = scanner
        .analyze_contract("0x1111111111111111111111111111111111111111", 1)
        .await;
    match err {
        Err(ScannerError::NetworkError(msg)) => assert!(msg.contains("500")),
        other => panic!("expected a network error, got {:?}", other),
    }
}

#[tokio::test]
async fn analyze_contract_caches_results() {
    use std::sync::atomic::{AtomicUsize, Ordering};
    use std::sync::Arc;

    let calls = Arc::new(AtomicUsize::new(0));
    let calls_in_handler = calls.clone();

    let url = spawn_rpc(move |req| {
        calls_in_handler.fetch_add(1, Ordering::SeqCst);
        if req["method"] == "eth_getCode" {
            rpc_ok(serde_json::json!(plausible_token_bytecode()))
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let addr = "0x5555555555555555555555555555555555555555";

    scanner.analyze_contract(addr, 1).await.unwrap();
    let after_first = calls.load(Ordering::SeqCst);
    assert!(after_first > 0, "the first analysis should hit the node");

    scanner.analyze_contract(addr, 1).await.unwrap();
    assert_eq!(
        calls.load(Ordering::SeqCst),
        after_first,
        "the second analysis should be served from cache"
    );
}

// ============================================================================
// Phishing
// ============================================================================

#[tokio::test]
async fn blocklist_is_authoritative() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    let count = scanner
        .load_domain_blocklist(r#"["evil.example.com", "phish.example.org"]"#)
        .await
        .unwrap();
    assert_eq!(count, 2);
    assert_eq!(scanner.blocklist_size().await, 2);

    let detection = scanner.check_phishing("EVIL.example.com").await.unwrap();
    assert!(detection.is_phishing);
    assert_eq!(detection.confidence, 1.0);
    assert_eq!(detection.threat_type, Some(ThreatType::DirectPhishing));
}

#[tokio::test]
async fn blocklist_object_form_carries_metadata() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    let count = scanner
        .load_domain_blocklist(
            r#"{"drainer.example.com": {"threat_type": "honeypot", "reports_count": 12}}"#,
        )
        .await
        .unwrap();
    assert_eq!(count, 1);

    let detection = scanner.check_phishing("drainer.example.com").await.unwrap();
    assert!(detection.is_phishing);
    assert_eq!(detection.threat_type, Some(ThreatType::Honeypot));
    assert_eq!(detection.reports_count, 12);
}

#[tokio::test]
async fn blocklist_rejects_malformed_files() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    for bad in ["not json", r#"[123]"#, r#""a string""#, r#"["not a domain"]"#] {
        assert!(
            scanner.load_domain_blocklist(bad).await.is_err(),
            "{:?} should have been rejected",
            bad
        );
    }
}

#[tokio::test]
async fn unlisted_domain_scores_by_heuristics_not_by_loop_repetition() {
    let scanner = SecurityScanner::new(ScannerConfig::default());

    // A plain domain is not phishing.
    let clean = scanner.check_phishing("example.com").await.unwrap();
    assert!(!clean.is_phishing);
    assert!(clean.confidence < 0.7);

    // A brand-impersonating domain is.
    let impostor = scanner.check_phishing("metamask-wallet.xyz").await.unwrap();
    assert!(impostor.is_phishing, "confidence was {}", impostor.confidence);
    assert!(impostor.confidence <= 1.0);
}

#[tokio::test]
async fn phishing_confidence_is_bounded() {
    let scanner = SecurityScanner::new(ScannerConfig::default());

    // Every heuristic firing at once must still not exceed 1.0, and the old
    // loop-repetition bug produced values well above it.
    let worst = scanner
        .check_phishing("metamask-uniswap-defi-swap-login-app-airdrop-claim.xyz")
        .await
        .unwrap();
    assert!(
        worst.confidence <= 1.0,
        "confidence {} exceeded 1.0",
        worst.confidence
    );
}

#[tokio::test]
async fn check_phishing_rejects_non_hostnames() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    assert!(scanner.check_phishing("http://example.com/x").await.is_err());
    assert!(scanner.check_phishing("nodot").await.is_err());
    assert!(scanner.check_phishing("").await.is_err());
}

// ============================================================================
// Transaction screening
// ============================================================================

#[tokio::test]
async fn analyze_transaction_flags_unlimited_approval() {
    let url = spawn_rpc(|_req| rpc_ok(serde_json::json!("0x1"))).await;
    let scanner = scanner_for(1, url);

    // approve(spender, 2^256-1)
    let mut calldata = vec![0x09, 0x5e, 0xa7, 0xb3];
    calldata.extend([0u8; 32]);
    calldata.extend([0xffu8; 32]);
    let tx = format!("0x{}", hex::encode(calldata));

    let risk = scanner.analyze_transaction(&tx, 1).await.unwrap();
    assert_eq!(risk.risk_score, 30);
    assert!(risk.warnings.iter().any(|w| w.message.contains("Unlimited")));
}

#[tokio::test]
async fn analyze_transaction_ignores_selectors_inside_arguments() {
    let url = spawn_rpc(|_req| rpc_ok(serde_json::json!("0x1"))).await;
    let scanner = scanner_for(1, url);

    // A transfer whose *argument* contains the approve selector bytes. The
    // old implementation searched the whole string and reported an approval.
    let mut calldata = vec![0xa9, 0x05, 0x9c, 0xbb]; // transfer(address,uint256)
    calldata.extend([0u8; 31]);
    calldata.push(0x01);
    calldata.extend_from_slice(&[0x09, 0x5e, 0xa7, 0xb3]); // selector bytes as data
    calldata.extend([0u8; 28]);
    let tx = format!("0x{}", hex::encode(calldata));

    let risk = scanner.analyze_transaction(&tx, 1).await.unwrap();
    assert!(
        risk.warnings.is_empty(),
        "no warning should fire for a plain transfer, got {:?}",
        risk.warnings
    );
    assert_eq!(risk.risk_score, 0);
}

#[tokio::test]
async fn analyze_transaction_reports_mev_as_unknown() {
    let url = spawn_rpc(|_req| rpc_ok(serde_json::json!("0x1"))).await;
    let scanner = scanner_for(1, url);

    let risk = scanner.analyze_transaction("0x", 1).await.unwrap();
    assert_eq!(risk.is_sandwich, None);
    assert_eq!(risk.is_front_run, None);
    assert_eq!(risk.targeted_by_mev, None);
    assert!(
        risk.simulation_result.is_none(),
        "no simulation should be claimed without running one"
    );
}

#[tokio::test]
async fn analyze_transaction_rejects_non_hex() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    assert!(scanner.analyze_transaction("not-hex", 1).await.is_err());
}

// ============================================================================
// Simulation and gas
// ============================================================================

#[tokio::test]
async fn simulate_transaction_reports_a_revert() {
    let url = spawn_rpc(|req| {
        if req["method"] == "eth_call" {
            (
                200,
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 1,
                    "error": {"code": 3, "message": "execution reverted: insufficient balance"}
                })
                .to_string(),
            )
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let sim = scanner
        .simulate_transaction(
            "0x1111111111111111111111111111111111111111",
            "0x2222222222222222222222222222222222222222",
            "0",
            "0xa9059cbb",
            1,
        )
        .await
        .unwrap();

    assert!(!sim.success);
    assert!(sim.reverted);
    assert!(sim
        .error_message
        .as_deref()
        .unwrap()
        .contains("insufficient balance"));
}

#[tokio::test]
async fn simulate_transaction_returns_real_return_data() {
    let url = spawn_rpc(|req| {
        if req["method"] == "eth_call" {
            rpc_ok(serde_json::json!("0x0000000000000000000000000000000000000000000000000000000000000001"))
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let sim = scanner
        .simulate_transaction(
            "0x1111111111111111111111111111111111111111",
            "0x2222222222222222222222222222222222222222",
            "0",
            "0x70a08231",
            1,
        )
        .await
        .unwrap();

    assert!(sim.success);
    assert!(!sim.reverted);
    assert_eq!(sim.logs.len(), 1);
    assert!(sim.logs[0].data.ends_with("0001"));
}

#[tokio::test]
async fn simulate_transaction_validates_addresses() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    assert!(scanner
        .simulate_transaction("bad", "0x2222222222222222222222222222222222222222", "0", "0x", 1)
        .await
        .is_err());
}

#[tokio::test]
async fn estimate_gas_parses_a_quantity() {
    let url = spawn_rpc(|req| {
        if req["method"] == "eth_estimateGas" {
            rpc_ok(serde_json::json!("0x5208"))
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    let gas = scanner
        .estimate_gas(
            "0x1111111111111111111111111111111111111111",
            "0x2222222222222222222222222222222222222222",
            "0",
            "0x",
            1,
        )
        .await
        .unwrap();
    assert_eq!(gas, 21000);
}

#[tokio::test]
async fn gas_price_parses_a_quantity() {
    let url = spawn_rpc(|req| {
        if req["method"] == "gasPrice" {
            rpc_ok(serde_json::json!("0x3b9aca00"))
        } else {
            rpc_ok(serde_json::Value::Null)
        }
    })
    .await;

    let scanner = scanner_for(1, url);
    assert_eq!(scanner.gas_price(1).await.unwrap(), 1_000_000_000);
}

#[tokio::test]
async fn gas_price_fails_on_unconfigured_chain() {
    let scanner = SecurityScanner::new(ScannerConfig::default());
    assert!(scanner.gas_price(1).await.is_err());
}

// ============================================================================
// Risk levels
// ============================================================================

#[test]
fn safety_score_boundaries() {
    // A contract's safety score: 100 is clean, 0 is hostile.
    assert_eq!(RiskLevel::from_safety_score(100), RiskLevel::Safe);
    assert_eq!(RiskLevel::from_safety_score(81), RiskLevel::Safe);
    assert_eq!(RiskLevel::from_safety_score(80), RiskLevel::Low);
    assert_eq!(RiskLevel::from_safety_score(61), RiskLevel::Low);
    assert_eq!(RiskLevel::from_safety_score(60), RiskLevel::Medium);
    assert_eq!(RiskLevel::from_safety_score(41), RiskLevel::Medium);
    assert_eq!(RiskLevel::from_safety_score(40), RiskLevel::High);
    assert_eq!(RiskLevel::from_safety_score(21), RiskLevel::High);
    assert_eq!(RiskLevel::from_safety_score(20), RiskLevel::Critical);
    assert_eq!(RiskLevel::from_safety_score(0), RiskLevel::Critical);
}

#[test]
fn risk_score_boundaries() {
    // A transaction's risk score: 0 is clean, 100 is hostile.
    assert_eq!(RiskLevel::from_risk_score(0), RiskLevel::Safe);
    assert_eq!(RiskLevel::from_risk_score(20), RiskLevel::Safe);
    assert_eq!(RiskLevel::from_risk_score(21), RiskLevel::Low);
    assert_eq!(RiskLevel::from_risk_score(40), RiskLevel::Low);
    assert_eq!(RiskLevel::from_risk_score(41), RiskLevel::Medium);
    assert_eq!(RiskLevel::from_risk_score(60), RiskLevel::Medium);
    assert_eq!(RiskLevel::from_risk_score(61), RiskLevel::High);
    assert_eq!(RiskLevel::from_risk_score(80), RiskLevel::High);
    assert_eq!(RiskLevel::from_risk_score(81), RiskLevel::Critical);
    assert_eq!(RiskLevel::from_risk_score(100), RiskLevel::Critical);
}
