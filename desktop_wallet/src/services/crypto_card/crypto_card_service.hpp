#ifndef CRYPTO_CARD_SERVICE_HPP
#define CRYPTO_CARD_SERVICE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
#include <chrono>

using namespace std;
using namespace std::chrono;

enum class CardType { VIRTUAL, PHYSICAL };
enum class CardStatus { ACTIVE, BLOCKED, EXPIRED };

struct VirtualCard {
    string id;
    string userId;
    string address;
    string network;
    double balance;
    double dailyLimit;
    CardType type;
    CardStatus status;
    uint64_t createdAt;
    uint64_t expiresAt;
};

struct CardTransaction {
    string id;
    string cardId;
    string type;
    double amount;
    string currency;
    string merchant;
    string status;
    uint64_t timestamp;
};

class CryptoCardService {
private:
    mutex mutex_;
    unordered_map<string, VirtualCard> cards_;
    unordered_map<string, vector<CardTransaction>> cardTransactions_;
    uint64_t cardCounter_;

public:
    CryptoCardService() : cardCounter_(0) {}

    VirtualCard createVirtualCard(string userId, string network, double dailyLimit) {
        lock_guard<mutex> lock(mutex_);
        string cardId = "CARD-" + to_string(++cardCounter_);
        
        VirtualCard card;
        card.id = cardId;
        card.userId = userId;
        card.address = "0x" + to_string(cardCounter_);
        card.network = network;
        card.balance = 0;
        card.dailyLimit = dailyLimit;
        card.type = CardType::VIRTUAL;
        card.status = CardStatus::ACTIVE;
        card.createdAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        card.expiresAt = card.createdAt + 365ULL * 24 * 60 * 60 * 1000;
        
        cards_[cardId] = card;
        return card;
    }

    bool topUp(string cardId, double amount) {
        lock_guard<mutex> lock(mutex_);
        auto it = cards_.find(cardId);
        if (it != cards_.end() && it->second.status == CardStatus::ACTIVE) {
            it->second.balance += amount;
            return true;
        }
        return false;
    }

    double getBalance(string cardId) {
        lock_guard<mutex> lock(mutex_);
        auto it = cards_.find(cardId);
        if (it != cards_.end()) {
            return it->second.balance;
        }
        return 0;
    }

    bool blockCard(string cardId) {
        lock_guard<mutex> lock(mutex_);
        auto it = cards_.find(cardId);
        if (it != cards_.end()) {
            it->second.status = CardStatus::BLOCKED;
            return true;
        }
        return false;
    }

    vector<CardTransaction> getTransactions(string cardId) {
        lock_guard<mutex> lock(mutex_);
        return cardTransactions_[cardId];
    }
};

#endif
