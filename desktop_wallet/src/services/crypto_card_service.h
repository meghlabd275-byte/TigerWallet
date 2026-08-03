// Crypto Card Service - C++ Desktop Implementation
// Virtual and Physical Crypto Cards

#ifndef CRYPTO_CARD_SERVICE_H
#define CRYPTO_CARD_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>

namespace tigerwallet {

struct CryptoCard {
    std::string id;
    std::string userId;
    std::string cardNumber;
    std::string cardHolder;
    std::string expiryDate;
    std::string cvv;
    std::string type;      // VIRTUAL, PHYSICAL
    std::string network;   // VISA, MASTERCARD
    std::string status;    // ACTIVE, FROZEN, TERMINATED
    double dailyLimit;
    double monthlyLimit;
    double dailySpent;
    double monthlySpent;
    bool applePayEnabled;
    bool googlePayEnabled;
    time_t createdAt;
    
    CryptoCard() : dailyLimit(10000), monthlyLimit(100000), dailySpent(0), monthlySpent(0),
                   applePayEnabled(false), googlePayEnabled(false), createdAt(0) {}
};

struct CardTransaction {
    std::string id;
    std::string cardId;
    std::string userId;
    double amount;
    std::string currency;
    std::string merchantName;
    std::string merchantCategory;
    std::string type;
    std::string status;
    time_t timestamp;
    
    CardTransaction() : amount(0), timestamp(0) {}
};

struct CardFundingSource {
    std::string id;
    std::string userId;
    std::string token;
    std::string symbol;
    double balance;
    double dailyLimit;
    double monthlyLimit;
    bool isDefault;
    
    CardFundingSource() : balance(0), dailyLimit(10000), monthlyLimit(100000), isDefault(false) {}
};

class CryptoCardService {
private:
    std::map<std::string, std::vector<CryptoCard>> cards_;
    std::map<std::string, std::vector<CardTransaction>> transactions_;
    std::map<std::string, std::vector<CardFundingSource>> fundingSources_;
    
public:
    CryptoCardService();
    ~CryptoCardService();
    
    // Get user's cards
    std::vector<CryptoCard> getCards(const std::string& userId);
    
    // Create virtual card
    CryptoCard createVirtualCard(const std::string& userId, const std::string& cardHolder,
                                 const std::string& type, const std::string& network,
                                 double dailyLimit = 10000, double monthlyLimit = 100000);
    
    // Create physical card
    CryptoCard createPhysicalCard(const std::string& userId, const std::string& cardHolder,
                                  const std::string& shippingAddress,
                                  double dailyLimit = 10000, double monthlyLimit = 100000);
    
    // Freeze card
    void freezeCard(const std::string& userId, const std::string& cardId);
    
    // Unfreeze card
    void unfreezeCard(const std::string& userId, const std::string& cardId);
    
    // Terminate card
    void terminateCard(const std::string& userId, const std::string& cardId);
    
    // Get transactions
    std::vector<CardTransaction> getTransactions(const std::string& userId, const std::string& cardId = "");
    
    // Process payment
    CardTransaction processPayment(const std::string& userId, const std::string& cardId,
                                   double amount, const std::string& currency,
                                   const std::string& merchantName);
    
    // Get funding sources
    std::vector<CardFundingSource> getFundingSources(const std::string& userId);
    
    // Add funding source
    void addFundingSource(const std::string& userId, const std::string& token,
                         const std::string& symbol, double balance, bool isDefault = false);
    
    // Set card limit
    void setCardLimit(const std::string& userId, const std::string& cardId,
                     double dailyLimit, double monthlyLimit);
    
    // Enable Apple Pay
    void enableApplePay(const std::string& userId, const std::string& cardId);
    
    // Enable Google Pay
    void enableGooglePay(const std::string& userId, const std::string& cardId);
};

} // namespace tigerwallet

#endif // CRYPTO_CARD_SERVICE_H
