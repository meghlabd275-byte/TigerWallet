// Package roles defines the 13 scoped sub-admin roles a WL client can assign
// to admins in their WL admin panel, plus the role→endpoint-scope mapping used
// by the RequireScope middleware for per-endpoint authorization.
//
// The 6 product scopes:
//   trading_admin     — WL futures, margin, options, copy, convert trading
//   p2p_admin         — WL p2p, on-ramp, off-ramp, p2p merchant+client
//   bot_admin         — all WL bots
//   listing_admin     — WL coin/token listing + trading-pair + listingManager/partner mgmt
//   liquidity_admin   — all WL liquidity sources
//   wallet_admin      — WL MasterWallet + WL-UserWallet management
//
// The 7 other-services scopes:
//   customer_service_admin — customer service / support tickets
//   marketing_admin        — marketing & promotion
//   kyc_admin              — KYC review
//   card_admin             — WL-Branded CryptoCard
//   reward_admin           — reward system
//   security_admin         — security (WL client only)
//   compliance_admin       — compliance / audit / reports
//
// Plus the WL client owner role:
//   wl_client — the WL client themselves; can do everything in their tenancy
//               EXCEPT withdraw funds/revenue (that needs SuperAdmin co-sign).
package roles

// Scoped roles assignable by a WL client.
const (
	WLClient              = "wl_client"
	TradingAdmin          = "trading_admin"
	P2PAdmin              = "p2p_admin"
	BotAdmin              = "bot_admin"
	ListingAdmin          = "listing_admin"
	LiquidityAdmin        = "liquidity_admin"
	WalletAdmin           = "wallet_admin"
	CustomerServiceAdmin  = "customer_service_admin"
	MarketingAdmin        = "marketing_admin"
	KYCAdmin              = "kyc_admin"
	CardAdmin             = "card_admin"
	RewardAdmin           = "reward_admin"
	SecurityAdmin         = "security_admin"
	ComplianceAdmin       = "compliance_admin"
)

// ValidScopes is the whitelist of assignable scopes. A WL client cannot invent
// arbitrary scope strings.
var ValidScopes = map[string]bool{
	WLClient: true,
	TradingAdmin: true,
	P2PAdmin: true,
	BotAdmin: true,
	ListingAdmin: true,
	LiquidityAdmin: true,
	WalletAdmin: true,
	CustomerServiceAdmin: true,
	MarketingAdmin: true,
	KYCAdmin: true,
	CardAdmin: true,
	RewardAdmin: true,
	SecurityAdmin: true,
	ComplianceAdmin: true,
}

// AllScopes returns the list of assignable scope names (for the frontend role picker).
func AllScopes() []string {
	return []string{
		WLClient, TradingAdmin, P2PAdmin, BotAdmin, ListingAdmin, LiquidityAdmin, WalletAdmin,
		CustomerServiceAdmin, MarketingAdmin, KYCAdmin, CardAdmin, RewardAdmin, SecurityAdmin, ComplianceAdmin,
	}
}

// IsValid checks a scope is in the whitelist.
func IsValid(scope string) bool { return ValidScopes[scope] }

// ScopeGroups maps each scope to a human-readable group label (for the UI).
var ScopeGroups = map[string]string{
	TradingAdmin:         "Trading (futures, margin, options, copy, convert)",
	P2PAdmin:             "P2P & Fiat (p2p, on-ramp, off-ramp, merchant)",
	BotAdmin:             "Bots (all WL bots)",
	ListingAdmin:         "Listing (coin/token, trading pairs, listingManager)",
	LiquidityAdmin:       "Liquidity (all liquidity sources)",
	WalletAdmin:          "Wallets (MasterWallet + UserWallet)",
	CustomerServiceAdmin: "Customer Service (tickets, support)",
	MarketingAdmin:       "Marketing & Promotion",
	KYCAdmin:             "KYC (review)",
	CardAdmin:            "WL-Branded CryptoCard",
	RewardAdmin:          "Reward System",
	SecurityAdmin:        "Security (WL client only)",
	ComplianceAdmin:      "Compliance (audit, reports, SLA)",
	WLClient:             "WL Client Owner (full tenancy control)",
}
