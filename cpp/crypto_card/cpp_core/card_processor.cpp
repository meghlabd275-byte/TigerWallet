/**
 * TigerWallet Crypto Card Processor - Implementation
 * High-performance C++ card processing engine for crypto card operations
 */

#include "card_processor.h"

namespace tiger {
namespace card {

// SecureRandom Implementation
uint8_t SecureRandom::next_byte() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 255);
    return static_cast<uint8_t>(dis(gen));
}

void SecureRandom::next_bytes(std::vector<uint8_t>& buffer) {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 255);
    
    for (auto& b : buffer) {
        b = static_cast<uint8_t>(dis(gen));
    }
}

uint32_t SecureRandom::next_uint32() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<uint32_t> dis(0, UINT32_MAX);
    return dis(gen);
}

// LuhnValidator Implementation
bool LuhnValidator::validate(const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number) {
    int sum = 0;
    bool alternate = false;
    
    for (int i = CARD_NUMBER_LENGTH - 1; i >= 0; i--) {
        int n = card_number[i];
        
        if (alternate) {
            n *= 2;
            if (n > 9) {
                n -= 9;
            }
        }
        
        sum += n;
        alternate = !alternate;
    }
    
    return (sum % 10) == 0;
}

uint8_t LuhnValidator::calculate_check_digit(const std::array<uint8_t, CARD_NUMBER_LENGTH - 1>& partial) {
    int sum = 0;
    bool alternate = true;
    
    for (int i = CARD_NUMBER_LENGTH - 2; i >= 0; i--) {
        int n = partial[i];
        
        if (alternate) {
            n *= 2;
            if (n > 9) {
                n -= 9;
            }
        }
        
        sum += n;
        alternate = !alternate;
    }
    
    return (10 - (sum % 10)) % 10;
}

// CardNumberGenerator Implementation
std::array<uint8_t, CARD_NUMBER_LENGTH> CardNumberGenerator::generate(CardNetwork network, 
                                                                      const std::string& user_id) {
    std::array<uint8_t, CARD_NUMBER_LENGTH> card_number = {};
    
    // Get IIN based on network
    std::array<uint8_t, 6> iin = {};
    switch (network) {
        case CardNetwork::VISA:
            iin = {4, 0, 0, 0, 0, 0};
            break;
        case CardNetwork::MASTERCARD:
            iin = {5, 1, 0, 0, 0, 0};
            break;
        case CardNetwork::AMEX:
            iin = {3, 4, 0, 0, 0, 0};
            break;
        case CardNetwork::UNIONPAY:
            iin = {6, 2, 0, 0, 0, 0};
            break;
        default:
            iin = {4, 0, 0, 0, 0, 0};
    }
    
    for (size_t i = 0; i < 6; i++) {
        card_number[i] = iin[i];
    }
    
    // Add user ID hash for uniqueness
    uint8_t hash[32] = {};
    for (size_t i = 0; i < user_id.size() && i < 32; i++) {
        hash[i % 32] ^= static_cast<uint8_t>(user_id[i]);
    }
    
    for (size_t i = 6; i < 15; i++) {
        card_number[i] = hash[i - 6];
    }
    
    // Calculate check digit
    std::array<uint8_t, CARD_NUMBER_LENGTH - 1> partial;
    for (size_t i = 0; i < CARD_NUMBER_LENGTH - 1; i++) {
        partial[i] = card_number[i];
    }
    
    card_number[15] = LuhnValidator::calculate_check_digit(partial);
    
    return card_number;
}

// CVVGenerator Implementation
std::array<uint8_t, CVV_LENGTH> CVVGenerator::generate(
    const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number,
    uint16_t expiry_month,
    uint16_t expiry_year,
    const std::string& service_code) {
    
    std::array<uint8_t, CVV_LENGTH> cvv = {};
    
    // Combine data for CVV generation
    std::vector<uint8_t> data;
    data.insert(data.end(), card_number.begin(), card_number.end());
    data.push_back(static_cast<uint8_t>(expiry_month));
    data.push_back(static_cast<uint8_t>(expiry_year % 100));
    data.insert(data.end(), service_code.begin(), service_code.end());
    
    // Simple hash for CVV (in production use proper cryptographic hash)
    uint32_t hash = 0;
    for (size_t i = 0; i < data.size(); i++) {
        hash = hash * 31 + data[i];
    }
    
    for (size_t i = 0; i < CVV_LENGTH; i++) {
        cvv[i] = (hash >> (i * 3)) % 10;
    }
    
    return cvv;
}

// InMemoryCardRepository Implementation
bool InMemoryCardRepository::save_card(const CardData& card) {
    std::lock_guard<std::mutex> lock(mutex_);
    cards_[card.card_id] = card;
    return true;
}

std::optional<CardData> InMemoryCardRepository::get_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = cards_.find(card_id);
    if (it != cards_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::optional<CardData> InMemoryCardRepository::get_card_by_number(const std::array<uint8_t, CARD_NUMBER_LENGTH>& card_number) {
    std::lock_guard<std::mutex> lock(mutex_);
    // In production, use proper hash lookup
    for (const auto& [id, card] : cards_) {
        bool match = true;
        for (size_t i = 0; i < CARD_NUMBER_LENGTH; i++) {
            if (card.card_number[i] != card_number[i]) {
                match = false;
                break;
            }
        }
        if (match) return card;
    }
    return std::nullopt;
}

std::vector<CardData> InMemoryCardRepository::get_user_cards(const std::string& user_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<CardData> result;
    
    for (const auto& [id, card] : cards_) {
        if (card.user_id == user_id && card.status != CardStatus::CANCELLED) {
            result.push_back(card);
        }
    }
    
    return result;
}

bool InMemoryCardRepository::update_card(const CardData& card) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (cards_.find(card.card_id) != cards_.end()) {
        cards_[card.card_id] = card;
        return true;
    }
    return false;
}

bool InMemoryCardRepository::delete_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    return cards_.erase(card_id) > 0;
}

bool InMemoryCardRepository::save_transaction(const Transaction& tx) {
    std::lock_guard<std::mutex> lock(mutex_);
    transactions_[tx.transaction_id] = tx;
    return true;
}

std::optional<Transaction> InMemoryCardRepository::get_transaction(const std::string& transaction_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = transactions_.find(transaction_id);
    if (it != transactions_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<Transaction> InMemoryCardRepository::get_card_transactions(const std::string& card_id,
                                                                      const std::chrono::system_clock::time_point& start,
                                                                      const std::chrono::system_clock::time_point& end) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<Transaction> result;
    
    for (const auto& [id, tx] : transactions_) {
        if (tx.card_id == card_id && 
            tx.timestamp >= start && 
            tx.timestamp <= end) {
            result.push_back(tx);
        }
    }
    
    return result;
}

bool InMemoryCardRepository::update_card_limits(const std::string& card_id, const CardLimits& limits) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = cards_.find(card_id);
    if (it != cards_.end()) {
        it->second.daily_limit = limits.daily_limit;
        it->second.monthly_limit = limits.monthly_limit;
        it->second.max_single_transaction = limits.max_single_transaction;
        it->second.min_single_transaction = limits.min_single_transaction;
        return true;
    }
    return false;
}

// RiskEngine Implementation
RiskAssessment RiskEngine::assess_transaction(const Transaction& tx, const CardData& card) {
    RiskAssessment assessment;
    assessment.score = 0;
    assessment.approved = true;
    
    // Check amount limits
    if (tx.amount > card.max_single_transaction) {
        assessment.flags.push_back("Exceeds maximum single transaction limit");
        assessment.score += 30;
    }
    
    if (tx.amount < card.min_single_transaction) {
        assessment.flags.push_back("Below minimum transaction amount");
        assessment.score += 20;
    }
    
    // Check daily limit
    auto now = std::chrono::system_clock::now();
    auto day_start = now - std::chrono::hours(24);
    
    auto& history = transaction_history_[tx.card_id];
    uint32_t daily_total = 0;
    
    for (const auto& tx_time : history) {
        if (tx_time >= day_start) {
            daily_total += tx.amount;
        }
    }
    
    if (daily_total + tx.amount > card.daily_limit) {
        assessment.flags.push_back("Exceeds daily spending limit");
        assessment.score += 25;
    }
    
    // Check for unusual location
    if (tx.location.empty() || tx.location == "UNKNOWN") {
        assessment.flags.push_back("Unknown transaction location");
        assessment.score += 15;
    }
    
    // Check for high risk merchant
    std::vector<std::string> high_risk_merchants = {
        "Gambling", "Casino", "Adult", "Weapons", "Dark Web"
    };
    
    for (const auto& risk_merchant : high_risk_merchants) {
        if (tx.merchant_name.find(risk_merchant) != std::string::npos ||
            tx.merchant_category.find(risk_merchant) != std::string::npos) {
            assessment.flags.push_back("High risk merchant category");
            assessment.score += 40;
            break;
        }
    }
    
    // Check for multiple failed transactions
    auto failed_count = failed_transactions_[tx.card_id];
    if (failed_count >= 3) {
        assessment.flags.push_back("Multiple failed transactions detected");
        assessment.score += 35;
    }
    
    // IP address checks
    if (tx.ip_address.empty()) {
        assessment.flags.push_back("Missing IP address");
        assessment.score += 10;
    }
    
    // Velocity check
    int recent_count = 0;
    auto hour_ago = now - std::chrono::hours(1);
    for (const auto& tx_time : history) {
        if (tx_time >= hour_ago) {
            recent_count++;
        }
    }
    
    if (recent_count >= 10) {
        assessment.flags.push_back("High transaction velocity");
        assessment.score += 25;
    }
    
    // Determine approval status
    if (assessment.score >= 50) {
        assessment.approved = false;
        assessment.review_status = "REJECTED";
        assessment.recommendation = "Transaction blocked due to high risk score";
    } else if (assessment.score >= 25) {
        assessment.approved = true;
        assessment.review_status = "REVIEW_REQUIRED";
        assessment.recommendation = "Manual review recommended";
    } else {
        assessment.approved = true;
        assessment.review_status = "APPROVED";
        assessment.recommendation = "Transaction approved";
    }
    
    return assessment;
}

void RiskEngine::record_failed_transaction(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    failed_transactions_[card_id]++;
}

void RiskEngine::record_successful_transaction(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    failed_transactions_[card_id] = 0;
    transaction_history_[card_id].push_back(std::chrono::system_clock::now());
    
    if (transaction_history_[card_id].size() > 1000) {
        transaction_history_[card_id].erase(
            transaction_history_[card_id].begin(),
            transaction_history_[card_id].end() - 1000
        );
    }
}

// CryptoPaymentProcessor Implementation
void CryptoPaymentProcessor::update_rate(const std::string& from, const std::string& to, double rate) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string key = from + "_" + to;
    rates_[key] = {from, to, rate, std::chrono::system_clock::now(), 0};
}

std::optional<int64_t> CryptoPaymentProcessor::convert_to_crypto(int64_t fiat_amount,
                                                                   const std::string& fiat_currency,
                                                                   const std::string& crypto_currency) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string key = fiat_currency + "_" + crypto_currency;
    auto it = rates_.find(key);
    
    if (it == rates_.end()) {
        key = crypto_currency + "_" + fiat_currency;
        it = rates_.find(key);
        
        if (it == rates_.end()) {
            return std::nullopt;
        }
        
        return static_cast<int64_t>(fiat_amount / it->second.rate * 100000000);
    }
    
    return static_cast<int64_t>(fiat_amount * it->second.rate * 100000000);
}

std::optional<double> CryptoPaymentProcessor::get_rate(const std::string& from, const std::string& to) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string key = from + "_" + to;
    auto it = rates_.find(key);
    
    if (it != rates_.end()) {
        return it->second.rate;
    }
    
    key = to + "_" + from;
    it = rates_.find(key);
    
    if (it != rates_.end()) {
        return 1.0 / it->second.rate;
    }
    
    return std::nullopt;
}

// CardProcessor Implementation
CardProcessor::CardProcessor() 
    : repository_(std::make_unique<InMemoryCardRepository>()) {
    
    // Initialize default crypto rates
    crypto_processor_.update_rate("USD", "BTC", 0.000025);
    crypto_processor_.update_rate("USD", "ETH", 0.0004);
    crypto_processor_.update_rate("USD", "USDT", 1.0);
    crypto_processor_.update_rate("USD", "EUR", 0.92);
    crypto_processor_.update_rate("USD", "GBP", 0.79);
}

std::optional<CardData> CardProcessor::create_card(const CardHolder& holder,
                                                     CardType card_type,
                                                     CardNetwork network,
                                                     CurrencyCode currency) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Check card limit
    auto existing_cards = repository_->get_user_cards(holder.user_id);
    if (existing_cards.size() >= MAX_CARDS_PER_USER) {
        return std::nullopt;
    }
    
    // Generate card number
    auto card_number = CardNumberGenerator::generate(network, holder.user_id);
    
    // Validate card number
    if (!LuhnValidator::validate(card_number)) {
        return std::nullopt;
    }
    
    // Generate CVV
    auto cvv = CVVGenerator::generate(card_number, 12, 2027, "000");
    
    // Create card data
    CardData card;
    card.card_id = "CARD_" + std::to_string(SecureRandom::next_uint32());
    card.user_id = holder.user_id;
    card.card_number = card_number;
    card.cvv = cvv;
    card.expiry_month = 12;
    card.expiry_year = 2029;
    card.card_type = card_type;
    card.status = CardStatus::PENDING;
    card.network = network;
    card.currency = currency;
    card.card_holder_name = holder.name;
    card.billing_address = holder.billing_address;
    card.daily_limit = 1000000;
    card.monthly_limit = 10000000;
    card.daily_spent = 0;
    card.monthly_spent = 0;
    card.max_single_transaction = 500000;
    card.min_single_transaction = 100;
    card.contactless_enabled = true;
    card.online_payments_enabled = true;
    card.international_enabled = true;
    card.created_at = std::chrono::system_clock::now();
    card.updated_at = std::chrono::system_clock::now();
    card.expires_at = std::chrono::system_clock::now() + std::chrono::hours(24 * 365 * 3);
    
    if (repository_->save_card(card)) {
        return card;
    }
    
    return std::nullopt;
}

bool CardProcessor::activate_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto card_opt = repository_->get_card(card_id);
    if (!card_opt) {
        return false;
    }
    
    auto card = *card_opt;
    
    if (card.status != CardStatus::PENDING) {
        return false;
    }
    
    card.status = CardStatus::ACTIVE;
    card.updated_at = std::chrono::system_clock::now();
    
    return repository_->update_card(card);
}

bool CardProcessor::block_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto card_opt = repository_->get_card(card_id);
    if (!card_opt) {
        return false;
    }
    
    auto card = *card_opt;
    card.status = CardStatus::BLOCKED;
    card.updated_at = std::chrono::system_clock::now();
    
    return repository_->update_card(card);
}

bool CardProcessor::freeze_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto card_opt = repository_->get_card(card_id);
    if (!card_opt) {
        return false;
    }
    
    auto card = *card_opt;
    card.status = CardStatus::FROZEN;
    card.updated_at = std::chrono::system_clock::now();
    
    return repository_->update_card(card);
}

bool CardProcessor::unfreeze_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto card_opt = repository_->get_card(card_id);
    if (!card_opt) {
        return false;
    }
    
    auto card = *card_opt;
    card.status = CardStatus::ACTIVE;
    card.updated_at = std::chrono::system_clock::now();
    
    return repository_->update_card(card);
}

bool CardProcessor::cancel_card(const std::string& card_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto card_opt = repository_->get_card(card_id);
    if (!card_opt) {
        return false;
    }
    
    auto card = *card_opt;
    card.status = CardStatus::CANCELLED;
    card.updated_at = std::chrono::system_clock::now();
    
    return repository_->update_card(card);
}

std::optional<Transaction> CardProcessor::process_transaction(const Transaction& request) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto card_opt = repository_->get_card(request.card_id);
    if (!card_opt) {
        return std::nullopt;
    }
    
    auto card = *card_opt;
    
    if (card.status != CardStatus::ACTIVE) {
        return std::nullopt;
    }
    
    auto risk = risk_engine_.assess_transaction(request, card);
    
    Transaction tx = request;
    tx.status = risk.approved ? TransactionStatus::PENDING : TransactionStatus::FLAGGED;
    tx.risk_score = risk.score;
    tx.risk_reason = risk.recommendation;
    
    if (!risk.approved) {
        risk_engine_.record_failed_transaction(card.card_id);
        repository_->save_transaction(tx);
        return tx;
    }
    
    if (!request.crypto_currency.empty()) {
        auto crypto_amount = crypto_processor_.convert_to_crypto(
            request.amount,
            "USD",
            request.crypto_currency
        );
        
        if (crypto_amount) {
            tx.crypto_amount = *crypto_amount;
        }
    }
    
    std::stringstream ss;
    ss << "AUTH" << std::setfill('0') << std::setw(8) 
       << SecureRandom::next_uint32() % 100000000;
    tx.authorization_code = ss.str();
    
    card.daily_spent += tx.amount;
    card.monthly_spent += tx.amount;
    card.updated_at = std::chrono::system_clock::now();
    repository_->update_card(card);
    
    risk_engine_.record_successful_transaction(card.card_id);
    
    tx.status = TransactionStatus::COMPLETED;
    tx.timestamp = std::chrono::system_clock::now();
    
    repository_->save_transaction(tx);
    
    return tx;
}

std::optional<CardData> CardProcessor::get_card_details(const std::string& card_id) {
    return repository_->get_card(card_id);
}

std::vector<CardData> CardProcessor::get_user_cards(const std::string& user_id) {
    return repository_->get_user_cards(user_id);
}

std::vector<Transaction> CardProcessor::get_card_transactions(const std::string& card_id, uint32_t days) {
    auto now = std::chrono::system_clock::now();
    auto start = now - std::chrono::hours(24 * days);
    
    return repository_->get_card_transactions(card_id, start, now);
}

bool CardProcessor::update_card_limits(const std::string& card_id, const CardLimits& limits) {
    return repository_->update_card_limits(card_id, limits);
}

std::string CardProcessor::get_masked_card_number(const std::string& card_id) {
    auto card_opt = repository_->get_card(card_id);
    if (!card_opt) {
        return "";
    }
    
    auto& card = *card_opt;
    std::stringstream ss;
    
    ss << static_cast<int>(card.card_number[0])
       << static_cast<int>(card.card_number[1])
       << static_cast<int>(card.card_number[2])
       << static_cast<int>(card.card_number[3])
       << " **** **** "
       << static_cast<int>(card.card_number[12])
       << static_cast<int>(card.card_number[13])
       << static_cast<int>(card.card_number[14])
       << static_cast<int>(card.card_number[15]);
    
    return ss.str();
}

void CardProcessor::update_crypto_rate(const std::string& from, const std::string& to, double rate) {
    crypto_processor_.update_rate(from, to, rate);
}

} // namespace card
} // namespace tiger
