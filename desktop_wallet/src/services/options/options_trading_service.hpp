/**
 * Options Trading Service - C++ Desktop Implementation
 * High-performance options pricing and trading with ultra-low latency
 * 
 * Features:
 * - Black-Scholes pricing model
 * - Greeks calculation (Delta, Gamma, Theta, Vega, Rho)
 * - Order matching engine
 * - Position management
 * - Risk calculation
 */

#ifndef OPTIONS_TRADING_SERVICE_HPP
#define OPTIONS_TRADING_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <mutex>
#include <memory>
#include <chrono>
#include <cmath>
#include <optional>
#include <functional>
#include <boost/multiprecision/cpp_dec_float.hpp>

using namespace std;
using namespace std::chrono;
using boost::multiprecision::cpp_dec_float_50;

// Constants
constexpr double RISK_FREE_RATE = 0.05;
constexpr double PI = 3.14159265358979323846;

// Option types
enum class OptionType { CALL, PUT };
enum class OptionStyle { EUROPEAN, AMERICAN };
enum class OrderSide { BUY, SELL };
enum class OrderStatus { PENDING, FILLED, CANCELLED, EXPIRED };
enum class PositionStatus { OPEN, CLOSED, LIQUIDATED };

// Forward declarations
struct Option;
struct Order;
struct Position;
struct Greeks;
struct PriceQuote;

// Price quote for underlying
struct PriceQuote {
    std::string symbol;
    double bid;
    double ask;
    double last;
    uint64_t timestamp;
    uint64_t volume;
};

// Greeks calculation result
struct Greeks {
    double delta;    // Price sensitivity
    double gamma;    // Delta sensitivity
    double theta;    // Time decay (per day)
    double vega;     // Volatility sensitivity
    double rho;     // Interest rate sensitivity
    
    Greeks() : delta(0), gamma(0), theta(0), vega(0), rho(0) {}
};

// Option contract
struct Option {
    std::string id;
    std::string underlying;
    double strikePrice;
    double spotPrice;
    double impliedVolatility;
    double timeToExpiry;  // In years
    OptionType type;
    OptionStyle style;
    double markPrice;
    double bidPrice;
    double askPrice;
    double openInterest;
    double volume24h;
    uint64_t lastUpdate;
    
    Option() : strikePrice(0), spotPrice(0), impliedVolatility(0), 
               timeToExpiry(0), type(OptionType::CALL), style(OptionStyle::EUROPEAN),
               markPrice(0), bidPrice(0), askPrice(0), openInterest(0), volume24h(0), lastUpdate(0) {}
};

// Order structure
struct Order {
    std::string id;
    std::string optionId;
    std::string userId;
    OrderSide side;
    OrderStatus status;
    int quantity;
    double price;
    double filledPrice;
    int filledQuantity;
    uint64_t createdAt;
    uint64_t updatedAt;
    std::string signature;
    
    Order() : side(OrderSide::BUY), status(OrderStatus::PENDING), 
              quantity(0), price(0), filledPrice(0), filledQuantity(0),
              createdAt(0), updatedAt(0) {}
};

// Position structure
struct Position {
    std::string id;
    std::string userId;
    std::string optionId;
    int quantity;
    double averagePrice;
    double realizedPnL;
    double unrealizedPnL;
    PositionStatus status;
    uint64_t openedAt;
    uint64_t closedAt;
    
    Position() : quantity(0), averagePrice(0), realizedPnL(0), 
                unrealizedPnL(0), status(PositionStatus::OPEN), 
                openedAt(0), closedAt(0) {}
};

// Statistics
struct OptionsStats {
    double totalVolume;
    double totalPremium;
    int openInterest;
    int totalTrades;
    double funding;
};

// Utility functions
inline double normalCDF(double x) {
    return 0.5 * std::erfc(-x / std::sqrt(2));
}

inline double normalPDF(double x) {
    return std::exp(-0.5 * x * x) / std::sqrt(2 * PI);
}

// Black-Scholes pricing
class BlackScholesModel {
public:
    /**
     * Calculate option price using Black-Scholes model
     * @param spot Current spot price
     * @param strike Strike price
     * @param timeToExpiry Time to expiry in years
     * @param volatility Implied volatility (decimal)
     * @param riskFreeRate Risk-free interest rate
     * @param type Option type (CALL/PUT)
     * @return Option price
     */
    static double calculatePrice(double spot, double strike, double timeToExpiry,
                                  double volatility, double riskFreeRate, OptionType type) {
        if (timeToExpiry <= 0 || volatility <= 0) {
            return 0.0;
        }
        
        double d1 = (std::log(spot / strike) + (riskFreeRate + 0.5 * volatility * volatility) * timeToExpiry) 
                    / (volatility * std::sqrt(timeToExpiry));
        double d2 = d1 - volatility * std::sqrt(timeToExpiry);
        
        if (type == OptionType::CALL) {
            return spot * normalCDF(d1) - strike * std::exp(-riskFreeRate * timeToExpiry) * normalCDF(d2);
        } else {
            return strike * std::exp(-riskFreeRate * timeToExpiry) * normalCDF(-d2) - spot * normalCDF(-d1);
        }
    }
    
    /**
     * Calculate all Greeks
     */
    static Greeks calculateGreeks(double spot, double strike, double timeToExpiry,
                                   double volatility, double riskFreeRate, OptionType type) {
        Greeks greeks;
        
        if (timeToExpiry <= 0 || volatility <= 0) {
            return greeks;
        }
        
        double sqrtT = std::sqrt(timeToExpiry);
        double d1 = (std::log(spot / strike) + (riskFreeRate + 0.5 * volatility * volatility) * timeToExpiry) 
                    / (volatility * sqrtT);
        double d2 = d1 - volatility * sqrtT;
        
        double nd1 = normalCDF(d1);
        double nd2 = normalCDF(d2);
        double n_d1 = normalPDF(d1);
        
        // Delta
        if (type == OptionType::CALL) {
            greeks.delta = nd1;
        } else {
            greeks.delta = nd1 - 1;
        }
        
        // Gamma (same for call and put)
        greeks.gamma = n_d1 / (spot * volatility * sqrtT);
        
        // Theta (per day)
        double thetaCommon = -spot * n_d1 * volatility / (2 * sqrtT);
        if (type == OptionType::CALL) {
            greeks.theta = (thetaCommon - riskFreeRate * strike * std::exp(-riskFreeRate * timeToExpiry) * nd2) / 365.0;
        } else {
            greeks.theta = (thetaCommon + riskFreeRate * strike * std::exp(-riskFreeRate * timeToExpiry) * normalCDF(-d2)) / 365.0;
        }
        
        // Vega (per 1% change, divided by 100)
        greeks.vega = spot * sqrtT * n_d1 / 100.0;
        
        // Rho (per 1% rate change, divided by 100)
        if (type == OptionType::CALL) {
            greeks.rho = strike * timeToExpiry * std::exp(-riskFreeRate * timeToExpiry) * nd2 / 100.0;
        } else {
            greeks.rho = -strike * timeToExpiry * std::exp(-riskFreeRate * timeToExpiry) * normalCDF(-d2) / 100.0;
        }
        
        return greeks;
    }
    
    /**
     * Calculate implied volatility using Newton-Raphson
     */
    static double calculateImpliedVolatility(double marketPrice, double spot, double strike,
                                              double timeToExpiry, double riskFreeRate,
                                              OptionType type, double precision = 0.0001,
                                              int maxIterations = 100) {
        double sigma = 0.5; // Initial guess
        
        for (int i = 0; i < maxIterations; i++) {
            double price = calculatePrice(spot, strike, timeToExpiry, sigma, riskFreeRate, type);
            Greeks greeks = calculateGreeks(spot, strike, timeToExpiry, sigma, riskFreeRate, type);
            
            double diff = marketPrice - price;
            
            if (std::abs(diff) < precision) {
                return sigma;
            }
            
            if (std::abs(greeks.vega) < 1e-10) {
                break;
            }
            
            sigma = sigma + diff / greeks.vega;
            sigma = std::max(0.01, std::min(sigma, 5.0)); // Bound volatility
        }
        
        return sigma;
    }
};

// Main Options Trading Service
class OptionsTradingService {
private:
    std::mutex mutex_;
    std::unordered_map<std::string, Option> options_;
    std::unordered_map<std::string, Order> orders_;
    std::unordered_map<std::string, Position> positions_;
    std::unordered_map<std::string, std::vector<Order>> userOrders_;
    std::unordered_map<std::string, std::vector<Position>> userPositions_;
    
    // Price feed callback
    std::function<void(const PriceQuote&)> priceFeedCallback_;
    
    // Database connection (would be PostgreSQL in production)
    void* dbConnection_;
    
public:
    OptionsTradingService() : dbConnection_(nullptr) {
        initializeOptions();
    }
    
    ~OptionsTradingService() {
        // Cleanup database connection
    }
    
    /**
     * Initialize available options
     */
    void initializeOptions() {
        std::lock_guard<std::mutex> lock(mutex_);
        
        // Initialize sample options (in production, load from database)
        std::vector<std::string> underlyings = {"BTC", "ETH", "SOL"};
        std::vector<double> strikes = {45000, 50000, 55000, 2500, 3000, 3500, 100, 150, 200};
        
        int idx = 0;
        for (const auto& underlying : underlyings) {
            for (size_t i = 0; i < strikes.size(); i++) {
                Option opt;
                opt.id = "OPT-" + std::to_string(idx++);
                opt.underlying = underlying;
                opt.strikePrice = strikes[i];
                opt.spotPrice = (underlying == "BTC") ? 50000 : (underlying == "ETH") ? 3000 : 150;
                opt.impliedVolatility = 0.5;
                opt.timeToExpiry = 0.25; // 3 months
                opt.type = (i % 2 == 0) ? OptionType::CALL : OptionType::PUT;
                opt.style = OptionStyle::EUROPEAN;
                opt.markPrice = BlackScholesModel::calculatePrice(
                    opt.spotPrice, opt.strikePrice, opt.timeToExpiry,
                    opt.impliedVolatility, RISK_FREE_RATE, opt.type);
                opt.bidPrice = opt.markPrice * 0.98;
                opt.askPrice = opt.markPrice * 1.02;
                opt.openInterest = 1000 + (idx * 100);
                opt.volume24h = 500 + (idx * 50);
                opt.lastUpdate = duration_cast<milliseconds>(
                    system_clock::now().time_since_epoch()).count();
                
                options_[opt.id] = opt;
            }
        }
    }
    
    /**
     * Get all available options
     */
    std::vector<Option> getOptions(const std::string& underlying = "") {
        std::lock_guard<std::mutex> lock(mutex_);
        std::vector<Option> result;
        
        for (const auto& pair : options_) {
            if (underlying.empty() || pair.second.underlying == underlying) {
                result.push_back(pair.second);
            }
        }
        
        return result;
    }
    
    /**
     * Get option by ID with Greeks
     */
    std::optional<std::pair<Option, Greeks>> getOptionWithGreeks(const std::string& optionId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = options_.find(optionId);
        if (it == options_.end()) {
            return std::nullopt;
        }
        
        const Option& opt = it->second;
        Greeks greeks = BlackScholesModel::calculateGreeks(
            opt.spotPrice, opt.strikePrice, opt.timeToExpiry,
            opt.impliedVolatility, RISK_FREE_RATE, opt.type);
        
        return std::make_pair(opt, greeks);
    }
    
    /**
     * Place order
     */
    Order placeOrder(const std::string& optionId, const std::string& userId,
                     OrderSide side, int quantity, double price) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        Order order;
        order.id = "ORD-" + std::to_string(duration_cast<milliseconds>(
            system_clock::now().time_since_epoch()).count());
        order.optionId = optionId;
        order.userId = userId;
        order.side = side;
        order.status = OrderStatus::PENDING;
        order.quantity = quantity;
        order.price = price;
        order.createdAt = duration_cast<milliseconds>(
            system_clock::now().time_since_epoch()).count();
        order.updatedAt = order.createdAt;
        
        // Check and fill if price matches
        auto optIt = options_.find(optionId);
        if (optIt != options_.end()) {
            const Option& opt = optIt->second;
            bool canFill = (side == OrderSide::BUY && price >= opt.askPrice) ||
                          (side == OrderSide::SELL && price <= opt.bidPrice);
            
            if (canFill) {
                order.status = OrderStatus::FILLED;
                order.filledPrice = (side == OrderSide::BUY) ? opt.askPrice : opt.bidPrice;
                order.filledQuantity = quantity;
                
                // Update or create position
                updatePosition(order);
            }
        }
        
        orders_[order.id] = order;
        userOrders_[userId].push_back(order);
        
        return order;
    }
    
    /**
     * Cancel order
     */
    bool cancelOrder(const std::string& orderId, const std::string& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = orders_.find(orderId);
        if (it == orders_.end() || it->second.userId != userId) {
            return false;
        }
        
        if (it->second.status == OrderStatus::PENDING) {
            it->second.status = OrderStatus::CANCELLED;
            it->second.updatedAt = duration_cast<milliseconds>(
                system_clock::now().time_since_epoch()).count();
            return true;
        }
        
        return false;
    }
    
    /**
     * Get user positions
     */
    std::vector<Position> getUserPositions(const std::string& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = userPositions_.find(userId);
        if (it == userPositions_.end()) {
            return {};
        }
        
        std::vector<Position> result;
        for (const auto& pos : it->second) {
            if (pos.status == PositionStatus::OPEN) {
                // Calculate unrealized PnL
                auto optIt = options_.find(pos.optionId);
                if (optIt != options_.end()) {
                    Position p = pos;
                    double currentPrice = optIt->second.markPrice;
                    p.unrealizedPnL = (currentPrice - pos.averagePrice) * pos.quantity * 
                                      ((pos.quantity > 0) ? 1 : -1);
                    result.push_back(p);
                }
            }
        }
        
        return result;
    }
    
    /**
     * Get user orders
     */
    std::vector<Order> getUserOrders(const std::string& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = userOrders_.find(userId);
        if (it == userOrders_.end()) {
            return {};
        }
        
        return it->second;
    }
    
    /**
     * Close position
     */
    bool closePosition(const std::string& positionId, const std::string& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto userIt = userPositions_.find(userId);
        if (userIt == userPositions_.end()) {
            return false;
        }
        
        for (auto& pos : userIt->second) {
            if (pos.id == positionId && pos.status == PositionStatus::OPEN) {
                auto optIt = options_.find(pos.optionId);
                if (optIt != options_.end()) {
                    // Calculate realized PnL
                    double closePrice = optIt->second.markPrice;
                    pos.realizedPnL = (closePrice - pos.averagePrice) * pos.quantity;
                }
                pos.status = PositionStatus::CLOSED;
                pos.closedAt = duration_cast<milliseconds>(
                    system_clock::now().time_since_epoch()).count();
                return true;
            }
        }
        
        return false;
    }
    
    /**
     * Get options statistics
     */
    OptionsStats getStats() {
        std::lock_guard<std::mutex> lock(mutex_);
        
        OptionsStats stats;
        stats.totalVolume = 0;
        stats.totalPremium = 0;
        stats.openInterest = 0;
        stats.totalTrades = 0;
        
        for (const auto& pair : options_) {
            stats.totalVolume += pair.second.volume24h * pair.second.markPrice;
            stats.openInterest += (int)pair.second.openInterest;
        }
        
        for (const auto& pair : orders_) {
            if (pair.second.status == OrderStatus::FILLED) {
                stats.totalTrades++;
                stats.totalPremium += pair.second.filledPrice * pair.second.filledQuantity;
            }
        }
        
        return stats;
    }
    
    /**
     * Calculate portfolio risk
     */
    double calculatePortfolioRisk(const std::string& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        double totalExposure = 0;
        auto positions = getUserPositions(userId);
        
        for (const auto& pos : positions) {
            auto optIt = options_.find(pos.optionId);
            if (optIt != options_.end()) {
                const Option& opt = optIt->second;
                Greeks greeks = BlackScholesModel::calculateGreeks(
                    opt.spotPrice, opt.strikePrice, opt.timeToExpiry,
                    opt.impliedVolatility, RISK_FREE_RATE, opt.type);
                
                // Value at Risk approximation
                double positionValue = pos.quantity * opt.markPrice;
                totalExposure += std::abs(positionValue * greeks.delta);
            }
        }
        
        return totalExposure;
    }
    
    /**
     * Set price feed callback
     */
    void setPriceFeedCallback(std::function<void(const PriceQuote&)> callback) {
        priceFeedCallback_ = callback;
    }
    
    /**
     * Update prices from feed
     */
    void updatePrice(const PriceQuote& quote) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        for (auto& pair : options_) {
            if (pair.second.underlying == quote.symbol) {
                pair.second.spotPrice = quote.last;
                pair.second.markPrice = BlackScholesModel::calculatePrice(
                    quote.last, pair.second.strikePrice, pair.second.timeToExpiry,
                    pair.second.impliedVolatility, RISK_FREE_RATE, pair.second.type);
                pair.second.bidPrice = pair.second.markPrice * 0.98;
                pair.second.askPrice = pair.second.markPrice * 1.02;
                pair.second.lastUpdate = duration_cast<milliseconds>(
                    system_clock::now().time_since_epoch()).count();
            }
        }
    }

private:
    void updatePosition(const Order& order) {
        std::string key = order.userId + "-" + order.optionId;
        
        auto& userPos = userPositions_[order.userId];
        
        // Find existing position
        for (auto& pos : userPos) {
            if (pos.optionId == order.optionId && pos.status == PositionStatus::OPEN) {
                // Update existing position
                int totalQty = pos.quantity + 
                    ((order.side == OrderSide::BUY) ? order.filledQuantity : -order.filledQuantity);
                
                if (totalQty == 0) {
                    pos.status = PositionStatus::CLOSED;
                    pos.closedAt = duration_cast<milliseconds>(
                        system_clock::now().time_since_epoch()).count();
                } else {
                    // Update average price
                    double totalCost = pos.averagePrice * std::abs(pos.quantity) + 
                                      order.filledPrice * order.filledQuantity;
                    pos.averagePrice = totalCost / std::abs(totalQty);
                    pos.quantity = totalQty;
                }
                return;
            }
        }
        
        // Create new position
        Position newPos;
        newPos.id = "POS-" + std::to_string(duration_cast<milliseconds>(
            system_clock::now().time_since_epoch()).count());
        newPos.userId = order.userId;
        newPos.optionId = order.optionId;
        newPos.quantity = (order.side == OrderSide::BUY) ? order.filledQuantity : -order.filledQuantity;
        newPos.averagePrice = order.filledPrice;
        newPos.status = PositionStatus::OPEN;
        newPos.openedAt = order.updatedAt;
        
        userPos.push_back(newPos);
        positions_[newPos.id] = newPos;
    }
};

#endif // OPTIONS_TRADING_SERVICE_HPP
