#include "wallet/wallet_manager.hpp"
#include "crypto/entropy.hpp"
#include "crypto/mnemonic.hpp"
#include "crypto/derivation.hpp"
#include <sodium/crypto_generichash.h>
#include <sodium/crypto_pwhash.h>
#include <sodium/randombytes.h>
#include <stdexcept>
#include <iostream>

namespace tigerwallet {

// Implementation of wallet manager
class WalletManager::Impl {
public:
    std::map<std::string, ChainConfig> chains_;
    std::map<std::string, TokenConfig> tokens_;
    std::map<std::string, Wallet> wallets_;
    Entropy entropy_;
    
    Impl() {
        initialize_default_chains();
        initialize_default_tokens();
    }
    
    void initialize_default_chains() {
        // EVM Chains
        chains_["ethereum"] = ChainConfig{
            .id = "ethereum",
            .name = "Ethereum",
            .symbol = "ETH",
            .chain_id = 1,
            .type = BlockchainType::EVM,
            .rpc_url = "https://eth.llamarpc.com",
            .explorer_url = "https://etherscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/279/small/ethereum.png",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 12000,
            .max_gas_price = 500000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["bsc"] = ChainConfig{
            .id = "bsc",
            .name = "BNB Smart Chain",
            .symbol = "BNB",
            .chain_id = 56,
            .type = BlockchainType::EVM,
            .rpc_url = "https://bsc-dataseed.binance.org",
            .explorer_url = "https://bscscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png",
            .decimals = 18,
            .gas_token = "BNB",
            .avg_block_time_ms = 3000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["polygon"] = ChainConfig{
            .id = "polygon",
            .name = "Polygon",
            .symbol = "MATIC",
            .chain_id = 137,
            .type = BlockchainType::EVM,
            .rpc_url = "https://polygon-rpc.com",
            .explorer_url = "https://polygonscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/4713/small/matic-token-icon.png",
            .decimals = 18,
            .gas_token = "MATIC",
            .avg_block_time_ms = 2000,
            .max_gas_price = 50000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["arbitrum"] = ChainConfig{
            .id = "arbitrum",
            .name = "Arbitrum One",
            .symbol = "ETH",
            .chain_id = 42161,
            .type = BlockchainType::EVM,
            .rpc_url = "https://arb1.arbitrum.io/rpc",
            .explorer_url = "https://arbiscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/16547/small/photo_2023-03-29_21.47.00.jpeg",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 250,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["optimism"] = ChainConfig{
            .id = "optimism",
            .name = "Optimism",
            .symbol = "ETH",
            .chain_id = 10,
            .type = BlockchainType::EVM,
            .rpc_url = "https://mainnet.optimism.io",
            .explorer_url = "https://optimistic.etherscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/25244/small/Optimism.png",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 2000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["base"] = ChainConfig{
            .id = "base",
            .name = "Base",
            .symbol = "ETH",
            .chain_id = 8453,
            .type = BlockchainType::EVM,
            .rpc_url = "https://mainnet.base.org",
            .explorer_url = "https://basescan.org",
            .logo_url = "https://assets.coingecko.com/coins/images/31054/small/base-dai.jpeg",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 2000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["avalanche"] = ChainConfig{
            .id = "avalanche",
            .name = "Avalanche C-Chain",
            .symbol = "AVAX",
            .chain_id = 43114,
            .type = BlockchainType::EVM,
            .rpc_url = "https://api.avax.network/ext/bc/C/rpc",
            .explorer_url = "https://snowtrace.io",
            .logo_url = "https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png",
            .decimals = 18,
            .gas_token = "AVAX",
            .avg_block_time_ms = 1000,
            .max_gas_price = 50000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["solana"] = ChainConfig{
            .id = "solana",
            .name = "Solana",
            .symbol = "SOL",
            .chain_id = -1,
            .type = BlockchainType::SOLANA,
            .rpc_url = "https://api.mainnet-beta.solana.com",
            .explorer_url = "https://solscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/4128/small/solana.png",
            .decimals = 9,
            .gas_token = "SOL",
            .avg_block_time_ms = 400,
            .max_gas_price = 100000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["tron"] = ChainConfig{
            .id = "tron",
            .name = "TRON",
            .symbol = "TRX",
            .chain_id = -1,
            .type = BlockchainType::TRON,
            .rpc_url = "https://api.trongrid.io",
            .explorer_url = "https://tronscan.org",
            .logo_url = "https://assets.coingecko.com/coins/images/1094/small/tron-logo.png",
            .decimals = 6,
            .gas_token = "TRX",
            .avg_block_time_ms = 3000,
            .max_gas_price = 10000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["aptos"] = ChainConfig{
            .id = "aptos",
            .name = "Aptos",
            .symbol = "APT",
            .chain_id = -1,
            .type = BlockchainType::APTOS,
            .rpc_url = "https://fullnode.mainnet.aptoslabs.com",
            .explorer_url = "https://aptoscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/26455/small/aptos_round.png",
            .decimals = 8,
            .gas_token = "APT",
            .avg_block_time_ms = 1000,
            .max_gas_price = 100000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["cosmos"] = ChainConfig{
            .id = "cosmos",
            .name = "Cosmos Hub",
            .symbol = "ATOM",
            .chain_id = -1,
            .type = BlockchainType::COSMOS,
            .rpc_url = "https://rpc.cosmos.network",
            .explorer_url = "https://ping.pub/cosmos",
            .logo_url = "https://assets.coingecko.com/coins/images/1481/small/cosmos_hub.png",
            .decimals = 6,
            .gas_token = "ATOM",
            .avg_block_time_ms = 7000,
            .max_gas_price = 1000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["ton"] = ChainConfig{
            .id = "ton",
            .name = "TON",
            .symbol = "TON",
            .chain_id = -1,
            .type = BlockchainType::EVM,
            .rpc_url = "https://toncenter.com/api/v2",
            .explorer_url = "https://tonscan.org",
            .logo_url = "https://assets.coingecko.com/coins/images/17980/small/ton_symbol.png",
            .decimals = 9,
            .gas_token = "TON",
            .avg_block_time_ms = 5000,
            .max_gas_price = 1000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["sui"] = ChainConfig{
            .id = "sui",
            .name = "Sui",
            .symbol = "SUI",
            .chain_id = -1,
            .type = BlockchainType::SUI,
            .rpc_url = "https://fullnode.mainnet.sui.io",
            .explorer_url = "https://suiscan.xyz",
            .logo_url = "https://assets.coingecko.com/coins/images/26375/small/sui_asset.jpeg",
            .decimals = 9,
            .gas_token = "SUI",
            .avg_block_time_ms = 1000,
            .max_gas_price = 100000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["near"] = ChainConfig{
            .id = "near",
            .name = "NEAR Protocol",
            .symbol = "NEAR",
            .chain_id = -1,
            .type = BlockchainType::NEAR,
            .rpc_url = "https://rpc.mainnet.near.org",
            .explorer_url = "https://explorer.near.org",
            .logo_url = "https://assets.coingecko.com/coins/images/10365/small/near.jpg",
            .decimals = 24,
            .gas_token = "NEAR",
            .avg_block_time_ms = 1000,
            .max_gas_price = 1000000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["linea"] = ChainConfig{
            .id = "linea",
            .name = "Linea",
            .symbol = "ETH",
            .chain_id = 59144,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.linea.build",
            .explorer_url = "https://lineascan.build",
            .logo_url = "https://assets.coingecko.com/coins/images/28689/small/linea.png",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 2000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["scroll"] = ChainConfig{
            .id = "scroll",
            .name = "Scroll",
            .symbol = "ETH",
            .chain_id = 534352,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.scroll.io",
            .explorer_url = "https://scrollscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/29577/small/scroll.png",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 3000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["zksync"] = ChainConfig{
            .id = "zksync",
            .name = "zkSync Era",
            .symbol = "ETH",
            .chain_id = 324,
            .type = BlockchainType::EVM,
            .rpc_url = "https://mainnet.era.zksync.io",
            .explorer_url = "https://explorer.zksync.io",
            .logo_url = "https://assets.coingecko.com/coins/images/48689/small/sync.png",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 1000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["blast"] = ChainConfig{
            .id = "blast",
            .name = "Blast",
            .symbol = "ETH",
            .chain_id = 81457,
            .type = BlockchainType::EVM,
            .rpc_url = "https://blastl2-mainnet-public.united.blast.io",
            .explorer_url = "https://blastscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/35537/small/blast.png",
            .decimals = 18,
            .gas_token = "ETH",
            .avg_block_time_ms = 2000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["mantle"] = ChainConfig{
            .id = "mantle",
            .name = "Mantle",
            .symbol = "MNT",
            .chain_id = 5000,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.mantle.xyz",
            .explorer_url = "https://mantlescan.info",
            .logo_url = "https://assets.coingecko.com/coins/images/29631/small/mantle.png",
            .decimals = 18,
            .gas_token = "MNT",
            .avg_block_time_ms = 2000,
            .max_gas_price = 10000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["bitcoin"] = ChainConfig{
            .id = "bitcoin",
            .name = "Bitcoin",
            .symbol = "BTC",
            .chain_id = -1,
            .type = BlockchainType::BITCOIN,
            .rpc_url = "https://btc-rpc.allthatblock.com",
            .explorer_url = "https://blockstream.info",
            .logo_url = "https://assets.coingecko.com/coins/images/1/small/bitcoin.png",
            .decimals = 8,
            .gas_token = "BTC",
            .avg_block_time_ms = 600000,
            .max_gas_price = 100000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["dogecoin"] = ChainConfig{
            .id = "dogecoin",
            .name = "Dogecoin",
            .symbol = "DOGE",
            .chain_id = -1,
            .type = BlockchainType::BITCOIN,
            .rpc_url = "https://dogecoin-rpc.allthatblock.com",
            .explorer_url = "https://doge.town",
            .logo_url = "https://assets.coingecko.com/coins/images/5/small/dogecoin.png",
            .decimals = 8,
            .gas_token = "DOGE",
            .avg_block_time_ms = 60000,
            .max_gas_price = 10000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["litecoin"] = ChainConfig{
            .id = "litecoin",
            .name = "Litecoin",
            .symbol = "LTC",
            .chain_id = -1,
            .type = BlockchainType::BITCOIN,
            .rpc_url = "https://litecoin-rpc.allthatblock.com",
            .explorer_url = "https://blockchair.com/litecoin",
            .logo_url = "https://assets.coingecko.com/coins/images/2/small/litecoin.png",
            .decimals = 8,
            .gas_token = "LTC",
            .avg_block_time_ms = 150000,
            .max_gas_price = 100000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["fantom"] = ChainConfig{
            .id = "fantom",
            .name = "Fantom",
            .symbol = "FTM",
            .chain_id = 250,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.fantom.network",
            .explorer_url = "https://ftmscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/4001/small/Fantom_round.png",
            .decimals = 18,
            .gas_token = "FTM",
            .avg_block_time_ms = 2000,
            .max_gas_price = 50000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["celo"] = ChainConfig{
            .id = "celo",
            .name = "Celo",
            .symbol = "CELO",
            .chain_id = 42220,
            .type = BlockchainType::EVM,
            .rpc_url = "https://forno.celo.org",
            .explorer_url = "https://celoscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/5568/small/celo.png",
            .decimals = 18,
            .gas_token = "CELO",
            .avg_block_time_ms = 5000,
            .max_gas_price = 5000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["klaytn"] = ChainConfig{
            .id = "klaytn",
            .name = "Klaytn",
            .symbol = "KLAY",
            .chain_id = 8217,
            .type = BlockchainType::EVM,
            .rpc_url = "https://public-en-cypress.klaytn.net",
            .explorer_url = "https://scope.klaytn.com",
            .logo_url = "https://assets.coingecko.com/coins/images/9672/small/klaytn.png",
            .decimals = 18,
            .gas_token = "KLAY",
            .avg_block_time_ms = 1000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["cronos"] = ChainConfig{
            .id = "cronos",
            .name = "Cronos",
            .symbol = "CRO",
            .chain_id = 25,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.cronos.org",
            .explorer_url = "https://cronoscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/7310/small/cro.png",
            .decimals = 18,
            .gas_token = "CRO",
            .avg_block_time_ms = 6000,
            .max_gas_price = 10000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["moonbeam"] = ChainConfig{
            .id = "moonbeam",
            .name = "Moonbeam",
            .symbol = "GLMR",
            .chain_id = 1284,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.api.moonbeam.network",
            .explorer_url = "https://moonbeam.moonscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/17759/small/Moonbeam_Network_Icon.png",
            .decimals = 18,
            .gas_token = "GLMR",
            .avg_block_time_ms = 12000,
            .max_gas_price = 10000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["astar"] = ChainConfig{
            .id = "astar",
            .name = "Astar",
            .symbol = "ASTR",
            .chain_id = 592,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.astar.network",
            .explorer_url = "https://astar.explorer.mainnet.solarflare.io",
            .logo_url = "https://assets.coingecko.com/coins/images/22617/small/astr.png",
            .decimals = 18,
            .gas_token = "ASTR",
            .avg_block_time_ms = 12000,
            .max_gas_price = 10000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["polkadot"] = ChainConfig{
            .id = "polkadot",
            .name = "Polkadot",
            .symbol = "DOT",
            .chain_id = -1,
            .type = BlockchainType::COSMOS,
            .rpc_url = "https://rpc.polkadot.io",
            .explorer_url = "https://polkadot.subscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/12171/small/polkadot.png",
            .decimals = 10,
            .gas_token = "DOT",
            .avg_block_time_ms = 6000,
            .max_gas_price = 1000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["algorand"] = ChainConfig{
            .id = "algorand",
            .name = "Algorand",
            .symbol = "ALGO",
            .chain_id = -1,
            .type = BlockchainType::ALGORAND,
            .rpc_url = "https://mainnet-api.algorand.network",
            .explorer_url = "https://algoexplorer.io",
            .logo_url = "https://assets.coingecko.com/coins/images/4380/small/download.png",
            .decimals = 6,
            .gas_token = "ALGO",
            .avg_block_time_ms = 4000,
            .max_gas_price = 1000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["tezos"] = ChainConfig{
            .id = "tezos",
            .name = "Tezos",
            .symbol = "XTZ",
            .chain_id = -1,
            .type = BlockchainType::TEZOS,
            .rpc_url = "https://mainnet.api.tez.ie",
            .explorer_url = "https://tzstats.com",
            .logo_url = "https://assets.coingecko.com/coins/images/976/small/Tezos-logo.png",
            .decimals = 6,
            .gas_token = "XTZ",
            .avg_block_time_ms = 30000,
            .max_gas_price = 1000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["cardano"] = ChainConfig{
            .id = "cardano",
            .name = "Cardano",
            .symbol = "ADA",
            .chain_id = -1,
            .type = BlockchainType::CARDANO,
            .rpc_url = "https://cardano-mainnet.blockfrost.io/api/v0",
            .explorer_url = "https://cardanoscan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/975/small/cardano.png",
            .decimals = 6,
            .gas_token = "ADA",
            .avg_block_time_ms = 20000,
            .max_gas_price = 1000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["ripple"] = ChainConfig{
            .id = "ripple",
            .name = "XRP Ledger",
            .symbol = "XRP",
            .chain_id = -1,
            .type = BlockchainType::XRPL,
            .rpc_url = "https://xrplcluster.com",
            .explorer_url = "https://xrpscan.com",
            .logo_url = "https://assets.coingecko.com/coins/images/44/small/xrp-symbol-white-128.png",
            .decimals = 6,
            .gas_token = "XRP",
            .avg_block_time_ms = 4000,
            .max_gas_price = 1000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["injective"] = ChainConfig{
            .id = "injective",
            .name = "Injective",
            .symbol = "INJ",
            .chain_id = -1,
            .type = BlockchainType::COSMOS,
            .rpc_url = "https://public.api.injective.network",
            .explorer_url = "https://explorer.injective.network",
            .logo_url = "https://assets.coingecko.com/coins/images/12882/small/Secondary_Symbol.png",
            .decimals = 18,
            .gas_token = "INJ",
            .avg_block_time_ms = 1000,
            .max_gas_price = 500000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["sei"] = ChainConfig{
            .id = "sei",
            .name = "Sei",
            .symbol = "SEI",
            .chain_id = -1,
            .type = BlockchainType::COSMOS,
            .rpc_url = "https://rest.sei-apis.com",
            .explorer_url = "https://seistream.app",
            .logo_url = "https://assets.coingecko.com/coins/images/28205/small/Sei_Logo_-_Transparent.png",
            .decimals = 18,
            .gas_token = "SEI",
            .avg_block_time_ms = 1000,
            .max_gas_price = 1000000000ULL,
            .supports_eip1559 = false
        };
        
        chains_["immutable"] = ChainConfig{
            .id = "immutable",
            .name = "Immutable",
            .symbol = "IMX",
            .chain_id = -1,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.immutable.com",
            .explorer_url = "https://immutascan.io",
            .logo_url = "https://assets.coingecko.com/coins/images/17233/small/immutableX-symbol-BLK-RGB.png",
            .decimals = 18,
            .gas_token = "IMX",
            .avg_block_time_ms = 1000,
            .max_gas_price = 100000000000ULL,
            .supports_eip1559 = true
        };
        
        chains_["pulsechain"] = ChainConfig{
            .id = "pulsechain",
            .name = "PulseChain",
            .symbol = "PLS",
            .chain_id = 369,
            .type = BlockchainType::EVM,
            .rpc_url = "https://rpc.pulsechain.com",
            .explorer_url = "https://scan.pulsechain.com",
            .logo_url = "https://assets.coingecko.com/coins/images/28195/small/pulsechain.png",
            .decimals = 18,
            .gas_token = "PLS",
            .avg_block_time_ms = 12000,
            .max_gas_price = 500000000000ULL,
            .supports_eip1559 = true
        };
    }
    
    void initialize_default_tokens() {
        // Native tokens
        tokens_["eth"] = TokenConfig{
            .id = "eth",
            .blockchain_id = "ethereum",
            .symbol = "ETH",
            .name = "Ethereum",
            .decimals = 18,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "120000000",
            .logo_url = "https://assets.coingecko.com/coins/images/279/small/ethereum.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 3450.0,
            .market_cap = 414000000000ULL,
            .volume_24h = 18000000000ULL
        };
        
        tokens_["btc"] = TokenConfig{
            .id = "btc",
            .blockchain_id = "bitcoin",
            .symbol = "BTC",
            .name = "Bitcoin",
            .decimals = 8,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "21000000",
            .logo_url = "https://assets.coingecko.com/coins/images/1/small/bitcoin.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 67500.0,
            .market_cap = 1320000000000ULL,
            .volume_24h = 35000000000ULL
        };
        
        tokens_["usdt"] = TokenConfig{
            .id = "usdt-eth",
            .blockchain_id = "ethereum",
            .symbol = "USDT",
            .name = "Tether USD",
            .decimals = 6,
            .contract_address = "0xdAC17F958D2ee523a2206206994597C13D831ec7",
            .type = "erc20",
            .total_supply = "140000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/325/small/Tether.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 1.0,
            .market_cap = 140000000000ULL,
            .volume_24h = 65000000000ULL
        };
        
        tokens_["usdc"] = TokenConfig{
            .id = "usdc-eth",
            .blockchain_id = "ethereum",
            .symbol = "USDC",
            .name = "USD Coin",
            .decimals = 6,
            .contract_address = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
            .type = "erc20",
            .total_supply = "45000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/6319/small/USD_Coin_icon.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 1.0,
            .market_cap = 45000000000ULL,
            .volume_24h = 6000000000ULL
        };
        
        tokens_["bnb"] = TokenConfig{
            .id = "bnb-bsc",
            .blockchain_id = "bsc",
            .symbol = "BNB",
            .name = "BNB",
            .decimals = 18,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "200000000",
            .logo_url = "https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 580.0,
            .market_cap = 87000000000ULL,
            .volume_24h = 1800000000ULL
        };
        
        tokens_["sol"] = TokenConfig{
            .id = "sol",
            .blockchain_id = "solana",
            .symbol = "SOL",
            .name = "Solana",
            .decimals = 9,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "590000000",
            .logo_url = "https://assets.coingecko.com/coins/images/4128/small/solana.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 145.0,
            .market_cap = 68000000000ULL,
            .volume_24h = 3500000000ULL
        };
        
        tokens_["trx"] = TokenConfig{
            .id = "trx",
            .blockchain_id = "tron",
            .symbol = "TRX",
            .name = "TRON",
            .decimals = 6,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "100000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/1094/small/tron-logo.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 0.12,
            .market_cap = 10500000000ULL,
            .volume_24h = 800000000ULL
        };
        
        tokens_["doge"] = TokenConfig{
            .id = "doge",
            .blockchain_id = "dogecoin",
            .symbol = "DOGE",
            .name = "Dogecoin",
            .decimals = 8,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "140000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/5/small/dogecoin.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 0.12,
            .market_cap = 17000000000ULL,
            .volume_24h = 1500000000ULL
        };
        
        tokens_["ada"] = TokenConfig{
            .id = "ada",
            .blockchain_id = "cardano",
            .symbol = "ADA",
            .name = "Cardano",
            .decimals = 6,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "45000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/975/small/cardano.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 0.45,
            .market_cap = 16000000000ULL,
            .volume_24h = 500000000ULL
        };
        
        tokens_["xrp"] = TokenConfig{
            .id = "xrp",
            .blockchain_id = "ripple",
            .symbol = "XRP",
            .name = "XRP",
            .decimals = 6,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "100000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/44/small/xrp-symbol-white-128.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 0.62,
            .market_cap = 34000000000ULL,
            .volume_24h = 2500000000ULL
        };
        
        tokens_["dot"] = TokenConfig{
            .id = "dot",
            .blockchain_id = "polkadot",
            .symbol = "DOT",
            .name = "Polkadot",
            .decimals = 10,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "1400000000",
            .logo_url = "https://assets.coingecko.com/coins/images/12171/small/polkadot.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 7.5,
            .market_cap = 10500000000ULL,
            .volume_24h = 350000000ULL
        };
        
        tokens_["avax"] = TokenConfig{
            .id = "avax",
            .blockchain_id = "avalanche",
            .symbol = "AVAX",
            .name = "Avalanche",
            .decimals = 18,
            .contract_address = std::nullopt,
            .type = "native",
            .total_supply = "740000000",
            .logo_url = "https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 35.0,
            .market_cap = 14000000000ULL,
            .volume_24h = 600000000ULL
        };
        
        tokens_["link"] = TokenConfig{
            .id = "link-eth",
            .blockchain_id = "ethereum",
            .symbol = "LINK",
            .name = "Chainlink",
            .decimals = 18,
            .contract_address = "0x514910771AF9Ca656af840dff83E8264EcF986CA",
            .type = "erc20",
            .total_supply = "1000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/877/small/chainlink-new-logo.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 14.5,
            .market_cap = 8500000000ULL,
            .volume_24h = 500000000ULL
        };
        
        tokens_["uni"] = TokenConfig{
            .id = "uni-eth",
            .blockchain_id = "ethereum",
            .symbol = "UNI",
            .name = "Uniswap",
            .decimals = 18,
            .contract_address = "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984",
            .type = "erc20",
            .total_supply = "1000000000",
            .logo_url = "https://assets.coingecko.com/coins/images/12504/small/uniswap-uni.png",
            .is_active = true,
            .is_popular = true,
            .price_usd = 9.8,
            .market_cap = 7500000000ULL,
            .volume_24h = 300000000ULL
        };
    }
};

WalletManager::WalletManager() : pImpl(std::make_unique<Impl>()) {}
WalletManager::~WalletManager() = default;

void WalletManager::initialize(const std::vector<ChainConfig>& chains) {
    for (const auto& chain : chains) {
        pImpl->chains_[chain.id] = chain;
    }
}

std::vector<ChainConfig> WalletManager::get_supported_chains() const {
    std::vector<ChainConfig> result;
    for (const auto& [id, chain] : pImpl->chains_) {
        result.push_back(chain);
    }
    return result;
}

bool WalletManager::add_chain(const ChainConfig& config) {
    pImpl->chains_[config.id] = config;
    return true;
}

bool WalletManager::update_chain(const std::string& chain_id, const ChainConfig& config) {
    if (pImpl->chains_.find(chain_id) == pImpl->chains_.end()) {
        return false;
    }
    pImpl->chains_[chain_id] = config;
    return true;
}

bool WalletManager::delete_chain(const std::string& chain_id) {
    return pImpl->chains_.erase(chain_id) > 0;
}

bool WalletManager::add_token(const TokenConfig& config) {
    pImpl->tokens_[config.id] = config;
    return true;
}

bool WalletManager::update_token(const std::string& token_id, const TokenConfig& config) {
    if (pImpl->tokens_.find(token_id) == pImpl->tokens_.end()) {
        return false;
    }
    pImpl->tokens_[token_id] = config;
    return true;
}

bool WalletManager::delete_token(const std::string& token_id) {
    return pImpl->tokens_.erase(token_id) > 0;
}

std::optional<Wallet> WalletManager::create_wallet(
    const std::string& user_id,
    WalletType type,
    const std::string& blockchain_id,
    const std::string& password
) {
    // Generate entropy and mnemonic
    std::vector<uint8_t> entropy_bytes = pImpl->entropy_.generate(256);
    std::string mnemonic = Mnemonic::entropy_to_mnemonic(entropy_bytes);
    
    // Derive key from password
    std::vector<uint8_t> seed = Mnemonic::mnemonic_to_seed(mnemonic, password);
    std::vector<uint8_t> private_key = Derivation::derive_key(seed, 0, 0);
    
    // Generate wallet address based on blockchain type
    auto chain_it = pImpl->chains_.find(blockchain_id);
    if (chain_it == pImpl->chains_.end()) {
        return std::nullopt;
    }
    
    std::string address;
    std::vector<uint8_t> public_key;
    
    switch (chain_it->second.type) {
        case BlockchainType::EVM:
            public_key = Derivation::derive_evm_key(private_key);
            address = "0x" + Crypto::pubkey_to_address(public_key).substr(2);
            break;
        case BlockchainType::SOLANA:
            public_key = Derivation::derive_solana_key(private_key);
            address = Crypto::base58_encode(public_key);
            break;
        case BlockchainType::TRON:
            public_key = Derivation::derive_evm_key(private_key);
            address = "T" + Crypto::base58_encode(public_key).substr(1, 33);
            break;
        default:
            public_key = Derivation::derive_evm_key(private_key);
            address = "0x" + Crypto::pubkey_to_address(public_key).substr(2);
    }
    
    // Create wallet object
    Wallet wallet;
    wallet.id = "wallet_" + std::to_string(pImpl->wallets_.size() + 1);
    wallet.user_id = user_id;
    wallet.type = type;
    wallet.address = address;
    wallet.blockchain_id = blockchain_id;
    wallet.public_key = public_key;
    wallet.derivation_path = "m/44'/60'/0'/0/0";
    wallet.created_at = std::to_string(std::time(nullptr));
    wallet.updated_at = wallet.created_at;
    wallet.is_active = true;
    
    pImpl->wallets_[wallet.id] = wallet;
    
    return wallet;
}

std::optional<Wallet> WalletManager::get_wallet(const std::string& wallet_id) {
    auto it = pImpl->wallets_.find(wallet_id);
    if (it == pImpl->wallets_.end()) {
        return std::nullopt;
    }
    return it->second;
}

std::vector<Wallet> WalletManager::get_user_wallets(const std::string& user_id) {
    std::vector<Wallet> result;
    for (const auto& [id, wallet] : pImpl->wallets_) {
        if (wallet.user_id == user_id) {
            result.push_back(wallet);
        }
    }
    return result;
}

} // namespace tigerwallet
