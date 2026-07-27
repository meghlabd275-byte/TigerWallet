/**
 * TigerWallet High-Frequency Trading Engine
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Order Matching Engine
 * - Market Making
 * - Arbitrage Detection
 * - Smart Order Routing
 * - Risk Management
 * - Real-time Analytics
 */

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <set>
#include <queue>
#include <thread>
#include <mutex>
#include <atomic>
#include <chrono>
#include <memory>
#include <optional>
#include <functional>
#include <algorithm>
#include <cstdint>
#include <sstream>
#include <iomanip>
#include <random>

// ============================================================================
// Configuration
// ============================================================================

constexpr int HF_PORT = 9103;
constexpr size_t MAX_ORDERS = 1000000;
constexpr size_t MAX_TRADES = 10000000;
constexpr auto TICK_INTERVAL = std::chrono::microseconds(100);  // 10k Hz
constexpr auto HEARTBEAT_MS = 100;

// ============================================================================
// Types
// ============================================================================

using OrderID = uint64_t;
using UserID = uint64_t;
using Price = double;
using Quantity = double;
using Timestamp = uint64_t;

enum class OrderSide : uint8_t { BUY, SELL };
enum class OrderType : uint8_t { MARKET, LIMIT, STOP, STOP_LIMIT };
enum class OrderStatus : uint8_t { PENDING, OPEN, PARTIALLY_FILLED, FILLED, CANCELLED, REJECTED };
enum class TimeInForce : uint8_t { GTC, IOC, FOK, GTD };

struct Order {
    OrderID order_id;
    UserID user_id;
    std::string symbol;
    OrderSide side;
    OrderType type;
    OrderStatus status;
    Price price;
    Quantity quantity;
    Quantity filled_quantity;
    Price stop_price;
    TimeInForce tif;
    Timestamp created_at;
    Timestamp updated_at;
    Timestamp expires_at;
};

struct Trade {
    OrderID order_id;
    OrderID match_order_id;
    UserID buyer_id;
    UserID seller_id;
    std::string symbol;
    Price price;
    Quantity quantity;
    Timestamp timestamp;
    uint64_t trade_id;
};

struct OrderBookLevel {
    Price price;
    Quantity quantity;
    OrderID order_id;
};

struct MarketData {
    std::string symbol;
    Price last_price;
    Price bid_price;
    Price ask_price;
    Quantity bid_quantity;
    Quantity ask_quantity;
    Price high_24h;
    Price low_24h;
    Price volume_24h;
    Price turnover_24h;
    Timestamp timestamp;
};

// ============================================================================
// Order Book (Priority Queue Based)
// ============================================================================

class OrderBook {
private:
    std::priority_queue<std::pair<Price, OrderID>> buy_orders_;
    std::priority_queue<std::pair<Price, OrderID>, std::vector<std::pair<Price, OrderID>>, std::greater<>> sell_orders_;
    
    std::unordered_map<OrderID, Order> orders_;
    std::unordered_map<OrderID, Price> order_prices_;
    std::unordered_map<std::string, std::vector<Order>> symbol_orders_;
    
    mutable std::mutex mutex_;
    std::string symbol_;

public:
    OrderBook(const std::string& symbol) : symbol_(symbol) {}
    
    OrderID addOrder(const Order& order) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        orders_[order.order_id] = order;
        symbol_orders_[symbol_].push_back(order);
        
        if (order.status == OrderStatus::OPEN) {
            if (order.side == OrderSide::BUY) {
                buy_orders_.push({order.price, order.order_id});
            } else {
                sell_orders_.push({order.price, order.order_id});
            }
            order_prices_[order.order_id] = order.price;
        }
        
        return order.order_id;
    }
    
    bool cancelOrder(OrderID order_id) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = orders_.find(order_id);
        if (it == orders_.end()) return false;
        
        it->second.status = OrderStatus::CANCELLED;
        order_prices_.erase(order_id);
        
        return true;
    }
    
    std::optional<Order> getOrder(OrderID order_id) const {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = orders_.find(order_id);
        if (it != orders_.end()) {
            return it->second;
        }
        return std::nullopt;
    }
    
    std::vector<Trade> matchOrders() {
        std::lock_guard<std::mutex> lock(mutex_);
        
        std::vector<Trade> trades;
        static uint64_t trade_counter = 0;
        
        while (!buy_orders_.empty() && !sell_orders_.empty()) {
            auto [buy_price, buy_order_id] = buy_orders_.top();
            auto [sell_price, sell_order_id] = sell_orders_.top();
            
            if (buy_price < sell_price) break;
            
            auto& buy_order = orders_[buy_order_id];
            auto& sell_order = orders_[sell_order_id];
            
            if (buy_order.status != OrderStatus::OPEN || sell_order.status != OrderStatus::OPEN) {
                if (buy_order.status != OrderStatus::OPEN) buy_orders_.pop();
                if (sell_order.status != OrderStatus::OPEN) sell_orders_.pop();
                continue;
            }
            
            Quantity match_qty = std::min(
                buy_order.quantity - buy_order.filled_quantity,
                sell_order.quantity - sell_order.filled_quantity
            );
            
            Price match_price = sell_price;
            
            Trade trade;
            trade.order_id = buy_order_id;
            trade.match_order_id = sell_order_id;
            trade.buyer_id = buy_order.user_id;
            trade.seller_id = sell_order.user_id;
            trade.symbol = symbol_;
            trade.price = match_price;
            trade.quantity = match_qty;
            trade.timestamp = getCurrentTimestamp();
            trade.trade_id = ++trade_counter;
            
            trades.push_back(trade);
            
            buy_order.filled_quantity += match_qty;
            sell_order.filled_quantity += match_qty;
            
            if (buy_order.filled_quantity >= buy_order.quantity) {
                buy_order.status = OrderStatus::FILLED;
                buy_orders_.pop();
            } else {
                buy_order.status = OrderStatus::PARTIALLY_FILLED;
            }
            
            if (sell_order.filled_quantity >= sell_order.quantity) {
                sell_order.status = OrderStatus::FILLED;
                sell_orders_.pop();
            } else {
                sell_order.status = OrderStatus::PARTIALLY_FILLED;
            }
            
            orders_[buy_order_id] = buy_order;
            orders_[sell_order_id] = sell_order;
        }
        
        return trades;
    }
    
    MarketData getMarketData() const {
        std::lock_guard<std::mutex> lock(mutex_);
        
        MarketData data;
        data.symbol = symbol_;
        data.timestamp = getCurrentTimestamp();
        
        if (!buy_orders_.empty()) {
            data.bid_price = buy_orders_.top().first;
        }
        if (!sell_orders_.empty()) {
            data.ask_price = sell_orders_.top().first;
        }
        
        return data;
    }

private:
    static Timestamp getCurrentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// ============================================================================
// Risk Management
// ============================================================================

class RiskManager {
private:
    struct UserRisk {
        double max_position_size;
        double max_order_value;
        double max_daily_volume;
        double current_position;
        double daily_volume;
        Timestamp last_reset;
    };
    
    std::unordered_map<UserID, UserRisk> user_risks_;
    mutable std::mutex mutex_;

public:
    bool checkOrderRisk(const Order& order) const {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = user_risks_.find(order.user_id);
        if (it == user_risks_.end()) return true;
        
        const auto& risk = it->second;
        
        if (order.side == OrderSide::BUY) {
            if (order.quantity > risk.max_position_size) return false;
            if (order.quantity * order.price > risk.max_order_value) return false;
            if (risk.daily_volume + order.quantity * order.price > risk.max_daily_volume) return false;
        }
        
        return true;
    }
    
    void updatePosition(UserID user_id, Quantity quantity, Price price) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto& risk = user_risks_[user_id];
        risk.current_position += quantity;
        risk.daily_volume += quantity * price;
    }
    
    void resetDailyLimits(UserID user_id) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = user_risks_.find(user_id);
        if (it != user_risks_.end()) {
            it->second.daily_volume = 0;
            it->second.last_reset = getCurrentTimestamp();
        }
    }
    
private:
    static Timestamp getCurrentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// ============================================================================
// Market Maker
// ============================================================================

class MarketMaker {
private:
    struct MMConfig {
        std::string symbol;
        Price base_price;
        Price spread;
        Quantity min_quantity;
        Quantity max_quantity;
        int order_refresh_ms;
        bool enabled;
    };
    
    struct MMOrder {
        OrderID order_id;
        OrderSide side;
        Price price;
        Quantity quantity;
    };
    
    MMConfig config_;
    std::vector<MMOrder> active_orders_;
    std::atomic<bool> running_{false};
    std::thread worker_thread_;
    std::function<Order(const Order&)> order_callback_;
    std::function<OrderID(OrderID)> cancel_callback_;
    mutable std::mutex mutex_;

public:
    MarketMaker(const MMConfig& config) : config_(config) {}
    
    void setCallbacks(
        std::function<Order(const Order&)> order_cb,
        std::function<OrderID(OrderID)> cancel_cb
    ) {
        order_callback_ = order_cb;
        cancel_callback_ = cancel_cb;
    }
    
    void start() {
        running_ = true;
        worker_thread_ = std::thread([this]() {
            while (running_) {
                refreshOrders();
                std::this_thread::sleep_for(std::chrono::milliseconds(config_.order_refresh_ms));
            }
        });
    }
    
    void stop() {
        running_ = false;
        if (worker_thread_.joinable()) {
            worker_thread_.join();
        }
    }
    
    void setBasePrice(Price price) {
        config_.base_price = price;
    }

private:
    void refreshOrders() {
        std::lock_guard<std::mutex> lock(mutex_);
        
        // Cancel existing orders
        for (const auto& order : active_orders_) {
            if (cancel_callback_) {
                cancel_callback_(order.order_id);
            }
        }
        active_orders_.clear();
        
        if (!config_.enabled || !order_callback_) return;
        
        // Create new spread orders
        Price spread = config_.spread;
        
        // Bid order
        Order bid_order;
        bid_order.order_id = generateOrderID();
        bid_order.price = config_.base_price - spread;
        bid_order.quantity = config_.min_quantity;
        bid_order.side = OrderSide::BUY;
        
        // Ask order
        Order ask_order;
        ask_order.order_id = generateOrderID();
        ask_order.price = config_.base_price + spread;
        ask_order.quantity = config_.min_quantity;
        ask_order.side = OrderSide::SELL;
        
        if (order_callback_) {
            active_orders_.push_back({bid_order.order_id, OrderSide::BUY, bid_order.price, bid_order.quantity});
            active_orders_.push_back({ask_order.order_id, OrderSide::SELL, ask_order.price, ask_order.quantity});
        }
    }
    
    static OrderID generateOrderID() {
        static std::atomic<uint64_t> counter{0};
        return ++counter;
    }
};

// ============================================================================
// Arbitrage Detector
// ============================================================================

class ArbitrageDetector {
private:
    struct PriceSource {
        std::string name;
        Price bid_price;
        Price ask_price;
        Timestamp last_update;
    };
    
    std::unordered_map<std::string, std::vector<PriceSource>> prices_;
    double min_profit_threshold_ = 0.001;  // 0.1%
    mutable std::mutex mutex_;

public:
    void addPriceSource(const std::string& symbol, const std::string& source, Price bid, Price ask) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        prices_[symbol].push_back({
            source,
            bid,
            ask,
            getCurrentTimestamp()
        });
    }
    
    struct ArbitrageOpportunity {
        std::string symbol;
        std::string buy_source;
        std::string sell_source;
        Price buy_price;
        Price sell_price;
        Quantity max_quantity;
        double profit_percentage;
    };
    
    std::vector<ArbitrageOpportunity> findOpportunities(const std::string& symbol) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        std::vector<ArbitrageOpportunity> opportunities;
        
        auto it = prices_.find(symbol);
        if (it == prices_.end() || it->second.size() < 2) return opportunities;
        
        const auto& sources = it->second;
        
        for (size_t i = 0; i < sources.size(); i++) {
            for (size_t j = 0; j < sources.size(); j++) {
                if (i == j) continue;
                
                Price buy_price = sources[i].ask_price;
                Price sell_price = sources[j].bid_price;
                
                double profit_pct = (sell_price - buy_price) / buy_price;
                
                if (profit_pct > min_profit_threshold_) {
                    opportunities.push_back({
                        symbol,
                        sources[i].name,
                        sources[j].name,
                        buy_price,
                        sell_price,
                        1000000.0,  // Max quantity
                        profit_pct
                    });
                }
            }
        }
        
        return opportunities;
    }
    
    void setMinProfitThreshold(double threshold) {
        min_profit_threshold_ = threshold;
    }

private:
    static Timestamp getCurrentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// ============================================================================
// Smart Order Router
// ============================================================================

class SmartOrderRouter {
private:
    struct RouterConfig {
        std::vector<std::string> venues;
        bool enable_sor;
        bool enable_split;
        double max_slippage;
    };
    
    struct Venue {
        std::string name;
        Price bid_price;
        Price ask_price;
        Quantity available_quantity;
        double fee;
        double latency_ms;
    };
    
    RouterConfig config_;
    std::unordered_map<std::string, std::vector<Venue>> venue_data_;

public:
    void setConfig(const RouterConfig& config) {
        config_ = config;
    }
    
    struct SORResult {
        std::string symbol;
        Price avg_price;
        Quantity total_filled;
        double total_fee;
        std::vector<std::pair<std::string, Quantity>> splits;
    };
    
    SORResult routeOrder(const Order& order) {
        SORResult result;
        result.symbol = order.symbol;
        
        if (!config_.enable_sor || order.type != OrderType::MARKET) {
            // Direct routing
            auto it = venue_data_.find(order.symbol);
            if (it != venue_data_.end() && !it->second.empty()) {
                const auto& venue = it->second[0];
                result.avg_price = venue.ask_price;
                result.total_filled = std::min(order.quantity, venue.available_quantity);
                result.total_fee = result.total_filled * venue.fee;
                result.splits.push_back({venue.name, result.total_filled});
            }
            return result;
        }
        
        // Smart Order Routing - split across venues
        auto it = venue_data_.find(order.symbol);
        if (it == venue_data_.end()) return result;
        
        const auto& venues = it->second;
        Quantity remaining = order.quantity;
        Price total_cost = 0;
        
        for (const auto& venue : venues) {
            if (remaining <= 0) break;
            
            Quantity fill = std::min(remaining, venue.available_quantity);
            double cost = fill * venue.ask_price;
            
            result.splits.push_back({venue.name, fill});
            result.total_fee += fill * venue.fee;
            total_cost += cost;
            remaining -= fill;
        }
        
        result.total_filled = order.quantity - remaining;
        result.avg_price = result.total_filled > 0 ? total_cost / result.total_filled : 0;
        
        return result;
    }
    
    void updateVenueData(const std::string& symbol, const std::vector<Venue>& venues) {
        venue_data_[symbol] = venues;
    }
};

// ============================================================================
// High-Frequency Trading Engine
// ============================================================================

class HFTEngine {
private:
    std::unordered_map<std::string, std::shared_ptr<OrderBook>> order_books_;
    RiskManager risk_manager_;
    MarketMaker market_maker_;
    ArbitrageDetector arb_detector_;
    SmartOrderRouter sor_;
    
    std::atomic<bool> running_{false};
    std::thread matching_thread_;
    std::thread analytics_thread_;
    
    std::atomic<uint64_t> total_orders_{0};
    std::atomic<uint64_t> total_trades_{0};
    std::atomic<double> total_volume_{0};

public:
    HFTEngine() : market_maker_({"BTC/USDT", 50000.0, 10.0, 0.01, 1.0, 1000, false}) {
        // Initialize order books
        order_books_["BTC/USDT"] = std::make_shared<OrderBook>("BTC/USDT");
        order_books_["ETH/USDT"] = std::make_shared<OrderBook>("ETH/USDT");
        order_books_["SOL/USDT"] = std::make_shared<OrderBook>("SOL/USDT");
    }
    
    void start() {
        running_ = true;
        
        // Start matching engine
        matching_thread_ = std::thread([this]() {
            while (running_) {
                for (auto& [symbol, book] : order_books_) {
                    auto trades = book->matchOrders();
                    for (const auto& trade : trades) {
                        processTrade(trade);
                    }
                }
                std::this_thread::sleep_for(TICK_INTERVAL);
            }
        });
        
        // Start analytics
        analytics_thread_ = std::thread([this]() {
            while (running_) {
                updateAnalytics();
                std::this_thread::sleep_for(std::chrono::seconds(1));
            }
        });
        
        std::cout << "HFT Engine started" << std::endl;
    }
    
    void stop() {
        running_ = false;
        
        if (matching_thread_.joinable()) matching_thread_.join();
        if (analytics_thread_.joinable()) analytics_thread_.join();
        
        std::cout << "HFT Engine stopped" << std::endl;
    }
    
    OrderID submitOrder(const Order& order) {
        // Check risk
        if (!risk_manager_.checkOrderRisk(order)) {
            std::cerr << "Order rejected: risk check failed" << std::endl;
            return 0;
        }
        
        auto it = order_books_.find(order.symbol);
        if (it == order_books_.end()) {
            std::cerr << "Unknown symbol: " << order.symbol << std::endl;
            return 0;
        }
        
        OrderID order_id = it->second->addOrder(order);
        total_orders_.fetch_add(1);
        
        return order_id;
    }
    
    bool cancelOrder(const std::string& symbol, OrderID order_id) {
        auto it = order_books_.find(symbol);
        if (it == order_books_.end()) return false;
        
        return it->second->cancelOrder(order_id);
    }
    
    MarketData getMarketData(const std::string& symbol) const {
        auto it = order_books_.find(symbol);
        if (it != order_books_.end()) {
            return it->second->getMarketData();
        }
        return MarketData{};
    }

private:
    void processTrade(const Trade& trade) {
        total_trades_.fetch_add(1);
        total_volume_.fetch_add(trade.price * trade.quantity);
        
        // Update risk manager
        if (trade.buyer_id > 0) {
            risk_manager_.updatePosition(trade.buyer_id, trade.quantity, trade.price);
        }
    }
    
    void updateAnalytics() {
        static int tick = 0;
        tick++;
        
        if (tick % 10 == 0) {
            std::cout << "[TPS] Orders: " << total_orders_.load() 
                      << " Trades: " << total_trades_.load()
                      << " Volume: $" << std::fixed << std::setprecision(2) << total_volume_.load()
                      << std::endl;
        }
    }
};

// ============================================================================
// Main
// ============================================================================

int main() {
    std::cout << "TigerWallet HFT Engine Starting..." << std::endl;
    std::cout << "Port: " << HF_PORT << std::endl;
    
    HFTEngine engine;
    
    // Start engine
    engine.start();
    
    // Submit sample orders
    Order buy_order;
    buy_order.order_id = 1;
    buy_order.user_id = 1001;
    buy_order.symbol = "BTC/USDT";
    buy_order.side = OrderSide::BUY;
    buy_order.type = OrderType::LIMIT;
    buy_order.status = OrderStatus::OPEN;
    buy_order.price = 50000.0;
    buy_order.quantity = 0.5;
    buy_order.filled_quantity = 0;
    
    Order sell_order;
    sell_order.order_id = 2;
    sell_order.user_id = 1002;
    sell_order.symbol = "BTC/USDT";
    sell_order.side = OrderSide::SELL;
    sell_order.type = OrderType::LIMIT;
    sell_order.status = OrderStatus::OPEN;
    sell_order.price = 50000.0;
    sell_order.quantity = 0.5;
    sell_order.filled_quantity = 0;
    
    engine.submitOrder(buy_order);
    engine.submitOrder(sell_order);
    
    // Get market data
    auto data = engine.getMarketData("BTC/USDT");
    std::cout << "BTC/USDT - Bid: " << data.bid_price << " Ask: " << data.ask_price << std::endl;
    
    // Run for a bit
    std::this_thread::sleep_for(std::chrono::seconds(3));
    
    // Stop engine
    engine.stop();
    
    std::cout << "HFT Engine initialized successfully!" << std::endl;
    
    return 0;
}
