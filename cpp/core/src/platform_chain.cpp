// TigerSmartChain support. See include/platform_chain.hpp.
//
// Everything here reads live chain state through RpcClient. Nothing is
// estimated, cached from a fixture, or defaulted to a plausible number: if the
// node cannot be reached, or the pool does not hold RUSD, the function reports
// that no value is available.

#include "platform_chain.hpp"

#include <algorithm>
#include <cctype>
#include <cstdlib>
#include <string>
#include <vector>

namespace tigerwallet {
namespace platform_chain {
namespace {

// Lowercases an ASCII string.
std::string ToLower(const std::string& s) {
    std::string out;
    out.reserve(s.size());
    for (const char c : s) {
        out.push_back(static_cast<char>(
            std::tolower(static_cast<unsigned char>(c))));
    }
    return out;
}

// Trims ASCII whitespace from both ends.
std::string Trim(const std::string& s) {
    const auto begin = s.find_first_not_of(" \t\r\n");
    if (begin == std::string::npos) return {};
    const auto end = s.find_last_not_of(" \t\r\n");
    return s.substr(begin, end - begin + 1);
}

// Reads an environment variable, returning an empty string when unset.
std::string Env(const char* name) {
    const char* value = std::getenv(name);
    return value == nullptr ? std::string() : Trim(value);
}

// Formats a decimal big integer as a human-readable amount with `decimals`
// fractional digits.
//
// The value arrives as a decimal string, not a machine integer, because a
// uint256 balance does not fit in any integer type available here and must
// survive intact to the UI. Everything below is string arithmetic, so no
// precision is lost and no floating-point rounding is introduced.
std::string FormatDecimalString(const std::string& raw, uint8_t decimals) {
    if (decimals == 0) {
        const auto first = raw.find_first_not_of('0');
        return first == std::string::npos ? "0" : raw.substr(first);
    }
    std::string padded = raw;
    if (padded.size() <= decimals) {
        padded.insert(0, decimals + 1 - padded.size(), '0');
    }
    const std::string whole = padded.substr(0, padded.size() - decimals);
    std::string frac = padded.substr(padded.size() - decimals);
    while (!frac.empty() && frac.back() == '0') frac.pop_back();
    const auto first_whole = whole.find_first_not_of('0');
    const std::string whole_out =
        first_whole == std::string::npos ? "0" : whole.substr(first_whole);
    if (frac.empty()) return whole_out;
    return whole_out + "." + frac;
}

// Converts a hex quantity into its decimal string, exactly. A 32-byte word is
// reduced one hex digit at a time by repeated multiplication and addition on
// the decimal representation, which keeps every digit.
std::string HexToDecimalString(const std::string& hex) {
    std::string decimal = "0";
    for (const char c : hex) {
        int digit;
        if (c >= '0' && c <= '9') {
            digit = c - '0';
        } else if (c >= 'a' && c <= 'f') {
            digit = c - 'a' + 10;
        } else if (c >= 'A' && c <= 'F') {
            digit = c - 'A' + 10;
        } else {
            return {};
        }
        // decimal = decimal * 16 + digit
        std::string scaled;
        scaled.reserve(decimal.size() + 1);
        int carry = 0;
        for (auto it = decimal.rbegin(); it != decimal.rend(); ++it) {
            const int product = (*it - '0') * 16 + carry;
            scaled.push_back(static_cast<char>('0' + product % 10));
            carry = product / 10;
        }
        while (carry > 0) {
            scaled.push_back(static_cast<char>('0' + carry % 10));
            carry /= 10;
        }
        std::reverse(scaled.begin(), scaled.end());

        int add_carry = digit;
        for (auto it = scaled.rbegin(); it != scaled.rend() && add_carry > 0; ++it) {
            const int sum = (*it - '0') + add_carry;
            *it = static_cast<char>('0' + sum % 10);
            add_carry = sum / 10;
        }
        if (add_carry > 0) scaled.insert(scaled.begin(), static_cast<char>('0' + add_carry));
        decimal = scaled;
    }
    return decimal;
}


// Converts a decimal string to a double for presentation maths only.
double DecimalToDouble(const std::string& decimal) {
    if (decimal.empty()) return 0.0;
    try {
        const double value = std::stod(decimal);
        return value >= 0.0 ? value : 0.0;
    } catch (...) {
        return 0.0;
    }
}

// Strips the 0x prefix from a hex string.
std::string Strip0x(const std::string& value) {
    if (value.size() >= 2 && value[0] == '0' &&
        (value[1] == 'x' || value[1] == 'X')) {
        return value.substr(2);
    }
    return value;
}

}  // namespace

bool IsHexAddress(const std::string& value) {
    // The 0x prefix is required. Accepting a bare 40-character string as well
    // would make the accepted set ambiguous, and every caller in this module
    // gets its addresses from an ABI payload or an environment variable where
    // the prefix is part of the format.
    if (value.size() < 2 || value[0] != '0' || (value[1] != 'x' && value[1] != 'X')) {
        return false;
    }
    const std::string hex = Strip0x(value);
    if (hex.size() != 40) return false;
    for (const char c : hex) {
        if (!std::isxdigit(static_cast<unsigned char>(c))) return false;
    }
    return true;
}

std::optional<std::string> AddressFromAbiWord(const std::string& word) {
    const std::string hex = Strip0x(word);
    if (hex.size() < 64) return std::nullopt;
    const std::string tail = hex.substr(24, 40);
    if (!IsHexAddress("0x" + tail)) return std::nullopt;
    return "0x" + ToLower(tail);
}

std::optional<PoolReserves> ParseReservesHex(const std::string& hex_result) {
    const std::string hex = Strip0x(hex_result);
    if (hex.size() < 192) return std::nullopt;

    // A uint112 needs at most 28 hex digits, so the high 8 bytes of each word
    // must be zero. A word that fails this is not a reserve, and truncating it
    // would silently produce a wrong price.
    const std::string word0 = hex.substr(0, 64);
    const std::string word1 = hex.substr(64, 64);
    const auto high_is_zero = [](const std::string& word) {
        return word.substr(0, 24).find_first_not_of('0') == std::string::npos;
    };
    if (!high_is_zero(word0) || !high_is_zero(word1)) return std::nullopt;

    PoolReserves out;
    out.reserve0 = HexToDecimalString(word0);
    out.reserve1 = HexToDecimalString(word1);
    if (out.reserve0.empty() || out.reserve1.empty()) return std::nullopt;
    return out;
}

std::optional<double> PoolPriceUsd(const std::string& reserve0,
                                   const std::string& reserve1,
                                   const std::string& token0,
                                   const std::string& token1) {
    if (!IsHexAddress(token0) || !IsHexAddress(token1)) return std::nullopt;
    if (reserve0.empty() || reserve1.empty()) return std::nullopt;
    if (reserve0.find_first_not_of("0123456789") != std::string::npos) return std::nullopt;
    if (reserve1.find_first_not_of("0123456789") != std::string::npos) return std::nullopt;

    const std::string t0 = ToLower(token0);
    const std::string t1 = ToLower(token1);
    const std::string rusd = ToLower(kRusdAddress);

    std::string asset_reserve;
    std::string rusd_reserve;
    if (t0 == rusd && t1 != rusd) {
        rusd_reserve = reserve0;
        asset_reserve = reserve1;
    } else if (t1 == rusd && t0 != rusd) {
        rusd_reserve = reserve1;
        asset_reserve = reserve0;
    } else {
        // Neither leg is RUSD, or both are (which cannot happen for a real
        // pool). Either way the ratio is not a dollar price.
        return std::nullopt;
    }

    if (asset_reserve == "0" || rusd_reserve == "0") {
        // An empty pool has no price. That is not the same as a price of zero,
        // so it is reported as unavailable.
        return std::nullopt;
    }

    const double asset = DecimalToDouble(asset_reserve);
    const double dollars = DecimalToDouble(rusd_reserve);
    if (!(asset > 0.0)) return std::nullopt;
    const double price = dollars / asset;
    if (!(price > 0.0)) return std::nullopt;
    return price;
}

std::vector<PlatformAsset> PlatformAssets() {
    return {
        PlatformAsset{kNativeSymbol, "Tiger", "", 18, AssetKind::kNative, false},
        PlatformAsset{kWtgrSymbol, kWtgrName, kWtgrAddress, 18,
                      AssetKind::kGenesisPredeploy, false},
        PlatformAsset{kRusdSymbol, kRusdName, kRusdAddress, 18,
                      AssetKind::kGenesisPredeploy, true},
    };
}

std::optional<PlatformAsset> AssetBySymbol(const std::string& symbol) {
    const std::string want = ToLower(Trim(symbol));
    if (want.empty()) return std::nullopt;
    for (const auto& asset : PlatformAssets()) {
        if (ToLower(asset.symbol) == want) return asset;
    }
    return std::nullopt;
}

std::optional<PlatformAsset> AssetByAddress(const std::string& address) {
    if (!IsHexAddress(Trim(address))) return std::nullopt;
    const std::string want = ToLower(Trim(address));
    for (const auto& asset : PlatformAssets()) {
        if (!asset.address.empty() && ToLower(asset.address) == want) {
            return asset;
        }
    }
    return std::nullopt;
}

std::string RpcUrlFromEnv() { return Env(kRpcEnv); }

std::optional<std::string> PairAddressFromEnv() {
    const std::string pair = Env(kPairEnv);
    if (!IsHexAddress(pair)) return std::nullopt;
    return pair;
}

ChainConfig ChainConfigEntry() {
    ChainConfig config;
    config.chain_id = kChainId;
    config.name = "TigerSmartChain";
    config.native_symbol = kNativeSymbol;
    config.native_decimals = 18;
    // No explorer is bundled: the platform runs its own, and a guessed URL
    // would send users to a page that does not exist.
    config.explorer_url = "";
    config.rpc_url = RpcUrlFromEnv();
    config.is_evm = true;
    config.enabled = true;
    return config;
}

std::string FormatUnits(const std::string& raw, uint8_t decimals) {
    const std::string digits = Trim(raw);
    if (digits.empty() || digits.find_first_not_of("0123456789") != std::string::npos) {
        return "0";
    }
    return FormatDecimalString(digits, decimals);
}

std::optional<double> ReadTgrPriceUsd(const RpcClient& client) {
    const auto pair = PairAddressFromEnv();
    if (!pair.has_value()) return std::nullopt;

    // getReserves() -> (uint112, uint112, uint32)
    const auto reserves_call = client.EthCall(*pair, "0x0902f1ac");
    if (!reserves_call.ok) return std::nullopt;
    const auto reserves = ParseReservesHex(reserves_call.value);
    if (!reserves.has_value()) return std::nullopt;

    // token0() / token1() decide which reserve is RUSD. Read them rather than
    // assume, because guessing would invert the price.
    const auto token0_call = client.EthCall(*pair, "0x0dfe1681");
    const auto token1_call = client.EthCall(*pair, "0x4f1a4a83");
    if (!token0_call.ok || !token1_call.ok) return std::nullopt;
    const auto t0 = AddressFromAbiWord(token0_call.value);
    const auto t1 = AddressFromAbiWord(token1_call.value);
    if (!t0.has_value() || !t1.has_value()) return std::nullopt;

    return PoolPriceUsd(reserves->reserve0, reserves->reserve1, *t0, *t1);
}

HoldingsResult ReadHoldings(const RpcClient& client, const std::string& address,
                            const std::optional<double>& tgr_price_usd) {
    HoldingsResult result;
    if (!IsHexAddress(address)) {
        result.error = RpcError{-32602, "not a 20-byte address: " + address};
        return result;
    }

    for (const auto& asset : PlatformAssets()) {
        PlatformHolding holding;
        holding.asset = asset;

        if (asset.is_contract()) {
            // balanceOf(address): selector 0x70a08231 + the address left-padded
            // to 32 bytes.
            const std::string padded = std::string(24, '0') + ToLower(Strip0x(address));
            const auto call = client.EthCall(asset.address, "0x70a08231" + padded);
            if (!call.ok) {
                // A reverting token must not hide the other assets. Record the
                // error against this asset by leaving the balance empty.
                holding.balance_raw = "";
                holding.balance = "";
                holding.value_usd = std::nullopt;
                result.holdings.push_back(std::move(holding));
                continue;
            }
            const std::string hex = Strip0x(call.value);
            const std::string raw = hex.empty() ? std::string() : HexToDecimalString(hex);
            if (raw.empty()) {
                holding.balance_raw = "";
                holding.balance = "";
                holding.value_usd = std::nullopt;
                result.holdings.push_back(std::move(holding));
                continue;
            }
            holding.balance_raw = raw;
        } else {
            const auto balance = client.GetBalance(address);
            if (!balance.ok) {
                result.error = balance.error;
                return result;
            }
            const std::string hex = Strip0x(balance.value);
            const std::string raw = hex.empty() ? std::string() : HexToDecimalString(hex);
            if (raw.empty()) {
                result.error = RpcError{-32603, "native balance is not a valid quantity"};
                return result;
            }
            holding.balance_raw = raw;
        }

        holding.balance = FormatUnits(holding.balance_raw, asset.decimals);
        if (asset.pegged_usd) {
            // RUSD is redeemable one-for-one by design, so its fiat value is the
            // balance itself. That is a property of the asset, not a guessed rate.
            holding.value_usd = DecimalToDouble(holding.balance);
        } else if (tgr_price_usd.has_value()) {
            holding.value_usd = DecimalToDouble(holding.balance) * *tgr_price_usd;
        } else {
            holding.value_usd = std::nullopt;
        }
        result.holdings.push_back(std::move(holding));
    }

    result.ok = true;
    return result;
}

}  // namespace platform_chain
}  // namespace tigerwallet
