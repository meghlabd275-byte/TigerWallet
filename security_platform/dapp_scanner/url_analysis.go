package main

// url_analysis.go — phishing and URL reputation checks.
//
// A URL is judged from its own structure plus the operator's blocklist. No
// verdict is invented: a URL that is not on the blocklist and shows no
// structural signal is reported as unknown rather than safe, and the caller
// decides how to present that.

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLVerdict is the outcome of checking a URL.
type URLVerdict struct {
	URL       string    `json:"url"`
	Host      string    `json:"host"`
	Verdict   string    `json:"verdict"` // blocked, suspicious, unknown
	Reasons   []Finding `json:"reasons"`
	RiskScore int       `json:"risk_score"` // 0 (no signal) .. 100 (blocked)
}

// brandNames are the wallet and exchange names most frequently impersonated.
// A host that contains one of these while not being the brand's own domain is
// a strong impersonation signal.
var brandNames = []string{
	"trustwallet", "metamask", "phantom", "ledger", "trezor", "safepal",
	"okx", "bitget", "coinbase", "binance", "tigerwallet", "rainbow",
	"zerion", "walletconnect", "uniswap", "pancakeswap", "opensea",
}

// officialDomains maps a brand to its registrable domain. A host equal to or
// ending in the official domain is not an impersonation.
var officialDomains = map[string]string{
	"trustwallet":   "trustwallet.com",
	"metamask":      "metamask.io",
	"phantom":       "phantom.app",
	"ledger":        "ledger.com",
	"trezor":        "trezor.io",
	"safepal":       "safepal.com",
	"okx":           "okx.com",
	"bitget":        "bitget.com",
	"coinbase":      "coinbase.com",
	"binance":       "binance.com",
	"tigerwallet":   "tigerwallet.com",
	"rainbow":       "rainbow.me",
	"zerion":        "zerion.io",
	"walletconnect": "walletconnect.com",
	"uniswap":       "uniswap.org",
	"pancakeswap":   "pancakeswap.finance",
	"opensea":       "opensea.io",
}

// freeHostingSuffixes are shared hosting domains where a credential-stealing
// page can be published for free. They are not malicious by themselves.
var freeHostingSuffixes = []string{
	".000webhostapp.com", ".weebly.com", ".wixsite.com", ".blogspot.com",
	".github.io", ".netlify.app", ".vercel.app", ".pages.dev", ".glitch.me",
	".firebaseapp.com", ".web.app", ".repl.co", ".duckdns.org", ".no-ip.org",
}

// riskyTLDs are TLDs with a high abuse rate relative to their legitimate use.
var riskyTLDs = []string{
	".tk", ".ml", ".ga", ".cf", ".gq", ".top", ".click", ".cyou", ".rest",
	".buzz", ".monster", ".quest", ".sbs", ".lol", ".icu",
}

// ipLiteral reports whether host is an IP address rather than a name.
func ipLiteral(host string) bool {
	return net.ParseIP(strings.Trim(host, "[]")) != nil
}

// isPrivateHost reports whether host resolves syntactically to a private,
// loopback or link-local range. Phishing pages are sometimes hosted on an
// internal address to bypass domain-based filtering.
func isPrivateHost(host string) bool {
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// registrableDomain returns the last two labels of a host, which is a
// conservative approximation of the registrable domain. It is only used to
// decide whether a host is the brand's own domain, so an over-approximation
// here can only make the check more permissive for multi-part TLDs, never
// produce a false accusation.
func registrableDomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// checkURL assesses a URL. blockedHosts is the operator's blocklist.
func checkURL(rawURL string, blockedHosts map[string]bool) URLVerdict {
	verdict := URLVerdict{URL: rawURL, Verdict: "unknown"}

	host := extractHost(rawURL)
	if host == "" {
		verdict.Verdict = "suspicious"
		verdict.Reasons = append(verdict.Reasons, Finding{
			ID:          "unparsable_url",
			Severity:    "medium",
			Description: "The URL could not be parsed into a host, so its destination cannot be established.",
		})
		verdict.RiskScore = 50
		return verdict
	}
	verdict.Host = host

	if blockedHosts[host] {
		verdict.Verdict = "blocked"
		verdict.Reasons = append(verdict.Reasons, Finding{
			ID:          "blocklisted_host",
			Severity:    "critical",
			Description: "The host is on the operator's phishing blocklist.",
		})
		verdict.RiskScore = 100
		return verdict
	}

	score := 0
	add := func(f Finding) {
		verdict.Reasons = append(verdict.Reasons, f)
		score += severityWeight(f.Severity)
	}

	// A punycode label means the name contains non-ASCII characters. This is
	// how homograph attacks encode a lookalike of a Latin brand name.
	if strings.Contains(host, "xn--") {
		add(Finding{
			ID:          "punycode_host",
			Severity:    "high",
			Description: "The host contains a punycode label, which encodes non-ASCII characters. This is the standard mechanism for a homograph attack that renders as a lookalike of a well-known name.",
		})
	}

	if ipLiteral(host) {
		sev := "medium"
		desc := "The URL points directly at an IP address instead of a domain name, which is unusual for a legitimate dApp."
		if isPrivateHost(host) {
			sev = "high"
			desc = "The URL points at a private, loopback or link-local address. A page served from an internal address can bypass domain-based filtering and may be probing the local network."
		}
		add(Finding{ID: "ip_literal_host", Severity: sev, Description: desc})
	}

	// Impersonation: the host names a brand but is not that brand's domain.
	registrable := registrableDomain(host)
	for _, brand := range brandNames {
		if !strings.Contains(host, brand) {
			continue
		}
		official, known := officialDomains[brand]
		if known && (host == official || strings.HasSuffix(host, "."+official) || registrable == official) {
			continue
		}
		add(Finding{
			ID:          "brand_impersonation",
			Severity:    "critical",
			Description: fmt.Sprintf("The host contains %q but is not a subdomain of the official domain %s. This is the shape of a lookalike domain used to harvest credentials.", brand, official),
		})
		break
	}

	// Excessive subdomain depth is used to push the real domain out of view in
	// a mobile address bar.
	if strings.Count(host, ".") >= 4 {
		add(Finding{
			ID:          "deep_subdomain",
			Severity:    "medium",
			Description: fmt.Sprintf("The host has %d labels. Deep subdomain chains are used to push the real registrable domain out of view in a truncated address bar.", strings.Count(host, ".")+1),
		})
	}

	for _, tld := range riskyTLDs {
		if strings.HasSuffix(host, tld) {
			add(Finding{
				ID:          "risky_tld",
				Severity:    "medium",
				Description: fmt.Sprintf("The host uses the %s top-level domain, which has a high abuse rate relative to its legitimate use.", tld),
			})
			break
		}
	}

	for _, suffix := range freeHostingSuffixes {
		if strings.HasSuffix(host, suffix) {
			add(Finding{
				ID:          "free_hosting",
				Severity:    "low",
				Description: fmt.Sprintf("The host is hosted on the shared domain %s. Anyone can publish a page there, so the reputation of the domain says nothing about the page.", suffix),
			})
			break
		}
	}

	// Credential-harvesting path keywords.
	lowerURL := strings.ToLower(rawURL)
	for _, keyword := range []string{"seed-phrase", "seedphrase", "mnemonic", "private-key", "privatekey", "recovery-phrase", "connect-wallet", "verify-wallet", "validate-wallet"} {
		if strings.Contains(lowerURL, keyword) {
			add(Finding{
				ID:          "credential_harvest_keyword",
				Severity:    "critical",
				Description: fmt.Sprintf("The URL contains %q. A legitimate wallet never asks for a recovery phrase or private key through a web page.", keyword),
			})
			break
		}
	}

	if score > 100 {
		score = 100
	}
	verdict.RiskScore = score

	// A single critical signal is sufficient to block. Brand impersonation and
	// a credential-harvesting path are not weak evidence that must accumulate
	// before the URL is refused.
	critical := false
	for _, r := range verdict.Reasons {
		if r.Severity == "critical" {
			critical = true
			break
		}
	}

	switch {
	case critical || score >= 70:
		verdict.Verdict = "blocked"
	case score > 0:
		verdict.Verdict = "suspicious"
	default:
		verdict.Verdict = "unknown"
	}
	return verdict
}

// extractHost returns the lower-cased host of a URL without a leading "www.".
func extractHost(rawURL string) string {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ""
	}
	host = strings.TrimPrefix(host, "www.")
	return host
}
