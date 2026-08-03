// Crypto Card Service Implementation
#include "crypto_card_service.h"
#include <sstream>
#include <iomanip>
#include <random>
#include <algorithm>

namespace tigerwallet {

CryptoCardService::CryptoCardService() {}
CryptoCardService::~CryptoCardService() {}

std::vector<CryptoCard> CryptoCardService::getCards(const std::string& userId) {
    if (cards_.find(userId) != cards_.end()) {
        return cards_[userId];
    }
    return std::vector<CryptoCard>();
}

std::string generateCardNumber() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 9);
    
    std::string number = "4532";
    for (int i = 0; i < 12; i++) {
        number += std::to_string(dis(gen));
    }
    return number;
}

std::string generateCVV() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 999);
    std::stringstream ss;
    ss << std::setw(3) << std::setfill('0') << dis(gen);
    return ss.str();
}

std::string generateExpiry() {
    time_t now = time(nullptr);
    tm* local = localtime(&now);
    int month = (local->tm_mon + 1) % 12 + 1;
    int year = (local->tm_year + 3) % 100;
    std::stringstream ss;
    ss << std::setw(2) << std::setfill('0') << month << "/" << year;
    return ss.str();
}

CryptoCard CryptoCardService::createVirtualCard(const std::string& userId, const std::string& cardHolder,
                                                 const std::string& type, const std::string& network,
                                                 double dailyLimit, double monthlyLimit) {
    CryptoCard card;
    std::stringstream ss;
    ss << "card_" << time(nullptr);
    card.id = ss.str();
    card.userId = userId;
    card.cardNumber = generateCardNumber();
    card.cardHolder = cardHolder;
    card.expiryDate = generateExpiry();
    card.cvv = generateCVV();
    card.type = type;
    card.network = network;
    card.status = "ACTIVE";
    card.dailyLimit = dailyLimit;
    card.monthlyLimit = monthlyLimit;
    card.dailySpent = 0;
    card.monthlySpent = 0;
    card.applePayEnabled = true;
    card.googlePayEnabled = true;
    card.createdAt = time(nullptr);
    
    cards_[userId].push_back(card);
    return card;
}

CryptoCard CryptoCardService::createPhysicalCard(const std::string& userId, const std::string& cardHolder,
                                                 const std::string& shippingAddress,
                                                 double dailyLimit, double monthlyLimit) {
    CryptoCard card = createVirtualCard(userId, cardHolder, "PHYSICAL", "VISA", dailyLimit, monthlyLimit);
    card.status = "PENDING_ACTIVATION";
    return card;
}

void CryptoCardService::freezeCard(const std::string& userId, const std::string& cardId) {
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.status = "FROZEN";
            break;
        }
    }
}

void CryptoCardService::unfreezeCard(const std::string& userId, const std::string& cardId) {
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.status = "ACTIVE";
            break;
        }
    }
}

void CryptoCardService::terminateCard(const std::string& userId, const std::string& cardId) {
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.status = "TERMINATED";
            card.applePayEnabled = false;
            card.googlePayEnabled = false;
            break;
        }
    }
}

std::vector<CardTransaction> CryptoCardService::getTransactions(const std::string& userId, const std::string& cardId) {
    if (transactions_.find(userId) != transactions_.end()) {
        if (!cardId.empty()) {
            std::vector<CardTransaction> result;
            for (const auto& txn : transactions_[userId]) {
                if (txn.cardId == cardId) {
                    result.push_back(txn);
                }
            }
            return result;
        }
        return transactions_[userId];
    }
    return std::vector<CardTransaction>();
}

CardTransaction CryptoCardService::processPayment(const std::string& userId, const std::string& cardId,
                                                   double amount, const std::string& currency,
                                                   const std::string& merchantName) {
    CardTransaction txn;
    std::stringstream ss;
    ss << "txn_" << time(nullptr);
    txn.id = ss.str();
    txn.cardId = cardId;
    txn.userId = userId;
    txn.amount = amount;
    txn.currency = currency;
    txn.merchantName = merchantName;
    txn.merchantCategory = "RETAIL";
    txn.type = "PURCHASE";
    txn.status = "COMPLETED";
    txn.timestamp = time(nullptr);
    
    // Update card spending
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.dailySpent += amount;
            card.monthlySpent += amount;
            break;
        }
    }
    
    transactions_[userId].push_back(txn);
    return txn;
}

std::vector<CardFundingSource> CryptoCardService::getFundingSources(const std::string& userId) {
    if (fundingSources_.find(userId) != fundingSources_.end()) {
        return fundingSources_[userId];
    }
    return std::vector<CardFundingSource>();
}

void CryptoCardService::addFundingSource(const std::string& userId, const std::string& token,
                                         const std::string& symbol, double balance, bool isDefault) {
    CardFundingSource source;
    std::stringstream ss;
    ss << "fund_" << time(nullptr);
    source.id = ss.str();
    source.userId = userId;
    source.token = token;
    source.symbol = symbol;
    source.balance = balance;
    source.dailyLimit = 10000;
    source.monthlyLimit = 100000;
    source.isDefault = isDefault;
    
    fundingSources_[userId].push_back(source);
}

void CryptoCardService::setCardLimit(const std::string& userId, const std::string& cardId,
                                     double dailyLimit, double monthlyLimit) {
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.dailyLimit = dailyLimit;
            card.monthlyLimit = monthlyLimit;
            break;
        }
    }
}

void CryptoCardService::enableApplePay(const std::string& userId, const std::string& cardId) {
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.applePayEnabled = true;
            break;
        }
    }
}

void CryptoCardService::enableGooglePay(const std::string& userId, const std::string& cardId) {
    auto& userCards = cards_[userId];
    for (auto& card : userCards) {
        if (card.id == cardId) {
            card.googlePayEnabled = true;
            break;
        }
    }
}

} // namespace tigerwallet
