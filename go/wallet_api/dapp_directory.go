package main

// dapp_directory.go — curated dApp directory. Real, live protocol URLs +
// categories + supported chains. This is the backend equivalent of Trust
// Wallet's / MetaMask's curated dApp listings, exposed via a public REST
// endpoint so every client (web, mobile, desktop, extension) renders the same
// directory instead of hardcoding its own copy.
//
// Only verifiable fields are stored (name, url, category, chains, short
// description). No fabricated metrics (no invented user counts or ratings) —
// clients that want social proof must fetch it from a real analytics source.

// DAppEntry is a single curated dApp in the directory.
type DAppEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Logo        string   `json:"logo"`        // emoji glyph (portable, no asset CDN dependency)
	Chains      []string `json:"chains"`      // e.g. ["ethereum","arbitrum","optimism"]
	Verified    bool     `json:"verified"`    // true = official domain confirmed in curation
}

// dappDirectory is the canonical curated list. Every entry's URL is a real,
// reachable protocol front-end. Categories match the ones the frontend
// already renders (defi, nft, bridge, social, domain, wallet, game).
var dappDirectory = []DAppEntry{
	// ---- DeFi / DEX ----
	{ID: "uniswap", Name: "Uniswap", URL: "https://app.uniswap.org", Category: "defi", Description: "Decentralized trading protocol", Logo: "🦄", Chains: []string{"ethereum", "arbitrum", "optimism", "polygon", "base"}, Verified: true},
	{ID: "curve", Name: "Curve", URL: "https://curve.fi", Category: "defi", Description: "Stablecoin AMM and yield", Logo: "🔵", Chains: []string{"ethereum", "polygon", "arbitrum", "optimism"}, Verified: true},
	{ID: "1inch", Name: "1inch", URL: "https://app.1inch.io", Category: "defi", Description: "DEX aggregator", Logo: "1️⃣", Chains: []string{"ethereum", "bsc", "polygon", "arbitrum", "optimism"}, Verified: true},
	{ID: "sushi", Name: "SushiSwap", URL: "https://www.sushi.com", Category: "defi", Description: "Multi-chain DEX", Logo: "🍣", Chains: []string{"ethereum", "polygon", "arbitrum", "bsc"}, Verified: true},
	{ID: "balancer", Name: "Balancer", URL: "https://app.balancer.fi", Category: "defi", Description: "Programmable liquidity pools", Logo: "⚖️", Chains: []string{"ethereum", "polygon", "arbitrum"}, Verified: true},
	{ID: "jupiter", Name: "Jupiter", URL: "https://jup.ag", Category: "defi", Description: "Solana DEX aggregator", Logo: "🪐", Chains: []string{"solana"}, Verified: true},
	     {ID: "raydium", Name: "Raydium", URL: "https://raydium.io", Category: "defi", Description: "Solana AMM and liquidity", Logo: "ray", Chains: []string{"solana"}, Verified: true},

	// ---- Lending ----
	{ID: "aave", Name: "Aave", URL: "https://app.aave.com", Category: "defi", Description: "Non-custodial liquidity protocol", Logo: "👻", Chains: []string{"ethereum", "polygon", "avalanche", "arbitrum", "optimism"}, Verified: true},
	{ID: "compound", Name: "Compound", URL: "https://app.compound.finance", Category: "defi", Description: "Algorithmic money market", Logo: "📊", Chains: []string{"ethereum"}, Verified: true},

	// ---- NFT ----
	{ID: "opensea", Name: "OpenSea", URL: "https://opensea.io", Category: "nft", Description: "NFT marketplace", Logo: "🌊", Chains: []string{"ethereum", "polygon"}, Verified: true},
	{ID: "magiceden", Name: "Magic Eden", URL: "https://magiceden.io", Category: "nft", Description: "Cross-chain NFT marketplace", Logo: "🧙", Chains: []string{"solana", "ethereum", "polygon"}, Verified: true},

	// ---- Bridge ----
	{ID: "stargate", Name: "Stargate", URL: "https://stargate.finance", Category: "bridge", Description: "Cross-chain bridge", Logo: "🌉", Chains: []string{"ethereum", "avalanche", "polygon", "arbitrum", "optimism", "bsc"}, Verified: true},
	{ID: "across", Name: "Across", URL: "https://across.to", Category: "bridge", Description: "Optimistic cross-chain bridge", Logo: "🌉", Chains: []string{"ethereum", "arbitrum", "optimism"}, Verified: true},

	// ---- Staking ----
	{ID: "lido", Name: "Lido", URL: "https://lido.fi", Category: "staking", Description: "Liquid staking for ETH and staked assets", Logo: "💧", Chains: []string{"ethereum"}, Verified: true},
	{ID: "rocketpool", Name: "Rocket Pool", URL: "https://rocketpool.net", Category: "staking", Description: "Decentralized ETH staking", Logo: "🚀", Chains: []string{"ethereum"}, Verified: true},

	// ---- Domain ----
	{ID: "ens", Name: "ENS", URL: "https://app.ens.domains", Category: "domain", Description: "Ethereum Name Service", Logo: "🔷", Chains: []string{"ethereum"}, Verified: true},
	{ID: "space-id", Name: "SPACE ID", URL: "https://space.id", Category: "domain", Description: "Web3 domain name service", Logo: "🌐", Chains: []string{"bsc", "arbitrum"}, Verified: true},

	// ---- Social ----
	{ID: "lens", Name: "Lens Protocol", URL: "https://lens.xyz", Category: "social", Description: "Decentralized social graph", Logo: "🌿", Chains: []string{"polygon"}, Verified: true},
	{ID: "farcaster", Name: "Farcaster", URL: "https://warpcast.com", Category: "social", Description: "Sufficiently decentralized social network", Logo: "🟣", Chains: []string{"ethereum", "optimism"}, Verified: true},

	// ---- Game ----
	{ID: "axie", Name: "Axie Infinity", URL: "https://app.axieinfinity.com", Category: "game", Description: "NFT-based battle game", Logo: "🐉", Chains: []string{"ethereum", "ronin"}, Verified: true},
}

// listDApps returns the directory, optionally filtered by category and/or chain.
func listDApps(category, chain string) []DAppEntry {
	if category == "" && chain == "" {
		return dappDirectory
	}
	out := make([]DAppEntry, 0, len(dappDirectory))
	for _, d := range dappDirectory {
		if category != "" && d.Category != category {
			continue
		}
		if chain != "" {
			found := false
			for _, c := range d.Chains {
				if c == chain {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

// getDApp returns a single dApp by id.
func getDApp(id string) *DAppEntry {
	for i := range dappDirectory {
		if dappDirectory[i].ID == id {
			return &dappDirectory[i]
		}
	}
	return nil
}

// dAppCategories returns the distinct categories present in the directory
// with their counts, for the directory sidebar.
func dAppCategories() []map[string]interface{} {
	counts := map[string]int{}
	order := []string{}
	for _, d := range dappDirectory {
		if _, ok := counts[d.Category]; !ok {
			order = append(order, d.Category)
		}
		counts[d.Category]++
	}
	out := make([]map[string]interface{}, 0, len(order)+1)
	out = append(out, map[string]interface{}{"id": "all", "name": "All", "count": len(dappDirectory)})
	for _, c := range order {
		out = append(out, map[string]interface{}{"id": c, "name": c, "count": counts[c]})
	}
	return out
}
