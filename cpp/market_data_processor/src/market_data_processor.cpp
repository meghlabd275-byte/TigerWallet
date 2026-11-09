/**
 * TigerWallet Market Data Processor Implementation
 * High-Performance C++ Market Data Processing
 */

#include "market_data_processor.h"
#include <algorithm>
#include <cmath>
#include <numeric>

namespace tiger {
namespace market_data {

// ============================================================================
// Data Processing Implementation
// ============================================================================

void MarketDataProcessor::process_tick(const Tick& tick) {
    auto start = std::chrono::high_resolution_clock::now();

    auto data = get_or_create_symbol(tick.symbol);

    {
        std::unique_lock lock(mutex_);

        // Update tick buffer
        data->tick_buffer.push_back(tick);
        if (data->tick_buffer.size() > config_.max_tick_buffer) {
            data->tick_buffer.pop_front();
        }

        // Update history
        data->tick_history.push_back(tick);
        if (data->tick_history.size() > config_.tick_retention_minutes * 60) {
            data->tick_history.pop_front();
        }

        // Update statistics
        data->statistics.high = std::max(data->statistics.high, tick.high_24h);
        data->statistics.low = std::min(data->statistics.low == 0 ? tick.low_24h : data->statistics.low, tick.low_24h);

        data->ticks_processed++;
        total_ticks_processed_++;
    }

    // Calculate aggregated price
    calculate_aggregated_price(data);

    // Trigger callback
    if (tick_callback_) {
        tick_callback_(tick);
    }

    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start).count();

    tick_processing_time_us_ += static_cast<uint64_t>(duration);
}

void MarketDataProcessor::process_trade(const Trade& trade) {
    const auto start = std::chrono::high_resolution_clock::now();
    auto data = get_or_create_symbol(trade.symbol);

    {
        std::unique_lock lock(mutex_);

        // Update trade buffer
        data->trade_buffer.push_back(trade);
        if (data->trade_buffer.size() > config_.max_trade_buffer) {
            data->trade_buffer.pop_front();
        }

        // Update history
        data->trade_history.push_back(trade);
        if (data->trade_history.size() > config_.trade_retention_minutes * 60) {
            data->trade_history.pop_front();
        }

        // Update statistics
        data->statistics.volume += trade.quantity;
        data->statistics.quote_volume += trade.quantity * trade.price;
        data->statistics.trade_count++;
        data->statistics.close = trade.price;

        if (data->statistics.start_time == 0) {
            data->statistics.start_time = trade.timestamp;
        }
        data->statistics.end_time = trade.timestamp;

        // Calculate VWAP
        double total_value = 0.0;
        double total_qty = 0.0;
        for (const auto& t : data->trade_history) {
            total_value += t.price * t.quantity;
            total_qty += t.quantity;
        }
        if (total_qty > 0) {
            data->statistics.vwap = total_value / total_qty;
        }

        data->trades_processed++;
        total_trades_processed_++;
    }

    // Calculate aggregated price
    calculate_aggregated_price(data);

    // Trigger callback
    if (trade_callback_) {
        trade_callback_(trade);
    }

    const auto end = std::chrono::high_resolution_clock::now();
    const auto duration = std::chrono::duration_cast<std::chrono::microseconds>(
                              end - start)
                              .count();
    trade_processing_time_us_ += static_cast<uint64_t>(duration);
}

void MarketDataProcessor::process_order_book(const OrderBook& order_book) {
    const auto start = std::chrono::high_resolution_clock::now();
    auto data = get_or_create_symbol(order_book.symbol);

    {
        std::unique_lock lock(mutex_);
        update_order_book(data, order_book);

        data->order_books_processed++;
        total_order_books_processed_++;
    }

    // Calculate aggregated price
    calculate_aggregated_price(data);

    // Trigger callback
    if (order_book_callback_) {
        order_book_callback_(order_book);
    }

    const auto end = std::chrono::high_resolution_clock::now();
    const auto duration = std::chrono::duration_cast<std::chrono::microseconds>(
                              end - start)
                              .count();
    order_book_processing_time_us_ += static_cast<uint64_t>(duration);
}

void MarketDataProcessor::process_candle(const Candle& candle) {
    auto data = get_or_create_symbol(candle.symbol);

    {
        std::unique_lock lock(mutex_);

        // Store candle
        data->candles[candle.timestamp] = candle;

        // Trigger callback
        if (candle_callback_) {
            candle_callback_(candle);
        }
    }
}

void MarketDataProcessor::process_tick_batch(const std::vector<Tick>& ticks) {
    for (const auto& tick : ticks) {
        process_tick(tick);
    }
}

void MarketDataProcessor::process_trade_batch(const std::vector<Trade>& trades) {
    for (const auto& trade : trades) {
        process_trade(trade);
    }
}

// ============================================================================
// Order Book Access
// ============================================================================

std::optional<OrderBook> MarketDataProcessor::get_order_book(const std::string& symbol) const {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        return it->second->current_order_book;
    }
    return std::nullopt;
}

std::vector<OrderBook> MarketDataProcessor::get_all_order_books() const {
    std::shared_lock lock(mutex_);
    std::vector<OrderBook> result;
    result.reserve(symbol_data_.size());

    for (const auto& [symbol, data] : symbol_data_) {
        result.push_back(data->current_order_book);
    }

    return result;
}

// ============================================================================
// Aggregated Data
// ============================================================================

AggregatedPrice MarketDataProcessor::get_aggregated_price(const std::string& symbol) const {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        return it->second->aggregated_price;
    }
    return AggregatedPrice{};
}

// ============================================================================
// Statistics
// ============================================================================

MarketStatistics MarketDataProcessor::get_market_statistics(
    const std::string& symbol,
    uint64_t start_time,
    uint64_t end_time
) {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        // Filter by time range
        MarketStatistics stats = it->second->statistics;

        // Calculate additional metrics
        double total_value = 0.0;
        double total_qty = 0.0;
        double max_slippage = 0.0;

        for (const auto& trade : it->second->trade_history) {
            if (trade.timestamp >= start_time && trade.timestamp <= end_time) {
                total_value += trade.price * trade.quantity;
                total_qty += trade.quantity;
            }
        }

        if (total_qty > 0) {
            stats.vwap = total_value / total_qty;
            stats.avg_trade_size = total_qty / std::max(1, stats.trade_count);
        }

        return stats;
    }

    return MarketStatistics{};
}

// ============================================================================
// Historical Data
// ============================================================================

std::vector<Tick> MarketDataProcessor::get_ticks(
    const std::string& symbol,
    uint64_t start_time,
    uint64_t end_time,
    int limit
) {
    std::shared_lock lock(mutex_);
    std::vector<Tick> result;

    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        for (const auto& tick : it->second->tick_history) {
            if (tick.timestamp >= start_time && tick.timestamp <= end_time) {
                result.push_back(tick);
                if ((int)result.size() >= limit) break;
            }
        }
    }

    return result;
}

std::vector<Trade> MarketDataProcessor::get_trades(
    const std::string& symbol,
    uint64_t start_time,
    uint64_t end_time,
    int limit
) {
    std::shared_lock lock(mutex_);
    std::vector<Trade> result;

    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        for (const auto& trade : it->second->trade_history) {
            if (trade.timestamp >= start_time && trade.timestamp <= end_time) {
                result.push_back(trade);
                if ((int)result.size() >= limit) break;
            }
        }
    }

    return result;
}

std::vector<Candle> MarketDataProcessor::get_candles(
    const std::string& symbol,
    uint64_t start_time,
    uint64_t end_time,
    const std::string& interval
) {
    std::shared_lock lock(mutex_);
    std::vector<Candle> result;

    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        for (const auto& [timestamp, candle] : it->second->candles) {
            if (timestamp >= start_time && timestamp <= end_time) {
                result.push_back(candle);
            }
        }
    }

    std::sort(result.begin(), result.end(),
        [](const Candle& a, const Candle& b) { return a.timestamp < b.timestamp; });

    return result;
}

// ============================================================================
// Technical Indicators
// ============================================================================

BollingerBands MarketDataProcessor::calculate_bollinger_bands(
    const std::string& symbol,
    int period,
    double std_dev_mult
) {
    std::shared_lock lock(mutex_);
    BollingerBands bands;

    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end() || it->second->trade_history.size() < (size_t)period) {
        return bands;
    }

    std::vector<double> prices;
    prices.reserve(period);

    auto& history = it->second->trade_history;
    for (int i = (int)history.size() - period; i < (int)history.size(); i++) {
        if (i >= 0) {
            prices.push_back(history[i].price);
        }
    }

    if (prices.size() < (size_t)period) return bands;

    // Calculate SMA (middle band)
    double sum = std::accumulate(prices.begin(), prices.end(), 0.0);
    bands.middle = sum / prices.size();

    // Calculate standard deviation
    double std_dev = calculate_std_dev(prices);

    // Calculate bands
    bands.upper = bands.middle + (std_dev_mult * std_dev);
    bands.lower = bands.middle - (std_dev_mult * std_dev);

    return bands;
}

RSI MarketDataProcessor::calculate_rsi(const std::string& symbol, int period) {
    std::shared_lock lock(mutex_);
    RSI rsi;

    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end() || it->second->trade_history.size() < (size_t)(period + 1)) {
        return rsi;
    }

    auto& history = it->second->trade_history;

    double gains = 0.0;
    double losses = 0.0;

    for (size_t i = history.size() - period; i < history.size(); i++) {
        if (i > 0) {
            double change = history[i].price - history[i-1].price;
            if (change > 0) gains += change;
            else losses -= change;
        }
    }

    double avg_gain = gains / period;
    double avg_loss = losses / period;

    if (avg_loss == 0) {
        rsi.value = 100.0;
        rsi.is_overbought = true;
    } else {
        double rs = avg_gain / avg_loss;
        rsi.value = 100.0 - (100.0 / (1.0 + rs));

        rsi.is_overbought = rsi.value >= 70;
        rsi.is_oversold = rsi.value <= 30;
    }

    return rsi;
}

MACD MarketDataProcessor::calculate_macd(const std::string& symbol, int fast, int slow, int signal) {
    std::shared_lock lock(mutex_);
    MACD macd;

    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end()) {
        return macd;
    }

    auto& history = it->second->trade_history;
    if (history.size() < (size_t)slow) {
        return macd;
    }

    // The smoothing factors for each exponential moving average.
    const double fast_mult = 2.0 / (fast + 1);
    const double slow_mult = 2.0 / (slow + 1);
    const double signal_mult = 2.0 / (signal + 1);

    // The signal line is the EMA of the MACD line, so the MACD line has to be
    // accumulated first. It is only defined from the point where both slow
    // and fast averages have seen enough history to be seeded, which is after
    // `slow` samples when both are seeded from the same first price.
    std::vector<double> macd_series;
    macd_series.reserve(history.size());

    double fast_ema = history[0].price;
    double slow_ema = history[0].price;
    for (size_t i = 0; i < history.size(); i++) {
        const double price = history[i].price;
        if (i > 0) {
            fast_ema = (price * fast_mult) + (fast_ema * (1.0 - fast_mult));
            slow_ema = (price * slow_mult) + (slow_ema * (1.0 - slow_mult));
        }
        macd_series.push_back(fast_ema - slow_ema);
    }

    // Seed the signal EMA from the first MACD value and smooth the rest.
    double signal_ema = macd_series[0];
    for (size_t i = 1; i < macd_series.size(); i++) {
        signal_ema = (macd_series[i] * signal_mult) +
                     (signal_ema * (1.0 - signal_mult));
    }

    macd.macd_line = macd_series.back();
    macd.signal_line = signal_ema;
    macd.histogram = macd.macd_line - macd.signal_line;

    return macd;
}

MovingAverage MarketDataProcessor::calculate_moving_averages(const std::string& symbol, int period) {
    std::shared_lock lock(mutex_);
    MovingAverage ma;

    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end() || it->second->trade_history.size() < (size_t)period) {
        return ma;
    }

    auto& history = it->second->trade_history;
    std::vector<double> prices;
    prices.reserve(period);

    for (int i = (int)history.size() - period; i < (int)history.size(); i++) {
        if (i >= 0) {
            prices.push_back(history[i].price);
        }
    }

    // SMA
    double sum = std::accumulate(prices.begin(), prices.end(), 0.0);
    ma.sma = sum / prices.size();

    // EMA
    double ema_mult = 2.0 / (period + 1);
    ma.ema = prices[0];
    for (size_t i = 1; i < prices.size(); i++) {
        ma.ema = (prices[i] * ema_mult) + (ma.ema * (1 - ema_mult));
    }

    // WMA
    double wma_sum = 0.0;
    double weight_sum = 0.0;
    for (size_t i = 0; i < prices.size(); i++) {
        double weight = i + 1;
        wma_sum += prices[i] * weight;
        weight_sum += weight;
    }
    ma.wma = wma_sum / weight_sum;

    return ma;
}

// ============================================================================
// Market Analysis
// ============================================================================

double MarketDataProcessor::calculate_vwap(const std::string& symbol, int window_seconds) {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end()) {
        return 0.0;
    }

    uint64_t cutoff_time = it->second->statistics.end_time - (window_seconds * 1000);

    double total_value = 0.0;
    double total_qty = 0.0;

    for (const auto& trade : it->second->trade_history) {
        if (trade.timestamp >= cutoff_time) {
            total_value += trade.price * trade.quantity;
            total_qty += trade.quantity;
        }
    }

    return total_qty > 0 ? total_value / total_qty : 0.0;
}

double MarketDataProcessor::calculate_slippage(const std::string& symbol, double order_size) {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end()) {
        return 0.0;
    }

    const auto& order_book = it->second->current_order_book;
    if (order_book.bids.empty()) {
        return 0.0;
    }

    double remaining = order_size;
    double expected_price = order_book.bids[0].price;
    double actual_cost = 0.0;

    // Calculate actual cost with order book
    for (const auto& level : order_book.asks) {
        double fill_qty = std::min(remaining, level.quantity);
        actual_cost += fill_qty * level.price;
        remaining -= fill_qty;

        if (remaining <= 0) break;
    }

    if (order_size > 0) {
        double expected_cost = order_size * expected_price;
        return (actual_cost - expected_cost) / expected_cost;
    }

    return 0.0;
}

double MarketDataProcessor::calculate_market_impact(const std::string& symbol, double order_size) {
    // Simplified market impact model
    return std::sqrt(order_size / 1000000.0) * 0.01;
}

double MarketDataProcessor::calculate_liquidity(const std::string& symbol, double price_range_percent) {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it == symbol_data_.end()) {
        return 0.0;
    }

    const auto& order_book = it->second->current_order_book;
    if (order_book.asks.empty()) {
        return 0.0;
    }

    double mid_price = order_book.asks[0].price;
    double price_range = mid_price * (price_range_percent / 100.0);
    double min_price = mid_price - price_range;
    double max_price = mid_price + price_range;

    double total_volume = 0.0;

    // Sum volume in price range (asks)
    for (const auto& level : order_book.asks) {
        if (level.price <= max_price) {
            total_volume += level.quantity;
        }
    }

    // Sum volume in price range (bids)
    for (const auto& level : order_book.bids) {
        if (level.price >= min_price) {
            total_volume += level.quantity;
        }
    }

    return total_volume;
}

// ============================================================================
// Cleanup
// ============================================================================

void MarketDataProcessor::clear_symbol_data(const std::string& symbol) {
    std::unique_lock lock(mutex_);
    symbol_data_.erase(symbol);
}

// ============================================================================
// Helper Methods
// ============================================================================

std::shared_ptr<MarketDataProcessor::SymbolData>
MarketDataProcessor::get_or_create_symbol(const std::string& symbol) {
    std::unique_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        return it->second;
    }

    auto data = std::make_shared<SymbolData>();
    symbol_data_[symbol] = data;
    return data;
}

void MarketDataProcessor::update_order_book(
    std::shared_ptr<SymbolData>& data,
    const OrderBook& order_book
) {
    data->current_order_book = order_book;

    // Update bid levels
    data->bid_levels.clear();
    for (const auto& level : order_book.bids) {
        data->bid_levels[level.price] = level;
    }

    // Update ask levels
    data->ask_levels.clear();
    for (const auto& level : order_book.asks) {
        data->ask_levels[level.price] = level;
    }
}

void MarketDataProcessor::update_statistics(
    std::shared_ptr<SymbolData>& data,
    const Trade& trade
) {
    // Statistics are updated in process_trade
    (void)data;
    (void)trade;
}

void MarketDataProcessor::calculate_aggregated_price(std::shared_ptr<SymbolData>& data) {
    auto& agg = data->aggregated_price;
    agg.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();

    // Get best bid/ask from order book
    if (!data->bid_levels.empty()) {
        agg.best_bid = data->bid_levels.begin()->first;
        agg.total_bid_volume = 0;
        for (const auto& level : data->bid_levels) {
            agg.total_bid_volume += level.second.quantity;
        }
    }

    if (!data->ask_levels.empty()) {
        agg.best_ask = data->ask_levels.begin()->first;
        agg.total_ask_volume = 0;
        for (const auto& level : data->ask_levels) {
            agg.total_ask_volume += level.second.quantity;
        }
    }

    // Calculate spread and mid price
    if (agg.best_bid > 0 && agg.best_ask > 0) {
        agg.spread = agg.best_ask - agg.best_bid;
        agg.mid_price = (agg.best_bid + agg.best_ask) / 2.0;
    }

    // Calculate volume weighted prices
    double bid_value = 0.0;
    double bid_vol = 0.0;
    for (const auto& level : data->bid_levels) {
        bid_value += level.first * level.second.quantity;
        bid_vol += level.second.quantity;
    }
    if (bid_vol > 0) {
        agg.volume_weighted_bid = bid_value / bid_vol;
    }

    double ask_value = 0.0;
    double ask_vol = 0.0;
    for (const auto& level : data->ask_levels) {
        ask_value += level.first * level.second.quantity;
        ask_vol += level.second.quantity;
    }
    if (ask_vol > 0) {
        agg.volume_weighted_ask = ask_value / ask_vol;
    }
}

void MarketDataProcessor::cleanup_old_data(std::shared_ptr<SymbolData>& data) {
    // Cleanup old data if needed
    (void)data;
}

double MarketDataProcessor::calculate_volatility(const std::vector<double>& prices) const {
    if (prices.size() < 2) return 0.0;

    double mean = std::accumulate(prices.begin(), prices.end(), 0.0) / prices.size();

    double sq_sum = 0.0;
    for (double p : prices) {
        sq_sum += (p - mean) * (p - mean);
    }

    return std::sqrt(sq_sum / (prices.size() - 1));
}

double MarketDataProcessor::calculate_std_dev(const std::vector<double>& values) const {
    return calculate_volatility(values);
}

} // namespace market_data
} // namespace tiger
