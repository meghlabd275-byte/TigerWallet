#ifndef FIAT_RAMP_SERVICE_HPP
#define FIAT_RAMP_SERVICE_HPP

#include <string>
#include <unordered_map>
#include <mutex>
#include <chrono>

using namespace std;
using namespace std::chrono;

struct FiatOrder {
    string id;
    string userId;
    string type;  // BUY or SELL
    double amount;
    string currency;
    string crypto;
    string status;
    string paymentMethod;
    uint64_t createdAt;
    uint64_t completedAt;
};

class FiatRampService {
private:
    mutex mutex_;
    unordered_map<string, FiatOrder> orders_;
    uint64_t orderCounter_;

public:
    FiatRampService() : orderCounter_(0) {}

    string buyCrypto(string userId, double amount, string currency, string crypto, string paymentMethod) {
        lock_guard<mutex> lock(mutex_);
        string orderId = "FIAT-" + to_string(++orderCounter_);
        
        FiatOrder order;
        order.id = orderId;
        order.userId = userId;
        order.type = "BUY";
        order.amount = amount;
        order.currency = currency;
        order.crypto = crypto;
        order.paymentMethod = paymentMethod;
        order.status = "PENDING";
        order.createdAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        
        orders_[orderId] = order;
        return orderId;
    }

    string sellCrypto(string userId, double amount, string crypto, string currency, string bankAccount) {
        lock_guard<mutex> lock(mutex_);
        string orderId = "FIAT-" + to_string(++orderCounter_);
        
        FiatOrder order;
        order.id = orderId;
        order.userId = userId;
        order.type = "SELL";
        order.amount = amount;
        order.currency = currency;
        order.crypto = crypto;
        order.paymentMethod = bankAccount;
        order.status = "PENDING";
        order.createdAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        
        orders_[orderId] = order;
        return orderId;
    }

    FiatOrder getOrderStatus(string orderId) {
        lock_guard<mutex> lock(mutex_);
        auto it = orders_.find(orderId);
        if (it != orders_.end()) {
            return it->second;
        }
        return {};
    }

    bool completeOrder(string orderId) {
        lock_guard<mutex> lock(mutex_);
        auto it = orders_.find(orderId);
        if (it != orders_.end()) {
            it->second.status = "COMPLETED";
            it->second.completedAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
            return true;
        }
        return false;
    }
};

#endif
