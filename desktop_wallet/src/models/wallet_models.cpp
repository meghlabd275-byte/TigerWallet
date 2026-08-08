/**
 * TigerWallet Desktop - Models Implementation
 */

#include "models/wallet_models.h"
#include <iostream>
#include <sstream>
#include <iomanip>
#include <random>
#include <cmath>
#include <curl/curl.h>
#include <openssl/sha.h>
#include <openssl/rand.h>
#include <algorithm>

namespace tiger {
namespace wallet {

// ============================================================================
// Real Token Data Implementation - Fetches from CoinGecko API
// ============================================================================

static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
    ((std::string*)userp)->append((char*)contents, size * nmemb);
    return size * nmemb;
}

std::vector<RealTokenData> RealTokenData::fetchFromAPI() {
    std::vector<RealTokenData> tokens;
    
    CURL* curl = curl_easy_init();
    if (!curl) return tokens;
    
    std::string readBuffer;
    
    curl_easy_setopt(curl, CURLOPT_URL, "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=500&page=1&sparkline=false");
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &readBuffer);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);
    
    CURLcode res = curl_easy_perform(curl);
    curl_easy_cleanup(curl);
    
    if (res != CURLE_OK) {
        std::cerr << "Failed to fetch tokens from CoinGecko: " << curl_easy_strerror(res) << std::endl;
        return tokens;
    }
    
    // Note: In production, parse JSON response here using a JSON library
    // For now, the service layer handles API calls
    
    return tokens;
}

RealTokenData RealTokenData::getToken(const std::vector<RealTokenData>& tokens, const std::string& symbol) {
    for (const auto& token : tokens) {
        if (token.symbol == symbol) {
            return token;
        }
    }
    return RealTokenData{};
}

std::vector<RealTokenData> RealTokenData::getTopTokens(const std::vector<RealTokenData>& tokens, int limit) {
    std::vector<RealTokenData> sorted = tokens;
    std::sort(sorted.begin(), sorted.end(), [](const RealTokenData& a, const RealTokenData& b) {
        return a.market_cap > b.market_cap;
    });
    
    if (limit > 0 && limit < (int)sorted.size()) {
        sorted.resize(limit);
    }
    return sorted;
}

std::vector<RealTokenData> RealTokenData::searchTokens(const std::vector<RealTokenData>& tokens, const std::string& query) {
    std::vector<RealTokenData> results;
    std::string lowerQuery = query;
    std::transform(lowerQuery.begin(), lowerQuery.end(), lowerQuery.begin(), ::tolower);
    
    for (const auto& token : tokens) {
        std::string lowerName = token.name;
        std::string lowerSymbol = token.symbol;
        std::transform(lowerName.begin(), lowerName.end(), lowerName.begin(), ::tolower);
        std::transform(lowerSymbol.begin(), lowerSymbol.end(), lowerSymbol.begin(), ::tolower);
        
        if (lowerName.find(lowerQuery) != std::string::npos || 
            lowerSymbol.find(lowerQuery) != std::string::npos) {
            results.push_back(token);
        }
    }
    return results;
}

// ============================================================================
// Token Implementation
// ============================================================================

std::string Token::getDisplayBalance() const {
    std::ostringstream oss;
    oss << std::fixed << std::setprecision(4) << balance;
    return oss.str();
}

std::string Token::getDisplayPrice() const {
    std::ostringstream oss;
    oss << "$" << std::fixed << std::setprecision(2) << price;
    return oss.str();
}

std::string Token::getDisplayValue() const {
    std::ostringstream oss;
    oss << "$" << std::fixed << std::setprecision(2) << balance_usd;
    return oss.str();
}

// ============================================================================
// Wallet Implementation
// ============================================================================

std::string Wallet::getShortAddress() const {
    if (address.length() > 10) {
        return address.substr(0, 6) + "..." + address.substr(address.length() - 4);
    }
    return address;
}

// ============================================================================
// Transaction Implementation
// ============================================================================

std::string Transaction::statusToString(TransactionStatus status) {
    switch (status) {
        case TransactionStatus::PENDING: return "pending";
        case TransactionStatus::CONFIRMED: return "confirmed";
        case TransactionStatus::FAILED: return "failed";
        default: return "unknown";
    }
}

std::string Transaction::typeToString(TransactionType type) {
    switch (type) {
        case TransactionType::SEND: return "send";
        case TransactionType::RECEIVE: return "receive";
        case TransactionType::SWAP: return "swap";
        case TransactionType::STAKE: return "stake";
        case TransactionType::UNSTAKE: return "unstake";
        case TransactionType::APPROVE: return "approve";
        case TransactionType::CONTRACT_INTERACTION: return "contract_interaction";
        case TransactionType::NFT_TRANSFER: return "nft_transfer";
        default: return "unknown";
    }
}

// ============================================================================
// SwapQuote Implementation
// ============================================================================

std::string SwapQuote::getDisplayFromAmount() const {
    std::ostringstream oss;
    oss << std::fixed << std::setprecision(8) << from_amount << " " << from_token;
    return oss.str();
}

std::string SwapQuote::getDisplayToAmount() const {
    std::ostringstream oss;
    oss << std::fixed << std::setprecision(8) << to_amount << " " << to_token;
    return oss.str();
}

// ============================================================================
// PriceInfo Implementation
// ============================================================================

bool PriceInfo::isPositive() const {
    return change_percent_24h >= 0;
}

std::string PriceInfo::getFormattedPrice() const {
    std::ostringstream oss;
    if (price >= 1.0) {
        oss << "$" << std::fixed << std::setprecision(2) << price;
    } else {
        oss << "$" << std::fixed << std::setprecision(6) << price;
    }
    return oss.str();
}

std::string PriceInfo::getFormattedChange() const {
    std::ostringstream oss;
    oss << (change_percent_24h >= 0 ? "+" : "")
        << std::fixed << std::setprecision(2) << change_percent_24h << "%";
    return oss.str();
}

std::string PriceInfo::getFormattedMarketCap() const {
    std::ostringstream oss;
    if (market_cap >= 1e12) {
        oss << "$" << std::fixed << std::setprecision(2) << (market_cap / 1e12) << "T";
    } else if (market_cap >= 1e9) {
        oss << "$" << std::fixed << std::setprecision(2) << (market_cap / 1e9) << "B";
    } else if (market_cap >= 1e6) {
        oss << "$" << std::fixed << std::setprecision(2) << (market_cap / 1e6) << "M";
    } else {
        oss << "$" << std::fixed << std::setprecision(0) << market_cap;
    }
    return oss.str();
}

// ============================================================================
// Utility Functions
// ============================================================================

std::string generateUUID() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 15);
    std::uniform_int_distribution<> dis2(8, 11);

    std::ostringstream oss;
    oss << std::hex;
    for (int i = 0; i < 8; i++) oss << dis(gen);
    oss << "-";
    for (int i = 0; i < 4; i++) oss << dis(gen);
    oss << "-4";
    for (int i = 0; i < 3; i++) oss << dis(gen);
    oss << "-";
    oss << dis2(gen);
    for (int i = 0; i < 3; i++) oss << dis(gen);
    oss << "-";
    for (int i = 0; i < 12; i++) oss << dis(gen);
    return oss.str();
}

std::string getCurrentTimestamp() {
    auto now = std::chrono::system_clock::now();
    auto time_t = std::chrono::system_clock::to_time_t(now);
    std::ostringstream oss;
    oss << std::put_time(std::gmtime(&time_t), "%Y-%m-%dT%H:%M:%SZ");
    return oss.str();
}

double hexToDouble(const std::string& hex, int decimals) {
    std::string clean_hex = hex;
    if (clean_hex.find("0x") == 0 || clean_hex.find("0X") == 0) {
        clean_hex = clean_hex.substr(2);
    }
    
    unsigned long long value = 0;
    std::istringstream iss(clean_hex);
    iss >> std::hex >> value;
    
    double divisor = std::pow(10.0, decimals);
    return static_cast<double>(value) / divisor;
}

std::string doubleToHex(double value, int decimals) {
    unsigned long long raw_value = static_cast<unsigned long long>(value * std::pow(10, decimals));
    std::ostringstream oss;
    oss << "0x" << std::hex << raw_value;
    return oss.str();
}

} // namespace wallet
} // namespace tiger
