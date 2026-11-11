use super::phishing::{heuristic_score, inspect, normalize_host};
use super::SecurityEngine;

fn blocklist(entries: &[&str]) -> Vec<String> {
    entries.iter().map(|s| s.to_string()).collect()
}

#[test]
fn normalize_host_extracts_the_host_from_urls() {
    assert_eq!(normalize_host("https://Example.COM/path?q=1").as_deref(), Some("example.com"));
    assert_eq!(normalize_host("example.com:8443").as_deref(), Some("example.com"));
    assert_eq!(normalize_host("  EXAMPLE.com  ").as_deref(), Some("example.com"));
    // Userinfo must not survive normalisation.
    assert_eq!(normalize_host("https://user@evil.com/").as_deref(), Some("evil.com"));
    assert_eq!(normalize_host("https://user:pass@evil.com/x").as_deref(), Some("evil.com"));
}

#[test]
fn normalize_host_rejects_unusable_input() {
    assert_eq!(normalize_host(""), None);
    assert_eq!(normalize_host("   "), None);
    assert_eq!(normalize_host("localhost"), None);
    assert_eq!(normalize_host("not a host"), None);
    // A path-only string has no registrable host.
    assert_eq!(normalize_host("/just/a/path"), None);
}

#[test]
fn a_legitimate_brand_domain_is_not_flagged() {
    let verdict = inspect("https://metamask.io", &[], &[]);
    assert!(!verdict.is_phishing, "the brand's own domain was flagged: {:?}", verdict.signals);
}

#[test]
fn a_brand_subdomain_of_an_unrelated_domain_is_flagged() {
    let verdict = inspect("https://metamask.secure-login.xyz", &[], &[]);
    assert!(verdict.is_phishing, "expected a flag, score was {}", verdict.confidence);
    assert!(verdict.signals.iter().any(|s| s.contains("subdomain")));
}

#[test]
fn a_brand_embedded_in_a_lure_domain_is_flagged() {
    let verdict = inspect("https://trustwallet-airdrop-claim.com", &[], &[]);
    assert!(verdict.is_phishing, "expected a flag, score was {}", verdict.confidence);
}

#[test]
fn a_homoglyph_imitation_is_flagged() {
    let verdict = inspect("https://metamask.0rg.com", &[], &[]);
    // "0rg" is not the brand's label, so the embedded-brand rule catches it.
    assert!(verdict.is_phishing, "expected a flag, score was {}", verdict.confidence);
}

#[test]
fn a_plain_domain_earns_no_score() {
    let (score, signals) = heuristic_score("example.com");
    assert_eq!(score, 0.0, "unexpected signals: {:?}", signals);
    assert!(signals.is_empty());
}

#[test]
fn a_high_risk_tld_alone_is_not_enough_to_block() {
    let verdict = inspect("https://my-notes.xyz", &[], &[]);
    assert!(!verdict.is_phishing, "a single weak signal blocked a domain");
    assert!(verdict.confidence > 0.0, "the TLD signal should still be reported");
}

#[test]
fn a_blocklist_entry_is_conclusive() {
    let list = blocklist(&["evil.example"]);
    let verdict = inspect("https://evil.example/login", &list, &[]);
    assert!(verdict.is_phishing);
    assert_eq!(verdict.confidence, 100.0);
}

#[test]
fn a_blocklist_entry_also_matches_subdomains() {
    let list = blocklist(&["evil.example"]);
    let verdict = inspect("https://wallet.evil.example", &list, &[]);
    assert!(verdict.is_phishing);
}

#[test]
fn an_allowlist_entry_overrides_the_heuristics() {
    let allow = blocklist(&["metamask.secure-login.xyz"]);
    let verdict = inspect("https://metamask.secure-login.xyz", &[], &allow);
    assert!(!verdict.is_phishing, "allowlist did not suppress the heuristics");
    assert_eq!(verdict.confidence, 0.0);
}

#[test]
fn an_allowlist_entry_overrides_the_blocklist() {
    let block = blocklist(&["example.com"]);
    let allow = blocklist(&["example.com"]);
    let verdict = inspect("https://example.com", &block, &allow);
    assert!(!verdict.is_phishing, "allowlist must take precedence over blocklist");
}

#[test]
fn a_raw_ip_host_is_flagged() {
    let verdict = inspect("http://192.168.1.1/login", &[], &[]);
    assert!(verdict.signals.iter().any(|s| s.contains("IP address")));
}

#[test]
fn a_punycode_host_is_flagged() {
    let verdict = inspect("https://xn--metamask-9d7b.com", &[], &[]);
    assert!(verdict.signals.iter().any(|s| s.contains("punycode")));
}

#[test]
fn an_unparseable_input_is_reported_without_claiming_phishing() {
    let verdict = inspect("not a host", &[], &[]);
    assert!(!verdict.is_phishing);
    assert_eq!(verdict.confidence, 0.0);
    assert!(verdict.signals.iter().any(|s| s.contains("not a usable host")));
}

#[test]
fn the_engine_blocks_domains_an_operator_adds() {
    let engine = SecurityEngine::new();

    // Nothing is blocked out of the box: the engine ships no hardcoded list.
    assert!(!engine.detect_phishing("evil.example").is_phishing);

    engine.block_domain("evil.example");
    let verdict = engine.detect_phishing("evil.example");
    assert!(verdict.is_phishing);
    assert!(verdict.recommendations.iter().any(|r| r.contains("seed phrase")));
}

#[test]
fn the_engine_flags_a_phishing_heuristic_without_any_blocklist() {
    let engine = SecurityEngine::new();
    let verdict = engine.detect_phishing("https://metamask.secure-login.xyz");
    assert!(verdict.is_phishing, "heuristics did not flag the domain");
    assert!(!verdict.matched_patterns.is_empty());
}

#[test]
fn the_engine_leaves_a_legitimate_domain_alone() {
    let engine = SecurityEngine::new();
    let verdict = engine.detect_phishing("https://app.uniswap.org");
    assert!(!verdict.is_phishing, "legitimate domain flagged: {:?}", verdict.matched_patterns);
}

#[test]
fn the_engine_normalises_before_matching_the_blocklist() {
    let engine = SecurityEngine::new();
    engine.block_domain("evil.example");

    let verdict = engine.detect_phishing("https://EVIL.example/path");
    assert!(verdict.is_phishing, "case and path were not normalised");
    assert_eq!(verdict.domain, "evil.example");
}
