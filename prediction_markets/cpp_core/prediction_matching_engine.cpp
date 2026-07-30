/**
 * TigerWallet Prediction Markets - Matching Engine Implementation
 * High-performance C++ implementation
 */

#include "prediction_matching_engine.h"
#include <algorithm>
#include <cstring>
#include <iostream>

namespace tiger {
namespace prediction {

// ============================================================================
// Order Matching Implementation
// ============================================================================

std::vector<Trade> OrderBook::match_orders(uint64_t timestamp) {
    std::vector<Trade> trades;
    trades.reserve(100);  // Pre-allocate
    
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    // Simple FIFO matching: match best bid with best ask
    while (best_bid_price_ > 0 && best_ask_price_ <= MAX_PRICE &&
           best_bid_price_ >= best_ask_price_) {
        
        // Get best bid and ask price levels
        auto bid_idx = price_to_index(best_bid_price_);
        auto ask_idx = price_to_index(best_ask_price_);
        
        if (bid_idx >= PRICE_LEVELS || ask_idx >= PRICE_LEVELS) break;
        
        auto& bid_level = bid_levels_[bid_idx];
        auto& ask_level = ask_levels_[ask_idx];
        
        if (bid_level.orders.empty() || ask_level.orders.empty()) break;
        
        // Get the first order from each side
        uint64_t bid_order_id = bid_level.orders.front();
        uint64_t ask_order_id = ask_level.orders.front();
        
        auto bid_it = orders_.find(bid_order_id);
        auto ask_it = orders_.find(ask_order_id);
        
        if (bid_it == orders_.end() || ask_it == orders_.end()) break;
        
        Order& bid_order = bid_it->second;
        Order& ask_order = ask_it->second;
        
        // Calculate fill amount
        uint64_t bid_remaining = bid_order.amount - bid_order.filled_amount;
        uint64_t ask_remaining = ask_order.amount - ask_order.filled_amount;
        uint64_t fill_amount = std::min(bid_remaining, ask_remaining);
        
        // Execute trade at the price of the order that was in the book first
        uint64_t trade_price = (bid_order.timestamp < ask_order.timestamp) 
                               ? bid_order.price 
                               : ask_order.price;
        
        // Create trade
        Trade trade = {};
        trade.trade_id = 0;  // Will be assigned by market manager
        trade.order_id = ask_order_id;
        trade.market_id = bid_order.market_id;
        trade.outcome_id = outcome_id_;
        trade.maker_order_id = (bid_order.timestamp < ask_order.timestamp) 
                               ? bid_order_id 
                               : ask_order_id;
        trade.taker_order_id = (bid_order.timestamp < ask_order_timestamp) 
                               ? ask_order_id 
                               : bid_order_id;
        trade.price = trade_price;
        trade.amount = fill_amount;
        trade.timestamp = timestamp;
        trade.fees = (fill_amount * 30) / 100000;  // 0.03% fee
        
        trades.push_back(trade);
        
        // Update orders
        bid_order.filled_amount += fill_amount;
        ask_order.filled_amount += fill_amount;
        
        // Update price levels
        bid_level.total_amount -= fill_amount;
        ask_level.total_amount -= fill_amount;
        
        // Remove filled orders
        if (bid_order.filled_amount >= bid_order.amount) {
            bid_level.orders.pop_front();
            bid_level.order_count--;
            orders_.erase(bid_it);
        }
        
        if (ask_order.filled_amount >= ask_order.amount) {
            ask_level.orders.pop_front();
            ask_level.order_count--;
            orders_.erase(ask_it);
        }
        
        // Update best prices
        while (bid_idx < PRICE_LEVELS && bid_levels_[bid_idx].total_amount == 0) {
            bid_idx++;
            if (bid_idx >= PRICE_LEVELS) {
                best_bid_price_ = 0;
                break;
            }
            best_bid_price_ = bid_levels_[bid_idx].price;
        }
        
        while (ask_idx < PRICE_LEVELS && ask_levels_[ask_idx].total_amount == 0) {
            if (ask_idx == 0) {
                best_ask_price_ = MAX_PRICE;
                break;
            }
            ask_idx--;
            best_ask_price_ = ask_levels_[ask_idx].price;
        }
    }
    
    return trades;
}

// ============================================================================
// Prediction Matching Engine Implementation
// ============================================================================

PredictionMatchingEngine::PredictionMatchingEngine(uint32_t num_threads)
    : running_(false), 
      num_threads_(num_threads > 0 ? num_threads : 4),
      total_orders_processed_(0),
      total_trades_executed_(0),
      total_volume_(0),
      last_match_time_(0) {
    
    market_manager_ = std::make_unique<MarketManager>();
}

PredictionMatchingEngine::~PredictionMatchingEngine() {
    stop();
}

void PredictionMatchingEngine::start() {
    if (running_.load()) return;
    
    running_.store(true);
    
    // Pin threads to CPU cores for better performance
    cpu_set_t cpus;
    CPU_ZERO(&cpus);
    
    for (uint32_t i = 0; i < num_threads_; ++i) {
        CPU_SET(i, &cpus);
        worker_threads_.emplace_back([this, i]() {
            CPU_ZERO(&cpus);
            CPU_SET(i, &cpus);
            TIGER_CPU_PIN(cpus);
            worker_thread();
        });
    }
    
    std::cout << "[PredictionEngine] Started " << num_threads_ 
              << " worker threads" << std::endl;
}

void PredictionMatchingEngine::stop() {
    if (!running_.load()) return;
    
    running_.store(false);
    queue_cv_.notify_all();
    
    for (auto& thread : worker_threads_) {
        if (thread.joinable()) {
            thread.join();
        }
    }
    
    worker_threads_.clear();
    std::cout << "[PredictionEngine] Stopped" << std::endl;
}

void PredictionMatchingEngine::worker_thread() {
    while (running_.load()) {
        Order order;
        
        {
            std::unique_lock<std::mutex> lock(queue_mutex_);
            queue_cv_.wait_for(lock, std::chrono::milliseconds(100), [this]() {
                return !running_.load() || !order_queue_.empty();
            });
            
            if (!running_.load()) break;
            
            if (order_queue_.empty()) continue;
            
            order = order_queue_.front();
            order_queue_.pop_front();
        }
        
        // Record order latency
        auto order_time = std::chrono::high_resolution_clock::now();
        auto order_latency = std::chrono::duration_cast<std::chrono::nanoseconds>(
            order_time.time_since_epoch()
        ).count() - static_cast<int64_t>(order.timestamp);
        
        // Update latency stats
        uint64_t latency = static_cast<uint64_t>(order_latency);
        auto& lat_stats = order_latency_;
        
        uint64_t old_min = lat_stats.min_latency_ns.load();
        while (latency < old_min && 
               !lat_stats.min_latency_ns.compare_exchange_weak(old_min, latency)) {}
        
        uint64_t old_max = lat_stats.max_latency_ns.load();
        while (latency > old_max && 
               !lat_stats.max_latency_ns.compare_exchange_weak(old_max, latency)) {}
        
        lat_stats.total_latency_ns.fetch_add(latency);
        lat_stats.count.fetch_add(1);
        
        // Process order
        auto match_start = std::chrono::high_resolution_clock::now();
        auto trades = process_order(order);
        auto match_end = std::chrono::high_resolution_clock::now();
        
        // Record match latency
        auto match_latency = std::chrono::duration_cast<std::chrono::nanoseconds>(
            match_end - match_start
        ).count();
        
        auto& match_stats = match_latency_;
        uint64_t m_latency = static_cast<uint64_t>(match_latency);
        
        old_min = match_stats.min_latency_ns.load();
        while (m_latency < old_min && 
               !match_stats.min_latency_ns.compare_exchange_weak(old_min, m_latency)) {}
        
        old_max = match_stats.max_latency_ns.load();
        while (m_latency > old_max && 
               !match_stats.max_latency_ns.compare_exchange_weak(old_max, m_latency)) {}
        
        match_stats.total_latency_ns.fetch_add(m_latency);
        match_stats.count.fetch_add(1);
        
        // Update stats
        total_orders_processed_.fetch_add(1);
        total_trades_executed_.fetch_add(trades.size());
        
        uint64_t volume = 0;
        for (const auto& trade : trades) {
            volume += trade.amount * trade.price;
        }
        total_volume_.fetch_add(volume);
        
        last_match_time_.store(
            std::chrono::duration_cast<std::chrono::milliseconds>(
                match_end.time_since_epoch()
            ).count()
        );
    }
}

std::vector<Trade> PredictionMatchingEngine::process_order(const Order& order) {
    std::vector<Trade> trades;
    
    auto* order_book = market_manager_->get_order_book(
        order.market_id, order.outcome_id
    );
    
    if (!order_book) return trades;
    
    // For market orders, get the best price
    Order processed_order = order;
    if (order.order_type == OrderType::MARKET) {
        auto [best_bid, best_ask] = order_book->get_best_prices();
        processed_order.price = (order.side == OrderSide::BUY) ? best_ask : best_bid;
        
        if (processed_order.price == 0 || processed_order.price == MAX_PRICE) {
            // No liquidity for market order
            return trades;
        }
    }
    
    // Add order to book
    order_book->add_order(processed_order);
    
    // Match orders
    auto timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    trades = order_book->match_orders(timestamp);
    
    // Assign trade IDs
    for (auto& trade : trades) {
        trade.trade_id = market_manager_->next_trade_id_++;
    }
    
    return trades;
}

uint64_t PredictionMatchingEngine::submit_order(const Order& order) {
    uint64_t order_id = market_manager_->next_order_id_++;
    
    Order submitted_order = order;
    submitted_order.order_id = order_id;
    submitted_order.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    {
        std::lock_guard<std::mutex> lock(queue_mutex_);
        order_queue_.push_back(submitted_order);
    }
    
    queue_cv_.notify_one();
    
    return order_id;
}

bool PredictionMatchingEngine::cancel_order(
    uint64_t market_id,
    uint32_t outcome_id,
    uint64_t order_id,
    uint32_t user_id
) {
    auto* order_book = market_manager_->get_order_book(market_id, outcome_id);
    if (!order_book) return false;
    
    auto order = order_book->get_order(order_id);
    if (!order.has_value()) return false;
    
    if (order->user_id != user_id) return false;
    
    return order_book->remove_order(order_id);
}

uint32_t PredictionMatchingEngine::cancel_all_orders(uint32_t user_id) {
    uint32_t count = 0;
    
    // This would iterate through all markets and outcomes
    // For now, return 0 - implementation depends on market manager iteration
    return count;
}

PredictionMatchingEngine::EngineStats PredictionMatchingEngine::get_stats() const {
    EngineStats stats = {};
    
    stats.total_orders = total_orders_processed_.load();
    stats.total_trades = total_trades_executed_.load();
    stats.total_volume = total_volume_.load();
    stats.last_match_ms = last_match_time_.load();
    
    auto& order_stats = order_latency_;
    auto& match_stats = match_latency_;
    
    uint64_t order_count = order_stats.count.load();
    uint64_t match_count = match_stats.count.load();
    
    if (order_count > 0) {
        stats.avg_order_latency_us = order_stats.total_latency_ns.load() / order_count / 1000;
        stats.min_order_latency_us = order_stats.min_latency_ns.load() / 1000;
        stats.max_order_latency_us = order_stats.max_latency_ns.load() / 1000;
    }
    
    if (match_count > 0) {
        stats.avg_match_latency_us = match_stats.total_latency_ns.load() / match_count / 1000;
        stats.min_match_latency_us = match_stats.min_latency_ns.load() / 1000;
        stats.max_match_latency_us = match_stats.max_latency_ns.load() / 1000;
    }
    
    {
        std::lock_guard<std::mutex> lock(queue_mutex_);
        stats.queue_size = static_cast<uint64_t>(order_queue_.size());
    }
    
    return stats;
}

std::vector<Market> PredictionMatchingEngine::get_featured_markets() const {
    auto featured_ids = market_manager_->get_featured_markets();
    std::vector<Market> markets;
    markets.reserve(featured_ids.size());
    
    for (auto id : featured_ids) {
        auto market = market_manager_->get_market(id);
        if (market.has_value()) {
            markets.push_back(market.value());
        }
    }
    
    return markets;
}

std::vector<std::pair<uint64_t, uint64_t>> PredictionMatchingEngine::get_market_depth(
    uint64_t market_id,
    uint32_t outcome_id,
    uint32_t levels,
    bool bids
) const {
    auto* order_book = market_manager_->get_order_book(market_id, outcome_id);
    if (!order_book) return {};
    
    return order_book->get_depth(levels, bids);
}

} // namespace prediction
} // namespace tiger
