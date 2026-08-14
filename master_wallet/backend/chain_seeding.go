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

// bech32PrefixForChainID returns the Cosmos-SDK bech32 account address prefix
// for a registered non-EVM chain identified by its chain_id (namespace
// >= 9,000,000,000). All 23 Cosmos-SDK chains share ChainType "cosmos" in the
// registry, so the per-chain bech32 prefix MUST be resolved by chain_id
// (not by chain_type, which is generic "cosmos"). Using chain_type alone
// would force the "cosmos" prefix onto Osmosis/Injective/Terra/etc. and
// produce wrong addresses. Falls back to "cosmos" for unmapped chains.
func bech32PrefixForChainID(chainID int64) string {
	switch chainID {
	case 9000000118:
		return "cosmos" // Cosmos Hub
	case 9000026317:
		return "osmo" // Osmosis
	case 9000000330:
		return "terra" // Terra Classic
	case 9000073068:
		return "inj" // Injective
	case 9000014648:
		return "celestia" // Celestia
	case 9000049823:
		return "dydx" // dYdX
	case 9000073741:
		return "sei" // Sei
	case 9000041857:
		return "kujira" // Kujira
	case 9000012099:
		return "stride" // Stride
	case 9000090063:
		return "neutron" // Neutron
	case 9000005267:
		return "juno" // Juno
	case 9000007183:
		return "akash" // Akash
	case 9000018759:
		return "persistence" // Persistence
	case 9000034677:
		return "evmos" // Evmos
	case 9000054841:
		return "canto" // Canto
	case 9000003318:
		return "kava" // Kava
	case 9000062954:
		return "cro" // Cronos (crypto.org chain)
	case 9000016892:
		return "stars" // Stargaze
	case 9000021252:
		return "saga" // Saga
	case 9000086660:
		return "noble" // Noble
	case 9000040572:
		return "axelar" // Axelar
	case 9000007153:
		return "umee" // UMEE
	case 9000000529:
		return "secret" // Secret Network
	default:
		return "cosmos" // unknown Cosmos-SDK chain -> canonical prefix
	}
}

// cosmosChainMeta returns the canonical chain_id string and base fee denom for
// a Cosmos-SDK chain identified by its numeric TigerWallet chain_id. The SignDoc
// MUST carry the correct chain_id string and denom for the signature to be
// valid on the target chain (Osmosis uses "osmosis-1" + "uosmo", Injective uses
// "injective-1" + "inj", etc.). Falls back to cosmoshub-4/uatom.
func cosmosChainMeta(chainID int64) (chainIDStr, denom string) {
	switch chainID {
	case 9000000118:
		return "cosmoshub-4", "uatom" // Cosmos Hub
	case 9000026317:
		return "osmosis-1", "uosmo" // Osmosis
	case 9000000330:
		return "columbus-5", "ulunc" // Terra Classic
	case 9000073068:
		return "injective-1", "inj" // Injective
	case 9000014648:
		return "mocha-4", "utia" // Celestia (test mainnet mocha; mainnet celestia)
	case 9000049823:
		return "dydx-chain-1", "adydx" // dYdX
	case 9000073741:
		return "atlantic-2", "usei" // Sei
	case 9000041857:
		return "kaiyo-1", "ukuji" // Kujira
	case 9000012099:
		return "stride-1", "ustrd" // Stride
	case 9000090063:
		return "pion-1", "untrn" // Neutron
	case 9000005267:
		return "juno-1", "ujuno" // Juno
	case 9000007183:
		return "akashnet-2", "uakt" // Akash
	case 9000018759:
		return "core-1", "uxprt" // Persistence
	case 9000034677:
		return "evmos_9001-2", "aevmos" // Evmos
	case 9000054841:
		return "canto_7700-1", "acanto" // Canto
	case 9000003318:
		return "kava_2222-10", "ukava" // Kava
	case 9000062954:
		return "crypto-org-chain-mainnet-1", "basecro" // Cronos (crypto.org)
	case 9000016892:
		return "stargaze-1", "ustars" // Stargaze
	case 9000021252:
		return "ssc-1", "usaga" // Saga
	case 9000086660:
		return "noble-1", "uusdc" // Noble
	case 9000040572:
		return "axelar-dojo-1", "uaxl" // Axelar
	case 9000007153:
		return "umee-1", "uumee" // UMEE
	case 9000000529:
		return "secret-4", "uscrt" // Secret Network
	default:
		return "cosmoshub-4", "uatom"
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
