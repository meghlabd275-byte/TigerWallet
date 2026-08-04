#ifndef LIQUID_STAKING_SERVICE_HPP
#define LIQUID_STAKING_SERVICE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
#include <chrono>

using namespace std;
using namespace std::chrono;

struct LiquidPool {
    string id;
    string token;
    string liquidToken;
    double apy;
    double totalStaked;
    uint64_t lastUpdate;
};

struct LiquidPosition {
    string id;
    string poolId;
    double stakedAmount;
    double liquidTokenAmount;
    double pendingRewards;
    uint64_t stakedAt;
};

class LiquidStakingService {
private:
    mutex mutex_;
    unordered_map<string, LiquidPool> pools_;
    unordered_map<string, vector<LiquidPosition>> userPositions_;

public:
    LiquidStakingService() {
        lock_guard<mutex> lock(mutex_);
        pools_["eth"] = {"eth", "ETH", "stETH", 0.05, 10000000, duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count()};
        pools_["sol"] = {"sol", "SOL", "mSOL", 0.08, 5000000, duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count()};
    }

    vector<LiquidPool> getPools() {
        lock_guard<mutex> lock(mutex_);
        vector<LiquidPool> result;
        for (auto& p : pools_) result.push_back(p.second);
        return result;
    }

    bool stake(string poolId, double amount) {
        lock_guard<mutex> lock(mutex_);
        auto it = pools_.find(poolId);
        if (it != pools_.end()) {
            it->second.totalStaked += amount;
            return true;
        }
        return false;
    }

    bool unstake(string poolId, double amount) {
        lock_guard<mutex> lock(mutex_);
        return true;
    }

    double claimRewards(string poolId) {
        lock_guard<mutex> lock(mutex_);
        return 0.0;
    }
};

#endif
