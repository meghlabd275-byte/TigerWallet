package main

// chain_seeding.go — Seeds the 120 EVM + 66 non-EVM canonical mainnet chains
// into the user_chains_evm / user_chains_nonevm tables on first boot. The
// master wallet owner can add/remove/update any chain at runtime via the REST
// API. This seeding runs only when the tables are empty (idempotent).

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

// seedDefaultUserChains inserts the 120 EVM + 66 non-EVM canonical mainnet
// chains into the user_chains_evm and user_chains_nonevm tables if they are
// empty. Idempotent — only seeds on first boot (when row count is 0).
func seedDefaultUserChains(ctx context.Context, s *Store) {
	if s == nil || s.DB() == nil {
		return
	}

	// Seed EVM chains (120).
	var evmCount int
	if err := s.DB().QueryRow(ctx, `SELECT COUNT(*) FROM user_chains_evm`).Scan(&evmCount); err != nil {
		log.Printf("WARN: seed EVM chains count: %v", err)
		return
	}
	if evmCount == 0 {
		batch := make([][]any, 0, len(defaultEVMChains))
		for _, c := range defaultEVMChains {
			batch = append(batch, []any{
				c.ChainID, c.Name, c.Symbol, c.RPCURL, c.ExplorerURL,
				c.Decimals, c.DerivationPath,
			})
		}
		inserted, err := s.DB().CopyFrom(ctx,
			pgx.Identifier{"user_chains_evm"},
			[]string{"chain_id", "name", "symbol", "rpc_url", "explorer_url", "decimals", "derivation_path"},
			pgx.CopyFromRows(batch))
		if err != nil {
			log.Printf("WARN: seed EVM chains copy: %v", err)
			return
		}
		log.Printf("Seeded %d EVM chains into user_chains_evm", inserted)
	}

	// Seed non-EVM chains (66).
	var nonevmCount int
	if err := s.DB().QueryRow(ctx, `SELECT COUNT(*) FROM user_chains_nonevm`).Scan(&nonevmCount); err != nil {
		log.Printf("WARN: seed non-EVM chains count: %v", err)
		return
	}
	if nonevmCount == 0 {
		batch := make([][]any, 0, len(defaultNonEVMChains))
		for _, c := range defaultNonEVMChains {
			chainType := c.ChainType
			if chainType == "" {
				chainType = "nonevm"
			}
			prefix := bech32PrefixForChainType(chainType)
			batch = append(batch, []any{
				c.ChainID, c.Name, c.Symbol, chainType, c.RPCURL, c.ExplorerURL,
				c.Decimals, c.DerivationPath, prefix,
			})
		}
		inserted, err := s.DB().CopyFrom(ctx,
			pgx.Identifier{"user_chains_nonevm"},
			[]string{"chain_id", "name", "symbol", "chain_type", "rpc_url", "explorer_url", "decimals", "derivation_path", "address_prefix"},
			pgx.CopyFromRows(batch))
		if err != nil {
			log.Printf("WARN: seed non-EVM chains copy: %v", err)
			return
		}
		log.Printf("Seeded %d non-EVM chains into user_chains_nonevm", inserted)
	}
}

// bech32PrefixForChainType returns the Cosmos-SDK bech32 address prefix for a
// given non-EVM chain type. Returns "" for non-Cosmos chains (Bitcoin/Solana
// use their own address formats — base58check / base58, not bech32).
func bech32PrefixForChainType(chainType string) string {
	switch chainType {
	case "cosmos", "cosmoshub":
		return "cosmos"
	case "osmosis", "osmo":
		return "osmo"
	case "terra", "terra2":
		return "terra"
	case "kava":
		return "kava"
	case "binance", "bnbchain":
		return "bnb"
	case "akash":
		return "akash"
	case "secret":
		return "secret"
	case "celestia":
		return "celestia"
	case "juno":
		return "juno"
	case "stargaze":
		return "stars"
	case "injective":
		return "inj"
	case "persistent", "pstake":
		return "pstake"
	case "umee":
		return "umee"
	case "quicksilver":
		return "quick"
	case "stride":
		return "stride"
	case "noble":
		return "noble"
	case "dydx":
		return "dydx"
	case "neutron":
		return "neutron"
	case "gravitybridge":
		return "gravity"
	case "evmos":
		return "evmos"
	case "regen":
		return "regen"
	case "kichain", "ki":
		return "ki"
	case "sifchain":
		return "sif"
	case "agoric":
		return "agoric"
	case "cheqd":
		return "cheqd"
	case "dig", "digchain":
		return "dig"
	case "desmos":
		return "desmos"
	case "likecoin":
		return "like"
	case "starname", "iov":
		return "star"
	case "medibloc", "panacea":
		return "panacea"
	case "comdex":
		return "comdex"
	case "bandchain", "band":
		return "band"
	case "irisnet", "iris":
		return "iaa"
	case "cryptoorgchain", "crypto-com":
		return "cro"
	case "sentinel":
		return "sent"
	case "persistence":
		return "persistence"
	case "sommelier":
		return "somm"
	default:
		return ""
	}
}
