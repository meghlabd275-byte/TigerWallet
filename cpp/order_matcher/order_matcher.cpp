/**
 * TigerWallet High-Performance Order Matcher Implementation
 * C++ with Ultra-Low Latency
 */

#include "order_matcher.hpp"
#include <algorithm>

namespace tigerwallet {
namespace orderbook {

// ============================================================================
// Trade Repository Implementation
// ============================================================================

void TradeRepository::addTrade(const Trade& trade) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    trades_[trade.tradeId] = trade;
    tradesByTime_[trade.timestamp].push_back(trade.tradeId);
    tradesBySymbol_[trade.symbol].push_back(trade.tradeId);
    tradesByUser_[trade.userId].push_back(trade.tradeId);
    tradesByUser_[trade.counterUserId].push_back(trade.tradeId);
}

std::vector<Trade> TradeRepository::getTrades(const std::string& symbol, uint64_t since) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<Trade> result;
    auto it = tradesBySymbol_.find(symbol);
    if (it == tradesBySymbol_.end()) return result;
    
    for (const auto& tradeId : it->second) {
        auto tradeIt = trades_.find(tradeId);
        if (tradeIt != trades_.end() && tradeIt->second.timestamp >= since) {
            result.push_back(tradeIt->second);
        }
    }
    
    std::sort(result.begin(), result.end(), 
        [](const Trade& a, const Trade& b) { return a.timestamp > b.timestamp; });
    
    return result;
}

std::vector<Trade> TradeRepository::getUserTrades(const std::string& userId, uint64_t since) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<Trade> result;
    auto it = tradesByUser_.find(userId);
    if (it == tradesByUser_.end()) return result;
    
    for (const auto& tradeId : it->second) {
        auto tradeIt = trades_.find(tradeId);
        if (tradeIt != trades_.end() && tradeIt->second.timestamp >= since) {
            result.push_back(tradeIt->second);
        }
    }
    
    std::sort(result.begin(), result.end(), 
        [](const Trade& a, const Trade& b) { return a.timestamp > b.timestamp; });
    
    return result;
}

Trade* TradeRepository::getTrade(const std::string& tradeId) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = trades_.find(tradeId);
    if (it != trades_.end()) {
        return &it->second;
    }
    return nullptr;
}

double TradeRepository::getVolume24h(const std::string& symbol) {
    uint64_t since = getCurrentTimestamp() - 86400;
    auto trades = getTrades(symbol, since);
    
    double volume = 0.0;
    for (const auto& trade : trades) {
        volume += trade.quantity;
    }
    return volume;
}

double TradeRepository::getQuoteVolume24h(const std::string& symbol) {
    uint64_t since = getCurrentTimestamp() - 86400;
    auto trades = getTrades(symbol, since);
    
    double volume = 0.0;
    for (const auto& trade : trades) {
        volume += trade.quantity * trade.price;
    }
    return volume;
}

uint64_t TradeRepository::getTradeCount24h(const std::string& symbol) {
    uint64_t since = getCurrentTimestamp() - 86400;
    auto trades = getTrades(symbol, since);
    return trades.size();
}

// ============================================================================
// Order Matcher Engine Implementation
// ============================================================================

OrderMatcherEngine& OrderMatcherEngine::getInstance() {
    static OrderMatcherEngine instance;
    return instance;
}

OrderMatcherEngine::OrderMatcherEngine()
    : riskChecksEnabled_(true)
    , totalOrders_(0)
    , filledOrders_(0)
    , cancelledOrders_(0)
    , totalTrades_(0)
    , totalVolume_(0.0)
{
    tradeRepo_ = std::make_unique<TradeRepository>();
    feeCalc_ = std::make_unique<FeeCalculator>();
    riskMgr_ = std::make_unique<RiskManager>();
}

OrderMatcherEngine::~OrderMatcherEngine() {
    // Cleanup
}

void OrderMatcherEngine::addMarket(const Market& market) {
    std::lock_guard<std::shared_mutex> lock(mutex_);
    
    markets_[market.symbol] = market;
    orderBooks_[market.symbol] = std::make_unique<OrderBook>();
    orderBooks_[market.symbol]->data.symbol = market.symbol;
}

void OrderMatcherEngine::removeMarket(const std::string& symbol) {
    std::lock_guard<std::shared_mutex> lock(mutex_);
    
    markets_.erase(symbol);
    orderBooks_.erase(symbol);
}

std::optional<Market> OrderMatcherEngine::getMarket(const std::string& symbol) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(symbol);
    if (it != markets_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<Market> OrderMatcherEngine::getAllMarkets() {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<Market> result;
    for (const auto& pair : markets_) {
        result.push_back(pair.second);
    }
    return result;
}

std::string OrderMatcherEngine::submitOrder(std::shared_ptr<Order> order) {
    std::lock_guard<std::mutex> orderLock(orderMutex_);
    
    // Generate order ID
    order->orderId = generateOrderId();
    order->createdAt = getCurrentTimestampMillis();
    order->updatedAt = order->createdAt;
    order->remainingQuantity = order->quantity;
    order->filledQuantity = 0.0;
    
    // Validate order
    if (!validateOrder(order)) {
        order->status = OrderStatus::REJECTED;
        orders_[order->orderId] = order;
        return order->orderId;
    }
    
    // Risk checks
    if (riskChecksEnabled_ && !riskMgr_->checkOrder(order)) {
        order->status = OrderStatus::REJECTED;
        orders_[order->orderId] = order;
        return order->orderId;
    }
    
    totalOrders_++;
    
    // Process order based on type
    std::vector<Trade> trades;
    
    if (order->isMarket()) {
        order->status = OrderStatus::OPEN;
        trades = matchMarketOrder(order);
    } else {
        trades = matchLimitOrder(order);
    }
    
    // Add to tracking
    orders_[order->orderId] = order;
    userOrders_[order->userId].push_back(order->orderId);
    symbolOrders_[order->symbol].push_back(order->orderId);
    
    // Record trades
    for (const auto& trade : trades) {
        tradeRepo_->addTrade(trade);
    }
    
    return order->orderId;
}

bool OrderMatcherEngine::cancelOrder(const std::string& orderId) {
    std::lock_guard<std::mutex> orderLock(orderMutex_);
    
    auto it = orders_.find(orderId);
    if (it == orders_.end()) return false;
    
    auto order = it->second;
    if (!order->isActive()) return false;
    
    // Remove from order book
    auto bookIt = orderBooks_.find(order->symbol);
    if (bookIt != orderBooks_.end()) {
        bookIt->second->removeOrder(orderId, order->side);
    }
    
    order->status = OrderStatus::CANCELLED;
    order->updatedAt = getCurrentTimestampMillis();
    cancelledOrders_++;
    
    return true;
}

bool OrderMatcherEngine::modifyOrder(const std::string& orderId, double newPrice, double newQuantity) {
    std::lock_guard<std::mutex> orderLock(orderMutex_);
    
    auto it = orders_.find(orderId);
    if (it == orders_.end()) return false;
    
    auto order = it->second;
    if (!order->isActive()) return false;
    
    // Validate new price and quantity
    auto marketIt = markets_.find(order->symbol);
    if (marketIt == markets_.end()) return false;
    
    const Market& market = marketIt->second;
    if (!validatePrice(market, newPrice) || !validateQuantity(market, newQuantity)) {
        return false;
    }
    
    // Cancel old order and submit new one
    auto bookIt = orderBooks_.find(order->symbol);
    if (bookIt != orderBooks_.end()) {
        bookIt->second->removeOrder(orderId, order->side);
    }
    
    order->price = newPrice;
    order->quantity = newQuantity;
    order->remainingQuantity = newQuantity - order->filledQuantity;
    order->updatedAt = getCurrentTimestampMillis();
    
    // Re-add to order book
    if (order->isActive() && order->isLimit()) {
        auto bookIt = orderBooks_.find(order->symbol);
        if (bookIt != orderBooks_.end()) {
            bookIt->second->addOrder(order);
        }
    }
    
    return true;
}

std::optional<std::shared_ptr<Order>> OrderMatcherEngine::getOrder(const std::string& orderId) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = orders_.find(orderId);
    if (it != orders_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<std::shared_ptr<Order>> OrderMatcherEngine::getUserOrders(const std::string& userId) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<std::shared_ptr<Order>> result;
    auto it = userOrders_.find(userId);
    if (it == userOrders_.end()) return result;
    
    for (const auto& orderId : it->second) {
        auto orderIt = orders_.find(orderId);
        if (orderIt != orders_.end()) {
            result.push_back(orderIt->second);
        }
    }
    
    return result;
}

std::vector<std::shared_ptr<Order>> OrderMatcherEngine::getOpenOrders(const std::string& symbol) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<std::shared_ptr<Order>> result;
    auto it = symbolOrders_.find(symbol);
    if (it == symbolOrders_.end()) return result;
    
    for (const auto& orderId : it->second) {
        auto orderIt = orders_.find(orderId);
        if (orderIt != orders_.end() && orderIt->second->isActive()) {
            result.push_back(orderIt->second);
        }
    }
    
    return result;
}

OrderBook* OrderMatcherEngine::getOrderBook(const std::string& symbol) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = orderBooks_.find(symbol);
    if (it != orderBooks_.end()) {
        return it->second.get();
    }
    return nullptr;
}

std::vector<PriceLevel> OrderMatcherEngine::getDepth(const std::string& symbol, int levels) {
    auto* book = getOrderBook(symbol);
    if (!book) return {};
    
    std::vector<PriceLevel> result;
    
    auto bids = book->getTopBids(levels);
    auto asks = book->getTopAsks(levels);
    
    result.insert(result.end(), bids.begin(), bids.end());
    result.insert(result.end(), asks.begin(), asks.end());
    
    return result;
}

std::vector<Trade> OrderMatcherEngine::getRecentTrades(const std::string& symbol, int limit) {
    auto trades = tradeRepo_->getTrades(symbol, 0);
    if (trades.size() > static_cast<size_t>(limit)) {
        trades.resize(limit);
    }
    return trades;
}

std::vector<Trade> OrderMatcherEngine::getUserTrades(const std::string& userId, int limit) {
    auto trades = tradeRepo_->getUserTrades(userId, 0);
    if (trades.size() > static_cast<size_t>(limit)) {
        trades.resize(limit);
    }
    return trades;
}

double OrderMatcherEngine::getLastPrice(const std::string& symbol) {
    auto* book = getOrderBook(symbol);
    if (!book) return 0.0;
    return book->data.lastPrice;
}

double OrderMatcherEngine::get24hVolume(const std::string& symbol) {
    return tradeRepo_->getVolume24h(symbol);
}

double OrderMatcherEngine::get24hHigh(const std::string& symbol) {
    auto* book = getOrderBook(symbol);
    if (!book) return 0.0;
    return book->data.highPrice;
}

double OrderMatcherEngine::get24hLow(const std::string& symbol) {
    auto* book = getOrderBook(symbol);
    if (!book) return 0.0;
    return book->data.lowPrice;
}

bool OrderMatcherEngine::enableRiskChecks(bool enable) {
    riskChecksEnabled_ = enable;
    return true;
}

void OrderMatcherEngine::setMaxOrderSize(double max) {
    riskMgr_->setMaxOrderSize(max);
}

void OrderMatcherEngine::setMaxNotional(double max) {
    riskMgr_->setMaxNotional(max);
}

OrderMatcherEngine::EngineStats OrderMatcherEngine::getStats() const {
    EngineStats stats;
    stats.totalOrders = totalOrders_;
    stats.filledOrders = filledOrders_;
    stats.cancelledOrders = cancelledOrders_;
    stats.totalTrades = totalTrades_;
    stats.totalVolume = totalVolume_;
    stats.ordersPerSecond = 0; // Would calculate in production
    return stats;
}

void OrderMatcherEngine::setMarketStatus(const std::string& symbol, MarketStatus status) {
    std::lock_guard<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(symbol);
    if (it != markets_.end()) {
        it->second.status = status;
    }
}

// ============================================================================
// Core Matching Logic
// ============================================================================

std::vector<Trade> OrderMatcherEngine::matchOrder(std::shared_ptr<Order> order) {
    std::lock_guard<std::mutex> matchLock(matchMutex_);
    
    if (order->isMarket()) {
        return matchMarketOrder(order);
    } else {
        return matchLimitOrder(order);
    }
}

std::vector<Trade> OrderMatcherEngine::matchLimitOrder(std::shared_ptr<Order> order) {
    std::vector<Trade> trades;
    
    auto* book = getOrCreateOrderBook(order->symbol);
    if (!book) return trades;
    
    // Check if order can match immediately
    double bestPrice = order->isBuy() ? book->getBestAsk() : book->getBestBid();
    
    bool canMatch = false;
    if (order->isBuy() && bestPrice > 0 && order->price >= bestPrice) canMatch = true;
    if (!order->isBuy() && bestPrice > 0 && order->price <= bestPrice) canMatch = true;
    
    if (canMatch) {
        return matchMarketOrder(order);
    }
    
    // Add to order book
    book->addOrder(order);
    order->status = OrderStatus::OPEN;
    
    return trades;
}

std::vector<Trade> OrderMatcherEngine::matchMarketOrder(std::shared_ptr<Order> order) {
    std::vector<Trade> trades;
    
    auto* book = getOrCreateOrderBook(order->symbol);
    if (!book) return trades;
    
    double remaining = order->remainingQuantity;
    auto& levels = order->isBuy() ? book->getAsks() : book->getBids();
    
    for (auto& pair : levels) {
        if (remaining <= 0) break;
        
        auto& level = pair.second;
        auto levelOrders = level->getOrders(100);
        
        for (auto& counterOrder : levelOrders) {
            if (remaining <= 0) break;
            
            double fillQty = std::min(remaining, counterOrder->remainingQuantity);
            double fillPrice = counterOrder->price;
            
            // Create trade
            Trade trade;
            trade.tradeId = generateTradeId();
            trade.orderId = order->orderId;
            trade.counterOrderId = counterOrder->orderId;
            trade.symbol = order->symbol;
            trade.side = order->side;
            trade.price = fillPrice;
            trade.quantity = fillQty;
            trade.timestamp = getCurrentTimestampMillis();
            trade.userId = order->userId;
            trade.counterUserId = counterOrder->userId;
            
            // Calculate fees
            bool isMaker = (order->isBuy() && order->createdAt > counterOrder->createdAt) ||
                          (!order->isBuy() && order->createdAt < counterOrder->createdAt);
            trade.fee = feeCalc_->calculateFee(fillQty, fillPrice, isMaker);
            
            trades.push_back(trade);
            
            // Update orders
            order->filledQuantity += fillQty;
            order->remainingQuantity -= fillQty;
            
            counterOrder->filledQuantity += fillQty;
            counterOrder->remainingQuantity -= fillQty;
            
            remaining -= fillQty;
            
            // Update status
            if (counterOrder->remainingQuantity <= 0.00000001) {
                counterOrder->status = OrderStatus::FILLED;
                filledOrders_++;
            } else {
                counterOrder->status = OrderStatus::PARTIALLY_FILLED;
            }
        }
    }
    
    // Update order status
    if (order->remainingQuantity <= 0.00000001) {
        order->status = OrderStatus::FILLED;
        filledOrders_++;
    } else if (order->filledQuantity > 0) {
        order->status = OrderStatus::PARTIALLY_FILLED;
    } else {
        // Could not fill at all
        if (order->tif == TimeInForce::IOC || order->tif == TimeInForce::FOK) {
            order->status = OrderStatus::CANCELLED;
            cancelledOrders_++;
        } else {
            order->status = OrderStatus::OPEN;
            // Add remaining to book
            book->addOrder(order);
        }
    }
    
    // Update statistics
    totalTrades_ += trades.size();
    for (const auto& trade : trades) {
        totalVolume_ += trade.quantity * trade.price;
    }
    
    // Update order book data
    updateOrderBookData(book, order->symbol);
    
    return trades;
}

// ============================================================================
// Order Validation
// ============================================================================

bool OrderMatcherEngine::validateOrder(const std::shared_ptr<Order>& order) {
    // Check market exists
    auto marketIt = markets_.find(order->symbol);
    if (marketIt == markets_.end()) return false;
    
    const Market& market = marketIt->second;
    
    // Check market status
    if (market.status != MarketStatus::OPEN) return false;
    
    // Check order type
    if (!market.allowMarketOrders && order->isMarket()) return false;
    if (!market.allowLimitOrders && order->isLimit()) return false;
    
    // Validate price and quantity
    if (!validatePrice(market, order->price)) return false;
    if (!validateQuantity(market, order->quantity)) return false;
    
    return true;
}

bool OrderMatcherEngine::validatePrice(const Market& market, double price) {
    if (price <= 0) return false;
    if (price < market.minPrice || price > market.maxPrice) return false;
    return true;
}

bool OrderMatcherEngine::validateQuantity(const Market& market, double quantity) {
    if (quantity <= 0) return false;
    if (quantity < market.minQuantity || quantity > market.maxQuantity) return false;
    return true;
}

// ============================================================================
// Order Book Management
// ============================================================================

OrderBook* OrderMatcherEngine::getOrCreateOrderBook(const std::string& symbol) {
    std::lock_guard<std::shared_mutex> lock(mutex_);
    
    auto it = orderBooks_.find(symbol);
    if (it != orderBooks_.end()) {
        return it->second.get();
    }
    
    // Create new order book
    auto book = std::make_unique<OrderBook>();
    book->data.symbol = symbol;
    OrderBook* ptr = book.get();
    orderBooks_[symbol] = std::move(book);
    
    return ptr;
}

void OrderMatcherEngine::updateOrderBookData(OrderBook* book, const std::string& symbol) {
    if (!book) return;
    
    auto now = getCurrentTimestampMillis();
    
    // Update high/low/open prices
    double bestBid = book->getBestBid();
    double bestAsk = book->getBestAsk();
    
    if (bestBid > 0) {
        if (book->data.openPrice <= 0) book->data.openPrice = bestBid;
        if (book->data.highPrice <= 0 || bestBid > book->data.highPrice) book->data.highPrice = bestBid;
        if (book->data.lowPrice <= 0 || bestBid < book->data.lowPrice) book->data.lowPrice = bestBid;
    }
    
    // Update volume
    book->data.volume24h = tradeRepo_->getVolume24h(symbol);
    book->data.quoteVolume24h = tradeRepo_->getQuoteVolume24h(symbol);
    book->data.tradeCount24h = tradeRepo_->getTradeCount24h(symbol);
    
    book->data.lastUpdate = now;
}

// ============================================================================
// Price Calculation
// ============================================================================

double OrderMatcherEngine::calculateFillPrice(double orderPrice, double matchPrice, double quantity) {
    // VWAP-style calculation
    return matchPrice;
}

double OrderMatcherEngine::calculateAveragePrice(const std::vector<Trade>& trades) {
    if (trades.empty()) return 0.0;
    
    double totalValue = 0.0;
    double totalQty = 0.0;
    
    for (const auto& trade : trades) {
        totalValue += trade.price * trade.quantity;
        totalQty += trade.quantity;
    }
    
    return totalQty > 0 ? totalValue / totalQty : 0.0;
}

// ============================================================================
// ID Generation
// ============================================================================

std::string OrderMatcherEngine::generateOrderId() {
    static std::random_device rd;
    static std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 15);
    
    std::stringstream ss;
    ss << "ORD-";
    ss << std::hex << std::setfill('0') << std::setw(8) << dis(gen);
    ss << std::setw(8) << dis(gen);
    ss << std::setw(8) << dis(gen);
    return ss.str();
}

std::string OrderMatcherEngine::generateTradeId() {
    static std::random_device rd;
    static std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 15);
    
    std::stringstream ss;
    ss << "TRD-";
    ss << std::hex << std::setfill('0') << std::setw(8) << dis(gen);
    ss << std::setw(8) << dis(gen);
    ss << std::setw(8) << dis(gen);
    return ss.str();
}

} // namespace orderbook
} // namespace tigerwallet
