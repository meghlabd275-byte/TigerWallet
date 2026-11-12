// TigerSmartChain support: the platform's own EVM network (chain id 7710).
//
// TigerSmartChain is where TigerWallet's own assets live. Nobody else runs this
// network, so this header owns the authoritative description of it:
//
//   TGR   the native coin. Like ETH on Ethereum it has no contract address;
//         balances are read with eth_getBalance.
//   WTGR  the ERC-20 wrapper of TGR, used by the swap pools.
//   RUSD  the USD stablecoin, minted against a treasury reserve.
//
// WTGR and RUSD are genesis predeploys: the genesis block already contains
// their runtime bytecode, so their addresses are fixed by consensus and
// identical on every node from block zero. That is why they are constants here
// and not something an operator deploys and then configures.
//
// The addresses are duplicated from tiger_smart_chain/chain/predeploy.go
// because the two live in different languages. test_platform_chain.cpp parses
// that Go source and fails if the two ever disagree, so the duplication cannot
// drift silently.

#ifndef TIGERWALLET_PLATFORM_CHAIN_HPP
#define TIGERWALLET_PLATFORM_CHAIN_HPP

#include <cstdint>
#include <optional>
#include <string>
#include <vector>

#include "chain.hpp"
#include "rpc.hpp"

namespace tigerwallet {
namespace platform_chain {

// EIP-155 chain id of TigerSmartChain mainnet.
inline constexpr uint64_t kChainId = 7710;

// EIP-155 chain id of the public testnet.
inline constexpr uint64_t kTestnetChainId = 7711;

// Native coin symbol.
inline constexpr const char* kNativeSymbol = "TGR";

// Environment variable holding the node's JSON-RPC endpoint.
inline constexpr const char* kRpcEnv = "TSC_RPC_URL";

// Environment variable holding the TGR/RUSD pool used to price TGR.
inline constexpr const char* kPairEnv = "TSC_TGR_RUSD_PAIR";

// RUSD stablecoin - genesis predeploy, fixed by consensus.
inline constexpr const char* kRusdAddress = "0x0000000000000000000000000000000000007710";

// WTGR wrapped-native token - genesis predeploy, fixed by consensus.
inline constexpr const char* kWtgrAddress = "0x0000000000000000000000000000000000007711";

// Display metadata, kept in step with the chain's own predeploy source.
inline constexpr const char* kRusdName = "Royal Tiger USD";
inline constexpr const char* kRusdSymbol = "RUSD";
inline constexpr const char* kWtgrName = "Wrapped TGR";
inline constexpr const char* kWtgrSymbol = "WTGR";

// How a platform asset is issued on the chain.
enum class AssetKind {
    // The chain's own coin. No contract; read with eth_getBalance.
    kNative,
    // An ERC-20 contract that already exists in the genesis block.
    kGenesisPredeploy,
};

// A first-class platform asset.
struct PlatformAsset {
    std::string symbol;
    std::string name;
    std::string address;  // empty for the native coin
    uint8_t decimals = 18;
    AssetKind kind = AssetKind::kNative;
    bool pegged_usd = false;

    bool is_contract() const { return kind == AssetKind::kGenesisPredeploy; }
};

// Every platform asset, in display order.
std::vector<PlatformAsset> PlatformAssets();

// Looks up an asset by symbol, case-insensitively. std::nullopt when absent.
std::optional<PlatformAsset> AssetBySymbol(const std::string& symbol);

// Looks up an asset by contract address, case-insensitively. The native coin
// has no address and can never be found this way.
std::optional<PlatformAsset> AssetByAddress(const std::string& address);

// Returns true when the string is exactly 40 hex digits with a 0x prefix.
bool IsHexAddress(const std::string& value);

// Decodes the low 20 bytes of a 32-byte ABI word into a lowercase address.
// std::nullopt when the word is short or the tail is not an address.
std::optional<std::string> AddressFromAbiWord(const std::string& word);

// Decodes a getReserves() return value into (reserve0, reserve1) as decimal
// strings, exactly. The two reserves are uint112, so the high 8 bytes of each
// word must be zero; a word that is not a valid uint112 is rejected rather than
// truncated. std::nullopt when the payload is short or malformed.
struct PoolReserves {
    std::string reserve0;
    std::string reserve1;
};
std::optional<PoolReserves> ParseReservesHex(const std::string& hex_result);

// Derives the dollar price of the non-RUSD leg of a two-token pool.
//
// `token0` and `token1` are the pool's token order, which decides which reserve
// belongs to RUSD. Both legs are 18-decimal assets on TigerSmartChain, so the
// ratio of the reserves is the price directly. std::nullopt when neither token
// is RUSD, when a reserve is zero (an empty pool has no price, which is not the
// same as a price of zero), or when the inputs are not addresses.
std::optional<double> PoolPriceUsd(const std::string& reserve0,
                                   const std::string& reserve1,
                                   const std::string& token0,
                                   const std::string& token1);

// The TigerSmartChain registry entry. The RPC endpoint is read from
// TSC_RPC_URL; when the variable is unset the entry still exists but carries an
// empty rpc_url, so a caller reports the chain as unconfigured rather than
// dialling a guessed host.
ChainConfig ChainConfigEntry();

// Returns the endpoint from TSC_RPC_URL, or an empty string when unset.
std::string RpcUrlFromEnv();

// Returns the TGR/RUSD pool address from TSC_TGR_RUSD_PAIR, or std::nullopt
// when unset or malformed.
std::optional<std::string> PairAddressFromEnv();

// A platform asset together with a live balance and fiat value.
struct PlatformHolding {
    PlatformAsset asset;
    // Balance in the asset's smallest unit (wei) as a decimal string. Kept as a
    // string because a uint256 does not fit in any integer type here and the
    // value must survive intact to the UI.
    std::string balance_raw;
    // Balance as a decimal string, e.g. "1.5".
    std::string balance;
    // USD value, or std::nullopt when no price source answered.
    std::optional<double> value_usd;
};

// Formats a raw decimal string amount with the given decimals. Pure string
// arithmetic, so no precision is lost on the way to display. Trailing
// fractional zeros are trimmed. A non-numeric input yields "0".
std::string FormatUnits(const std::string& raw, uint8_t decimals);

// Reads every platform asset balance for an address from a live node.
//
// Each asset is read independently: a token contract that reverts does not hide
// the native balance, it produces an error for that asset only. The returned
// result is a failure only when the native balance itself cannot be read.
struct HoldingsResult {
    bool ok = false;
    std::vector<PlatformHolding> holdings;
    RpcError error;
};

HoldingsResult ReadHoldings(const RpcClient& client, const std::string& address,
                            const std::optional<double>& tgr_price_usd);

// Reads the TGR price in USD from the live TGR/RUSD pool.
//
// RUSD is the dollar leg, so the ratio of the two reserves is the price
// directly and no external oracle is involved. Returns std::nullopt when the
// pool is unconfigured, unreachable, empty, or does not hold RUSD - an empty
// pool has no price, which is different from a price of zero.
std::optional<double> ReadTgrPriceUsd(const RpcClient& client);

}  // namespace platform_chain
}  // namespace tigerwallet

#endif  // TIGERWALLET_PLATFORM_CHAIN_HPP
