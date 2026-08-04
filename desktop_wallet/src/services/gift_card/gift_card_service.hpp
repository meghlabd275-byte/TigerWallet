#ifndef GIFT_CARD_SERVICE_HPP
#define GIFT_CARD_SERVICE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
#include <chrono>
#include <random>

using namespace std;
using namespace std::chrono;

struct GiftCardBrand {
    string id;
    string name;
    string logoUrl;
    double discount;
    vector<double> denominations;
    bool isActive;
};

struct GiftCard {
    string id;
    string code;
    string brandId;
    double amount;
    string status;  // ACTIVE, REDEEMED, EXPIRED
    string purchasedBy;
    string redeemedBy;
    uint64_t purchasedAt;
    uint64_t redeemedAt;
    uint64_t expiresAt;
};

class GiftCardService {
private:
    mutex mutex_;
    unordered_map<string, GiftCardBrand> brands_;
    unordered_map<string, GiftCard> giftCards_;
    unordered_map<string, vector<GiftCard>> userCards_;
    uint64_t cardCounter_;
    random_device rd_;

public:
    GiftCardService() : cardCounter_(0) {
        initializeBrands();
    }

    void initializeBrands() {
        lock_guard<mutex> lock(mutex_);
        
        GiftCardBrand amazon;
        amazon.id = "amazon";
        amazon.name = "Amazon";
        amazon.discount = 0.03;
        amazon.denominations = {10, 25, 50, 100, 200};
        amazon.isActive = true;
        brands_["amazon"] = amazon;

        GiftCardBrand apple;
        apple.id = "apple";
        apple.name = "Apple";
        apple.discount = 0.02;
        apple.denominations = {15, 25, 50, 100};
        apple.isActive = true;
        brands_["apple"] = apple;

        GiftCardBrand google;
        google.id = "google";
        google.name = "Google Play";
        google.discount = 0.04;
        google.denominations = {10, 25, 50, 100};
        google.isActive = true;
        brands_["google"] = google;
    }

    vector<GiftCardBrand> getBrands() {
        lock_guard<mutex> lock(mutex_);
        vector<GiftCardBrand> result;
        for (auto& b : brands_) {
            if (b.second.isActive) {
                result.push_back(b.second);
            }
        }
        return result;
    }

    GiftCard purchaseGiftCard(string userId, string brandId, double amount) {
        lock_guard<mutex> lock(mutex_);
        
        auto brandIt = brands_.find(brandId);
        if (brandIt == brands_.end()) {
            return {};
        }

        string cardId = "GC-" + to_string(++cardCounter_);
        string code = generateCode();

        GiftCard card;
        card.id = cardId;
        card.code = code;
        card.brandId = brandId;
        card.amount = amount;
        card.status = "ACTIVE";
        card.purchasedBy = userId;
        card.purchasedAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        card.expiresAt = card.purchasedAt + 365ULL * 24 * 60 * 60 * 1000;

        giftCards_[cardId] = card;
        userCards_[userId].push_back(card);
        
        return card;
    }

    bool redeemGiftCard(string code, string userId) {
        lock_guard<mutex> lock(mutex_);
        
        for (auto& pair : giftCards_) {
            GiftCard& card = pair.second;
            if (card.code == code && card.status == "ACTIVE") {
                card.status = "REDEEMED";
                card.redeemedBy = userId;
                card.redeemedAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
                return true;
            }
        }
        return false;
    }

    vector<GiftCard> getUserCards(string userId) {
        lock_guard<mutex> lock(mutex_);
        return userCards_[userId];
    }

private:
    string generateCode() {
        const string chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
        string code;
        for (int i = 0; i < 16; i++) {
            code += chars[rd_() % chars.length()];
        }
        return code;
    }
};

#endif
