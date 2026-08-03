#include "order_book.h"
#include <iostream>
#include <thread>
#include <vector>
#include <random>
#include <chrono>

using namespace tigerwallet;

int main() {
    std::cout << "TigerWallet High-Performance Matching Engine\n";
    std::cout << "=============================================\n\n";
    
    // Create matching engine
    MatchingEngine engine;
    
    // Add order books for different trading pairs
    engine.add_order_book("BTC/USDT");
    engine.add_order_book("ETH/USDT");
    
    std::cout << "Order books initialized for BTC/USDT and ETH/USDT\n\n";
    
    // Simulate high-frequency trading
    auto start = std::chrono::high_resolution_clock::now();
    
    const int num_orders = 100000;
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_real_distribution<> price_dist(42000, 45000);
    std::uniform_real_distribution<> qty_dist(0.001, 1.0);
    
    for (int i = 0; i < num_orders; i++) {
        double price = price_dist(gen);
        double qty = qty_dist(gen);
        OrderSide side = (i % 2 == 0) ? OrderSide::BUY : OrderSide::SELL;
        
        engine.process_order("user_" + std::to_string(i % 100),
                           "BTC/USDT", side, price, qty);
    }
    
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    
    std::cout << "Processed " << num_orders << " orders in " << duration.count() << "ms\n";
    std::cout << "Throughput: " << (num_orders * 1000 / duration.count()) << " orders/sec\n\n";
    
    // Get market state
    auto state = engine.get_market_state("BTC/USDT");
    std::cout << "Market State:\n";
    std::cout << "  Best Bid: $" << state.best_bid << "\n";
    std::cout << "  Best Ask: $" << state.best_ask << "\n";
    std::cout << "  Mid Price: $" << state.mid_price << "\n";
    std::cout << "  Spread: $" << state.spread << "\n";
    
    return 0;
}
