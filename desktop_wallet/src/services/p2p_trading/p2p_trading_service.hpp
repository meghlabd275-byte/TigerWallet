#ifndef TIGERWALLET_P2P_TRADING_HPP
#define TIGERWALLET_P2P_TRADING_HPP

#include <string>
#include <vector>
#include <map>
#include <mutex>
#include <chrono>
#include <atomic>
#include <functional>
#include <sstream>
#include <iomanip>
#include <algorithm>
#include <random>

namespace tigerwallet {

// =============================================================================
// P2P TRADING TYPES
// =============================================================================

enum class P2POrderSide {
    BUY,
    SELL
};

enum class P2POrderStatus {
    PENDING,
    OPEN,
    FILLED,
    PARTIALLY_FILLED,
    CANCELLED,
    EXPIRED
};

enum class P2PTradeStatus {
    PENDING_PAYMENT,
    PAYMENT_RECEIVED,
    CRYPTO_RELEASED,
    DISPUTE_OPEN,
    COMPLETED,
    CANCELLED
};

enum class PaymentMethod {
    BANK_TRANSFER,
    UPI,
    PAYTM,
    GOOGLE_PAY,
    PHONEPE,
    CASH,
    CRYPTO
};

struct P2PToken {
    std::string symbol;
    std::string name;
    std::string contract_address;
    std::string chain;
    double min_amount;
    double max_amount;
};

struct P2PUser {
    std::string address;
    std::string username;
    double rating;
    uint32_t trade_count;
    uint32_t successful_trades;
    bool is_verified;
    bool is_merchant;
    std::vector<std::string> trusted_users;
    std::vector<std::string> blocked_users;
};

struct P2POrder {
    std::string order_id;
    std::string maker;
    P2POrderSide side;
    P2PToken token;
    double amount;
    double price;
    double filled_amount;
    double remaining_amount;
    std::vector<PaymentMethod> payment_methods;
    P2POrderStatus status;
    std::string fiat_currency;
    double min_limit;
    double max_limit;
    uint64_t created_at;
    uint64_t expires_at;
    uint64_t updated_at;
    std::string terms;
    bool is_merchant_order;
    uint32_t rate_limit;
    
    P2POrder() : side(P2POrderSide::BUY), amount(0), price(0), filled_amount(0),
                 remaining_amount(0), status(P2POrderStatus::PENDING),
                 min_limit(0), max_limit(0), created_at(0), expires_at(0),
                 updated_at(0), is_merchant_order(false), rate_limit(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"orderId\":\"" << order_id << "\",";
        oss << "\"side\":\"" << (side == P2POrderSide::BUY ? "BUY" : "SELL") << "\",";
        oss << "\"token\":\"" << token.symbol << "\",";
        oss << "\"amount\":" << std::fixed << std::setprecision(8) << amount << ",";
        oss << "\"price\":" << price << ",";
        oss << "\"filledAmount\":" << filled_amount << ",";
        oss << "\"status\":\"" << static_cast<int>(status) << "\",";
        oss << "\"fiatCurrency\":\"" << fiat_currency << "\"";
        oss << "}";
        return oss.str();
    }
};

struct P2PTrade {
    std::string trade_id;
    std::string order_id;
    std::string maker;
    std::string taker;
    P2PToken token;
    double amount;
    double price;
    double total_value;
    PaymentMethod payment_method;
    P2PTradeStatus status;
    std::string fiat_transaction_id;
    uint64_t created_at;
    uint64_t payment_deadline;
    uint64_t released_at;
    std::string dispute_reason;
    std::vector<std::string> evidence;
    
    P2PTrade() : amount(0), price(0), total_value(0), status(P2PTradeStatus::PENDING_PAYMENT),
                 created_at(0), payment_deadline(0), released_at(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"tradeId\":\"" << trade_id << "\",";
        oss << "\"orderId\":\"" << order_id << "\",";
        oss << "\"maker\":\"" << maker << "\",";
        oss << "\"taker\":\"" << taker << "\",";
        oss << "\"amount\":" << std::fixed << std::setprecision(8) << amount << ",";
        oss << "\"price\":" << price << ",";
        oss << "\"totalValue\":" << total_value << ",";
        oss << "\"status\":\"" << static_cast<int>(status) << "\"";
        oss << "}";
        return oss.str();
    }
};

struct P2PMerchant {
    std::string address;
    std::string business_name;
    std::string description;
    std::vector<std::string> supported_fiat;
    std::vector<PaymentMethod> supported_payments;
    double trading_limit_daily;
    double trading_limit_monthly;
    double volume_30d;
    uint32_t total_trades;
    double success_rate;
    std::string verification_level;
    std::vector<std::string> badges;
    
    P2PMerchant() : trading_limit_daily(0), trading_limit_monthly(0), volume_30d(0),
                    total_trades(0), success_rate(0) {}
};

// =============================================================================
// P2P TRADING SERVICE IMPLEMENTATION
// =============================================================================

class P2PTradingService {
private:
    std::map<std::string, P2POrder> orders;
    std::map<std::string, P2PTrade> trades;
    std::map<std::string, P2PUser> users;
    std::map<std::string, P2PMerchant> merchants;
    std::map<std::string, std::vector<std::string>> user_orders;
    std::map<std::string, std::vector<std::string>> user_trades;
    
    std::mutex orders_mutex;
    std::mutex trades_mutex;
    std::mutex users_mutex;
    
    std::atomic<uint64_t> order_counter{1};
    std::atomic<uint64_t> trade_counter{1};
    
    bool initialized;
    
    std::vector<P2PToken> supported_tokens;
    
    uint64_t getCurrentTimestamp() const {
        return std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
    
    std::string generateOrderId() {
        std::ostringstream oss;
        oss << "P2P-ORD-" << order_counter.fetch_add(1) << "-" << getCurrentTimestamp();
        return oss.str();
    }
    
    std::string generateTradeId() {
        std::ostringstream oss;
        oss << "P2P-TRADE-" << trade_counter.fetch_add(1) << "-" << getCurrentTimestamp();
        return oss.str();
    }
    
    void initializeDefaultTokens() {
        supported_tokens = {
            {"ETH", "Ethereum", "0x0000000000000000000000000000000000000000", "ethereum", 0.001, 100.0},
            {"BTC", "Bitcoin", "0x0000000000000000000000000000000000000000", "bitcoin", 0.0001, 10.0},
            {"USDT", "Tether", "0xdAC17F958D2ee523a2206206994597C13D831ec7", "ethereum", 10.0, 100000.0},
            {"USDC", "USD Coin", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "ethereum", 10.0, 100000.0},
            {"MATIC", "Polygon", "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0", "polygon", 10.0, 50000.0},
            {"BNB", "BNB", "0xB8c77482e45F1F44dE1745F52C74426C631bDD52", "bsc", 0.01, 1000.0}
        };
    }
    
public:
    P2PTradingService() : initialized(false) {}
    
    ~P2PTradingService() {}
    
    bool initialize() {
        if (initialized) return true;
        
        std::lock_guard<std::mutex> lock(orders_mutex);
        initializeDefaultTokens();
        
        initialized = true;
        return true;
    }
    
    // Get supported tokens
    std::vector<P2PToken> getSupportedTokens() const {
        return supported_tokens;
    }
    
    // Create buy order
    P2POrder createBuyOrder(const std::string& user_address,
                           const std::string& token_symbol,
                           double amount,
                           double price,
                           const std::string& fiat_currency,
                           std::vector<PaymentMethod> payment_methods,
                           double min_limit,
                           double max_limit) {
        P2POrder order;
        order.order_id = generateOrderId();
        order.maker = user_address;
        order.side = P2POrderSide::BUY;
        
        // Find token
        for (const auto& t : supported_tokens) {
            if (t.symbol == token_symbol) {
                order.token = t;
                break;
            }
        }
        
        order.amount = amount;
        order.price = price;
        order.remaining_amount = amount;
        order.fiat_currency = fiat_currency;
        order.payment_methods = payment_methods;
        order.min_limit = min_limit;
        order.max_limit = max_limit;
        order.status = P2POrderStatus::OPEN;
        order.created_at = getCurrentTimestamp();
        order.expires_at = order.created_at + (24 * 60 * 60); // 24 hours
        order.updated_at = order.created_at;
        
        std::lock_guard<std::mutex> lock(orders_mutex);
        orders[order.order_id] = order;
        user_orders[user_address].push_back(order.order_id);
        
        return order;
    }
    
    // Create sell order
    P2POrder createSellOrder(const std::string& user_address,
                            const std::string& token_symbol,
                            double amount,
                            double price,
                            const std::string& fiat_currency,
                            std::vector<PaymentMethod> payment_methods,
                            double min_limit,
                            double max_limit) {
        P2POrder order;
        order.order_id = generateOrderId();
        order.maker = user_address;
        order.side = P2POrderSide::SELL;
        
        for (const auto& t : supported_tokens) {
            if (t.symbol == token_symbol) {
                order.token = t;
                break;
            }
        }
        
        order.amount = amount;
        order.price = price;
        order.remaining_amount = amount;
        order.fiat_currency = fiat_currency;
        order.payment_methods = payment_methods;
        order.min_limit = min_limit;
        order.max_limit = max_limit;
        order.status = P2POrderStatus::OPEN;
        order.created_at = getCurrentTimestamp();
        order.expires_at = order.created_at + (24 * 60 * 60);
        order.updated_at = order.created_at;
        
        std::lock_guard<std::mutex> lock(orders_mutex);
        orders[order.order_id] = order;
        user_orders[user_address].push_back(order.order_id);
        
        return order;
    }
    
    // Get open orders with filters
    std::vector<P2POrder> getOpenOrders(const std::string& token_symbol,
                                        const std::string& fiat_currency,
                                        P2POrderSide side,
                                        PaymentMethod payment_method) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        std::vector<P2POrder> result;
        
        for (const auto& pair : orders) {
            const P2POrder& order = pair.second;
            
            if (order.status != P2POrderStatus::OPEN) continue;
            if (order.expires_at < getCurrentTimestamp()) continue;
            
            if (!token_symbol.empty() && order.token.symbol != token_symbol) continue;
            if (!fiat_currency.empty() && order.fiat_currency != fiat_currency) continue;
            if (side != P2POrderSide::BUY && side != P2POrderSide::SELL && order.side != side) continue;
            
            bool has_payment = false;
            for (auto pm : order.payment_methods) {
                if (pm == payment_method) {
                    has_payment = true;
                    break;
                }
            }
            if (payment_method != PaymentMethod::CRYPTO && !has_payment) continue;
            
            result.push_back(order);
        }
        
        // Sort by price (best price first)
        if (side == P2POrderSide::BUY) {
            std::sort(result.begin(), result.end(), 
                [](const P2POrder& a, const P2POrder& b) { return a.price > b.price; });
        } else {
            std::sort(result.begin(), result.end(), 
                [](const P2POrder& a, const P2POrder& b) { return a.price < b.price; });
        }
        
        return result;
    }
    
    // Take order (create trade)
    P2PTrade takeOrder(const std::string& order_id,
                       const std::string& taker_address,
                       double amount) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        
        auto it = orders.find(order_id);
        if (it == orders.end() || it->second.status != P2POrderStatus::OPEN) {
            return P2PTrade();
        }
        
        P2POrder& order = it->second;
        
        if (amount < order.min_limit || amount > order.max_limit) {
            return P2PTrade();
        }
        
        if (amount > order.remaining_amount) {
            amount = order.remaining_amount;
        }
        
        // Create trade
        P2PTrade trade;
        trade.trade_id = generateTradeId();
        trade.order_id = order_id;
        trade.maker = order.maker;
        trade.taker = taker_address;
        trade.token = order.token;
        trade.amount = amount;
        trade.price = order.price;
        trade.total_value = amount * order.price;
        trade.payment_method = order.payment_methods[0];
        trade.status = P2PTradeStatus::PENDING_PAYMENT;
        trade.created_at = getCurrentTimestamp();
        trade.payment_deadline = trade.created_at + (30 * 60); // 30 minutes
        
        // Update order
        order.filled_amount += amount;
        order.remaining_amount = order.amount - order.filled_amount;
        
        if (order.remaining_amount <= 0) {
            order.status = P2POrderStatus::FILLED;
        } else {
            order.status = P2POrderStatus::PARTIALLY_FILLED;
        }
        
        order.updated_at = getCurrentTimestamp();
        
        std::lock_guard<std::mutex> trade_lock(trades_mutex);
        trades[trade.trade_id] = trade;
        user_trades[taker_address].push_back(trade.trade_id);
        
        return trade;
    }
    
    // Confirm payment
    bool confirmPayment(const std::string& trade_id, const std::string& fiat_transaction_id) {
        std::lock_guard<std::mutex> lock(trades_mutex);
        
        auto it = trades.find(trade_id);
        if (it == trades.end()) return false;
        
        P2PTrade& trade = it->second;
        if (trade.status != P2PTradeStatus::PENDING_PAYMENT) return false;
        
        trade.status = P2PTradeStatus::PAYMENT_RECEIVED;
        trade.fiat_transaction_id = fiat_transaction_id;
        
        return true;
    }
    
    // Release crypto
    bool releaseCrypto(const std::string& trade_id) {
        std::lock_guard<std::mutex> lock(trades_mutex);
        
        auto it = trades.find(trade_id);
        if (it == trades.end()) return false;
        
        P2PTrade& trade = it->second;
        if (trade.status != P2PTradeStatus::PAYMENT_RECEIVED) return false;
        
        trade.status = P2PTradeStatus::CRYPTO_RELEASED;
        trade.released_at = getCurrentTimestamp();
        
        // Update order status
        std::lock_guard<std::mutex> order_lock(orders_mutex);
        auto order_it = orders.find(trade.order_id);
        if (order_it != orders.end()) {
            if (order_it->second.remaining_amount <= 0) {
                order_it->second.status = P2POrderStatus::FILLED;
            }
        }
        
        return true;
    }
    
    // Cancel order
    bool cancelOrder(const std::string& order_id, const std::string& user_address) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        
        auto it = orders.find(order_id);
        if (it == orders.end()) return false;
        
        P2POrder& order = it->second;
        if (order.maker != user_address) return false;
        if (order.status != P2POrderStatus::OPEN) return false;
        
        order.status = P2POrderStatus::CANCELLED;
        order.updated_at = getCurrentTimestamp();
        
        return true;
    }
    
    // Open dispute
    bool openDispute(const std::string& trade_id, const std::string& user_address, 
                    const std::string& reason) {
        std::lock_guard<std::mutex> lock(trades_mutex);
        
        auto it = trades.find(trade_id);
        if (it == trades.end()) return false;
        
        P2PTrade& trade = it->second;
        if (trade.maker != user_address && trade.taker != user_address) return false;
        
        trade.status = P2PTradeStatus::DISPUTE_OPEN;
        trade.dispute_reason = reason;
        
        return true;
    }
    
    // Get user's orders
    std::vector<P2POrder> getUserOrders(const std::string& user_address) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        std::vector<P2POrder> result;
        
        auto it = user_orders.find(user_address);
        if (it != user_orders.end()) {
            for (const auto& order_id : it->second) {
                auto order_it = orders.find(order_id);
                if (order_it != orders.end()) {
                    result.push_back(order_it->second);
                }
            }
        }
        
        return result;
    }
    
    // Get user's trades
    std::vector<P2PTrade> getUserTrades(const std::string& user_address) {
        std::lock_guard<std::mutex> lock(trades_mutex);
        std::vector<P2PTrade> result;
        
        auto it = user_trades.find(user_address);
        if (it != user_trades.end()) {
            for (const auto& trade_id : it->second) {
                auto trade_it = trades.find(trade_id);
                if (trade_it != trades.end()) {
                    result.push_back(trade_it->second);
                }
            }
        }
        
        return result;
    }
    
    // Get trade details
    P2PTrade getTrade(const std::string& trade_id) {
        std::lock_guard<std::mutex> lock(trades_mutex);
        auto it = trades.find(trade_id);
        if (it != trades.end()) {
            return it->second;
        }
        return P2PTrade();
    }
    
    // Get order book
    std::map<double, double> getOrderBook(const std::string& token_symbol,
                                          P2POrderSide side) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        std::map<double, double> order_book;
        
        for (const auto& pair : orders) {
            const P2POrder& order = pair.second;
            
            if (order.status != P2POrderStatus::OPEN) continue;
            if (order.token.symbol != token_symbol) continue;
            if (order.side != side) continue;
            if (order.expires_at < getCurrentTimestamp()) continue;
            
            order_book[order.price] += order.remaining_amount;
        }
        
        return order_book;
    }
    
    // Become merchant
    P2PMerchant becomeMerchant(const std::string& user_address,
                              const std::string& business_name,
                              const std::string& description,
                              std::vector<std::string> supported_fiat,
                              std::vector<PaymentMethod> supported_payments,
                              double daily_limit,
                              double monthly_limit) {
        P2PMerchant merchant;
        merchant.address = user_address;
        merchant.business_name = business_name;
        merchant.description = description;
        merchant.supported_fiat = supported_fiat;
        merchant.supported_payments = supported_payments;
        merchant.trading_limit_daily = daily_limit;
        merchant.trading_limit_monthly = monthly_limit;
        
        std::lock_guard<std::mutex> lock(users_mutex);
        merchants[user_address] = merchant;
        
        return merchant;
    }
    
    // Get merchant
    P2PUser getUserProfile(const std::string& address) {
        std::lock_guard<std::mutex> lock(users_mutex);
        auto it = users.find(address);
        if (it != users.end()) {
            return it->second;
        }
        return P2PUser();
    }
};

} // namespace tigerwallet

#endif // TIGERWALLET_P2P_TRADING_HPP
