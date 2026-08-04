#ifndef LAUNCHPAD_SERVICE_HPP
#define LAUNCHPAD_SERVICE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
#include <chrono>

using namespace std;
using namespace std::chrono;

struct Launch {
    string id;
    string name;
    string symbol;
    string description;
    string tokenAddress;
    double price;
    double hardCap;
    double softCap;
    double raised;
    uint64_t startTime;
    uint64_t endTime;
    string status;
};

struct Participation {
    string id;
    string launchId;
    double amount;
    double tokenAmount;
    string status;
    uint64_t participatedAt;
};

class LaunchpadService {
private:
    mutex mutex_;
    unordered_map<string, Launch> launches_;
    unordered_map<string, vector<Participation>> participations_;
    
public:
    LaunchpadService() { initializeLaunches(); }
    
    void initializeLaunches() {
        lock_guard<mutex> lock(mutex_);
        launches_["ido1"] = {"ido1", "TigerToken", "TIGER", "Tiger Launch", "0xABC", 0.1, 1000000, 100000, 50000, 0, 0, "ACTIVE"};
    }
    
    vector<Launch> getActiveLaunches() {
        lock_guard<mutex> lock(mutex_);
        vector<Launch> result;
        for (auto& p : launches_) result.push_back(p.second);
        return result;
    }
    
    Participation participate(string launchId, double amount) {
        lock_guard<mutex> lock(mutex_);
        Participation p;
        p.id = "PART-" + to_string(duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count());
        p.launchId = launchId;
        p.amount = amount;
        p.tokenAmount = amount * 10;
        p.status = "CONFIRMED";
        p.participatedAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        participations_[launchId].push_back(p);
        return p;
    }
    
    bool claimTokens(string launchId) {
        return true;
    }
};

#endif
