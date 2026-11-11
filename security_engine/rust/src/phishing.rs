//! Phishing and malicious-site detection.
//!
//! The previous implementation matched a handful of literal strings, which
//! detects nothing an attacker would actually register. This module scores a
//! domain from independent signals and combines that with an operator-managed
//! blocklist and allowlist. Every signal is deterministic and explainable, so
//! an operator can see why a domain was flagged.

use serde::{Deserialize, Serialize};

/// Outcome of inspecting a domain.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhishingVerdict {
    pub domain: String,
    pub is_phishing: bool,
    pub confidence: f64,
    pub signals: Vec<String>,
    pub recommendations: Vec<String>,
}

/// Brand names attackers impersonate by combining with a lure word or by
/// registering a look-alike domain.
const IMPERSONATED_BRANDS: &[&str] = &[
    "metamask", "trustwallet", "phantom", "coinbase", "binance", "okx", "bitget",
    "safepal", "rabby", "ledger", "trezor", "exodus", "atomicwallet", "tigerwallet",
    "walletconnect", "uniswap", "pancakeswap", "opensea", "aave", "curve",
];

/// Words that appear in giveaway and drainer campaigns.
const LURE_WORDS: &[&str] = &[
    "airdrop", "claim", "reward", "bonus", "giveaway", "gift", "free", "prize",
    "winner", "eligible", "verify-wallet", "validate", "restore", "unlock",
    "migration", "sync", "support", "helpdesk", "refund", "bonus-claim",
];

/// Top-level domains with a poor reputation for legitimate finance sites.
const HIGH_RISK_TLDS: &[&str] = &[
    ".xyz", ".top", ".click", ".link", ".gq", ".cf", ".tk", ".ml", ".ga", ".work",
    ".loan", ".rest", ".buzz", ".monster", ".quest", ".cyou", ".sbs",
];

/// Substrings that let a domain imitate a brand visually.
const HOMOGLYPH_RISK: &[&str] = &["0", "1", "5", "rn", "vv"];

/// Common top-level domains that a homoglyph attack imitates, for example
/// `0rg` in place of `org`.
const IMITATED_TLDS: &[&str] = &["com", "org", "net", "io", "co", "app", "dev"];

/// Normalise a URL or host into a bare lowercase host.
///
/// Returns `None` for input that is not a usable host, so callers never score
/// a domain derived from unparseable text.
pub fn normalize_host(input: &str) -> Option<String> {
    let trimmed = input.trim();
    if trimmed.is_empty() {
        return None;
    }

    let without_scheme = trimmed
        .split_once("://")
        .map(|(_, rest)| rest)
        .unwrap_or(trimmed);

    let host_and_port = without_scheme
        .split(['/', '?', '#'])
        .next()
        .unwrap_or("");

    // Drop userinfo, which is a common obfuscation trick.
    let host = host_and_port
        .rsplit_once('@')
        .map(|(_, host)| host)
        .unwrap_or(host_and_port);

    let host = host.split(':').next().unwrap_or("");

    let host = host.trim_matches('.').to_lowercase();

    if host.is_empty() || !host.contains('.') {
        return None;
    }
    if host.len() > 253 {
        return None;
    }
    if !host
        .chars()
        .all(|c| c.is_ascii_alphanumeric() || c == '.' || c == '-' || c == '_')
    {
        return None;
    }

    Some(host)
}

/// Return the registrable-ish label set of a host, split on dots.
fn labels(host: &str) -> Vec<&str> {
    host.split('.').filter(|s| !s.is_empty()).collect()
}

/// Detect a brand name embedded in a domain that is not the brand's own site.
///
/// `example-metamask.xyz` impersonates MetaMask; `metamask.io` does not,
/// because the brand is the registrable label.
fn impersonation_signal(host: &str) -> Option<(String, f64)> {
    let parts = labels(host);
    if parts.len() < 2 {
        return None;
    }

    // The registrable label is the second-to-last for a simple host.
    let registrable = parts[parts.len() - 2];

    for brand in IMPERSONATED_BRANDS {
        if !host.contains(brand) {
            continue;
        }

        // The brand's own domain uses the brand as the registrable label.
        if registrable == *brand {
            return None;
        }

        // Brand as a subdomain label of an unrelated domain.
        if parts[..parts.len() - 2].iter().any(|p| *p == *brand) {
            return Some((format!("impersonates '{brand}' as a subdomain"), 45.0));
        }

        // Brand embedded inside a longer registrable label.
        if registrable.contains(brand) && registrable != *brand {
            return Some((
                format!("brand '{brand}' embedded in the registrable domain '{registrable}'"),
                40.0,
            ));
        }
    }

    // A look-alike TLD on a near-miss spelling of a brand.
    for brand in IMPERSONATED_BRANDS {
        let stripped: String = registrable.chars().filter(|c| c.is_ascii_alphabetic()).collect();
        let brand_alpha: String = brand.chars().filter(|c| c.is_ascii_alphabetic()).collect();
        if stripped != registrable && stripped == brand_alpha {
            return Some((
                format!("'{registrable}' differs from '{brand}' only by non-letters"),
                45.0,
            ));
        }
    }

    None
}

/// Detect homoglyph substitutions that imitate a brand.
fn homoglyph_signal(host: &str) -> Option<(String, f64)> {
    let parts = labels(host);
    if parts.len() < 2 {
        return None;
    }
    let registrable = parts[parts.len() - 2];

    if !HOMOGLYPH_RISK.iter().any(|token| registrable.contains(token)) {
        return None;
    }

    for brand in IMPERSONATED_BRANDS {
        let collapsed: String = registrable
            .replace('0', "o")
            .replace('1', "l")
            .replace('5', "s")
            .replace("rn", "m")
            .replace("vv", "w");
        if collapsed == *brand {
            return Some((format!("homoglyph imitation of '{brand}'"), 45.0));
        }
    }

    None
}

/// Detect a top-level domain that is a homoglyph of a common one.
///
/// `metamask.0rg.com` uses `0rg` where `org` is expected. That label is never
/// a legitimate TLD, so its presence is conclusive evidence of an imitation.
fn tld_homoglyph_signal(host: &str) -> Option<(String, f64)> {
    for label in labels(host) {
        // A real TLD needs no inspection.
        if IMITATED_TLDS.contains(&label) {
            continue;
        }

        let collapsed: String = label
            .replace('0', "o")
            .replace('1', "l")
            .replace('5', "s")
            .replace('3', "e")
            .replace("rn", "m");

        if collapsed != label && IMITATED_TLDS.contains(&collapsed.as_str()) {
            return Some((
                format!("label '{label}' imitates the top-level domain '{collapsed}'"),
                40.0,
            ));
        }
    }

    None
}

/// Detect a lure word in the host.
fn lure_signal(host: &str) -> Option<(String, f64)> {
    for word in LURE_WORDS {
        if host.contains(word) {
            return Some((format!("lure word '{word}' in the host"), 25.0));
        }
    }
    None
}

/// Detect a high-risk TLD.
fn tld_signal(host: &str) -> Option<(String, f64)> {
    for tld in HIGH_RISK_TLDS {
        if host.ends_with(tld) {
            return Some((format!("high-risk top-level domain '{tld}'"), 15.0));
        }
    }
    None
}

/// Detect excessive subdomain depth, used to push the real domain out of view.
fn depth_signal(host: &str) -> Option<(String, f64)> {
    let parts = labels(host);
    if parts.len() > 4 {
        return Some((format!("{} subdomain labels", parts.len()), 15.0));
    }
    None
}

/// Detect a punycode label, which can hide non-ASCII homoglyphs.
fn punycode_signal(host: &str) -> Option<(String, f64)> {
    if labels(host).iter().any(|p| p.starts_with("xn--")) {
        return Some(("punycode label present (possible homoglyph)".to_string(), 20.0));
    }
    None
}

/// Detect a raw IP address used as the host.
fn raw_ip_signal(host: &str) -> Option<(String, f64)> {
    let parts = labels(host);
    if parts.len() == 4 && parts.iter().all(|p| p.parse::<u8>().is_ok()) {
        return Some(("raw IP address used as the host".to_string(), 30.0));
    }
    None
}

/// Score a host on its own, ignoring operator-managed lists.
///
/// The score is the sum of the matching signal weights, capped at 100.
pub fn heuristic_score(host: &str) -> (f64, Vec<String>) {
    let mut score = 0.0;
    let mut signals = Vec::new();

    let checks: [fn(&str) -> Option<(String, f64)>; 7] = [
        impersonation_signal,
        homoglyph_signal,
        tld_homoglyph_signal,
        lure_signal,
        tld_signal,
        depth_signal,
        punycode_signal,
    ];

    for check in checks {
        if let Some((description, weight)) = check(host) {
            signals.push(description);
            score += weight;
        }
    }

    if let Some((description, weight)) = raw_ip_signal(host) {
        signals.push(description);
        score += weight;
    }

    (score.min(100.0), signals)
}

/// Inspect a URL or host against the operator's lists and the heuristics.
///
/// A blocklist entry is conclusive. An allowlist entry is conclusive in the
/// other direction and suppresses the heuristics, so an operator can clear a
/// false positive permanently.
pub fn inspect(
    input: &str,
    blocklist: &[String],
    allowlist: &[String],
) -> PhishingVerdict {
    let host = match normalize_host(input) {
        Some(host) => host,
        None => {
            return PhishingVerdict {
                domain: input.trim().to_string(),
                is_phishing: false,
                confidence: 0.0,
                signals: vec!["input is not a usable host".to_string()],
                recommendations: Vec::new(),
            }
        }
    };

    let listed = |list: &[String]| {
        list.iter().any(|entry| {
            let entry = entry.trim().to_lowercase();
            !entry.is_empty() && (entry == host || host.ends_with(&format!(".{entry}")))
        })
    };

    if listed(allowlist) {
        return PhishingVerdict {
            domain: host,
            is_phishing: false,
            confidence: 0.0,
            signals: vec!["domain is on the operator allowlist".to_string()],
            recommendations: Vec::new(),
        };
    }

    if listed(blocklist) {
        return PhishingVerdict {
            domain: host,
            is_phishing: true,
            confidence: 100.0,
            signals: vec!["domain is on the operator blocklist".to_string()],
            recommendations: blocking_recommendations(),
        };
    }

    let (score, signals) = heuristic_score(&host);

    // A single weak signal is not enough to block a domain; require either a
    // strong signal or several weak ones.
    let is_phishing = score >= 60.0;

    PhishingVerdict {
        domain: host,
        is_phishing,
        confidence: score,
        signals,
        recommendations: if is_phishing {
            blocking_recommendations()
        } else {
            Vec::new()
        },
    }
}

fn blocking_recommendations() -> Vec<String> {
    vec![
        "Do not enter your seed phrase on this site".to_string(),
        "Do not connect your wallet".to_string(),
        "Reach the service through its official app or bookmark".to_string(),
    ]
}
