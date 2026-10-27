// Ultra-Low Latency Trading Engine - C++ Implementation
// Designed for microsecond-level trading performance

#ifndef TRADING_ENGINE_ULTRA_HPP
#define TRADING_ENGINE_ULTRA_HPP

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <string>
#include <unordered_map>
#include <vector>
#include <array>
#include <cstdint>
#include <cstring>
#include <cstdio>

// ============================================================================
// High-Performance Types
// ============================================================================

using Timestamp = int64_t;
using OrderID = uint64_t;
using UserID = uint64_t;
using Price = double;
using Quantity = double;

// Constants for ultra-low latency.
// FastHashMap masks its index with (N - 1), so N must be a power of two; 50000
// is not, and the static assertion in the class rejected it.
constexpr size_t MAX_PAIRS = 65536;
constexpr size_t MAX_ORDERS = 1048576;
constexpr size_t CACHE_LINE_SIZE = 64;

// ============================================================================
// Enums
// ============================================================================

enum class OrderType : uint8_t {
    MARKET = 0,
    LIMIT = 1,
    STOP = 2,
    STOP_LIMIT = 3
};

enum class OrderSide : uint8_t {
    BUY = 0,
    SELL = 1
};

enum class OrderStatus : uint8_t {
    PENDING = 0,
    OPEN = 1,
    FILLED = 2,
    PARTIALLY_FILLED = 3,
    CANCELLED = 4,
    REJECTED = 5
};

enum class MarginMode : uint8_t {
    CROSS = 0,
    ISOLATED = 1
};

enum class PositionSide : uint8_t {
    LONG = 0,
    SHORT = 1
};

// ============================================================================
// Cache-Aligned Structures
// ============================================================================

struct TradingPair {
    char symbol[32];
    char base[16];
    char quote[16];
    std::atomic<Price> price;
    std::atomic<Price> high24h;
    std::atomic<Price> low24h;
    std::atomic<Price> volume24h;
    std::atomic<double> change24h;
    bool isPreInstalled;
    bool status;
    double minOrderSize;
    double maxOrderSize;
    double makerFee;
    double takerFee;
    
    // Pad to a cache line. The hand-written size subtracted 124 from a
    // 64-byte constant, so the bound underflowed and the array size was
    // ~1.8e19. The padding is now derived from the members that precede it.
    char _padding[CACHE_LINE_SIZE > (sizeof(symbol) + sizeof(base) + sizeof(quote) +
        sizeof(std::atomic<Price>) * 4 + sizeof(std::atomic<double>) +
        sizeof(bool) * 2 + sizeof(double) * 4)
        ? CACHE_LINE_SIZE - (sizeof(symbol) + sizeof(base) + sizeof(quote) +
            sizeof(std::atomic<Price>) * 4 + sizeof(std::atomic<double>) +
            sizeof(bool) * 2 + sizeof(double) * 4)
        : 1];
} __attribute__((aligned(CACHE_LINE_SIZE)));

struct alignas(CACHE_LINE_SIZE) Order {
    OrderID id;
    UserID userId;
    char symbol[32];
    OrderSide side;
    OrderType type;
    Quantity size;
    Quantity filled;
    Price price;
    Price stopPrice;
    int leverage;
    MarginMode marginMode;
    OrderStatus status;
    Timestamp createTime;
    Timestamp updateTime;
    
    char _padding[CACHE_LINE_SIZE > (sizeof(OrderID) + sizeof(UserID) + sizeof(symbol) +
        sizeof(OrderSide) + sizeof(OrderType) + sizeof(Quantity) * 2 + sizeof(Price) * 2 +
        sizeof(int) + sizeof(MarginMode) + sizeof(OrderStatus) + sizeof(Timestamp) * 2)
        ? CACHE_LINE_SIZE - (sizeof(OrderID) + sizeof(UserID) + sizeof(symbol) +
            sizeof(OrderSide) + sizeof(OrderType) + sizeof(Quantity) * 2 + sizeof(Price) * 2 +
            sizeof(int) + sizeof(MarginMode) + sizeof(OrderStatus) + sizeof(Timestamp) * 2)
        : 1];
};

struct alignas(CACHE_LINE_SIZE) Position {
    OrderID id;
    UserID userId;
    char symbol[32];
    PositionSide side;
    Quantity size;
    Price entryPrice;
    Price markPrice;
    int leverage;
    Quantity margin;
    MarginMode marginMode;
    double pnl;
    double pnlPercent;
    Price liquidationPrice;
    Timestamp openTime;
    
    char _padding[CACHE_LINE_SIZE > (sizeof(OrderID) + sizeof(UserID) + sizeof(symbol) +
        sizeof(PositionSide) + sizeof(Quantity) * 2 + sizeof(Price) * 3 + sizeof(int) +
        sizeof(MarginMode) + sizeof(double) * 2 + sizeof(Timestamp))
        ? CACHE_LINE_SIZE - (sizeof(OrderID) + sizeof(UserID) + sizeof(symbol) +
            sizeof(PositionSide) + sizeof(Quantity) * 2 + sizeof(Price) * 3 + sizeof(int) +
            sizeof(MarginMode) + sizeof(double) * 2 + sizeof(Timestamp))
        : 1];
};

// ============================================================================
// Lock-Free Ring Buffer for Order Processing
// ============================================================================

template<typename T, size_t N>
class LockFreeRingBuffer {
    static_assert((N & (N - 1)) == 0, "Size must be power of 2");
    
public:
    LockFreeRingBuffer()
        : buffer_(std::make_unique<std::array<T, N>>()), head_(0), tail_(0) {}

    LockFreeRingBuffer(const LockFreeRingBuffer&) = delete;
    LockFreeRingBuffer& operator=(const LockFreeRingBuffer&) = delete;
    
    bool push(const T& item) {
        size_t head = head_.load(std::memory_order_relaxed);
        size_t next_head = (head + 1) & (N - 1);
        if (next_head == tail_.load(std::memory_order_acquire)) {
            return false;
        }
        (*buffer_)[head] = item;
        head_.store(next_head, std::memory_order_release);
        return true;
    }
    
    bool pop(T& item) {
        size_t tail = tail_.load(std::memory_order_relaxed);
        if (tail == head_.load(std::memory_order_acquire)) {
            return false;
        }
        item = (*buffer_)[tail];
        tail_.store((tail + 1) & (N - 1), std::memory_order_release);
        return true;
    }
    
    size_t size() const {
        size_t head = head_.load(std::memory_order_relaxed);
        size_t tail = tail_.load(std::memory_order_relaxed);
        return (head - tail) & (N - 1);
    }
    
    bool empty() const {
        return head_.load(std::memory_order_relaxed) == tail_.load(std::memory_order_relaxed);
    }

private:
    // 65536 slots of a cache-line-aligned Order is 4 MB. Held by value this
    // became part of the engine object and, together with the hash tables,
    // pushed it past the default 8 MB stack, so constructing the engine
    // crashed. On the heap the object stays small.
    std::unique_ptr<std::array<T, N>> buffer_;
    std::atomic<size_t> head_;
    std::atomic<size_t> tail_;
};

// ============================================================================
// Ultra-Fast Hash Map using open addressing
// ============================================================================

template<typename Key, typename Value, size_t N>
class FastHashMap {
    // N must be a power of two because the probe index is masked with (N - 1).
    static_assert((N & (N - 1)) == 0, "Size must be power of two");

public:
    FastHashMap()
        : keys_(std::make_unique<std::array<Key, N>>()),
          values_(std::make_unique<std::array<Value, N>>()),
          occupied_(std::make_unique<std::array<bool, N>>()),
          tombstones_(std::make_unique<std::array<bool, N>>()) {
        // Occupancy is tracked in its own bitmap. The previous version used
        // `keys_[pos] == 0` to mean "empty", which cannot work when Key is
        // std::string: there is no integer zero to compare against, so the
        // class did not compile for the string-keyed instantiations it is used
        // with. A bitmap works for any key type.
        occupied_->fill(false);
        tombstones_->fill(false);
    }

    FastHashMap(const FastHashMap&) = delete;
    FastHashMap& operator=(const FastHashMap&) = delete;

    bool insert(const Key& key, const Value& value) {
        const size_t idx = hash(key);
        for (size_t i = 0; i < kMaxProbe; ++i) {
            const size_t pos = (idx + i) & (N - 1);
            if (!(*occupied_)[pos]) {
                (*keys_)[pos] = key;
                (*values_)[pos] = value;
                (*occupied_)[pos] = true;
                ++count_;
                return true;
            }
            if ((*keys_)[pos] == key) {
                // Overwrite an existing key rather than adding a second entry.
                (*values_)[pos] = value;
                return true;
            }
        }
        return false; // The probe window is full.
    }

    Value* find(const Key& key) {
        const size_t pos = findPos(key);
        return pos == npos ? nullptr : &(*values_)[pos];
    }

    const Value* find(const Key& key) const {
        const size_t pos = findPos(key);
        return pos == npos ? nullptr : &(*values_)[pos];
    }

    bool erase(const Key& key) {
        const size_t pos = findPos(key);
        if (pos == npos) return false;
        // Marking the slot free would break probe chains for keys that
        // collided with this one, so the slot keeps its key and is tombstoned.
        (*occupied_)[pos] = false;
        (*tombstones_)[pos] = true;
        --count_;
        return true;
    }

    size_t size() const { return count_; }

private:
    static constexpr size_t npos = static_cast<size_t>(-1);
    static constexpr size_t kMaxProbe = 8;

    size_t hash(const Key& key) const {
        // The original mixed the key with a 64-bit constant, which only
        // applies to integral keys. std::hash covers the string keys used here.
        return std::hash<Key>{}(key) & (N - 1);
    }

    size_t findPos(const Key& key) const {
        const size_t idx = hash(key);
        for (size_t i = 0; i < kMaxProbe; ++i) {
            const size_t pos = (idx + i) & (N - 1);
            if (!(*occupied_)[pos] && !(*tombstones_)[pos]) {
                return npos; // A never-used slot ends the probe chain.
            }
            if ((*occupied_)[pos] && (*keys_)[pos] == key) {
                return pos;
            }
        }
        return npos;
    }

    // The four tables are ~32 MB at the sizes this engine uses (65536 string
    // keys plus a million order slots). Held by value they became part of the
    // engine object, which then could not be constructed on an 8 MB stack. On
    // the heap the engine object stays small and the tables are sized at
    // runtime.
    std::unique_ptr<std::array<Key, N>> keys_;
    std::unique_ptr<std::array<Value, N>> values_;
    std::unique_ptr<std::array<bool, N>> occupied_;
    std::unique_ptr<std::array<bool, N>> tombstones_;
    size_t count_ = 0;
};

// ============================================================================
// Trading Engine Core
// ============================================================================

class UltraLowLatencyTradingEngine {
public:
    UltraLowLatencyTradingEngine();
    ~UltraLowLatencyTradingEngine();
    
    // Trading pair management
    void addTradingPair(const char* symbol, const char* base, const char* quote,
                       Price price, bool isPreInstalled);
    TradingPair* getTradingPair(const char* symbol);
    std::vector<TradingPair*> getPreInstalledPairs();
    size_t getTotalPairs() const { return pairsCount_.load(); }
    
    // Order operations - Ultra low latency
    OrderID createOrder(UserID userId, const char* symbol, OrderSide side,
                       OrderType type, Quantity size, Price price, int leverage);
    bool cancelOrder(OrderID orderId);
    bool modifyOrder(OrderID orderId, Price newPrice, Quantity newSize);
    
    // Position management
    Position* getPosition(UserID userId, const char* symbol);
    std::vector<Position*> getUserPositions(UserID userId);

    // Opens or replaces the position for this user and symbol. Without this
    // there was no way for a position to exist, so getPosition always returned
    // nullptr and getUserPositions always returned an empty vector.
    Position* openPosition(UserID userId, const char* symbol, PositionSide side,
                            Quantity size, Price entryPrice, int leverage);
    
    // Market data - Lock-free reads
    Price getPrice(const char* symbol) const;
    void updatePrice(const char* symbol, Price newPrice);
    
    // Order matching - High frequency
    void processOrderBook();
    
    // PnL calculation
    double calculatePnL(Position* position) const;
    
    // 50,000+ pairs support
    void initializePairs();
    
private:
    // Lock-free data structures for ultra-low latency
    FastHashMap<std::string, TradingPair*, MAX_PAIRS> pairs_;
    FastHashMap<OrderID, Order*, MAX_ORDERS> orders_;
    FastHashMap<std::string, Position*, MAX_PAIRS> positions_;
    
    // Order processing ring buffer
    LockFreeRingBuffer<Order, 65536> orderQueue_;
    
    // FastHashMap cannot be iterated, so ownership and the pre-installed
    // subset are tracked alongside it.
    std::vector<TradingPair*> ownedPairs_;
    std::vector<TradingPair*> preInstalledPairs_;
    std::vector<Order*> ownedOrders_;
    std::vector<Position*> ownedPositions_;

    std::atomic<size_t> pairsCount_;
    std::atomic<size_t> ordersCount_;
    
    std::atomic<OrderID> nextOrderId_;
    
    // High-resolution timer
    Timestamp getCurrentTimestamp() const {
        auto now = std::chrono::high_resolution_clock::now();
        return std::chrono::duration_cast<std::chrono::nanoseconds>(
            now.time_since_epoch()
        ).count();
    }
};

// ============================================================================
// Inline Implementation
// ============================================================================

inline UltraLowLatencyTradingEngine::UltraLowLatencyTradingEngine() 
    : pairsCount_(0), ordersCount_(0), nextOrderId_(1) {
    initializePairs();
}

inline UltraLowLatencyTradingEngine::~UltraLowLatencyTradingEngine() {
    // The engine owns every TradingPair and Order it allocated. The previous
    // empty destructor leaked all of them.
    for (TradingPair* pair : ownedPairs_) {
        delete pair;
    }
    ownedPairs_.clear();
    preInstalledPairs_.clear();
    for (Order* order : ownedOrders_) {
        delete order;
    }
    ownedOrders_.clear();
    for (Position* position : ownedPositions_) {
        delete position;
    }
    ownedPositions_.clear();
}

inline void UltraLowLatencyTradingEngine::initializePairs() {
    // This used to seed 50,000 pairs: 40 real-looking ones with hardcoded
    // prices, then 49,800 synthetic "TOKEN<n>/USDT" pairs priced from the loop
    // counter. getTotalPairs() then reported 50,000 and getPreInstalledPairs()
    // reported nothing. A pair is now only present once something with a real
    // market has registered it through addTradingPair, so the count reflects
    // pairs that actually exist.
    //
    // Nothing is pre-registered here on purpose. A caller that wants a pair
    // supplies its symbol and a price obtained from a real source.
}

inline void UltraLowLatencyTradingEngine::addTradingPair(
    const char* symbol, const char* base, const char* quote,
    Price price, bool isPreInstalled) {
    
    TradingPair* pair = new TradingPair();
    strncpy(pair->symbol, symbol, sizeof(pair->symbol) - 1);
    strncpy(pair->base, base, sizeof(pair->base) - 1);
    strncpy(pair->quote, quote, sizeof(pair->quote) - 1);
    
    pair->price.store(price, std::memory_order_relaxed);
    pair->high24h.store(price * 1.05, std::memory_order_relaxed);
    pair->low24h.store(price * 0.95, std::memory_order_relaxed);
    pair->volume24h.store(0, std::memory_order_relaxed);
    pair->change24h.store(0, std::memory_order_relaxed);
    pair->isPreInstalled = isPreInstalled;
    pair->status = true;
    pair->minOrderSize = 0.001;
    pair->maxOrderSize = 1000000;
    pair->makerFee = 0.02;
    pair->takerFee = 0.04;
    
    if (!pairs_.insert(std::string(symbol), pair)) {
        // The table is full, so the pair is not registered. Do not leak it and
        // do not count it.
        delete pair;
        return;
    }
    ownedPairs_.push_back(pair);
    if (isPreInstalled) {
        preInstalledPairs_.push_back(pair);
    }
    pairsCount_.fetch_add(1, std::memory_order_relaxed);
}

inline TradingPair* UltraLowLatencyTradingEngine::getTradingPair(const char* symbol) {
    // find() returns a pointer to the stored value, and the stored value is
    // itself a TradingPair*, so this is one dereference, not two.
    TradingPair** slot = pairs_.find(std::string(symbol));
    return slot ? *slot : nullptr;
}

inline std::vector<TradingPair*> UltraLowLatencyTradingEngine::getPreInstalledPairs() {
    std::vector<TradingPair*> result;
    // addTradingPair records pre-installed pairs here, because FastHashMap has
    // no iteration interface and the previous version returned an empty vector
    // unconditionally.
    result.reserve(preInstalledPairs_.size());
    for (TradingPair* pair : preInstalledPairs_) {
        result.push_back(pair);
    }
    return result;
}

inline OrderID UltraLowLatencyTradingEngine::createOrder(
    UserID userId, const char* symbol, OrderSide side,
    OrderType type, Quantity size, Price price, int leverage) {
    
    OrderID orderId = nextOrderId_.fetch_add(1, std::memory_order_relaxed);
    
    Order* order = new Order();
    order->id = orderId;
    order->userId = userId;
    strncpy(order->symbol, symbol, sizeof(order->symbol) - 1);
    order->side = side;
    order->type = type;
    order->size = size;
    order->filled = 0;
    order->price = price;
    order->stopPrice = 0;
    order->leverage = leverage;
    order->marginMode = MarginMode::CROSS;
    order->status = OrderStatus::OPEN;
    order->createTime = getCurrentTimestamp();
    order->updateTime = order->createTime;
    
    if (!orders_.insert(orderId, order)) {
        // The order table is full. Report failure rather than returning an id
        // for an order that was never stored.
        delete order;
        return 0;
    }
    ownedOrders_.push_back(order);
    ordersCount_.fetch_add(1, std::memory_order_relaxed);
    
    return orderId;
}

inline bool UltraLowLatencyTradingEngine::cancelOrder(OrderID orderId) {
    Order** slot = orders_.find(orderId);
    Order* order = slot ? *slot : nullptr;
    if (order && order->status == OrderStatus::OPEN) {
        order->status = OrderStatus::CANCELLED;
        order->updateTime = getCurrentTimestamp();
        return true;
    }
    return false;
}

inline bool UltraLowLatencyTradingEngine::modifyOrder(OrderID orderId, Price newPrice, Quantity newSize) {
    Order** slot = orders_.find(orderId);
    Order* order = slot ? *slot : nullptr;
    if (order && order->status == OrderStatus::OPEN) {
        order->price = newPrice;
        order->size = newSize;
        order->updateTime = getCurrentTimestamp();
        return true;
    }
    return false;
}

inline Position* UltraLowLatencyTradingEngine::getPosition(UserID userId, const char* symbol) {
    char key[64];
    snprintf(key, sizeof(key), "%lu_%s",
             static_cast<unsigned long>(userId), symbol);
    Position** slot = positions_.find(std::string(key));
    return slot ? *slot : nullptr;
}

inline std::vector<Position*> UltraLowLatencyTradingEngine::getUserPositions(UserID userId) {
    std::vector<Position*> result;
    for (Position* position : ownedPositions_) {
        if (position == nullptr) continue;
        // Compare the recorded owner. Rebuilding the key from the requested
        // userId and the position's symbol would match every position, since
        // the requested id is the same on both sides of the comparison.
        if (position->userId == userId) {
            result.push_back(position);
        }
    }
    return result;
}

inline Position* UltraLowLatencyTradingEngine::openPosition(
    UserID userId, const char* symbol, PositionSide side,
    Quantity size, Price entryPrice, int leverage) {

    if (symbol == nullptr || size <= 0 || entryPrice <= 0) return nullptr;

    char key[64];
    snprintf(key, sizeof(key), "%lu_%s",
             static_cast<unsigned long>(userId), symbol);
    const std::string positionKey(key);

    Position** existing = positions_.find(positionKey);
    if (existing != nullptr && *existing != nullptr) {
        // Replace the existing position's figures rather than creating a
        // second one under the same key, which the table would reject.
        Position* position = *existing;
        position->side = side;
        position->size = size;
        position->entryPrice = entryPrice;
        position->markPrice = entryPrice;
        position->leverage = leverage;
        position->margin = (size * entryPrice) / (leverage > 0 ? leverage : 1);
        position->openTime = getCurrentTimestamp();
        return position;
    }

    Position* position = new Position();
    position->id = nextOrderId_.fetch_add(1, std::memory_order_relaxed);
    position->userId = userId;
    strncpy(position->symbol, symbol, sizeof(position->symbol) - 1);
    position->symbol[sizeof(position->symbol) - 1] = '\0';
    position->side = side;
    position->size = size;
    position->entryPrice = entryPrice;
    position->markPrice = entryPrice;
    position->leverage = leverage;
    position->margin = (size * entryPrice) / (leverage > 0 ? leverage : 1);
    position->marginMode = MarginMode::CROSS;
    position->pnl = 0;
    position->pnlPercent = 0;
    position->liquidationPrice = 0;
    position->openTime = getCurrentTimestamp();

    if (!positions_.insert(positionKey, position)) {
        delete position;
        return nullptr;
    }
    ownedPositions_.push_back(position);
    return position;
}

inline Price UltraLowLatencyTradingEngine::getPrice(const char* symbol) const {
    const TradingPair* const* slot = pairs_.find(std::string(symbol));
    const TradingPair* pair = slot ? *slot : nullptr;
    if (pair) {
        return pair->price.load(std::memory_order_acquire);
    }
    return 0;
}

inline void UltraLowLatencyTradingEngine::updatePrice(const char* symbol, Price newPrice) {
    TradingPair** slot = pairs_.find(std::string(symbol));
    TradingPair* pair = slot ? *slot : nullptr;
    if (pair) {
        Price oldPrice = pair->price.load(std::memory_order_relaxed);
        pair->price.store(newPrice, std::memory_order_release);
        
        // Update high/low
        Price high = pair->high24h.load(std::memory_order_relaxed);
        Price low = pair->low24h.load(std::memory_order_relaxed);
        
        if (newPrice > high) pair->high24h.store(newPrice, std::memory_order_relaxed);
        if (newPrice < low) pair->low24h.store(newPrice, std::memory_order_relaxed);
        
        // Update change
        double change = ((newPrice - oldPrice) / oldPrice) * 100.0;
        pair->change24h.store(change, std::memory_order_relaxed);
    }
}

inline void UltraLowLatencyTradingEngine::processOrderBook() {
    // High-frequency order book processing
    Order order;
    while (orderQueue_.pop(order)) {
        // Process order immediately
        // This would match against existing orders in the book
    }
}

inline double UltraLowLatencyTradingEngine::calculatePnL(Position* position) const {
    if (!position) return 0;
    
    if (position->side == PositionSide::LONG) {
        return (position->markPrice - position->entryPrice) * position->size;
    } else {
        return (position->entryPrice - position->markPrice) * position->size;
    }
}

#endif // TRADING_ENGINE_ULTRA_HPP
