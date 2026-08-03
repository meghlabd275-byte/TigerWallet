#include "order_book.h"
#include <chrono>
#include <algorithm>

namespace tigerwallet {

uint64_t Order::get_current_timestamp() {
    return std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
}

OrderBook::OrderBook(const std::string& symbol) : symbol_(symbol) {}
OrderBook::~OrderBook() = default;

void OrderBook::add_order(std::shared_ptr<Order> order) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    order->timestamp = Order::get_current_timestamp();
    orders_[order->id] = order;
    
    if (order->side == OrderSide::BUY) {
        bids_[order->price].push_back(order);
    } else {
        asks_[order->price].push_back(order);
    }
}

void OrderBook::cancel_order(const std::string& order_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = orders_.find(order_id);
    if (it != orders_.end()) {
        it->second->status = OrderStatus::CANCELLED;
        orders_.erase(it);
    }
}

bool OrderBook::match_orders() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    bool matched = false;
    
    while (!bids_.empty() && !asks_.empty()) {
        auto best_bid = bids_.begin()->first;
        auto best_ask = asks_.begin()->first;
        
        if (best_bid < best_ask) break;
        
        auto& bid_queue = bids_.begin()->second;
        auto& ask_queue = asks_.begin()->second;
        
        auto& bid_order = bid_queue.front();
        auto& ask_order = ask_queue.front();
        
        double match_price = ask_order->price;
        double match_quantity = std::min(
            bid_order->quantity - bid_order->filled_quantity,
            ask_order->quantity - ask_order->filled_quantity
        );
        
        bid_order->filled_quantity += match_quantity;
        ask_order->filled_quantity += match_quantity;
        matched = true;
        
        if (bid_order->filled_quantity >= bid_order->quantity) {
            bid_order->status = OrderStatus::FILLED;
            bid_queue.pop_front();
            if (bid_queue.empty()) bids_.erase(bids_.begin());
        }
        
        if (ask_order->filled_quantity >= ask_order->quantity) {
            ask_order->status = OrderStatus::FILLED;
            ask_queue.pop_front();
            if (ask_queue.empty()) asks_.erase(asks_.begin());
        }
    }
    
    return matched;
}

double OrderBook::get_best_bid() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return bids_.empty() ? 0 : bids_.begin()->first;
}

double OrderBook::get_best_ask() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return asks_.empty() ? 0 : asks_.begin()->first;
}

double OrderBook::get_mid_price() const {
    double bid = get_best_bid();
    double ask = get_best_ask();
    return (bid + ask) / 2.0;
}

double OrderBook::get_spread() const {
    return get_best_ask() - get_best_bid();
}

size_t OrderBook::get_bid_count() const {
    std::lock_guard<std::mutex> lock(mutex_);
    size_t count = 0;
    for (const auto& q : bids_) count += q.second.size();
    return count;
}

size_t OrderBook::get_ask_count() const {
    std::lock_guard<std::mutex> lock(mutex_);
    size_t count = 0;
    for (const auto& q : asks_) count += q.second.size();
    return count;
}

// MatchingEngine Implementation
MatchingEngine::MatchingEngine() {}

void MatchingEngine::add_order_book(const std::string& symbol) {
    std::lock_guard<std::mutex> lock(books_mutex_);
    order_books_[symbol] = std::make_unique<OrderBook>(symbol);
}

void MatchingEngine::remove_order_book(const std::string& symbol) {
    std::lock_guard<std::mutex> lock(books_mutex_);
    order_books_.erase(symbol);
}

bool MatchingEngine::process_order(const std::string& user_id, const std::string& symbol,
                                   OrderSide side, double price, double quantity) {
    std::lock_guard<std::mutex> lock(books_mutex_);
    
    auto it = order_books_.find(symbol);
    if (it == order_books_.end()) return false;
    
    auto order = std::make_shared<Order>(user_id, symbol, side, OrderType::LIMIT, price, quantity);
    it->second->add_order(order);
    
    bool matched = it->second->match_orders();
    orders_processed_++;
    
    return matched;
}

bool MatchingEngine::cancel_order(const std::string& order_id) {
    std::lock_guard<std::mutex> lock(books_mutex_);
    
    for (auto& book : order_books_) {
        book.second->cancel_order(order_id);
        return true;
    }
    return false;
}

MatchingEngine::MarketState MatchingEngine::get_market_state(const std::string& symbol) {
    std::lock_guard<std::mutex> lock(books_mutex_);
    
    MarketState state;
    state.symbol = symbol;
    
    auto it = order_books_.find(symbol);
    if (it != order_books_.end()) {
        state.best_bid = it->second->get_best_bid();
        state.best_ask = it->second->get_best_ask();
        state.mid_price = it->second->get_mid_price();
        state.spread = it->second->get_spread();
    }
    
    state.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    return state;
}

} // namespace tigerwallet
