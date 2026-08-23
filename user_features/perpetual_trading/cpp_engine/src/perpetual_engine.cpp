/**
 * TigerWallet - Perpetual Trading Engine Implementation
 * Ultra-low latency C++ implementation
 */

#include "perpetual_engine.h"
#include <algorithm>
#include <cmath>
#include <thread>
#include <chrono>

namespace tiger {
namespace perpetual {

// ============ Order Implementation ============

Decimal Order::get_notional() const {
    return (Decimal)price * filled_quantity;
}

bool Order::is_active() const {
    return status == OrderStatus::Open || 
           status == OrderStatus::PartiallyFilled ||
           status == OrderStatus::Pending;
}

// ============ Position Implementation ============

Decimal Position::get_margin_required(Decimal margin_ratio) const {
    if (size == 0) return 0;
    Decimal position_value = (Decimal)abs((int64_t)size) * entry_price;
    return position_value * margin_ratio;
}

Decimal Position::calculate_pnl(Price current_price) const {
    if (size == 0) return 0;
    
    Decimal price_diff;
    if (side == PositionSide::Long) {
        price_diff = (Decimal)(current_price - entry_price);
    } else {
        price_diff = (Decimal)(entry_price - current_price);
    }
    
    return price_diff * abs((int64_t)size);
}

bool Position::should_liquidate(Price current_price) const {
    if (side == PositionSide::Long) {
        return current_price <= liquidation_price;
    } else {
        return current_price >= liquidation_price;
    }
}

// ============ OrderBook Implementation ============

std::optional<Price> OrderBook::best_bid() const {
    if (bids.empty()) return std::nullopt;
    return bids[0].price;
}

std::optional<Price> OrderBook::best_ask() const {
    if (asks.empty()) return std::nullopt;
    return asks[0].price;
}

Price OrderBook::get_spread() const {
    auto bid = best_bid();
    auto ask = best_ask();
    if (!bid || !ask) return 0;
    return *ask - *bid;
}

std::vector<std::pair<Price, Quantity>> OrderBook::get_depth(int levels) const {
    std::vector<std::pair<Price, Quantity>> depth;
    
    for (int i = 0; i < std::min(levels, (int)bids.size()); i++) {
        depth.push_back({bids[i].price, bids[i].quantity});
    }
    
    for (int i = 0; i < std::min(levels, (int)asks.size()); i++) {
        depth.push_back({asks[i].price, asks[i].quantity});
    }
    
    return depth;
}

void OrderBook::add_order(const Order& order) {
    if (order.side == Side::Buy) {
        OrderBookLevel level = {order.price, order.remaining_quantity, order.order_id};
        bids.push_back(level);
        std::sort(bids.begin(), bids.end(), 
            [](const OrderBookLevel& a, const OrderBookLevel& b) {
                return a.price > b.price;
            });
    } else {
        OrderBookLevel level = {order.price, order.remaining_quantity, order.order_id};
        asks.push_back(level);
        std::sort(asks.begin(), asks.end(),
            [](const OrderBookLevel& a, const OrderBookLevel& b) {
                return a.price < b.price;
            });
    }
}

void OrderBook::remove_order(OrderID order_id) {
    bids.erase(std::remove_if(bids.begin(), bids.end(),
        [order_id](const OrderBookLevel& level) {
            return level.order_id == order_id;
        }), bids.end());
    
    asks.erase(std::remove_if(asks.begin(), asks.end(),
        [order_id](const OrderBookLevel& level) {
            return level.order_id == order_id;
        }), asks.end());
}

std::vector<Order> OrderBook::match(Order& incoming) {
    std::vector<Order> matched_orders;
    
    auto& book_side = (incoming.side == Side::Buy) ? asks : bids;
    auto& book_side_other = (incoming.side == Side::Buy) ? bids : asks;
    
    // Market order - match at best price
    if (incoming.order_type == OrderType::Market) {
        if (incoming.side == Side::Buy && !asks.empty()) {
            incoming.price = asks[0].price;
        } else if (incoming.side == Side::Sell && !bids.empty()) {
            incoming.price = bids[0].price;
        }
    }
    
    // Match against book
    for (auto it = book_side.begin(); it != book_side.end() && incoming.remaining_quantity > 0;) {
        // Check price condition
        bool price_matches = (incoming.side == Side::Buy) ? 
            (it->price <= incoming.price) : (it->price >= incoming.price);
        
        if (!price_matches) break;
        
        // Calculate trade quantity
        Quantity trade_qty = std::min(incoming.remaining_quantity, it->quantity);
        
        // Update quantities
        incoming.remaining_quantity -= trade_qty;
        it->quantity -= trade_qty;
        
        // Create matched order
        Order matched = {};
        matched.order_id = it->order_id;
        matched.filled_quantity = trade_qty;
        matched.status = (it->quantity == 0) ? OrderStatus::Filled : OrderStatus::PartiallyFilled;
        matched_orders.push_back(matched);
        
        // Remove if fully filled
        if (it->quantity == 0) {
            it = book_side.erase(it);
        } else {
            ++it;
        }
    }
    
    return matched_orders;
}

// ============ MarketData Implementation ============

Decimal MarketData::calculate_funding(Decimal position_size) const {
    return position_size * funding_rate / 24; // Hourly funding
}

// ============ AccountRisk Implementation ============

Decimal AccountRisk::get_margin_ratio() const {
    if (total_position_value == 0) return 0;
    return total_margin_used / total_position_value;
}

bool AccountRisk::is_undermargined(Decimal min_ratio) const {
    return get_margin_ratio() < min_ratio;
}

bool AccountRisk::should_liquidate(Decimal liq_ratio) const {
    return get_margin_ratio() < liq_ratio;
}

// ============ LiquidationEngine Implementation ============

LiquidationEngine::LiquidationEngine(const RiskLimits& limits) : limits_(limits) {}

std::vector<LiquidationResult> LiquidationEngine::check_liquidations(
    const std::map<UserID, Position>& positions,
    const std::map<Market, MarketData>& markets
) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<LiquidationResult> results;
    
    for (const auto& [key, position] : positions) {
        if (position.status != PositionStatus::Open && 
            position.status != PositionStatus::Partial) {
            continue;
        }
        
        auto market_it = markets.find(position.market);
        if (market_it == markets.end()) continue;
        
        const auto& market = market_it->second;
        
        if (position.should_liquidate(market.mark_price)) {
            results.push_back(execute_liquidation(position, market));
        }
    }
    
    return results;
}

LiquidationResult LiquidationEngine::execute_liquidation(
    const Position& position,
    const MarketData& market
) {
    LiquidationResult result = {};
    result.user_id = position.user_id;
    result.market = position.market;
    result.side = position.side;
    result.liquidated_size = position.size;
    result.liquidation_price = position.liquidation_price;
    result.partial = false;
    
    // Calculate penalty
    Decimal position_value = (Decimal)abs((int64_t)position.size) * market.mark_price;
    result.penalty = get_liquidation_penalty(position_value);
    
    return result;
}

Price LiquidationEngine::calculate_liquidation_price(
    const Position& position,
    Decimal collateral,
    Decimal margin_ratio
) {
    if (position.side == PositionSide::Long) {
        // Long: liquidation when price drops
        Decimal price = (collateral * (Decimal)100000000 - 
                       (Decimal)position.size * limits_.liquidation_ratio * (Decimal)100000000) /
                       (Decimal)position.size;
        return (Price)price;
    } else {
        // Short: liquidation when price rises
        Decimal price = (collateral * (Decimal)100000000 + 
                       (Decimal)position.size * limits_.liquidation_ratio * (Decimal)100000000) /
                       (Decimal)position.size;
        return (Price)price;
    }
}

Decimal LiquidationEngine::get_liquidation_penalty(Decimal position_value) {
    return position_value * Decimal(5) / Decimal(10000); // 0.05%
}

// ============ MatchingEngine Implementation ============

MatchingEngine::MatchingEngine() {}

std::variant<Order, std::string> MatchingEngine::process_order(
    const Order& order,
    OrderBook& book,
    AccountRisk& account
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Check margin
    Decimal order_value = order.get_notional();
    Decimal required_margin = order_value / limits_.max_leverage;
    
    if (account.available_margin < required_margin) {
        return std::string("Insufficient margin");
    }
    
    // Match order
    Order mutable_order = order;
    auto matched = book.match(mutable_order);
    
    // Process trades
    for (const auto& match : matched) {
        Trade trade = {};
        trade.trade_id = trades_.size() + 1;
        trade.maker_order_id = match.order_id;
        trade.taker_order_id = mutable_order.order_id;
        trade.market = order.market;
        trade.side = order.side;
        trade.price = mutable_order.price;
        trade.quantity = match.filled_quantity;
        trade.timestamp = utils::get_current_timestamp();
        trade.maker_fee = calculate_maker_fee((Decimal)trade.price * trade.quantity);
        trade.taker_fee = calculate_taker_fee((Decimal)trade.price * trade.quantity);
        
        trades_.push_back(trade);
    }
    
    // Add remaining to book if limit order
    if (mutable_order.remaining_quantity > 0 && 
        mutable_order.order_type == OrderType::Limit) {
        book.add_order(mutable_order);
    }
    
    return mutable_order;
}

bool MatchingEngine::cancel_order(OrderID order_id, OrderBook& book) {
    std::lock_guard<std::mutex> lock(mutex_);
    book.remove_order(order_id);
    return true;
}

MarketData MatchingEngine::get_market_data(Market market) const {
    auto it = market_data_.find(market);
    if (it != market_data_.end()) {
        return it->second;
    }
    return {};
}

void MatchingEngine::update_market_data(Market market, const MarketData& data) {
    std::lock_guard<std::mutex> lock(mutex_);
    market_data_[market] = data;
}

std::vector<Trade> MatchingEngine::match_order(
    Order& order,
    OrderBook& book
) {
    return book.match(order);
}

Decimal MatchingEngine::calculate_maker_fee(Decimal notional) {
    return notional * Decimal(2) / Decimal(10000); // 0.02%
}

Decimal MatchingEngine::calculate_taker_fee(Decimal notional) {
    return notional * Decimal(5) / Decimal(10000); // 0.05%
}

// ============ PositionManager Implementation ============

PositionManager::PositionManager() {}

Position PositionManager::open_position(
    UserID user_id,
    Market market,
    PositionSide side,
    Quantity size,
    Price entry_price,
    Decimal margin
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    Position& pos = positions_[key];
    
    pos.user_id = user_id;
    pos.market = market;
    pos.side = side;
    pos.size = size;
    pos.open_size = size;
    pos.entry_price = entry_price;
    pos.status = PositionStatus::Open;
    pos.unrealized_pnl = 0;
    pos.opened_at = utils::get_current_timestamp();
    pos.updated_at = utils::get_current_timestamp();
    
    return pos;
}

Position PositionManager::close_position(
    UserID user_id,
    Market market,
    Quantity close_size,
    Price exit_price
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    auto it = positions_.find(key);
    
    if (it == positions_.end()) {
        return {};
    }
    
    Position& pos = it->second;
    
    // Calculate realized P&L
    if (pos.side == PositionSide::Long) {
        pos.realized_pnl += (Decimal)(exit_price - pos.entry_price) * close_size;
    } else {
        pos.realized_pnl += (Decimal)(pos.entry_price - exit_price) * close_size;
    }
    
    // Update position
    pos.size -= close_size;
    pos.open_size -= close_size;
    
    if (pos.size == 0) {
        pos.status = PositionStatus::Closed;
    }
    
    pos.updated_at = utils::get_current_timestamp();
    
    return pos;
}

void PositionManager::update_position(
    UserID user_id,
    Market market,
    const Position& update
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    positions_[key] = update;
}

std::optional<Position> PositionManager::get_position(UserID user_id, Market market) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    auto it = positions_.find(key);
    
    if (it != positions_.end()) {
        return it->second;
    }
    
    return std::nullopt;
}

std::vector<Position> PositionManager::get_user_positions(UserID user_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<Position> result;
    for (const auto& [key, pos] : positions_) {
        if (key.user_id == user_id && pos.status == PositionStatus::Open) {
            result.push_back(pos);
        }
    }
    
    return result;
}

void PositionManager::realize_pnl(UserID user_id, Market market, Decimal pnl) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    auto it = positions_.find(key);
    
    if (it != positions_.end()) {
        it->second.realized_pnl += pnl;
    }
    
    realized_pnl_[user_id] += pnl;
}

void PositionManager::update_unrealized_pnl(
    UserID user_id,
    Market market,
    Price mark_price
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    auto it = positions_.find(key);
    
    if (it != positions_.end()) {
        it->second.unrealized_pnl = it->second.calculate_pnl(mark_price);
        it->second.mark_price = mark_price;
        it->second.updated_at = utils::get_current_timestamp();
    }
}

bool PositionManager::check_margin(
    UserID user_id,
    Decimal required_margin
) {
    // Would check with AccountManager
    return true;
}

void PositionManager::liquidate_position(
    UserID user_id,
    Market market,
    Price liquidation_price
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    PositionKey key{user_id, market};
    auto it = positions_.find(key);
    
    if (it != positions_.end()) {
        it->second.status = PositionStatus::Liquidated;
        it->second.size = 0;
    }
}

// ============ AccountManager Implementation ============

AccountManager::AccountManager() {}

void AccountManager::deposit(UserID user_id, Decimal amount) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Account& account = get_or_create_account(user_id);
    account.balance += amount;
    account.last_updated = utils::get_current_timestamp();
}

bool AccountManager::withdraw(UserID user_id, Decimal amount) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Account& account = get_or_create_account(user_id);
    
    if (account.balance - account.frozen_balance < amount) {
        return false;
    }
    
    account.balance -= amount;
    account.last_updated = utils::get_current_timestamp();
    return true;
}

AccountRisk AccountManager::get_account_risk(UserID user_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    AccountRisk risk = {};
    risk.user_id = user_id;
    
    auto it = accounts_.find(user_id);
    if (it != accounts_.end()) {
        risk.total_collateral = it->second.balance;
        risk.realized_pnl_24h = it->second.total_realized_pnl;
    }
    
    return risk;
}

void AccountManager::update_margin(UserID user_id, Decimal delta) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Account& account = get_or_create_account(user_id);
    account.balance += delta;
    account.last_updated = utils::get_current_timestamp();
}

void AccountManager::freeze_margin(UserID user_id, Decimal amount) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Account& account = get_or_create_account(user_id);
    if (account.balance - account.frozen_balance >= amount) {
        account.frozen_balance += amount;
    }
}

void AccountManager::unfreeze_margin(UserID user_id, Decimal amount) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Account& account = get_or_create_account(user_id);
    account.frozen_balance = std::max(Decimal(0), account.frozen_balance - amount);
}

bool AccountManager::can_open_position(
    UserID user_id,
    Decimal required_margin,
    Decimal order_value
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = accounts_.find(user_id);
    if (it == accounts_.end()) return false;
    
    return (it->second.balance - it->second.frozen_balance) >= required_margin;
}

AccountManager::Account& AccountManager::get_or_create_account(UserID user_id) {
    auto it = accounts_.find(user_id);
    if (it != accounts_.end()) {
        return it->second;
    }
    
    return accounts_[user_id] = {user_id, 0, 0, 0, 0, utils::get_current_timestamp()};
}

// ============ FundingManager Implementation ============

FundingManager::FundingManager() {}

FundingRate FundingManager::calculate_funding_rate(
    Market market,
    Price index_price,
    Price mark_price
) {
    FundingRate rate = {};
    rate.market = market;
    rate.interest_rate = Decimal(3) / Decimal(10000); // 0.03% per day
    rate.premium_rate = calculate_premium(index_price, mark_price);
    rate.funding_rate = rate.interest_rate + rate.premium_rate;
    rate.last_funding_time = utils::get_current_timestamp();
    rate.next_funding_time = rate.last_funding_time + 8 * 3600; // Every 8 hours
    
    update_funding_rate(market, rate);
    return rate;
}

void FundingManager::process_funding_payment(
    PositionManager& position_mgr,
    UserID user_id,
    Market market,
    Decimal position_size,
    const FundingRate& funding
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Decimal payment = position_size * funding.funding_rate;
    position_mgr.realize_pnl(user_id, market, payment);
}

const FundingRate& FundingManager::get_funding_rate(Market market) const {
    static FundingRate empty_rate;
    auto it = funding_rates_.find(market);
    if (it != funding_rates_.end()) {
        return it->second;
    }
    return empty_rate;
}

void FundingManager::update_funding_rate(Market market, const FundingRate& rate) {
    std::lock_guard<std::mutex> lock(mutex_);
    funding_rates_[market] = rate;
}

Decimal FundingManager::calculate_premium(
    Price index_price,
    Price mark_price
) {
    if (index_price == 0) return 0;
    
    Decimal diff = (Decimal)(mark_price - index_price) * 10000 / index_price;
    
    // Clamp to [-0.03%, 0.03%]
    if (diff > Decimal(3)) diff = Decimal(3);
    if (diff < Decimal(-3)) diff = Decimal(-3);
    
    return diff / Decimal(10000);
}

// ============ OrderManager Implementation ============

OrderManager::OrderManager(MatchingEngine& matching_engine)
    : matching_engine_(matching_engine), next_order_id_(1) {}

std::variant<OrderID, std::string> OrderManager::place_order(
    const Order& order
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Validate order
    auto error = validate_order(order);
    if (error) {
        return *error;
    }
    
    OrderID order_id = next_order_id_++;
    Order new_order = order;
    new_order.order_id = order_id;
    new_order.created_at = utils::get_current_timestamp();
    new_order.updated_at = new_order.created_at;
    
    orders_[order_id] = new_order;
    user_orders_[order.user_id].push_back(order_id);
    
    return order_id;
}

bool OrderManager::cancel_order(OrderID order_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = orders_.find(order_id);
    if (it == orders_.end()) return false;
    
    it->second.status = OrderStatus::Cancelled;
    it->second.updated_at = utils::get_current_timestamp();
    
    return true;
}

bool OrderManager::modify_order(
    OrderID order_id,
    Price new_price,
    Quantity new_quantity
) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = orders_.find(order_id);
    if (it == orders_.end()) return false;
    
    it->second.price = new_price;
    it->second.quantity = new_quantity;
    it->second.updated_at = utils::get_current_timestamp();
    
    return true;
}

std::optional<Order> OrderManager::get_order(OrderID order_id) const {
    auto it = orders_.find(order_id);
    if (it != orders_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<Order> OrderManager::get_user_orders(UserID user_id) const {
    std::vector<Order> result;
    
    auto it = user_orders_.find(user_id);
    if (it != user_orders_.end()) {
        for (OrderID order_id : it->second) {
            auto order_it = orders_.find(order_id);
            if (order_it != orders_.end() && order_it->second.is_active()) {
                result.push_back(order_it->second);
            }
        }
    }
    
    return result;
}

void OrderManager::process_expired_orders() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    Timestamp now = utils::get_current_timestamp();
    
    for (auto& [order_id, order] : orders_) {
        if (order.is_active() && order.expires_at > 0 && now >= order.expires_at) {
            order.status = OrderStatus::Expired;
            order.updated_at = now;
        }
    }
}

size_t OrderManager::get_open_orders_count(UserID user_id) const {
    return get_user_orders(user_id).size();
}

std::optional<std::string> OrderManager::validate_order(const Order& order) {
    if (order.quantity <= 0) {
        return "Invalid quantity";
    }
    
    if (order.price <= 0) {
        return "Invalid price";
    }
    
    if (order.order_type == OrderType::Limit && order.price <= 0) {
        return "Limit order requires price";
    }
    
    if (order.order_type == OrderType::StopMarket || 
        order.order_type == OrderType::StopLimit) {
        if (order.stop_price <= 0) {
            return "Stop order requires stop price";
        }
    }
    
    return std::nullopt;
}

// ============ TradingEngine Implementation ============

TradingEngine::TradingEngine() 
    : running_(false),
      matching_engine_(new MatchingEngine()),
      position_manager_(new PositionManager()),
      account_manager_(new AccountManager()),
      order_manager_(new OrderManager(*matching_engine_)),
      liquidation_engine_(new LiquidationEngine(RiskLimits{})),
      funding_manager_(new FundingManager()) {}

TradingEngine::~TradingEngine() {
    stop();
}

void TradingEngine::initialize(const RiskLimits& limits) {
    limits_ = limits;
    liquidation_engine_ = std::make_unique<LiquidationEngine>(limits);
}

void TradingEngine::start() {
    running_ = true;
    processing_thread_ = std::thread(&TradingEngine::processing_loop, this);
}

void TradingEngine::stop() {
    running_ = false;
    if (processing_thread_.joinable()) {
        processing_thread_.join();
    }
}

std::variant<OrderID, std::string> TradingEngine::place_order(
    UserID user_id,
    Market market,
    Side side,
    OrderType order_type,
    Price price,
    Quantity quantity,
    const std::string& client_order_id
) {
    Order order = {};
    order.user_id = user_id;
    order.market = market;
    order.side = side;
    order.order_type = order_type;
    order.status = OrderStatus::Pending;
    order.price = price;
    order.quantity = quantity;
    order.remaining_quantity = quantity;
    order.client_order_id = client_order_id;
    order.created_at = utils::get_current_timestamp();
    
    return order_manager_->place_order(order);
}

bool TradingEngine::cancel_order(OrderID order_id) {
    return order_manager_->cancel_order(order_id);
}

std::variant<OrderID, std::string> TradingEngine::close_position(
    UserID user_id,
    Market market,
    Quantity quantity
) {
    auto position = position_manager_->get_position(user_id, market);
    if (!position) {
        return std::string("No position to close");
    }
    
    Side close_side = (position->side == PositionSide::Long) ? Side::Sell : Side::Buy;
    
    return place_order(user_id, market, close_side, OrderType::Market, 0, quantity);
}

bool TradingEngine::set_risk_controls(
    UserID user_id,
    Market market,
    Price stop_loss,
    Price take_profit
) {
    auto position = position_manager_->get_position(user_id, market);
    if (!position) return false;
    
    position->stop_loss = stop_loss;
    position->take_profit = take_profit;
    
    position_manager_->update_position(user_id, market, *position);
    return true;
}

std::optional<Position> TradingEngine::get_position(UserID user_id, Market market) {
    return position_manager_->get_position(user_id, market);
}

AccountRisk TradingEngine::get_account_risk(UserID user_id) {
    return account_manager_->get_account_risk(user_id);
}

MarketData TradingEngine::get_market_data(Market market) {
    return matching_engine_->get_market_data(market);
}

const std::vector<Trade>& TradingEngine::get_trades() const {
    return matching_engine_->get_trades();
}

void TradingEngine::on_market_data_update(Market market, const MarketData& data) {
    matching_engine_->update_market_data(market, data);
    update_positions();
}

std::vector<LiquidationResult> TradingEngine::liquidate_positions() {
    // Would check all positions and liquidate
    return {};
}

void TradingEngine::processing_loop() {
    while (running_) {
        try {
            // Process expired orders
            order_manager_->process_expired_orders();
            
            // Update positions with market data
            update_positions();
            
            // Check for liquidations
            liquidate_positions();
            
            // Sleep for 100ms
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        } catch (const std::exception& e) {
            // Log error and continue
        }
    }
}

void TradingEngine::update_positions() {
    // Update unrealized P&L for all positions
    // Would iterate through all positions and update with current mark prices
}

// ============ Utilities ============

namespace utils {
    Price price_from_double(double price) {
        return (Price)(price * 1e8);
    }
    
    double price_to_double(Price price) {
        return (double)price / 1e8;
    }
    
    Quantity quantity_from_double(double qty) {
        return (Quantity)(qty * 1e8);
    }
    
    double quantity_to_double(Quantity qty) {
        return (double)qty / 1e8;
    }
    
    Decimal decimal_from_double(double value, int decimals) {
        Decimal mult = 1;
        for (int i = 0; i < decimals; i++) mult *= 10;
        return (Decimal)(value * mult);
    }
    
    double decimal_to_double(Decimal value, int decimals) {
        Decimal div = 1;
        for (int i = 0; i < decimals; i++) div *= 10;
        return (double)(value / div);
    }
    
    Timestamp get_current_timestamp() {
        auto now = std::chrono::system_clock::now();
        return std::chrono::duration_cast<std::chrono::seconds>(
            now.time_since_epoch()
        ).count();
    }
    
    Timestamp timestamp_from_datetime(int year, int month, int day, int hour, int min, int sec) {
        // Simplified - would use proper time library
        return (Timestamp)year * 10000000000ULL + 
               (Timestamp)month * 100000000ULL + 
               (Timestamp)day * 1000000ULL + 
               (Timestamp)hour * 10000ULL + 
               (Timestamp)min * 100ULL + 
               (Timestamp)sec;
    }
}

} // namespace perpetual
} // namespace tiger
