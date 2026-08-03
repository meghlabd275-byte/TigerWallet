#ifndef ORDER_BOOK_H
#define ORDER_BOOK_H

#include <string>
#include <map>
#include <set>
#include <queue>
#include <memory>
#include <mutex>
#include <atomic>

namespace tigerwallet {

enum class OrderSide { BUY, SELL };
enum class OrderType { LIMIT, MARKET };
enum class OrderStatus { PENDING, FILLED, PARTIALLY_FILLED, CANCELLED };

struct Order {
    std::string id;
    std::string user_id;
    std::string symbol;
    OrderSide side;
    OrderType type;
    double price;
    double quantity;
    double filled_quantity;
    OrderStatus status;
    uint64_t timestamp;
    int priority;
    
    Order() : price(0), quantity(0), filled_quantity(0), timestamp(0), priority(0) {}
};

class OrderBook {
public:
    OrderBook(const std::string& symbol);
    ~OrderBook();
    
    void add_order(std::shared_ptr<Order> order);
    void cancel_order(const std::string& order_id);
    bool match_orders();
    
    double get_best_bid() const;
    double get_best_ask() const;
    double get_mid_price() const;
    double get_spread() const;
    size_t get_bid_count() const;
    size_t get_ask_count() const;

private:
    std::string symbol_;
    std::map<double, std::deque<std::shared_ptr<Order>>, std::greater<double>> bids_;
    std::map<double, std::deque<std::shared_ptr<Order>>> asks_;
    std::map<std::string, std::shared_ptr<Order>> orders_;
    mutable std::mutex mutex_;
};

class MatchingEngine {
public:
    MatchingEngine();
    
    void add_order_book(const std::string& symbol);
    void remove_order_book(const std::string& symbol);
    
    bool process_order(const std::string& user_id, const std::string& symbol,
                      OrderSide side, double price, double quantity);
    bool cancel_order(const std::string& order_id);
    
    struct MarketState {
        std::string symbol;
        double best_bid;
        double best_ask;
        double mid_price;
        double spread;
        uint64_t timestamp;
    };
    
    MarketState get_market_state(const std::string& symbol);

private:
    std::map<std::string, std::unique_ptr<OrderBook>> order_books_;
    std::mutex books_mutex_;
    std::atomic<uint64_t> orders_processed_{0};
};

} // namespace tigerwallet

#endif
