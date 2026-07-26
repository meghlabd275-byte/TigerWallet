/**
 * TigerWallet Crypto Card Processor - Header File
 * High-performance C++ card processing engine for crypto card operations
 * Supports virtual and physical cards with multi-currency support
 */

#ifndef TIGER_CARD_PROCESSOR_H
#define TIGER_CARD_PROCESSOR_H

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <mutex>
#include <thread>
#include <chrono>
#include <memory>
#include <optional>
#include <array>
#include <cstdint>
#include <sstream>
#include <iomanip>
#include <random>
#include <functional>

#ifdef CRYPTOPP_AVAILABLE
#include <cryptopp/aes.h>
#include <cryptopp/modes.h>
#include <cryptopp/sha.h>
#include <cryptopp/ripemd.h>
#include <cryptopp/base64.h>
#endif

namespace tiger {
namespace card {

// Constants
constexpr size_t CARD_NUMBER_LENGTH = 16;
constexpr size_t CVV_LENGTH = 3;
constexpr size_t MAX_CARDS_PER_USER = 10;
constexpr uint32_t MAX_TRANSACTION_AMOUNT = 1000000;
constexpr uint32_t MIN_TRANSACTION_AMOUNT = 1;

// Enums
enum class CardType : uint8_t {
    VIRTUAL = 0x01,
    PHYSICAL = 0x02,
    VIRTUAL_ONE_TIME = 0x03,
    METAL = 0x04
};

enum class CardStatus : uint8_t {
    PENDING = 0x01,
    ACTIVE = 0x02,
    BLOCKED = 0x03,
    EXPIRED = 0x04,
    CANCELLED = 0x05,
    FROZEN = 0x06
};

enum class CardNetwork : uint8_t {
    VISA = 0x01,
    MASTERCARD = 0x02,
    AMEX = 0x03,
    UNIONPAY = 0x04
};

enum class CurrencyCode : uint16_t {
    USD = 840,
    EUR = 978,
    GBP = 826,
    JPY = 392,
    CNY = 156,
    BTC = 0,
    ETH = 0,
    USDT = 0
};

enum class TransactionType : uint8_t {
    PURCHASE = 0x01,
    WITHDRAWAL = 0x02,
    REFUND = 0x03,
    TRANSFER = 0x04,
    TOP_UP = 0x05,
    FEE = 0x06
};

enum class TransactionStatus : uint8_t {
    PENDING = 0x01,
    COMPLETED = 0x02,
    FAILED = 0x03,
    CANCELLED = 0x04,
    FLAGGED = 0x05
};

// Structures
struct CardHolder {
    std::string user_id;
    std::string name;
    std::string email;
    std::string phone;
    std::string billing_address;
    std::string country;
    std::string city;
    std::string postal_code;
    std::string kyc_level;
    uint8_t risk_level;
};

struct CardData {
    std::string card_id;
    std::string user_id;
    std::array<uint8_t, CARD_NUMBER_LENGTH> card_number;
    std::array<uint8_t, CVV_LENGTH> cvv;
    uint16_t expiry_month;
    uint16_t expiry_year;
    CardType card_type;
    CardStatus status;
    CardNetwork network;
    CurrencyCode currency;
    std::string card_holder_name;
    std::string billing_address;
    uint32_t daily_limit;
    uint32_t monthly_limit;
    uint32_t daily_spent;
    uint32_t monthly_spent;
    uint32_t max_single_transaction;
    uint32_t min_single_transaction;
    std::vector<std::string> allowed_merchants;
    std::vector<std::string> blocked_merchants;
    bool contactless_enabled;
    bool online_payments_enabled;
    bool international_enabled;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    std::chrono::system_clock::time_point expires_at;
};

struct Transaction {
    std::string transaction_id;
    std::string card_id;
    std::string user_id;
    TransactionType type;
    TransactionStatus status;
    CurrencyCode currency;
    int64_t amount;
    int64_t fee;
    int64_t crypto_amount;
    std::string crypto_currency;
    std::string merchant_id;
    std::string merchant_name;
    std::string merchant_category;
    std::string terminal_id;
    std::string location;
    std::string ip_address;
    std::string description;
    std::string reference_id;
    std::string authorization_code;
    std::chrono::system_clock::time_point timestamp;
    std::chrono::system_clock::time_point settled_at;
    std::string blockchain_tx_hash;
    uint8_t risk_score;
    std::string risk_reason;
};

struct CardLimits {
    uint32_t daily_limit;
    uint32_t monthly_limit;
    uint32_t max_single_transaction;
    uint32_t min_single_transaction;
    uint32_t daily_withdrawal_limit;
    uint32_t monthly_withdrawal_limit;
};

struct RiskAssessment {
    uint8_t score;
    bool approved;
    std::vector<std::string> flags;
    std::string recommendation;
    std::string review_status;
};

struct CryptoRate {
    std::string from_currency;
    std::string to_currency;
    double rate;
    std::chrono::system_clock::time_point updated_at;
    uint64_t block_number;
};

// Utility Classes
class SecureRandom {
public:
    static uint8_t next_byte();
    static void next_bytes(std::vector<uint8_t>& buffer);
    static uint32_t next_uint32();
};

class LuhnValidator {
public:
    static bool validate(const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number);
    static uint8_t calculate_check_digit(const std::array<uint8_t, CARD_NUMBER_LENGTH - 1>& partial);
};

class CardNumberGenerator {
public:
    static std::array<uint8_t, CARD_NUMBER_LENGTH> generate(CardNetwork network, const std::string& user_id);
};

class CVVGenerator {
public:
    static std::array<uint8_t, CVV_LENGTH> generate(
        const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number,
        uint16_t expiry_month,
        uint16_t expiry_year,
        const std::string& service_code);
};

// Repository Interface
class ICardRepository {
public:
    virtual ~ICardRepository() = default;
    virtual bool save_card(const CardData& card) = 0;
    virtual std::optional<CardData> get_card(const std::string& card_id) = 0;
    virtual std::optional<CardData> get_card_by_number(const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number) = 0;
    virtual std::vector<CardData> get_user_cards(const std::string& user_id) = 0;
    virtual bool update_card(const CardData& card) = 0;
    virtual bool delete_card(const std::string& card_id) = 0;
    virtual bool save_transaction(const Transaction& tx) = 0;
    virtual std::optional<Transaction> get_transaction(const std::string& transaction_id) = 0;
    virtual std::vector<Transaction> get_card_transactions(const std::string& card_id, 
                                                           const std::chrono::system_clock::time_point& start,
                                                           const std::chrono::system_clock::time_point& end) = 0;
    virtual bool update_card_limits(const std::string& card_id, const CardLimits& limits) = 0;
};

// In-Memory Repository
class InMemoryCardRepository : public ICardRepository {
private:
    std::map<std::string, CardData> cards_;
    std::map<std::string, Transaction> transactions_;
    std::map<std::string, std::string> card_number_hash_map_;
    std::mutex mutex_;
    
public:
    bool save_card(const CardData& card) override;
    std::optional<CardData> get_card(const std::string& card_id) override;
    std::optional<CardData> get_card_by_number(const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number) override;
    std::vector<CardData> get_user_cards(const std::string& user_id) override;
    bool update_card(const CardData& card) override;
    bool delete_card(const std::string& card_id) override;
    bool save_transaction(const Transaction& tx) override;
    std::optional<Transaction> get_transaction(const std::string& transaction_id) override;
    std::vector<Transaction> get_card_transactions(const std::string& card_id,
                                                    const std::chrono::system_clock::time_point& start,
                                                    const std::chrono::system_clock::time_point& end) override;
    bool update_card_limits(const std::string& card_id, const CardLimits& limits) override;
};

// Risk Engine
class RiskEngine {
private:
    std::unordered_map<std::string, uint32_t> failed_transactions_;
    std::unordered_map<std::string, std::vector<std::chrono::system_clock::time_point>> transaction_history_;
    std::mutex mutex_;
    
public:
    RiskAssessment assess_transaction(const Transaction& tx, const CardData& card);
    void record_failed_transaction(const std::string& card_id);
    void record_successful_transaction(const std::string& card_id);
};

// Crypto Payment Processor
class CryptoPaymentProcessor {
private:
    std::map<std::string, CryptoRate> rates_;
    std::mutex mutex_;
    
public:
    void update_rate(const std::string& from, const std::string& to, double rate);
    std::optional<int64_t> convert_to_crypto(int64_t fiat_amount,
                                               const std::string& fiat_currency,
                                               const std::string& crypto_currency);
    std::optional<double> get_rate(const std::string& from, const std::string& to);
};

// Main Card Processor
class CardProcessor {
private:
    std::unique_ptr<ICardRepository> repository_;
    RiskEngine risk_engine_;
    CryptoPaymentProcessor crypto_processor_;
    std::mutex mutex_;
    
public:
    CardProcessor();
    
    std::optional<CardData> create_card(const CardHolder& holder,
                                         CardType card_type,
                                         CardNetwork network,
                                         CurrencyCode currency);
    
    bool activate_card(const std::string& card_id);
    bool block_card(const std::string& card_id);
    bool freeze_card(const std::string& card_id);
    bool unfreeze_card(const std::string& card_id);
    bool cancel_card(const std::string& card_id);
    
    std::optional<Transaction> process_transaction(const Transaction& request);
    std::optional<CardData> get_card_details(const std::string& card_id);
    std::vector<CardData> get_user_cards(const std::string& user_id);
    std::vector<Transaction> get_card_transactions(const std::string& card_id, uint32_t days = 30);
    bool update_card_limits(const std::string& card_id, const CardLimits& limits);
    std::string get_masked_card_number(const std::string& card_id);
    
    void update_crypto_rate(const std::string& from, const std::string& to, double rate);
};

} // namespace card
} // namespace tiger

#endif // TIGER_CARD_PROCESSOR_H
