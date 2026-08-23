package store

// catalog.go — the authoritative catalog of products and fetchers that the
// TigerWallet SuperAdmin can license and govern for a White-level client.
//
// This is the single source of truth for "what is available to a WL client".
// Every product must have a self-hosted implementation (wl_* backend) and a
// unique product identifier used in the license heartbeat. Validating against
// this list — rather than accepting arbitrary strings — guarantees that a WL
// client (or a malformed request) can never (a) be licensed for a product that
// doesn't exist, (b) be granted access to a TigerWallet-SuperAdmin-internal
// product, or (c) disable/enable a "fetcher" that the product never serves.

// Product identifiers. These MUST match WL_PRODUCT on the self-hosted backend.
const (
	ProductMasterWallet   = "master_wallet"
	ProductUserWallet     = "user_wallet"
	ProductBots           = "bots"
	ProductProjectParty   = "project_party"
	ProductCard           = "card"
	ProductLiquidity      = "liquidity"
	ProductWLAdminPanel   = "white_label_admin"
)

// ValidProducts is the closed set of self-hosted, White-level-tier products.
// Any request naming a product outside this set is rejected. Note that
// TigerWallet SuperAdmin surfaces (admin, super_admin, license_service,
// kill_switch, permission_service, gateway, security, etc.) are intentionally
// absent: a WL client can never be licensed for, nor self-host, those.
var ValidProducts = map[string]bool{
	ProductMasterWallet:  true,
	ProductUserWallet:    true,
	ProductBots:          true,
	ProductProjectParty:  true,
	ProductCard:          true,
	ProductLiquidity:     true,
	ProductWLAdminPanel:  true,
}

// FetchersByProduct enumerates the per-product functional categories that the
// SuperAdmin can independently enable/disable. These align with the path
// segment used by wlgate.CategoryFetcher / adminFetcher on each self-hosted
// backend, so the SuperAdmin UI shows exactly the toggles the product honors.
var FetchersByProduct = map[string][]string{
	ProductMasterWallet: {
		"master-wallet", "balance", "sign", "transactions", "revenue-payout",
		"withdrawal-request", "sub-wallets", "policies", "fees", "auto-sign",
		"users", "analytics", "notifications", "webhooks", "audit", "multisig",
		"transfer", "sweep", "user-chains", "user-tokens", "feature-flags",
	},
	ProductUserWallet: {
		"wallets", "send", "sign", "transactions", "balance", "tokens", "nfts",
		"gas", "price", "chains", "swap", "staking", "non_evm",
		"address-book", "devices", "keystore", "admin",
	},
	ProductBots: {
		"bots", "subscriptions", "fees", "api-keys", "users", "cex", "dex",
		"fee-addresses", "stats",
	},
	ProductProjectParty: {
		"coins", "tokens", "listings", "launchpad", "liquidity", "market-making",
		"orders", "pricing", "fees", "kyc", "analytics", "audit", "compliance",
		"holders", "trending", "favorites", "featured", "search", "history",
		"market", "volume", "transactions", "users", "status", "pay", "calculate",
	},
	ProductCard: {
		"cards", "best", "quote", "depth", "routes", "sources", "pools", "auth",
	},
	ProductLiquidity: {
		"pools", "best", "quote", "depth", "routes", "sources", "auth",
	},
	ProductWLAdminPanel: {
		"admins", "admin-roles", "admin-permissions", "users", "kyc", "transactions",
		"withdrawals", "tokens", "pairs", "blockchains", "fees", "notifications",
		"audit-logs", "feature-flags", "ip-whitelist", "futures", "options",
		"copy-trading", "convert", "onramp", "offramp", "p2p-clients", "partners",
		"rewards", "marketing", "tickets", "sessions", "stats", "wl-bots",
		"wl-cards", "wl-liquidity", "auth",
	},
}

// IsValidProduct reports whether the string is a licenseable WL product.
func IsValidProduct(product string) bool {
	return ValidProducts[product]
}

// IsValidFetcher reports whether fetcher is a known toggle for the product.
// "*" is always valid (whole-product toggle).
func IsValidFetcher(product, fetcher string) bool {
	if fetcher == "*" {
		return true
	}
	for _, f := range FetchersByProduct[product] {
		if f == fetcher {
			return true
		}
	}
	return false
}

// NormalizeProducts coerces a client-specified product list into a canonical
// allowed_products list: it de-duplicates, drops unknown products, and expands
// the implicit default when the caller supplied nothing at all.
func NormalizeProducts(products []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range products {
		if p == "" || seen[p] {
			continue
		}
		if IsValidProduct(p) {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// DefaultProductList returns the canonical default for a freshly-onboarded
// White-level client (the four core self-hosted products).
func DefaultProductList() []string {
	return []string{ProductMasterWallet, ProductUserWallet, ProductBots, ProductProjectParty}
}

// IsProductAllowed reports whether the client's allowed_products entitle it to
// run the given product. A product is allowed when it is explicitly listed or
// when the client holds the 'all' license wildcard for that product. Any
// product outside ValidProducts is never allowed.
func IsProductAllowed(allowed []string, product string) bool {
	if !IsValidProduct(product) {
		return false
	}
	for _, p := range allowed {
		if p == product || p == "all" {
			return true
		}
	}
	return false
}