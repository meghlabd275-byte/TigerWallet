#ifndef FARMING_SERVICE_HPP
#define FARMING_SERVICE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
#include <chrono>

using namespace std;
using namespace std::chrono;

struct FarmPool {
    string id;
    string token;
    string rewardToken;
    double apy;
    double tvl;
};

struct FarmPosition {
    string id;
    string farmId;
    double stakedAmount;
    double pendingRewards;
    uint64_t stakedAt;
};

class FarmingService {
private:
    mutex mutex_;
    unordered_map<string, FarmPool> farms_;
    unordered_map<string, vector<FarmPosition>> userPositions_;

public:
    FarmingService() {
        lock_guard<mutex> lock(mutex_);
        farms_["uniswap"] = {"uniswap", "UNI", "ETH", 0.25, 5000000};
        farms_["sushiswap"] = {"sushiswap", "SUSHI", "ETH", 0.20, 3000000};
        farms_["pancakeswap"] = {"pancakeswap", "CAKE", "BNB", 0.30, 4000000};
    }

    vector<FarmPool> getFarms() {
        lock_guard<mutex> lock(mutex_);
        vector<FarmPool> result;
        for (auto& p : farms_) result.push_back(p.second);
        return result;
    }

    bool stake(string farmId, double amount) {
        lock_guard<mutex> lock(mutex_);
        auto it = farms_.find(farmId);
        if (it != farms_.end()) {
            it->second.tvl += amount;
            return true;
        }
        return false;
    }

    bool unstake(string farmId, double amount) {
        return true;
    }

    double harvest(string farmId) {
        return 0.0;
    }
};

#endif
